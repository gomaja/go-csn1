package registry_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
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

func TestDescriptorFailureSourceSurvivesSharedMetadata(t *testing.T) {
	want := runtime.Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.7"}
	_, decodeErr := classmark.DecodeClassmark3ValuePart(nil)
	encodeErr := classmark.EncodeClassmark3ValuePartTo(nil, classmark.Classmark3ValuePart{})
	for _, tc := range []struct {
		name, operation string
		err             error
	}{
		{"empty input", "decode", decodeErr},
		{"nil writer", "encode", encodeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var located *runtime.OperationError
			if !errors.As(tc.err, &located) || located.Operation != tc.operation || located.Location.Source != want || located.Location.Path != "Classmark3ValuePart" {
				t.Fatalf("failure lost its operation or source: %#v", tc.err)
			}
		})
	}
}

func TestGeneratedCatalogueReturnsDetachedEntries(t *testing.T) {
	first := classmark.Descriptors()
	if len(first) == 0 || cap(first) != len(first) {
		t.Fatal("invalid catalogue extent")
	}
	name, extent := first[0].Name, first[0].Extent
	first[0].Name = "changed"
	first[0].Decode = nil
	first[0].Extent.MinimumBits++
	second := classmark.Descriptors()
	if second[0].Name != name || second[0].Decode == nil || second[0].Extent != extent {
		t.Fatal("a returned entry changed the generated catalogue")
	}
}
