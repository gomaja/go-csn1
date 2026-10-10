package uecapability

import (
	"bytes"
	"testing"
)

func TestReplaceClassmark2InDecodedGERANCS(t *testing.T) {
	// Synthetic Classmark 2 with received spare bits and A5/2 set.
	// TS 24.008 V20.1.0 §10.5.1.6 requires fresh sender bits to be zero.
	wire := []byte{0x33, 3, 0xa0, 0x80, 0x41, 0, 0, 0, 0, 0}
	for _, tc := range []struct {
		name        string
		replacement []byte
		want        []byte
	}{
		{"fresh", nil, []byte{0x20, 0, 0}},
		{"received", []byte{0x20, 0x80, 1}, []byte{0x20, 0x80, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := DecodeGERANCS(wire)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := EncodeGERANCS(d.Value)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("unchanged %x: %v", replay, err)
			}
			d.Value.Classmark2 = Classmark2ValuePart{RevisionLevel: 1}
			if tc.replacement != nil {
				c, err := DecodeClassmark2ValuePart(tc.replacement)
				if err != nil {
					t.Fatal(err)
				}
				d.Value.Classmark2 = c.Value
			}
			want := bytes.Clone(wire)
			copy(want[2:5], tc.want)
			out, err := EncodeGERANCS(d.Value)
			if err != nil || !bytes.Equal(out, want) {
				t.Fatalf("plain %x, want %x: %v", out, want, err)
			}
			canonical, err := EncodeGERANCSCanonical(d.Value)
			if err != nil || !bytes.Equal(canonical, []byte{0x33, 3, 0x20, 0, 0, 0}) {
				t.Fatalf("canonical %x: %v", canonical, err)
			}
		})
	}
}
