package runtime

import "testing"

// TS 24.007 V20.0.0 Annex B §B.1.1.5 defines case and whitespace matching.
// The additional names exercise the existing runtime normalization behavior.
func TestChoiceBindingCopiesPreserveNamesAndSnapshots(t *testing.T) {
	for _, name := range []string{"NO_NCELL_M", "  A  B\t ", "é COUNT", "\u2003É\u00a0Count\n", "\xffA B", "\x00A  B", ""} {
		t.Run(name, func(t *testing.T) {
			r := NewReader(nil)
			r.Set(name, 1)
			snapshot := r.Fork()
			var outer, inner ChoiceErrors
			candidate := outer.Fork(r)
			candidate.Set(name, 2)
			nested := inner.Fork(candidate)
			nested.Set(name, 3)
			candidate.Commit(nested)
			inner.Release()
			r.Commit(candidate)
			outer.Release()
			for _, check := range []struct {
				reader *Reader
				want   int
			}{{r, 3}, {snapshot, 1}} {
				got, err := check.reader.ExpressionResult(check.reader.ExpressionVariable(name).Operand())
				if err != nil || got != check.want {
					t.Fatalf("name %q: got %d, %v; want %d", name, got, err, check.want)
				}
			}
		})
	}
}
