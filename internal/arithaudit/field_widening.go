package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type allocationFieldType struct {
	expression ast.Expr
	file       citationFile
	directory  string
}

type allocationFieldPackage struct {
	files        map[string]citationFile
	declarations map[string]allocationFieldType
	functions    map[string]*ast.FuncDecl
}

// checkAllocationFieldWidening follows only declared fields, explicit pointer
// dereferences and parentheses from v. Its terminal type must be a builtin
// unsigned integer of at most 64 bits. Go specification, Selectors and
// Conversions between numeric types: https://go.dev/ref/spec#Selectors and
// https://go.dev/ref/spec#Conversions_between_numeric_types. This proves the
// conversion is exact; it does not prove pointer validity or allocation meaning.
func (c *citationChecker) checkAllocationFieldWidening(file citationFile, fn *ast.FuncDecl, finding Finding) (bool, error) {
	if !strings.HasPrefix(finding.Expression, "uint64(") {
		return false, nil
	}
	var operand ast.Expr
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !isName(call.Fun, "uint64") || file.set.Position(call.Pos()).Line != finding.Line {
			return true
		}
		if finding.Column > 0 && file.set.Position(call.Pos()).Column != finding.Column {
			return true
		}
		var printed bytes.Buffer
		if err := format.Node(&printed, file.set, call); err != nil || printed.String() != finding.Expression {
			return true
		}
		switch call.Args[0].(type) {
		case *ast.SelectorExpr, *ast.StarExpr, *ast.ParenExpr:
			if call.Fun.(*ast.Ident).Obj == nil {
				operand = call.Args[0]
			}
		}
		return true
	})
	if operand == nil {
		return false, nil
	}
	fail := func() (bool, error) {
		return true, fmt.Errorf("unverified allocation field widening: %s", finding.Key())
	}
	directory := path.Dir(finding.File)
	if _, ok := c.allocationFieldPackage(directory); !ok {
		return fail()
	}
	typ, ok := c.allocationFieldOperand(allocationFieldType{operand, file, directory}, fn)
	if !ok {
		return fail()
	}
	typ, ok = c.allocationFieldUnderlying(typ, make(map[*ast.TypeSpec]bool))
	if !ok {
		return fail()
	}
	terminal, ok := typ.expression.(*ast.Ident)
	if !ok || terminal.Obj != nil {
		return fail()
	}
	switch terminal.Name {
	case "uint8", "uint16", "uint32", "uint64", "uint", "byte":
		return true, nil
	}
	return fail()
}

// Load the complete production namespace, including alternate build files.
// A nil parser Obj alone cannot establish an integer builtin's identity.
func (c *citationChecker) allocationFieldPackage(directory string) (*allocationFieldPackage, bool) {
	entries, err := os.ReadDir(filepath.Join(c.root, filepath.FromSlash(directory)))
	if err != nil {
		return nil, false
	}
	pkg := &allocationFieldPackage{files: make(map[string]citationFile), declarations: make(map[string]allocationFieldType), functions: make(map[string]*ast.FuncDecl)}
	packageName := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := c.source(path.Join(directory, entry.Name()))
		if err != nil || packageName != "" && file.file.Name.Name != packageName {
			return nil, false
		}
		packageName = file.file.Name.Name
		for builtin := range integerConversions {
			if file.file.Scope.Lookup(builtin) != nil {
				return nil, false
			}
		}
		pkg.files[entry.Name()] = file
		for _, declaration := range file.file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				if declaration.Tok != token.TYPE {
					continue
				}
				for _, spec := range declaration.Specs {
					typ := spec.(*ast.TypeSpec)
					if _, exists := pkg.declarations[typ.Name.Name]; exists || typ.TypeParams != nil {
						return nil, false
					}
					pkg.declarations[typ.Name.Name] = allocationFieldType{typ.Type, file, directory}
				}
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					if pkg.functions[declaration.Name.Name] != nil {
						return nil, false
					}
					pkg.functions[declaration.Name.Name] = declaration
				}
			}
		}
	}
	return pkg, packageName != ""
}

func (c *citationChecker) allocationFieldImport(typ allocationFieldType, name *ast.Ident) (string, bool) {
	if name.Obj != nil {
		return "", false
	}
	for _, spec := range typ.file.file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		const prefix = "github.com/gomaja/go-csn1/"
		if err != nil || !strings.HasPrefix(importPath, prefix) {
			continue
		}
		directory := strings.TrimPrefix(importPath, prefix)
		if directory == "" || directory == "." || path.IsAbs(directory) || path.Clean(directory) != directory || strings.HasPrefix(directory, "../") {
			continue
		}
		pkg, ok := c.allocationFieldPackage(directory)
		if !ok {
			continue
		}
		for _, file := range pkg.files {
			alias := file.file.Name.Name
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			if alias == name.Name {
				return directory, true
			}
			break
		}
	}
	return "", false
}

