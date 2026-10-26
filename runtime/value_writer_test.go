package runtime

import (
	"bytes"
	"testing"
)

func TestWithWireValueRestoresContainingWriter(t *testing.T) {
	w := NewWriter()
	if err := w.WriteUint(0xff, 8); err != nil {
		t.Fatal(err)
	}
	wire := Seal(struct{}{}, []byte{0}, 1, BitString{Bytes: []byte{0}, BitLength: 7}, WireInfo{})
	if err := w.WithWireValue(wire, func() error { return w.WriteUint(0, 1) }); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0xa5, 8); err != nil {
		t.Fatal(err)
	}
	actual, err := w.Bytes()
	if err != nil || !bytes.Equal(actual, []byte{0xff, 0, 0xa5}) {
		t.Fatalf("scoped replay %x: %v", actual, err)
	}
	if w.wire.sealed {
		t.Fatal("nested value replaced containing state")
	}
}
