package ies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type publicationVector struct {
	Hex        string
	Bits       int
	Value      string
	PresentNil []string
}

func publicationField(value reflect.Value, path string) reflect.Value {
	for _, part := range strings.Split(path, ".") {
		for value.Kind() == reflect.Pointer {
			value = value.Elem()
		}
		if value.Kind() == reflect.Slice {
			index, err := strconv.Atoi(part)
			if err != nil {
				panic(err)
			}
			value = value.Index(index)
		} else {
			value = value.FieldByName(part)
		}
	}
	return value
}

func publicationFresh(t testing.TB, typ reflect.Type, vector publicationVector) reflect.Value {
	t.Helper()
	value := reflect.New(typ)
	if err := json.Unmarshal([]byte(vector.Value), value.Interface()); err != nil {
		t.Fatal(err)
	}
	for _, path := range vector.PresentNil {
		field := publicationField(value.Elem(), path)
		field.Set(reflect.New(field.Type().Elem()))
	}
	return value.Elem()
}

// The bit widths below are from TS 44.060 V19.0.0 tables 12.48a.4.1,
// 12.65.1, 12.8.2.1, 12.8.4.1 and delegated table 12.10a.1.
func publicationInvalidFields(t *testing.T, d runtime.Descriptor, typ reflect.Type, vector publicationVector) {
	t.Helper()
	widths := map[string]int{"MAIO1": 6, "MAIO2": 6, "EXTENDEDDYNAMICALLOCATION": 1, "USFGRANULARITY": 1, "P0C1": 4, "P0C2": 4, "PRMODEC1": 1, "PRMODEC2": 1, "UPLINKTFIASSIGNMENT": 5, "NUSF": 4, "NTS": 4, "NPAIRS": 3, "USF": 3, "USFList": 3, "USF2": 3, "GAMMA": 5, "ALPHAC1": 4, "ALPHAC2": 4, "ALPHA": 4, "TSC": 3, "ARFCN": 10, "ARFCN1": 10, "ARFCN2": 10, "MANUMBER": 4, "CHANGEMARK1": 2, "CHANGEMARK2": 2, "HSN": 6, "LengthOfMAFrequencyListContents": 4, "MALENGTH": 6, "RFLNUMBER": 4, "ARFCNINDEX": 6}
	var walk func(reflect.Value, string, string)
	walk = func(value reflect.Value, path, name string) {
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return
			}
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Struct:
			if value.Type() == reflect.TypeFor[runtime.BitString]() {
				invalid := publicationFresh(t, typ, vector)
				field := publicationField(invalid, path)
				field.FieldByName("BitLength").SetInt(value.FieldByName("BitLength").Int() + 1)
				if _, err := d.CanonicalAtLength(invalid.Interface(), (vector.Bits+7)/8); err == nil {
					t.Errorf("invalid %s bit length accepted", path)
				}
				return
			}
			for i := 0; i < value.NumField(); i++ {
				f := value.Type().Field(i)
				if f.Name == "Wire" || value.Type() == reflect.TypeFor[runtime.BitString]() {
					continue
				}
				p := f.Name
				if path != "" {
					p = path + "." + p
				}
				walk(value.Field(i), p, f.Name)
			}
		case reflect.Slice:
			for i := 0; i < value.Len(); i++ {
				walk(value.Index(i), fmt.Sprintf("%s.%d", path, i), name)
			}
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			width, ok := widths[name]
			if name == "Alternative" {
				width = 8
				ok = true
			}
			if !ok {
				t.Fatalf("missing field boundary for %s", path)
			}
			invalid := publicationFresh(t, typ, vector)
			field := publicationField(invalid, path)
			for field.Kind() == reflect.Pointer {
				field = field.Elem()
			}
			if name == "Alternative" {
				field.SetUint(255)
			} else {
				field.SetUint(uint64(1) << uint(width))
			}
			if _, err := d.CanonicalAtLength(invalid.Interface(), (vector.Bits+7)/8); err == nil {
				t.Errorf("invalid %s accepted", path)
			}
		}
	}
	walk(publicationFresh(t, typ, vector), "", "")
}

func publicationCarrier(wire []byte, bits int) []byte {
	out := make([]byte, (bits+8+7)/8)
	for i := 0; i < bits+8; i++ {
		var bit byte
		switch {
		case i < 3:
			bit = (5 >> uint(2-i)) & 1
		case i < bits+3:
			j := i - 3
			bit = (wire[j/8] >> uint(7-j%8)) & 1
		default:
			bit = (21 >> uint(bits+7-i)) & 1
		}
		out[i/8] |= bit << uint(7-i%8)
	}
	return out
}

