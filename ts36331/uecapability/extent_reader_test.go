package uecapability

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestManualReadersRejectBelowMinimumExtent(t *testing.T) {
	for _, tc := range []struct {
		name        string
		data        []byte
		bits        int
		kind        runtime.ErrorKind
		decodeBytes func([]byte) error
		decodeFrom  func(*runtime.Reader) error
	}{
		{"Classmark 2", []byte{0x20, 0, 0}, 23, runtime.Truncated,
			func(b []byte) error { _, err := DecodeClassmark2ValuePart(b); return err },
			func(r *runtime.Reader) error { _, err := DecodeClassmark2ValuePartFrom(r); return err }},
		{"GERAN CS without Classmark 3", []byte{0x33, 3, 0x57, 0x18, 0xa0, 0}, 40, runtime.InvalidValue,
			func(b []byte) error { _, err := DecodeGERANCS(b); return err },
			func(r *runtime.Reader) error { _, err := DecodeGERANCSFrom(r); return err }},
		{"GERAN CS one bit short", []byte{0x33, 3, 0x57, 0x18, 0xa0, 0}, 47, runtime.InvalidValue,
			func(b []byte) error { _, err := DecodeGERANCS(b); return err },
			func(r *runtime.Reader) error { _, err := DecodeGERANCSFrom(r); return err }},
		{"GERAN PS", []byte{0x10, 0x01}, 15, runtime.Truncated,
			func(b []byte) error { _, err := DecodeGERANPS(b); return err },
			func(r *runtime.Reader) error { _, err := DecodeGERANPSFrom(r); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.decodeBytes(tc.data[:len(tc.data)-1]); err == nil {
				t.Fatal("byte decoder accepted a shorter container")
			}
			r := runtime.NewReader(tc.data)
			if _, err := r.PushLimit(tc.bits); err != nil {
				t.Fatal(err)
			}
			err := tc.decodeFrom(r)
			var op *runtime.OperationError
			if !errors.As(err, &op) || op.Kind != tc.kind {
				t.Fatalf("reader accepted %d bits or returned wrong error: %v", tc.bits, err)
			}
		})
	}
}

func TestManualCanonicalLengthOperationsEnforceExtents(t *testing.T) {
	for _, tc := range []struct {
		name             string
		minimum, maximum int
		encode           func(int) error
	}{
		{"Classmark 2", 3, 3, func(n int) error {
			_, err := EncodeClassmark2ValuePartCanonicalAtLength(Classmark2ValuePart{}, n)
			return err
		}},
		{"GERAN CS", 6, 37, func(n int) error {
			_, err := EncodeGERANCSCanonicalAtLength(GERANCS{}, n)
			return err
		}},
		{"GERAN PS", 2, 50, func(n int) error {
			_, err := EncodeGERANPSCanonicalAtLength(GERANPS{}, n)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{tc.minimum - 1, tc.maximum + 1} {
				var extent *runtime.ExtentError
				if err := tc.encode(n); !errors.As(err, &extent) || extent.Actual != n || extent.Minimum != tc.minimum || extent.Maximum != tc.maximum {
					t.Fatalf("length %d: expected extent %d..%d, got %v", n, tc.minimum, tc.maximum, err)
				}
			}
		})
	}
}
