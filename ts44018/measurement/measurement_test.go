package measurement

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestEnhancedMeasurementReportPreRel8Bitmap(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55: redundant no-report positions
	// remain typed bitmap positions, independent of the neighbour list size.
	for _, count := range []int{0, 1, 2, 96} {
		entries := make([]*uint8, count)
		value := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries,
			BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
		wire, err := EncodeEnhancedMeasurementReport(value)
		if err != nil {
			t.Fatalf("count %d encode: %v", count, err)
		}
		decoded, err := DecodeEnhancedMeasurementReport(wire)
		if err != nil {
			t.Fatalf("count %d decode %x: %v", count, wire, err)
		}
		if decoded.Value.REPORTINGQUANTITYList == nil || len(*decoded.Value.REPORTINGQUANTITYList) != 96 || decoded.BitsConsumed != len(wire)*8 || decoded.Tail.BitLength != 0 {
			t.Fatalf("count %d boundary: %+v", count, decoded)
		}
		again, err := EncodeEnhancedMeasurementReport(decoded.Value)
		if err != nil || !bytes.Equal(again, wire) {
			t.Fatalf("count %d round trip %x -> %x: %v", count, wire, again, err)
		}
		if count == 2 && !bytes.Equal(wire, []byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}) {
			t.Fatalf("issue reproducer changed: %x", wire)
		}
	}
}

func TestEnhancedMeasurementReportCanonicalAfterBitmapEdit(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55: fresh pre-Rel-8 values fill all
	// 96 bitmap positions with no-report bits, even when only two are set
	// explicitly by the caller.
	entries := make([]*uint8, 2)
	fresh := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries,
		BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
	want, err := EncodeEnhancedMeasurementReport(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, []byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Fatalf("wrong canonical bitmap: %x", want)
	}
	decoded, err := DecodeEnhancedMeasurementReport(want)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.REPORTINGQUANTITYList == nil || len(*decoded.Value.REPORTINGQUANTITYList) != 96 {
		t.Fatalf("decoded bitmap is not structurally complete: %+v", decoded.Value)
	}
	for i, report := range *decoded.Value.REPORTINGQUANTITYList {
		if report != nil {
			t.Fatalf("position %d is not no-report", i)
		}
	}
	edited := decoded.Value
	two := (*edited.REPORTINGQUANTITYList)[:2]
	edited.REPORTINGQUANTITYList = &two
	if _, err := EncodeEnhancedMeasurementReport(edited); err == nil || !strings.Contains(err.Error(), "encoded semantic boundary") {
		t.Fatalf("plain re-encode did not reject changed semantic boundary: %v", err)
	}
	canonical, err := EncodeEnhancedMeasurementReportCanonical(edited)
	if err != nil || !bytes.Equal(canonical, want) {
		t.Fatalf("canonical re-encode %x: %v", canonical, err)
	}
	if edited.Wire.BitsConsumed != len(want)*8 || len(*edited.REPORTINGQUANTITYList) != 2 {
		t.Fatal("canonical encoding modified the caller's value")
	}
	canonicalDecoded, err := DecodeEnhancedMeasurementReport(canonical)
	if err != nil || !reflect.DeepEqual(canonicalDecoded.Value, decoded.Value) {
		t.Fatalf("canonical result differs from expanded fresh value: %v", err)
	}
}

