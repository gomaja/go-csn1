package uecapability

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestGERANWrapperWireState(t *testing.T) {
	cm2, err := DecodeClassmark2ValuePart([]byte{0, 0, 0, 0xaa})
	if err != nil {
		t.Fatal(err)
	}
	cm2.Value.Wire.Tail = runtime.BitString{Bytes: []byte{0x55}, BitLength: 8}
	if out, err := EncodeClassmark2ValuePart(cm2.Value); err != nil || !bytes.Equal(out, []byte{0, 0, 0, 0x55}) {
		t.Fatalf("edited Classmark 2 tail: %x, %v", out, err)
	}
	cm2.Value.Wire.Tail.BitLength = 16
	if _, err := EncodeClassmark2ValuePart(cm2.Value); err == nil {
		t.Fatal("accepted inconsistent Classmark 2 tail")
	}
	cs, err := DecodeGERANCS([]byte{0x33, 3, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	cs.Value.Wire.BitsConsumed--
	if _, err := EncodeGERANCS(cs.Value); err == nil {
		t.Fatal("accepted inconsistent GERAN CS boundary")
	}
	ps, err := DecodeGERANPS([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	ps.Value.Wire.Tail = runtime.BitString{Bytes: []byte{0xaa}, BitLength: 8}
	if _, err := EncodeGERANPS(ps.Value); err == nil {
		t.Fatal("accepted invented GERAN PS tail")
	}
}
