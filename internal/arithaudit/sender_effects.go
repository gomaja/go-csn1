package arithaudit

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// generatedReaderPackageClosed requires the proved source to be the complete
// production declaration namespace. Extra documentation files may contribute
// comments, but unparsed declarations could add callbacks or shadow builtins.
func (c *citationChecker) generatedReaderPackageClosed(file citationFile) bool {
	for _, spec := range file.file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil || imported == "unsafe" {
			// Scalar metadata cannot establish reader independence when source
			// can reconstruct arbitrary pointers through aliases or addresses.
			return false
		}
	}
	name := file.set.Position(file.file.Pos()).Filename
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
		return false
	}
	directory := path.Dir(name)
	entries, err := os.ReadDir(filepath.Join(c.root, filepath.FromSlash(directory)))
	if err != nil {
		return false
	}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		full := filepath.Join(c.root, filepath.FromSlash(directory), entry.Name())
		contents, err := os.ReadFile(full)
		if err != nil {
			return false
		}
		if entry.Name() == path.Base(name) {
			if !bytes.Equal(contents, file.source) {
				return false
			}
			found = true
			continue
		}
		sibling, err := parser.ParseFile(token.NewFileSet(), entry.Name(), contents, 0)
		if err != nil || len(sibling.Decls) != 0 {
			return false
		}
	}
	return found
}

// A declaration can bind several names. Preserve both its node identity and
// its name so grouped declarations and shadowed locals remain distinct.
type generatedDeclarationKey struct {
	declaration ast.Node
	name        string
}

func generatedDeclaration(identifier *ast.Ident) generatedDeclarationKey {
	if identifier == nil || identifier.Obj == nil {
		return generatedDeclarationKey{}
	}
	declaration, ok := identifier.Obj.Decl.(ast.Node)
	if !ok {
		// Unknown declaration shapes cannot share a proved source binding.
		declaration = identifier
	}
	return generatedDeclarationKey{declaration: declaration, name: identifier.Obj.Name}
}

// generatedMetadataType admits only reader-free literal metadata. An alias,
// interface or wrapper containing a reader cannot qualify through its name.
func (c *citationChecker) generatedMetadataType(file citationFile, expr ast.Expr, visiting map[generatedDeclarationKey]bool) bool {
	switch typ := expr.(type) {
	case *ast.Ident:
		if typ.Obj == nil {
			switch typ.Name {
			case "string", "bool", "int", "uint", "uint8", "uint16", "uint32", "uint64", "int8", "int16", "int32", "int64":
				return true
			}
			return false
		}
		if typ.Obj.Kind != ast.Typ || visiting[generatedDeclaration(typ)] {
			return false
		}
		declaration, ok := typ.Obj.Decl.(*ast.TypeSpec)
		if !ok {
			return false
		}
		visiting[generatedDeclaration(typ)] = true
		defer delete(visiting, generatedDeclaration(typ))
		return c.generatedMetadataType(file, declaration.Type, visiting)
	case *ast.StarExpr:
		return c.generatedMetadataType(file, typ.X, visiting)
	case *ast.StructType:
		for _, field := range typ.Fields.List {
			if !c.generatedMetadataType(file, field.Type, visiting) {
				return false
			}
		}
		return true
	case *ast.SelectorExpr:
		qualifier, ok := typ.X.(*ast.Ident)
		if !ok || qualifier.Obj != nil {
			return false
		}
		directory, ok := c.generatedReaderImport(file, qualifier.Name)
		return ok && directory == "runtime" && (typ.Sel.Name == "Source" || typ.Sel.Name == "ErrorKind")
	}
	return false
}

