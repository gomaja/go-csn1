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

type compactReducedMAVector struct {
	Hex        string
	Bits       int
	Value      string
	PresentNil []string
}

func compactReducedMAField(value reflect.Value, path string) reflect.Value {
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

func compactReducedMAFresh(t testing.TB, typ reflect.Type, vector compactReducedMAVector) reflect.Value {
	t.Helper()
	value := reflect.New(typ)
	if err := json.Unmarshal([]byte(vector.Value), value.Interface()); err != nil {
		t.Fatal(err)
	}
	for _, path := range vector.PresentNil {
		field := compactReducedMAField(value.Elem(), path)
		field.Set(reflect.New(field.Type().Elem()))
	}
	return value.Elem()
}

// TS 44.060 V19.0.0 §12.29 table 12.29.1, DOCX paragraphs 23514–23517.
func compactReducedMAInvalidFields(t *testing.T, d runtime.Descriptor, typ reflect.Type, vector compactReducedMAVector) {
	t.Helper()
	widths := map[string]int{"LengthOfReducedMABitmap": 7, "MAIO2": 6}
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
				invalid := compactReducedMAFresh(t, typ, vector)
				field := compactReducedMAField(invalid, path)
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
			invalid := compactReducedMAFresh(t, typ, vector)
			field := compactReducedMAField(invalid, path)
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
	walk(compactReducedMAFresh(t, typ, vector), "", "")
}

func compactReducedMACarrier(wire []byte, bits int) []byte {
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

func checkCOMPACTReducedMAVectors(t *testing.T, clause, name string, typ reflect.Type, vectors []compactReducedMAVector) {
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
			expected := compactReducedMAFresh(t, typ, vector)
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
			r := runtime.NewReader(compactReducedMACarrier(wire, vector.Bits))
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
			compactReducedMAInvalidFields(t, d, typ, vector)
		})
	}
}

// TS 44.060 V19.0.0 §12.29, DOCX paragraphs 23514-23517.
func TestClause1229Vectors(t *testing.T) {
	checkCOMPACTReducedMAVectors(t, "12.29", "COMPACT reduced MA IE", reflect.TypeFor[COMPACTReducedMAIE](), []compactReducedMAVector{
		{"00", 8, `{"LengthOfReducedMABitmap":0,"ReducedMABitmap":{"Bytes":"","BitLength":0},"MAIO2":null}`, nil},
		{"01fc", 14, `{"LengthOfReducedMABitmap":0,"ReducedMABitmap":{"Bytes":"","BitLength":0},"MAIO2":63}`, nil},
		{"0300", 9, `{"LengthOfReducedMABitmap":1,"ReducedMABitmap":{"Bytes":"gA==","BitLength":1},"MAIO2":null}`, nil},
		{"03fe", 15, `{"LengthOfReducedMABitmap":1,"ReducedMABitmap":{"Bytes":"gA==","BitLength":1},"MAIO2":63}`, nil},
		{"0f48", 15, `{"LengthOfReducedMABitmap":7,"ReducedMABitmap":{"Bytes":"pA==","BitLength":7},"MAIO2":null}`, nil},
		{"0f4bf8", 21, `{"LengthOfReducedMABitmap":7,"ReducedMABitmap":{"Bytes":"pA==","BitLength":7},"MAIO2":63}`, nil},
		{"114a", 16, `{"LengthOfReducedMABitmap":8,"ReducedMABitmap":{"Bytes":"pQ==","BitLength":8},"MAIO2":null}`, nil},
		{"114bfc", 22, `{"LengthOfReducedMABitmap":8,"ReducedMABitmap":{"Bytes":"pQ==","BitLength":8},"MAIO2":63}`, nil},
		{"ff4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b48", 135, `{"LengthOfReducedMABitmap":127,"ReducedMABitmap":{"Bytes":"paWlpaWlpaWlpaWlpaWlpA==","BitLength":127},"MAIO2":null}`, nil},
		{"ff4b4b4b4b4b4b4b4b4b4b4b4b4b4b4b4bf8", 141, `{"LengthOfReducedMABitmap":127,"ReducedMABitmap":{"Bytes":"paWlpaWlpaWlpaWlpaWlpA==","BitLength":127},"MAIO2":63}`, nil},
	})
}

// TS 44.060 V19.0.0 §12.29 paragraph 23520 permits a zero-length bitmap.
func TestCOMPACTReducedMAFreshZeroBitmap(t *testing.T) {
	for _, value := range []COMPACTReducedMAIE{{}, {ReducedMABitmap: runtime.BitString{Bytes: []byte{}}}} {
		wire, err := EncodeCOMPACTReducedMAIECanonicalAtLength(value, 1)
		if err != nil || !bytes.Equal(wire, []byte{0}) {
			t.Fatalf("zero bitmap canonical %x: %v", wire, err)
		}
	}
}
func FuzzCOMPACTReducedMAReplay(f *testing.F) {
	for _, wire := range [][]byte{{0}, {1, 252}, {3, 254}, {15, 75, 248}} {
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		decoded, err := DecodeCOMPACTReducedMAIE(wire)
		if err != nil {
			return
		}
		replay, err := EncodeCOMPACTReducedMAIE(decoded.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %v", replay, err)
		}
	})
}
