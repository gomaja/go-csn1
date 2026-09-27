package runtime

import "testing"

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
	if err != nil || len(encoded) != 1 || encoded[0] != 0x6b {
		t.Fatalf("wire: %x, %v", encoded, err)
	}
}
