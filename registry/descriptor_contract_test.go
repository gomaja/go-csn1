package registry_test

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/registry"
)

func TestManualDescriptorConvention(t *testing.T) {
	for _, tc := range []struct {
		standard, name string
		wire           []byte
		max            int
	}{
		{"TS 24.008", "Mobile Station Classmark 2 value part", []byte{0x20, 0, 0}, 3},
		{"TS 36.331", "geran-cs", []byte{0x33, 3, 0x20, 0, 0, 0}, 37},
		{"TS 36.331", "geran-ps", []byte{0, 0}, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := registry.Lookup(tc.standard, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if d.MaxOctets != tc.max {
				t.Errorf("MaxOctets=%d, want %d", d.MaxOctets, tc.max)
			}
			if d.DecodeFrom == nil || d.CanonicalAtLength == nil {
				t.Error("missing nested or exact-length operation")
			}
			v, err := d.Decode(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := d.Encode(v)
			if err != nil || !bytes.Equal(encoded, tc.wire) {
				t.Errorf("Encode(Decode)=%x, %v", encoded, err)
			}
			if _, err = d.Encode(&v); err == nil {
				t.Error("pointer accepted")
			}
			if _, err = d.Encode(nil); err == nil {
				t.Error("nil accepted")
			}
		})
	}
}
