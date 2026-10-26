package uecapability

import (
	"bytes"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"testing"
)

func TestTypedManualNestedOperations(t *testing.T) {
	for _, offset := range []int{0, 1, 8, 17} {
		t.Run(string(rune('A'+offset)), func(t *testing.T) {
			raw := []byte{0x20, 0, 0}
			input := runtime.NewWriter()
			if err := input.WriteUint(0, offset); err != nil {
				t.Fatal(err)
			}
			for _, b := range raw {
				if err := input.WriteUint(uint64(b), 8); err != nil {
					t.Fatal(err)
				}
			}
			if err := input.WritePaddingTo((offset + 24 + 7) / 8 * 8); err != nil {
				t.Fatal(err)
			}
			wire, err := input.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			r := runtime.NewReader(wire)
			if _, err := r.ReadUint(offset); err != nil {
				t.Fatal(err)
			}
			v, err := DecodeClassmark2ValuePartFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if r.Position() != offset+24 {
				t.Fatal("wrong nested extent")
			}
			w := runtime.NewWriter()
			if err := w.WriteUint(0, offset); err != nil {
				t.Fatal(err)
			}
			if err := EncodeClassmark2ValuePartTo(w, v); err != nil {
				t.Fatal(err)
			}
			if err := w.WritePaddingTo((offset + 24 + 7) / 8 * 8); err != nil {
				t.Fatal(err)
			}
			actual, err := w.Bytes()
			if err != nil || !bytes.Equal(actual, wire) {
				t.Fatalf("nested Classmark 2: %x %v", actual, err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		wire []byte
		from func(*runtime.Reader) (any, error)
		to   func(*runtime.Writer, any) error
	}{
		{"CS", []byte{0x33, 3, 0x20, 0, 0, 0}, func(r *runtime.Reader) (any, error) { return DecodeGERANCSFrom(r) }, func(w *runtime.Writer, v any) error { return EncodeGERANCSTo(w, v.(GERANCS)) }},
		{"PS", []byte{0x10, 0xb1, 0}, func(r *runtime.Reader) (any, error) { return DecodeGERANPSFrom(r) }, func(w *runtime.Writer, v any) error { return EncodeGERANPSTo(w, v.(GERANPS)) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := runtime.NewReader(append(append([]byte{0xff}, tc.wire...), 0xa5))
			_, _ = r.ReadUint(8)
			if _, err := r.PushLimit(len(tc.wire) * 8); err != nil {
				t.Fatal(err)
			}
			v, err := tc.from(r)
			if err != nil {
				t.Fatal(err)
			}
			w := runtime.NewWriter()
			if err := w.WriteUint(0xff, 8); err != nil {
				t.Fatal(err)
			}
			if err := tc.to(w, v); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(0xa5, 8); err != nil {
				t.Fatal(err)
			}
			actual, err := w.Bytes()
			expected := append(append([]byte{0xff}, tc.wire...), 0xa5)
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("nested %s: got %x want %x: %v", tc.name, actual, expected, err)
			}
		})
	}
}
func TestManualNestedErrorsAreAbsolute(t *testing.T) {
	r := runtime.NewReader([]byte{0xff, 0, 0})
	_, _ = r.ReadUint(8)
	_, err := DecodeClassmark2ValuePartFrom(r)
	var op *runtime.OperationError
	var de *runtime.DecodeError
	if !errors.As(err, &op) || !errors.As(err, &de) || op.Location.BitOffset != 24 || de.Offset != 24 || op.Location.Source.Standard != "TS 24.008" {
		t.Fatalf("truncated manual origin: %+v", err)
	}

	_, baseError := msrac.DecodeMSRACapabilityValuePart([]byte{0x1a, 0x13})
	var baseDecode *runtime.DecodeError
	if !errors.As(baseError, &baseDecode) {
		t.Fatal("missing synthetic truncation control", baseError)
	}
	r = runtime.NewReader([]byte{0xff, 0x1a, 0x13})
	_, _ = r.ReadUint(8)
	_, _ = r.PushLimit(16)
	_, err = DecodeGERANPSFrom(r)
	if !errors.As(err, &op) || !errors.As(err, &de) || op.Location.BitOffset != baseDecode.Offset+8 || de.Offset != baseDecode.Offset+8 {
		t.Fatalf("nested capability origin: %+v; base %+v", err, baseError)
	}
}
