package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkGeneratedGroupedSpare proves that an initially zero uint64 appends
// exactly one bit in each of at most 64 iterations. This proves arithmetic
// structure independently of the source's spare interpretation.
func (c *citationChecker) checkGeneratedGroupedSpare(f Finding) (bool, error) {
	if f.Expression != "v <<= 1" {
		return false, nil
	}
	file, err := c.source(f.File)
	if err != nil {
		return true, err
	}
	fail := func() (bool, error) { return true, fmt.Errorf("grouped spare accumulator differs: %s", f.Key()) }
	fn := functionAt(file, f.Line)
	if fn == nil || fn.Body == nil || fn.Type.Params == nil || fn.Type.Results == nil {
		return fail()
	}
	text := func(n ast.Node) string {
		a, b := file.set.Position(n.Pos()).Offset, file.set.Position(n.End()).Offset
		return strings.ReplaceAll(generatedWhitespace.ReplaceAllString(string(file.source[a:b]), ""), ";", "")
	}
	if !allocationRuntimeImport(file.file) || len(fn.Type.Params.List) != 1 || text(fn.Type.Params.List[0]) != "r*runtime.Reader" || text(fn.Type.Results) != "(runtime.BitString,error)" {
		return fail()
	}
	safe := true
	for _, name := range []string{"runtime", "uint64", "len", "nil", "error"} {
		if file.file.Scope.Lookup(name) != nil {
			safe = false
		}
	}
	ast.Inspect(fn, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			switch id.Name {
			case "runtime", "uint64", "len", "nil", "error":
				if id.Obj != nil {
					safe = false
				}
			}
		}
		return true
	})
	if !safe || len(fn.Body.List) < 7 {
		return fail()
	}
	statements := fn.Body.List
	enter, ok := statements[0].(*ast.IfStmt)
	if !ok || enter.Init == nil || enter.Else != nil || text(enter.Cond) != "err!=nil" || text(enter.Body) != "{returnruntime.BitString{},err}" {
		return fail()
	}
	assignment, ok := enter.Init.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || text(assignment.Lhs[0]) != "err" || len(assignment.Rhs) != 1 {
		return fail()
	}
	call, ok := assignment.Rhs[0].(*ast.CallExpr)
	if !ok || text(call.Fun) != "r.Enter" || len(call.Args) != 1 {
		return fail()
	}
	label, ok := call.Args[0].(*ast.BasicLit)
	if !ok || label.Kind != token.STRING {
		return fail()
	}
	if text(statements[1]) != "deferr.Leave()" || text(statements[3]) != "fieldStart:=0" || text(statements[4]) != "ifr.Collecting(){fieldStart=r.Position()}" || text(statements[5]) != "varvuint64" {
		return fail()
	}
	width, ok := statements[2].(*ast.AssignStmt)
	if !ok || width.Tok != token.DEFINE || len(width.Lhs) != 1 || text(width.Lhs[0]) != "width" || len(width.Rhs) != 1 {
		return fail()
	}
	literal, ok := width.Rhs[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.INT {
		return fail()
	}
	bound, err := strconv.ParseUint(literal.Value, 0, 64)
	if err != nil || bound < 1 || bound > 64 {
		return fail()
	}
	loop, ok := statements[6].(*ast.ForStmt)
	if !ok || len(loop.Body.List) != 4 {
		return fail()
	}
	guard, ok := loop.Body.List[1].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || text(guard.Cond) != "err!=nil" || len(guard.Body.List) != 1 {
		return fail()
	}
	ret, ok := guard.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 || text(ret.Results[0]) != "runtime.BitString{}" {
		return fail()
	}
	errorText := text(ret.Results[1])
	located := "locateIndex(operationLocation0.failure(err,\"\",r.Position(),width)," + text(label) + ",i)"
	if errorText != "err" && errorText != located {
		return fail()
	}
	shape := "fori:=0i<widthi++{bit,err:=r.ReadSpare()iferr!=nil{returnruntime.BitString{}," + errorText + "}v<<=1iflen(bit.Bytes)>0&&bit.Bytes[0]&0x80!=0{v|=1}}"
	if text(loop) != shape {
		return fail()
	}
	shift, ok := loop.Body.List[2].(*ast.AssignStmt)
	if !ok || shift.Tok != token.SHL_ASSIGN || file.set.Position(shift.Pos()).Line != f.Line || (f.Column > 0 && file.set.Position(shift.Pos()).Column != f.Column) {
		return fail()
	}
	// Names must resolve to the reader parameter and the exact local declarations.
	objects := map[string]*ast.Ident{}
	objects["r"] = fn.Type.Params.List[0].Names[0]
	objects["width"] = width.Lhs[0].(*ast.Ident)
	objects["fieldStart"] = statements[3].(*ast.AssignStmt).Lhs[0].(*ast.Ident)
	objects["v"] = statements[5].(*ast.DeclStmt).Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0]
	init := loop.Init.(*ast.AssignStmt)
	objects["i"] = init.Lhs[0].(*ast.Ident)
	read := loop.Body.List[0].(*ast.AssignStmt)
	objects["bit"] = read.Lhs[0].(*ast.Ident)
	objects["err"] = read.Lhs[1].(*ast.Ident)
	for _, object := range objects {
		if object == nil || object.Obj == nil {
			return fail()
		}
	}
	ast.Inspect(loop, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if object, exists := objects[id.Name]; exists && id.Obj != object.Obj {
				safe = false
			}
		}
		return true
	})
	for _, statement := range statements[:6] {
		ast.Inspect(statement, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && (id.Name == "r" || id.Name == "fieldStart") {
				if id.Obj != objects[id.Name].Obj {
					safe = false
				}
			}
			return true
		})
	}
	if !safe {
		return fail()
	}
	return true, nil
}

