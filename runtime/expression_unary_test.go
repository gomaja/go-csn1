package runtime

import "testing"

func TestValidateExpressionUnarySubtractionAfterHyphenatedLabel(t *testing.T) {
	// TS 24.007 V20.0.0 Annex B.2.1 p4108 permits arithmetic expressions.
	// A unary minus after subtraction belongs to the next operand, rather
	// than an attempted extension of the preceding label.
	for _, raw := range []string{
		"N_E-UTRAN - -val(x)",
		"N_E-UTRAN - -(x)",
		"N_E-UTRAN - -q(1)",
		"N_E-UTRAN - -p(val(x))",
		"q(8) - -N_E-UTRAN - -val(x) / q((4)) + q(val(N_E-UTRAN))",
	} {
		t.Run(raw, func(t *testing.T) {
			w := NewWriter()
			w.Set("N_E-UTRAN", 5)
			w.Set("x", 2)
			if _, err := w.Eval(raw); err != nil {
				t.Fatalf("Eval(%q): %v", raw, err)
			}
			if err := ValidateExpression(raw); err != nil {
				t.Fatalf("ValidateExpression(%q): %v", raw, err)
			}
		})
	}
}

func TestValidateExpressionKeepsHyphenatedReservedSuffix(t *testing.T) {
	for _, label := range []string{"x-val", "x-p", "x-q", "x--val", "x--p", "x--q"} {
		w := NewWriter()
		w.Set(label, 7)
		if n, err := w.Eval(label); err != nil || n != 7 {
			t.Fatalf("Eval(%q) = %d, %v", label, n, err)
		}
		if err := ValidateExpression(label); err != nil {
			t.Errorf("ValidateExpression(%q): %v", label, err)
		}
	}
}
