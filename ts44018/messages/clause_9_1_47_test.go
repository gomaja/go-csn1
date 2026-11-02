package messages

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type uplinkFreeVector struct {
	Name, Hex       string
	Bits, Mandatory int
	Numbers         map[string]uint64
	Absent          []string
	Strings         map[string]string
}

// TS 44.018 V19.0.0 §9.1.47, paragraphs 10468-10482; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const uplinkFreeVectorJSON = `[{"Name":"access-0-identity-None-release-L","Hex":"082b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":0},"Absent":["UplinkIdentityCode","EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-None-release-H0","Hex":"080b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":0},"Absent":["UplinkIdentityCode","EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-None-release-H10","Hex":"08132b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":0,"EmergencyInd":0},"Absent":["UplinkIdentityCode"],"Strings":{}},{"Name":"access-0-identity-None-release-H11","Hex":"081b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":0,"EmergencyInd":1},"Absent":["UplinkIdentityCode"],"Strings":{}},{"Name":"access-0-identity-0-release-L","Hex":"08402b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":0},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-0-release-H0","Hex":"0840ab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":0},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-0-release-H10","Hex":"0840cb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":0,"EmergencyInd":0},"Absent":[],"Strings":{}},{"Name":"access-0-identity-0-release-H11","Hex":"0840eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":0,"EmergencyInd":1},"Absent":[],"Strings":{}},{"Name":"access-0-identity-63-release-L","Hex":"087f2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":63},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-63-release-H0","Hex":"087fab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":63},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-0-identity-63-release-H10","Hex":"087fcb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":63,"EmergencyInd":0},"Absent":[],"Strings":{}},{"Name":"access-0-identity-63-release-H11","Hex":"087feb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":0,"UplinkIdentityCode":63,"EmergencyInd":1},"Absent":[],"Strings":{}},{"Name":"access-1-identity-None-release-L","Hex":"08ab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":1},"Absent":["UplinkIdentityCode","EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-None-release-H0","Hex":"088b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":1},"Absent":["UplinkIdentityCode","EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-None-release-H10","Hex":"08932b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":1,"EmergencyInd":0},"Absent":["UplinkIdentityCode"],"Strings":{}},{"Name":"access-1-identity-None-release-H11","Hex":"089b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":10,"Numbers":{"UplinkAccessRequest":1,"EmergencyInd":1},"Absent":["UplinkIdentityCode"],"Strings":{}},{"Name":"access-1-identity-0-release-L","Hex":"08c02b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":0},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-0-release-H0","Hex":"08c0ab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":0},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-0-release-H10","Hex":"08c0cb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":0,"EmergencyInd":0},"Absent":[],"Strings":{}},{"Name":"access-1-identity-0-release-H11","Hex":"08c0eb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":0,"EmergencyInd":1},"Absent":[],"Strings":{}},{"Name":"access-1-identity-63-release-L","Hex":"08ff2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":63},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-63-release-H0","Hex":"08ffab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":63},"Absent":["EmergencyInd"],"Strings":{}},{"Name":"access-1-identity-63-release-H10","Hex":"08ffcb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":63,"EmergencyInd":0},"Absent":[],"Strings":{}},{"Name":"access-1-identity-63-release-H11","Hex":"08ffeb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":16,"Numbers":{"UplinkAccessRequest":1,"UplinkIdentityCode":63,"EmergencyInd":1},"Absent":[],"Strings":{}}]`

func uplinkFreeVectors(t testing.TB) []uplinkFreeVector {
	t.Helper()
	var rows []uplinkFreeVector
	if err := json.Unmarshal([]byte(uplinkFreeVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func uplinkFreeFields(value any) map[string][]reflect.Value {
	fields := map[string][]reflect.Value{}
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, name string) {
		if !v.IsValid() {
			return
		}
		for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if v.IsNil() {
				return
			}
			v = v.Elem()
		}
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() {
			return
		}
		if v.Type() == reflect.TypeFor[runtime.BitString]() {
			fields[name] = append(fields[name], v)
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i), v.Type().Field(i).Name)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), name)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fields[name] = append(fields[name], v)
		}
	}
	walk(reflect.ValueOf(value), "")
	return fields
}

