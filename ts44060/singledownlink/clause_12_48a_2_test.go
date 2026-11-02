package singledownlink

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §12.48a.2 defines these synthetic codewords. The caller owns the exact IE
// extent. TS 44.060 V19.0.0 §§11.1.2–11.1.3.2 define the labelled remainder:
// preserve the accepted typed prefix, consume only its containing region and
// refuse the designation when sending a fresh or canonical value.
type assignmentVector struct {
	name, bits  string
	errorOffset int
	fields      map[string]uint64
}

func assignmentVectors() []assignmentVector {
	return []assignmentVector{
		{"False-0-0-0", "0100100101001010010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-1-0", "01001010001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-1-1", "01001010011001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-1-2", "010010101010100101001111001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-1-3-0", "0100101011", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-1-3-1", "01001010111", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-1-3-5", "010010101110101", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-1-3-33", "0100101011101101101101101101101101101101101", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-2-0", "010010110010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-2-1", "010010110110010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-2-2", "01001011101010010100111100100000010111111010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-0-2-3-0", "0100101111", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-2-3-1", "01001011111", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-2-3-5", "010010111110101", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-0-2-3-33", "0100101111101101101101101101101101101101101", 10, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-0-0", "01001100101001010010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-1-0", "010011010001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-1-1", "010011010011001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-1-2", "0100110101010100101001111001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-1-3-0", "01001101011", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-1-3-1", "010011010111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-1-3-5", "0100110101110101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-1-3-33", "01001101011101101101101101101101101101101101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-2-0", "0100110110010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-2-1", "0100110110110010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-2-2", "010011011101010010100111100100000010111111010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-1-2-3-0", "01001101111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-2-3-1", "010011011111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-2-3-5", "0100110111110101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-1-2-3-33", "01001101111101101101101101101101101101101101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-0-0", "01001110101001010010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-1-0", "010011110001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-1-1", "010011110011001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-1-2", "0100111101010100101001111001001010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-1-3-0", "01001111011", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-1-3-1", "010011110111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-1-3-5", "0100111101110101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-1-3-33", "01001111011101101101101101101101101101101101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-2-0", "0100111110010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-2-1", "0100111110110010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-2-2", "010011111101010010100111100100000010111111010010110010100110", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"False-2-2-3-0", "01001111111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-2-3-1", "010011111111", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-2-3-5", "0100111111110101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"False-2-2-3-33", "01001111111101101101101101101101101101101101", 11, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-0-0", "10001110010010100101100111100100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-1-0", "1000111001010001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-1-1", "1000111001010011001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-1-2", "10001110010101010100101001111001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-1-3-0", "100011100101011", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-1-3-1", "1000111001010111", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-1-3-5", "10001110010101110101", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-1-3-33", "100011100101011101101101101101101101101101101101", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-2-0", "10001110010110010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-2-1", "10001110010110110010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-2-2", "1000111001011101010010100111100100000010111111010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-0-2-3-0", "100011100101111", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-2-3-1", "1000111001011111", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-2-3-5", "10001110010111110101", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-0-2-3-33", "100011100101111101101101101101101101101101101101", 15, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-0-0", "100011100110010100101100111100100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-1-0", "10001110011010001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-1-1", "10001110011010011001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-1-2", "100011100110101010100101001111001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-1-3-0", "1000111001101011", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-1-3-1", "10001110011010111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-1-3-5", "100011100110101110101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-1-3-33", "1000111001101011101101101101101101101101101101101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-2-0", "100011100110110010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-2-1", "100011100110110110010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-2-2", "10001110011011101010010100111100100000010111111010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-1-2-3-0", "1000111001101111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-2-3-1", "10001110011011111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-2-3-5", "100011100110111110101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-1-2-3-33", "1000111001101111101101101101101101101101101101101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-0-0", "100011100111010100101100111100100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-1-0", "10001110011110001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-1-1", "10001110011110011001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-1-2", "100011100111101010100101001111001001100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-1-3-0", "1000111001111011", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-1-3-1", "10001110011110111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-1-3-5", "100011100111101110101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-1-3-33", "1000111001111011101101101101101101101101101101101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-2-0", "100011100111110010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-2-1", "100011100111110110010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-2-2", "10001110011111101010010100111100100000010111111010010110100001111010011101000", -1, map[string]uint64{"CONTROLACK": 1, "DOWNLINKTFIASSIGNMENT": 9, "EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2, "RLCMODE": 1}},
		{"True-2-2-3-0", "1000111001111111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-2-3-1", "10001110011111111", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-2-3-5", "100011100111111110101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
		{"True-2-2-3-33", "1000111001111111101101101101101101101101101101101", 16, map[string]uint64{"EGPRSLevel": 1, "LINKQUALITYMEASUREMENTMODE": 2}},
	}
}
func assignmentBits(s string) []byte {
	out := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func assignmentFields(v reflect.Value, out map[string][]uint64) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			assignmentFields(v.Elem(), out)
		}
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			n := v.Type().Field(i).Name
			if n == "Wire" || n == "Alternative" || n == "Location" || n == "Bits" {
				continue
			}
			f := v.Field(i)
			switch f.Kind() {
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				out[n] = append(out[n], f.Uint())
			default:
				assignmentFields(f, out)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			assignmentFields(v.Index(i), out)
		}
	}
}
func assignmentClearWire(v reflect.Value) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			assignmentClearWire(v.Elem())
		}
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if v.Type().Field(i).Name == "Wire" && f.CanSet() {
				f.SetZero()
			} else {
				assignmentClearWire(f)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			assignmentClearWire(v.Index(i))
		}
	}
}
func TestClause12_48a_2(t *testing.T) {
	d, err := Lookup("Single Downlink Assignment 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range assignmentVectors() {
		t.Run(tc.name, func(t *testing.T) {
			for phase := 0; phase < 8; phase++ {
				prefix := strings.Repeat("0", phase)
				input := assignmentBits(prefix + tc.bits + "101")
				unbounded := runtime.NewReader(input)
				if _, err := unbounded.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeSingleDownlinkAssignment2IEFrom(unbounded); !errors.Is(err, runtime.ErrContextRequired) || unbounded.Position() != phase {
					t.Fatalf("missing caller extent advanced to %d: %v", unbounded.Position(), err)
				}
				c := runtime.NewReceiverCollector()
				r, err := runtime.NewChoiceReaderBits(input, phase+len(tc.bits)+3, c)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(tc.bits))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := d.DecodeFrom(r)
				if err != nil {
					t.Fatalf("phase %d descriptor decode: %v", phase, err)
				}
				value := decoded.(runtime.Decoded[SingleDownlinkAssignment2IE]).Value
				componentReader, err := runtime.NewChoiceReaderBits(input, phase+len(tc.bits)+3, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := componentReader.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				if _, err := componentReader.PushLimit(len(tc.bits)); err != nil {
					t.Fatal(err)
				}
				component, err := DecodeSingleDownlinkAssignment2IEFrom(componentReader)
				if err != nil || !runtime.SemanticallyEqual(value, component) || componentReader.Position() != r.Position() {
					t.Fatalf("typed component parity: %v", err)
				}
				if r.Position() != phase+len(tc.bits) {
					t.Fatalf("consumed %d want %d", r.Position(), phase+len(tc.bits))
				}
				fields := map[string][]uint64{}
				assignmentFields(reflect.ValueOf(value), fields)
				for key, want := range tc.fields {
					if len(fields[key]) == 0 || fields[key][0] != want {
						t.Fatalf("typed %s=%v want first %d", key, fields[key], want)
					}
				}
				report := c.TakeReceiverReport()
				if end, ok := any(value).(interface{ ReceiverRegionEnds() bool }); tc.errorOffset >= 0 {
					if !ok || !end.ReceiverRegionEnds() {
						t.Fatal("reserved receiver end did not propagate to the root")
					}
				} else if ok && end.ReceiverRegionEnds() {
					t.Fatal("ordinary value terminates a receiver region")
				}
				if tc.errorOffset >= 0 {
					if len(report.Regions) != 1 || string(report.Regions[0].Label) != "PDCH pairs configuration error" || report.Regions[0].Location.BitOffset != phase+tc.errorOffset || report.Regions[0].Location.Width != len(tc.bits)-tc.errorOffset {
						t.Fatalf("region %+v", report)
					}
				} else if len(report.Regions) != 0 {
					t.Fatalf("unexpected region %+v", report)
				}
				if err := r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				sentinel, err := r.ReadUint(3)
				if err != nil || sentinel != 5 {
					t.Fatalf("sentinel %d %v", sentinel, err)
				}
				w := runtime.NewWriter()
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				old, err = w.PushLimit(len(tc.bits))
				if err != nil {
					t.Fatal(err)
				}
				if err := EncodeSingleDownlinkAssignment2IETo(w, value); err != nil {
					t.Fatalf("phase %d replay: %v", phase, err)
				}
				if err := w.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteUint(5, 3); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != phase+len(tc.bits)+3 || !bytes.Equal(out.Bytes, input) {
					t.Fatalf("phase %d replay bit length %d: %v", phase, out.BitLength, err)
				}
				canonical := runtime.Canonical(value)
				w = runtime.NewWriter()
				w.WithWire(canonical.Wire)
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				_, err = w.PushLimit(len(tc.bits))
				if err != nil {
					t.Fatal(err)
				}
				err = EncodeSingleDownlinkAssignment2IETo(w, canonical)
				if tc.errorOffset >= 0 {
					var op *runtime.OperationError
					if !errors.As(err, &op) || op.Kind != runtime.InvalidValue {
						t.Fatalf("reserved sender: %v", err)
					}
					if _, err := EncodeSingleDownlinkAssignment2IECanonicalAtLength(value, (len(tc.bits)+7)/8); !errors.As(err, &op) {
						t.Fatalf("canonical reserved sender: %v", err)
					}
				} else if err != nil {
					t.Fatalf("canonical bit sender: %v", err)
				}
				if tc.errorOffset < 0 {
					if w.Position() != phase+len(tc.bits) {
						t.Fatalf("canonical width %d want %d", w.Position(), phase+len(tc.bits))
					}
					out, err := w.FinishBits(runtime.BitString{})
					if err != nil {
						t.Fatal(err)
					}
					cr, err := runtime.NewReaderBits(out.Bytes, out.BitLength, nil)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := cr.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					if _, err := cr.PushLimit(len(tc.bits)); err != nil {
						t.Fatal(err)
					}
					round, err := DecodeSingleDownlinkAssignment2IEFrom(cr)
					if err != nil || !runtime.SemanticallyEqual(canonical, round) {
						t.Fatalf("canonical typed semantics: %v", err)
					}
				}
				fresh := runtime.Canonical(value)
				assignmentClearWire(reflect.ValueOf(&fresh).Elem())
				fw := runtime.NewWriter()
				if err := fw.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				if _, err := fw.PushLimit(len(tc.bits)); err != nil {
					t.Fatal(err)
				}
				freshErr := EncodeSingleDownlinkAssignment2IETo(fw, fresh)
				if tc.errorOffset >= 0 {
					var op *runtime.OperationError
					if !errors.As(freshErr, &op) || op.Kind != runtime.InvalidValue {
						t.Fatalf("fresh reserved sender: %v", freshErr)
					}
				} else if freshErr != nil {
					t.Fatalf("fresh sender: %v", freshErr)
				}

			}
			decoded, err := d.DecodeBits(assignmentBits(tc.bits), len(tc.bits))
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.(runtime.Decoded[SingleDownlinkAssignment2IE]).Value
			w := runtime.NewWriter()
			if err := EncodeSingleDownlinkAssignment2IETo(w, v); err != nil {
				t.Fatal(err)
			}
			bits, err := w.FinishBits(runtime.BitString{})
			if err != nil || bits.BitLength != len(tc.bits) || !bytes.Equal(bits.Bytes, assignmentBits(tc.bits)) {
				t.Fatalf("bounded descriptor replay: %v", err)
			}
		})
	}
}
func TestClause12_48a_2Truncations(t *testing.T) {
	for _, tc := range assignmentVectors() {
		if tc.errorOffset >= 0 {
			continue
		}
		data := assignmentBits(tc.bits)
		for cut := 0; cut < len(tc.bits); cut++ {
			d, err := Lookup("Single Downlink Assignment 2 IE")
			if err != nil {
				t.Fatal(err)
			}
			boxed, err := d.DecodeBits(data, cut)
			if err != nil {
				continue
			}
			v := boxed.(runtime.Decoded[SingleDownlinkAssignment2IE]).Value
			w := runtime.NewWriter()
			if err := EncodeSingleDownlinkAssignment2IETo(w, v); err != nil {
				t.Fatalf("accepted prefix at %d does not replay: %v", cut, err)
			}
			out, err := w.FinishBits(runtime.BitString{})
			want := assignmentBits(tc.bits[:cut])
			if err != nil || out.BitLength != cut || !bytes.Equal(out.Bytes, want) {
				t.Fatalf("accepted prefix replay differs at bound %d: %v", cut, err)
			}
		}
	}
}
func TestClause12_48a_2GrowingRegion(t *testing.T) {
	for _, tc := range assignmentVectors() {
		if tc.errorOffset < 0 {
			continue
		}
		for _, n := range []int{256, 4096, 65536, 262144} {
			body := tc.bits[:tc.errorOffset] + strings.Repeat("1", n)
			data := assignmentBits(body)
			d, err := Lookup("Single Downlink Assignment 2 IE")
			if err != nil {
				t.Fatal(err)
			}
			var before, after goruntime.MemStats
			goruntime.ReadMemStats(&before)
			_, _, report, err := d.DecodeBitsWithReport(data, len(body))
			goruntime.ReadMemStats(&after)
			if err != nil || len(report.Regions) != 1 || report.Regions[0].Location.Width != n {
				t.Fatalf("region size %d: %+v %v", n, report, err)
			}
			bound := uint64(len(data))*64 + 128*1024
			if used := after.TotalAlloc - before.TotalAlloc; !raceEnabled && used > bound {
				t.Fatalf("region size %d allocated %d bound %d", n, used, bound)
			}
		}
		return
	}
	t.Fatal("no reserved region vector")
}
func FuzzClause12_48a_2(f *testing.F) {
	for _, tc := range assignmentVectors() {
		if len(tc.bits) < 128 {
			f.Add(assignmentBits(tc.bits), uint16(len(tc.bits)))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, length uint16) {
		if len(data) > 1024 {
			return
		}
		bits := int(length)
		if bits > len(data)*8 {
			bits = len(data) * 8
		}
		d, err := Lookup("Single Downlink Assignment 2 IE")
		if err != nil {
			t.Fatal(err)
		}
		plain, pe := d.DecodeBits(data, bits)
		reported, _, _, re := d.DecodeBitsWithReport(data, bits)
		if fmt.Sprint(pe) != fmt.Sprint(re) || !reflect.DeepEqual(plain, reported) {
			t.Fatalf("report parity %v / %v", pe, re)
		}
		if pe != nil {
			return
		}
		v := plain.(runtime.Decoded[SingleDownlinkAssignment2IE]).Value
		w := runtime.NewWriter()
		if err := EncodeSingleDownlinkAssignment2IETo(w, v); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		want := append([]byte(nil), data[:(bits+7)/8]...)
		if bits%8 != 0 {
			want[len(want)-1] &= 0xff << uint(8-bits%8)
		}
		if err != nil || out.BitLength != bits || !bytes.Equal(out.Bytes, want) {
			t.Fatalf("bounded replay length %d want %d: %v", out.BitLength, bits, err)
		}
	})
}

