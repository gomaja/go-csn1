package uecapability

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestClassmark2CanonicalReservedBits(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.1.6 figure 10.5.6 and table 10.5.6b:
	// three spare bits and A5/2 are sent as zero and ignored on reception.
	for _, input := range []string{"800000", "008000", "000040", "000001", "808041", "ffffff", "a00000", "208000", "200040", "200001", "a08041", "bfffff"} {
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
			if decoded.Value.RevisionLevel == 0 || decoded.Value.RevisionLevel == 3 {
				if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
					t.Errorf("reserved revision canonical: %v", err)
				}
			} else if err != nil || !bytes.Equal(canonical, want) {
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
	wire, _ := hex.DecodeString("3303a000000000000000")
	d, err := DecodeGERANCS(wire)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := EncodeGERANCS(d.Value)
	if err != nil || !bytes.Equal(plain, wire) {
		t.Fatalf("plain %x: %v", plain, err)
	}
	canonical, err := EncodeGERANCSCanonical(d.Value)
	want, _ := hex.DecodeString("330320000000")
	if err != nil || !bytes.Equal(canonical, want) {
		t.Fatalf("canonical %x, want %x: %v", canonical, want, err)
	}
}
