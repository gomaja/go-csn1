package runtime

import (
	"strings"
	"testing"
)

func TestValidateExpressionSyntax(t *testing.T) {
	for _, raw := range []string{"val(", "q(val(NR_OF_TDD_CELLS)", "", "()", "val()", "1 +", "* 2", "1 2", "1 + )", "1)"} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateExpression(raw); err == nil {
				t.Errorf("accepted %q", raw)
			}
		})
	}
	for _, raw := range []string{"val(N)", "val(Extension Length) + 1", "val(N_E-UTRAN + 1)", "val(N_E-UTRAN - 1)", "p(val(N))", "q(Count)", "1 / 0", "1048576 * 1048576", "-1", "val(x-y z)", "(1+2)*3"} {
		t.Run(raw, func(t *testing.T) {
			if err := ValidateExpression(raw); err != nil {
				t.Errorf("refused %q: %v", raw, err)
			}
		})
	}
}

func TestValidateExpressionKeepsBindingsDynamic(t *testing.T) {
	const arithmetic = "val(x - y)"
	if err := ValidateExpression(arithmetic); err != nil {
		t.Fatal(err)
	}
	if got, err := eval(arithmetic, map[string]uint64{"x": 5, "y": 2}); err != nil || got != 3 {
		t.Fatalf("Eval = %d, %v; want 3", got, err)
	}
	const label = "val(x-y)"
	if err := ValidateExpression(label); err != nil {
		t.Fatal(err)
	}
	if _, err := eval(label, map[string]uint64{"x": 5, "y": 2}); err == nil {
		t.Fatal("unbound label evaluated as subtraction")
	}
	if got, err := eval(label, map[string]uint64{"x": 5, "y": 2, "x-y": 7}); err != nil || got != 7 {
		t.Fatalf("label = %d, %v; want 7", got, err)
	}
	if err := ValidateExpression("val(abc)"); err != nil {
		t.Fatal(err)
	}
	if _, err := eval("val(abc)", map[string]uint64{"a bc": 1, "ab c": 2}); err == nil {
		t.Fatal("runtime ambiguity accepted")
	}
}

func FuzzValidateExpression(f *testing.F) {
	for _, raw := range []string{"val(N)", "val(x-y)", "(1+2)*3", "max(val(N))", "val(", "1+", "q(N)"} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > 4096 {
			t.Skip()
		}
		_ = ValidateExpression(raw)
	})
}

func TestValidateExpressionLongLabelsAvoidPrefixAllocations(t *testing.T) {
	for _, words := range []int{32, 256, 2048} {
		raw := "val(" + strings.Repeat("label ", words) + "end)"
		allocations := testing.AllocsPerRun(5, func() {
			if err := ValidateExpression(raw); err != nil {
				panic(err)
			}
		})
		// Token storage grows geometrically. Checking a longer possible label
		// must not allocate each prefix as dynamic binding lookup does.
		if allocations > 32 {
			t.Errorf("%d words: %g allocations, want at most 32", words, allocations)
		}
	}
}
