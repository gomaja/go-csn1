package runtime

import (
	"errors"
	"fmt"
	"testing"
)

func TestIgnoredFixedBitKeepsItsBoundaryAndWireValue(t *testing.T) {
	r := NewReader([]byte{0x40})
	got, err := r.ReadIgnoredFixed(1)
	if err != nil || got.BitLength != 1 || got.Bytes[0] != 0 {
		t.Fatalf("ignored bit: %+v, %v", got, err)
	}
	next, err := r.ReadUint(1)
	if err != nil || next != 1 || r.Position() != 2 {
		t.Fatalf("following bit: %d at %d, %v", next, r.Position(), err)
	}
	w := NewWriter()
	w.WithWire(r.Wire())
	if err := w.WriteIgnoredFixed(1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(next, 1); err != nil {
		t.Fatal(err)
	}
	encoded, err := w.Finish(BitString{})
	if err != nil || len(encoded) != 1 || encoded[0] != 0x40 {
		t.Fatalf("wire: %x, %v", encoded, err)
	}
}

// TS 44.060 V19.0.0 §12.24 and TS 44.018 V19.0.0 §10.5.2.33b: the decoder
// keeps ignored fallback bits only when the known arm fails on them.
func TestCheckIgnoredFallbackRepeatsTheDecoderTrial(t *testing.T) {
	knownOne := func(r *Reader) error { return r.Expect("1") }
	for _, tc := range []struct {
		name        string
		bits        uint64
		width       int
		toBound     bool
		wantRejects bool
	}{
		{"bounded bits the known arm decodes", 0b101, 3, true, true},
		{"bounded bits the known arm rejects", 0b011, 3, true, false},
		{"one bit the known arm decodes", 1, 1, false, true},
		{"one bit the known arm rejects", 0, 1, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter()
			if err := w.WriteUint(0b10, 2); err != nil {
				t.Fatal(err)
			}
			var old int
			if tc.toBound {
				var err error
				if old, err = w.PushLimit(tc.width); err != nil {
					t.Fatal(err)
				}
			}
			mark := w.MarkFallback()
			if err := w.WriteUint(tc.bits, tc.width); err != nil {
				t.Fatal(err)
			}
			err := w.CheckIgnoredFallback(mark, tc.toBound, knownOne)
			var fallback *FallbackError
			if rejected := errors.As(err, &fallback); rejected != tc.wantRejects {
				t.Fatalf("CheckIgnoredFallback = %v, want rejection %v", err, tc.wantRejects)
			}
			if tc.wantRejects && (!errors.Is(err, ErrFallbackKnown) || fallback.Position != 2) {
				t.Fatalf("fallback error %+v does not name bit 2", fallback)
			}
			if !tc.wantRejects && err != nil {
				t.Fatal(err)
			}
			if tc.toBound {
				if err := w.PopLimit(old); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// The trial reads only the completed bound: bits the decoder cannot see
// yet, or an unfilled bound that PopLimit rejects, are not tried.
func TestCheckIgnoredFallbackWaitsForTheBound(t *testing.T) {
	w := NewWriter()
	old, err := w.PushLimit(4)
	if err != nil {
		t.Fatal(err)
	}
	mark := w.MarkFallback()
	if err := w.WriteUint(1, 1); err != nil {
		t.Fatal(err)
	}
	called := false
	if err := w.CheckIgnoredFallback(mark, true, func(*Reader) error { called = true; return nil }); err != nil || called {
		t.Fatalf("unfilled bound tried: %v, called %v", err, called)
	}
	if err := w.PopLimit(old); !errors.Is(err, ErrBoundConflict) {
		t.Fatalf("PopLimit = %v, want the bound conflict", err)
	}
}

// The trial sees the bindings and the L/H position the decoder has.
func TestCheckIgnoredFallbackKeepsBindingsAndPosition(t *testing.T) {
	w := NewWriter()
	w.Set("N", 2)
	if err := w.WriteUint(0, 2); err != nil {
		t.Fatal(err)
	}
	mark := w.MarkFallback()
	// 0x2b has bit 2 set, so L is transmitted as 1 at offset 2; a trial
	// that restarted the L/H pattern at offset 0 would expect 0.
	if err := w.WriteLiteral("L"); err != nil {
		t.Fatal(err)
	}
	known := func(r *Reader) error {
		if n, err := r.Eval("N"); err != nil || n != 2 {
			return fmt.Errorf("binding N = %d, %v", n, err)
		}
		return r.Expect("L")
	}
	if err := w.CheckIgnoredFallback(mark, false, known); !errors.Is(err, ErrFallbackKnown) {
		t.Fatalf("CheckIgnoredFallback = %v, want ErrFallbackKnown", err)
	}
}
