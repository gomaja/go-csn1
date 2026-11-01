package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkGeneratedCoordinateArithmetic proves the local coordinate operations.
// Remainder by the nonzero constant 8 is defined for every signed int. A checked
// runtime.Reader position is at most runtime/expr.go's maxBits (1<<20), so
// adding the TS 44.060 V19.0.0 section 11.2.9d content extent 170 fits int32.
// The addition is admitted only directly after Reader.Check and a phase guard;
// no intervening statement can replace or mutate that checked reader.
func checkGeneratedCoordinateArithmetic(file citationFile, fn *ast.FuncDecl, f Finding) error {
	fail := func() error { return fmt.Errorf("generated content coordinate proof differs: %s", f.Key()) }
	var expression *ast.BinaryExpr
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		binary, ok := n.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		at := file.set.Position(binary.Pos())
		if at.Line == f.Line && (f.Column == 0 || at.Column == f.Column) && emrPrintedNode(file, binary) == f.Expression {
			expression = binary
		}
		return true
	})
	if expression == nil || f.Kind != "arithmetic" {
		return fail()
	}
	if expression.Op == token.REM {
		if divisor, ok := expression.Y.(*ast.BasicLit); ok && divisor.Kind == token.INT && divisor.Value == "8" {
			return nil
		}
		return fail()
	}
	if f.Expression != "r.Position() + 170" || expression.Op != token.ADD || fn.Recv != nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return fail()
	}
	runtimeImport := false
	for _, spec := range file.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fail()
		}
		if path == "github.com/gomaja/go-csn1/runtime" && (spec.Name == nil || spec.Name.Name == "runtime") {
			runtimeImport = true
		}
	}
	if !runtimeImport {
		return fail()
	}
	parameter := fn.Type.Params.List[0]
	if len(parameter.Names) != 1 || parameter.Names[0].Name != "r" || emrPrintedNode(file, parameter.Type) != "*runtime.Reader" || len(fn.Body.List) < 3 {
		return fail()
	}
	guard, ok := fn.Body.List[0].(*ast.IfStmt)
	if !ok || guard.Else != nil || emrPrintedNode(file, guard.Init) != "err := r.Check()" || emrPrintedNode(file, guard.Cond) != "err != nil" || len(guard.Body.List) != 1 {
		return fail()
	}
	if _, ok := guard.Body.List[0].(*ast.ReturnStmt); !ok {
		return fail()
	}
	phase, ok := fn.Body.List[1].(*ast.IfStmt)
	if !ok || phase.Init != nil || phase.Else != nil || emrPrintedNode(file, phase.Cond) != "r.Position()%8 != 6" || len(phase.Body.List) != 1 {
		return fail()
	}
	if _, ok := phase.Body.List[0].(*ast.ReturnStmt); !ok {
		return fail()
	}
	limit, ok := fn.Body.List[2].(*ast.IfStmt)
	if !ok || limit.Init != nil || limit.Else != nil || emrPrintedNode(file, limit.Cond) != "r.Remaining() > 170" || len(limit.Body.List) != 1 {
		return fail()
	}
	ret, ok := limit.Body.List[0].(*ast.ReturnStmt)
	if !ok || expression.Pos() < ret.Pos() || expression.End() > ret.End() {
		return fail()
	}
	return nil
}

// RemainingCanonicalTarget is at most 1<<20: runtime/canonical_target.go:11 `w.wire.targetBits > maxBits`.
// Translating that target by the authenticated 14-bit block origin fits int32.
// Require the query's own successful branch and its first assignment, so no
// intervening statement can replace the bounded target.
func checkGeneratedCanonicalOriginArithmetic(file citationFile, fn *ast.FuncDecl, f Finding) error {
	fail := func() error { return fmt.Errorf("generated canonical origin proof differs: %s", f.Key()) }
	if f.Kind != "arithmetic" || f.Expression != "target + 14" {
		return fail()
	}
	if len(fn.Body.List) < 2 || emrPrintedNode(file, fn.Body.List[0]) != "probe := runtime.NewWriter()" || emrPrintedNode(file, fn.Body.List[1]) != "probe.WithWire(v.Wire)" {
		return fail()
	}
	runtimeImport := false
	for _, spec := range file.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/gomaja/go-csn1/runtime" && (spec.Name == nil || spec.Name.Name == "runtime") {
			runtimeImport = true
		}
	}
	if !runtimeImport {
		return fail()
	}
	valid := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		branch, ok := n.(*ast.IfStmt)
		if !ok || branch.Init == nil || emrPrintedNode(file, branch.Init) != "target, present := probe.RemainingCanonicalTarget()" || emrPrintedNode(file, branch.Cond) != "present" || len(branch.Body.List) == 0 {
			return true
		}
		first := branch.Body.List[0]
		if emrPrintedNode(file, first) != "v = runtime.CanonicalTo(v, target+14)" {
			return true
		}
		ast.Inspect(first, func(n ast.Node) bool {
			if binary, ok := n.(*ast.BinaryExpr); ok {
				at := file.set.Position(binary.Pos())
				if at.Line == f.Line && (f.Column == 0 || at.Column == f.Column) && emrPrintedNode(file, binary) == f.Expression {
					valid = true
				}
			}
			return true
		})
		return true
	})
	if !valid {
		return fail()
	}
	return nil
}

