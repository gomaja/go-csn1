package runtime

import (
	"errors"
	"testing"
)

type canonicalChild struct {
	Value uint8
	Wire  WireInfo
}

type canonicalParent struct {
	Child *canonicalChild
	List  []canonicalChild
	Wire  WireInfo
}

func TestCanonicalCopiesAndClearsNestedWire(t *testing.T) {
	original := canonicalParent{
		Child: &canonicalChild{Value: 7, Wire: WireInfo{BitsConsumed: 8, Tail: BitString{Bytes: []byte{0xaa}, BitLength: 8}}},
		List:  []canonicalChild{{Value: 9, Wire: WireInfo{BitsConsumed: 8}}},
		Wire:  WireInfo{BitsConsumed: 16},
	}
	canonical := Canonical(original)
	if canonical.Wire.BitsConsumed != 0 || canonical.Child.Wire.BitsConsumed != 0 || canonical.List[0].Wire.BitsConsumed != 0 {
		t.Fatalf("wire state retained: %+v", canonical)
	}
	if canonical.Child.Value != 7 || canonical.List[0].Value != 9 {
		t.Fatalf("semantic value changed: %+v", canonical)
	}
	canonical.Child.Value = 1
	canonical.List[0].Value = 2
	if original.Child.Value != 7 || original.List[0].Value != 9 || original.Child.Wire.BitsConsumed != 8 {
		t.Fatalf("original value modified: %+v", original)
	}
}

func TestCanonicalNilInterface(t *testing.T) {
	var value any
	if Canonical(value) != nil {
		t.Fatal("nil interface changed")
	}
}

func TestCanonicalExtentIsRequiredAndEnforced(t *testing.T) {
	encode := func(uint8) ([]byte, error) { return make([]byte, 22), nil }
	decode := func([]byte) (Decoded[uint8], error) { return Decoded[uint8]{Value: 7}, nil }
	for _, tc := range []struct {
		name         string
		run          func() error
		wantRequired bool
	}{
		{"unspecified", func() error { _, err := CanonicalEncode(uint8(7), 0, 0, false, encode, decode, nil); return err }, true},
		{"direct fresh candidate", func() error { _, err := CanonicalEncode(uint8(7), 0, 21, false, encode, decode, nil); return err }, false},
		{"explicit overlong target", func() error { _, err := CanonicalEncodeAtLength(uint8(7), 22, 0, 21, encode, decode, nil); return err }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var extent *ExtentError
			if err := tc.run(); !errors.As(err, &extent) || extent.Required != tc.wantRequired {
				t.Fatalf("want typed extent error with required=%t, got %v", tc.wantRequired, err)
			}
		})
	}
}
