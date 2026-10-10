package restoctets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"strings"
	"testing"
)

func TestReservedAlternatives(t *testing.T) {
	// TS 44.018 V19.0.0 tables 10.5.2.16.1 and 10.5.2.78.1.
	ack := AcknowledgedAccessRequestStruct{ShortIDChoice: AcknowledgedAccessRequestStructShortIDChoice{Alternative: AcknowledgedAccessRequestStructShortIDChoiceAlternativeAlt11, Alt11: &struct{}{}}}
	ia := IARestOctets{CompressedInterRATHOINFOINDChoice: IARestOctetsCompressedInterRATHOINFOINDChoice{Alternative: IARestOctetsCompressedInterRATHOINFOINDChoiceAlternativeEGPRSPacketUplinkAssignment, EGPRSPacketUplinkAssignment: &IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignment{EGPRSPacketUplinkAssignmentChoice: IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentEGPRSPacketUplinkAssignmentChoice{Alternative: IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentEGPRSPacketUplinkAssignmentChoiceAlternativeAlt1, Alt1: &struct{}{}}, ImplicitRejectPSChoice: IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentImplicitRejectPSChoice{Alternative: IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentImplicitRejectPSChoiceAlternativeAltL, AltL: &struct{}{}}}}}
	for _, tc := range []struct {
		name, wire       string
		decode           func([]byte) error
		plain, canonical func() ([]byte, error)
	}{
		{"ack-11", "c0", func(b []byte) error { _, err := DecodeAcknowledgedAccessRequestStruct(b); return err }, func() ([]byte, error) { return EncodeAcknowledgedAccessRequestStruct(ack) }, func() ([]byte, error) { return EncodeAcknowledgedAccessRequestStructCanonical(ack) }},
		{"ia-LH-1", "6b", func(b []byte) error { _, err := DecodeIARestOctets(b); return err }, func() ([]byte, error) { return EncodeIARestOctets(ia) }, func() ([]byte, error) { return EncodeIARestOctetsCanonical(ia) }},
		{"multiple-blocks-0", "000000", func(b []byte) error { _, err := DecodeMultipleBlocksPacketDownlinkAssignment(b); return err }, func() ([]byte, error) {
			return EncodeMultipleBlocksPacketDownlinkAssignment(MultipleBlocksPacketDownlinkAssignment{})
		}, func() ([]byte, error) {
			return EncodeMultipleBlocksPacketDownlinkAssignmentCanonicalAtLength(MultipleBlocksPacketDownlinkAssignment{}, 3)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, _ := hex.DecodeString(tc.wire)
			err := tc.decode(wire)
			var de *runtime.DecodeError
			if !errors.As(err, &de) || de.Kind != runtime.InvalidValue || !strings.Contains(err.Error(), "reserved CSN.1 alternative") {
				t.Errorf("decode: want reserved alternative invalid-value, got %v", err)
			}
			for name, encode := range map[string]func() ([]byte, error){"plain": tc.plain, "canonical": tc.canonical} {
				_, err := encode()
				var bound *runtime.BoundError
				if err == nil || !strings.Contains(err.Error(), "reserved CSN.1 alternative") || errors.As(err, &bound) {
					t.Errorf("%s: want value constraint, got %v", name, err)
				}
			}
		})
	}
}

func TestAcknowledgedAccessRequestAcceptedAlternatives(t *testing.T) {
	for _, wire := range [][]byte{{0, 0}, {0x40}, {0x80, 0}} {
		d, err := DecodeAcknowledgedAccessRequestStruct(wire)
		if err != nil {
			t.Fatal(err)
		}
		out, err := EncodeAcknowledgedAccessRequestStruct(d.Value)
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("plain %x: %v", out, err)
		}
		out, err = EncodeAcknowledgedAccessRequestStructCanonical(d.Value)
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("canonical %x: %v", out, err)
		}
	}
}

func TestAllocatedBlocksSourceTable(t *testing.T) {
	// TS 44.018 V19.0.0 table 10.5.2.16.1: raw codes 0–8 are assigned;
	// 9–15 are reserved. A distribution assignment with a zero TMGI is synthetic.
	for count := uint8(0); count < 16; count++ {
		wire := []byte{0, 0, count<<4 | 8, 0, 0, 0}
		d, err := DecodeMultipleBlocksPacketDownlinkAssignment(wire)
		if count <= 8 {
			if err != nil {
				t.Fatalf("control count %d: %v", count, err)
			}
			plain, err := EncodeMultipleBlocksPacketDownlinkAssignment(d.Value)
			if err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("control %d plain %x: %v", count, plain, err)
			}
			canonical, err := EncodeMultipleBlocksPacketDownlinkAssignmentCanonicalAtLength(d.Value, 6)
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("control %d canonical %x: %v", count, canonical, err)
			}
			continue
		}
		var de *runtime.DecodeError
		if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
			t.Errorf("decode count %d: %v", count, err)
		}
		valid, _ := DecodeMultipleBlocksPacketDownlinkAssignment([]byte{0, 0, 8, 0, 0, 0})
		value := valid.Value
		value.NUMBEROFALLOCATEDBLOCKS = count
		for name, encode := range map[string]func() ([]byte, error){"plain": func() ([]byte, error) { return EncodeMultipleBlocksPacketDownlinkAssignment(value) }, "canonical": func() ([]byte, error) { return EncodeMultipleBlocksPacketDownlinkAssignmentCanonicalAtLength(value, 6) }} {
			if _, err := encode(); err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
				t.Errorf("%s count %d: %v", name, count, err)
			}
		}
	}
}