func (c *citationChecker) allocationFieldUnderlying(typ allocationFieldType, seen map[*ast.TypeSpec]bool) (allocationFieldType, bool) {
	for {
		pkg, ok := c.allocationFieldPackage(typ.directory)
		if !ok {
			return allocationFieldType{}, false
		}
		switch expression := typ.expression.(type) {
		case *ast.ParenExpr:
			typ.expression = expression.X
		case *ast.Ident:
			declaration, exists := pkg.declarations[expression.Name]
			if !exists {
				return typ, expression.Obj == nil && integerConversions[expression.Name]
			}
			object := declaration.file.file.Scope.Lookup(expression.Name)
			if object == nil || expression.Obj != nil && expression.Obj != object {
				return allocationFieldType{}, false
			}
			spec, ok := object.Decl.(*ast.TypeSpec)
			if !ok || seen[spec] {
				return allocationFieldType{}, false
			}
			seen[spec] = true
			typ = declaration
		case *ast.SelectorExpr:
			name, ok := expression.X.(*ast.Ident)
			if !ok {
				return allocationFieldType{}, false
			}
			directory, ok := c.allocationFieldImport(typ, name)
			if !ok || !ast.IsExported(expression.Sel.Name) {
				return allocationFieldType{}, false
			}
			imported, ok := c.allocationFieldPackage(directory)
			if !ok {
				return allocationFieldType{}, false
			}
			declaration, exists := imported.declarations[expression.Sel.Name]
			if !exists {
				return allocationFieldType{}, false
			}
			object := declaration.file.file.Scope.Lookup(expression.Sel.Name)
			if object == nil {
				return allocationFieldType{}, false
			}
			spec, ok := object.Decl.(*ast.TypeSpec)
			if !ok || seen[spec] {
				return allocationFieldType{}, false
			}
			seen[spec] = true
			typ = declaration
		default:
			return typ, true
		}
	}
}

func (c *citationChecker) allocationFieldOperand(typ allocationFieldType, fn *ast.FuncDecl) (allocationFieldType, bool) {
	switch expression := typ.expression.(type) {
	case *ast.ParenExpr:
		typ.expression = expression.X
		return c.allocationFieldOperand(typ, fn)
	case *ast.Ident:
		if expression.Name != "v" || expression.Obj == nil {
			return allocationFieldType{}, false
		}
		if fn.Type.Params != nil {
			for _, parameter := range fn.Type.Params.List {
				for _, name := range parameter.Names {
					if name.Obj == expression.Obj {
						typ.expression = parameter.Type
						return typ, true
					}
				}
			}
		}
		assignment, ok := expression.Obj.Decl.(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) < 1 || len(assignment.Rhs) != 1 {
			return allocationFieldType{}, false
		}
		first, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || first.Obj != expression.Obj {
			return allocationFieldType{}, false
		}
		for _, statement := range fn.Body.List {
			if statement == assignment {
				return c.allocationFieldResult(typ, assignment.Rhs[0])
			}
		}
	case *ast.StarExpr, *ast.SelectorExpr:
		var base ast.Expr
		switch expression := expression.(type) {
		case *ast.StarExpr:
			base = expression.X
		case *ast.SelectorExpr:
			base = expression.X
		}
		baseType := typ
		baseType.expression = base
		baseType, ok := c.allocationFieldOperand(baseType, fn)
		if !ok {
			return allocationFieldType{}, false
		}
		baseType, ok = c.allocationFieldUnderlying(baseType, make(map[*ast.TypeSpec]bool))
		if !ok {
			return allocationFieldType{}, false
		}
		pointer, isPointer := baseType.expression.(*ast.StarExpr)
		if _, dereference := expression.(*ast.StarExpr); dereference {
			if !isPointer {
				return allocationFieldType{}, false
			}
			baseType.expression = pointer.X
			return baseType, true
		}
		if isPointer {
			baseType.expression = pointer.X
			baseType, ok = c.allocationFieldUnderlying(baseType, make(map[*ast.TypeSpec]bool))
			if !ok {
				return allocationFieldType{}, false
			}
		}
		structure, ok := baseType.expression.(*ast.StructType)
		if !ok {
			return allocationFieldType{}, false
		}
		selector := expression.(*ast.SelectorExpr)
		for _, field := range structure.Fields.List {
			for _, name := range field.Names {
				if name.Name == selector.Sel.Name {
					baseType.expression = field.Type
					return baseType, true
				}
			}
		}
	}
	return allocationFieldType{}, false
}

func (c *citationChecker) allocationFieldResult(typ allocationFieldType, expression ast.Expr) (allocationFieldType, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return allocationFieldType{}, false
	}
	var name string
	switch callee := call.Fun.(type) {
	case *ast.Ident:
		if callee.Obj == nil || callee.Obj.Kind != ast.Fun {
			return allocationFieldType{}, false
		}
		name = callee.Name
	case *ast.SelectorExpr:
		qualifier, ok := callee.X.(*ast.Ident)
		if !ok || !ast.IsExported(callee.Sel.Name) {
			return allocationFieldType{}, false
		}
		typ.directory, ok = c.allocationFieldImport(typ, qualifier)
		if !ok {
			return allocationFieldType{}, false
		}
		name = callee.Sel.Name
	default:
		return allocationFieldType{}, false
	}
	pkg, ok := c.allocationFieldPackage(typ.directory)
	if !ok {
		return allocationFieldType{}, false
	}
	fn := pkg.functions[name]
	if fn == nil || fn.Type.TypeParams != nil || fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return allocationFieldType{}, false
	}
	for _, file := range pkg.files {
		for _, declaration := range file.file.Decls {
			if declaration == fn {
				return allocationFieldType{fn.Type.Results.List[0].Type, file, typ.directory}, true
			}
		}
	}
	return allocationFieldType{}, false
}
