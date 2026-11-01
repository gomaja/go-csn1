package registry_test

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/neighbourcelldata"
	"github.com/gomaja/go-csn1/ts44060/servingcelldata"
)

// TS 44.060 V19.0.0 §§11.2.9e pp16187–16197 and 11.2.17a
// pp16647–16657 assign PD/CD_LENGTH sender codes. Their continuation rows
// consume an explicit containing extent; the reviewed readings preserve
// received reserved codes for replay only.
func containerSenderErrorFamilies(t *testing.T) {
	t.Helper()
	for _, definitions := range [][]runtime.Descriptor{neighbourcelldata.Descriptors(), servingcelldata.Descriptors()} {
		for _, d := range definitions {
			if d.Name != "Container repetition struct" {
				continue
			}

			// Exported wire metadata can be edited after sealing. Reject it
			// before subtracting the writer origin from the received extent.
			for _, bits := range []int{math.MinInt, -1, 0, 7, 1<<20 + 1, math.MaxInt} {
				decoded, err := d.Decode([]byte{31, 0xa5})
				if err != nil {
					t.Fatal(err)
				}
				value := reflect.ValueOf(decoded).FieldByName("Value")
				edited := reflect.New(value.Type()).Elem()
				edited.Set(value)
				wire := edited.FieldByName("Wire").Interface().(runtime.WireInfo)
				wire.TransmittedBits = bits
				edited.FieldByName("Wire").Set(reflect.ValueOf(wire))
				_, err = d.Encode(edited.Interface())
				requireContainerSenderFamily(t, d, err, "invalid received continuation extent", false)
			}
			for _, tc := range []struct {
				name    string
				wire    []byte
				message string
				context bool
			}{
				{"reserved PD", []byte{0x41, 0xa5, 0}, "value outside sender code set", false},
				{"reserved finite length", append(append([]byte{19}, bytes.Repeat([]byte{0xa5}, 19)...), 0), "value outside sender code set", false},
				{"continuation context", []byte{0x1f, 0xa5}, "continuation octets require a containing extent", true},
			} {
				t.Run(d.Clause+"/"+tc.name, func(t *testing.T) {
					decoded, err := d.Decode(tc.wire)
					if err != nil {
						t.Fatal(err)
					}
					value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
					replay, err := d.Encode(value)
					if err != nil || !bytes.Equal(replay, tc.wire) {
						t.Fatalf("replay: %v", err)
					}
					_, err = d.Encode(runtime.Canonical(value))
					requireContainerSenderFamily(t, d, err, tc.message, tc.context)
					if !tc.context {
						_, err = d.CanonicalAtLength(value, len(tc.wire))
						requireContainerSenderFamily(t, d, err, tc.message, false)
					}
				})
			}
		}
	}
}

func requireContainerSenderFamily(t *testing.T, d runtime.Descriptor, err error, message string, context bool) {
	t.Helper()
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || operation.Kind != runtime.InvalidValue || operation.Location.Source.Standard != d.Standard || operation.Location.Source.Clause != d.Clause {
		t.Fatalf("typed sender error: %v", err)
	}
	if context {
		if !errors.Is(err, runtime.ErrContextRequired) {
			t.Fatalf("context sentinel: %v", err)
		}
	} else if errors.Is(err, runtime.ErrContextRequired) {
		t.Fatalf("unexpected context sentinel: %v", err)
	}
	for operation.Cause != nil {
		if inner, ok := operation.Cause.(*runtime.OperationError); ok {
			operation = inner
			continue
		}
		err = operation.Cause
		if context {
			if err.Error() != runtime.ErrContextRequired.Error()+": "+message {
				t.Fatalf("context family: %v", err)
			}
		} else if err.Error() != message {
			t.Fatalf("sender family: %v", err)
		}
		return
	}
	if operation.Detail != message {
		t.Fatalf("sender family: %v", operation)
	}
}
