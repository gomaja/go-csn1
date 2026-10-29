package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// CheckGeneratedLoopArithmetic verifies the two emitted index patterns:
// counted repeats and recursive-list selectors. Their generator sites are
// csn1/gen/direct.go:1045 `count,err:=w.Eval(%q)`, csn1/gen/direct.go:1042
// `for i:=0;i<count;i++` and csn1/gen/direct.go:1609 `if i+1<len(v.Entries)`.
func CheckGeneratedLoopArithmetic(name string, source []byte) (increments, nextIndexes int, err error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, name, source, 0)
	if err != nil {
		return 0, 0, err
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Name.Name == "constrainedWidth" {
			start, end := set.Position(fn.Pos()).Offset, set.Position(fn.End()).Offset
			body := generatedWhitespace.ReplaceAllString(string(source[start:end]), "")
			if !strings.Contains(body, "fori:=len(path)-1;i>=0;i--") || !strings.Contains(body, "path=path[i+1:]") {
				return 0, 0, fmt.Errorf("%s: unbounded field-path scan", name)
			}
			continue // Its decrement and successor are checked in residual.go.
		}
		var counted []*ast.ForStmt
		var ranged []*ast.RangeStmt
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.ForStmt:
				counted = append(counted, stmt)
			case *ast.RangeStmt:
				ranged = append(ranged, stmt)
			}
			return true
		})
		var fault error
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if fault != nil {
				return false
			}
			switch stmt := node.(type) {
			case *ast.IncDecStmt:
				id, ok := stmt.X.(*ast.Ident)
				if !ok || id.Name != "i" {
					break
				}
				if stmt.Tok != token.INC || !boundedForIncrement(fn.Body, counted, stmt) {
					fault = fmt.Errorf("%s: loop increment has no bounded count", name)
					return false
				}
				increments++
			case *ast.BinaryExpr:
				id, ok := stmt.X.(*ast.Ident)
				if !ok || id.Name != "i" || stmt.Op != token.ADD || !isOneLiteral(stmt.Y) {
					break
				}
				if !boundedRangeNext(set, fn.Body, ranged, stmt) {
					fault = fmt.Errorf("%s: i+1 is not bounded by its ranged slice", name)
					return false
				}
				nextIndexes++
			}
			return true
		})
		if fault != nil {
			return 0, 0, fault
		}
	}
	return increments, nextIndexes, nil
}

func boundedForIncrement(body *ast.BlockStmt, loops []*ast.ForStmt, inc *ast.IncDecStmt) bool {
	for _, loop := range loops {
		if loop.Post != inc {
			continue
		}
		if fixedLoopCondition(loop.Cond) && (zeroLoopInit(loop.Init) || lengthLoopInit(loop.Init)) {
			return true
		}
		if !zeroLoopInit(loop.Init) || !countLoopCondition(loop.Cond) {
			continue
		}
		if hasNameWrite(loop.Body, "count") {
			continue
		}
		bounded, unsafe := false, false
		ast.Inspect(body, func(node ast.Node) bool {
			if !writesName(node, "count") {
				return true
			}
			assign, ok := node.(*ast.AssignStmt)
			valid := ok && boundedCountAssignment(body, assign)
			bounded = bounded || valid && assign.Pos() < loop.Pos()
			// A write after this loop can reach it on an enclosing loop's
			// next iteration, so every write in the function must be bounded.
			unsafe = unsafe || !valid
			return true
		})
		if bounded && !unsafe {
			return true
		}
	}
	return false
}

// boundedCountAssignment accepts only the first result of reviewed runtime
// sources, or a single literal/divided remaining count. A multi-assignment
// of unrelated values does not bind the first RHS to count.
//
// This is a deliberately conservative syntax proof, not data-flow analysis:
// every count write in a function containing a counted loop is reviewed,
// including writes lexically after it. Choice counters without counted
// loops are checked separately by CheckGeneratedChoiceCounters.
func boundedCountAssignment(body *ast.BlockStmt, assign *ast.AssignStmt) bool {
	expr, ok := plainAssignmentValue(assign, "count")
	if !ok {
		return false
	}
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.INT && len(assign.Lhs) == 1 {
		n, err := strconv.ParseUint(literal.Value, 0, 32)
		return err == nil && n <= 1<<20
	}
	if call, ok := expr.(*ast.CallExpr); ok && len(assign.Lhs) <= 2 {
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && (isName(selector.X, "r") || isName(selector.X, "w")) {
			switch selector.Sel.Name {
			case "Eval", "EvalTokens", "SpareCount", "ExpressionResult":
				return len(call.Args) == 1
			case "SpareFillCount":
				return len(call.Args) == 0
			}
		}
	}
	if binary, ok := expr.(*ast.BinaryExpr); ok && binary.Op == token.QUO && len(assign.Lhs) == 1 {
		lhs, lhsOK := binary.X.(*ast.Ident)
		divisor, divisorOK := binary.Y.(*ast.BasicLit)
		// BoundedRemaining is at most the checked input length.
		return lhsOK && lhs.Name == "remaining" && divisorOK && divisor.Value == "8" && boundedRemainingAssignment(body, assign.Pos())
	}
	return false
}

