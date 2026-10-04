package restoctets

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type utranFrequency struct {
	arfcns                   []uint16
	priority, low, qrxlevmin *uint8
	high                     uint8
}

func u8(v uint8) *uint8 { return &v }

// utranVectors are synthetic. TS 44.018 V19.0.0 §10.5.2.37o table
// 10.5.2.37o.1 prints two bit-identical bandwidth alternatives, which the
// source correction merges into one optional field (gomaja/go-csn1#20).
// pycrate 0.7.11 si_23_rest_octets decodes each description, alone and
// inside SI 23 Rest Octets, to the same bandwidth and neighbour values.
var utranVectors = []struct {
	name, description, si23 string
	bandwidth               *uint8
	frequencies             []utranFrequency
}{
	{"bandwidth absent", "00", "2068032b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b", nil, nil},
	{"bandwidth present", "d0", "206e802b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b", u8(5), nil},
	{"bandwidth and frequencies", "9ea3c5a93f4a1e95c3f600", "206cf51e2d49fa50f4ae1fb0012b2b2b2b2b2b2b", u8(1), []utranFrequency{
		{arfcns: []uint16{10812}, priority: u8(3), high: 10, qrxlevmin: u8(7)},
		{arfcns: []uint16{10563, 10588}, high: 31, low: u8(12)},
	}},
	{"frequency without bandwidth", "751e2d49c0", "206ba8f16a4e012b2b2b2b2b2b2b2b2b2b2b2b2b", nil, []utranFrequency{
		{arfcns: []uint16{10812}, priority: u8(3), high: 10, qrxlevmin: u8(7)},
	}},
}

func TestUTRANFDDTDDDescriptionDecodes(t *testing.T) {
	for _, tc := range utranVectors {
		t.Run(tc.name, func(t *testing.T) {
			wire, _ := hex.DecodeString(tc.description)
			decoded, err := DecodeUTRANFDDTDDDescriptionStruct(wire)
			if err != nil {
				t.Fatal(err)
			}
			checkUTRANDescription(t, decoded.Value, tc.bandwidth, tc.frequencies)
			if plain, err := EncodeUTRANFDDTDDDescriptionStruct(decoded.Value); err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("plain re-encode %x, %v", plain, err)
			}
			canonical, err := EncodeUTRANFDDTDDDescriptionStructCanonicalAtLength(decoded.Value, len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("canonical %x, %v", canonical, err)
			}

			si23, _ := hex.DecodeString(tc.si23)
			message, err := DecodeSI23RestOctets(si23)
			if err != nil {
				t.Fatal(err)
			}
			nested := findUTRANDescriptions(reflect.ValueOf(message.Value))
			if len(nested) != 1 {
				t.Fatalf("SI 23 carries %d UTRAN FDD/TDD descriptions", len(nested))
			}
			checkUTRANDescription(t, nested[0], tc.bandwidth, tc.frequencies)
			if plain, err := EncodeSI23RestOctets(message.Value); err != nil || !bytes.Equal(plain, si23) {
				t.Fatalf("SI 23 plain re-encode %x, %v", plain, err)
			}
			canonical, err = EncodeSI23RestOctetsCanonical(message.Value)
			if err != nil || !bytes.Equal(canonical, si23) {
				t.Fatalf("SI 23 canonical %x, %v", canonical, err)
			}
			again, err := DecodeSI23RestOctets(canonical)
			if err != nil || !runtime.SemanticallyEqual(message.Value, again.Value) {
				t.Fatalf("SI 23 canonical decodes to a different value: %v", err)
			}
		})
	}
}

func checkUTRANDescription(t *testing.T, v UTRANFDDTDDDescriptionStruct, bandwidth *uint8, frequencies []utranFrequency) {
	t.Helper()
	if !reflect.DeepEqual(v.BandwidthFDDOrTDD, bandwidth) {
		t.Fatalf("bandwidth %v, want %v", v.BandwidthFDDOrTDD, bandwidth)
	}
	if len(v.RepeatedUTRANFDDTDDNeighbourFrequencyAndPriorityList) != len(frequencies) {
		t.Fatalf("%d neighbour frequency entries, want %d", len(v.RepeatedUTRANFDDTDDNeighbourFrequencyAndPriorityList), len(frequencies))
	}
	for i, entry := range v.RepeatedUTRANFDDTDDNeighbourFrequencyAndPriorityList {
		got, want := entry.RepeatedUTRANFDDTDDNeighbourFrequencyAndPriority, frequencies[i]
		var arfcns []uint16
		for _, a := range got.UTRANARFCNList {
			arfcns = append(arfcns, a.UTRANARFCN)
		}
		if !reflect.DeepEqual(arfcns, want.arfcns) || !reflect.DeepEqual(got.UTRANPRIORITY, want.priority) || got.THRESHUTRANHigh != want.high ||
			!reflect.DeepEqual(got.THRESHUTRANLow, want.low) || !reflect.DeepEqual(got.UTRANQRXLEVMIN, want.qrxlevmin) {
			t.Fatalf("entry %d: %+v, want %+v", i, got, want)
		}
	}
}

func findUTRANDescriptions(v reflect.Value) []UTRANFDDTDDDescriptionStruct {
	var out []UTRANFDDTDDDescriptionStruct
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			out = append(out, findUTRANDescriptions(v.Elem())...)
		}
	case reflect.Struct:
		if description, ok := v.Interface().(UTRANFDDTDDDescriptionStruct); ok {
			return []UTRANFDDTDDDescriptionStruct{description}
		}
		for i := range v.NumField() {
			if v.Type().Field(i).IsExported() {
				out = append(out, findUTRANDescriptions(v.Field(i))...)
			}
		}
	case reflect.Slice:
		for i := range v.Len() {
			out = append(out, findUTRANDescriptions(v.Index(i))...)
		}
	}
	return out
}
