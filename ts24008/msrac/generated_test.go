package msrac

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestConstructedAccessTechnologyList(t *testing.T) {
	entry := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeLength,
		Length:      &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLength{AccessTechnologyType: 15, Length: 1},
	}}
	v := MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{
		Entries: []MSRACapabilityValuePartStructElement{entry, entry},
	}}
	encoded, err := EncodeMSRACapabilityValuePart(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("synthetic MS RA capability: %x", encoded)
	decoded, err := DecodeMSRACapabilityValuePart(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Value.MSRACapabilityValuePartStruct.Entries) != 2 {
		t.Fatalf("entries=%d", len(decoded.Value.MSRACapabilityValuePartStruct.Entries))
	}
	reencoded, err := EncodeMSRACapabilityValuePart(decoded.Value)
	if err != nil || !bytes.Equal(encoded, reencoded) {
		t.Fatalf("reencode=%x err=%v", reencoded, err)
	}
}

func TestFreshEncodingUsesZeroSpareBits(t *testing.T) {
	// TS 24.007 V20.0.0 Annex B.1.2.1 Rule B7 requires emitted
	// spare bits to be zero; TS 24.008 V20.1.0 §10.5.5.12a ends
	// the MS RA capability with repeated spare bits.
	want := []byte{0x19, 0x30, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	d, err := DecodeMSRACapabilityValuePart(want)
	if err != nil {
		t.Fatal(err)
	}
	d.Value.Wire = runtime.WireInfo{}
	got, err := EncodeMSRACapabilityValuePart(d.Value)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("fresh spare bits %x, %v; want %x", got, err, want)
	}
}

func TestTruncatedAccessTechnologyReportsTruncation(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a needs the first access
	// technology entry in full before a branch can be selected.
	_, err := DecodeMSRACapabilityValuePart([]byte{0x1a, 0x13})
	var de *runtime.DecodeError
	if !errors.As(err, &de) || de.Kind != runtime.Truncated || de.Offset < 8 {
		t.Fatalf("short MS RA capability = %v; want truncation at input end", err)
	}
}

func TestOversizedValueRetainsAllExcessAsTail(t *testing.T) {
	// TS 24.007 V20.0.0 §11.4.2 ignores excess type 4 value bits.
	// TS 24.008 V20.1.0 §10.5.5.12a bounds the known part at 50 octets.
	for _, extra := range []byte{0x2b, 0x00, 0x80} {
		wire := append([]byte{0x10, 0xb1, 0}, bytes.Repeat([]byte{0x2b}, 47)...)
		wire = append(wire, extra)
		d, err := DecodeMSRACapabilityValuePart(wire)
		if err != nil {
			t.Fatalf("excess %02x: %v", extra, err)
		}
		if d.BitsConsumed != 400 || d.Tail.BitLength != 8 || d.Tail.Bytes[0] != extra {
			t.Fatalf("excess %02x: consumed=%d tail=%+v", extra, d.BitsConsumed, d.Tail)
		}
		encoded, err := EncodeMSRACapabilityValuePart(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("excess %02x round trip %x: %v", extra, encoded, err)
		}
		canonical, err := EncodeMSRACapabilityValuePartCanonical(d.Value)
		if err != nil || len(canonical) > 50 {
			t.Fatalf("excess %02x: canonical length=%d err=%v", extra, len(canonical), err)
		}
		if _, err := EncodeMSRACapabilityValuePartCanonicalAtLength(d.Value, 51); err == nil {
			t.Fatal("canonical-at-length accepted an extra octet")
		}
	}
}

