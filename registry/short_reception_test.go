package registry_test

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestShortReceptionRequiresReplayAndCanonicalExtent(t *testing.T) {
	d := runtime.Descriptor{
		Extent:    runtime.Extent{MinimumBits: 9, MaximumBits: 16, Finite: true},
		Encode:    func(any) ([]byte, error) { return []byte{0}, nil },
		Canonical: func(any) ([]byte, error) { return []byte{0, 0}, nil },
	}
	if _, err := canonicalAfterShortReplay(d, nil, []byte{0}, 8, runtime.SI4ACSZero); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*runtime.Descriptor)
	}{
		{"replay failure", func(d *runtime.Descriptor) {
			d.Encode = func(any) ([]byte, error) { return nil, errors.New("replay failure") }
		}},
		{"replay mismatch", func(d *runtime.Descriptor) { d.Encode = func(any) ([]byte, error) { return []byte{1}, nil } }},
		{"canonical failure", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("canonical failure") }
		}},
		{"canonical underflow", func(d *runtime.Descriptor) { d.Canonical = func(any) ([]byte, error) { return []byte{0}, nil } }},
		{"canonical overflow", func(d *runtime.Descriptor) { d.Canonical = func(any) ([]byte, error) { return []byte{0, 0, 0}, nil } }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := d
			tc.change(&changed)
			if _, err := canonicalAfterShortReplay(changed, nil, []byte{0}, 8, runtime.SI4ACSZero); err == nil {
				t.Fatal("invalid short reception accepted")
			}
		})
	}
	for _, bits := range []int{-1, 9} {
		if _, err := canonicalAfterShortReplay(d, nil, []byte{0}, bits, runtime.SI4ACSZero); err == nil {
			t.Fatal("invalid significant extent accepted")
		}
	}
	d.Encode = func(any) ([]byte, error) { return []byte{0xf8}, nil }
	if _, err := canonicalAfterShortReplay(d, nil, []byte{0xff}, 5, runtime.SI4ACSZero); err != nil {
		t.Fatal(err)
	}
}
