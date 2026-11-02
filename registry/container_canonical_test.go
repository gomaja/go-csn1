package registry

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.2.9e pp16161–16172,16195 and 11.2.17a
// pp16628–16639,16655: CD_LENGTH=31 consumes the containing remainder.
// The content headers occupy 18 bits; the helper starts directly at PD.
// Exact bit bounds keep these synthetic witnesses independent of byte padding.
var containerCanonicalFixtures = []struct{ clause, name, prefix string }{
	{"11.2.9e", "Container repetition struct", ""},
	{"11.2.9e", "Packet Neighbour Cell Data message content", "000000000100100010"},
	{"11.2.17a", "Container repetition struct", ""},
	{"11.2.17a", "Packet Serving Cell Data message content", "000000000000010001"},
}

func containerCanonicalLengthSeed(t *testing.T, d runtime.Descriptor) (any, bool) {
	t.Helper()
	if d.Standard != "TS 44.060" {
		return nil, false
	}
	for _, fixture := range containerCanonicalFixtures {
		if fixture.clause != d.Clause || fixture.name != d.Name {
			continue
		}
		bits := fixture.prefix + "0001111110100101" // PD=0, continuation, one data octet.
		data := make([]byte, (len(bits)+7)/8)
		for i, bit := range bits {
			if bit == '1' {
				data[i/8] |= 1 << uint(7-i%8)
			}
		}
		r := sourceReader(t, d, data)
		if _, err := r.PushLimit(len(bits)); err != nil {
			t.Fatal(err)
		}
		value, err := d.DecodeFrom(r)
		if err != nil || r.Position() != SourceOrigin(d)+len(bits) {
			t.Fatalf("%s %s witness decode at %d/%d: %v", d.Clause, d.Name, r.Position(), len(bits), err)
		}
		return value, true
	}
	return nil, false
}

func TestContainerCanonicalLengthWitnesses(t *testing.T) {
	count := 0
	for _, d := range descriptors() {
		value, ok := containerCanonicalLengthSeed(t, d)
		if !ok {
			continue
		}
		count++
		_, err := d.CanonicalAtLength(value, 6)
		var bound *runtime.BoundError
		var operation *runtime.OperationError
		if !errors.As(err, &bound) || bound.Kind != runtime.LengthBound || !errors.As(err, &operation) || operation.Location.Source.Clause != d.Clause {
			t.Errorf("%s %s bound-length witness: %v", d.Clause, d.Name, err)
		}
	}
	if count != 4 {
		t.Fatalf("length witnesses=%d, want 4", count)
	}
}

func TestProbeCanonicalContainers(t *testing.T) {
	count := 0
	for _, d := range descriptors() {
		if d.Standard != "TS 44.060" || d.Clause != "11.2.9e" && d.Clause != "11.2.17a" {
			continue
		}
		count++
		wire := []byte{0xff} // Reserved PD and an empty continuation at helper level.
		if d.Name != "Container repetition struct" {
			wire = []byte{0}
		} // Complete received error region.

		decoded, err := d.Decode(wire)
		if err != nil {
			t.Fatal(err)
		}
		value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
		_, err = d.Canonical(value)
		var extent *runtime.ExtentError
		var operation *runtime.OperationError
		if !errors.As(err, &extent) || !extent.Required || !errors.As(err, &operation) || operation.Operation != "canonical" {
			t.Fatalf("typed containing-length requirement: %v", err)
		}
		accepted, err := probeCanonical(d, wire)
		if !accepted || err != nil {
			t.Errorf("%s %s: accepted=%v error=%v", d.Clause, d.Name, accepted, err)
		}
	}
	if count != 4 {
		t.Fatalf("container probes=%d", count)
	}
}

func TestProbeCanonicalContainerRefusalsStayNarrow(t *testing.T) {
	var definition runtime.Descriptor
	for _, d := range descriptors() {
		if d.Clause == "11.2.9e" && d.Name == "Container repetition struct" {
			definition = d
			break
		}
	}
	if definition.Decode == nil {
		t.Fatal("missing container descriptor")
	}
	wire := []byte{0xff}
	for _, tc := range []struct {
		name   string
		change func(*runtime.Descriptor)
	}{
		{"unrelated clause", func(d *runtime.Descriptor) { d.Clause = "12.1" }},
		{"unrelated name", func(d *runtime.Descriptor) { d.Name = "other" }},
		{"unrelated standard", func(d *runtime.Descriptor) { d.Standard = "other" }},
		{"unexpected canonical error", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("unexpected") }
		}},
		{"untyped reservation", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("value outside sender code set") }
		}},
		{"unexpected extent error", func(d *runtime.Descriptor) {
			d.CanonicalAtLength = func(any, int) ([]byte, error) { return nil, errors.New("unexpected extent") }
		}},
		{"plain replay error", func(d *runtime.Descriptor) {
			d.Encode = func(any) ([]byte, error) { return nil, errors.New("plain failure") }
		}},
		{"plain replay mismatch", func(d *runtime.Descriptor) { d.Encode = func(any) ([]byte, error) { return []byte{0}, nil } }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := definition
			tc.change(&d)
			if _, err := probeCanonical(d, wire); err == nil {
				t.Fatal("accepted unexpected error or changed replay")
			}
		})
	}
	for _, err := range []error{
		nil,
		errors.New("value outside sender code set"),
		&runtime.OperationError{Operation: "canonical", Kind: runtime.InvalidSchema, Detail: "value outside sender code set", Location: runtime.Location{Source: runtime.Source{Standard: "TS 44.060", Clause: "11.2.9e"}}},
		&runtime.OperationError{Operation: "canonical", Kind: runtime.InvalidValue, Detail: "value outside sender code set plus another failure", Location: runtime.Location{Source: runtime.Source{Standard: "TS 44.060", Clause: "11.2.9e"}}},
	} {
		if canonicalContainerReservation(definition, err) {
			t.Fatalf("accepted unexpected reservation: %v", err)
		}
	}
}
