package msnetcap

import (
	"bytes"
	"reflect"
	"testing"
)

func TestGEA1IsReceivedLayout(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12: send GEA/1 as zero; accept any value.
	for _, wire := range [][]byte{{0}, {0x80}, {0xff}} {
		d, err := DecodeMSNetworkCapabilityValuePart(wire)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.ValueOf(d.Value.GEA1Bits).FieldByName("GEA1").IsValid() {
			t.Error("GEA1 remains a typed field")
		}
		if d.Value.ReceivedGEA1() != (wire[0]&0x80 != 0) {
			t.Fatal("received GEA/1 accessor disagrees")
		}
		if len(d.Value.GEA1Bits.Wire.Spare) != 1 {
			t.Fatal("local GEA/1 layout missing")
		}
		plain, err := EncodeMSNetworkCapabilityValuePart(d.Value)
		if err != nil || !bytes.Equal(plain, wire) {
			t.Fatalf("plain %x: %v", plain, err)
		}
		canonical, err := EncodeMSNetworkCapabilityValuePartCanonical(d.Value)
		if err != nil || len(canonical) == 0 || canonical[0] != wire[0]&0x7f {
			t.Fatalf("canonical %x: %v", canonical, err)
		}
	}
}

func TestFreshReceivedGEA1(t *testing.T) {
	if (MSNetworkCapabilityValuePart{}).ReceivedGEA1() {
		t.Fatal("fresh GEA/1 set")
	}
}
