package registry_test

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

func TestDescriptorReaderKeepsExistingEmptyFailures(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		width      int
	}{
		{"SI1 Rest Octets", "SI1 Rest Octets", 8},
		{"IA Rest Octets", "IARestOctets/IARestOctetsCompressedInterRATHOINFOINDChoiceRCC/IARestOctetsCompressedInterRATHOINFOINDChoiceRCCAltLL", 2},
		{"IAR Rest Octets", "IAR Rest Octets", 24},
	} {
		d, err := registry.Lookup("TS 44.018", tc.name)
		if err != nil {
			t.Fatal(err)
		}
		r := runtime.NewReader(nil)
		if _, err := r.PushLimit(0); err != nil {
			t.Fatal(err)
		}
		_, err = d.DecodeFrom(r)
		var located *runtime.OperationError
		if !errors.As(err, &located) || located.Kind != runtime.Truncated || located.Location.Path != tc.path || located.Location.BitOffset != 0 || located.Location.Width != tc.width || errors.Is(err, runtime.ErrEmptyValue) {
			t.Errorf("%s: empty reader failure changed: %v", tc.name, err)
		}
	}
}

// A bounded reader exposes the same value bytes as the byte entry point.
// The extent describes canonical bytes; receivers may retain excess input.
func TestDescriptorDecodeEntryPointBoundaries(t *testing.T) {
	descriptors := registry.Descriptors()
	if len(descriptors) != 237 {
		t.Fatalf("catalogue has %d descriptors, want 237", len(descriptors))
	}
	for _, d := range descriptors {
		checkDescriptorTypedBoundaries(t, d)
		lengths := []int{0, (d.Extent.MinimumBits - 1) / 8, (d.Extent.MinimumBits + 7) / 8}
		if d.Extent.Finite {
			lengths = append(lengths, (d.Extent.MaximumBits+7)/8, (d.Extent.MaximumBits+7)/8+1)
		}
		for _, n := range lengths {
			if n < 0 {
				continue
			}
			for _, fill := range []byte{0, 0x2b, 0xff} {
				data := bytes.Repeat([]byte{fill}, n)
				if d.DecodeWithContext != nil {
					for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
						decoded, byteErr := d.DecodeWithContext(data, acs)
						bounded := runtime.NewReader(data)
						if _, err := bounded.PushLimit(n * 8); err != nil {
							t.Fatal(err)
						}
						_, readerErr := d.DecodeFromWithContext(bounded, acs)
						if (byteErr == nil) != (readerErr == nil) {
							t.Errorf("%s %s ACS=%d length=%d fill=%02x: byte=%v reader=%v", d.Standard, d.Name, acs, n, fill, byteErr, readerErr)
						}
						if byteErr == nil {
							checkDescriptorEncodes(t, d, decoded, data, &acs)
						}
					}
					continue
				}
				decoded, byteErr := d.Decode(data)
				reader := runtime.NewReader(data)
				if _, err := reader.PushLimit(n * 8); err != nil {
					t.Fatal(err)
				}
				_, readerErr := d.DecodeFrom(reader)
				if (byteErr == nil) != (readerErr == nil) {
					t.Errorf("%s %s length=%d fill=%02x: byte=%v reader=%v", d.Standard, d.Name, n, fill, byteErr, readerErr)
				}
				if byteErr == nil {
					checkDescriptorEncodes(t, d, decoded, data, nil)
				}
			}
		}
		if d.Extent.MinimumBits > 0 {
			bits := d.Extent.MinimumBits - 1
			if d.DecodeWithContext != nil {
				for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
					r := runtime.NewReader(make([]byte, (bits+7)/8))
					if _, err := r.PushLimit(bits); err != nil {
						t.Fatal(err)
					}
					_, err := d.DecodeFromWithContext(r, acs)
					if err == nil {
						t.Errorf("%s %s ACS=%d accepted %d bits below minimum %d", d.Standard, d.Name, acs, bits, d.Extent.MinimumBits)
					}
				}
			} else {
				r := runtime.NewReader(make([]byte, (bits+7)/8))
				if _, err := r.PushLimit(bits); err != nil {
					t.Fatal(err)
				}
				if _, err := d.DecodeFrom(r); err == nil {
					t.Errorf("%s %s accepted %d bits below minimum %d", d.Standard, d.Name, bits, d.Extent.MinimumBits)
				}
			}
		}
	}
}

