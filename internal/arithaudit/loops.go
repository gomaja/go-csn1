package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
)

// CheckGeneratedLoopArithmetic verifies the two emitted index patterns:
// counted repeats and recursive-list selectors. Their generator sites are
// csn1/gen/direct.go:625-628,665 and csn1/gen/direct.go:853.
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
		if loop.Post != inc || !zeroLoopInit(loop.Init) || !countLoopCondition(loop.Cond) {
			continue
		}
		countMutated := false
		ast.Inspect(loop.Body, func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.IncDecStmt:
				id, ok := stmt.X.(*ast.Ident)
				countMutated = countMutated || ok && id.Name == "count"
			case *ast.AssignStmt:
				for _, lhs := range stmt.Lhs {
					id, ok := lhs.(*ast.Ident)
					countMutated = countMutated || ok && id.Name == "count"
				}
			}
			return !countMutated
		})
		if countMutated {
			continue
		}
		bounded, unsafe := false, false
		ast.Inspect(body, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok || assign.Pos() >= loop.Pos() || len(assign.Lhs) == 0 || len(assign.Rhs) == 0 {
				return true
			}
			id, ok := assign.Lhs[0].(*ast.Ident)
			if !ok || id.Name != "count" {
				return true
			}
			valid := false
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if ok {
				if selector, ok := call.Fun.(*ast.SelectorExpr); ok && (selector.Sel.Name == "Eval" || selector.Sel.Name == "SpareCount") {
					valid = true
				}
			}
			if binary, ok := assign.Rhs[0].(*ast.BinaryExpr); ok && binary.Op == token.QUO {
				lhs, lhsOK := binary.X.(*ast.Ident)
				divisor, divisorOK := binary.Y.(*ast.BasicLit)
				if lhsOK && lhs.Name == "remaining" && divisorOK && divisor.Value == "8" && boundedRemainingAssignment(body, assign.Pos()) {
					valid = true // BoundedRemaining is at most the checked input length.
				}
			}
			bounded = true
			unsafe = unsafe || !valid
			return true
		})
		if bounded && !unsafe {
			return true
		}
	}
	return false
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
