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
	branches := []Classmark3ValuePartNode2{
		{Alternative: Classmark3ValuePartNode2Alternative1, Alt1: &Classmark3ValuePartNode2Alt1{A5Bits: A5Bits{A57: 1}}},
		{Alternative: Classmark3ValuePartNode2Alternative2, Alt2: &Classmark3ValuePartNode2Alt2{
			MultibandSupported: Classmark3ValuePartNode2Alt2Node1{Alternative: Classmark3ValuePartNode2Alt2Node1Alternative1, Alt1: &struct{}{}},
			A5Bits:             A5Bits{A54: 1}, AssociatedRadioCapability1: 2, AssociatedRadioCapability2: 3,
		}},
		{Alternative: Classmark3ValuePartNode2Alternative3, Alt3: &Classmark3ValuePartNode2Alt3{
			MultibandSupported: Classmark3ValuePartNode2Alt3Node1{Alternative: Classmark3ValuePartNode2Alt3Node1Alternative2, Alt2: &struct{}{}},
			A5Bits:             A5Bits{A56: 1}, Items: make([]runtime.BitString, 4), AssociatedRadioCapability1: 4,
		}},
	}
	for i, choice := range branches {
		v := Classmark3ValuePart{Choice: choice}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatalf("branch %d encode: %v", i, err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatalf("branch %d decode: %v", i, err)
		}
		if decoded.Value.Choice.Alternative != choice.Alternative {
			t.Fatalf("branch %d selected %d", i, decoded.Value.Choice.Alternative)
		}
		reencoded, err := EncodeClassmark3ValuePart(decoded.Value)
		if err != nil || !bytes.Equal(encoded, reencoded) {
			t.Fatalf("branch %d round trip: %v", i, err)
		}
		if i == 0 {
			decoded.Value.Choice.Alt1.A5Bits.A54 = 1
			edited, err := EncodeClassmark3ValuePart(decoded.Value)
			if err != nil || bytes.Equal(encoded, edited) {
				t.Fatalf("edited value was not encoded: %v", err)
			}
			changed, err := DecodeClassmark3ValuePart(edited)
			if err != nil || changed.Value.Choice.Alt1.A5Bits.A54 != 1 {
				t.Fatalf("edited value lost: %v", err)
			}
		}
	}
}

func FuzzClassmarkDefinitions(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{0xff, 0x2b})
	minimal, err := EncodeClassmark3ValuePart(Classmark3ValuePart{Choice: Classmark3ValuePartNode2{
		Alternative: Classmark3ValuePartNode2Alternative1, Alt1: &Classmark3ValuePartNode2Alt1{},
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
	v := Classmark3ValuePart{Choice: Classmark3ValuePartNode2{Alternative: Classmark3ValuePartNode2Alternative1, Alt1: &Classmark3ValuePartNode2Alt1{}}}
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
	v := Classmark3ValuePart{Choice: Classmark3ValuePartNode2{Alternative: Classmark3ValuePartNode2Alternative1, Alt1: &Classmark3ValuePartNode2Alt1{}}}
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
			g := Classmark3ValuePartNode11Alt2Node2{Alternative: Classmark3ValuePartNode11Alt2Node2Alternative(gsm)}
			switch gsm {
			case 0:
				g.Alt1 = &struct{}{}
			case 1:
				g.Alt2 = &struct{}{}
			case 2:
				g.Alt3 = &struct{}{}
			}
			tg := Classmark3ValuePartNode28Alt2Node2{Alternative: Classmark3ValuePartNode28Alt2Node2Alternative(tgsm)}
			switch tgsm {
			case 0:
				tg.Alt1 = &struct{}{}
			case 1:
				tg.Alt2 = &struct{}{}
			case 2:
				tg.Alt3 = &struct{}{}
			}
			v := Classmark3ValuePart{
				Choice:   Classmark3ValuePartNode2{Alternative: Classmark3ValuePartNode2Alternative1, Alt1: &Classmark3ValuePartNode2Alt1{}},
				Choice8:  &Classmark3ValuePartNode11Alt2{GSM400BandsSupported: g},
				Choice16: &Classmark3ValuePartNode28Alt2{TGSM400BandsSupported: tg},
			}
			encoded, err := EncodeClassmark3ValuePart(v)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeClassmark3ValuePart(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Value.Choice8 == nil || decoded.Value.Choice16 == nil || decoded.Value.Choice8.GSM400BandsSupported.Alternative != g.Alternative || decoded.Value.Choice16.TGSM400BandsSupported.Alternative != tg.Alternative {
				t.Fatalf("band choices %d/%d lost", gsm, tgsm)
			}
			t.Logf("band choice %d/%d: %x", gsm, tgsm, encoded)
		}
	}
}

func TestClassmark3MultibandSubchoices(t *testing.T) {
	for i := 0; i < 2; i++ {
		band := Classmark3ValuePartNode2Alt2Node1{Alternative: Classmark3ValuePartNode2Alt2Node1Alternative(i)}
		if i == 0 {
			band.Alt1 = &struct{}{}
		} else {
			band.Alt2 = &struct{}{}
		}
		v := Classmark3ValuePart{Choice: Classmark3ValuePartNode2{Alternative: Classmark3ValuePartNode2Alternative2,
			Alt2: &Classmark3ValuePartNode2Alt2{MultibandSupported: band}}}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Value.Choice.Alt2 == nil || decoded.Value.Choice.Alt2.MultibandSupported.Alternative != band.Alternative {
			t.Fatalf("multiband alt2/%d lost", i)
		}
		t.Logf("multiband alt2/%d: %x", i, encoded)
	}
	for i := 0; i < 3; i++ {
		band := Classmark3ValuePartNode2Alt3Node1{Alternative: Classmark3ValuePartNode2Alt3Node1Alternative(i)}
		switch i {
		case 0:
			band.Alt1 = &struct{}{}
		case 1:
			band.Alt2 = &struct{}{}
		case 2:
			band.Alt3 = &struct{}{}
		}
		v := Classmark3ValuePart{Choice: Classmark3ValuePartNode2{Alternative: Classmark3ValuePartNode2Alternative3,
			Alt3: &Classmark3ValuePartNode2Alt3{MultibandSupported: band, Items: make([]runtime.BitString, 4)}}}
		encoded, err := EncodeClassmark3ValuePart(v)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeClassmark3ValuePart(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Value.Choice.Alt3 == nil || decoded.Value.Choice.Alt3.MultibandSupported.Alternative != band.Alternative {
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
		choice := value.FieldByName("Alt1")
		if choice.IsValid() && choice.Kind() == reflect.Pointer {
			alt.SetUint(0)
			setAllOptionalPointers(choice)
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
