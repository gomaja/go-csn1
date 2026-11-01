package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkGeneratedAllocationArithmetic proves the generated population conversion
// and bitmap accumulator locally. TS 44.060 V19.0.0 §12.42 pp23778–23792
// defines N and M from the source allocation. A nonnegative host int widens
// exactly to uint64. The bitmap loop appends at most 64 individually masked
// bits to its initially zero uint64 accumulator.
func checkGeneratedAllocationArithmetic(file citationFile, fn *ast.FuncDecl, finding Finding) (bool, error) {
	text := func(n ast.Node) string {
		a, b := file.set.Position(n.Pos()).Offset, file.set.Position(n.End()).Offset
		return strings.ReplaceAll(generatedWhitespace.ReplaceAllString(string(file.source[a:b]), ""), ";", "")
	}
	fail := func() (bool, error) {
		return true, fmt.Errorf("generated allocation arithmetic differs: %s", finding.Key())
	}
	if finding.Expression == "uint64(value)" {
		var block *ast.BlockStmt
		var at int
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			b, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i, s := range b.List {
				if file.set.Position(s.Pos()).Line <= finding.Line && file.set.Position(s.End()).Line >= finding.Line {
					block = b
					at = i
				}
			}
			return true
		})
		if block == nil || at < 1 {
			return false, nil
		}
		guard, ok := block.List[at-1].(*ast.IfStmt)
		if !ok || guard.Init != nil || guard.Else != nil || !allocationReturns(guard.Body) {
			return false, nil
		}
		comparison, ok := guard.Cond.(*ast.BinaryExpr)
		if !ok || comparison.Op != token.LSS {
			return false, nil
		}
		operand, ok := comparison.X.(*ast.Ident)
		if !ok || operand.Name != "value" {
			return false, nil
		}
		minimum, ok := comparison.Y.(*ast.BasicLit)
		if !ok || minimum.Kind != token.INT {
			return fail()
		}
		n, err := strconv.ParseUint(minimum.Value, 0, 63)
		if err != nil || n > 1<<20 {
			return fail()
		}
		converted := false
		ast.Inspect(block.List[at], func(n ast.Node) bool {
			cast, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := cast.Fun.(*ast.Ident)
			if !ok || name.Name != "uint64" || name.Obj != nil || len(cast.Args) != 1 {
				return true
			}
			value, ok := cast.Args[0].(*ast.Ident)
			if ok && value.Name == "value" && value.Obj != nil && value.Obj == operand.Obj {
				converted = true
			}
			return true
		})
		if !converted || !allocationPopulationInt(operand, fn, text) || (!strings.HasPrefix(text(block.List[at]), "r.Set(") && !strings.HasPrefix(text(block.List[at]), "w.Set(")) {
			return fail()
		}
		// Set must be the entire statement, so no assignment, closure or address
		// escape can intervene between the rejecting guard and the conversion.
		stmt, ok := block.List[at].(*ast.ExprStmt)
		if !ok {
			return fail()
		}
		call, ok := stmt.X.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 || text(call.Args[1]) != "uint64(value)" {
			return fail()
		}
		key, literal := call.Args[0].(*ast.BasicLit)
		if !literal || key.Kind != token.STRING || !allocationRuntimeImport(file.file) {
			return fail()
		}
		return true, nil
	}
	var parent *ast.BlockStmt
	var index int
	var loop *ast.ForStmt
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		b, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, s := range b.List {
			f, ok := s.(*ast.ForStmt)
			if ok && file.set.Position(f.Pos()).Line <= finding.Line && file.set.Position(f.End()).Line >= finding.Line {
				loop = f
				parent = b
				index = i
			}
		}
		return true
	})
	if loop == nil || !strings.Contains(text(loop), "bitmap=") {
		return false, nil
	}
	const shape = "forbit:=0bit<v.BitLengthbit++{bitmap=bitmap<<1|uint64((v.Bytes[bit/8]>>uint(7-bit%8))&1)}"
	if text(loop) != shape || index != 2 || text(parent.List[index-1]) != "varbitmapuint64" {
		return fail()
	}
	guard, ok := parent.List[index-2].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || text(guard.Cond) != "v.BitLength>64" || !allocationReturns(guard.Body) {
		return fail()
	}
	// The successful expression body validates the BitString's byte storage.
	// Pin its local declaration and the immediate return-on-error call site.
	if !allocationRuntimeImport(file.file) || !allocationBitStringValidated(fn, parent, text) {
		return fail()
	}
	return true, nil
}

