package registry

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
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
			ok, err := probeCanonical(definition, vector)
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
				contextDefinition := definition
				contextDefinition.Decode = func(b []byte) (any, error) { return definition.DecodeWithContext(b, context) }
				contextDefinition.Encode = func(v any) ([]byte, error) { return definition.EncodeWithContext(v, context) }
				contextDefinition.Canonical = func(v any) ([]byte, error) { return definition.CanonicalWithContext(v, context) }
				contextDefinition.CanonicalAtLength = nil
				contextDefinition.MaxOctets = 20
				ok, err := probeCanonical(contextDefinition, bytes.Repeat([]byte{0x2b}, 20))
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

func probeCanonical(definition runtime.Descriptor, wire []byte) (accepted bool, err error) {
	decode, plain, encode := definition.Decode, definition.Encode, definition.Canonical
	atLength, maxOctets := definition.CanonicalAtLength, definition.MaxOctets
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
	if plain == nil {
		return true, fmt.Errorf("missing plain encoder")
	}
	replay, replayErr := plain(value)
	if replayErr != nil {
		return true, fmt.Errorf("plain encode: %w", replayErr)
	}
	if !bytes.Equal(replay, wire) {
		return true, fmt.Errorf("plain encoding changed received bytes: got %x, want %x", replay, wire)
	}
	canonical, err := encode(value)
	var extent *runtime.ExtentError
	if errors.As(err, &extent) && extent.Required && atLength != nil {
		// These component definitions have no standalone source maximum.
		// Supply an extent explicitly, first using the fresh value's own
		// length, then the caller's input extent as a search ceiling.
		fresh, freshErr := plain(runtime.Canonical(value))
		if freshErr == nil {
			canonical, err = atLength(value, len(fresh))
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
		if canonicalCapabilityDefinition(definition) && canonicalCapabilityReservation(err) {
			return true, nil
		}
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

// TS 24.008 V20.1.0 §§10.5.1.6, 10.5.1.7 and 10.5.5.12a
// preserve received capability codes while their canonical sender encodings
// reject reserved source-table values. GERAN containers delegate to those IEs
// (TS 36.331 V19.4.0 UE-CapabilityRAT-ContainerList field descriptions).
func canonicalCapabilityDefinition(definition runtime.Descriptor) bool {
	switch definition.Standard + "/" + definition.Clause {
	case "TS 24.008/10.5.1.6":
		return definition.Name == "Mobile Station Classmark 2 value part"
	case "TS 24.008/10.5.1.7":
		switch definition.Name {
		case "Classmark 3 Value part", "HSCSD Multi Slot Capability", "8-PSK Struct", "Single Band Support":
			return true
		}
	case "TS 24.008/10.5.5.12a":
		switch definition.Name {
		case "MS RA capability value part", "MS RA capability value part struct", "Access capabilities struct", "Content", "Multislot capability struct", "Enhanced Flexible Timeslot Assignment struct", "DLMC Capability struct":
			return true
		}
	case "TS 36.331/UE-CapabilityRAT-ContainerList field descriptions":
		return definition.Name == "geran-cs" || definition.Name == "geran-ps"
	}
	return false
}

// Generated sender constraints use an untyped error; keep this exception
// limited to the exact generator marker and the two Classmark 2 constraints.
func canonicalCapabilityReservation(err error) bool {
	switch err.Error() {
	case "value reserved by source table", "value reserved by source table: TS 24.008 table 10.5.6a Revision level", "value reserved by source table: TS 24.008 table 10.5.6a RF Power Capability":
		return true
	}
	return false
}

// TestProbeCanonicalCapabilityReservation keeps the receiver replay oracle
// active before allowing a sender-side capability reservation.
func TestProbeCanonicalCapabilityReservation(t *testing.T) {
	var definition runtime.Descriptor
	for _, candidate := range descriptors() {
		if candidate.Standard == "TS 24.008" && candidate.Clause == "10.5.1.6" {
			definition = candidate
			break
		}
	}
	if definition.Decode == nil {
		t.Fatal("missing Classmark 2 descriptor")
	}
	wire := []byte{0, 0, 0} // TS 24.008 table 10.5.6a: revision level 0.
	accepted, err := probeCanonical(definition, wire)
	if err != nil || !accepted {
		t.Fatalf("reserved receiver capability: accepted=%v error=%v", accepted, err)
	}
	for _, tc := range []struct {
		name   string
		change func(*runtime.Descriptor)
		want   string
	}{
		{"unrelated definition", func(d *runtime.Descriptor) { d.Standard = "TS 44.018" }, "encode:"},
		{"unaffected capability definition", func(d *runtime.Descriptor) { d.Name = "A5 bits" }, "encode:"},
		{"unexpected canonical error", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("unexpected") }
		}, "encode: unexpected"},
		{"reservation prefix lookalike", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("value reserved by source table but unexpected") }
		}, "but unexpected"},
		{"unexpected value error", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) {
				return nil, &runtime.DecodeError{Kind: runtime.InvalidValue, Detail: "unexpected value"}
			}
		}, "unexpected value"},
		{"plain replay failure", func(d *runtime.Descriptor) {
			d.Encode = func(any) ([]byte, error) { return nil, errors.New("plain failure") }
		}, "plain encode: plain failure"},
		{"plain replay mismatch", func(d *runtime.Descriptor) { d.Encode = func(any) ([]byte, error) { return []byte{0}, nil } }, "plain encoding changed received bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := definition
			tc.change(&candidate)
			_, err := probeCanonical(candidate, wire)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want failure containing %q", err, tc.want)
			}
		})
	}
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
		_, err := probeCanonical(definition, wire)
		if err != nil {
			t.Fatalf("%s §%s <%s>: %v", definition.Standard, definition.Clause, definition.Name, err)
		}
	})
}
