package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkGeneratedWholeOctetRemainder pins the complete emitted octet writer.
// The canonical minimum respects an active containing limit. Received metadata
// is checked in [0, 1<<20], with position in [0, bits], before subtraction.
// PushLimit validates the resulting complete-octet bound before it is used;
// remaining/8*8 is at most 1<<20 and leaves at most seven bits for padding.
// The exact body forbids intervening writes, aliases and operand reassignment,
// requires the value to fill that bound, and restores the containing limit.
func checkGeneratedWholeOctetRemainder(file citationFile, fn *ast.FuncDecl, finding Finding) error {
	fail := func() error { return fmt.Errorf("generated whole-octet remainder proof differs: %s", finding.Key()) }
	// Authenticate the methods whose numeric bounds this proof relies on.
	trustedRuntime := false
	for _, spec := range file.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/gomaja/go-csn1/runtime" && (spec.Name == nil || spec.Name.Name == "runtime") {
			trustedRuntime = true
		}
	}
	if !trustedRuntime {
		return fail()
	}
	text := func(node ast.Node) string {
		start, end := file.set.Position(node.Pos()).Offset, file.set.Position(node.End()).Offset
		return strings.ReplaceAll(generatedWhitespace.ReplaceAllString(string(file.source[start:end]), ""), ";", "")
	}
	if fn.Recv != nil || fn.Type.Params == nil || len(fn.Type.Params.List) != 2 || fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return fail()
	}
	if text(fn.Type.Params.List[0]) != "w*runtime.Writer" || text(fn.Type.Params.List[1]) != "v[]uint8" || text(fn.Type.Results.List[0]) != "error" {
		return fail()
	}
	var path, child string
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && isName(selector.X, "w") && selector.Sel.Name == "ComponentError" && len(call.Args) == 4 {
			if literal, ok := call.Args[1].(*ast.BasicLit); ok && literal.Kind == token.STRING {
				path = text(literal)
			}
		}
		if id, ok := call.Fun.(*ast.Ident); ok && strings.HasPrefix(id.Name, "encode") && len(call.Args) == 2 && isName(call.Args[0], "w") && isName(call.Args[1], "element") {
			child = id.Name
		}
		return true
	})
	if path == "" || child == "" {
		return fail()
	}
	want := `{remaining:=w.RemainingLimit()
 if target,present:=w.RemainingCanonicalTarget();present&&(remaining<0||target<remaining){remaining=target}
 if remaining<0{if bits,received:=w.ReceivedBits();received{if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,` + path + `,0,fmt.Errorf("invalid received whole-octet remainder extent"))};remaining=bits-w.Position()}}
 if remaining<0{return fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired)}
 oldBound,err:=w.PushLimit(remaining/8*8);if err!=nil{return err}
 if len(v)!=remaining/8{return w.LengthBoundError("whole-octet remainder must fill every complete octet")}
 for _,element:=range v{if err:=` + child + `(w,element);err!=nil{return err}}
 return w.PopLimit(oldBound)}`
	compact := strings.ReplaceAll(generatedWhitespace.ReplaceAllString(want, ""), ";", "")
	// Source emission returns the context error directly; the location pass
	// wraps exactly that return using the same authenticated component path.
	contextError := generatedWhitespace.ReplaceAllString(`fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired)`, "")
	located := strings.ReplaceAll(compact, contextError, `w.ComponentError(runtime.InvalidValue,`+path+`,0,`+contextError+`)`)
	if body := text(fn.Body); body != compact && body != located {
		return fail()
	}
	// Check the actual occurrence, so a later, unrelated subtraction cannot
	// inherit the proof from the guarded prefix.
	proven := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		binary, ok := node.(*ast.BinaryExpr)
		if !ok || text(binary) != strings.ReplaceAll(finding.Expression, " ", "") {
			return true
		}
		at := file.set.Position(binary.Pos())
		proven = at.Line == finding.Line && (finding.Column == 0 || at.Column == finding.Column)
		return true
	})
	if !proven {
		return fail()
	}
	return nil
}
