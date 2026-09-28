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
		readUint := false
		var failure error
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if failure != nil {
				return false
			}
			switch value := node.(type) {
			case *ast.AssignStmt:
				if len(value.Lhs) == 0 || !isName(value.Lhs[0], "v") {
					break
				}
				if len(value.Rhs) == 1 {
					if call, ok := value.Rhs[0].(*ast.CallExpr); ok {
						if selector, ok := call.Fun.(*ast.SelectorExpr); ok && isName(selector.X, "r") && selector.Sel.Name == "ReadUint" && len(call.Args) == 1 && isName(call.Args[0], "width") {
							readUint, readWidth, offset = true, width, 0
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
				if selector, ok := value.Fun.(*ast.SelectorExpr); ok {
					if isName(selector.X, "r") && selector.Sel.Name == "Eval" && len(value.Args) == 1 {
						literal, ok := value.Args[0].(*ast.BasicLit)
						if !ok {
							failure = fmt.Errorf("%s: nonliteral read width at %s", name, fset.Position(value.Pos()))
							break
						}
						raw, err := strconv.Unquote(literal.Value)
						if err != nil {
							failure = err
							break
						}
						width, err = strconv.Atoi(raw)
						if err != nil {
							width = -1
						}
					}
					break
				}
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
				default:
					break
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
