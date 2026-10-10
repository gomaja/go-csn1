package classmark

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalCapabilityReservations(t *testing.T) {
	// TS 24.008 V20.1.0 table 10.5.1.7 and TS 45.002 V19.0.0 §B.1:
	// HSCSD classes 1..18; 8PSK power codes 1..3; GSM Band codes 0..9.
	for _, tc := range []struct {
		name      string
		max       int
		valid     func(int) bool
		value     func(uint8) any
		plain     func(any) ([]byte, error)
		canonical func(any, int) ([]byte, error)
		decode    func([]byte) (any, error)
	}{
		{"HSCSD", 31, func(c int) bool { return c >= 1 && c <= 18 }, func(c uint8) any { return HSCSDMultiSlotCapability{HSCSDMultiSlotClass: c} }, func(v any) ([]byte, error) { return EncodeHSCSDMultiSlotCapability(v.(HSCSDMultiSlotCapability)) }, func(v any, n int) ([]byte, error) {
			return EncodeHSCSDMultiSlotCapabilityCanonicalAtLength(v.(HSCSDMultiSlotCapability), n)
		}, func(b []byte) (any, error) { d, e := DecodeHSCSDMultiSlotCapability(b); return d.Value, e }},
		{"GSM Band", 15, func(c int) bool { return c <= 9 }, func(c uint8) any { return SingleBandSupport{GSMBand: c} }, func(v any) ([]byte, error) { return EncodeSingleBandSupport(v.(SingleBandSupport)) }, func(v any, n int) ([]byte, error) {
			return EncodeSingleBandSupportCanonicalAtLength(v.(SingleBandSupport), n)
		}, func(b []byte) (any, error) { d, e := DecodeSingleBandSupport(b); return d.Value, e }},
		{"8PSK1", 3, func(c int) bool { return c >= 1 }, func(c uint8) any { return N8PSKStruct{N8PSKRFPowerCapability1: &c} }, func(v any) ([]byte, error) { return EncodeN8PSKStruct(v.(N8PSKStruct)) }, func(v any, n int) ([]byte, error) { return EncodeN8PSKStructCanonicalAtLength(v.(N8PSKStruct), n) }, func(b []byte) (any, error) { d, e := DecodeN8PSKStruct(b); return d.Value, e }},
		{"8PSK2", 3, func(c int) bool { return c >= 1 }, func(c uint8) any { return N8PSKStruct{N8PSKRFPowerCapability2: &c} }, func(v any) ([]byte, error) { return EncodeN8PSKStruct(v.(N8PSKStruct)) }, func(v any, n int) ([]byte, error) { return EncodeN8PSKStructCanonicalAtLength(v.(N8PSKStruct), n) }, func(b []byte) (any, error) { d, e := DecodeN8PSKStruct(b); return d.Value, e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for c := 0; c <= tc.max; c++ {
				wire, err := tc.plain(tc.value(uint8(c)))
				if err != nil {
					t.Fatal(err)
				}
				received, err := tc.decode(wire)
				if err != nil {
					t.Fatal(err)
				}
				out, err := tc.plain(received)
				if err != nil || !bytes.Equal(out, wire) {
					t.Fatalf("replay %x: %v", out, err)
				}
				_, err = tc.canonical(received, len(wire))
				if tc.valid(c) {
					if err != nil {
						t.Fatalf("control %d: %v", c, err)
					}
				} else if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
					t.Fatalf("reserved %d: %v", c, err)
				}
			}
		})
	}
}
