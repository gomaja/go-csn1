package ies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.11 defines the fields and selectors exercised below.
func TestClause1211(t *testing.T) {
	elementaryClause(t, "12.11", "Packet Request Reference IE", `[{"name":"zero","clause":"12.11","hex":"00000000","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":0,"FRAMENUMBER":0},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]},{"name":"one","clause":"12.11","hex":"00200020","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":1,"FRAMENUMBER":1},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]},{"name":"maximum","clause":"12.11","hex":"ffffffe0","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":2047,"FRAMENUMBER":65535},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]},{"name":"alternating","clause":"12.11","hex":"55555540","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":682,"FRAMENUMBER":43690},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]},{"name":"boundary-RANDOMACCESSINFORMATIONValue","clause":"12.11","hex":"ffe00000","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":2047,"FRAMENUMBER":0},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]},{"name":"boundary-FRAMENUMBER","clause":"12.11","hex":"001fffe0","bits":27,"values":{"RANDOMACCESSINFORMATIONValue":0,"FRAMENUMBER":65535},"widths":{"RANDOMACCESSINFORMATIONValue":11,"FRAMENUMBER":16},"absent":[],"present":[]}]`)
}

type elementaryVector struct {
	Name, Clause, Hex string
	Bits              int
	Values            map[string]uint64
	Widths            map[string]int
	Absent, Present   []string
}

func elementaryField(t *testing.T, value reflect.Value, path string) reflect.Value {
	t.Helper()
	for _, name := range strings.Split(path, ".") {
		for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
			if value.IsNil() {
				t.Fatalf("nil field in %s", path)
			}
			value = value.Elem()
		}
		if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
			index, err := strconv.Atoi(name)
			if err != nil || index < 0 || index >= value.Len() {
				t.Fatalf("invalid index %s in %s", name, path)
			}
			value = value.Index(index)
		} else {
			value = value.FieldByName(name)
		}
		if !value.IsValid() {
			t.Fatalf("missing typed field %s", path)
		}
	}
	return value
}

func elementaryNumber(t *testing.T, field reflect.Value) uint64 {
	t.Helper()
	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			t.Fatal("absent number")
		}
		field = field.Elem()
	}
	if field.Kind() == reflect.Struct {
		b, ok := field.Interface().(runtime.BitString)
		if !ok {
			t.Fatalf("unexpected numeric field %s", field.Type())
		}
		var n uint64
		for i := 0; i < b.BitLength; i++ {
			n = n<<1 | uint64((b.Bytes[i/8]>>uint(7-i%8))&1)
		}
		return n
	}
	return field.Uint()
}

