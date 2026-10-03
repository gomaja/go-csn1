package classmark

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecodeRejectsOversizedInput(t *testing.T) {
	if _, err := DecodeA5Bits(make([]byte, 131073)); err == nil {
		t.Fatal("unbounded input accepted")
	}
}

func TestClassmark3ValuePartExcessIgnoredAndPreserved(t *testing.T) {
	// TS 24.007 V20.0.0 §11.4.2: excess type 4 value bits are ignored.
	// TS 24.008 V20.1.0 §10.5.1.7 gives this value part 32 octets.
	for _, extra := range []byte{0, 0x80, 0x2b} {
		wire := make([]byte, 33)
		wire[0] = 0x60
		wire[32] = extra
		d, err := DecodeClassmark3ValuePart(wire)
		if err != nil {
			t.Fatalf("excess %02x: %v", extra, err)
		}
		if d.BitsConsumed != 256 || d.Tail.BitLength != 8 || d.Tail.Bytes[0] != extra {
			t.Fatalf("excess %02x: consumed=%d tail=%+v", extra, d.BitsConsumed, d.Tail)
		}
		got, err := EncodeClassmark3ValuePart(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("excess %02x: re-encode=%x err=%v", extra, got, err)
		}
		canonical, err := EncodeClassmark3ValuePartCanonical(d.Value)
		if err != nil || len(canonical) > 32 {
			t.Fatalf("excess %02x: canonical length=%d err=%v", extra, len(canonical), err)
		}
		if _, err := EncodeClassmark3ValuePartCanonicalAtLength(d.Value, 33); err == nil {
			t.Fatal("canonical-at-length accepted an extra octet")
		}
	}
}

func TestClassmark3TerminalSpareBitsConsumed(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.1.7 ends the value part with
	// <spare bits>; they belong to the value, not an undecoded tail.
	for _, wire := range [][]byte{
		{0x60, 0, 0, 0, 0, 0, 0, 0, 0, 0x20},
		{0, 0, 0, 0, 0, 0, 0, 0, 0},
	} {
		d, err := DecodeClassmark3ValuePart(wire)
		if err != nil {
			t.Fatal(err)
		}
		if d.BitsConsumed != len(wire)*8 || d.Tail.BitLength != 0 {
			t.Fatalf("%x consumed %d with tail %d", wire, d.BitsConsumed, d.Tail.BitLength)
		}
		encoded, err := EncodeClassmark3ValuePart(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("round trip %x -> %x: %v", wire, encoded, err)
		}
	}
}

func TestConstructedClassmark3MultibandBranches(t *testing.T) {
	branches := []Classmark3ValuePartMultibandSupportedChoice{
		{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &Classmark3ValuePartMultibandSupportedChoiceA5Bits{A5Bits: A5Bits{A57: 1}}},
		{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeAssociatedRadioCapability2, AssociatedRadioCapability2: &Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2{
			MultibandSupported: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2MultibandSupported{Alternative: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2MultibandSupportedAlternativeAlt101, Alt101: &struct{}{}},
			A5Bits:             A5Bits{A54: 1}, AssociatedRadioCapability1: 2, AssociatedRadioCapability2: 3,
		}},
		{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeAssociatedRadioCapability1, AssociatedRadioCapability1: &Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1{
			MultibandSupported: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1MultibandSupported{Alternative: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1MultibandSupportedAlternativeAlt010, Alt010: &struct{}{}},
			A5Bits:             A5Bits{A56: 1}, AssociatedRadioCapability1: 4,
		}},
	}
	for i, choice := range branches {
		v := Classmark3ValuePart{MultibandSupportedChoice: choice}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatalf("branch %d encode: %v", i, err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatalf("branch %d decode: %v", i, err)
		}
		if decoded.Value.MultibandSupportedChoice.Alternative != choice.Alternative {
			t.Fatalf("branch %d selected %d", i, decoded.Value.MultibandSupportedChoice.Alternative)
		}
		if i == 2 && len(decoded.Value.Wire.Spare) != 10 {
			t.Fatalf("fixed and trailing spare bits missing from wire state: %d", len(decoded.Value.Wire.Spare))
		}
		reencoded, err := EncodeClassmark3ValuePart(decoded.Value)
		if err != nil || !bytes.Equal(encoded, reencoded) {
			t.Fatalf("branch %d round trip: %v", i, err)
		}
		if i == 0 {
			decoded.Value.MultibandSupportedChoice.A5Bits.A5Bits.A54 = 1
			edited, err := EncodeClassmark3ValuePart(decoded.Value)
			if err != nil || bytes.Equal(encoded, edited) {
				t.Fatalf("edited value was not encoded: %v", err)
			}
			changed, err := DecodeClassmark3ValuePart(edited)
			if err != nil || changed.Value.MultibandSupportedChoice.A5Bits.A5Bits.A54 != 1 {
				t.Fatalf("edited value lost: %v", err)
			}
		}
	}
}

