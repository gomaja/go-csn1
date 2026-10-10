package msrac

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalCapabilityReservations(t *testing.T) {
	// TS 24.008 V20.1.0 table 10.5.146; decoding leaves network handling
	// implementation dependent (§8.1), while the sender must avoid reservations.
	for _, tc := range []struct {
		name      string
		max       int
		valid     func(int) bool
		value     func(uint8) any
		plain     func(any) ([]byte, error)
		canonical func(any, int) ([]byte, error)
		decode    func([]byte) (any, error)
	}{
		{"HSCSD", 31, func(c int) bool { return c >= 1 && c <= 18 }, func(c uint8) any { return MultislotCapabilityStruct{HSCSDMultislotClass: &c} }, func(v any) ([]byte, error) { return EncodeMultislotCapabilityStruct(v.(MultislotCapabilityStruct)) }, func(v any, n int) ([]byte, error) {
			return EncodeMultislotCapabilityStructCanonicalAtLength(v.(MultislotCapabilityStruct), n)
		}, func(b []byte) (any, error) { d, e := DecodeMultislotCapabilityStruct(b); return d.Value, e }},
		{"ECSD", 31, func(c int) bool { return c >= 1 && c <= 18 }, func(c uint8) any { return MultislotCapabilityStruct{ECSDMultislotClass: &c} }, func(v any) ([]byte, error) { return EncodeMultislotCapabilityStruct(v.(MultislotCapabilityStruct)) }, func(v any, n int) ([]byte, error) {
			return EncodeMultislotCapabilityStructCanonicalAtLength(v.(MultislotCapabilityStruct), n)
		}, func(b []byte) (any, error) { d, e := DecodeMultislotCapabilityStruct(b); return d.Value, e }},
		{"8PSK", 3, func(c int) bool { return c >= 1 }, func(c uint8) any { return Content{N8PSKPowerCapability: &c} }, func(v any) ([]byte, error) { return EncodeContent(v.(Content)) }, func(v any, n int) ([]byte, error) { return EncodeContentCanonicalAtLength(v.(Content), n) }, func(b []byte) (any, error) { d, e := DecodeContent(b); return d.Value, e }},
		{"reduction", 7, func(c int) bool { return c <= 6 }, func(c uint8) any {
			return Content{MultislotCapabilityReductionForDownlinkDualCarrierGroup: &ContentMultislotCapabilityReductionForDownlinkDualCarrierGroup{MultislotCapabilityReductionForDownlinkDualCarrier: c}}
		}, func(v any) ([]byte, error) { return EncodeContent(v.(Content)) }, func(v any, n int) ([]byte, error) { return EncodeContentCanonicalAtLength(v.(Content), n) }, func(b []byte) (any, error) { d, e := DecodeContent(b); return d.Value, e }},
		{"EFTA-reduction", 7, func(c int) bool { return c <= 6 }, func(c uint8) any {
			return EnhancedFlexibleTimeslotAssignmentStruct{AlternativeEFTAMultislotClassGroup: &EnhancedFlexibleTimeslotAssignmentStructAlternativeEFTAMultislotClassGroup{EFTAMultislotCapabilityReductionForDownlinkDualCarrier: c}}
		}, func(v any) ([]byte, error) {
			return EncodeEnhancedFlexibleTimeslotAssignmentStruct(v.(EnhancedFlexibleTimeslotAssignmentStruct))
		}, func(v any, n int) ([]byte, error) {
			return EncodeEnhancedFlexibleTimeslotAssignmentStructCanonicalAtLength(v.(EnhancedFlexibleTimeslotAssignmentStruct), n)
		}, func(b []byte) (any, error) {
			d, e := DecodeEnhancedFlexibleTimeslotAssignmentStruct(b)
			return d.Value, e
		}},
		{"DLMC-timeslots", 63, func(c int) bool { return c <= 61 }, func(c uint8) any { return DLMCCapabilityStruct{DLMCMaximumNumberOfDownlinkTimeslots: c} }, func(v any) ([]byte, error) { return EncodeDLMCCapabilityStruct(v.(DLMCCapabilityStruct)) }, func(v any, n int) ([]byte, error) {
			return EncodeDLMCCapabilityStructCanonicalAtLength(v.(DLMCCapabilityStruct), n)
		}, func(b []byte) (any, error) { d, e := DecodeDLMCCapabilityStruct(b); return d.Value, e }},
		{"EC-PCH", 3, func(c int) bool { return c <= 2 }, func(c uint8) any { return Content{ECPCHMonitoringSupport: c} }, func(v any) ([]byte, error) { return EncodeContent(v.(Content)) }, func(v any, n int) ([]byte, error) { return EncodeContentCanonicalAtLength(v.(Content), n) }, func(b []byte) (any, error) { d, e := DecodeContent(b); return d.Value, e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for c := 0; c <= tc.max; c++ {
				v := tc.value(uint8(c))
				wire, err := tc.plain(v)
				if err != nil {
					t.Fatal(err)
				}
				received, err := tc.decode(wire)
				if err != nil {
					t.Fatalf("decode %d: %v", c, err)
				}
				out, err := tc.plain(received)
				if err != nil || !bytes.Equal(out, wire) {
					t.Fatalf("replay %d %x: %v", c, out, err)
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
