package msnetcap

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestMSNetworkCapabilityValuePart(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12: one-octet earlier-release
	// content is zero extended for new fields. These synthetic vectors
	// also round-trip with pycrate 0.7.11's Rel-13 definition.
	for _, wire := range [][]byte{
		{0},
		{0xff, 0x12, 0x34, 0x56, 0x78, 0x90},
		bytes.Repeat([]byte{0xff}, 8),
	} {
		d, err := DecodeMSNetworkCapabilityValuePart(wire)
		if err != nil {
			t.Fatalf("decode %x: %v", wire, err)
		}
		if d.BitsConsumed != len(wire)*8 || d.Tail.BitLength != 0 {
			t.Fatalf("boundary %x: %d, %d", wire, d.BitsConsumed, d.Tail.BitLength)
		}
		if wire[0] == 0xff && (d.Value.GEA1Bits.GEA1 != 1 || d.Value.SMCapabilitiesViaDedicatedChannels != 1 || d.Value.SSScreeningIndicator != 3) {
			t.Fatalf("first octet fields: %+v", d.Value)
		}
		encoded, err := EncodeMSNetworkCapabilityValuePart(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("round trip %x -> %x: %v", wire, encoded, err)
		}
	}
	for _, tc := range []struct {
		wire []byte
		kind runtime.ErrorKind
	}{{nil, runtime.Truncated}, {make([]byte, 9), runtime.Limit}} {
		_, err := DecodeMSNetworkCapabilityValuePart(tc.wire)
		var de *runtime.DecodeError
		if !errors.As(err, &de) || de.Kind != tc.kind {
			t.Fatalf("length %d = %v; want %s", len(tc.wire), err, tc.kind)
		}
	}
}

func FuzzMSNetworkCapabilityValuePart(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{0xff, 0x12, 0x34, 0x56, 0x78, 0x90})
	f.Fuzz(func(t *testing.T, wire []byte) {
		d, err := DecodeMSNetworkCapabilityValuePart(wire)
		if err != nil {
			return
		}
		encoded, err := EncodeMSNetworkCapabilityValuePart(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("round trip %x -> %x: %v", wire, encoded, err)
		}
	})
}
