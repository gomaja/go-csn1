package ies

import (
	"bytes"
	"reflect"
	"testing"
)

func TestLockedIAClosureDefinitions(t *testing.T) {
	// TS 44.060 V19.0.0 §§12.5.2, 12.10d, 12.10f, 12.12,
	// 12.33: each IA dependency has a typed direct codec.
	for _, tc := range []struct {
		name string
		wire []byte
		bits int
	}{
		{"EGPRS Window Size IE", []byte{0x00}, 5},
		{"EGPRS Modulation and Coding IE", []byte{0x00}, 4},
		{"EGPRS Level IE", []byte{0x00}, 2},
		{"Packet Timing Advance IE", []byte{0x00}, 2},
		{"TMGI IE", []byte{0x00, 0x00, 0x00, 0x00}, 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := Lookup(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := definition.Decode(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			value := reflect.ValueOf(decoded)
			if got := int(value.FieldByName("BitsConsumed").Int()); got != tc.bits {
				t.Fatalf("consumed %d bits, want %d", got, tc.bits)
			}
			encoded, err := definition.Encode(value.FieldByName("Value").Interface())
			if err != nil || !bytes.Equal(encoded, tc.wire) {
				t.Fatalf("round trip %x -> %x: %v", tc.wire, encoded, err)
			}
		})
	}
	if _, err := DecodeEGPRSLevelIE([]byte{0xc0}); err == nil {
		t.Fatal("reserved 11 EGPRS Level value accepted")
	}
	if _, err := DecodeEGPRSWindowSizeIE([]byte{0xf8}); err == nil {
		t.Fatal("reserved 11111 EGPRS Window Size accepted")
	}
	if _, err := EncodeEGPRSWindowSizeIE(EGPRSWindowSizeIE{EGPRSWindowSize: 31}); err == nil {
		t.Fatal("reserved 11111 EGPRS Window Size encoded")
	}
	if _, err := DecodeEGPRSModulationAndCodingIE([]byte{0xc0}); err == nil {
		t.Fatal("reserved 1100 EGPRS MCS accepted")
	}
	if _, err := EncodeEGPRSModulationAndCodingIE(EGPRSModulationAndCodingIE{EGPRSModulationAndCodingScheme: 12}); err == nil {
		t.Fatal("reserved 1100 EGPRS MCS encoded")
	}
}
