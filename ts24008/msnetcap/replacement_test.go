package msnetcap

import (
	"bytes"
	"testing"
)

func TestReplaceGEA1BitsInDecodedValue(t *testing.T) {
	// Synthetic input; TS 24.008 V20.1.0 §10.5.5.12 sends GEA/1 as zero
	// and accepts either received value. Other received layout stays intact.
	wire := []byte{0x80, 0x00}
	for _, tc := range []struct {
		name string
		edit func(*MSNetworkCapabilityValuePart)
		want []byte
	}{
		{"unchanged", func(*MSNetworkCapabilityValuePart) {}, wire},
		{"fresh", func(v *MSNetworkCapabilityValuePart) { v.GEA1Bits = GEA1Bits{} }, []byte{0, 0}},
		{"in place", func(v *MSNetworkCapabilityValuePart) { v.SMCapabilitiesViaDedicatedChannels = 1 }, []byte{0xc0, 0}},
		{"received zero", func(v *MSNetworkCapabilityValuePart) {
			d, err := DecodeGEA1Bits([]byte{0})
			if err != nil {
				t.Fatal(err)
			}
			v.GEA1Bits = d.Value
		}, []byte{0, 0}},
		{"received one", func(v *MSNetworkCapabilityValuePart) {
			d, err := DecodeGEA1Bits([]byte{0x80})
			if err != nil {
				t.Fatal(err)
			}
			v.GEA1Bits = d.Value
		}, wire},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := DecodeMSNetworkCapabilityValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&d.Value)
			out, err := EncodeMSNetworkCapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(out, tc.want) {
				t.Fatalf("plain %x, want %x: %v", out, tc.want, err)
			}
			canonical, err := EncodeMSNetworkCapabilityValuePartCanonical(d.Value)
			if err != nil || !bytes.Equal(canonical, []byte{tc.want[0] & 0x7f}) {
				t.Fatalf("canonical %x: %v", canonical, err)
			}
			replay, err := EncodeMSNetworkCapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(replay, tc.want) {
				t.Fatalf("canonical changed source: %x: %v", replay, err)
			}
		})
	}
	fresh, err := EncodeGEA1Bits(GEA1Bits{})
	if err != nil || !bytes.Equal(fresh, []byte{0}) {
		t.Fatalf("fresh local encoding %x: %v", fresh, err)
	}
}

func FuzzReplaceGEA1Bits(f *testing.F) {
	f.Add([]byte{0x80, 0})
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, wire []byte) {
		d, err := DecodeMSNetworkCapabilityValuePart(wire)
		if err != nil {
			return
		}
		d.Value.GEA1Bits = GEA1Bits{}
		want := bytes.Clone(wire)
		want[0] &= 0x7f
		out, err := EncodeMSNetworkCapabilityValuePart(d.Value)
		if err != nil || !bytes.Equal(out, want) {
			t.Fatalf("fresh replacement %x, want %x: %v", out, want, err)
		}
	})
}
