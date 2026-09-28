package arithaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// CheckGeneratedUint64Conversions verifies the source of each generated
// uint64(v) conversion. Unsigned encoder parameters and Reader.ReadUint
// results cannot change value when converted to uint64.
func CheckGeneratedUint64Conversions(name string, source []byte) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, source, 0)
	if err != nil {
		return 0, err
	}
	checked := 0
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		unsigned := false
		if fn.Type.Params != nil {
			for _, field := range fn.Type.Params.List {
				if len(field.Names) != 1 || field.Names[0].Name != "v" {
					continue
				}
				typ, ok := field.Type.(*ast.Ident)
				unsigned = ok && (typ.Name == "uint8" || typ.Name == "uint16" || typ.Name == "uint32" || typ.Name == "uint64" || typ.Name == "uint")
			}
		}
		var failure error
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if failure != nil {
				return false
			}
			switch value := node.(type) {
			case *ast.AssignStmt:
				for i, lhs := range value.Lhs {
					if !isName(lhs, "v") {
						continue
					}
					unsigned = false
					if i != 0 || len(value.Rhs) != 1 {
						break
					}
					call, ok := value.Rhs[0].(*ast.CallExpr)
					if !ok {
						break
					}
					selector, ok := call.Fun.(*ast.SelectorExpr)
					unsigned = ok && isName(selector.X, "r") && selector.Sel.Name == "ReadUint" && len(call.Args) == 1
					break
				}
			case *ast.CallExpr:
				cast, ok := value.Fun.(*ast.Ident)
				if !ok || cast.Name != "uint64" || len(value.Args) != 1 || !isName(value.Args[0], "v") {
					break
				}
				if !unsigned {
					failure = fmt.Errorf("%s: uint64(v) lacks an unsigned source at %s", name, fset.Position(value.Pos()))
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
