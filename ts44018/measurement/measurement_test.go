package measurement

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestEnhancedMeasurementReportWireTailEdit(t *testing.T) {
	input := []byte{0x10, 0x01}
	decoded, err := DecodeEnhancedMeasurementReport(input)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.Wire.Tail = runtime.BitString{Bytes: []byte{0xaa}, BitLength: 8}
	if _, err := EncodeEnhancedMeasurementReport(decoded.Value); err == nil {
		t.Fatal("accepted an invented EMR tail")
	}
}

func TestEnhancedMeasurementReportVectors(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55 and §10.4 table 10.4.2:
	// Enhanced Measurement Report has uplink message type 00100.
	// The absent bitmap and Rel-8 bitmap branches were independently
	// encoded and decoded with pycrate 0.7.11.
	for _, tc := range []struct {
		name   string
		wire   []byte
		bitmap bool
		utran  bool
		si23   bool
	}{
		{"prior release", []byte{0x10, 0x01}, false, false, false},
		{"prior release ignored bits", []byte{0x10, 0x01, 0xff}, false, false, false},
		{"Rel-8 bitmap", []byte{0x10, 0x00, 0x00, 0x00}, true, false, false},
		{"Rel-9 absent UTRAN", []byte{0x10, 0x00, 0x00, 0x40}, true, false, false},
		{"Rel-9 UTRAN", []byte{0x10, 0x00, 0x00, 0x60, 0, 0, 0, 0, 0, 0, 0, 0x20}, true, true, false},
		{"Rel-11 SI23", []byte{0x10, 0x00, 0x00, 0x58}, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := DecodeEnhancedMeasurementReport(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.BitsConsumed != len(tc.wire)*8 || decoded.Value.MessageType != 4 || (decoded.Value.BITMAPLENGTHChoice.BITMAPLENGTH != nil) != tc.bitmap {
				t.Fatalf("structure or consumed bits: %+v", decoded)
			}
			if tc.bitmap && len(decoded.Value.BITMAPLENGTHChoice.BITMAPLENGTH.REPORTINGQUANTITYList) != 1 {
				t.Fatalf("bitmap entries: %+v", decoded.Value.BITMAPLENGTHChoice.BITMAPLENGTH)
			}
			if tc.utran || tc.si23 {
				branch := decoded.Value.BITMAPLENGTHChoice.BITMAPLENGTH.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
				if branch == nil || (branch.UTRANCSGMeasurementReport != nil) != tc.utran || (branch.SI23BAUSEDChoice.SI23BAUSED != nil) != tc.si23 {
					t.Fatalf("release fields: %+v", branch)
				}
			}
			encoded, err := EncodeEnhancedMeasurementReport(decoded.Value)
			if err != nil || !bytes.Equal(encoded, tc.wire) {
				t.Fatalf("round trip %x -> %x: %v", tc.wire, encoded, err)
			}
		})
	}
}

func TestEditedEnhancedMeasurementReportReencodesFields(t *testing.T) {
	if _, err := DecodeEnhancedMeasurementReport([]byte{0x14, 0x01}); err == nil {
		t.Fatal("accepted Measurement Information message type as EMR")
	}
	decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0x01, 0xff})
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.MessageType = 5
	if _, err := EncodeEnhancedMeasurementReport(decoded.Value); err == nil {
		t.Fatal("encoded Measurement Information message type as EMR")
	}
	decoded.Value.MessageType = 4
	decoded.Value.BAUSED = 1
	for _, clearWire := range []bool{false, true} {
		value := decoded.Value
		if clearWire {
			value.Wire = runtime.WireInfo{}
		}
		encoded, err := EncodeEnhancedMeasurementReport(value)
		want := []byte{0x10, 0x81, 0xff}
		if clearWire {
			want = []byte{0x10, 0x81}
		}
		if err != nil || !bytes.Equal(encoded, want) {
			t.Fatalf("edited BA_USED: %x, %v", encoded, err)
		}
	}
}

func FuzzMeasurementDefinitions(f *testing.F) {
	for _, seed := range [][]byte{{}, {0}, {0x10, 0x01}, {0x10, 0, 0, 0}, {0xff}, bytes.Repeat([]byte{0x2b}, 23)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, definition := range Descriptors() {
			decoded, err := definition.Decode(data)
			if err != nil {
				continue
			}
			value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			encoded, err := definition.Encode(value)
			if err != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("%s §%s: %x -> %x: %v", definition.Name, definition.Clause, data, encoded, err)
			}
		}
	})
}