func allocationReturns(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) != 1 {
		return false
	}
	_, ok := body.List[0].(*ast.ReturnStmt)
	return ok
}

func allocationBitStringValidated(fn *ast.FuncDecl, parent *ast.BlockStmt, text func(ast.Node) string) bool {
	before := parent.Pos()
	owner := func(s ast.Stmt) bool {
		v, ok := s.(*ast.IfStmt)
		return ok && v.Body == parent && v.Else == nil && v.Init != nil && !hasNameWrite(v.Init, "v") && !hasNameWrite(v.Cond, "v")
	}
	expected := fn.Name.Name + "ExpressionBody"
	for i, s := range fn.Body.List {
		if s.Pos() >= before {
			break
		}
		if i+1 >= len(fn.Body.List) {
			continue
		}
		if a, ok := s.(*ast.AssignStmt); ok && len(a.Lhs) == 2 && text(a.Lhs[0]) == "v" && len(a.Rhs) == 1 {
			call, ok := a.Rhs[0].(*ast.CallExpr)
			if !ok || text(call.Fun) != expected || text(call) != (expected+"(r)") {
				continue
			}
			guard, ok := fn.Body.List[i+1].(*ast.IfStmt)
			if !ok || guard.Init != nil || guard.Else != nil || text(guard.Cond) != "err!=nil" || !allocationReturns(guard.Body) || i+2 >= len(fn.Body.List) || !owner(fn.Body.List[i+2]) {
				continue
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Obj == nil {
				return false
			}
			body, ok := id.Obj.Decl.(*ast.FuncDecl)
			if !ok {
				return false
			}
			return allocationBitStringBody(body, true, text)
		}
		guard, ok := s.(*ast.IfStmt)
		if !ok || guard.Init == nil || guard.Else != nil || text(guard.Cond) != "err!=nil" || !allocationReturns(guard.Body) || !owner(fn.Body.List[i+1]) {
			continue
		}
		a, ok := guard.Init.(*ast.AssignStmt)
		if !ok || len(a.Rhs) != 1 || text(a.Rhs[0]) != expected+"(w,v)" {
			continue
		}
		call, ok := a.Rhs[0].(*ast.CallExpr)
		if !ok {
			return false
		}
		id, ok := call.Fun.(*ast.Ident)
		if !ok || id.Obj == nil {
			return false
		}
		body, ok := id.Obj.Decl.(*ast.FuncDecl)
		if !ok {
			return false
		}
		return allocationBitStringBody(body, false, text)
	}
	return false
}

func allocationRuntimeImport(file *ast.File) bool {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/gomaja/go-csn1/runtime" && (spec.Name == nil || spec.Name.Name == "runtime") {
			return true
		}
	}
	return false
}

