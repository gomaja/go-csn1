package arithaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// CheckGeneratedNarrowing proves each generated uint8/16/32 conversion fits
// the literal read width and any interpretation offset applied before it.
// A new generated expression shape fails closed until it is reviewed.
func CheckGeneratedNarrowing(name string, source []byte) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, source, 0)
	if err != nil {
		return 0, err
	}
	checked := 0
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "decode") || !hasNarrowingCast(fn.Body) {
			continue
		}
		width, readWidth, offset := -1, -1, uint64(0)
		readUint, unreviewedWidthWrite := false, false
		directWrites := directFunctionStatements(fn)
		var closureBodies []*ast.BlockStmt
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if writesName(node, "width") {
				assign, ok := node.(*ast.AssignStmt)
				if !ok || assignedReadWidth(assign) < 0 || !directWrites[node] {
					unreviewedWidthWrite = true
				}
			}
			switch value := node.(type) {
			case *ast.FuncLit:
				closureBodies = append(closureBodies, value.Body)
			case *ast.BranchStmt:
				unreviewedWidthWrite = unreviewedWidthWrite || value.Tok == token.GOTO
			case *ast.LabeledStmt:
				unreviewedWidthWrite = true
			}
			return true
		})
		if !reviewedValueWrites(fn, directWrites) || hasClosureNarrowing(fn.Body) || !reviewedLHMaskWrites(citationFile{set: fset, file: file, source: source}, fn) {
			return checked, fmt.Errorf("%s: unreviewed generated v write or closure cast in %s", name, fn.Name.Name)
		}
		var failure error
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if failure != nil {
				return false
			}
			switch value := node.(type) {
			case *ast.AssignStmt:
				if writesName(value, "width") {
					width = assignedReadWidth(value)
				}
				if len(value.Lhs) == 0 || !isName(value.Lhs[0], "v") {
					break
				}
				if value.Tok == token.XOR_ASSIGN && len(value.Rhs) == 1 && readUint && readWidth == 1 && offset == 0 && isLHBitMask(value.Rhs[0]) {
					break // one-bit L/H semantic conversion preserves the one-bit range
				}
				if len(value.Rhs) == 1 {
					if call, ok := value.Rhs[0].(*ast.CallExpr); ok {
						if selector, ok := call.Fun.(*ast.SelectorExpr); ok && isName(selector.X, "r") && selector.Sel.Name == "ReadUint" && len(call.Args) == 1 && isName(call.Args[0], "width") {
							readUint, readWidth, offset = true, width, 0
							if unreviewedWidthWrite {
								readWidth = -1
							}
							if insideClosure(call, closureBodies) {
								readWidth = -1
							}
							break
						}
					}
				}
				if value.Tok != token.ADD_ASSIGN || len(value.Rhs) != 1 {
					failure = fmt.Errorf("%s: unreviewed generated v assignment at %s", name, fset.Position(value.Pos()))
					break
				}
				literal, ok := value.Rhs[0].(*ast.BasicLit)
				if !ok {
					failure = fmt.Errorf("%s: nonliteral interpretation offset at %s", name, fset.Position(value.Pos()))
					break
				}
				increment, err := strconv.ParseUint(literal.Value, 0, 64)
				if err != nil || increment > ^uint64(0)-offset {
					failure = fmt.Errorf("%s: invalid interpretation offset at %s", name, fset.Position(value.Pos()))
					break
				}
				offset += increment
			case *ast.CallExpr:
				cast, ok := value.Fun.(*ast.Ident)
				if !ok || len(value.Args) != 1 || !isName(value.Args[0], "v") {
					break
				}
				var bits int
				switch cast.Name {
				case "uint8":
					bits = 8
				case "uint16":
					bits = 16
				case "uint32":
					bits = 32
				}
				if bits == 0 {
					break
				}
				if !readUint || readWidth < 0 || readWidth > bits {
					failure = fmt.Errorf("%s: uint%d(v) lacks a fitting fixed-width read at %s", name, bits, fset.Position(value.Pos()))
					break
				}
				readMaximum := uint64(1)<<uint(readWidth) - 1
				targetMaximum := uint64(1)<<uint(bits) - 1
				if offset > targetMaximum-readMaximum {
					failure = fmt.Errorf("%s: uint%d(v) overflows after offset %d at %s", name, bits, offset, fset.Position(value.Pos()))
					break
				}
				checked++
			}
			return true
		})
		if failure != nil {
			return checked, failure
		}
	}
	return checked, nil
}

