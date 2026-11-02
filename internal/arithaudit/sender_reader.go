package arithaudit

import (
	"go/ast"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// generatedReaderImport resolves an unshadowed package qualifier to its
// module-local directory. Foreign paths and missing imports fail closed.
func (c *citationChecker) generatedReaderImport(file citationFile, qualifier string) (string, bool) {
	var imported string
	for _, spec := range file.file.Imports {
		target, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return "", false
		}
		name := path.Base(target)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == qualifier {
			imported = target
			break
		}
	}
	if imported == "" || path.Clean(imported) != imported {
		return "", false
	}
	manifest, err := os.ReadFile(filepath.Join(c.root, "go.mod"))
	if err != nil {
		return "", false
	}
	var module string
	for _, line := range strings.Split(string(manifest), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			module = fields[1]
			break
		}
	}
	if module == "" || !strings.HasPrefix(imported, module+"/") {
		return "", false
	}
	return strings.TrimPrefix(imported, module+"/"), true
}

// generatedReaderCallee follows local declarations and module-local generated
// imports. Missing source and foreign dependencies cannot establish a proof.
func (c *citationChecker) generatedReaderCallee(file citationFile, call *ast.CallExpr) (citationFile, string, bool) {
	if id, ok := call.Fun.(*ast.Ident); ok {
		return file, id.Name, id.Obj != nil && id.Obj.Kind == ast.Fun
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return citationFile{}, "", false
	}
	qualifier, ok := selector.X.(*ast.Ident)
	if !ok || qualifier.Obj != nil {
		return citationFile{}, "", false
	}
	directory, ok := c.generatedReaderImport(file, qualifier.Name)
	if !ok {
		return citationFile{}, "", false
	}
	target, err := c.source(path.Join(directory, "generated.go"))
	return target, selector.Sel.Name, err == nil
}