func TestLengthDelimitedStructCrossesValueMaximum(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a gives each access-technology
	// struct its own seven-bit length. The third complete struct crosses
	// the current 50-octet value maximum (TS 24.007 V20.0.0 §11.4.2).
	wire, err := hex.DecodeString("1fe000000000000000000000000000000023fc000000000000000000000000000000047f800000000000000000000000000000002b")
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeMSRACapabilityValuePart(wire)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(d.Value.MSRACapabilityValuePartStruct.Entries); got != 3 {
		t.Fatalf("entries=%d, want 3", got)
	}
	if d.BitsConsumed != 417 || d.Tail.BitLength != 7 {
		t.Fatalf("consumed=%d tail=%+v", d.BitsConsumed, d.Tail)
	}
	if !reflect.DeepEqual(d.Tail, d.Value.Wire.Tail) {
		t.Fatalf("wire tail=%+v, decoded tail=%+v", d.Value.Wire.Tail, d.Tail)
	}
	encoded, err := EncodeMSRACapabilityValuePart(d.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("round trip %x: %v", encoded, err)
	}
	if _, err := EncodeMSRACapabilityValuePartCanonical(d.Value); err == nil {
		t.Fatal("canonical encoding accepted three structs beyond the 50-octet maximum")
	} else {
		var extent *runtime.ExtentError
		if !errors.As(err, &extent) {
			t.Fatalf("canonical error %T: %v; want extent", err, err)
		}
	}
	for _, suffix := range []byte{0x00, 0x7f} {
		changed := append([]byte(nil), wire...)
		changed[len(changed)-1] = suffix // the high bit ends the third struct
		decoded, err := DecodeMSRACapabilityValuePart(changed)
		if err != nil || decoded.BitsConsumed != 417 || decoded.Tail.BitLength != 7 {
			t.Fatalf("suffix %02x: consumed=%d tail=%+v err=%v", suffix, decoded.BitsConsumed, decoded.Tail, err)
		}
		encoded, err := EncodeMSRACapabilityValuePart(decoded.Value)
		if err != nil || !bytes.Equal(encoded, changed) {
			t.Fatalf("suffix %02x: round trip %x: %v", suffix, encoded, err)
		}
	}
	// The same boundary also occurs with distinct access technologies,
	// satisfying TS 24.008 V20.1.0 §10.5.5.12a's band conditions.
	distinct, err := hex.DecodeString("1fe000000000000000000000000000000027fc000000000000000000000000000000057f800000000000000000000000000000002b")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := DecodeMSRACapabilityValuePart(distinct)
	if err != nil {
		t.Fatal(err)
	}
	entries := parsed.Value.MSRACapabilityValuePartStruct.Entries
	if len(entries) != 3 {
		t.Fatalf("distinct access technologies: %d entries", len(entries))
	}
	for i, want := range []uint8{1, 3, 5} {
		part := entries[i].AccessTechnologyTypeChoice.AccessCapabilities
		if part == nil || part.AccessTechnologyType != want || part.AccessCapabilities.Length != 127 {
			t.Fatalf("entry %d: %+v, want type %d length 127", i, part, want)
		}
	}
	encoded, err = EncodeMSRACapabilityValuePart(parsed.Value)
	if err != nil || !bytes.Equal(encoded, distinct) {
		t.Fatalf("distinct access technologies round trip %x: %v", encoded, err)
	}
}

func TestRepeatedSpareBitsStayInWireState(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a permits arbitrary future-use
	// spare bits after the access-technology chain.
	wire := []byte{0xf0, 0x2f, 0x81, 0x2a}
	decoded, err := DecodeMSRACapabilityValuePart(wire)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Value.Wire.SpareCounts) == 0 || len(decoded.Value.Wire.Spare) == 0 {
		t.Fatal("repeated spare bits missing from wire record")
	}
	encoded, err := EncodeMSRACapabilityValuePart(decoded.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, wire) {
		t.Fatalf("repeated spare bits changed: %x, want %x", encoded, wire)
	}
}

func TestAccessTechnologyConstraintsOnEncode(t *testing.T) {
	base := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeLength,
		Length:      &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLength{AccessTechnologyType: 14, Length: 1},
	}}
	v := MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{Entries: []MSRACapabilityValuePartStructElement{base}}}
	if _, err := EncodeMSRACapabilityValuePart(v); err == nil {
		t.Fatal("== 1111 constraint ignored")
	}
	base.AccessTechnologyTypeChoice.Alternative = MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeAccessCapabilities
	base.AccessTechnologyTypeChoice.Length = nil
	base.AccessTechnologyTypeChoice.AccessCapabilities = &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAccessCapabilities{AccessTechnologyType: 15}
	v.MSRACapabilityValuePartStruct.Entries[0] = base
	if _, err := EncodeMSRACapabilityValuePart(v); err == nil {
		t.Fatal("exclude 1111 constraint ignored")
	}
}