func uplinkFreeCheckFields(t *testing.T, v UPLINKFREE, row uplinkFreeVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 2 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := uplinkFreeFields(v)
	for name, want := range row.Numbers {
		got := fields[name]
		if len(got) != 1 || got[0].Kind() == reflect.Struct || got[0].Uint() != want {
			t.Fatalf("%s: got %v want %d", name, got, want)
		}
	}
	for _, name := range row.Absent {
		if len(fields[name]) != 0 {
			t.Fatalf("%s unexpectedly present", name)
		}
	}
	for name, want := range row.Strings {
		got := fields[name]
		if len(got) != 1 {
			t.Fatalf("%s: %d bit strings", name, len(got))
		}
		b, ok := got[0].Interface().(runtime.BitString)
		if !ok || b.BitLength != len(want)*4 || hex.EncodeToString(b.Bytes) != want {
			t.Fatalf("%s: %#v want %s", name, b, want)
		}
	}
}

func uplinkFreeEqualBits(a, b []byte, n int) bool {
	if len(a)*8 < n || len(b)*8 < n {
		return false
	}
	for i := 0; i < n; i++ {
		if (a[i/8]>>uint(7-i%8))&1 != (b[i/8]>>uint(7-i%8))&1 {
			return false
		}
	}
	return true
}

func TestUPLINKFREESourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.47", "UPLINK FREE")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range uplinkFreeVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[UPLINKFREE])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			uplinkFreeCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeUPLINKFREETo(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !uplinkFreeEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeUPLINKFREETo(canonicalWriter, fresh); err != nil {
				t.Fatal(err)
			}
			canonical, err := canonicalWriter.FinishBits(runtime.BitString{})
			if err != nil || canonical.BitLength != row.Bits {
				t.Fatalf("canonical %#v: %v", canonical, err)
			}
			reread, err := d.DecodeBits(canonical.Bytes, canonical.BitLength)
			if err != nil {
				t.Fatal(err)
			}
			typed := reread.(runtime.Decoded[UPLINKFREE])
			uplinkFreeCheckFields(t, typed.Value, row)
			if !runtime.SemanticallyEqual(typed.Value, decoded.Value) {
				t.Fatal("canonical changed typed fields")
			}

			for phase := 0; phase < 8; phase++ {
				fresh := runtime.CanonicalTo(decoded.Value, row.Bits+phase)
				positioned := runtime.NewWriter()
				if err := positioned.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				positioned.WithWire(fresh.Wire)
				if _, err := positioned.PushLimit(row.Bits); err != nil {
					t.Fatal(err)
				}
				if err := EncodeUPLINKFREETo(positioned, fresh); err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				at, err := positioned.FinishBits(runtime.BitString{})
				if err != nil || at.BitLength != row.Bits+phase {
					t.Fatalf("phase %d extent %#v: %v", phase, at, err)
				}
				pr, err := runtime.NewChoiceReaderBits(at.Bytes, at.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := pr.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				reread, err := d.DecodeFrom(pr)
				if err != nil {
					t.Fatalf("phase %d decode: %v", phase, err)
				}
				uplinkFreeCheckFields(t, reread.(runtime.Decoded[UPLINKFREE]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeUPLINKFREETo(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !uplinkFreeEqualBits(edit.Bytes, want, row.Bits) {
				t.Fatalf("edited replay %#v: %v", edit, err)
			}
			for cut := 0; cut < row.Mandatory; cut++ {
				r, err := runtime.NewChoiceReaderBits(wire, cut, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := d.DecodeFrom(r); err == nil {
					t.Fatalf("accepted mandatory truncation %d", cut)
				}
				if r.Position() > cut {
					t.Fatalf("crossed %d-bit bound", cut)
				}
			}
			containing := make([]byte, (row.Bits+8+7)/8)
			for bit := 0; bit < row.Bits+8; bit++ {
				value := byte(0)
				if bit < row.Bits {
					value = wire[bit/8] >> uint(7-bit%8) & 1
				} else {
					value = 0xd7 >> uint(7-(bit-row.Bits)) & 1
				}
				containing[bit/8] |= value << uint(7-bit%8)
			}
			r, err := runtime.NewChoiceReaderBits(containing, row.Bits+8, nil)
			if err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.DecodeFrom(r); err != nil {
				t.Fatal(err)
			}
			if r.Position() != row.Bits {
				t.Fatal("wrong bounded consumption")
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(8)
			if err != nil || sentinel != 0xd7 {
				t.Fatalf("sentinel %x: %v", sentinel, err)
			}
			if row.Bits%8 == 0 {
				canonicalBytes, err := d.CanonicalAtLength(decoded, len(wire))
				if err != nil {
					t.Fatal(err)
				}
				reread, err := d.Decode(canonicalBytes)
				if err != nil {
					t.Fatal(err)
				}
				uplinkFreeCheckFields(t, reread.(runtime.Decoded[UPLINKFREE]).Value, row)
				stable, err := d.CanonicalAtLength(reread, len(wire))
				if err != nil || !bytes.Equal(stable, canonicalBytes) {
					t.Fatalf("unstable canonical %x: %v", stable, err)
				}
			}
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 2 {
					continue
				}
				bad := bytes.Clone(wire)
				bad[0] = (bad[0] & 0x83) | (candidate << 2)
				if _, err := d.DecodeBits(bad, row.Bits); err == nil {
					t.Fatalf("accepted message type %d", candidate)
				}
			}
		})
	}
}

func TestUPLINKFREEInputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.47", "UPLINK FREE")
	if err != nil {
		t.Fatal(err)
	}
	rows := uplinkFreeVectors(t)
	seed, err := hex.DecodeString(rows[len(rows)-1].Hex)
	if err != nil {
		t.Fatal(err)
	}
	// This is a decoder-storage test with caller-supplied bounds; these larger
	// buffers do not assert a radio carrier's permissible wire extent.
	for _, n := range []int{64, 256, 1024, 4096, 8192} {
		wire := append(bytes.Clone(seed), bytes.Repeat([]byte{0x2b}, n)...)
		beforeGC := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		for i := 0; i < 4; i++ {
			if _, err := d.Decode(wire); err != nil {
				debug.SetGCPercent(beforeGC)
				t.Fatal(err)
			}
		}
		goruntime.ReadMemStats(&after)
		debug.SetGCPercent(beforeGC)
		allocated := (after.TotalAlloc - before.TotalAlloc) / 4
		if !raceEnabled && allocated > uint64(len(wire))*256+262144 {
			t.Fatalf("storage grew beyond linear bound: %d input bytes, %d allocated bytes", len(wire), allocated)
		}
	}
}

func FuzzUPLINKFREE(f *testing.F) {
	d, err := LookupClause("9.1.47", "UPLINK FREE")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range uplinkFreeVectors(f) {
		b, err := hex.DecodeString(row.Hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b, row.Bits)
	}
	f.Fuzz(func(t *testing.T, data []byte, bits int) {
		if len(data) > 4096 || bits < 0 || bits > len(data)*8 {
			return
		}
		raw, err := d.DecodeBits(data, bits)
		if err != nil {
			return
		}
		decoded, ok := raw.(runtime.Decoded[UPLINKFREE])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeUPLINKFREETo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !uplinkFreeEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestUPLINKFREEFieldBounds(t *testing.T) {
	row := uplinkFreeVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.47", "UPLINK FREE")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[UPLINKFREE]).Value
	for _, field := range []string{"RRShortPD", "ShortLayer2Header", "MessageType"} {
		v := runtime.CanonicalTo(original, row.Bits)
		switch field {
		case "RRShortPD":
			v.RRShortPD = 2
		case "ShortLayer2Header":
			v.ShortLayer2Header = 4
		case "MessageType":
			v.MessageType = 31
		}
		if err := EncodeUPLINKFREETo(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}
