package registry_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

func TestCatalogueIsDetachedAndOrdered(t *testing.T) {
	entries := registry.Descriptors()
	if len(entries) == 0 {
		t.Fatal("empty catalogue")
	}
	for i, d := range entries {
		if d.DecodeFrom == nil || d.CanonicalAtLength == nil {
			t.Errorf("%s: incomplete operations", d.Name)
		}
		if d.Extent.Finite && d.MaxOctets != (d.Extent.MaximumBits+7)/8 {
			t.Errorf("%s: inconsistent finite ceiling", d.Name)
		}
		if !d.Extent.Finite && d.MaxOctets != 0 {
			t.Errorf("%s: invented finite ceiling", d.Name)
		}
		if d.Extent.Fixed && (!d.Extent.Finite || d.Extent.MinimumBits != d.Extent.MaximumBits) {
			t.Errorf("%s: inconsistent fixed extent", d.Name)
		}
		if d.Extent.Source.Standard != d.Standard || d.Extent.Source.Version != d.Version || d.Extent.Source.Clause != d.Clause {
			t.Errorf("%s: incorrect source", d.Name)
		}
		context := d.Name == "SI7 Rest Octets" || d.Name == "SI8 Rest Octets"
		if (d.DecodeWithContext != nil) != context || (d.DecodeFromWithContext != nil) != context || (d.EncodeWithContext != nil) != context || (d.CanonicalWithContext != nil) != context || (d.CanonicalAtLengthWithContext != nil) != context {
			t.Errorf("%s: ACS availability", d.Name)
		}
		if i > 0 {
			a, b := entries[i-1], d
			if a.Standard > b.Standard || a.Standard == b.Standard && (a.Clause > b.Clause || a.Clause == b.Clause && a.Name > b.Name) {
				t.Fatal("unstable catalogue order")
			}
		}
	}
	original := entries[0]
	entries[0].Name = "changed"
	entries[0].Decode = nil
	again := registry.Descriptors()
	if again[0].Name != original.Name || again[0].Decode == nil {
		t.Fatal("catalogue aliases caller storage")
	}
	looked, err := registry.LookupClause(original.Standard, original.Clause, original.Name)
	if err != nil || looked.Decode == nil {
		t.Fatal("catalogue mutation reached lookup", err)
	}
}
func TestDescriptorWrongTypesAreLocated(t *testing.T) {
	for _, d := range registry.Descriptors() {
		for _, value := range []any{nil, 0, "invalid", (*runtime.Decoded[int])(nil)} {
			for _, call := range []func() (any, error){func() (any, error) { return d.Encode(value) }, func() (any, error) { return d.Canonical(value) }, func() (any, error) { return d.CanonicalAtLength(value, 0) }} {
				_, err := call()
				var located *runtime.OperationError
				if !errors.As(err, &located) || located.Kind != runtime.InvalidValue || located.Location.BitOffset != -1 || located.Location.Width != 0 || located.Location.Path == "" || located.Location.Source.Standard != d.Standard {
					t.Fatalf("%s wrong type: %v", d.Name, err)
				}
			}
		}
	}
}
func TestContextDescriptorOperations(t *testing.T) {
	for _, name := range []string{"SI7 Rest Octets", "SI8 Rest Octets"} {
		d, err := registry.Lookup("TS 44.018", name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.Decode(bytes.Repeat([]byte{0x2b}, 20))
		if !errors.Is(err, runtime.ErrContextRequired) {
			t.Fatalf("missing ACS: %v", err)
		}
		for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
			raw := bytes.Repeat([]byte{0x2b}, 20)
			decoded, err := d.DecodeWithContext(raw, acs)
			if err != nil {
				t.Fatal(err)
			}
			plain, err := d.EncodeWithContext(decoded, acs)
			if err != nil || !bytes.Equal(plain, raw) {
				t.Fatalf("context round trip: %x %v", plain, err)
			}
			expected, err := d.CanonicalWithContext(decoded, acs)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := d.CanonicalAtLengthWithContext(decoded, 20, acs)
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("context extent: %x %v", actual, err)
			}
			for _, n := range []int{-1, 0, 19, 21} {
				_, err := d.CanonicalAtLengthWithContext(decoded, n, acs)
				var extent *runtime.ExtentError
				if !errors.As(err, &extent) {
					t.Fatalf("extent %d: %v", n, err)
				}
			}
			prefixed := append([]byte{0xff}, raw...)
			r := runtime.NewReader(prefixed)
			_, _ = r.ReadUint(8)
			if _, err := r.PushLimit(160); err != nil {
				t.Fatal(err)
			}
			nested, err := d.DecodeFromWithContext(r, acs)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.TypeOf(nested) != reflect.TypeOf(decoded) || reflect.ValueOf(nested).FieldByName("BitsConsumed").Int() != reflect.ValueOf(decoded).FieldByName("BitsConsumed").Int() {
				t.Fatalf("nested context wrapper: %T %+v", nested, nested)
			}
		}
		_, err = d.DecodeWithContext([]byte{0}, runtime.SI4ACS(2))
		var located *runtime.OperationError
		if !errors.As(err, &located) || located.Kind != runtime.InvalidValue || located.Location.BitOffset != -1 {
			t.Fatalf("invalid ACS: %v", err)
		}
	}
	_, err := registry.DecodeWithContext("TS 24.008", "Mobile Station Classmark 2 value part", nil, runtime.SI4ACSZero)
	var located *runtime.OperationError
	if !errors.Is(err, runtime.ErrUnsupported) || !errors.As(err, &located) || located.Operation != "context" {
		t.Fatal("non-ACS context", err)
	}
}
func TestLookupErrorsAreLocated(t *testing.T) {
	for _, call := range []func() (runtime.Descriptor, error){func() (runtime.Descriptor, error) { return registry.Lookup("unknown", "missing") }, func() (runtime.Descriptor, error) { return registry.LookupClause("TS 24.008", "missing", "name") }} {
		_, err := call()
		var located *runtime.OperationError
		if !errors.As(err, &located) || located.Operation != "lookup" || located.Kind != runtime.InvalidSchema || located.Location.BitOffset != -1 {
			t.Fatal(err)
		}
	}
}
