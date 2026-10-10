package uecapability

import (
	"bytes"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

func TestClassmark2SpareWireRecord(t *testing.T) {
	d, err := DecodeClassmark2ValuePart([]byte{0xa0, 0x80, 0x41, 0xaa})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Value.Wire.Spare) != 4 {
		t.Fatalf("spare records: %+v", d.Value.Wire.Spare)
	}
	for _, bit := range d.Value.Wire.Spare {
		if bit.BitLength != 1 || !bytes.Equal(bit.Bytes, []byte{0x80}) {
			t.Fatalf("spare bit: %+v", bit)
		}
	}
	d.Value.RFPowerCapability = 3
	out, err := EncodeClassmark2ValuePart(d.Value)
	if err != nil || !bytes.Equal(out, []byte{0xa3, 0x80, 0x41, 0xaa}) {
		t.Fatalf("edited value %x: %v", out, err)
	}
	canonical, err := EncodeClassmark2ValuePartCanonical(d.Value)
	if err != nil || !bytes.Equal(canonical, []byte{0x23, 0, 0}) {
		t.Fatalf("canonical %x: %v", canonical, err)
	}
	d.Value.Wire.Spare[0] = runtime.BitString{Bytes: []byte{0xc0}, BitLength: 2}
	if _, err := EncodeClassmark2ValuePart(d.Value); err == nil {
		t.Fatal("accepted invalid spare width")
	}
}
