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

// TS 44.018 V19.0.0 §10.5.2.58, paragraphs 21213–21215, 21218, imports the
// typed TS 44.060 component(s) through authenticated source bindings. All vectors are synthetic.
func TestMBMSPTMChannelDescriptionValuePartPublication(t *testing.T) {
	descriptor, err := LookupClause("10.5.2.58", "MBMS p-t-m Channel Description value part")
	if err != nil {
		t.Fatal(err)
	}
	vectors := []struct {
		name, wire, canonical string
		bits, semanticBits    int
		numbers               map[string]uint64
	}{
		{name: "absent", wire: "52ff", canonical: "5280", bits: 16, semanticBits: 9, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165}},
		{name: "arfcn", wire: "d141a5", canonical: "d141a5", bits: 24, semanticBits: 24, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165, "MBMSPTMChannelDescription.FrequencyParameters.TSC": 5}},
		{name: "indirect", wire: "b51352ff", canonical: "b5135280", bits: 32, semanticBits: 25, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165, "MBMSPTMChannelDescription.FrequencyParameters.TSC": 3}},
		{name: "indirect-marks", wire: "b513d697", canonical: "b513d694", bits: 32, semanticBits: 30, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165, "MBMSPTMChannelDescription.FrequencyParameters.TSC": 3}},
		{name: "direct1", wire: "a872552f", canonical: "a8725528", bits: 32, semanticBits: 29, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165, "MBMSPTMChannelDescription.FrequencyParameters.TSC": 2}},
		{name: "direct2", wire: "9cb34048d15a97", canonical: "9cb34048d15a94", bits: 56, semanticBits: 54, numbers: map[string]uint64{"MBMSPTMChannelDescription.DLTIMESLOTALLOCATION": 165, "MBMSPTMChannelDescription.FrequencyParameters.TSC": 1}},
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
			if value.Type().Name() != "MBMSPTMChannelDescriptionValuePart" {
				t.Fatalf("untyped root %T", decoded)
			}
			if f, ok := value.Type().FieldByName("MBMSPTMChannelDescription"); !ok || f.Type.PkgPath() != "github.com/gomaja/go-csn1/ts44060/ies" {
				t.Fatalf("lost delegated type for MBMSPTMChannelDescription")
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
			imported := replacement.FieldByName("Value").FieldByName("MBMSPTMChannelDescription")
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
				if !strings.HasSuffix(path, "DLTIMESLOTALLOCATION") {
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
				if fieldAt(mutable.FieldByName("Value"), path).OverflowUint(256) {
					continue
				}
				fieldAt(mutable.FieldByName("Value"), path).SetUint(256)
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

func FuzzMBMSPTMChannelDescriptionValuePartReplay(f *testing.F) {
	descriptor, err := LookupClause("10.5.2.58", "MBMS p-t-m Channel Description value part")
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{0x52, 0xff})
	f.Add([]byte{0xd1, 0x41, 0xa5})
	f.Add([]byte{0xb5, 0x13, 0x52, 0xff})
	f.Add([]byte{0xb5, 0x13, 0xd6, 0x97})
	f.Add([]byte{0xa8, 0x72, 0x55, 0x2f})
	f.Add([]byte{0x9c, 0xb3, 0x40, 0x48, 0xd1, 0x5a, 0x97})
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

// TS 44.018 V19.0.0 §10.5.2.58; the containing maximum is authenticated
// by the bound's generated source citation. An oversized received image
// still decodes and replays; this is an explicit canonical sender limit.
func TestMBMSPTMChannelDescriptionValuePartContainingMaximum(t *testing.T) {
	d, err := LookupClause("10.5.2.58", "MBMS p-t-m Channel Description value part")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := hex.DecodeString("52ff")
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
