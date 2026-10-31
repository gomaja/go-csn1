package runtime

import "testing"

func TestValStringSubtractionIgnoresHyphenatedBinding(t *testing.T) {
	// No source prints this spaced form. Strings retain lexical subtraction;
	// published token arrays have already lost the source's whitespace.
	assertValPaths(t, "val(N_E - UTRAN + 1)", []string{"val", "(", "N_E", "-", "UTRAN", "+", "1", ")"},
		map[string]uint64{"N_E": 9, "UTRAN": 3}, 7,
		func(variable func(string) ExpressionValue) ExpressionValue {
			return variable("N_E").Subtract(variable("UTRAN")).Add(ExpressionConstant(1))
		})
	r, w := NewReader(nil), NewWriter()
	ctx, err := NewExpressionContext(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, codec := range []interface{ Set(string, uint64) }{r, w} {
		codec.Set("N_E", 9)
		codec.Set("UTRAN", 3)
		codec.Set("N_E-UTRAN", 2)
	}
	tokens := []string{"val", "(", "N_E", "-", "UTRAN", "+", "1", ")"}
	for _, path := range []struct {
		name string
		eval func() (int, error)
		want int
	}{
		{"reader string", func() (int, error) { return r.Eval("val(N_E - UTRAN + 1)") }, 7},
		{"writer string", func() (int, error) { return w.Eval("val(N_E - UTRAN + 1)") }, 7},
		{"reader context string", func() (int, error) { return r.EvalWithContext("val(N_E - UTRAN + 1)", ctx) }, 7},
		{"writer context string", func() (int, error) { return w.EvalWithContext("val(N_E - UTRAN + 1)", ctx) }, 7},
		{"reader published tokens", func() (int, error) { return r.EvalTokens(tokens) }, 3},
		{"writer published tokens", func() (int, error) { return w.EvalTokens(tokens) }, 3},
		{"reader context published tokens", func() (int, error) { return r.EvalTokensWithContext(tokens, ctx) }, 3},
		{"writer context published tokens", func() (int, error) { return w.EvalTokensWithContext(tokens, ctx) }, 3},
	} {
		if got, err := path.eval(); err != nil || got != path.want {
			t.Errorf("%s = %d, %v; want %d", path.name, got, err, path.want)
		}
	}
}