func TestEnhancedMeasurementReportBterCarrierLimit(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55 sends this message on SACCH.
	// TS 44.006 V19.0.0 §§5.2, 8.8.3 count the short L2 header
	// inside the 21-octet Bter information field.
	wire := []byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	d, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		present int
		wantLen int
	}{
		{9, 21},
		{10, 22},
	} {
		v := d.Value
		list := append([]*uint8(nil), (*v.REPORTINGQUANTITYList)...)
		for i := 0; i < tc.present; i++ {
			zero := uint8(0)
			list[i] = &zero
		}
		v.REPORTINGQUANTITYList = &list
		out, err := EncodeEnhancedMeasurementReportCanonical(v)
		if tc.wantLen == 21 {
			if err != nil || len(out) != 21 {
				t.Fatalf("%d present: len=%d, error=%v", tc.present, len(out), err)
			}
			continue
		}
		var extent *runtime.ExtentError
		if !errors.As(err, &extent) || extent.Maximum != 21 {
			t.Fatalf("%d present: wanted typed 21-octet extent error, got %d octets, %v", tc.present, len(out), err)
		}
		if _, err := EncodeEnhancedMeasurementReportCanonicalAtLength(v, 22); !errors.As(err, &extent) {
			t.Fatalf("explicit 22-octet extent accepted: %v", err)
		}
		if _, err := EncodeEnhancedMeasurementReport(runtime.Canonical(v)); !errors.As(err, &extent) {
			t.Fatalf("fresh overlong EMR accepted: %v", err)
		}
	}
	var limit *runtime.DecodeError
	if _, err := DecodeEnhancedMeasurementReport(make([]byte, 22)); !errors.As(err, &limit) || limit.Kind != runtime.Limit {
		t.Fatalf("accepted an overlong Bter EMR: %v", err)
	}
	// The canonical bit target is octet-aligned. An extra bit after the
	// 168-bit Bter boundary cannot be represented in this carrier.
	v := d.Value
	list := append([]*uint8(nil), (*v.REPORTINGQUANTITYList)...)
	for i := 0; i < 9; i++ {
		zero := uint8(0)
		list[i] = &zero
	}
	v.REPORTINGQUANTITYList = &list
	if _, err := EncodeEnhancedMeasurementReport(runtime.CanonicalTo(v, 169)); err == nil {
		t.Fatal("accepted a canonical target one bit beyond the Bter boundary")
	}
	for i := range list {
		zero := uint8(0)
		list[i] = &zero
	}
	v.REPORTINGQUANTITYList = &list
	var extent *runtime.ExtentError
	if _, err := EncodeEnhancedMeasurementReportCanonical(v); !errors.As(err, &extent) || extent.Maximum != 21 {
		t.Fatalf("96 reported positions escaped the carrier limit: %v", err)
	}
}

func TestEnhancedMeasurementReportPreRel8WireBoundary(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55 permits omitted trailing no-report
	// positions only when the message cannot hold them.
	for _, tc := range []struct {
		wire []byte
		want int
	}{
		{[]byte{0x10, 0x02}, 1},
		{[]byte{0x10, 0x02, 0}, 9},
	} {
		d, err := DecodeEnhancedMeasurementReport(tc.wire)
		if err != nil || d.Value.REPORTINGQUANTITYList == nil || len(*d.Value.REPORTINGQUANTITYList) != tc.want {
			t.Fatalf("%x: got %+v, %v", tc.wire, d, err)
		}
		reencoded, err := EncodeEnhancedMeasurementReport(d.Value)
		if err != nil || !bytes.Equal(reencoded, tc.wire) {
			t.Fatalf("%x -> %x: %v", tc.wire, reencoded, err)
		}
	}
	for _, wire := range [][]byte{{0x10, 0x03}, {0x10, 0x02, 0x2b}} {
		_, err := DecodeEnhancedMeasurementReport(wire)
		var decoded *runtime.DecodeError
		if !errors.As(err, &decoded) || decoded.Kind != runtime.Truncated || decoded.Offset <= 15 {
			t.Fatalf("truncated report %x: %v", wire, err)
		}
	}
	entries := make([]*uint8, 97)
	value := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries}
	if _, err := EncodeEnhancedMeasurementReport(value); err == nil {
		t.Fatal("encoded more than 96 pre-Rel-8 bitmap positions")
	}
}

func FuzzEnhancedMeasurementReportBitmap(f *testing.F) {
	f.Add([]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	f.Add([]byte{0x10, 0, 0, 0})
	f.Fuzz(func(t *testing.T, wire []byte) {
		// TS 44.006 V19.0.0 §8.8.3: Bter SACCH N201 is 21 octets.
		if len(wire) > 21 {
			return
		}
		decoded, err := DecodeEnhancedMeasurementReport(wire)
		if err != nil {
			return
		}
		encoded, err := EncodeEnhancedMeasurementReport(decoded.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("%x -> %x: %v", wire, encoded, err)
		}
	})
}

func FuzzEnhancedMeasurementReportCanonical(f *testing.F) {
	f.Add([]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	f.Add([]byte{0x10, 0x90, 0x02, 0xc2})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 21 {
			return
		}
		d, err := DecodeEnhancedMeasurementReport(wire)
		if err != nil || d.Value.REPORTINGQUANTITYList == nil {
			return
		}
		original := d.Value.REPORTINGQUANTITYList
		entries := *original
		if len(entries) > 2 {
			for _, report := range entries[2:] {
				if report != nil {
					return
				}
			}
			entries = entries[:2]
		}
		d.Value.REPORTINGQUANTITYList = &entries
		canonical, err := EncodeEnhancedMeasurementReportCanonical(d.Value)
		if err != nil {
			t.Fatalf("canonical encode %x: %v", wire, err)
		}
		fresh, err := DecodeEnhancedMeasurementReport(canonical)
		if err != nil || !equivalentCanonicalEMR(d.Value, fresh.Value) {
			t.Fatalf("canonical decode %x: %v", canonical, err)
		}
		if len(*original) < len(entries) || len(*d.Value.REPORTINGQUANTITYList) != len(entries) {
			t.Fatal("canonical encoder modified caller")
		}
	})
}