func (c *citationChecker) generatedMetadataValue(file citationFile, expr ast.Expr, globals map[generatedDeclarationKey]bool, visiting map[generatedDeclarationKey]bool) bool {
	switch value := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		if value.Obj == nil {
			return value.Name == "true" || value.Name == "false" || value.Name == "nil"
		}
		if value.Obj.Kind == ast.Con {
			return true
		}
		if !globals[generatedDeclaration(value)] || visiting[generatedDeclaration(value)] {
			return false
		}
		declaration, ok := value.Obj.Decl.(*ast.ValueSpec)
		if !ok || len(declaration.Names) != 1 || len(declaration.Values) != 1 {
			return false
		}
		if declaration.Type != nil && !c.generatedMetadataType(file, declaration.Type, map[generatedDeclarationKey]bool{}) {
			return false
		}
		visiting[generatedDeclaration(value)] = true
		defer delete(visiting, generatedDeclaration(value))
		return c.generatedMetadataValue(file, declaration.Values[0], globals, visiting)
	case *ast.UnaryExpr:
		return (value.Op == token.AND || value.Op == token.SUB) && c.generatedMetadataValue(file, value.X, globals, visiting)
	case *ast.CompositeLit:
		if !c.generatedMetadataType(file, value.Type, map[generatedDeclarationKey]bool{}) {
			return false
		}
		for _, element := range value.Elts {
			if keyed, ok := element.(*ast.KeyValueExpr); ok {
				element = keyed.Value
			}
			if !c.generatedMetadataValue(file, element, globals, visiting) {
				return false
			}
		}
		return true
	case *ast.SelectorExpr:
		qualifier, ok := value.X.(*ast.Ident)
		if !ok || qualifier.Obj != nil {
			return false
		}
		directory, ok := c.generatedReaderImport(file, qualifier.Name)
		// Only the runtime's integer ErrorKind constants occur in this metadata.
		if !ok || directory != "runtime" {
			return false
		}
		switch value.Sel.Name {
		case "InvalidValue", "InvalidSchema", "Truncated", "Limit", "InvalidBranch":
			return true
		}
	}
	return false
}

func generatedObjectType(identifier *ast.Ident, visiting map[generatedDeclarationKey]bool) ast.Expr {
	if identifier == nil || identifier.Obj == nil {
		return nil
	}
	key := generatedDeclaration(identifier)
	if visiting[key] {
		return nil
	}
	visiting[key] = true
	defer delete(visiting, key)
	switch declaration := identifier.Obj.Decl.(type) {
	case *ast.Field:
		return declaration.Type
	case *ast.ValueSpec:
		if declaration.Type != nil {
			return declaration.Type
		}
		if len(declaration.Values) == 1 {
			if literal, ok := declaration.Values[0].(*ast.CompositeLit); ok {
				return literal.Type
			}
		}
	case *ast.AssignStmt:
		if len(declaration.Rhs) == 1 {
			if id, ok := declaration.Rhs[0].(*ast.Ident); ok {
				return generatedObjectType(id, visiting)
			}
		}
	}
	return nil
}

