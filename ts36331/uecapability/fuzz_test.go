package uecapability

import (
	"bytes"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"testing"
)

func FuzzGERANContainers(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x33, 3, 0, 0, 0, 0})
	f.Add([]byte{0xff, 0x2b})
	cm3, err := classmark.EncodeClassmark3ValuePart(classmark.Classmark3ValuePart{MultibandSupportedChoice: classmark.Classmark3ValuePartMultibandSupportedChoice{
		Alternative: classmark.Classmark3ValuePartMultibandSupportedChoiceAlternativeA5Bits, A5Bits: &classmark.Classmark3ValuePartMultibandSupportedChoiceA5Bits{},
	}})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(append([]byte{0x33, 3, 0, 0, 0}, cm3...))
	entry := msrac.MSRACapabilityValuePartStructElement{AccessTechnologyTypeChoice: msrac.MSRACapabilityValuePartStructElementAccessTechnologyTypeChoice{
		Alternative: msrac.MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceAlternativeLength,
		Length:      &msrac.MSRACapabilityValuePartStructElementAccessTechnologyTypeChoiceLength{AccessTechnologyType: 15, Length: 1},
	}}
	ps, err := msrac.EncodeMSRACapabilityValuePart(msrac.MSRACapabilityValuePart{MSRACapabilityValuePartStruct: msrac.MSRACapabilityValuePartStruct{Entries: []msrac.MSRACapabilityValuePartStructElement{entry}}})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(ps)
	f.Fuzz(func(t *testing.T, data []byte) {
		if d, err := DecodeClassmark2ValuePart(data); err == nil {
			out, err := EncodeClassmark2ValuePart(d.Value)
			if err != nil || !bytes.Equal(out, data) {
				t.Fatalf("classmark 2 round trip: %v", err)
			}
		}
		if d, err := DecodeGERANCS(data); err == nil {
			out, err := EncodeGERANCS(d.Value)
			if err != nil || !bytes.Equal(out, data) {
				t.Fatalf("GERAN CS round trip: %v", err)
			}
		}
		if d, err := DecodeGERANPS(data); err == nil {
			out, err := EncodeGERANPS(d.Value)
			if err != nil || !bytes.Equal(out, data) {
				t.Fatalf("GERAN PS round trip: %v", err)
			}
		}
	})
}