func assignedReadWidth(assign *ast.AssignStmt) int {
	expr, ok := plainAssignmentValue(assign, "width")
	if !ok {
		return -1
	}
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.INT && len(assign.Lhs) == 1 {
		if n, err := strconv.ParseInt(literal.Value, 0, 32); err == nil && n >= 0 && n <= 64 {
			return int(n)
		}
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || len(assign.Lhs) > 2 {
		return -1
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isName(selector.X, "r") || selector.Sel.Name != "Eval" {
		return -1
	}
	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return -1
	}
	raw, err := strconv.Unquote(literal.Value)
	if err == nil {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 && n <= 64 {
			return n
		}
	}
	return -1
}

func isLHBitMask(expr ast.Expr) bool {
	outer, ok := expr.(*ast.CallExpr)
	if !ok || len(outer.Args) != 1 || !isName(outer.Fun, "uint64") {
		return false
	}
	if isName(outer.Args[0], "lhL") {
		return true
	}
	return false
}

func hasNarrowingCast(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !isName(call.Args[0], "v") {
			return true
		}
		cast, ok := call.Fun.(*ast.Ident)
		if ok && (cast.Name == "uint8" || cast.Name == "uint16" || cast.Name == "uint32") {
			found = true
			return false
		}
		return true
	})
	return found
}

func isName(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// directFunctionStatements identifies writes whose source order is execution
// order. Nested writes, including shadowing declarations, cannot prove the
// width reaching a read. Function literals may execute after later writes;
// labels and goto can bypass a preceding write entirely.
// Go specification, Blocks, Function literals, Labeled statements and Goto
// statements: https://go.dev/ref/spec#Blocks, #Function_literals,
// #Labeled_statements and #Goto_statements.
func directFunctionStatements(fn *ast.FuncDecl) map[ast.Node]bool {
	statements := make(map[ast.Node]bool, len(fn.Body.List))
	for _, statement := range fn.Body.List {
		statements[statement] = true
	}
	return statements
}

// reviewedValueWrites requires every write to the read value to occur in
// execution order as a direct function-body statement. Address escape and
// same-spelled shadowing writes fail closed through writesName.
// Go specification, Assignments and Function literals:
// https://go.dev/ref/spec#Assignments and #Function_literals.
func reviewedValueWrites(fn *ast.FuncDecl, directWrites map[ast.Node]bool) bool {
	valid := true
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if !writesName(node, "v") {
			return true
		}
		assign, ok := node.(*ast.AssignStmt)
		if !ok || !directWrites[node] || !reviewedValueAssignment(assign) {
			valid = false
		}
		return true
	})
	return valid
}

func reviewedValueAssignment(assign *ast.AssignStmt) bool {
	if expr, ok := plainAssignmentValue(assign, "v"); ok && len(assign.Lhs) <= 2 {
		call, ok := expr.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !isName(call.Args[0], "width") {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		return ok && isName(selector.X, "r") && selector.Sel.Name == "ReadUint"
	}
	if len(assign.Lhs) != 1 || !isName(assign.Lhs[0], "v") || len(assign.Rhs) != 1 {
		return false
	}
	if assign.Tok == token.XOR_ASSIGN {
		return isLHBitMask(assign.Rhs[0])
	}
	literal, ok := assign.Rhs[0].(*ast.BasicLit)
	if assign.Tok != token.ADD_ASSIGN || !ok || literal.Kind != token.INT {
		return false
	}
	_, err := strconv.ParseUint(literal.Value, 0, 64)
	return err == nil
}

// A function literal can execute after writes that follow its source position.
// Go specification, Function literals: https://go.dev/ref/spec#Function_literals.
func hasClosureNarrowing(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if literal, ok := node.(*ast.FuncLit); ok && hasNarrowingCast(literal.Body) {
			found = true
		}
		return !found
	})
	return found
}

func insideClosure(node ast.Node, bodies []*ast.BlockStmt) bool {
	for _, body := range bodies {
		if body.Pos() <= node.Pos() && node.End() <= body.End() {
			return true
		}
	}
	return false
}
