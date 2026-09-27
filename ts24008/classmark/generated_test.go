package classmark

import (
	"bytes"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"testing"
)

func TestDecodeRejectsOversizedInput(t *testing.T) {
	if _, err := DecodeA5Bits(make([]byte, 131073)); err == nil {
		t.Fatal("unbounded input accepted")
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
			A5Bits:             A5Bits{A56: 1}, Items: make([]runtime.BitString, 4), AssociatedRadioCapability1: 4,
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
			AssociatedRadioCapability1: &Classmark3ValuePartMultibandSupportedChoiceAssociatedRadioCapability1{MultibandSupported: band, Items: make([]runtime.BitString, 4)}}}
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
