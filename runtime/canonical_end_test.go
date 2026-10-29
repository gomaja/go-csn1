package runtime

import (
	"errors"
	"testing"
)

func TestCanonicalNullRequiresEnd(t *testing.T) {
	// TS 24.007 V20.0.0 Annex B.1.1.2 p3987–3988: null has no symbols.
	for _, tc := range []struct {
		name                 string
		bits, target, bound  int
		canonical, wantError bool
	}{
		{"replay", 7, 8, -1, false, false},
		{"target remainder", 7, 8, -1, true, true},
		{"whole octet remainder", 8, 16, -1, true, true},
		{"target complete", 8, 8, -1, true, false},
		{"implicit octet remainder", 7, -1, -1, true, true},
		{"implicit octet complete", 8, -1, -1, true, false},
		{"inner complete", 7, 16, 7, true, false},
		{"inner remainder", 7, 16, 8, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter()
			w.WithWire(WireInfo{canonical: tc.canonical, targetSet: tc.target >= 0, targetBits: tc.target})
			if tc.bound >= 0 {
				if _, err := w.PushLimit(tc.bound); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.WriteUint(0, tc.bits); err != nil {
				t.Fatal(err)
			}
			err := w.RequireCanonicalEnd()
			var got *BoundError
			if errors.As(err, &got) != tc.wantError || (!tc.wantError && err != nil) {
				t.Fatalf("err=%v", err)
			}
			if got != nil {
				kind := CanonicalTarget
				limit := tc.target
				if limit < 0 {
					limit = (tc.bits + 7) / 8 * 8
				}
				if tc.bound >= 0 {
					kind = LengthBound
					limit = tc.bound
				}
				if got.Kind != kind || got.Limit != limit || got.Position != tc.bits {
					t.Fatalf("bound=%+v", got)
				}
			}
		})
	}
}
