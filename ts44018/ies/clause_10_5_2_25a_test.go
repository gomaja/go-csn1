package ies

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.25a, DOCX paragraphs 17795–17811.
// All inputs are synthetic. The value extent belongs to the containing IE.
func TestPacketChannelDescriptionPublication(t *testing.T) {
	descriptor, err := LookupClause("10.5.2.25a", "Packet Channel Description")
	if err != nil {
		t.Fatal(err)
	}
	vectors := []struct {
		name, wire, canonical string
		bits                  int
		numbers               map[string]uint64
		lengths               map[string]int
		invalid               map[string]uint64
		rejectCuts            []int
		boundedOnly           bool
	}{
		{name: "non-hopping", wire: "0fe7ff", canonical: "0fe3ff", bits: 24, numbers: map[string]uint64{"ChannelType": 1, "TN": 7, "TSC": 7, "ARFCNChoice.Alternative": 0, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 0, "ARFCNChoice.ARFCN.ARFCNChoice.ARFCN.ARFCN": 1023}, lengths: map[string]int{}, invalid: map[string]uint64{"ChannelType": 32, "TN": 8, "TSC": 8, "ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.ARFCN.ARFCN": 1024}, rejectCuts: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}, boundedOnly: false},
		{name: "indirect-0", wire: "0feff3", canonical: "0febf3", bits: 24, numbers: map[string]uint64{"ChannelType": 1, "TN": 7, "TSC": 7, "ARFCNChoice.Alternative": 0, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 1, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MAIO": 63, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MANUMBERIND": 0, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.Alternative": 0, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.SpareBits.SpareBits": 3}, lengths: map[string]int{}, invalid: map[string]uint64{"ChannelType": 32, "TN": 8, "TSC": 8, "ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MAIO": 64, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MANUMBERIND": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.SpareBits.SpareBits": 4}, rejectCuts: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}, boundedOnly: false},
		{name: "indirect-1", wire: "0fefff", canonical: "0febff", bits: 24, numbers: map[string]uint64{"ChannelType": 1, "TN": 7, "TSC": 7, "ARFCNChoice.Alternative": 0, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 1, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MAIO": 63, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MANUMBERIND": 1, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.Alternative": 1, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.CHANGEMARK1.CHANGEMARK1": 3}, lengths: map[string]int{}, invalid: map[string]uint64{"ChannelType": 32, "TN": 8, "TSC": 8, "ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MAIO": 64, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.MANUMBERIND": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.Alternative": 2, "ARFCNChoice.ARFCN.ARFCNChoice.MAIO.SpareBitsChoice.CHANGEMARK1.CHANGEMARK1": 4}, rejectCuts: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}, boundedOnly: false},
		{name: "direct", wire: "0fffff", canonical: "0fffff", bits: 24, numbers: map[string]uint64{"ChannelType": 1, "TN": 7, "TSC": 7, "ARFCNChoice.Alternative": 1, "ARFCNChoice.HSN.MAIO": 63, "ARFCNChoice.HSN.HSN": 63}, lengths: map[string]int{}, invalid: map[string]uint64{"ChannelType": 32, "TN": 8, "TSC": 8, "ARFCNChoice.Alternative": 2, "ARFCNChoice.HSN.MAIO": 64, "ARFCNChoice.HSN.HSN": 64}, rejectCuts: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}, boundedOnly: false},
	}
	fieldAt := func(value reflect.Value, path string) reflect.Value {
		for _, part := range strings.Split(path, ".") {
			for value.Kind() == reflect.Pointer {
				if value.IsNil() {
					t.Fatalf("nil pointer at %s", path)
				}
				value = value.Elem()
			}
			if value.Kind() == reflect.Slice {
				i, e := strconv.Atoi(part)
				if e != nil || i < 0 || i >= value.Len() {
					t.Fatalf("invalid index at %s", path)
				}
				value = value.Index(i)
			} else {
				value = value.FieldByName(part)
			}
			if !value.IsValid() {
				t.Fatalf("missing field %s", path)
			}
		}
		for value.Kind() == reflect.Pointer {
			if value.IsNil() {
				t.Fatalf("nil pointer at %s", path)
			}
			value = value.Elem()
		}
		return value
	}
	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			wire, e := hex.DecodeString(vector.wire)
			if e != nil {
				t.Fatal(e)
			}
			containing := make([]byte, (vector.bits+15)/8)
			for bit := 0; bit < vector.bits; bit++ {
				containing[bit/8] |= ((wire[bit/8] >> uint(7-bit%8)) & 1) << uint(7-bit%8)
			}
			for bit := 0; bit < 8; bit++ {
				pos := vector.bits + bit
				containing[pos/8] |= ((byte(0xa5) >> uint(7-bit)) & 1) << uint(7-pos%8)
			}
			reader := runtime.NewReader(containing)
			old, e := reader.PushLimit(vector.bits)
			if e != nil {
				t.Fatal(e)
			}
			boundedDecoded, e := descriptor.DecodeFrom(reader)
			if e != nil {
				t.Fatal(e)
			}
			typed := reflect.ValueOf(boundedDecoded).FieldByName("Value").Interface()
			if reflect.TypeOf(typed).Name() != "PacketChannelDescription" {
				t.Fatalf("untyped root %T", typed)
			}
			if reader.Position() != vector.bits {
				t.Fatalf("consumed %d of %d bits", reader.Position(), vector.bits)
			}
			if e = reader.PopLimit(old); e != nil {
				t.Fatal(e)
			}
			sentinel, e := reader.ReadUint(8)
			if e != nil || sentinel != 0xa5 {
				t.Fatalf("containing sentinel %x: %v", sentinel, e)
			}
			check := func(value any) {
				for path, want := range vector.numbers {
					if got := fieldAt(reflect.ValueOf(value), path).Uint(); got != want {
						t.Errorf("%s=%d want %d", path, got, want)
					}
				}
				for path, want := range vector.lengths {
					if got := fieldAt(reflect.ValueOf(value), path).Len(); got != want {
						t.Errorf("%s length=%d want %d", path, got, want)
					}
				}
			}
			check(typed)
			for _, cut := range vector.rejectCuts {
				truncated := runtime.NewReader(containing)
				if _, e = truncated.PushLimit(cut); e != nil {
					t.Fatal(e)
				}
				if _, e = descriptor.DecodeFrom(truncated); e == nil {
					t.Errorf("accepted malformed %d-bit prefix", cut)
				}
				if truncated.Position() > cut {
					t.Errorf("reader crossed %d-bit bound", cut)
				}
			}
			if vector.boundedOnly {
				encoded, e := descriptor.Encode(runtime.Canonical(typed))
				if e != nil || hex.EncodeToString(encoded) != vector.canonical {
					t.Fatalf("bounded canonical %x: %v", encoded, e)
				}
				freshReader := runtime.NewReader(encoded)
				if _, e = freshReader.PushLimit(vector.bits); e != nil {
					t.Fatal(e)
				}
				fresh, e := descriptor.DecodeFrom(freshReader)
				if e != nil {
					t.Fatal(e)
				}
				fresh = reflect.ValueOf(fresh).FieldByName("Value").Interface()
				check(fresh)
				stable, e := descriptor.Encode(runtime.Canonical(fresh))
				if e != nil || !bytes.Equal(stable, encoded) {
					t.Fatalf("unstable bounded canonical %x: %v", stable, e)
				}
				return
			}
			decoded, e := descriptor.Decode(wire)
			if e != nil {
				t.Fatal(e)
			}
			replay, e := descriptor.Encode(decoded)
			if e != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, e)
			}
			bare := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			check(bare)
			canonical, e := descriptor.CanonicalAtLength(bare, len(wire))
			if e != nil {
				t.Fatal(e)
			}
			if hex.EncodeToString(canonical) != vector.canonical {
				t.Fatalf("canonical %x want %s", canonical, vector.canonical)
			}
			fresh, e := descriptor.Decode(canonical)
			if e != nil {
				t.Fatal(e)
			}
			check(reflect.ValueOf(fresh).FieldByName("Value").Interface())
			stable, e := descriptor.CanonicalAtLength(fresh, len(wire))
			if e != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, e)
			}
			for path, bad := range vector.invalid {
				// Decode afresh for each edit so nested pointers and slices are independent.
				independent, e := descriptor.Decode(wire)
				if e != nil {
					t.Fatal(e)
				}
				mutable := reflect.New(reflect.TypeOf(independent)).Elem()
				mutable.Set(reflect.ValueOf(independent))
				target := fieldAt(mutable.FieldByName("Value"), path)
				if target.OverflowUint(bad) {
					t.Fatalf("invalid probe %s overflows Go storage", path)
				}
				target.SetUint(bad)
				if _, e = descriptor.CanonicalAtLength(mutable.Interface(), len(wire)); e == nil {
					t.Errorf("encoded out-of-range %s=%d", path, bad)
				}
			}
		})
	}
}

func FuzzPacketChannelDescriptionReplay(f *testing.F) {
	descriptor, err := LookupClause("10.5.2.25a", "Packet Channel Description")
	if err != nil {
		f.Fatal(err)
	}
	for _, s := range []string{"0fe7ff", "0feff3", "0fefff", "0fffff"} {
		wire, e := hex.DecodeString(s)
		if e != nil {
			f.Fatal(e)
		}
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 256 {
			return
		}
		value, e := descriptor.Decode(wire)
		if e != nil {
			return
		}
		replay, e := descriptor.Encode(value)
		if e != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %v", replay, e)
		}
	})
}
