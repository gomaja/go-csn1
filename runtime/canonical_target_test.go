package runtime

import (
	"testing"
)

func remainingCanonicalTarget(t *testing.T, w *Writer) (int, bool) {
	t.Helper()
	query, ok := any(w).(interface{ RemainingCanonicalTarget() (int, bool) })
	if !ok {
		t.Fatal("writer does not expose its explicit canonical target")
	}
	return query.RemainingCanonicalTarget()
}

func TestWriterRemainingCanonicalTarget(t *testing.T) {
	type value struct{ Wire WireInfo }
	for _, tc := range []struct {
		name                  string
		wire                  WireInfo
		position, bound, want int
		present               bool
	}{
		{"fresh", WireInfo{}, 0, -1, 0, false},
		{"ordinary canonical", Canonical(value{}).Wire, 0, -1, 0, false},
		{"ordinary octet completion", Canonical(value{}).Wire, 3, -1, 0, false},
		{"containing limit", WireInfo{}, 3, 16, 0, false},
		{"received extent", WireInfo{sealed: true, BitsConsumed: 3, TransmittedBits: 16}, 3, -1, 0, false},
		{"explicit target", CanonicalTo(value{}, 24).Wire, 3, -1, 21, true},
		{"zero target", CanonicalTo(value{}, 0).Wire, 0, -1, 0, true},
		{"exhausted target", CanonicalTo(value{}, 24).Wire, 24, -1, 0, true},
		{"target independent of local limit", CanonicalTo(value{}, 24).Wire, 3, 8, 21, true},
		{"internal write guard", WireInfo{targetSet: true, targetBits: 24}, 3, -1, 0, false},
		{"negative target", CanonicalTo(value{}, -1).Wire, 0, -1, 0, false},
		{"oversized target", CanonicalTo(value{}, maxBits+1).Wire, 0, -1, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter()
			w.WithWire(tc.wire)
			if tc.bound >= 0 {
				if _, err := w.PushLimit(tc.bound); err != nil {
					t.Fatal(err)
				}
			}
			if err := w.WriteUint(0, tc.position); err != nil {
				t.Fatal(err)
			}
			before := *w
			for range 2 {
				if n, ok := remainingCanonicalTarget(t, w); n != tc.want || ok != tc.present {
					t.Errorf("remaining = %d, %v; want %d, %v", n, ok, tc.want, tc.present)
				}
			}
			if before.bits != w.bits || before.limit != w.limit || before.bounded != w.bounded || before.wire.canonical != w.wire.canonical || before.wire.targetSet != w.wire.targetSet || before.wire.targetBits != w.wire.targetBits || before.stateErr != w.stateErr {
				t.Fatal("query changed writer state")
			}
		})
	}
	for _, position := range []int{-1, 25} {
		w := &Writer{bits: position, wire: CanonicalTo(value{}, 24).Wire}
		if n, ok := remainingCanonicalTarget(t, w); n != 0 || ok {
			t.Errorf("invalid position %d = %d, %v", position, n, ok)
		}
	}
	if n, ok := remainingCanonicalTarget(t, nil); n != 0 || ok {
		t.Fatalf("nil = %d, %v", n, ok)
	}
}

func TestCanonicalAtLengthExposesWriterTarget(t *testing.T) {
	type value struct{ Wire WireInfo }
	called := false
	encode := func(v value) ([]byte, error) {
		w := NewWriter()
		w.WithWire(v.Wire)
		if n, ok := remainingCanonicalTarget(t, w); n != 24 || !ok {
			t.Fatalf("callback target = %d, %v", n, ok)
		}
		called = true
		return []byte{0, 0, 0}, nil
	}
	decode := func([]byte) (Decoded[value], error) { return Decoded[value]{}, nil }
	if _, err := CanonicalEncodeAtLength(value{}, 3, 0, 3, encode, decode, nil); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("canonical callback was not called")
	}
}
