package runtime

import (
	"errors"
	"reflect"
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

// A plain encoder checks a definition's minimum on its own; its message must
// not invent a zero maximum.
func TestExtentErrorMessages(t *testing.T) {
	for _, tc := range []struct {
		err  ExtentError
		want string
	}{
		{ExtentError{Required: true}, "CSN.1 canonical encoding requires an explicit containing-message length"},
		{ExtentError{Actual: 5, Minimum: 20}, "CSN.1 extent 5 below minimum 20 octets"},
		{ExtentError{Actual: 22, Maximum: 21}, "CSN.1 extent 22 outside 0..21 octets"},
		{ExtentError{Actual: 6, Minimum: 10, Maximum: 10}, "CSN.1 extent 6 outside 10..10 octets"},
	} {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("%+v: %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestCanonicalCandidateOrderAndFailurePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name              string
		shortest, freshOK bool
		want              []int
	}{
		{"fresh succeeds", false, true, []int{-1}},
		{"search after fresh failure", false, false, []int{-1, 8, 16}},
		{"shortest before fresh", true, true, []int{8, 16}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []int
			value := canonicalChild{Value: 7, Wire: WireInfo{BitsConsumed: 99}}
			encode := func(v canonicalChild) ([]byte, error) {
				if !v.Wire.canonical || !v.Wire.rawErrors || v.Wire.BitsConsumed != 0 {
					t.Fatalf("candidate retained received state: %+v", v.Wire)
				}
				target := -1
				if v.Wire.targetSet {
					target = v.Wire.targetBits
				}
				calls = append(calls, target)
				if target == 16 || target == -1 && tc.freshOK {
					return []byte{7, 0}, nil
				}
				return nil, ErrBoundConflict
			}
			decode := func([]byte) (Decoded[canonicalChild], error) {
				return Decoded[canonicalChild]{Value: canonicalChild{Value: 7}}, nil
			}
			out, err := CanonicalEncode(value, 1, 2, tc.shortest, encode, decode, nil)
			if err != nil || len(out) != 2 || !reflect.DeepEqual(calls, tc.want) {
				t.Fatalf("out=%x err=%v calls=%v, want %v", out, err, calls, tc.want)
			}
			if value.Wire.BitsConsumed != 99 {
				t.Fatal("canonical encoding mutated received state")
			}
		})
	}
	cause := errors.New("fresh encoding failed")
	var raw, located int
	encode := func(v canonicalChild) ([]byte, error) {
		if v.Wire.rawErrors {
			raw++
			return nil, cause
		}
		located++
		return nil, ErrBoundConflict
	}
	decode := func([]byte) (Decoded[canonicalChild], error) {
		t.Fatal("failed encoder invoked decoder")
		return Decoded[canonicalChild]{}, nil
	}
	_, err := CanonicalEncode(canonicalChild{}, 1, 2, false, encode, decode, nil)
	if !errors.Is(err, ErrBoundConflict) || raw != 3 || located != 1 {
		t.Fatalf("err=%v raw=%d located=%d", err, raw, located)
	}
}

func TestCanonicalExactLengthRejectsBeforeDecoding(t *testing.T) {
	var encodes, decodes int
	encode := func(v canonicalChild) ([]byte, error) {
		encodes++
		if !v.Wire.targetSet || v.Wire.targetBits != 16 || v.Wire.rawErrors {
			t.Fatalf("exact target %+v", v.Wire)
		}
		return []byte{7}, nil
	}
	decode := func([]byte) (Decoded[canonicalChild], error) { decodes++; return Decoded[canonicalChild]{}, nil }
	_, err := CanonicalEncodeAtLength(canonicalChild{}, 3, 1, 2, encode, decode, nil)
	if encodes != 0 || decodes != 0 {
		t.Fatal("invalid extent reached a codec")
	}
	var extent *ExtentError
	if !errors.As(err, &extent) || extent.Actual != 3 || extent.Minimum != 1 || extent.Maximum != 2 {
		t.Fatal(err)
	}
	_, err = CanonicalEncodeAtLength(canonicalChild{}, 2, 1, 2, encode, decode, nil)
	if encodes != 1 || decodes != 0 || !errors.As(err, &extent) || extent.Actual != 1 || extent.Minimum != 2 || extent.Maximum != 2 {
		t.Fatalf("err=%v encodes=%d decodes=%d", err, encodes, decodes)
	}
}