func TestClause12_48a_2Edits(t *testing.T) {
	for _, tc := range assignmentVectors() {
		if tc.errorOffset >= 0 {
			continue
		}
		r, err := runtime.NewReaderBits(assignmentBits(tc.bits), len(tc.bits), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.PushLimit(len(tc.bits)); err != nil {
			t.Fatal(err)
		}
		v, err := DecodeSingleDownlinkAssignment2IEFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		v = runtime.Canonical(v)
		v.LINKQUALITYMEASUREMENTMODE = 1
		w := runtime.NewWriter()
		if _, err := w.PushLimit(len(tc.bits)); err != nil {
			t.Fatal(err)
		}
		if err := EncodeSingleDownlinkAssignment2IETo(w, v); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		if out.BitLength != len(tc.bits) || bytes.Equal(out.Bytes, assignmentBits(tc.bits)) {
			t.Fatal("semantic edit did not change the transmitted field")
		}
		rd, err := runtime.NewReaderBits(out.Bytes, out.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := rd.PushLimit(out.BitLength); err != nil {
			t.Fatal(err)
		}
		changed, err := DecodeSingleDownlinkAssignment2IEFrom(rd)
		if err != nil {
			t.Fatal(err)
		}
		if !runtime.SemanticallyEqual(v, changed) {
			t.Fatal("edited typed semantics changed on decode")
		}
		return
	}
	t.Fatal("no ordinary vector")
}