func checkDescriptorTypedBoundaries(t *testing.T, d runtime.Descriptor) {
	t.Helper()
	// Decode returns a typed wrapper even on error, so its zero Value gives
	// every descriptor a correctly typed argument, including unsupported ones.
	wrapper, _ := d.Decode(nil)
	value := reflect.ValueOf(wrapper)
	if value.Kind() != reflect.Struct || !value.FieldByName("Value").IsValid() {
		t.Fatalf("%s %s: Decode did not return a typed wrapper", d.Standard, d.Name)
	}
	zero := value.FieldByName("Value").Interface()
	_, _ = d.Encode(zero)
	_, _ = d.Canonical(zero)
	lengths := []int{-1, 0, (d.Extent.MinimumBits + 7) / 8}
	if d.Extent.Finite {
		lengths = append(lengths, (d.Extent.MaximumBits+7)/8, (d.Extent.MaximumBits+7)/8+1)
	}
	for _, n := range lengths {
		got, err := d.CanonicalAtLength(zero, n)
		if err == nil && len(got) != n {
			t.Errorf("%s %s canonical extent %d produced %d octets", d.Standard, d.Name, n, len(got))
		}
		if d.Extent.Finite && n >= 0 && (n < (d.Extent.MinimumBits+7)/8 || n > (d.Extent.MaximumBits+7)/8) && err == nil {
			t.Errorf("%s %s accepted %d octets outside canonical extent %d..%d bits", d.Standard, d.Name, n, d.Extent.MinimumBits, d.Extent.MaximumBits)
		}
	}
	if d.DecodeWithContext == nil {
		return
	}
	for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
		_, _ = d.EncodeWithContext(zero, acs)
		_, _ = d.CanonicalWithContext(zero, acs)
		for _, n := range lengths {
			got, err := d.CanonicalAtLengthWithContext(zero, n, acs)
			if err == nil && len(got) != n {
				t.Errorf("%s %s ACS=%d canonical extent %d produced %d octets", d.Standard, d.Name, acs, n, len(got))
			}
			if d.Extent.Finite && n >= 0 && (n < (d.Extent.MinimumBits+7)/8 || n > (d.Extent.MaximumBits+7)/8) && err == nil {
				t.Errorf("%s %s ACS=%d accepted %d octets outside canonical extent %d..%d bits", d.Standard, d.Name, acs, n, d.Extent.MinimumBits, d.Extent.MaximumBits)
			}
		}
	}
}

func checkDescriptorEncodes(t *testing.T, d runtime.Descriptor, decoded any, data []byte, acs *runtime.SI4ACS) {
	t.Helper()
	var replay, canonical []byte
	var err error
	if acs == nil {
		replay, err = d.Encode(decoded)
	} else {
		replay, err = d.EncodeWithContext(decoded, *acs)
	}
	if err != nil || !bytes.Equal(replay, data) {
		t.Errorf("%s %s encode replay: got %x, want %x, error %v", d.Standard, d.Name, replay, data, err)
	}
	if acs == nil {
		canonical, err = d.Canonical(decoded)
	} else {
		canonical, err = d.CanonicalWithContext(decoded, *acs)
	}
	if err == nil {
		var exact []byte
		if acs == nil {
			exact, err = d.CanonicalAtLength(decoded, len(canonical))
		} else {
			exact, err = d.CanonicalAtLengthWithContext(decoded, len(canonical), *acs)
		}
		if err != nil || len(exact) != len(canonical) {
			t.Errorf("%s %s canonical at natural length %d: got %d bytes, error %v", d.Standard, d.Name, len(canonical), len(exact), err)
		}
	}
	if !d.Extent.Finite {
		return
	}
	for _, n := range []int{-1, (d.Extent.MinimumBits - 1) / 8, (d.Extent.MaximumBits+7)/8 + 1} {
		if n >= 0 && n*8 >= d.Extent.MinimumBits && n*8 <= d.Extent.MaximumBits {
			continue
		}
		if acs == nil {
			_, err = d.CanonicalAtLength(decoded, n)
		} else {
			_, err = d.CanonicalAtLengthWithContext(decoded, n, *acs)
		}
		if err == nil {
			t.Errorf("%s %s accepted canonical extent %d outside %d..%d bits", d.Standard, d.Name, n, d.Extent.MinimumBits, d.Extent.MaximumBits)
		}
	}
}
