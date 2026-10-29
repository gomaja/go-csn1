package runtime

import (
	"bytes"
	"errors"
	"testing"
)

func TestCanonicalPaddingFillsEnclosingExtent(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.50 p10636 fixes SI10's carrier at
	// 160 bits; TS 24.007 V20.0.0 §B.1.2.2 paragraphs 4046–4053 emits L.
	for _, target := range []bool{false, true} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true, targetSet: target, targetBits: 176})
		if err := w.WriteUint(0, 11); err != nil {
			t.Fatal(err)
		}
		old, err := w.PushLimit(160)
		if err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(0, 5); err != nil {
			t.Fatal(err)
		}
		if err = w.WritePadding(); err != nil {
			t.Fatal(err)
		}
		if w.Position() != 171 {
			t.Fatalf("target=%t: filled to %d, want 171", target, w.Position())
		}
		if err = w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteLiteral("10101"); err != nil {
			t.Fatal(err)
		}
		got, err := w.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		want := append([]byte{0, 0}, bytes.Repeat([]byte{0x2b}, 20)...)
		want[21] = 0x35
		if !bytes.Equal(got, want) {
			t.Fatalf("target=%t: %x, want %x", target, got, want)
		}
	}
}

func TestCanonicalPaddingRetainsReceivedBits(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, sealed: true, TransmittedBits: 8, BitsConsumed: 8, Padding: []BitString{{Bytes: []byte{0xf0}, BitLength: 4}}})
	if err := w.WriteUint(0, 4); err != nil {
		t.Fatal(err)
	}
	if err := w.WritePadding(); err != nil {
		t.Fatal(err)
	}
	got, err := w.Bytes()
	if err != nil || !bytes.Equal(got, []byte{0x0f}) {
		t.Fatalf("replay=%x %v", got, err)
	}
}

func TestCanonicalPaddingRespectsShorterTarget(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 8})
	if _, err := w.PushLimit(16); err != nil {
		t.Fatal(err)
	}
	err := w.WritePadding()
	var bound *BoundError
	if !errors.As(err, &bound) || bound.Kind != CanonicalTarget {
		t.Fatalf("want canonical bound error, got %v", err)
	}
}
