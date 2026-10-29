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

func TestCanonicalNullPreservesTerminalBoundary(t *testing.T) {
	for _, padding := range []bool{false, true} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true})
		if err := w.WriteUint(0, 8); err != nil {
			t.Fatal(err)
		}
		if err := w.RequireCanonicalEnd(); err != nil {
			t.Fatal(err)
		}
		var bound *BoundError
		if err := w.PadMinimum(16, padding); !errors.As(err, &bound) || bound.Kind != CanonicalTarget || bound.Limit != 16 || bound.Position != 8 {
			t.Fatalf("minimum: %v", err)
		}
		if err := w.WriteUint(0, 1); !errors.As(err, &bound) || bound.Kind != CanonicalTarget || bound.Limit != 8 || bound.Position != 8 {
			t.Fatalf("following: %v", err)
		}
		if w.Position() != 8 {
			t.Fatalf("writer changed position: %d", w.Position())
		}
	}
}

func TestCanonicalNullReportsRequestedFixedExtent(t *testing.T) {
	encode := func(v struct{ Wire WireInfo }) ([]byte, error) {
		w := NewWriter()
		w.WithWire(v.Wire)
		if err := w.WriteUint(0, 3); err != nil {
			return nil, err
		}
		if err := w.RequireCanonicalEnd(); err != nil {
			return nil, err
		}
		return w.Finish(BitString{})
	}
	decode := func([]byte) (Decoded[struct{ Wire WireInfo }], error) {
		t.Fatal("refused bytes reached decode")
		return Decoded[struct{ Wire WireInfo }]{}, nil
	}
	out, err := CanonicalEncode(struct{ Wire WireInfo }{}, 19, 19, false, encode, decode, nil)
	var bound *BoundError
	if out != nil || !errors.As(err, &bound) || bound.Kind != CanonicalTarget || bound.Limit != 152 || bound.Position != 3 {
		t.Fatalf("out=%x bound=%+v error=%v", out, bound, err)
	}
}

func TestCanonicalNullBoundaryPrecedesInferredZeroExtension(t *testing.T) {
	// Null remains the selected arm even when the containing capability permits
	// inferred zeros (TS 24.007 V20.0.0 Annex B Rule B2; TS 24.008
	// V20.1.0 §10.5.5.12a). Later content must not pass its terminal boundary.
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true})
	w.SetZeroExtension(true)
	if err := w.WriteUint(0, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.RequireCanonicalEnd(); err != nil {
		t.Fatal(err)
	}
	var bound *BoundError
	if err := w.WriteUint(0, 1); !errors.As(err, &bound) || bound.Kind != CanonicalTarget || bound.Position != 8 || bound.Limit != 8 {
		t.Fatalf("inferred content after null: %v", err)
	}
}
