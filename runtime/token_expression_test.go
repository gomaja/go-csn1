package runtime

import (
	"reflect"
	"testing"
)

func TestTokenWidthKeepsDynamicLabels(t *testing.T) {
	tokens := []string{"val", "(", "Extension", "Length", ")", "+", "1"}
	for _, bindings := range []map[string]uint64{
		nil, {"Extension": 1}, {"Extension": 1, "Extension Length": 3},
		{"Extension Length": 3, "extensionlength": 4},
		{"Extension L ength": 3, "Extension Le ngth": 4},
		{"Extension": 1, "Extension L ength": 3, "Extension Le ngth": 4},
		{"Extension Length": 1 << 63},
	} {
		r := NewReader(nil)
		for k, v := range bindings {
			r.Set(k, v)
		}
		want, we := r.Eval("val(Extension Length) + 1")
		got, ge := r.EvalTokens(tokens)
		if got != want || !reflect.DeepEqual(ge, we) {
			t.Fatalf("bindings %v: (%d,%v) want (%d,%v)", bindings, got, ge, want, we)
		}
		var c ChoiceErrors
		arm := c.ForkScoped(r)
		arm.Set("Extension Length", 5)
		want, we = arm.Eval("val(Extension Length) + 1")
		got, ge = arm.EvalTokens(tokens)
		if got != want || !reflect.DeepEqual(c.Keep(arm, ge), c.Keep(arm, we)) {
			t.Fatal("trial overlay diverged")
		}
		c.Release()
	}
}

func TestTokenWidthAvoidsLexicalAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("pool allocation counts vary under race")
	}
	r := NewReader(nil)
	r.Set("Extension Length", 3)
	tokens := []string{"val", "(", "Extension", "Length", ")", "+", "1"}
	text := testing.AllocsPerRun(100, func() {
		if n, err := r.Eval("val(Extension Length) + 1"); err != nil || n != 4 {
			panic("text")
		}
	})
	parsed := testing.AllocsPerRun(100, func() {
		if n, err := r.EvalTokens(tokens); err != nil || n != 4 {
			panic("tokens")
		}
	})
	if parsed >= text {
		t.Fatalf("token evaluation allocates %v; text %v", parsed, text)
	}
	t.Logf("text %v allocations; tokens %v", text, parsed)
}