// generatedMonotoneReader checks every use of the reader, following actual
// helper bodies across module-local generated imports. Scoped forks start at
// the unchanged input reader; only those forks and their local selected aliases
// may be committed. A commit must immediately return from its block. Thus an
// old fork cannot overwrite progress made by a prior read or commit.
func (c *citationChecker) generatedMonotoneReader(file citationFile, name string, visiting map[string]bool) bool {
	key := file.set.Position(file.file.Pos()).Filename + ":" + name
	if visiting[key] {
		return true
	}
	var fn *ast.FuncDecl
	for _, decl := range file.file.Decls {
		if candidate, ok := decl.(*ast.FuncDecl); ok && candidate.Name.Name == name && candidate.Recv == nil {
			fn = candidate
			break
		}
	}
	if fn == nil || fn.Body == nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	if !c.generatedReaderEffects(file, fn, map[string]bool{}) {
		return false
	}
	parameter := fn.Type.Params.List[0]
	if len(parameter.Names) != 1 || generatedNodeText(file, parameter.Type) != "*runtime.Reader" {
		return false
	}
	if parameter.Names[0].Name == "_" {
		return true
	}
	if parameter.Names[0].Name != "r" {
		return false
	}
	root := parameter.Names[0].Obj
	readers := map[generatedDeclarationKey]bool{generatedDeclaration(parameter.Names[0]): true}
	choices := map[generatedDeclarationKey]bool{}
	valid := true
	references := map[*ast.Ident]bool{}
	visiting[key] = true
	defer delete(visiting, key)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if ok && len(spec.Names) == 1 && len(spec.Values) == 0 && spec.Type != nil && generatedNodeText(file, spec.Type) == "runtime.ChoiceErrors" {
			directory, ok := c.generatedReaderImport(file, "runtime")
			if !ok || directory != "runtime" {
				valid = false
				return false
			}
			choices[generatedDeclaration(spec.Names[0])] = true
		}
		return true
	})
	isReader := func(expr ast.Expr) bool {
		id, ok := expr.(*ast.Ident)
		return ok && id.Obj != nil && readers[generatedDeclaration(id)]
	}
	isChoiceCall := func(call *ast.CallExpr, method string) bool {
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != method {
			return false
		}
		id, ok := selector.X.(*ast.Ident)
		return ok && id.Obj != nil && choices[generatedDeclaration(id)]
	}
	isFork := func(expr ast.Expr) bool {
		call, ok := expr.(*ast.CallExpr)
		if !ok || !isChoiceCall(call, "ForkScoped") || len(call.Args) != 1 || call.Ellipsis.IsValid() {
			return false
		}
		id, ok := call.Args[0].(*ast.Ident)
		return ok && id.Obj == root
	}
	// Discover fork variables, then selected aliases, using declaration identity
	// so shadowed spellings do not inherit a different scope's provenance.
	forked := false
	for changed := true; changed; {
		changed = false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				return true
			}
			id, ok := assign.Lhs[0].(*ast.Ident)
			if !ok || id.Obj == nil || id.Obj == root {
				return true
			}
			borrowed := false
			if source, ok := assign.Rhs[0].(*ast.Ident); ok && source.Obj != root {
				borrowed = isReader(source)
			}
			if isFork(assign.Rhs[0]) {
				borrowed = true
				forked = true
			}
			if borrowed && !readers[generatedDeclaration(id)] {
				readers[generatedDeclaration(id)] = true
				changed = true
			}
			return true
		})
	}
	terminalCommits := map[*ast.CallExpr]bool{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok || len(block.List) < 2 {
			return true
		}
		statement, ok := block.List[len(block.List)-2].(*ast.ExprStmt)
		if !ok {
			return true
		}
		call, ok := statement.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		if _, ok := block.List[len(block.List)-1].(*ast.ReturnStmt); ok {
			terminalCommits[call] = true
		}
		return true
	})
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if !valid {
			return false
		}
		switch node := n.(type) {
		case *ast.AssignStmt:
			for index, lhs := range node.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok || !readers[generatedDeclaration(id)] {
					continue
				}
				if id.Obj == root || len(node.Lhs) != 1 || len(node.Rhs) != 1 || (node.Tok != token.DEFINE && node.Tok != token.ASSIGN) {
					valid = false
					return false
				}
				rhs := node.Rhs[index]
				if !isFork(rhs) && !isReader(rhs) {
					valid = false
					return false
				}
				if source, ok := rhs.(*ast.Ident); ok {
					if source.Obj == root {
						valid = false
						return false
					}
					references[source] = true
				}
				references[id] = true
			}
		case *ast.ValueSpec:
			for _, id := range node.Names {
				if readers[generatedDeclaration(id)] {
					if len(node.Values) != 0 {
						valid = false
						return false
					}
					references[id] = true
				}
				if choices[generatedDeclaration(id)] {
					references[id] = true
				}
			}
		case *ast.GoStmt:
			valid = false
			return false
		case *ast.FuncLit:
			ast.Inspect(node.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Obj != nil && (readers[generatedDeclaration(id)] || choices[generatedDeclaration(id)]) {
					valid = false
				}
				return valid
			})
			return false
		case *ast.CallExpr:
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
				if id, ok := selector.X.(*ast.Ident); ok && id.Obj != nil && choices[generatedDeclaration(id)] {
					switch selector.Sel.Name {
					case "ForkScoped", "Keep", "Release", "Track":
						references[id] = true
					default:
						valid = false
						return false
					}
				}
			}
			if node.Ellipsis.IsValid() {
				for _, arg := range node.Args {
					if isReader(arg) {
						valid = false
						return false
					}
				}
			}
			if method, ok := node.Fun.(*ast.SelectorExpr); ok && isReader(method.X) {
				receiver := method.X.(*ast.Ident)
				switch method.Sel.Name {
				case "Enter", "Leave", "EnterIndex", "LeaveIndex", "Check", "Position", "VirtualBits", "BoundedRemaining", "Remaining", "ExpressionResult", "ExpressionVariable", "Collecting", "Observe", "Set", "Error", "ErrorAt", "ErrorPath", "Discard":
				case "ReadSpare":
					if len(node.Args) != 0 || node.Ellipsis.IsValid() || forked && receiver.Obj == root {
						valid = false
						return false
					}
				case "ReadUint", "ReadBitString", "Expect", "ReadPadding", "ReadErrorRegion":
					if forked && receiver.Obj == root {
						valid = false
						return false
					}
				case "Commit":
					if !terminalCommits[node] || len(node.Args) != 1 || !isReader(node.Args[0]) {
						valid = false
						return false
					}
					source := node.Args[0].(*ast.Ident)
					if source.Obj == root {
						valid = false
						return false
					}
					references[source] = true
				default:
					valid = false
					return false
				}
				references[receiver] = true
			}
			for index, arg := range node.Args {
				if !isReader(arg) {
					continue
				}
				id := arg.(*ast.Ident)
				if isFork(node) || isChoiceCall(node, "Keep") && index == 0 && id.Obj == root && len(node.Args) == 2 {
					references[id] = true
					continue
				}
				if method, ok := node.Fun.(*ast.SelectorExpr); ok && method.Sel.Name == "Commit" && isReader(method.X) && references[id] {
					continue
				}
				target, callee, ok := c.generatedReaderCallee(file, node)
				if !ok || index != 0 || forked && id.Obj == root || !c.generatedMonotoneReader(target, callee, visiting) {
					valid = false
					return false
				}
				references[id] = true
			}
		}
		return valid
	})
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Obj != nil && (readers[generatedDeclaration(id)] || choices[generatedDeclaration(id)]) && !references[id] {
			valid = false
		}
		return valid
	})
	return valid
}

