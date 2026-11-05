package registry_test

import (
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
)

func TestHomeDescriptorsMatchRegistry(t *testing.T) {
	tests := []struct {
		standard, name           string
		minimum, maximum, octets int
		lookup                   func(string) (runtime.Descriptor, error)
		list                     func() []runtime.Descriptor
	}{
		{"TS 24.008", "Mobile Station Classmark 2 value part", 24, 24, 3, classmark.Lookup, classmark.Descriptors},
		{"TS 36.331", "geran-cs", 48, 296, 37, uecapability.Lookup, uecapability.Descriptors},
		{"TS 36.331", "geran-ps", 16, 400, 50, uecapability.Lookup, uecapability.Descriptors},
	}
	if got := len(registry.Descriptors()); got != 632 {
		t.Fatalf("registry has %d entries; want 632", got)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d, err := tc.lookup(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			r, err := registry.Lookup(tc.standard, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if d.Standard != tc.standard || d.Name != tc.name || d.Extent.MinimumBits != tc.minimum || d.Extent.MaximumBits != tc.maximum || d.MaxOctets != tc.octets || !d.Extent.Finite || d.Extent.Fixed != (tc.minimum == tc.maximum) || d.Extent.NeedsContext != (tc.minimum != tc.maximum) {
				t.Fatalf("home metadata: %+v", d)
			}
			count := 0
			for _, entry := range tc.list() {
				if entry.Name == tc.name {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("home catalogue contains %d entries", count)
			}
			count = 0
			for _, entry := range registry.Descriptors() {
				if entry.Standard == tc.standard && entry.Name == tc.name {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("registry contains %d entries", count)
			}
			gv, rv := reflect.ValueOf(d), reflect.ValueOf(r)
			for i := range gv.NumField() {
				a, b := gv.Field(i), rv.Field(i)
				if a.Kind() == reflect.Func {
					if a.Pointer() != b.Pointer() {
						t.Fatalf("registry copies a different function for %s", gv.Type().Field(i).Name)
					}
				} else if !reflect.DeepEqual(a.Interface(), b.Interface()) {
					t.Fatalf("metadata %s differs", gv.Type().Field(i).Name)
				}
			}
		})
	}
}