func checkPublicationVectors(t *testing.T, clause, name string, typ reflect.Type, vectors []publicationVector) {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	for i, vector := range vectors {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			wire, err := hex.DecodeString(vector.Hex)
			if err != nil {
				t.Fatal(err)
			}
			expected := publicationFresh(t, typ, vector)
			// Encode a freshly populated concrete type: received replay state cannot mask a field or selector defect.
			canonical, err := d.CanonicalAtLength(expected.Interface(), len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("fresh canonical %x want %x: %v", canonical, wire, err)
			}
			decoded, err := d.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			got := reflect.ValueOf(decoded).FieldByName("Value")
			if got.Type() != typ || !runtime.SemanticallyEqual(got.Interface(), expected.Interface()) {
				t.Fatalf("typed value %#v want %#v", got.Interface(), expected.Interface())
			}
			replay, err := d.Encode(decoded)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			stable, err := d.CanonicalAtLength(decoded, len(wire))
			if err != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, err)
			}
			for limit := 0; limit < vector.Bits; limit++ {
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(limit); err != nil {
					t.Fatal(err)
				}
				if _, err := d.DecodeFrom(r); err == nil {
					t.Fatalf("truncation at bit %d accepted", limit)
				}
				if r.Position() > limit {
					t.Fatalf("reader crossed bit %d", limit)
				}
			}
			r := runtime.NewReader(publicationCarrier(wire, vector.Bits))
			if _, err := r.ReadUint(3); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(vector.Bits)
			if err != nil {
				t.Fatal(err)
			}
			bounded, err := d.DecodeFrom(r)
			if err != nil || r.Position() != vector.Bits+3 {
				t.Fatalf("bounded reader at %d: %v", r.Position(), err)
			}
			boundedValue := reflect.ValueOf(bounded).FieldByName("Value")
			if !runtime.SemanticallyEqual(boundedValue.Interface(), expected.Interface()) {
				t.Fatal("bounded typed value differs")
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			tail, err := r.ReadUint(5)
			if err != nil || tail != 21 {
				t.Fatalf("caller suffix %d: %v", tail, err)
			}
			publicationInvalidFields(t, d, typ, vector)
		})
	}
}

