package restoctets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestDelegatedIANPMReservation(t *testing.T) {
	// TS 44.018 V19.0.0 table 10.5.2.16.1 delegates to TS 44.060
	// V19.0.0 table 12.45a.1: NPM 31 is reserved, without an interpretation.
	control, _ := hex.DecodeString("d0000000001f032b2b2b2b")
	for code := uint8(0); code < 32; code++ {
		wire := append([]byte(nil), control...)
		// NPM bits 44..48, independently verified by pycrate and tshark.
		for i := 0; i < 5; i++ {
			bit := 44 + i
			mask := byte(1 << uint(7-bit%8))
			wire[bit/8] &= ^mask
			if code&(1<<uint(4-i)) != 0 {
				wire[bit/8] |= mask
			}
		}
		d, err := DecodeIARestOctets(wire)
		if code <= 30 {
			if err != nil {
				t.Fatalf("NPM %d: %v", code, err)
			}
			plain, err := EncodeIARestOctets(d.Value)
			if err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("plain %x: %v", plain, err)
			}
			out, err := EncodeIARestOctetsCanonicalAtLength(d.Value, len(wire))
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("canonical %x: %v", out, err)
			}
			continue
		}
		var de *runtime.DecodeError
		if !errors.As(err, &de) || de.Kind != runtime.InvalidValue {
			t.Fatalf("decode %x: %v", wire, err)
		}
		valid, err := DecodeIARestOctets(control)
		if err != nil {
			t.Fatal(err)
		}
		if !setReservedScalar(reflect.ValueOf(&valid.Value).Elem(), "NPMTransferTime", 31) {
			t.Fatal("missing NPM field")
		}
		for _, encode := range []func(IARestOctets) ([]byte, error){EncodeIARestOctets, EncodeIARestOctetsCanonical} {
			if _, err := encode(valid.Value); err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
				t.Fatalf("encode: %v", err)
			}
		}
	}
}

func TestDelegatedIAPFIRemainsAccepted(t *testing.T) {
	// TS 44.018 table 10.5.2.16.1 delegates PFI to TS 24.008 table 10.5.161;
	// TS 44.018 §8.1 speaks of reservations in sub-clause 10. Receiver policy
	// for imported PFI codes 4..7 is deferred; all modes preserve the raw code.
	for _, input := range []string{"d000000000612b2b2b2b2b", "d000000000616b2b2b2b2b", "d00000000061ab2b2b2b2b", "d00000000061eb2b2b2b2b"} {
		wire, _ := hex.DecodeString(input)
		d, err := DecodeIARestOctets(wire)
		if err != nil {
			t.Fatal(err)
		}
		for _, encode := range []func(IARestOctets) ([]byte, error){EncodeIARestOctets, func(v IARestOctets) ([]byte, error) { return EncodeIARestOctetsCanonicalAtLength(v, len(wire)) }} {
			out, err := encode(d.Value)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("PFI %x: %v", out, err)
			}
		}
	}
}