func elementaryInput(t *testing.T, text string) []byte {
	t.Helper()
	wire, err := hex.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func elementaryClause(t *testing.T, clause, name, encodedVectors string) {
	t.Helper()
	descriptor, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	var vectors []elementaryVector
	if err := json.Unmarshal([]byte(encodedVectors), &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("empty vectors")
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			wire := elementaryInput(t, vector.Hex)
			value, err := descriptor.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			root := reflect.ValueOf(value).FieldByName("Value")
			check := func(decoded any) {
				r := reflect.ValueOf(decoded).FieldByName("Value")
				for path, want := range vector.Values {
					if got := elementaryNumber(t, elementaryField(t, r, path)); got != want {
						t.Errorf("%s=%d want %d", path, got, want)
					}
				}
				for _, path := range vector.Absent {
					if !elementaryField(t, r, path).IsNil() {
						t.Errorf("%s present", path)
					}
				}
				for _, path := range vector.Present {
					if elementaryField(t, r, path).IsNil() {
						t.Errorf("%s absent", path)
					}
				}
			}
			check(value)
			trailing := append(append([]byte{}, wire...), 0xa5, 0xff)
			withTail, tailErr := descriptor.Decode(trailing)
			if tailErr != nil {
				t.Fatalf("received tail: %v", tailErr)
			}
			check(withTail)
			tailReplay, tailErr := descriptor.Encode(withTail)
			if tailErr != nil || !bytes.Equal(tailReplay, trailing) {
				t.Fatalf("received tail replay %x: %v", tailReplay, tailErr)
			}
			replay, err := descriptor.Encode(value)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			fresh := runtime.Canonical(root.Interface())
			canonical, err := descriptor.CanonicalAtLength(fresh, len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
			}
			decoded, err := descriptor.Decode(canonical)
			if err != nil {
				t.Fatal(err)
			}
			check(decoded)
			stable, err := descriptor.CanonicalAtLength(decoded, len(wire))
			if err != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, err)
			}
			// TS 24.007 V20.0.0 Annex B: every field must fit inside the caller's bit extent.
			bounded := runtime.NewReader(append(append([]byte{}, wire...), 0xff))
			if _, err := bounded.PushLimit(vector.Bits); err != nil {
				t.Fatal(err)
			}
			from, err := descriptor.DecodeFrom(bounded)
			if err != nil || bounded.Position() != vector.Bits {
				t.Fatalf("bounded decode position %d want %d: %v", bounded.Position(), vector.Bits, err)
			}
			check(from)
			for cut := 0; cut < vector.Bits; cut++ {
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				if _, err := descriptor.DecodeFrom(r); err == nil {
					t.Fatalf("accepted truncated bit extent %d of %d", cut, vector.Bits)
				}
				if r.Position() > cut {
					t.Fatalf("reader escaped bit extent %d", cut)
				}
			}
			for path, width := range vector.Widths {
				bad := reflect.ValueOf(runtime.Canonical(root.Interface()))
				mutable := reflect.New(bad.Type()).Elem()
				mutable.Set(bad)
				field := elementaryField(t, mutable, path)
				for field.Kind() == reflect.Pointer {
					field = field.Elem()
				}
				if field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64 {
					for _, boundary := range []uint64{0, (uint64(1) << uint(width)) - 1} {
						field.SetUint(boundary)
						encoded, err := descriptor.CanonicalAtLength(mutable.Interface(), len(wire))
						if err != nil {
							t.Fatalf("rejected %s boundary %d: %v", path, boundary, err)
						}
						decoded, err := descriptor.Decode(encoded)
						if err != nil {
							t.Fatal(err)
						}
						decodedRoot := reflect.ValueOf(decoded).FieldByName("Value")
						if got := elementaryNumber(t, elementaryField(t, decodedRoot, path)); got != boundary {
							t.Fatalf("%s boundary decoded as %d, want %d", path, got, boundary)
						}
					}
				}
				if field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64 && width < field.Type().Bits() {
					field.SetUint(uint64(1) << uint(width))
					if _, err := descriptor.CanonicalAtLength(mutable.Interface(), len(wire)); err == nil {
						t.Errorf("accepted overflowing %s at width %d", path, width)
					}
				}
			}
		})
	}
}

func FuzzElementaryClauseRoundTrip(f *testing.F) {
	roots := []struct{ clause, name string }{
		{"12.11", "Packet Request Reference IE"}, {"12.13", "Power Control Parameters IE"}, {"12.14", "PRACH Control Parameters IE"}, {"12.16a", "G-RNTI IE"}, {"12.23", "Cell Identification IE"}, {"12.25", "PCCCH Organization Parameters IE"}, {"12.45", "MBMS In-band Signalling Indicator IE"}, {"12.7", "Channel Request Description IE"}, {"12.7a", "Iu mode Channel Request Description IE"}, {"12.7b", "Extended Channel Request Description IE"}, {"12.9", "Global Power Control Parameters IE"},
	}
	for i := range roots {
		for _, fill := range []byte{0, 0x2b, 0xff} {
			f.Add(uint8(i), bytes.Repeat([]byte{fill}, 8))
		}
	}
	f.Fuzz(func(t *testing.T, index uint8, wire []byte) {
		if len(wire) > 512 {
			return
		}
		root := roots[int(index)%len(roots)]
		d, err := LookupClause(root.clause, root.name)
		if err != nil {
			t.Fatal(err)
		}
		value, err := d.Decode(wire)
		if err != nil {
			return
		}
		replay, err := d.Encode(value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("accepted input did not replay: %x: %v", replay, err)
		}
	})
}

func elementaryRequestGrowth(t *testing.T, clause, name string, entryBits int) {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	var previous float64
	for _, count := range []int{16, 64, 256, 1024} {
		wire := make([]byte, (count*entryBits+7)/8)
		for i := 0; i < count-1; i++ {
			bit := (i+1)*entryBits - 1
			wire[bit/8] |= 1 << uint(7-bit%8)
		}
		value, err := d.Decode(wire)
		if err != nil {
			t.Fatal(err)
		}
		entries := reflect.ValueOf(value).FieldByName("Value").FieldByName("Entries")
		if entries.Len() != count || entries.Cap() > 2*count {
			t.Fatalf("%d entries: length %d capacity %d", count, entries.Len(), entries.Cap())
		}
		allocations := testing.AllocsPerRun(3, func() {
			if _, decodeErr := d.Decode(wire); decodeErr != nil {
				panic(decodeErr)
			}
		})
		if previous > 0 && allocations > 5*previous+8 {
			t.Fatalf("allocation count grew beyond a linear bound: %g to %g", previous, allocations)
		}
		previous = allocations
	}
}