// CheckInput rejects inputs exceeding runtime's 1<<20-bit budget. The next
// statement's short-input diagnostic adds the fixed nonnegative origin14 to
// InputBits(data), so its offset is in [14, 1048590] and fits int32. Pin both
// statements and the exact typed return: data cannot be replaced or escape
// between its check and the offset, and unrelated arithmetic gets no proof.
func checkGeneratedContentInputOffset(file citationFile, fn *ast.FuncDecl, f Finding) error {
	fail := func() error { return fmt.Errorf("generated content input offset proof differs: %s", f.Key()) }
	if f.Kind != "arithmetic" || f.Expression != "runtime.InputBits(data) + 14" || fn.Recv != nil || fn.Type.TypeParams != nil || fn.Type.Params == nil || len(fn.Body.List) < 2 {
		return fail()
	}
	if !strings.HasPrefix(fn.Name.Name, "Decode") && (!strings.HasPrefix(fn.Name.Name, "decode") || !strings.HasSuffix(fn.Name.Name, "Bytes")) {
		return fail()
	}
	text := func(n ast.Node) string { return generatedNodeText(file, n) }
	params := fn.Type.Params.List
	if len(params) < 1 || len(params) > 2 || text(params[0]) != "data[]byte" || (len(params) == 2 && text(params[1]) != "collector*runtime.Collector") {
		return fail()
	}
	slice, ok := params[0].Type.(*ast.ArrayType)
	if !ok || slice.Len != nil {
		return fail()
	}
	element, ok := slice.Elt.(*ast.Ident)
	if !ok || element.Name != "byte" || element.Obj != nil {
		return fail()
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 || text(fn.Type.Results.List[1]) != "error" {
		return fail()
	}
	result := fn.Type.Results.List[0]
	decoded, ok := result.Type.(*ast.IndexExpr)
	if !ok || len(result.Names) != 0 || text(decoded.X) != "runtime.Decoded" {
		return fail()
	}
	value, ok := decoded.Index.(*ast.Ident)
	if !ok {
		return fail()
	}
	runtimeImports := 0
	for _, spec := range file.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fail()
		}
		alias := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		if alias == "len" || alias == "byte" {
			return fail()
		}
		if alias == "runtime" {
			if path != "github.com/gomaja/go-csn1/runtime" {
				return fail()
			}
			runtimeImports++
		}
	}
	if runtimeImports != 1 {
		return fail()
	}
	guard, ok := fn.Body.List[0].(*ast.IfStmt)
	if !ok || guard.Else != nil || guard.Init == nil || guard.Cond == nil || text(guard.Init) != "err:=runtime.CheckInput(data)" || text(guard.Cond) != "err!=nil" || len(guard.Body.List) != 1 {
		return fail()
	}
	zero := "runtime.Decoded[" + value.Name + "]{}"
	if text(guard.Body.List[0]) != "return"+zero+",err" {
		return fail()
	}
	short, ok := fn.Body.List[1].(*ast.IfStmt)
	if !ok || short.Init != nil || short.Else != nil || len(short.Body.List) != 1 {
		return fail()
	}
	condition, ok := short.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.LSS {
		return fail()
	}
	length, ok := condition.X.(*ast.CallExpr)
	if !ok || len(length.Args) != 1 || length.Ellipsis.IsValid() || text(length.Args[0]) != "data" {
		return fail()
	}
	builtin, ok := length.Fun.(*ast.Ident)
	if !ok || builtin.Name != "len" || builtin.Obj != nil {
		return fail()
	}
	minimum, ok := condition.Y.(*ast.BasicLit)
	if !ok || minimum.Kind != token.INT {
		return fail()
	}
	n, err := strconv.ParseInt(minimum.Value, 0, 64)
	if err != nil || n <= 0 {
		return fail()
	}
	ret, ok := short.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 || text(ret.Results[0]) != zero {
		return fail()
	}
	address, ok := ret.Results[1].(*ast.UnaryExpr)
	if !ok || address.Op != token.AND {
		return fail()
	}
	diagnostic, ok := address.X.(*ast.CompositeLit)
	if !ok || text(diagnostic.Type) != "runtime.DecodeError" || len(diagnostic.Elts) != 3 {
		return fail()
	}
	var offset *ast.BinaryExpr
	for i, field := range diagnostic.Elts {
		kv, ok := field.(*ast.KeyValueExpr)
		if !ok {
			return fail()
		}
		switch i {
		case 0:
			if text(kv.Key) != "Kind" || text(kv.Value) != "runtime.Truncated" {
				return fail()
			}
		case 1:
			if text(kv.Key) != "Offset" {
				return fail()
			}
			offset, ok = kv.Value.(*ast.BinaryExpr)
			if !ok || offset.Op != token.ADD || text(offset) != "runtime.InputBits(data)+14" {
				return fail()
			}
		case 2:
			detail, ok := kv.Value.(*ast.BasicLit)
			if text(kv.Key) != "Detail" || !ok || detail.Kind != token.STRING {
				return fail()
			}
		}
	}
	trusted := true
	for _, statement := range fn.Body.List[:2] {
		ast.Inspect(statement, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok && id.Name == "runtime" && id.Obj != nil {
				trusted = false
			}
			return true
		})
	}
	if !trusted || offset == nil {
		return fail()
	}
	at := file.set.Position(offset.Pos())
	if at.Line != f.Line || (f.Column != 0 && at.Column != f.Column) {
		return fail()
	}
	return nil
}
