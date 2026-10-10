package runtime

import (
	"bytes"
	"testing"
)

func TestLocalLayoutBitDoesNotEnterRootSpare(t *testing.T) {
	r := NewReader([]byte{0x80})
	bit, err := r.ReadLayoutBit()
	if err != nil || bit.BitLength != 1 || !bytes.Equal(bit.Bytes, []byte{0x80}) {
		t.Fatalf("bit %+v: %v", bit, err)
	}
	if len(r.Wire().Spare) != 0 {
		t.Fatal("local bit leaked to root")
	}
	w := NewWriter()
	if err := w.WriteLayoutBit(bit); err != nil {
		t.Fatal(err)
	}
	out, err := w.Finish(BitString{})
	if err != nil || !bytes.Equal(out, []byte{0x80}) {
		t.Fatalf("plain %x: %v", out, err)
	}
	if err := NewWriter().WriteLayoutBit(BitString{BitLength: 2, Bytes: []byte{0}}); err == nil {
		t.Fatal("accepted invalid layout width")
	}
	r = NewReader(nil)
	r.SetZeroExtension(true)
	bit, err = r.ReadLayoutBit()
	if err != nil || bit.BitLength != 0 || r.Wire().ImplicitZeros != 1 {
		t.Fatalf("inferred %+v: %v", bit, err)
	}
}
