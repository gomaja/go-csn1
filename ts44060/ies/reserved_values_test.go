package ies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"strings"
	"testing"
)

func TestMBMSBearerIdentityLengthReservation(t *testing.T) {
	// TS 44.060 V19.0.0 §12.40 tables 12.40.1–2 allow lengths 1–5.
	for _, tc := range []struct {
		wire   string
		length uint8
	}{{"8000", 0}, {"e00000", 6}, {"f00000", 7}} {
		t.Run(tc.wire, func(t *testing.T) {
			wire, _ := hex.DecodeString(tc.wire)
			_, err := DecodeMBMSSessionParametersListIE(wire)
			var de *runtime.DecodeError
			if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
				t.Errorf("decode: want invalid-value, got %v", err)
			}
			value := MBMSSessionParametersListIE{LengthOfMBMSBearerIdentityGroupList: []MBMSSessionParametersListIELengthOfMBMSBearerIdentityGroupListEntry{{LengthOfMBMSBearerIdentity: tc.length, MBMSBearerIdentity: runtime.BitString{Bytes: make([]byte, (int(tc.length)+7)/8), BitLength: int(tc.length)}}}}
			for name, encode := range map[string]func() ([]byte, error){"plain": func() ([]byte, error) { return EncodeMBMSSessionParametersListIE(value) }, "canonical": func() ([]byte, error) { return EncodeMBMSSessionParametersListIECanonicalAtLength(value, len(wire)) }} {
				_, err := encode()
				if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
					t.Errorf("%s: want source-table value constraint, got %v", name, err)
				}
				var bound *runtime.BoundError
				if errors.As(err, &bound) {
					t.Errorf("%s: reservation returned extent error %v", name, err)
				}
			}
		})
	}
	for length := uint8(1); length <= 5; length++ {
		wire := []byte{0x80 | length<<4, 0, 0}
		d, err := DecodeMBMSSessionParametersListIE(wire)
		if err != nil {
			t.Fatalf("control length %d: %v", length, err)
		}
		if got := d.Value.LengthOfMBMSBearerIdentityGroupList[0].LengthOfMBMSBearerIdentity; got != length {
			t.Fatalf("length %d, want %d", got, length)
		}
		out, err := EncodeMBMSSessionParametersListIE(d.Value)
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("control plain %x: %v", out, err)
		}
		out, err = EncodeMBMSSessionParametersListIECanonicalAtLength(d.Value, len(wire))
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("control canonical %x: %v", out, err)
		}
	}
}

func TestReservedNetworkModeOfOperation(t *testing.T) {
	// TS 44.060 V19.0.0 table 12.24.2 reserves NMO 11.
	for code := uint8(0); code < 4; code++ {
		wire := []byte{code << 6, 0, 0}
		d, err := DecodeGPRSCellOptionsIE(wire)
		if code < 3 {
			if err != nil {
				t.Fatal(err)
			}
			plain, err := EncodeGPRSCellOptionsIE(d.Value)
			if err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("plain %x: %v", plain, err)
			}
			if _, err := EncodeGPRSCellOptionsIECanonicalAtLength(d.Value, 3); err != nil {
				t.Fatal(err)
			}
		} else {
			var de *runtime.DecodeError
			if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
				t.Fatalf("decode: %v", err)
			}
			value := GPRSCellOptionsIE{NMO: 3}
			for _, encode := range []func() ([]byte, error){func() ([]byte, error) { return EncodeGPRSCellOptionsIE(value) }, func() ([]byte, error) { return EncodeGPRSCellOptionsIECanonicalAtLength(value, 3) }} {
				if _, err := encode(); err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
					t.Fatalf("encode: %v", err)
				}
			}
		}
	}
}
