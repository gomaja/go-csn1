package registry

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/measurement"
)

// TestCanonicalAllDefinitions exercises every generated entry point with
// deterministic synthetic valid and invalid input. TS 44.018 V19.0.0 §8.9
// permits truncated rest-octet prefixes; §9.1.55 permits only redundant
// trailing no-report bitmap positions to change during canonical encoding.
func TestCanonicalAllDefinitions(t *testing.T) {
	var vectors [][]byte
	for _, length := range []int{0, 1, 2, 3, 4, 5, 8, 11, 17, 20, 23, 50} {
		for _, fill := range []byte{0, 0x2b, 0xff} {
			vectors = append(vectors, bytes.Repeat([]byte{fill}, length))
		}
	}
	vectors = append(vectors,
		[]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0x19, 0x30, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	)
	defs := descriptors()
	var accepted, rejected, failures int
	for _, definition := range defs {
		if definition.Canonical == nil {
			t.Errorf("%s §%s <%s>: missing canonical descriptor", definition.Standard, definition.Clause, definition.Name)
			failures++
			continue
		}
		for index, vector := range vectors {
			ok, err := probeCanonical(definition.Decode, definition.Encode, definition.Canonical, definition.CanonicalAtLength, definition.MaxOctets, vector)
			if err != nil {
				t.Errorf("%s §%s <%s> vector %d: %v", definition.Standard, definition.Clause, definition.Name, index, err)
				failures++
			} else if ok {
				accepted++
			} else {
				rejected++
			}
		}
		if definition.DecodeWithContext != nil {
			if definition.CanonicalWithContext == nil {
				t.Errorf("%s <%s>: missing canonical context descriptor", definition.Standard, definition.Name)
				failures++
				continue
			}
			for _, context := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
				decode := func(b []byte) (any, error) { return definition.DecodeWithContext(b, context) }
				encode := func(v any) ([]byte, error) { return definition.CanonicalWithContext(v, context) }
				ok, err := probeCanonical(decode, nil, encode, nil, 20, bytes.Repeat([]byte{0x2b}, 20))
				if err != nil || !ok {
					t.Errorf("%s <%s> ACS=%d: accepted=%v error=%v", definition.Standard, definition.Name, context, ok, err)
					failures++
				} else {
					accepted++
				}
			}
		}
	}
	t.Logf("definitions=%d vectors=%d accepted=%d rejected=%d failures=%d", len(defs), len(vectors), accepted, rejected, failures)
	if len(defs) != 235 || len(vectors) != 38 {
		t.Fatalf("unexpected probe breadth: %d definitions and %d vectors", len(defs), len(vectors))
	}
}

func probeCanonical(decode func([]byte) (any, error), plain, encode func(any) ([]byte, error), atLength func(any, int) ([]byte, error), maxOctets int, wire []byte) (accepted bool, err error) {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("panic: %v", panicValue)
		}
	}()
	decoded, err := decode(wire)
	if err != nil {
		return false, nil
	}
	value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
	canonical, err := encode(value)
	var extent *runtime.ExtentError
	if errors.As(err, &extent) && extent.Required && atLength != nil {
		// These component definitions have no standalone source maximum.
		// Supply an extent explicitly, first using the fresh value's own
		// length, then the caller's input extent as a search ceiling.
		if plain != nil {
			fresh, freshErr := plain(runtime.Canonical(value))
			if freshErr == nil {
				canonical, err = atLength(value, len(fresh))
			}
		}
		if err != nil {
			for octets := 0; octets <= len(wire); octets++ {
				candidate, candidateErr := atLength(value, octets)
				if candidateErr == nil {
					canonical, err = candidate, nil
					break
				}
			}
		}
	}
	if err != nil {
		return true, fmt.Errorf("encode: %w", err)
	}
	if maxOctets > 0 && len(canonical) > maxOctets {
		return true, fmt.Errorf("canonical output %d exceeds source maximum %d octets", len(canonical), maxOctets)
	}
	again, err := decode(canonical)
	if err != nil {
		return true, fmt.Errorf("decode canonical: %w", err)
	}
	other := reflect.ValueOf(again).FieldByName("Value").Interface()
	if a, ok := value.(measurement.EnhancedMeasurementReport); ok {
		b := other.(measurement.EnhancedMeasurementReport)
		return true, compareEMRProbe(a, b)
	}
	if !runtime.SemanticallyEqual(value, other) {
		return true, fmt.Errorf("canonical encoding changed typed semantics")
	}
	return true, nil
}

func compareEMRProbe(a, b measurement.EnhancedMeasurementReport) error {
	normalize := func(v measurement.EnhancedMeasurementReport) measurement.EnhancedMeasurementReport {
		v = runtime.Canonical(v)
		if v.REPORTINGQUANTITYList != nil {
			list := append([]*uint8(nil), (*v.REPORTINGQUANTITYList)...)
			for len(list) < 96 {
				list = append(list, nil)
			}
			v.REPORTINGQUANTITYList = &list
		}
		return v
	}
	if !runtime.SemanticallyEqual(normalize(a), normalize(b)) {
		return fmt.Errorf("canonical EMR changed typed semantics")
	}
	return nil
}

func FuzzCanonicalAllDefinitions(f *testing.F) {
	for _, seed := range []struct {
		index uint16
		wire  []byte
	}{
		{0, []byte{0}}, {12, []byte{0}}, {27, []byte{0x10, 0x02}},
		{100, bytes.Repeat([]byte{0x2b}, 20)}, {200, []byte{0xff, 0xff}},
	} {
		f.Add(seed.index, seed.wire)
	}
	definitions := descriptors()
	f.Fuzz(func(t *testing.T, index uint16, wire []byte) {
		if len(wire) > 50 {
			return
		}
		definition := definitions[int(index)%len(definitions)]
		if definition.Canonical == nil {
			t.Fatalf("missing canonical entry point for %s", definition.Name)
		}
		_, err := probeCanonical(definition.Decode, definition.Encode, definition.Canonical, definition.CanonicalAtLength, definition.MaxOctets, wire)
		if err != nil {
			t.Fatalf("%s §%s <%s>: %v", definition.Standard, definition.Clause, definition.Name, err)
		}
	})
}
