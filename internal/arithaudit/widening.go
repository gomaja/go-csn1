package arithaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// CheckGeneratedUint64Conversions verifies the source of each generated
// uint64(v) conversion. Its source-only proof covers unsigned encoder
// parameters and Reader.ReadUint results. Local function results require the
// complete production package namespace and are rejected by this entrypoint.
func CheckGeneratedUint64Conversions(name string, source []byte) (int, error) {
	return checkGeneratedUint64Conversions(name, source, false)
}

// checkGeneratedPackageUint64Conversions requires every non-test Go source in
// the package, including declarations in sibling files. A source-local nil Obj
// cannot establish builtin identity: the package block can shadow that name.
func checkGeneratedPackageUint64Conversions(name string, sources map[string][]byte) (int, error) {
	source, present := sources[name]
	if !present {
		return 0, fmt.Errorf("missing generated source %s", name)
	}
	var packageName string
	for sibling, contents := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), sibling, contents, 0)
		if err != nil {
			return 0, err
		}
		if packageName == "" {
			packageName = file.Name.Name
		}
		if file.Name.Name != packageName {
			return 0, fmt.Errorf("inconsistent package in %s", sibling)
		}
		for _, builtin := range []string{"int", "uint", "uint8", "uint16", "uint32", "uint64", "byte"} {
			if file.Scope.Lookup(builtin) != nil {
				return 0, fmt.Errorf("integer builtin %s shadowed in %s", builtin, sibling)
			}
		}
	}
	return checkGeneratedUint64Conversions(name, source, true)
}

func checkGeneratedUint64Conversions(name string, source []byte, packageKnown bool) (int, error) {
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
				if !unsigned && (!packageKnown || !localUnsignedResult(fn, value.Args[0])) {
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

// localUnsignedResult authenticates the declaration bound to the call, rather
// than its spelling. Go specification, Conversions between numeric types:
// https://go.dev/ref/spec#Conversions_between_numeric_types. Restrict this proof
// to an unchanged value from a direct declaration in the wrapper body.
func localUnsignedResult(wrapper *ast.FuncDecl, expr ast.Expr) bool {
	value, ok := expr.(*ast.Ident)
	if !ok || value.Obj == nil {
		return false
	}
	assignment, ok := value.Obj.Decl.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) == 0 || len(assignment.Rhs) != 1 {
		return false
	}
	first, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || first.Obj != value.Obj {
		return false
	}
	direct := false
	for _, stmt := range wrapper.Body.List {
		if stmt == assignment {
			direct = true
		}
	}
	if !direct {
		return false
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	callee, ok := call.Fun.(*ast.Ident)
	if !ok || callee.Obj == nil || callee.Obj.Kind != ast.Fun {
		return false
	}
	fn, ok := callee.Obj.Decl.(*ast.FuncDecl)
	if !ok || fn.Recv != nil || fn.Body == nil || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}
	typ, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	if !ok || typ.Obj != nil {
		return false
	}
	switch typ.Name {
	case "uint8", "uint16", "uint32", "uint64", "uint", "byte":
	default:
		return false
	}
	unchanged := true
	ast.Inspect(wrapper.Body, func(node ast.Node) bool {
		written := func(expr ast.Expr) {
			if id, ok := expr.(*ast.Ident); ok && id.Obj == value.Obj {
				unchanged = false
			}
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			if node != assignment {
				for _, lhs := range node.Lhs {
					written(lhs)
				}
			}
		case *ast.IncDecStmt:
			written(node.X)
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				written(node.X)
			}
		}
		return unchanged
	})
	return unchanged
}