// generatedReaderEffects closes the call graph around the reader-use proof.
// Every global must be reader-free literal metadata, and every call must be a
// checked local helper, a runtime primitive or a diagnostic standard-library
// operation. Local methods are checked too, including possible error callbacks.
func (c *citationChecker) generatedReaderEffects(file citationFile, fn *ast.FuncDecl, visiting map[string]bool) bool {
	if fn == nil || fn.Body == nil || !c.generatedReaderPackageClosed(file) {
		return false
	}
	key := "effects:" + file.set.Position(fn.Pos()).String()
	if visiting[key] {
		return true
	}
	visiting[key] = true
	defer delete(visiting, key)
	globals := map[generatedDeclarationKey]bool{}
	for _, declaration := range file.file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, item := range group.Specs {
			spec, ok := item.(*ast.ValueSpec)
			if ok {
				for _, name := range spec.Names {
					globals[generatedDeclaration(name)] = true
				}
			}
		}
	}
	valid := true
	admittedCalls := map[*ast.SelectorExpr]bool{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if !valid {
			return false
		}
		if id, ok := node.(*ast.Ident); ok && id.Obj != nil && id.Obj.Kind == ast.Fun {
			helper, ok := id.Obj.Decl.(*ast.FuncDecl)
			if !ok || !c.generatedReaderEffects(file, helper, visiting) {
				valid = false
				return false
			}
		}
		if id, ok := node.(*ast.Ident); ok && globals[generatedDeclaration(id)] {
			if !c.generatedMetadataValue(file, id, globals, map[generatedDeclarationKey]bool{}) {
				valid = false
				return false
			}
		}
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if qualifier, ok := selector.X.(*ast.Ident); ok && qualifier.Obj == nil && !admittedCalls[selector] {
				directory, local := c.generatedReaderImport(file, qualifier.Name)
				if !local || directory != "runtime" {
					valid = false
					return false
				}
				// Runtime-owned data used by generated diagnostics. Imported
				// opaque values could invoke an unproved formatter or error method.
				switch selector.Sel.Name {
				case "Reader", "ChoiceErrors", "Source", "ErrorKind", "ErrorRegion", "BitString", "OperationError", "Location", "DecodeError", "BoundError", "FallbackError", "ContextRequiredError", "ExtentError", "InvalidValue", "InvalidSchema", "Truncated", "Limit", "InvalidBranch", "ErrContextRequired":
				default:
					valid = false
					return false
				}
			}
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if c.generatedReaderExpression(file, call) {
			admittedCalls[call.Fun.(*ast.SelectorExpr)] = true
			return true
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			if id.Obj == nil {
				switch id.Name {
				case "uint8", "uint16", "uint32", "uint64", "uint", "int", "int8", "int16", "int32", "int64", "len", "cap", "append", "make", "new", "copy":
					return true
				}
				valid = false
				return false
			}
			if id.Obj.Kind == ast.Typ {
				// A type name may alias an unsafe pointer or callback-bearing
				// dependency. Only reader-free data conversions are admitted.
				if !c.generatedMetadataType(file, id, map[generatedDeclarationKey]bool{}) {
					valid = false
					return false
				}
				return true
			}
			helper, ok := id.Obj.Decl.(*ast.FuncDecl)
			if !ok || !c.generatedReaderEffects(file, helper, visiting) {
				valid = false
				return false
			}
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			valid = false
			return false
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok {
			valid = false
			return false
		}
		if receiver.Obj == nil {
			admittedCalls[selector] = true
			var imported string
			for _, spec := range file.file.Imports {
				name := spec.Name
				target, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					valid = false
					return false
				}
				if name != nil && name.Name == receiver.Name || name == nil && (target == receiver.Name) {
					imported = target
					break
				}
			}
			switch imported {
			case "fmt":
				if selector.Sel.Name == "Sprintf" || selector.Sel.Name == "Errorf" {
					return true
				}
			case "errors":
				if selector.Sel.Name == "As" || selector.Sel.Name == "Is" {
					return true
				}
			case "strconv":
				if selector.Sel.Name == "Itoa" {
					return true
				}
			case "strings":
				if selector.Sel.Name == "Cut" || selector.Sel.Name == "HasSuffix" || selector.Sel.Name == "TrimPrefix" {
					return true
				}
			}
			directory, local := c.generatedReaderImport(file, receiver.Name)
			if local && directory == "runtime" {
				switch selector.Sel.Name {
				case "IsRegionSyntaxError", "PreferTruncation", "ErrorLabel":
					return true
				case "BitString":
					// This runtime-owned struct conversion cannot execute callbacks.
					if len(call.Args) != 1 || call.Ellipsis.IsValid() {
						valid = false
						return false
					}
					return true
				case "DeferOperation", "DeferIndex":
					// The emitter passes its named normalizer. A method value,
					// callback variable or aggregate field must not evade closure.
					argumentCount := 8
					if selector.Sel.Name == "DeferIndex" {
						argumentCount = 4
					}
					if len(call.Args) != argumentCount {
						valid = false
						return false
					}
					normalizer, ok := call.Args[argumentCount-1].(*ast.Ident)
					if !ok || normalizer.Obj == nil || normalizer.Obj.Kind != ast.Fun {
						valid = false
						return false
					}
					helper, ok := normalizer.Obj.Decl.(*ast.FuncDecl)
					if !ok || !c.generatedReaderEffects(file, helper, visiting) {
						valid = false
						return false
					}
					return true
				}
			}
			target, name, ok := c.generatedReaderCallee(file, call)
			if ok {
				for _, decl := range target.file.Decls {
					if helper, ok := decl.(*ast.FuncDecl); ok && helper.Recv == nil && helper.Name.Name == name {
						if c.generatedReaderEffects(target, helper, visiting) {
							return true
						}
						break
					}
				}
			}
			valid = false
			return false
		}
		typ := generatedObjectType(receiver, map[generatedDeclarationKey]bool{})
		if typ == nil {
			valid = false
			return false
		}
		if pointer, ok := typ.(*ast.StarExpr); ok {
			typ = pointer.X
		}
		if imported, ok := typ.(*ast.SelectorExpr); ok {
			qualifier, ok := imported.X.(*ast.Ident)
			if !ok {
				valid = false
				return false
			}
			if qualifier.Name == "runtime" && (imported.Sel.Name == "Reader" || imported.Sel.Name == "ChoiceErrors") {
				// The containing monotone-reader proof checks the exact operation and
				// all argument/alias provenance for these runtime-owned objects.
				return true
			}
		}
		target, method, ok := c.generatedValueMethod(file, typ, selector.Sel.Name)
		if !ok || !c.generatedReaderEffects(target, method, visiting) {
			valid = false
			return false
		}
		return true
	})
	if !valid {
		return false
	}
	// Formatting and error matching can invoke methods implicitly. They must not
	// gain an unchecked path to a global reader through a generated local type.
	for _, decl := range file.file.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if ok && method.Recv != nil {
			switch method.Name.Name {
			case "Error", "As", "Is", "String", "GoString", "Format", "Unwrap":
				if !c.generatedReaderEffects(file, method, visiting) {
					return false
				}
			}
		}
	}
	return true
}