func FuzzClassmarkDefinitions(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{0xff, 0x2b})
	f.Add(append(append([]byte{0x60}, bytes.Repeat([]byte{0}, 31)...), 0x80))
	minimal, err := EncodeClassmark3ValuePart(Classmark3ValuePart{MultibandSupportedChoice: Classmark3ValuePartMultibandSupportedChoice{
		Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &Classmark3ValuePartMultibandSupportedChoiceA5Bits{},
	}})
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

func TestClassmark3ImplicitZeroExtensionPreservesPrefix(t *testing.T) {
	v := Classmark3ValuePart{MultibandSupportedChoice: Classmark3ValuePartMultibandSupportedChoice{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &Classmark3ValuePartMultibandSupportedChoiceA5Bits{}}}
	complete, err := EncodeClassmark3ValuePart(v)
	if err != nil {
		t.Fatal(err)
	}
	prefix := complete[:1]
	decoded, err := DecodeClassmark3ValuePart(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.Wire.ImplicitZeros == 0 {
		t.Fatal("implicit zero extension not recorded")
	}
	out, err := EncodeClassmark3ValuePart(decoded.Value)
	if err != nil || !bytes.Equal(out, prefix) {
		t.Fatalf("prefix changed: %v", err)
	}
}

func TestClassmark3EditedTransmittedBitKeepsOneOctetBoundary(t *testing.T) {
	decoded, err := DecodeClassmark3ValuePart([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.MultibandSupportedChoice.A5Bits.A5Bits.A57 = 1
	encoded, err := EncodeClassmark3ValuePart(decoded.Value)
	if err != nil || len(encoded) != 1 || encoded[0] != 0x08 {
		t.Fatalf("transmitted A5 edit: %x, %v", encoded, err)
	}
	decoded.Value.UCS2Treatment = 1
	if _, err := EncodeClassmark3ValuePart(decoded.Value); err == nil {
		t.Fatal("accepted edit to receiver-inferred bit")
	}
}

func TestClassmark3AllOptionalGroupsPresent(t *testing.T) {
	v := Classmark3ValuePart{MultibandSupportedChoice: Classmark3ValuePartMultibandSupportedChoice{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &Classmark3ValuePartMultibandSupportedChoiceA5Bits{}}}
	setAllOptionalPointers(reflect.ValueOf(&v))
	encoded, err := EncodeClassmark3ValuePart(v)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeClassmark3ValuePart(encoded)
	if err != nil {
		t.Fatal(err)
	}
	value := reflect.ValueOf(decoded.Value)
	optional := 0
	for i := 0; i < value.NumField(); i++ {
		if value.Field(i).Kind() == reflect.Pointer {
			optional++
			if value.Field(i).IsNil() {
				t.Fatalf("optional group %s absent", value.Type().Field(i).Name)
			}
		}
	}
	if optional < 18 {
		t.Fatalf("only %d optional groups checked", optional)
	}
	t.Logf("all-optional Classmark 3 vector: %x", encoded)
}

func TestClassmark3BandChoiceAlternatives(t *testing.T) {
	for gsm := 0; gsm < 3; gsm++ {
		for tgsm := 0; tgsm < 3; tgsm++ {
			g := Classmark3ValuePartGSM400BandsSupportedGroupGSM400BandsSupported{Alternative: Classmark3ValuePartGSM400BandsSupportedGroupGSM400BandsSupportedAlternative(gsm)}
			switch gsm {
			case 0:
				g.Alt01 = &struct{}{}
			case 1:
				g.Alt10 = &struct{}{}
			case 2:
				g.Alt11 = &struct{}{}
			}
			tg := Classmark3ValuePartTGSM400BandsSupportedGroupTGSM400BandsSupported{Alternative: Classmark3ValuePartTGSM400BandsSupportedGroupTGSM400BandsSupportedAlternative(tgsm)}
			switch tgsm {
			case 0:
				tg.Alt01 = &struct{}{}
			case 1:
				tg.Alt10 = &struct{}{}
			case 2:
				tg.Alt11 = &struct{}{}
			}
			v := Classmark3ValuePart{
				MultibandSupportedChoice:   Classmark3ValuePartMultibandSupportedChoice{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &Classmark3ValuePartMultibandSupportedChoiceA5Bits{}},
				GSM400BandsSupportedGroup:  &Classmark3ValuePartGSM400BandsSupportedGroup{GSM400BandsSupported: g},
				TGSM400BandsSupportedGroup: &Classmark3ValuePartTGSM400BandsSupportedGroup{TGSM400BandsSupported: tg},
			}
			encoded, err := EncodeClassmark3ValuePart(v)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeClassmark3ValuePart(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Value.GSM400BandsSupportedGroup == nil || decoded.Value.TGSM400BandsSupportedGroup == nil || decoded.Value.GSM400BandsSupportedGroup.GSM400BandsSupported.Alternative != g.Alternative || decoded.Value.TGSM400BandsSupportedGroup.TGSM400BandsSupported.Alternative != tg.Alternative {
				t.Fatalf("band choices %d/%d lost", gsm, tgsm)
			}
			t.Logf("band choice %d/%d: %x", gsm, tgsm, encoded)
		}
	}
}

func TestClassmark3MultibandSubchoices(t *testing.T) {
	for i := 0; i < 2; i++ {
		band := Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2MultibandSupported{Alternative: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2MultibandSupportedAlternative(i)}
		if i == 0 {
			band.Alt101 = &struct{}{}
		} else {
			band.Alt110 = &struct{}{}
		}
		v := Classmark3ValuePart{MultibandSupportedChoice: Classmark3ValuePartMultibandSupportedChoice{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeAssociatedRadioCapability2,
			AssociatedRadioCapability2: &Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability2{MultibandSupported: band}}}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Value.MultibandSupportedChoice.AssociatedRadioCapability2 == nil || decoded.Value.MultibandSupportedChoice.AssociatedRadioCapability2.MultibandSupported.Alternative != band.Alternative {
			t.Fatalf("multiband alt2/%d lost", i)
		}
		t.Logf("multiband alt2/%d: %x", i, encoded)
	}
	for i := 0; i < 3; i++ {
		band := Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1MultibandSupported{Alternative: Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1MultibandSupportedAlternative(i)}
		switch i {
		case 0:
			band.Alt001 = &struct{}{}
		case 1:
			band.Alt010 = &struct{}{}
		case 2:
			band.Alt100 = &struct{}{}
		}
		v := Classmark3ValuePart{MultibandSupportedChoice: Classmark3ValuePartMultibandSupportedChoice{Alternative: Classmark3ValuePartMultibandSupportedChoiceAlternativeAssociatedRadioCapability1,
			AssociatedRadioCapability1: &Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1{MultibandSupported: band}}}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Value.MultibandSupportedChoice.AssociatedRadioCapability1 == nil || decoded.Value.MultibandSupportedChoice.AssociatedRadioCapability1.MultibandSupported.Alternative != band.Alternative {
			t.Fatalf("multiband alt3/%d lost", i)
		}
		t.Logf("multiband alt3/%d: %x", i, encoded)
	}
}

func setAllOptionalPointers(value reflect.Value) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		setAllOptionalPointers(value.Elem())
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
				setAllOptionalPointers(choice)
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
			setAllOptionalPointers(field)
		}
	}
}
