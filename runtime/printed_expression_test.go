package runtime

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestPrintedLabelExpressionParity(t *testing.T) {
	ctx, _, _ := sourceFunctions(t)
	for _, label := range []string{"EPD A/N Extension length index", "a5/1"} {
		t.Run(label, func(t *testing.T) {
			// TS 44.060 V19.0.0 §11.2.6a p14465 prints the full EPD A/N label;
			// table 11.2.6a.1a gives 3+65 with DLDC_configuration=1.
			raw := "expanded_EPDAN_extension_length(val(" + label + "))"
			if err := ValidateExpression(raw); err != nil {
				t.Fatal(err)
			}
			tokens, err := expressionTokens(raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, halves := range []bool{false, true} {
				r, w := NewReader(nil), NewWriter()
				for _, codec := range []interface{ Set(string, uint64) }{r, w} {
					codec.Set(label, 3)
					codec.Set("DLDC_configuration", 1)
					if halves {
						codec.Set("EPD A", 8)
						codec.Set("N Extension length index", 2)
						codec.Set("a5", 8)
					}
				}
				for _, path := range []struct {
					name string
					eval func() (int, error)
				}{
					{"reader string", func() (int, error) { return r.EvalWithContext(raw, ctx) }},
					{"writer string", func() (int, error) { return w.EvalWithContext(raw, ctx) }},
					{"reader tokens", func() (int, error) { return r.EvalTokensWithContext(tokens, ctx) }},
					{"writer tokens", func() (int, error) { return w.EvalTokensWithContext(tokens, ctx) }},
					{"reader compiled", func() (int, error) {
						return r.ExpressionResultWithContext(r.ExpressionFunction("expanded_EPDAN_extension_length", r.ExpressionVariable(label), ctx))
					}},
					{"writer compiled", func() (int, error) {
						return w.ExpressionResultWithContext(w.ExpressionFunction("expanded_EPDAN_extension_length", w.ExpressionVariable(label), ctx))
					}},
				} {
					if n, e := path.eval(); e != nil || n != 68 {
						t.Errorf("%s halves=%v: %d, %v; want 68", path.name, halves, n, e)
					}
				}
			}
		})
	}
}

// Each paragraph is in the authenticated V19.0.0 source of the named standard.
func TestPrintedValArguments(t *testing.T) {
	for _, tc := range []struct {
		source, argument, label string
		add                     bool
	}{
		{"TS 44.060 §§11.2.3a, 11.2.9 pp13713,15472", "N_3G + 1", "N_3G", true},
		{"TS 44.060 §11.2.9d p16077; TS 44.018 §9.1.55 p11350", "BITMAP_LENGTH + 1", "BITMAP_LENGTH", true},
		{"TS 44.060 §12.48a.6 p25461", "PCID_Pattern_length + 1", "PCID_Pattern_length", true},
		{"TS 44.060 §12.48a.6 p25480", "PSC_Pattern_length + 1", "PSC_Pattern_length", true},
		{"TS 44.060 §§11.2.3a,11.2.9,11.2.9d pp13726,15479,16103; TS 44.018 §9.1.55 p11378", "N_E-UTRAN + 1", "N_E-UTRAN", true},
		{"TS 44.018 §§9.1.43p,9.1.43q,9.1.43r pp9756,9885,10059", "NumberOfOctets + 1", "NumberOfOctets", true},
		{"TS 44.018 §9.1.63 p11929", "Number of IMSI Digits + 1", "Number of IMSI Digits", true},
		{"TS 44.060 §11.2.6a p14465", "EPD A/N Extension length index", "EPD A/N Extension length index", false},
		{"TS 44.060 §11.2.6e p14681", "EPD A/N Type 2 Extension length", "EPD A/N Type 2 Extension length", false},
		{"synthetic cipher label", "a5/1", "a5/1", false},
	} {
		t.Run(tc.argument, func(t *testing.T) {
			t.Log(tc.source)
			raw := "val(" + tc.argument + ")"
			tokens, err := expressionTokens(raw)
			if err != nil {
				t.Fatal(err)
			}
			wantTokens := []string{"val", "("}
			// Words may be distinct tokens, but punctuation stays inside its label.
			wantTokens = append(wantTokens, strings.Fields(tc.label)...)
			if tc.add {
				wantTokens = append(wantTokens, "+", "1")
			}
			wantTokens = append(wantTokens, ")")
			if !reflect.DeepEqual(tokens, wantTokens) {
				t.Errorf("tokens = %q; want %q", tokens, wantTokens)
			}
			want := 7
			if tc.add {
				want++
			}
			assertValPaths(t, raw, tokens, map[string]uint64{tc.label: 7, "N_E": 19, "UTRAN": 3, "EPD A": 20, "N Extension length index": 2, "a5": 9}, want, func(variable func(string) ExpressionValue) ExpressionValue {
				v := variable(tc.label)
				if tc.add {
					v = v.Add(ExpressionConstant(1))
				}
				return v
			})
		})
	}
}

func assertValPaths(t *testing.T, raw string, tokens []string, bindings map[string]uint64, want int, compiled func(func(string) ExpressionValue) ExpressionValue) {
	t.Helper()
	if err := ValidateExpression(raw); err != nil {
		t.Fatal(err)
	}
	r, w := NewReader(nil), NewWriter()
	ctx, err := NewExpressionContext(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for label, n := range bindings {
		r.Set(label, n)
		w.Set(label, n)
	}
	for _, path := range []struct {
		name string
		eval func() (int, error)
	}{
		{"reader string", func() (int, error) { return r.Eval(raw) }},
		{"writer string", func() (int, error) { return w.Eval(raw) }},
		{"reader tokens", func() (int, error) { return r.EvalTokens(tokens) }},
		{"writer tokens", func() (int, error) { return w.EvalTokens(tokens) }},
		{"reader context string", func() (int, error) { return r.EvalWithContext(raw, ctx) }},
		{"writer context string", func() (int, error) { return w.EvalWithContext(raw, ctx) }},
		{"reader context tokens", func() (int, error) { return r.EvalTokensWithContext(tokens, ctx) }},
		{"writer context tokens", func() (int, error) { return w.EvalTokensWithContext(tokens, ctx) }},
		{"reader compiled", func() (int, error) { return r.ExpressionResult(compiled(r.ExpressionVariable)) }},
		{"writer compiled", func() (int, error) { return w.ExpressionResult(compiled(w.ExpressionVariable)) }},
	} {
		if n, err := path.eval(); err != nil || n != want {
			t.Errorf("%s: %d, %v; want %d", path.name, n, err, want)
		}
	}
}

func TestValOperatorSpacing(t *testing.T) {
	for _, op := range []string{"+", "-", "*", "/"} {
		for _, form := range []string{"a%sb", "a %sb", "a%s b"} {
			label := fmt.Sprintf(form, op)
			raw := "val(" + label + ")"
			tokens, err := expressionTokens(raw)
			if err != nil {
				t.Fatal(err)
			}
			assertValPaths(t, raw, tokens, map[string]uint64{label: 7, "a": 8, "b": 2}, 7, func(variable func(string) ExpressionValue) ExpressionValue { return variable(label) })
		}
		for _, space := range []string{" ", "\t", "\n"} {
			raw := "val(a" + space + op + space + "b)"
			tokens, err := expressionTokens(raw)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]int{"+": 10, "-": 6, "*": 16, "/": 4}[op]
			assertValPaths(t, raw, tokens, map[string]uint64{"a": 8, "b": 2}, want, func(variable func(string) ExpressionValue) ExpressionValue {
				a, b := variable("a"), variable("b")
				switch op {
				case "+":
					return a.Add(b)
				case "-":
					return a.Subtract(b)
				case "*":
					return a.Multiply(b)
				default:
					return a.Divide(b)
				}
			})
		}
	}
}

func TestExpressionOperatorsOutsideVal(t *testing.T) {
	for _, raw := range []string{"a/b", "val(a)/val(b)", "(1+val(a))/val(b)"} {
		want := 4
		tokens, err := expressionTokens(raw)
		if err != nil {
			t.Fatal(err)
		}
		assertValPaths(t, raw, tokens, map[string]uint64{"a": 8, "b": 2, "a/b": 7}, want, func(variable func(string) ExpressionValue) ExpressionValue {
			a := variable("a")
			if raw == "(1+val(a))/val(b)" {
				a = ExpressionConstant(1).Add(a)
			}
			return a.Divide(variable("b"))
		})
	}
}

func TestPunctuatedValLabelsStayUnresolvedWithoutTheirBinding(t *testing.T) {
	for _, label := range []string{"a/b", "x-y", "N_E-UTRAN+1", "a*b"} {
		raw := "val(" + label + ")"
		if err := ValidateExpression(raw); err != nil {
			t.Fatal(err)
		}
		r := NewReader(nil)
		r.Set("a", 8)
		r.Set("b", 2)
		r.Set("x", 8)
		r.Set("y", 2)
		r.Set("N_E-UTRAN", 8)
		if n, err := r.Eval(raw); err == nil {
			t.Errorf("%s resolved to %d without its label binding", raw, n)
		}
	}
}

func TestPunctuatedValLabelTokenGrowth(t *testing.T) {
	for _, size := range []int{32, 256, 2048} {
		raw := "val(" + strings.Repeat("A/N ", size) + "end)"
		if n := testing.AllocsPerRun(5, func() {
			if err := ValidateExpression(raw); err != nil {
				panic(err)
			}
		}); n > 32 {
			t.Fatalf("%d words: %g allocations; want at most 32", size, n)
		}
	}
}