// TS 44.060 V19.0.0 §12.48a.4, DOCX paragraphs 24832-24865.
func TestClause1248a4Vectors(t *testing.T) {
	checkPublicationVectors(t, "12.48a.4", "Dynamic Allocation 2 IE", reflect.TypeFor[DynamicAllocation2IE](), []publicationVector{
		{"a000", 10, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"a040", 13, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"a0a4", 14, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"a0c480", 17, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"a3a56d78", 29, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"a3c4d5e6f780", 41, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"a7a56d7a56d780", 49, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"a7c4d5e6f7c4d5e6f780", 73, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"af80", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"afe411f0", 28, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"af829f00", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"afe431f9f0", 37, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"af8e9f2f8dd3f0", 55, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"afe4f1f9f576f99bdd767f00", 91, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"af9e9f2f8dd3f09b2e8d93e0", 95, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"afe5f1f9f576f99bdd767f1179b556e993d9747e00", 163, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"afa0", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"afe623f0", 29, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"afa53e00", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"afe663f3eba1", 48, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"afbd3e5f1ba7e0", 55, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"afe7e3f3eaedf337baecfe2e874b54a89c", 134, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"bf80", 15, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"bf8200", 18, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"bf8520", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"bf8624", 22, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"bf9d2b6bc0", 34, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"bf9e26af37bc", 46, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"bfbd2b6bd2b6bc", 54, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"bfbe26af37be26af37bc", 78, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"bffc00", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"bfff208f80", 33, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"bffc14f8", 30, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"bfff218fcf80", 42, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"bffc74f97c6e9f80", 60, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"bfff278fcfabb7ccdeebb3f8", 96, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"bffcf4f97c6e9f84d9746c9f00", 100, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"bfff2f8fcfabb7ccdeebb3f88bcdaab74c9ecba3f0", 168, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"bffd00", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"bfff311f80", 34, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"bffd29f0", 30, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"bfff331f9f5d08", 53, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"bffde9f2f8dd3f00", 60, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"bfff3f1f9f576f99bdd767f1743a5aa544e0", 139, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"fe80", 16, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"fe8100", 19, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"fe8290", 20, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"fe8312", 23, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"fe8e95b5e0", 35, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"fe8f13579bde", 47, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"fe9e95b5e95b5e", 55, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"fe9f13579bdf13579bde", 79, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"febe00", 22, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"febf9047c0", 34, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"febe0a7c", 31, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"febf90c7e7c0", 43, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"febe3a7cbe374fc0", 61, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"febf93c7e7d5dbe66f75d9fc00", 97, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"febe7a7cbe374fc26cba364f80", 101, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"febf97c7e7d5dbe66f75d9fc45e6d55ba64f65d1f800", 169, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"febe80", 22, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"febf988fc0", 35, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"febe94f8", 31, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"febf998fcfae84", 54, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"febef4f97c6e9f80", 61, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"febf9f8fcfabb7ccdeebb3f8ba1d2d52a270", 140, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"fefe00", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"fefe08", 24, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"fefe1480", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"fefe1890", 28, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"fefe74adaf", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"fefe789abcdef0", 52, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"fefef4adaf4adaf0", 60, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"fefef89abcdef89abcdef0", 84, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"fefff000", 27, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"fefffc823e", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"fefff053e0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"fefffc863f3e", 48, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"fefff1d3e5f1ba7e00", 66, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"fefffc9e3f3eaedf337baecfe0", 102, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"fefff3d3e5f1ba7e1365d1b27c00", 106, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"fefffcbe3f3eaedf337baecfe22f36aadd327b2e8fc0", 174, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"fefff400", 27, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"fefffcc47e", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"fefff4a7c0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"fefffccc7e7d7420", 59, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"fefff7a7cbe374fc00", 66, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"fefffcfc7e7d5dbe66f75d9fc5d0e96a951380", 145, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":null},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"ff9400", 21, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"ff9408", 24, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"ff941480", 25, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"ff941890", 28, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"ff9474adaf", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"ff94789abcdef0", 52, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"ff94f4adaf4adaf0", 60, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"ff94f89abcdef89abcdef0", 84, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"ff95f000", 27, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"ff95fc823e", 39, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"ff95f053e0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"ff95fc863f3e", 48, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"ff95f1d3e5f1ba7e00", 66, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"ff95fc9e3f3eaedf337baecfe0", 102, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"ff95f3d3e5f1ba7e1365d1b27c00", 106, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"ff95fcbe3f3eaedf337baecfe22f36aadd327b2e8fc0", 174, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"ff95f400", 27, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"ff95fcc47e", 40, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"ff95f4a7c0", 36, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"ff95fccc7e7d7420", 59, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"ff95f7a7cbe374fc00", 66, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"ff95fcfc7e7d5dbe66f75d9fc5d0e96a951380", 145, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"ff97f000", 26, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
		{"ff97f040", 29, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[0]},"ALPHAC1":null}}`, nil},
		{"ff97f0a4", 30, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[null,1]},"ALPHAC1":null}}`, nil},
		{"ff97f0c480", 33, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":1,"USFList":[0,1]},"ALPHAC1":null}}`, nil},
		{"ff97f3a56d78", 45, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"ff97f3c4d5e6f780", 57, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":7,"USFList":[0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"ff97f7a56d7a56d780", 65, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[null,1,null,3,null,5,null,7,null,1,null,3,null,5,null,7]},"ALPHAC1":null}}`, nil},
		{"ff97f7c4d5e6f7c4d5e6f780", 89, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":15,"USFList":[0,1,2,3,4,5,6,7,0,1,2,3,4,5,6,7]},"ALPHAC1":null}}`, nil},
		{"ff97ff80", 32, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[null]},"NPAIRS":null}}}}`, nil},
		{"ff97ffe411f0", 44, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":0,"USFGroupList":[{"USF":0,"GAMMA":31}]},"NPAIRS":null}}}}`, nil},
		{"ff97ff829f00", 41, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"ff97ffe431f9f0", 53, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}]},"NPAIRS":null}}}}`, nil},
		{"ff97ff8e9f2f8dd3f0", 71, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"ff97ffe4f1f9f576f99bdd767f00", 107, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}]},"NPAIRS":null}}}}`, nil},
		{"ff97ff9e9f2f8dd3f09b2e8d93e0", 111, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24},null,{"USF":1,"GAMMA":22},null,{"USF":3,"GAMMA":20},null,{"USF":5,"GAMMA":18},null,{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"ff97ffe5f1f9f576f99bdd767f1179b556e993d9747e00", 179, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":0,"NTS":{"NTS":15,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24},{"USF":0,"GAMMA":23},{"USF":1,"GAMMA":22},{"USF":2,"GAMMA":21},{"USF":3,"GAMMA":20},{"USF":4,"GAMMA":19},{"USF":5,"GAMMA":18},{"USF":6,"GAMMA":17},{"USF":7,"GAMMA":16}]},"NPAIRS":null}}}}`, nil},
		{"ff97ffa0", 32, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[null],"USF2GroupList":null}}}}}`, nil},
		{"ff97ffe623f0", 45, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":0,"USFGroupList":[{"USF":0,"GAMMA":31}],"USF2GroupList":[null]}}}}}`, nil},
		{"ff97ffa53e00", 41, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[null,{"USF":1,"GAMMA":30}],"USF2GroupList":null}}}}}`, nil},
		{"ff97ffe663f3eba1", 64, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":1,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1}]}}}}}`, nil},
		{"ff97ffbd3e5f1ba7e0", 71, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":null,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[null,{"USF":1,"GAMMA":30},null,{"USF":3,"GAMMA":28},null,{"USF":5,"GAMMA":26},null,{"USF":7,"GAMMA":24}],"USF2GroupList":null}}}}}`, nil},
		{"ff97ffe7e3f3eaedf337baecfe2e874b54a89c", 150, `{"EXTENDEDDYNAMICALLOCATION":1,"P0C1Group":{"P0C1":15,"PRMODEC1":1,"P0C2Group":{"P0C2":9,"PRMODEC2":0}},"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":31,"NUSFChoice":{"Alternative":1,"NUSF":null,"ALPHAC1":{"ALPHAC1":15,"ALPHAC2":9,"NTSChoice":{"Alternative":1,"NTS":null,"NPAIRS":{"NPAIRS":7,"USFGroupList":[{"USF":0,"GAMMA":31},{"USF":1,"GAMMA":30},{"USF":2,"GAMMA":29},{"USF":3,"GAMMA":28},{"USF":4,"GAMMA":27},{"USF":5,"GAMMA":26},{"USF":6,"GAMMA":25},{"USF":7,"GAMMA":24}],"USF2GroupList":[null,{"USF2":6,"GAMMA":1},{"USF2":5,"GAMMA":null},null,{"USF2":3,"GAMMA":null},{"USF2":2,"GAMMA":5},null,{"USF2":0,"GAMMA":7}]}}}}}`, nil},
		{"2000", 10, `{"EXTENDEDDYNAMICALLOCATION":0,"P0C1Group":null,"USFGRANULARITY":1,"UPLINKTFIASSIGNMENT":null,"NUSFChoice":{"Alternative":0,"NUSF":{"NUSF":0,"USFList":[null]},"ALPHAC1":null}}`, nil},
	})
}