// generatedGroupedSpareChild requires the same complete bounded accumulator
// shape when a sequence discards the spare value. A monotone unrelated child
// cannot inherit permission to discard its result.
func generatedGroupedSpareChild(file citationFile, name string) bool {
	for _, decl := range file.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != name || fn.Body == nil {
			continue
		}
		valid := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			shift, ok := n.(*ast.AssignStmt)
			if !ok || shift.Tok != token.SHL_ASSIGN {
				return true
			}
			pos := file.set.Position(shift.Pos())
			checker := &citationChecker{files: map[string]citationFile{"generated.go": file}}
			handled, err := checker.checkGeneratedGroupedSpare(Finding{File: "generated.go", Line: pos.Line, Column: pos.Column, Expression: "v <<= 1"})
			valid = handled && err == nil
			return false
		})
		return valid
	}
	return false
}

// boundedGroupedSpareIncrement shares the full accumulator proof: width is
// a local constant in [1,64], i starts at zero, and the only counter update
// is the loop's post-increment. Thus i++ cannot exceed 64.
func boundedGroupedSpareIncrement(file citationFile, fn *ast.FuncDecl, loops []*ast.ForStmt, inc *ast.IncDecStmt) bool {
	for _, loop := range loops {
		if len(fn.Body.List) > 6 && fn.Body.List[6] == loop && loop.Post == inc && generatedGroupedSpareChild(file, fn.Name.Name) {
			return true
		}
	}
	return false
}

// checkStructuralSpareFindings proves these generated expressions directly,
// without allowlist exceptions. Unrecognized expressions stay in the counted
// inventory; a recognized expression with a changed body fails closed.
func (c *citationChecker) checkStructuralSpareFindings(findings []Finding, allowed map[string]Classification) ([]Finding, error) {
	remaining := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if strings.HasSuffix(f.File, "/generated.go") {
			switch f.Expression {
			case "i++":
				if _, exists := allowed[f.Signature()]; !exists {
					file, err := c.source(f.File)
					if err != nil {
						return nil, err
					}
					if !generatedSpareIncrement(file, f) {
						return nil, fmt.Errorf("generated spare increment differs: %s", f.Key())
					}
					continue
				}

			case "v <<= 1", "r.Position() - senderStart", "r.VirtualBits() - senderVirtual":
				if err := c.checkGeneratedResidual(f); err != nil {
					return nil, err
				}
				continue
			}
		}
		remaining = append(remaining, f)
	}
	return remaining, nil
}

// generatedSpareIncrement admits only the accumulator's own increment or the
// complete fixed spare writer. In particular, a second loop cannot borrow a
// nearby bound and no counter write or escape can be inserted into either body.
func generatedSpareIncrement(file citationFile, f Finding) bool {
	fn := functionAt(file, f.Line)
	if fn == nil || fn.Body == nil {
		return false
	}
	var loops []*ast.ForStmt
	var inc *ast.IncDecStmt
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if loop, ok := n.(*ast.ForStmt); ok {
			loops = append(loops, loop)
		}
		if node, ok := n.(*ast.IncDecStmt); ok && file.set.Position(node.Pos()).Line == f.Line && (f.Column == 0 || file.set.Position(node.Pos()).Column == f.Column) {
			inc = node
		}
		return true
	})
	if inc == nil || inc.Tok != token.INC || !isName(inc.X, "i") {
		return false
	}
	if boundedGroupedSpareIncrement(file, fn, loops, inc) {
		return true
	}
	text := func(n ast.Node) string { return generatedNodeText(file, n) }
	if !allocationRuntimeImport(file.file) || fn.Recv != nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 2 || text(fn.Type.Params.List[0]) != "w*runtime.Writer" || text(fn.Type.Params.List[1]) != "_runtime.BitString" || fn.Type.Results == nil || text(fn.Type.Results) != "error" || len(fn.Body.List) != 2 {
		return false
	}
	loop, ok := fn.Body.List[0].(*ast.ForStmt)
	if !ok || loop.Post != inc || !zeroLoopInit(loop.Init) || len(loop.Body.List) != 1 {
		return false
	}
	cond, ok := loop.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.LSS || !isName(cond.X, "i") {
		return false
	}
	literal, ok := cond.Y.(*ast.BasicLit)
	if !ok || literal.Kind != token.INT {
		return false
	}
	width, err := strconv.ParseUint(literal.Value, 0, 64)
	if err != nil || width < 1 || width > 64 {
		return false
	}
	var label string
	ast.Inspect(loop.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 4 || text(call.Fun) != "w.ComponentError" {
			return true
		}
		if path, ok := call.Args[1].(*ast.BasicLit); ok && path.Kind == token.STRING {
			label = text(path)
		}
		return true
	})
	if label == "" {
		return false
	}
	return text(loop) == "fori:=0i<"+text(literal)+"i++{iferr:=w.WriteSpare()err!=nil{returnlocateIndex(w.ComponentError(runtime.InvalidValue,"+label+",0,err),"+label+",i)}}" && text(fn.Body.List[1]) == "returnnil"
}
