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

func TestGERANContainersRetainExcessType4Bits(t *testing.T) {
	// TS 36.331 V19.4.0 UE-CapabilityRAT-ContainerList carries
	// Classmark 3 and MS RA capability value parts. TS 24.007 V20.0.0
	// §11.4.2 permits a longer type 4 value part at the receiver.
	cs := append([]byte{0x33, 3, 0, 0, 0}, make([]byte, 32)...)
	cs[5] = 0x60
	cs = append(cs, 0x80)
	decodedCS, err := DecodeGERANCS(cs)
	if err != nil {
		t.Fatal(err)
	}
	if decodedCS.Tail.BitLength != 8 || !bytes.Equal(decodedCS.Tail.Bytes, []byte{0x80}) {
		t.Fatalf("GERAN CS excess: %+v", decodedCS.Tail)
	}
	plainCS, err := EncodeGERANCS(decodedCS.Value)
	if err != nil || !bytes.Equal(plainCS, cs) {
		t.Fatalf("GERAN CS round trip %x: %v", plainCS, err)
	}
	canonicalCS, err := EncodeGERANCSCanonical(decodedCS.Value)
	if err != nil || len(canonicalCS) > 5+32 {
		t.Fatalf("GERAN CS canonical length %d: %v", len(canonicalCS), err)
	}

	ps := append([]byte{0x10, 0xb1, 0}, bytes.Repeat([]byte{0x2b}, 47)...)
	ps = append(ps, 0x00)
	decodedPS, err := DecodeGERANPS(ps)
	if err != nil {
		t.Fatal(err)
	}
	if decodedPS.Tail.BitLength != 8 || !bytes.Equal(decodedPS.Tail.Bytes, []byte{0x00}) {
		t.Fatalf("GERAN PS excess: %+v", decodedPS.Tail)
	}
	plainPS, err := EncodeGERANPS(decodedPS.Value)
	if err != nil || !bytes.Equal(plainPS, ps) {
		t.Fatalf("GERAN PS round trip %x: %v", plainPS, err)
	}
	canonicalPS, err := EncodeGERANPSCanonical(decodedPS.Value)
	if err != nil || len(canonicalPS) > 50 {
		t.Fatalf("GERAN PS canonical length %d: %v", len(canonicalPS), err)
	}
}