func FuzzCarrierAllocationReplay(f *testing.F) {
	roots := [][2]string{{"12.48a.4", "Dynamic Allocation 2 IE"}, {"12.65", "Dynamic Allocation 3 IE"}, {"12.8.2", "Dual Carrier Frequency Parameters IE"}, {"12.8.4", "DLMC Frequency Parameters IE"}}
	f.Add(uint8(0), []byte{0xa0, 0x00})
	f.Add(uint8(0), []byte{0xfe, 0xfe, 0x00})
	f.Add(uint8(0), []byte{0x20, 0x00})
	f.Add(uint8(1), []byte{0xc0})
	f.Add(uint8(1), []byte{0xfe, 0x00})
	f.Add(uint8(1), []byte{0x00})
	f.Add(uint8(2), []byte{0xe0})
	f.Add(uint8(2), []byte{0xf2, 0xff, 0xff, 0x93, 0xfe, 0x00})
	f.Add(uint8(2), []byte{0xff, 0xfb, 0xff, 0xf8, 0x00, 0x89, 0x11, 0x9a, 0x22, 0xab, 0x33, 0xbc, 0x44, 0xcd, 0x55, 0xde, 0x66, 0xef, 0x77, 0xf8, 0x81, 0x08})
	f.Add(uint8(3), []byte{0xe7, 0xfe})
	f.Add(uint8(3), []byte{0xf7, 0xff, 0x93, 0xfe, 0x00})
	f.Add(uint8(3), []byte{0xff, 0xfe, 0x00, 0x22, 0x44, 0x66, 0x88, 0xaa, 0xcc, 0xef, 0x11, 0x33, 0x55, 0x77, 0x99, 0xbb, 0xdd, 0xfe, 0x20, 0x42})
	f.Fuzz(func(t *testing.T, index uint8, wire []byte) {
		if len(wire) > 512 {
			return
		}
		root := roots[int(index)%len(roots)]
		d, err := LookupClause(root[0], root[1])
		if err != nil {
			t.Fatal(err)
		}
		value, err := d.Decode(wire)
		if err != nil {
			return
		}
		replay, err := d.Encode(value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %v", replay, err)
		}
	})
}
