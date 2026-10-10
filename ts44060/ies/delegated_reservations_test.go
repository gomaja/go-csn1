package ies

import (
	"bytes"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"strings"
	"testing"
)

func TestDelegatedMPRACHReservation(t *testing.T) {
	// TS 44.060 V19.0.0 §12.41 table 12.41.2 delegates S to table 12.14.2.
	for code := uint8(0); code < 16; code++ {
		wire := []byte{code << 2}
		d, err := DecodeMPRACHControlParametersIE(wire)
		if code <= 9 {
			if err != nil {
				t.Fatalf("S %d: %v", code, err)
			}
			out, err := EncodeMPRACHControlParametersIE(d.Value)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("plain %x: %v", out, err)
			}
			out, err = EncodeMPRACHControlParametersIECanonical(d.Value)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("canonical %x: %v", out, err)
			}
			continue
		}
		requireDelegatedValueError(t, err, true)
		v, err := DecodeMPRACHControlParametersIE([]byte{0})
		if err != nil {
			t.Fatal(err)
		}
		v.Value.S = code
		_, err = EncodeMPRACHControlParametersIE(v.Value)
		requireDelegatedValueError(t, err, false)
		_, err = EncodeMPRACHControlParametersIECanonical(v.Value)
		requireDelegatedValueError(t, err, false)
	}
}

func TestDelegatedMBMSNPMReservation(t *testing.T) {
	// TS 44.060 V19.0.0 §12.40 table 12.40.2 delegates NPM to table 12.45a.1.
	for code := uint8(0); code < 32; code++ {
		wire := []byte{0x90, 1, code << 3}
		d, err := DecodeMBMSSessionParametersListIE(wire)
		if code <= 30 {
			if err != nil {
				t.Fatalf("NPM %d: %v", code, err)
			}
			if *d.Value.LengthOfMBMSBearerIdentityGroupList[0].NPMTransferTime != code {
				t.Fatalf("NPM decoded wrong: %+v", d.Value)
			}
			out, err := EncodeMBMSSessionParametersListIE(d.Value)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("plain %x: %v", out, err)
			}
			out, err = EncodeMBMSSessionParametersListIECanonicalAtLength(d.Value, 3)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("canonical %x: %v", out, err)
			}
			continue
		}
		requireDelegatedValueError(t, err, true)
		v, err := DecodeMBMSSessionParametersListIE([]byte{0x90, 1, 0xf0})
		if err != nil {
			t.Fatal(err)
		}
		*v.Value.LengthOfMBMSBearerIdentityGroupList[0].NPMTransferTime = code
		_, err = EncodeMBMSSessionParametersListIE(v.Value)
		requireDelegatedValueError(t, err, false)
		_, err = EncodeMBMSSessionParametersListIECanonicalAtLength(v.Value, 3)
		requireDelegatedValueError(t, err, false)
	}
}

func requireDelegatedValueError(t *testing.T, err error, decode bool) {
	t.Helper()
	if decode {
		var de *runtime.DecodeError
		if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
			t.Fatalf("want invalid-value, got %v", err)
		}
	}
	if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
		t.Fatalf("want source table value error, got %v", err)
	}
}
