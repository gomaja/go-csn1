package msrac

import (
	"bytes"
	"reflect"
	"testing"
)

func TestA52IsReceivedLayout(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a: send A5/2 as zero; accept any value.
	for _, wire := range [][]byte{{0}, {0x40}, {0xfe}} {
		d, err := DecodeA5Bits(wire)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.ValueOf(d.Value).FieldByName("A52").IsValid() {
			t.Error("A52 remains a typed field")
		}
		plain, err := EncodeA5Bits(d.Value)
		if err != nil || !bytes.Equal(plain, wire) {
			t.Fatalf("plain %x: %v", plain, err)
		}
		canonical, err := EncodeA5BitsCanonicalAtLength(d.Value, 1)
		want := []byte{wire[0] & 0xbe}
		if err != nil || !bytes.Equal(canonical, want) {
			t.Fatalf("canonical %x, want %x: %v", canonical, want, err)
		}
	}
}

func TestReceivedA52BelongsToAccessTechnology(t *testing.T) {
	// Synthetic two-technology vector independently decoded by pycrate 0.7.11.
	// TS 24.008 V20.1.0 §10.5.5.12a: each A5 bits value belongs to its entry.
	wire := []byte{0x12, 0x07, 0x80, 0x13, 0x20, 0x70, 0}
	d, err := DecodeMSRACapabilityValuePart(wire)
	if err != nil {
		t.Fatal(err)
	}
	entries := d.Value.MSRACapabilityValuePartStruct.Entries
	if len(entries) != 2 {
		t.Fatalf("entries %d", len(entries))
	}
	a := entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits
	b := entries[1].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits
	if a == nil || b == nil || !a.ReceivedA52() || b.ReceivedA52() {
		t.Fatalf("local A5/2: %+v %+v", a, b)
	}
	if len(a.Wire.Spare) != 1 || len(b.Wire.Spare) != 1 {
		t.Fatal("local layout missing")
	}
	plain, err := EncodeMSRACapabilityValuePart(d.Value)
	if err != nil || !bytes.Equal(plain, wire) {
		t.Fatalf("plain %x: %v", plain, err)
	}
	canonical, err := EncodeMSRACapabilityValuePartCanonical(d.Value)
	want := []byte{0x12, 0x07, 0, 0x13, 0x20, 0x70, 0}
	if err != nil || !bytes.Equal(canonical, want) {
		t.Fatalf("canonical %x: %v", canonical, err)
	}
	if (A5Bits{}).ReceivedA52() {
		t.Fatal("fresh A5/2 set")
	}
}
