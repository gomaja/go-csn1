package ies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.60, paragraphs 21308–21310, 21313, imports the
// typed TS 44.060 component(s) through authenticated source bindings. All vectors are synthetic.
func TestMPRACHDescriptionValuePartPublication(t *testing.T) {
	descriptor, err := LookupClause("10.5.2.60", "MPRACH description value part")
	if err != nil {
		t.Fatal(err)
	}
	vectors := []struct {
		name, wire, canonical string
		bits, semanticBits    int
		numbers               map[string]uint64
	}{
		{name: "absent-control-absent", wire: "3a", canonical: "3a", bits: 8, semanticBits: 8, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "absent-control-empty", wire: "3b10", canonical: "3b10", bits: 16, semanticBits: 16, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "absent-control-full", wire: "3bd2ad64adbf", canonical: "3bd2ad64ad80", bits: 48, semanticBits: 42, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "arfcn-control-absent", wire: "d14175", canonical: "d14174", bits: 24, semanticBits: 23, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "arfcn-control-empty", wire: "d1417621", canonical: "d1417620", bits: 32, semanticBits: 31, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "arfcn-control-full", wire: "d14177a55ac95b7f", canonical: "d14177a55ac95b00", bits: 64, semanticBits: 57, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-control-absent", wire: "b5133a", canonical: "b5133a", bits: 24, semanticBits: 24, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-control-empty", wire: "b5133b10", canonical: "b5133b10", bits: 32, semanticBits: 32, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-control-full", wire: "b5133bd2ad64adbf", canonical: "b5133bd2ad64ad80", bits: 64, semanticBits: 58, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-marks-control-absent", wire: "b513d5d7", canonical: "b513d5d0", bits: 32, semanticBits: 29, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-marks-control-empty", wire: "b513d5d887", canonical: "b513d5d880", bits: 40, semanticBits: 37, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "indirect-marks-control-full", wire: "b513d5de956b256d", canonical: "b513d5de956b256c", bits: 64, semanticBits: 63, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct1-control-absent", wire: "a87253af", canonical: "a87253a0", bits: 32, semanticBits: 28, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct1-control-empty", wire: "a87253b10f", canonical: "a87253b100", bits: 40, semanticBits: 36, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct1-control-full", wire: "a87253bd2ad64adb", canonical: "a87253bd2ad64ad8", bits: 64, semanticBits: 62, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct2-control-absent", wire: "9cb34048d159d7", canonical: "9cb34048d159d0", bits: 56, semanticBits: 53, numbers: map[string]uint64{"MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct2-control-empty", wire: "9cb34048d159d887", canonical: "9cb34048d159d880", bits: 64, semanticBits: 61, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
		{name: "direct2-control-full", wire: "9cb34048d159de956b256d", canonical: "9cb34048d159de956b256c", bits: 88, semanticBits: 87, numbers: map[string]uint64{"MPRACHDescription.MPRACHControlParameters.ACCCONTRCLASS": 42330, "MPRACHDescription.MPRACHControlParameters.MAXRETRANS": 2, "MPRACHDescription.MPRACHControlParameters.PERSISTENCELEVEL": 6, "MPRACHDescription.MPRACHControlParameters.S": 4, "MPRACHDescription.MPRACHControlParameters.TXINT": 5, "MPRACHDescription.MPRACHTIMESLOTNUMBER": 3, "MPRACHDescription.USF": 5}},
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
					t.Fatalf("invalid index %s", path)
				}
				value = value.Index(i)
			} else {
				value = value.FieldByName(part)
			}
			if !value.IsValid() {
				t.Fatalf("missing typed field %s", path)
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
	bare := func(v any) reflect.Value { return reflect.ValueOf(v).FieldByName("Value") }
	for _, vector := range vectors {
		t.Run(vector.name, func(t *testing.T) {
			wire, e := hex.DecodeString(vector.wire)
			if e != nil {
				t.Fatal(e)
			}
			decoded, e := descriptor.Decode(wire)
			if e != nil {
				t.Fatal(e)
			}
			value := bare(decoded)
			if value.Type().Name() != "MPRACHDescriptionValuePart" {
				t.Fatalf("untyped root %T", decoded)
			}
			if f, ok := value.Type().FieldByName("MPRACHDescription"); !ok || f.Type.PkgPath() != "github.com/gomaja/go-csn1/ts44060/ies" {
				t.Fatalf("lost delegated type for MPRACHDescription")
			}
			for path, want := range vector.numbers {
				if got := fieldAt(value, path).Uint(); got != want {
					t.Errorf("%s=%d want %d", path, got, want)
				}
			}
			replay, e := descriptor.Encode(decoded)
			if e != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, e)
			}
			canonical, e := descriptor.CanonicalAtLength(value.Interface(), len(wire))
			if e != nil || hex.EncodeToString(canonical) != vector.canonical {
				t.Fatalf("canonical %x want %s: %v", canonical, vector.canonical, e)
			}
			fresh, e := descriptor.Decode(canonical)
			if e != nil {
				t.Fatal(e)
			}
			for path, want := range vector.numbers {
				if got := fieldAt(bare(fresh), path).Uint(); got != want {
					t.Errorf("canonical %s=%d want %d", path, got, want)
				}
			}
			stable, e := descriptor.CanonicalAtLength(fresh, len(wire))
			if e != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, e)
			}
			// Append a sentinel outside the explicit carrier bound at every bit phase.
			for phase := 0; phase < 8; phase++ {
				containing := make([]byte, (phase+vector.bits+8+7)/8)
				for bit := 0; bit < vector.bits; bit++ {
					at := phase + bit
					containing[at/8] |= ((wire[bit/8] >> uint(7-bit%8)) & 1) << uint(7-at%8)
				}
				for bit := 0; bit < 8; bit++ {
					at := phase + vector.bits + bit
					containing[at/8] |= ((byte(0xa5) >> uint(7-bit)) & 1) << uint(7-at%8)
				}
				reader := runtime.NewReader(containing)
				if _, e = reader.ReadUint(phase); e != nil {
					t.Fatal(e)
				}
				old, e := reader.PushLimit(vector.bits)
				if e != nil {
					t.Fatal(e)
				}
				bounded, e := descriptor.DecodeFrom(reader)
				if e != nil {
					t.Fatalf("phase %d: %v", phase, e)
				}
				if reader.Position() != phase+vector.bits {
					t.Fatalf("phase %d consumed %d", phase, reader.Position())
				}
				for path, want := range vector.numbers {
					if got := fieldAt(bare(bounded), path).Uint(); got != want {
						t.Errorf("phase %d %s=%d want %d", phase, path, got, want)
					}
				}
				if e = reader.PopLimit(old); e != nil {
					t.Fatal(e)
				}
				sentinel, e := reader.ReadUint(8)
				if e != nil || sentinel != 0xa5 {
					t.Fatalf("phase %d sentinel=%x: %v", phase, sentinel, e)
				}
			}
			for cut := 0; cut < vector.semanticBits; cut++ {
				r := runtime.NewReader(wire)
				if _, e = r.PushLimit(cut); e != nil {
					t.Fatal(e)
				}
				if _, e = descriptor.DecodeFrom(r); e == nil {
					t.Errorf("accepted truncated %d-bit body", cut)
				}
				if r.Position() > cut {
					t.Errorf("crossed %d-bit bound", cut)
				}
			}
			independent, e := descriptor.Decode(wire)
			if e != nil {
				t.Fatal(e)
			}
			replacement := reflect.New(reflect.TypeOf(independent)).Elem()
			replacement.Set(reflect.ValueOf(independent))
			imported := replacement.FieldByName("Value").FieldByName("MPRACHDescription")
			imported.Set(reflect.Zero(imported.Type()))
			replaced, e := descriptor.CanonicalAtLength(replacement.Interface(), len(wire))
			if e != nil {
				t.Fatalf("fresh nested replacement: %v", e)
			}
			if _, e = descriptor.Decode(replaced); e != nil {
				t.Fatalf("fresh nested replacement decode: %v", e)
			}
			// An ordinary field edit retains the imported concrete type and replay state.
			for path, want := range vector.numbers {
				if !strings.HasSuffix(path, "USF") {
					continue
				}
				independent, e := descriptor.Decode(wire)
				if e != nil {
					t.Fatal(e)
				}
				mutable := reflect.New(reflect.TypeOf(independent)).Elem()
				mutable.Set(reflect.ValueOf(independent))
				fieldAt(mutable.FieldByName("Value"), path).SetUint(want ^ 1)
				edited, e := descriptor.Encode(mutable.Interface())
				if e != nil {
					t.Fatal(e)
				}
				if bytes.Equal(edited, wire) {
					t.Fatal("typed edit did not change wire")
				}
				round, e := descriptor.Decode(edited)
				if e != nil || fieldAt(bare(round), path).Uint() != want^1 {
					t.Fatalf("typed edit: %v", e)
				}
				if fieldAt(mutable.FieldByName("Value"), path).OverflowUint(8) {
					continue
				}
				fieldAt(mutable.FieldByName("Value"), path).SetUint(8)
				if _, e = descriptor.CanonicalAtLength(mutable.Interface(), len(wire)); e == nil {
					t.Fatal("accepted out-of-range typed field")
				}
			}
		})
	}
	// Additional spare octets change neither the delegated fields nor the work class.
	for _, n := range []int{16, 64, 128} {
		wire, e := hex.DecodeString(vectors[0].wire)
		if e != nil {
			t.Fatal(e)
		}
		wire = append(wire, make([]byte, n)...)
		allocations := testing.AllocsPerRun(1, func() {
			if _, e := descriptor.Decode(wire); e != nil {
				t.Fatal(e)
			}
		})
		if !raceEnabled && allocations > float64(32*len(wire)+128) {
			t.Fatalf("nonlinear allocation count at %d bytes: %g", len(wire), allocations)
		}
		value, e := descriptor.Decode(wire)
		if e != nil {
			t.Fatal(e)
		}
		replay, e := descriptor.Encode(value)
		if e != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("growing spare %d: %v", n, e)
		}
	}
}