func allocationPopulationInt(value *ast.Ident, fn *ast.FuncDecl, text func(ast.Node) string) bool {
	if value.Obj == nil {
		return false
	}
	switch declaration := value.Obj.Decl.(type) {
	case *ast.ValueSpec:
		typ, ok := declaration.Type.(*ast.Ident)
		return ok && typ.Name == "int" && typ.Obj == nil && len(declaration.Names) == 1 && len(declaration.Values) == 0
	case *ast.AssignStmt:
		if declaration.Tok != token.DEFINE || len(declaration.Lhs) != 2 || len(declaration.Rhs) != 1 || !isName(declaration.Lhs[0], "value") {
			return false
		}
		call, ok := declaration.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ExpressionResultWithContext" {
			return false
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || fn.Type.Params == nil {
			return false
		}
		for _, parameter := range fn.Type.Params.List {
			if len(parameter.Names) == 1 && parameter.Names[0].Obj == receiver.Obj && receiver.Obj != nil {
				return receiver.Name == "r" && text(parameter.Type) == "*runtime.Reader" || receiver.Name == "w" && text(parameter.Type) == "*runtime.Writer"
			}
		}
	}
	return false
}

// Pin every success path in the local BitString body: only the final read or
// write validates the original value; an earlier success or a substituted value
// cannot authenticate storage used by the wrapper's subsequent bitmap loop.
func allocationBitStringBody(fn *ast.FuncDecl, reader bool, text func(ast.Node) string) bool {
	if fn.Body == nil || fn.Type.Params == nil || fn.Type.Results == nil {
		return false
	}
	receiver, zero, tail := "w", "", "returnnil"
	expectedParameters := []string{"w*runtime.Writer", "vruntime.BitString"}
	expectedResults := []string{"error"}
	if reader {
		receiver = "r"
		zero = "runtime.BitString{},"
		tail = "returnruntime.BitString(v),nil"
		expectedParameters = []string{"r*runtime.Reader"}
		expectedResults = []string{"runtime.BitString", "error"}
	}
	if len(fn.Type.Params.List) != len(expectedParameters) || len(fn.Type.Results.List) != len(expectedResults) {
		return false
	}
	for i, p := range fn.Type.Params.List {
		if text(p) != expectedParameters[i] {
			return false
		}
	}
	for i, p := range fn.Type.Results.List {
		if text(p) != expectedResults[i] {
			return false
		}
	}
	statements := fn.Body.List
	length := 4
	if reader {
		length = 5
	}
	if len(statements) == length+2 {
		enter, ok := statements[0].(*ast.IfStmt)
		if !ok || enter.Init == nil || enter.Else != nil || text(enter.Cond) != "err!=nil" || text(enter.Body) != "{return"+zero+"err}" {
			return false
		}
		assign, ok := enter.Init.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			return false
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || text(call.Fun) != receiver+".Enter" {
			return false
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return false
		}
		if text(statements[1]) != "defer"+receiver+".Leave()" {
			return false
		}
		statements = statements[2:]
	}
	if len(statements) != length {
		return false
	}
	width, ok := statements[0].(*ast.AssignStmt)
	if !ok || width.Tok != token.DEFINE || len(width.Lhs) != 2 || text(width.Lhs[0]) != "width" || text(width.Lhs[1]) != "err" || len(width.Rhs) != 1 {
		return false
	}
	expression, ok := width.Rhs[0].(*ast.CallExpr)
	if !ok || len(expression.Args) != 1 || text(expression.Fun) != receiver+".ExpressionResultWithContext" {
		return false
	}
	operand, ok := expression.Args[0].(*ast.CallExpr)
	if !ok || len(operand.Args) != 0 {
		return false
	}
	selector, ok := operand.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Operand" {
		return false
	}
	variable, ok := selector.X.(*ast.CallExpr)
	if !ok || len(variable.Args) != 1 || text(variable.Fun) != receiver+".ExpressionVariable" {
		return false
	}
	literal, ok := variable.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	if text(statements[1]) != "iferr!=nil{return"+zero+"err}" || text(statements[len(statements)-1]) != tail {
		return false
	}
	if reader {
		return text(statements[2]) == "v,err:=r.ReadBitString(width)" && text(statements[3]) == "iferr!=nil{returnruntime.BitString{},err}"
	}
	return text(statements[2]) == "iferr:=w.WriteBitString(v,width)err!=nil{returnerr}"
}
