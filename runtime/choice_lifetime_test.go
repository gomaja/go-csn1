package runtime

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestChoiceTrackRejectsReleasedHandle(t *testing.T) {
	r := NewReader([]byte{1})
	if err := r.Enter("a"); err != nil {
		t.Fatal(err)
	}
	var first ChoiceErrors
	arm := first.ForkScoped(r)
	err := arm.Error(InvalidValue, "first")
	first.Track(err)
	first.Release()
	var second ChoiceErrors
	defer func() {
		got := recover()
		if got != "CSN.1 choice diagnostic already consumed or released" {
			t.Errorf("Track panic = %v", got)
		}
		if second.head != nil {
			t.Error("released handle retained")
		}
	}()
	second.Track(err)
}

func TestReleasedTrialMutatorsRefuse(t *testing.T) {
	root := NewReader([]byte{0xf0, 0x0f})
	var trials ChoiceErrors
	arm := trials.ForkScoped(root)
	if _, err := arm.ReadUint(7); err != nil {
		t.Fatal(err)
	}
	arm.SetZeroExtension(true)
	trials.Release()
	// Read-only state is undefined after Release. Mutators that check identity
	// must still refuse the dead trial rather than modifying pooled storage.
	for name, call := range map[string]func(){
		"Set":            func() { arm.Set("x", 1) },
		"ForkScoped":     func() { _ = trials.ForkScoped(arm) },
		"CommitReceiver": func() { arm.Commit(root) },
		"CommitArgument": func() { root.Commit(arm) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("released trial accepted")
				}
			}()
			call()
		})
	}
}

func TestReaderAccessorLifetimeDocumentation(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "direct.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{"Position": false, "Remaining": false, "BoundedRemaining": false, "VirtualBits": false, "Wire": false, "Tail": false, "ZeroExtension": false}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Recv == nil {
			continue
		}
		star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		recv, ok := star.X.(*ast.Ident)
		if !ok || recv.Name != "Reader" {
			continue
		}
		if _, wanted := names[fn.Name.Name]; !wanted {
			continue
		}
		doc := fn.Doc.Text()
		if !strings.Contains(doc, "Release") || !strings.Contains(doc, "undefined") {
			t.Errorf("%s does not document released state", fn.Name.Name)
		}
		names[fn.Name.Name] = true
	}
	for name, found := range names {
		if !found {
			t.Errorf("missing accessor %s", name)
		}
	}
}
