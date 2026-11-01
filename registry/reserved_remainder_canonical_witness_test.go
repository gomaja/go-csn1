package registry

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.2.29,11.2.31,12.42b import Pulse Format IE
// (§12.8.3). Its bitmap is a fixed intersection and a priority fallback;
// these source-derived values exercise both public canonical error classes.
func reservedRemainderCanonicalWitnesses(t *testing.T, d runtime.Descriptor, record func(string, error, int)) {
	t.Helper()
	pkg := ""
	if d.Standard == "TS 44.060" {
		switch d.Clause + "/" + d.Name {
		case "11.2.29/Packet Uplink Assignment message content":
			pkg = "packetuplinkassignment"
		case "11.2.31/Packet Timeslot Reconfigure message content":
			pkg = "packettimeslotreconfigure"
		case "12.42b/PS Handover Radio Resources 3 IE":
			pkg = "pshandoverradioresources3"
		}
	}
	if pkg == "" {
		return
	}
	data, e := os.ReadFile("../ts44060/" + pkg + "/testdata/clause_vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var vectors []struct {
		Bits           string
		Extent, Origin int
	}
	if e = json.Unmarshal(data, &vectors); e != nil {
		t.Fatal(e)
	}
	var base any
	for _, tc := range vectors {
		if tc.Origin != SourceOrigin(d) {
			continue
		}
		raw := make([]byte, (len(tc.Bits)+7)/8)
		for i, c := range tc.Bits {
			if c == '1' {
				raw[i/8] |= 1 << uint(7-i%8)
			}
		}
		r, e := runtime.NewReaderBits(raw, tc.Extent, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.ReadUint(tc.Origin); e != nil {
			t.Fatal(e)
		}
		if _, e = r.PushLimit(tc.Extent - tc.Origin); e != nil {
			t.Fatal(e)
		}
		v, e := d.DecodeFrom(r)
		if e != nil {
			t.Fatal(e)
		}
		value := reflect.ValueOf(v).FieldByName("Value").Interface()
		copy := deepCopy(value)
		if reservedRemainderBitmap(copy).IsValid() {
			base = value
			break
		}
	}
	if base == nil {
		t.Fatal("source bitmap witness absent")
	}
	changed := deepCopy(base)
	bitmap := reservedRemainderBitmap(changed)
	bitmap.FieldByName("PulseFormatBitmapLength").SetUint(0)
	_, e = d.CanonicalAtLength(changed.Interface(), 23)
	var bound *runtime.BoundError
	if !errors.As(e, &bound) || bound.Kind != runtime.LengthBound {
		t.Fatalf("source bitmap bound witness: %v", e)
	}
	record(entryAtLength, e, 23)
	changed = deepCopy(base)
	bitmap = reservedRemainderBitmap(changed)
	length := bitmap.FieldByName("PulseFormatBitmapLength").Uint() + 1
	fallback := bitmap.FieldByName("PulseFormatBitmap")
	fallback.FieldByName("Known").SetZero()
	fallback.FieldByName("Alternative").SetUint(1)
	bits := runtime.BitString{Bytes: make([]byte, (length+7)/8), BitLength: int(length)}
	for i := range bits.Bytes {
		bits.Bytes[i] = 0xff
	}
	fallback.FieldByName("Ignored").Set(reflect.ValueOf(&bits))
	_, e = d.CanonicalAtLength(changed.Interface(), 23)
	var ambiguous *runtime.FallbackError
	if !errors.As(e, &ambiguous) {
		t.Fatalf("source bitmap fallback witness: %v", e)
	}
	record(entryAtLength, e, 23)
}

func reservedRemainderBitmap(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return reflect.Value{}
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		return reservedRemainderBitmap(v.Elem())
	}
	if v.Kind() == reflect.Struct {
		if a, b := v.FieldByName("PulseFormatBitmapLength"), v.FieldByName("PulseFormatBitmap"); a.IsValid() && a.Kind() == reflect.Uint8 && b.IsValid() && b.Kind() == reflect.Struct {
			return v
		}
		for i := 0; i < v.NumField(); i++ {
			if !v.Type().Field(i).IsExported() || v.Type().Field(i).Name == "Wire" {
				continue
			}
			if x := reservedRemainderBitmap(v.Field(i)); x.IsValid() {
				return x
			}
		}
	}
	if v.Kind() == reflect.Slice {
		for i := 0; i < v.Len(); i++ {
			if x := reservedRemainderBitmap(v.Index(i)); x.IsValid() {
				return x
			}
		}
	}
	return reflect.Value{}
}

func TestReservedRemainderCanonicalWitnesses(t *testing.T) {
	for _, key := range []struct{ clause, name string }{{"11.2.31", "Packet Timeslot Reconfigure message content"}, {"12.42b", "PS Handover Radio Resources 3 IE"}, {"11.2.29", "Packet Uplink Assignment message content"}} {
		d, e := LookupClause("TS 44.060", key.clause, key.name)
		if e != nil {
			t.Fatal(e)
		}
		seen := map[string]bool{}
		reservedRemainderCanonicalWitnesses(t, d, func(_ string, e error, _ int) { seen[classify(e, 23, entryAtLength)] = true })
		if !seen[classLength] || !seen[classFallback] {
			t.Fatal("incomplete canonical witnesses")
		}
	}
}
