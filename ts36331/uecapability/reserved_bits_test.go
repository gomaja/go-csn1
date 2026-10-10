package uecapability

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"
)

func TestClassmark2CanonicalReservedBits(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.1.6 figure 10.5.6 and table 10.5.6b:
	// three spare bits and A5/2 are sent as zero and ignored on reception.
	for _, input := range []string{"800000", "008000", "000040", "000001", "808041", "ffffff"} {
		t.Run(input, func(t *testing.T) {
			wire, _ := hex.DecodeString(input)
			decoded, err := DecodeClassmark2ValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			plain, err := EncodeClassmark2ValuePart(decoded.Value)
			if err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("plain %x: %v", plain, err)
			}
			want := []byte{wire[0] & 0x7f, wire[1] & 0x7f, wire[2] & 0xbe}
			canonical, err := EncodeClassmark2ValuePartCanonical(decoded.Value)
			if err != nil || !bytes.Equal(canonical, want) {
				t.Errorf("canonical %x, want %x: %v", canonical, want, err)
			}
			for _, field := range []string{"Spare3", "Spare4", "Spare5", "A52ReservedBit"} {
				if reflect.ValueOf(decoded.Value).FieldByName(field).IsValid() {
					t.Errorf("received bit still exposed as typed field %s", field)
				}
			}
		})
	}
	fresh, err := EncodeClassmark2ValuePart(Classmark2ValuePart{})
	if err != nil || !bytes.Equal(fresh, []byte{0, 0, 0}) {
		t.Fatalf("fresh %x: %v", fresh, err)
	}
}

func TestGERANCSCanonicalClassmark2ReservedBits(t *testing.T) {
	wire, _ := hex.DecodeString("33038000000000000000")
	d, err := DecodeGERANCS(wire)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := EncodeGERANCS(d.Value)
	if err != nil || !bytes.Equal(plain, wire) {
		t.Fatalf("plain %x: %v", plain, err)
	}
	canonical, err := EncodeGERANCSCanonical(d.Value)
	want, _ := hex.DecodeString("330300000000")
	if err != nil || !bytes.Equal(canonical, want) {
		t.Fatalf("canonical %x, want %x: %v", canonical, want, err)
	}
}