func TestInterpretedReservedAlphaRemainsAccepted(t *testing.T) {
	// TS 44.018 V19.0.0 table 10.5.2.16.1 maps ALPHA 11–15 to alpha=1.0.
	// The value keeps the received raw code so plain encoding is exact.
	for alpha := uint8(11); alpha <= 15; alpha++ {
		wire := []byte{0, 0x80 | alpha<<2, 0, 0, 0}
		// Extended RA 5 + absent technologies 1 + multiblock selector 1 +
		// ALPHA-present 1 + ALPHA 4 + GAMMA 5 + start 16 + count 2 + absent P0 1.
		wire[0] = 1
		wire[1] = alpha << 4
		d, err := DecodeEGPRSPacketUplinkAssignment(wire)
		if err != nil {
			t.Fatalf("ALPHA %d: %v", alpha, err)
		}
		plain, err := EncodeEGPRSPacketUplinkAssignment(d.Value)
		if err != nil || !bytes.Equal(plain, wire) {
			t.Fatalf("ALPHA plain %x: %v", plain, err)
		}
		if _, err := EncodeEGPRSPacketUplinkAssignmentCanonicalAtLength(d.Value, len(wire)); err != nil {
			t.Fatalf("ALPHA canonical: %v", err)
		}
	}
}

func TestAdditionalReservedSourceTableRows(t *testing.T) {
	for _, tc := range []struct {
		clause, name, field             string
		wire                            []byte
		offset, width, minimum, maximum int
	}{
		{"10.5.2.16", "EGPRS Packet Uplink Assignment", "NUMBEROFRADIOBLOCKSALLOCATED", []byte{0, 0, 0, 0}, 29, 2, 0, 1},
		{"10.5.2.23", "ETWS Primary Notification struct", "TotalNoOfSegmentsForETWSPrimaryNotification", []byte{8, 0}, 1, 4, 1, 15},
		{"10.5.2.23", "ETWS Primary Notification struct", "SegmentNumber", []byte{0x90, 0}, 1, 4, 2, 15},
		{"10.5.2.37b", "PEO IMM Cell Group Definition struct", "TimeoutReadCompleteSI", []byte{0, 0, 0, 0, 0}, 5, 2, 0, 2},
	} {
		t.Run(tc.field, func(t *testing.T) {
			codec, err := LookupClause(tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			valid, err := codec.Decode(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			for code := 0; code < 1<<tc.width; code++ {
				wire := append([]byte(nil), tc.wire...)
				for i := 0; i < tc.width; i++ {
					bit := tc.offset + i
					mask := byte(1 << uint(7-bit%8))
					wire[bit/8] &= ^mask
					if code&(1<<uint(tc.width-1-i)) != 0 {
						wire[bit/8] |= mask
					}
				}
				decoded, err := codec.Decode(wire)
				accepted := code >= tc.minimum && code <= tc.maximum
				if accepted {
					if err != nil {
						t.Fatalf("control %d: %v", code, err)
					}
					value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
					for name, encode := range map[string]func() ([]byte, error){"plain": func() ([]byte, error) { return codec.Encode(value) }, "canonical": func() ([]byte, error) { return codec.CanonicalAtLength(value, len(wire)) }} {
						out, err := encode()
						if err != nil || !bytes.Equal(out, wire) {
							t.Fatalf("control %d %s %x: %v", code, name, out, err)
						}
					}
					continue
				}
				var de *runtime.DecodeError
				if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
					t.Errorf("decode %d: %v", code, err)
				}
				copy := reflect.New(reflect.ValueOf(valid).FieldByName("Value").Type()).Elem()
				copy.Set(reflect.ValueOf(valid).FieldByName("Value"))
				if !setReservedScalar(copy, tc.field, uint64(code)) {
					t.Fatalf("missing field %s", tc.field)
				}
				for name, encode := range map[string]func() ([]byte, error){"plain": func() ([]byte, error) { return codec.Encode(copy.Interface()) }, "canonical": func() ([]byte, error) { return codec.CanonicalAtLength(copy.Interface(), len(wire)) }} {
					if _, err := encode(); err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
						t.Errorf("%s %d: %v", name, code, err)
					}
				}
			}
		})
	}
}

func setReservedScalar(value reflect.Value, name string, code uint64) bool {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		return setReservedScalar(value.Elem(), name, code)
	}
	if value.Kind() != reflect.Struct {
		return false
	}
	if field := value.FieldByName(name); field.IsValid() && field.CanSet() && field.Kind() == reflect.Uint8 {
		field.SetUint(code)
		return true
	}
	for i := 0; i < value.NumField(); i++ {
		if value.Type().Field(i).Name != "Wire" && setReservedScalar(value.Field(i), name, code) {
			return true
		}
	}
	return false
}