// generatedValueMethod resolves a method from the declared parameter type,
// following module-local imports instead of trusting the method's spelling.
func (c *citationChecker) generatedValueMethod(file citationFile, typ ast.Expr, name string) (citationFile, *ast.FuncDecl, bool) {
	if pointer, ok := typ.(*ast.StarExpr); ok {
		typ = pointer.X
	}
	if imported, ok := typ.(*ast.SelectorExpr); ok {
		qualifier, ok := imported.X.(*ast.Ident)
		if !ok || qualifier.Obj != nil {
			return citationFile{}, nil, false
		}
		directory, ok := c.generatedReaderImport(file, qualifier.Name)
		if !ok {
			return citationFile{}, nil, false
		}
		target, err := c.source(path.Join(directory, "generated.go"))
		if err != nil {
			return citationFile{}, nil, false
		}
		file = target
		typ = imported.Sel
	}
	id, ok := typ.(*ast.Ident)
	if !ok {
		return citationFile{}, nil, false
	}
	for _, decl := range file.file.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if !ok || method.Name.Name != name || method.Recv == nil || len(method.Recv.List) != 1 {
			continue
		}
		receiver := method.Recv.List[0].Type
		if pointer, ok := receiver.(*ast.StarExpr); ok {
			receiver = pointer.X
		}
		receiverName, ok := receiver.(*ast.Ident)
		if ok && receiverName.Name == id.Name {
			return file, method, true
		}
	}
	return citationFile{}, nil, false
}

// generatedPurePredicate admits only a single boolean return computed from
// the value parameter, constants and recursively proven generated helpers.
// Globals, callbacks and statements that could change reader state are refused.
func (c *citationChecker) generatedPurePredicate(file citationFile, fn *ast.FuncDecl, visiting map[string]bool) bool {
	key := file.set.Position(fn.Pos()).String()
	if visiting[key] || !c.generatedReaderPackageClosed(file) || fn.Body == nil || len(fn.Body.List) != 1 || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 || generatedNodeText(file, fn.Type.Results.List[0].Type) != "bool" {
		return false
	}
	var formal *ast.Field
	if fn.Recv != nil {
		if len(fn.Recv.List) != 1 || fn.Type.Params != nil && len(fn.Type.Params.List) != 0 {
			return false
		}
		formal = fn.Recv.List[0]
	} else {
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
			return false
		}
		formal = fn.Type.Params.List[0]
	}
	if len(formal.Names) != 1 {
		return false
	}
	result, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	var pure func(ast.Expr) bool
	pure = func(expr ast.Expr) bool {
		switch value := expr.(type) {
		case *ast.Ident:
			return value.Obj == formal.Names[0].Obj || value.Obj != nil && value.Obj.Kind == ast.Con || value.Obj == nil && (value.Name == "true" || value.Name == "false" || value.Name == "nil")
		case *ast.BasicLit:
			return true
		case *ast.ParenExpr:
			return pure(value.X)
		case *ast.StarExpr:
			return pure(value.X)
		case *ast.SelectorExpr:
			return pure(value.X)
		case *ast.UnaryExpr:
			return value.Op == token.NOT && pure(value.X)
		case *ast.BinaryExpr:
			switch value.Op {
			case token.LAND, token.LOR, token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				return pure(value.X) && pure(value.Y)
			}
		case *ast.CallExpr:
			if value.Ellipsis.IsValid() {
				return false
			}
			for _, arg := range value.Args {
				if !pure(arg) {
					return false
				}
			}
			if selector, ok := value.Fun.(*ast.SelectorExpr); ok {
				if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Obj == formal.Names[0].Obj {
					if len(value.Args) != 0 {
						return false
					}
					target, method, ok := c.generatedValueMethod(file, formal.Type, selector.Sel.Name)
					return ok && c.generatedPurePredicate(target, method, visiting)
				}
			}
			if len(value.Args) != 1 {
				return false
			}
			target, name, ok := c.generatedReaderCallee(file, value)
			if !ok {
				return false
			}
			for _, decl := range target.file.Decls {
				if helper, ok := decl.(*ast.FuncDecl); ok && helper.Recv == nil && helper.Name.Name == name {
					return c.generatedPurePredicate(target, helper, visiting)
				}
			}
		}
		return false
	}
	return pure(result.Results[0])
}