// generatedReaderExpression proves the emitted ExpressionValue chain from its
// constructors. The pinned runtime value contains only n int64 and err error;
// Operand and Multiply operate on value receivers without a reader reference.
// A matching method name on an unproved receiver cannot inherit this contract.
func (c *citationChecker) generatedReaderExpression(file citationFile, expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if receiver, ok := selector.X.(*ast.Ident); ok {
		if receiver.Obj == nil && selector.Sel.Name == "ExpressionConstant" && len(call.Args) == 1 {
			directory, ok := c.generatedReaderImport(file, receiver.Name)
			literal, isLiteral := call.Args[0].(*ast.BasicLit)
			return ok && directory == "runtime" && isLiteral && literal.Kind == token.INT
		}
		if receiver.Obj != nil && selector.Sel.Name == "ExpressionVariable" && len(call.Args) == 1 {
			field, ok := receiver.Obj.Decl.(*ast.Field)
			if !ok || generatedNodeText(file, field.Type) != "*runtime.Reader" {
				return false
			}
			directory, ok := c.generatedReaderImport(file, "runtime")
			literal, isLiteral := call.Args[0].(*ast.BasicLit)
			return ok && directory == "runtime" && isLiteral && literal.Kind == token.STRING
		}
	}
	if !c.generatedReaderExpression(file, selector.X) {
		return false
	}
	switch selector.Sel.Name {
	case "Operand":
		return len(call.Args) == 0
	case "Multiply":
		return len(call.Args) == 1 && c.generatedReaderExpression(file, call.Args[0])
	}
	return false
}