func FuzzMPRACHDescriptionValuePartReplay(f *testing.F) {
	descriptor, err := LookupClause("10.5.2.60", "MPRACH description value part")
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{0x3a})
	f.Add([]byte{0x3b, 0x10})
	f.Add([]byte{0x3b, 0xd2, 0xad, 0x64, 0xad, 0xbf})
	f.Add([]byte{0xd1, 0x41, 0x75})
	f.Add([]byte{0xd1, 0x41, 0x76, 0x21})
	f.Add([]byte{0xd1, 0x41, 0x77, 0xa5, 0x5a, 0xc9, 0x5b, 0x7f})
	f.Add([]byte{0xb5, 0x13, 0x3a})
	f.Add([]byte{0xb5, 0x13, 0x3b, 0x10})
	f.Add([]byte{0xb5, 0x13, 0x3b, 0xd2, 0xad, 0x64, 0xad, 0xbf})
	f.Add([]byte{0xb5, 0x13, 0xd5, 0xd7})
	f.Add([]byte{0xb5, 0x13, 0xd5, 0xd8, 0x87})
	f.Add([]byte{0xb5, 0x13, 0xd5, 0xde, 0x95, 0x6b, 0x25, 0x6d})
	f.Add([]byte{0xa8, 0x72, 0x53, 0xaf})
	f.Add([]byte{0xa8, 0x72, 0x53, 0xb1, 0x0f})
	f.Add([]byte{0xa8, 0x72, 0x53, 0xbd, 0x2a, 0xd6, 0x4a, 0xdb})
	f.Add([]byte{0x9c, 0xb3, 0x40, 0x48, 0xd1, 0x59, 0xd7})
	f.Add([]byte{0x9c, 0xb3, 0x40, 0x48, 0xd1, 0x59, 0xd8, 0x87})
	f.Add([]byte{0x9c, 0xb3, 0x40, 0x48, 0xd1, 0x59, 0xde, 0x95, 0x6b, 0x25, 0x6d})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 512 {
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

// TS 44.018 V19.0.0 §10.5.2.60; the containing maximum is authenticated
// by the bound's generated source citation. An oversized received image
// still decodes and replays; this is an explicit canonical sender limit.
func TestMPRACHDescriptionValuePartContainingMaximum(t *testing.T) {
	d, err := LookupClause("10.5.2.60", "MPRACH description value part")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := hex.DecodeString("3a")
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	out, err := d.CanonicalAtLength(v, 255)
	if err != nil || len(out) != 255 {
		t.Fatalf("at source bound: length=%d error=%v", len(out), err)
	}
	out, err = d.CanonicalAtLength(v, 255+1)
	var extent *runtime.ExtentError
	if out != nil || !errors.As(err, &extent) || extent.Maximum != 255 {
		t.Fatalf("past source bound: %x %v", out, err)
	}
	received := append(append([]byte(nil), wire...), make([]byte, 255+1)...)
	decoded, err := d.Decode(received)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := d.Encode(decoded)
	if err != nil || !bytes.Equal(replay, received) {
		t.Fatalf("tolerant replay %x %v", replay, err)
	}
}
