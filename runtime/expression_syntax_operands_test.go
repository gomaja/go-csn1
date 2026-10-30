package runtime

import (
	"math/rand"
	"strings"
	"testing"
)

func TestValidateExpressionSubtractedCalls(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int
	}{
		{"x - val(N)", 3},
		{"val(x - val(N))", 3},
		{"x-val(N)", 3},
		{"N_E-UTRAN Count - val(N)", 7},
		{"val(N_E-UTRAN Count - val(x - val(N)))", 6},
		{"x - p(0)", 5},
		{"x - q(0)", 5},
		{"x - (val(N))", 3},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			w := NewWriter()
			w.Set("x", 5)
			w.Set("N", 2)
			w.Set("N_E-UTRAN Count", 9)
			raw := tc.raw
			if got, err := w.Eval(raw); err != nil || got != tc.want {
				t.Fatalf("Eval(%q) = %d, %v; want %d", raw, got, err, tc.want)
			}
			if err := ValidateExpression(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestValidateExpressionRefusesUnsupportedSourceCalls(t *testing.T) {
	// Keep the expressions rejected by the compiler's generation boundary.
	for _, raw := range []string{
		// TS 44.060 V19.0.0 §11.2.21.
		"1+max(val(CELL_PARAMS_POINTER))",
		// TS 44.060 V19.0.0 §11.2.6a.
		"expanded_EPDAN_extension_length(val(EPD A/N Extension length index))",
		// TS 44.018 V19.0.0 §§9.1.49a and 9.1.54.
		"8 * val(talker identity length",
		"q(NR_OF_TDD_CELLS",
		"val(", "x -", "x - val(", "x - val()", "x - unknown(N)",
		"val(x - val(N)", "x + * val(N)", "x - (N", "x(N)",
	} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateExpression(raw); err == nil {
				t.Fatalf("accepted %q", raw)
			}
		})
	}
}

func TestValidateExpressionPreservesEvaluationBindings(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		bindings  map[string]uint64
		want      int
		failure   string
	}{
		{"subtraction", "x-y", map[string]uint64{"x": 5, "y": 2}, 3, ""},
		{"longest label", "x-y", map[string]uint64{"x": 5, "y": 2, "x-y": 7}, 7, ""},
		{"multiword longest label", "N_E-UTRAN Count - val(N)", map[string]uint64{"N_E": 8, "N_E-UTRAN Count": 9, "N": 2}, 7, ""},
		{"compact label", "abc - val(N)", map[string]uint64{"a bc": 5, "N": 2}, 3, ""},
		{"ambiguous label", "abc - val(N)", map[string]uint64{"a bc": 5, "ab c": 6, "N": 2}, 0, "ambiguous"},
		{"exact label", "abc - val(N)", map[string]uint64{"a bc": 5, "ab c": 6, "abc": 9, "N": 2}, 7, ""},
		{"unbound label", "x - val(N)", map[string]uint64{"N": 2}, 0, "unresolved"},
		{"bound call prefix", "x - val(N)", map[string]uint64{"x": 5, "N": 2, "x-val": 7}, 0, "unexpected expression token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateExpression(tc.raw); err != nil {
				t.Fatal(err)
			}
			w, r := NewWriter(), NewReader(nil)
			for name, value := range tc.bindings {
				w.Set(name, value)
				r.Set(name, value)
			}
			tokens, err := expressionTokens(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			for _, evaluator := range []struct {
				name string
				eval func() (int, error)
			}{
				{"Writer.Eval", func() (int, error) { return w.Eval(tc.raw) }},
				{"Writer.EvalTokens", func() (int, error) { return w.EvalTokens(tokens) }},
				{"Reader.Eval", func() (int, error) { return r.Eval(tc.raw) }},
				{"Reader.EvalTokens", func() (int, error) { return r.EvalTokens(tokens) }},
			} {
				got, err := evaluator.eval()
				if tc.failure != "" {
					if err == nil || !strings.Contains(err.Error(), tc.failure) {
						t.Errorf("%s = %d, %v; want %q", evaluator.name, got, err, tc.failure)
					}
				} else if err != nil || got != tc.want {
					t.Errorf("%s = %d, %v; want %d", evaluator.name, got, err, tc.want)
				}
			}
		})
	}
}

func FuzzValidateEvaluableExpression(f *testing.F) {
	for seed := int64(0); seed < 128; seed++ {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		rng := rand.New(rand.NewSource(seed))
		labels := []string{"x", "N", "N_E-UTRAN", "Extension Length", "x-y z", "alpha-beta gamma", "a bc", "ab c", "abc"}
		w := NewWriter()
		for _, label := range labels {
			if rng.Intn(4) != 0 {
				w.Set(label, uint64(rng.Intn(12)))
			}
		}
		var generate func(int) string
		generate = func(depth int) string {
			if depth == 0 {
				if rng.Intn(2) == 0 {
					return labels[rng.Intn(len(labels))]
				}
				return string(rune('1' + rng.Intn(9)))
			}
			switch rng.Intn(8) {
			case 0:
				return "val(" + generate(depth-1) + ")"
			case 1:
				return "(" + generate(depth-1) + ")"
			case 2:
				return "-" + generate(depth-1)
			case 3:
				return "p(" + generate(depth-1) + ")"
			case 4:
				return "q(" + generate(depth-1) + ")"
			default:
				left, right := generate(depth-1), generate(depth-1)
				return left + []string{" + ", " - ", " * ", " / "}[rng.Intn(4)] + right
			}
		}
		raw := generate(4)
		if got, err := w.Eval(raw); err == nil {
			if err := ValidateExpression(raw); err != nil {
				t.Fatalf("seed %d: Eval(%q) = %d, but syntax refused: %v", seed, raw, got, err)
			}
		}
	})
}