func equivalentCanonicalEMR(a, b EnhancedMeasurementReport) bool {
	// TS 44.018 V19.0.0 §9.1.55: omitted redundant trailing
	// no-report entries are equivalent to explicit zero entries.
	normalize := func(v EnhancedMeasurementReport) EnhancedMeasurementReport {
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
	return runtime.SemanticallyEqual(normalize(a), normalize(b))
}

func TestCanonicalEnhancedMeasurementReportAtBitmapBoundary(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55 permits trailing no-report positions
	// to be omitted when they do not fit. The absent following choice is
	// part of the decoded value and must remain absent after canonicalization.
	for _, wire := range [][]byte{{0x10, 0x02}, {0x10, 0x02, 0x00}} {
		decoded, err := DecodeEnhancedMeasurementReport(wire)
		if err != nil {
			t.Fatalf("decode %x: %v", wire, err)
		}
		canonical, err := EncodeEnhancedMeasurementReportCanonical(decoded.Value)
		if err != nil {
			t.Fatalf("canonical encode %x: %v", wire, err)
		}
		again, err := DecodeEnhancedMeasurementReport(canonical)
		if err != nil {
			t.Fatalf("canonical decode %x: %v", canonical, err)
		}
		if !equivalentCanonicalEMR(decoded.Value, again.Value) {
			t.Fatalf("canonical %x changes typed value of %x", canonical, wire)
		}
		framed, err := EncodeEnhancedMeasurementReportCanonicalAtLength(decoded.Value, len(wire))
		if err != nil || len(framed) != len(wire) {
			t.Fatalf("framed canonical %x: %v", framed, err)
		}
		if _, err := EncodeEnhancedMeasurementReportCanonicalAtLength(decoded.Value, 1); err == nil {
			t.Fatal("framing too short for the typed value was accepted")
		}
	}
}

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

func TestEUTRANMeasurementReportHyphenatedCount(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55 prints val(N_E-UTRAN+1).
	// The complete field name is N_E-UTRAN, then the arithmetic adds one.
	wire := []byte{0x04, 0x96, 0xed}
	d, err := DecodeEUTRANMeasurementReportStruct(wire)
	if err != nil {
		t.Fatal(err)
	}
	if d.Value.NEUTRAN != 0 || len(d.Value.EUTRANFREQUENCYINDEXGroupList) != 1 {
		t.Fatalf("wrong E-UTRAN count: %+v", d.Value)
	}
	encoded, err := EncodeEUTRANMeasurementReportStruct(d.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("E-UTRAN round trip %x -> %x: %v", wire, encoded, err)
	}
}

func TestEnhancedMeasurementReportEUTRANSynthetic(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55. Wireshark 4.6.8 also dissects
	// this synthetic E-UTRAN report inside an Enhanced Measurement Report.
	wire := append([]byte{0x10, 0x90, 0x02, 0xc3, 0x54, 0xb6, 0xda, 0x11, 0x53}, bytes.Repeat([]byte{0x2b}, 12)...)
	d, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil {
		t.Fatal(err)
	}
	if d.BitsConsumed != len(wire)*8 || d.Tail.BitLength != 0 {
		t.Fatalf("report boundary %d, tail %d", d.BitsConsumed, d.Tail.BitLength)
	}
	encoded, err := EncodeEnhancedMeasurementReport(d.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("report round trip %x -> %x: %v", wire, encoded, err)
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
