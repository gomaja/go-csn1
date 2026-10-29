package runtime

import "testing"

func TestCanonicalBindingsMatchNamedBindings(t *testing.T) {
	for _, name := range []string{"", "a b", "value", "\x00COUNT", "élan", "σ"} {
		r, ordinary := NewReader(nil), NewReader(nil)
		var scope ChoiceErrors
		for _, trial := range []bool{false, true} {
			if trial {
				r, ordinary = scope.ForkScoped(r), scope.ForkScoped(ordinary)
			}
			r.SetCanonical(name, 7)
			ordinary.Set(name, 7)
			a, aok := r.binding(name)
			b, bok := ordinary.binding(name)
			if a != b || aok != bok || !aok {
				t.Fatalf("%q: trial %v: %d %v != %d %v", name, trial, a, aok, b, bok)
			}
		}
		scope.Release()
	}
}

func BenchmarkReaderCanonicalBinding(b *testing.B) {
	r := NewReader(nil)
	const name = "enhancedmeasurementreportbitmaplengthchoicebitmaplengthbitmaplength"
	b.ReportAllocs()
	for b.Loop() {
		r.SetCanonical(name, 7)
	}
}

func TestCanonicalBindingsKeepForksIndependent(t *testing.T) {
	r := NewReader(nil)
	r.SetCanonical("value", 1)
	fork := r.Fork()
	fork.SetCanonical("value", 2)
	if got, err := r.Eval("value"); err != nil || got != 1 {
		t.Fatalf("fork changed parent: %d %v", got, err)
	}
	r.SetCanonical("value", 3)
	if got, err := fork.Eval("value"); err != nil || got != 2 {
		t.Fatalf("parent changed fork: %d %v", got, err)
	}
}

func TestCanonicalBindingsRejectCopiedTrial(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	arm := trials.ForkScoped(r)
	copied := *arm
	defer func() {
		if recover() == nil {
			t.Fatal("copied trial accepted")
		}
	}()
	copied.SetCanonical("value", 2)
}
