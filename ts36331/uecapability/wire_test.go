package uecapability

import (
	"bytes"
	"math"
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

func TestGERANCSRejectsOverflowedNestedBitCount(t *testing.T) {
	decoded, err := DecodeGERANCS([]byte{0x33, 3, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	value := decoded.Value
	value.Wire = runtime.WireInfo{}
	value.Classmark3.Wire = runtime.WireInfo{BitsConsumed: math.MaxInt}
	if _, err := EncodeGERANCS(value); err == nil {
		t.Fatal("accepted nested bit count that overflows the wrapper offset")
	}
}

func TestGERANWrapperRejectsOversizedInput(t *testing.T) {
	oversized := make([]byte, 1<<17+1)
	oversized[0], oversized[1] = 0x33, 3
	if _, err := DecodeClassmark2ValuePart(oversized); err == nil {
		t.Fatal("Classmark 2 accepted oversized input")
	}
	if _, err := DecodeGERANCS(oversized); err == nil {
		t.Fatal("GERAN CS accepted oversized input")
	}
}