func lengthLoopInit(stmt ast.Stmt) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 || assign.Tok != token.DEFINE {
		return false
	}
	index, ok := assign.Lhs[0].(*ast.Ident)
	if !ok || index.Name != "i" {
		return false
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	fn, ok := call.Fun.(*ast.Ident)
	return ok && fn.Name == "len"
}

func fixedLoopCondition(expr ast.Expr) bool {
	if both, ok := expr.(*ast.BinaryExpr); ok && both.Op == token.LAND {
		return fixedLoopCondition(both.X) || fixedLoopCondition(both.Y)
	}
	comparison, ok := expr.(*ast.BinaryExpr)
	if !ok || comparison.Op != token.LSS {
		return false
	}
	index, ok := comparison.X.(*ast.Ident)
	if !ok || index.Name != "i" {
		return false
	}
	bound, ok := comparison.Y.(*ast.BasicLit)
	if !ok || bound.Kind != token.INT {
		return false
	}
	value, err := strconv.ParseUint(bound.Value, 0, 32)
	return err == nil && value <= 1<<20
}

func boundedRemainingAssignment(body *ast.BlockStmt, before token.Pos) bool {
	found, unsafe := false, false
	ast.Inspect(body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || assign.Pos() >= before || len(assign.Lhs) == 0 || len(assign.Rhs) == 0 {
			return true
		}
		id, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || id.Name != "remaining" {
			return true
		}
		found = true
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			unsafe = true
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		unsafe = unsafe || !ok || selector.Sel.Name != "BoundedRemaining"
		return true
	})
	return found && !unsafe
}

func zeroLoopInit(stmt ast.Stmt) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 || assign.Tok != token.DEFINE {
		return false
	}
	id, ok := assign.Lhs[0].(*ast.Ident)
	return ok && id.Name == "i" && isZeroLiteral(assign.Rhs[0])
}

func countLoopCondition(expr ast.Expr) bool {
	cond, ok := expr.(*ast.BinaryExpr)
	if !ok || cond.Op != token.LSS {
		return false
	}
	i, okI := cond.X.(*ast.Ident)
	count, okCount := cond.Y.(*ast.Ident)
	return okI && okCount && i.Name == "i" && count.Name == "count"
}

func isOneLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "1"
}

func boundedRangeNext(set *token.FileSet, body *ast.BlockStmt, ranges []*ast.RangeStmt, addition *ast.BinaryExpr) bool {
	for _, loop := range ranges {
		if addition.Pos() < loop.Body.Pos() || addition.End() > loop.Body.End() {
			continue
		}
		id, ok := loop.Key.(*ast.Ident)
		if !ok || id.Name != "i" {
			continue
		}
		var source bytes.Buffer
		if format.Node(&source, set, loop.X) != nil {
			continue
		}
		found := false
		ast.Inspect(body, func(node ast.Node) bool {
			stmt, ok := node.(*ast.IfStmt)
			if !ok || stmt.Cond.Pos() > addition.Pos() || addition.End() > stmt.Cond.End() {
				return true
			}
			comparison, ok := stmt.Cond.(*ast.BinaryExpr)
			if !ok || comparison.Op != token.LSS || comparison.X != addition {
				return true
			}
			call, ok := comparison.Y.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			function, ok := call.Fun.(*ast.Ident)
			if !ok || function.Name != "len" {
				return true
			}
			var bound bytes.Buffer
			if format.Node(&bound, set, call.Args[0]) == nil && bound.String() == source.String() {
				found = true
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}
