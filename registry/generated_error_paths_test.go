package registry_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGeneratedFailuresUseActivePaths(t *testing.T) {
	files := []string{"ts24008/classmark/generated.go", "ts24008/msrac/generated.go", "ts24008/msnetcap/generated.go", "ts36331/uecapability/generated.go", "ts44018/cellselection/generated.go", "ts44018/measurement/generated.go", "ts44018/restoctets/generated.go", "ts44060/ies/generated.go"}
	encoderSites, plainDecoderSites := 0, 0
	for _, name := range files {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", filepath.FromSlash(name)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
				continue
			}
			first := fn.Type.Params.List[0]
			if len(first.Names) == 0 {
				continue
			}
			receiver := first.Names[0].Name
			codec := (strings.HasPrefix(fn.Name.Name, "encode") && receiver == "w") || (strings.HasPrefix(fn.Name.Name, "decode") && receiver == "r")
			if !codec {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if sel.Sel.Name == "OperationError" {
					t.Errorf("%s %s still uses an explicit bare error path", name, fn.Name)
				}
				if sel.Sel.Name == "ComponentError" {
					encoderSites++
				}
				if sel.Sel.Name == "failure" && len(call.Args) >= 2 {
					if inner, ok := call.Args[0].(*ast.CallExpr); ok && isErrorFormat(inner) {
						plainDecoderSites++
						path, ok := call.Args[1].(*ast.CallExpr)
						if !ok || !isReaderPath(path) {
							t.Errorf("%s %s plain failure has no active reader path", name, fn.Name)
						}
					}
				}
				return true
			})
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if _, ok := n.(*ast.FuncLit); ok {
					return false
				}
				ret, ok := n.(*ast.ReturnStmt)
				if !ok || len(ret.Results) == 0 {
					return true
				}
				if call, ok := ret.Results[len(ret.Results)-1].(*ast.CallExpr); ok && isErrorFormat(call) {
					t.Errorf("%s %s returns an unlocated plain error", name, fn.Name)
				}
				return true
			})
		}
	}
	if encoderSites == 0 || plainDecoderSites == 0 {
		t.Fatalf("vacuous census: %d encoder sites, %d plain decoder sites", encoderSites, plainDecoderSites)
	}
	t.Logf("checked %d encoder failure sites and %d plain decoder failure sites", encoderSites, plainDecoderSites)
}

func isErrorFormat(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Errorf" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "fmt"
}
func isReaderPath(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ErrorPath" {
		return false
	}
	receiver, ok := sel.X.(*ast.Ident)
	return ok && receiver.Name == "r"
}

func TestGeneratedEncoderErrorFamilyInventory(t *testing.T) {
	// Each family has an executable regression in component_errors_test.go in
	// its generated package, or the public list-element regression here.
	want := map[string]bool{
		"%w: continuation count requires enclosing SI value": true, "L/H field outside 0..1": true,
		"SI message list exceeds fixed IE": true, "choice alternative mismatch": true, "choice requires exactly one alternative": true,
		"equality constraint failed": true, "excluded value": true, "extension fallback requires exactly one arm": true,
		"fallback arm mismatch": true, "interpreted field below 1": true, "invalid choice alternative": true, "invalid fallback arm": true,
		"invalid truncation point": true, "literal-zero padding requires enclosing length bound": true,
		"recursive list requires an element": true, "repeat count mismatch": true, "reserved CSN.1 alternative": true,
		"reserved Non-GSM protocol discriminator": true, "reserved container length code": true, "value reserved by source table": true,
		"zero-length SI stop record belongs in Wire": true,
		// The public measurement ambiguous-release refusal and nonzero-tail
		// controls exercise the emitted zero-tail check and its trial rejection.
		"release additions encode an all-zero tail indistinguishable from redundant legacy bitmap positions": true,
		"nonzero release tail is not redundant legacy positions":                                             true,
	}
	found := map[string]bool{}
	files, err := filepath.Glob(filepath.Join("..", "ts*", "*", "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "encode") {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isErrorFormat(call) || len(call.Args) == 0 {
					return true
				}
				text, ok := call.Args[0].(*ast.BasicLit)
				if !ok {
					return true
				}
				message, err := strconv.Unquote(text.Value)
				if err != nil {
					t.Fatal(err)
				}
				// Byte wrappers enforce definition extents outside the rewrite pass.
				if strings.HasSuffix(fn.Name.Name, "Bytes") {
					return true
				}
				found[message] = true
				if !want[message] {
					t.Errorf("uncovered encoder family %q in %s", message, fn.Name)
				}
				return true
			})
		}
	}
	for message := range want {
		if !found[message] {
			t.Errorf("stale encoder family %q", message)
		}
	}
}
