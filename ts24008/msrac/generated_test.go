package msrac

import (
	"bytes"
	"reflect"
	"testing"
)

func TestConstructedAccessTechnologyList(t *testing.T) {
	entry := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
		Alternative: MSRACapabilityValuePartStructElementNode1Alternative2,
		Alt2:        &MSRACapabilityValuePartStructElementNode1Alt2{AccessTechnologyType: 15, Length: 1},
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

func TestAccessTechnologyConstraintsOnEncode(t *testing.T) {
	base := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
		Alternative: MSRACapabilityValuePartStructElementNode1Alternative2,
		Alt2:        &MSRACapabilityValuePartStructElementNode1Alt2{AccessTechnologyType: 14, Length: 1},
	}}
	v := MSRACapabilityValuePart{MSRACapabilityValuePartStruct: MSRACapabilityValuePartStruct{Entries: []MSRACapabilityValuePartStructElement{base}}}
	if _, err := EncodeMSRACapabilityValuePart(v); err == nil {
		t.Fatal("== 1111 constraint ignored")
	}
	base.Choice.Alternative = MSRACapabilityValuePartStructElementNode1Alternative1
	base.Choice.Alt2 = nil
	base.Choice.Alt1 = &MSRACapabilityValuePartStructElementNode1Alt1{AccessTechnologyType: 15}
	v.MSRACapabilityValuePartStruct.Entries[0] = base
	if _, err := EncodeMSRACapabilityValuePart(v); err == nil {
		t.Fatal("exclude 1111 constraint ignored")
	}
}

func TestMixedAccessTechnologiesAndAdditionalList(t *testing.T) {
	content := Content{RFPowerCapability: 4, Choice: &ContentNode2Alt2{A5Bits: A5Bits{A51: 1, A53: 1}}}
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
	first := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
		Alternative: MSRACapabilityValuePartStructElementNode1Alternative1,
		Alt1: &MSRACapabilityValuePartStructElementNode1Alt1{AccessTechnologyType: 1, AccessCapabilities: AccessCapabilitiesStruct{
			Length: uint8(decodedContent.BitsConsumed), Content: AccessCapabilitiesStructNode2Content{AccessCapabilities: content},
		}},
	}}
	second := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
		Alternative: MSRACapabilityValuePartStructElementNode1Alternative2,
		Alt2: &MSRACapabilityValuePartStructElementNode1Alt2{AccessTechnologyType: 15, Length: 21, Content: MSRACapabilityValuePartStructElementNode1Alt2Node3Content{
			Items: []MSRACapabilityValuePartStructElementNode1Alt2Node3ContentNode1Item{
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
	if len(got) != 2 || got[1].Choice.Alt2 == nil || len(got[1].Choice.Alt2.Content.Items) != 2 {
		t.Fatalf("access technology structure: %+v", got)
	}
	reencoded, err := EncodeMSRACapabilityValuePart(decoded.Value)
	if err != nil || !bytes.Equal(encoded, reencoded) {
		t.Fatalf("round trip: %v", err)
	}
}

func TestContentOptionalGroupsAcrossLengthBoundedVectors(t *testing.T) {
	groups := [][]string{{"Choice", "Choice2", "Choice3"}, {"Choice4", "Choice5", "Choice6"}, {"Choice7", "Choice8", "Choice9"}}
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
		entry := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
			Alternative: MSRACapabilityValuePartStructElementNode1Alternative1,
			Alt1: &MSRACapabilityValuePartStructElementNode1Alt1{AccessTechnologyType: 1, AccessCapabilities: AccessCapabilitiesStruct{
				Length: uint8(decoded.BitsConsumed), Content: AccessCapabilitiesStructNode2Content{AccessCapabilities: content},
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
		choice := value.FieldByName("Alt1")
		if choice.IsValid() && choice.Kind() == reflect.Pointer {
			alt.SetUint(0)
			fillContentPointers(choice)
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
	entry := MSRACapabilityValuePartStructElement{Choice: MSRACapabilityValuePartStructElementNode1{
		Alternative: MSRACapabilityValuePartStructElementNode1Alternative2,
		Alt2:        &MSRACapabilityValuePartStructElementNode1Alt2{AccessTechnologyType: 15, Length: 1},
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