func TestMixedAccessTechnologiesAndAdditionalList(t *testing.T) {
	content := Content{RFPowerCapability: 4, A5Bits: &A5Bits{A51: 1, A53: 1}}
	encodedContent, err := EncodeContent(content)
	if err != nil {
		t.Fatal(err)
	}
	decodedContent, err := DecodeContent(encodedContent)
	if err != nil {
		t.Fatal(err)
	}
	if decodedContent.BitsConsumed > 127 {
		t.Fatal("access capabilities length exceeds 7 bits")
	}
	first := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeAccessCapabilities,
		AccessCapabilities: &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAccessCapabilities{AccessTechnologyType: 1, AccessCapabilities: AccessCapabilitiesStruct{
			Length: uint8(decodedContent.BitsConsumed), Content: AccessCapabilitiesStructAccessCapabilitiesAccessCapabilities{AccessCapabilities: content},
		}},
	}}
	second := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeLength,
		Length: &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLength{AccessTechnologyType: 15, Length: 21, Content: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLengthAdditionalAccessTechnologiesAdditionalAccessTechnologies{
			AdditionalAccessTechnologiesList: []MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLengthAdditionalAccessTechnologiesAdditionalAccessTechnologiesAdditionalAccessTechnologiesListEntry{
				{AdditionalAccessTechnologies: AdditionalAccessTechnologiesStruct{AccessTechnologyType: 3, GMSKPowerClass: 1, N8PSKPowerClass: 2}},
				{AdditionalAccessTechnologies: AdditionalAccessTechnologiesStruct{AccessTechnologyType: 7, GMSKPowerClass: 4, N8PSKPowerClass: 2}},
			},
		}},
	}}
	v := MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{Entries: []MSRACapabilityValuePartStructElement{first, second}}}
	encoded, err := EncodeMSRACapabilityValuePart(v)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("synthetic mixed access technologies: %x", encoded)
	decoded, err := DecodeMSRACapabilityValuePart(encoded)
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.Value.MSRACapabilityValuePartStruct.Entries
	if len(got) != 2 || got[1].AccessTechnologyTypeChoice.Length == nil || len(got[1].AccessTechnologyTypeChoice.Length.Content.AdditionalAccessTechnologiesList) != 2 {
		t.Fatalf("access technology structure: %+v", got)
	}
	reencoded, err := EncodeMSRACapabilityValuePart(decoded.Value)
	if err != nil || !bytes.Equal(encoded, reencoded) {
		t.Fatalf("round trip: %v", err)
	}
}

func TestContentOptionalGroupsAcrossLengthBoundedVectors(t *testing.T) {
	groups := [][]string{{"A5Bits", "MultislotCapability", "N8PSKPowerCapability"}, {"ExtendedDTMGPRSMultiSlotClassGroup", "HighMultislotCapability", "DTMGPRSHighMultiSlotClassGroup"}, {"MultislotCapabilityReductionForDownlinkDualCarrierGroup", "DLMCCapability", "MSSyncAccuracy"}}
	for i, selected := range groups {
		content := Content{RFPowerCapability: 4}
		for _, name := range selected {
			fillContentPointers(reflect.ValueOf(&content).Elem().FieldByName(name))
		}
		encoded, err := EncodeContent(content)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeContent(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.BitsConsumed > 127 {
			t.Fatalf("vector %d requires %d bits, exceeding 7-bit length", i, decoded.BitsConsumed)
		}
		for _, name := range selected {
			if reflect.ValueOf(decoded.Value).FieldByName(name).IsNil() {
				t.Fatalf("vector %d lost %s", i, name)
			}
		}
		entry := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
			Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeAccessCapabilities,
			AccessCapabilities: &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAccessCapabilities{AccessTechnologyType: 1, AccessCapabilities: AccessCapabilitiesStruct{
				Length: uint8(decoded.BitsConsumed), Content: AccessCapabilitiesStructAccessCapabilitiesAccessCapabilities{AccessCapabilities: content},
			}},
		}}
		value := MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{Entries: []MSRACapabilityValuePartStructElement{entry}}}
		wire, err := EncodeMSRACapabilityValuePart(value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeMSRACapabilityValuePart(wire); err != nil {
			t.Fatal(err)
		}
		t.Logf("Content optional group vector %d: %x (%d bits)", i, wire, decoded.BitsConsumed)
	}
}

func fillContentPointers(value reflect.Value) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		fillContentPointers(value.Elem())
		return
	}
	if value.Kind() != reflect.Struct {
		return
	}
	if alt := value.FieldByName("Alternative"); alt.IsValid() {
		for i := 1; i < value.NumField(); i++ {
			choice := value.Field(i)
			if choice.Kind() == reflect.Pointer {
				alt.SetUint(0)
				fillContentPointers(choice)
				break
			}
		}
		return
	}
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if value.Type().Field(i).Name == "Wire" {
			continue
		}
		if field.Kind() == reflect.Pointer || field.Kind() == reflect.Struct {
			fillContentPointers(field)
		}
	}
}

func FuzzMSRADefinitions(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{0xff, 0x2b})
	f.Add(append(append([]byte{0x10, 0xb1, 0}, bytes.Repeat([]byte{0x2b}, 47)...), 0))
	entry := MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeLength,
		Length:      &MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLength{AccessTechnologyType: 15, Length: 1},
	}}
	minimal, err := EncodeMSRACapabilityValuePart(MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{Entries: []MSRACapabilityValuePartStructElement{entry}}})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(minimal)
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, name := range Definitions() {
			d, err := Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := d.Decode(data)
			if err != nil {
				continue
			}
			value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			encoded, err := d.Encode(value)
			if err != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("%s round trip: %v", name, err)
			}
		}
	})
}
