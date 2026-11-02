package vgcsreconfigure

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type reconfigureVector struct {
	Name, Hex       string
	Bits, Mandatory int
	Numbers         map[string]uint64
	Absent          []string
	Strings         map[string]string
}

// TS 44.018 V19.0.0 §9.1.21h, paragraphs 7638-7667; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const reconfigureVectorJSON = `[{"Name":"form-0-start-0-segment-0-tail-0","Hex":"18123456000000","Bits":50,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0},"Absent":[],"Strings":{}},{"Name":"form-0-start-0-segment-0-tail-1","Hex":"18123456000014d4d4d4c0","Bits":83,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0},"Absent":[],"Strings":{}},{"Name":"form-0-start-0-segment-1-tail-0","Hex":"18123456000040","Bits":50,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1},"Absent":[],"Strings":{}},{"Name":"form-0-start-0-segment-1-tail-1","Hex":"18123456000054d4d4d4c0","Bits":83,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1},"Absent":[],"Strings":{}},{"Name":"form-0-start-65535-segment-0-tail-0","Hex":"181234567fff80","Bits":50,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0},"Absent":[],"Strings":{}},{"Name":"form-0-start-65535-segment-0-tail-1","Hex":"181234567fff94d4d4d4c0","Bits":83,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0},"Absent":[],"Strings":{}},{"Name":"form-0-start-65535-segment-1-tail-0","Hex":"181234567fffc0","Bits":50,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1},"Absent":[],"Strings":{}},{"Name":"form-0-start-65535-segment-1-tail-1","Hex":"181234567fffd4d4d4d4c0","Bits":83,"Mandatory":50,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1},"Absent":[],"Strings":{}},{"Name":"form-1-start-0-segment-0-tail-0","Hex":"181234568069400000","Bits":67,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-0-segment-0-tail-1","Hex":"18123456806940000a6a6a6a60","Bits":100,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-0-segment-1-tail-0","Hex":"181234568069400020","Bits":67,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-0-segment-1-tail-1","Hex":"18123456806940002a6a6a6a60","Bits":100,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-65535-segment-0-tail-0","Hex":"1812345680697fffc0","Bits":67,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-65535-segment-0-tail-1","Hex":"1812345680697fffca6a6a6a60","Bits":100,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-65535-segment-1-tail-0","Hex":"1812345680697fffe0","Bits":67,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-1-start-65535-segment-1-tail-1","Hex":"1812345680697fffea6a6a6a60","Bits":100,"Mandatory":67,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"Length":1},"Absent":[],"Strings":{"Value":"a5"}},{"Name":"form-2-start-0-segment-0-tail-0","Hex":"18123456822969696969696969400000","Bits":123,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-0-segment-0-tail-1","Hex":"1812345682296969696969696940000a6a6a6a60","Bits":156,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-0-segment-1-tail-0","Hex":"18123456822969696969696969400020","Bits":123,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-0-segment-1-tail-1","Hex":"1812345682296969696969696940002a6a6a6a60","Bits":156,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-65535-segment-0-tail-0","Hex":"181234568229696969696969697fffc0","Bits":123,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-65535-segment-0-tail-1","Hex":"181234568229696969696969697fffca6a6a6a60","Bits":156,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-65535-segment-1-tail-0","Hex":"181234568229696969696969697fffe0","Bits":123,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-2-start-65535-segment-1-tail-1","Hex":"181234568229696969696969697fffea6a6a6a60","Bits":156,"Mandatory":123,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"Length":8},"Absent":[],"Strings":{"Value":"a5a5a5a5a5a5a5a5"}},{"Name":"form-3-start-0-segment-0-tail-0","Hex":"18123456c048d159e26af37bc00000","Bits":115,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-0-segment-0-tail-1","Hex":"18123456c048d159e26af37bc0000a6a6a6a60","Bits":148,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":0,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-0-segment-1-tail-0","Hex":"18123456c048d159e26af37bc00020","Bits":115,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-0-segment-1-tail-1","Hex":"18123456c048d159e26af37bc0002a6a6a6a60","Bits":148,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":0,"AdditionalSegment":1,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-65535-segment-0-tail-0","Hex":"18123456c048d159e26af37bffffc0","Bits":115,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-65535-segment-0-tail-1","Hex":"18123456c048d159e26af37bffffca6a6a6a60","Bits":148,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":0,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-65535-segment-1-tail-0","Hex":"18123456c048d159e26af37bffffe0","Bits":115,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}},{"Name":"form-3-start-65535-segment-1-tail-1","Hex":"18123456c048d159e26af37bffffea6a6a6a60","Bits":148,"Mandatory":115,"Numbers":{"ChannelDescription":1193046,"StartingTime":65535,"AdditionalSegment":1,"FrequencyShortList":81985529216486895},"Absent":[],"Strings":{}}]`

func reconfigureVectors(t testing.TB) []reconfigureVector {
	t.Helper()
	var rows []reconfigureVector
	if err := json.Unmarshal([]byte(reconfigureVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func reconfigureFields(value any) map[string][]reflect.Value {
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

func reconfigureCheckFields(t *testing.T, v VBSVGCSRECONFIGURE, row reconfigureVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 6 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := reconfigureFields(v)
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

func reconfigureEqualBits(a, b []byte, n int) bool {
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

func TestVBSVGCSRECONFIGURESourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range reconfigureVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[VBSVGCSRECONFIGURE])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			reconfigureCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeVBSVGCSRECONFIGURETo(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !reconfigureEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeVBSVGCSRECONFIGURETo(canonicalWriter, fresh); err != nil {
				t.Fatal(err)
			}
			canonical, err := canonicalWriter.FinishBits(runtime.BitString{})
			// TS 44.018 V19.0.0 §9.1.21h p7645: the receiver-only
			// no-string suffix carries no semantic bits. A fresh writer emits
			// its zero selector, without fabricating sender padding.
			canonicalBits := row.Mandatory
			if decoded.Value.Alternatives.NoString != nil {
				canonicalBits++
			}
			if err != nil || canonical.BitLength != canonicalBits {
				t.Fatalf("canonical %#v: %v", canonical, err)
			}
			reread, err := d.DecodeBits(canonical.Bytes, canonical.BitLength)
			if err != nil {
				t.Fatal(err)
			}
			typed := reread.(runtime.Decoded[VBSVGCSRECONFIGURE])
			reconfigureCheckFields(t, typed.Value, row)
			if !runtime.SemanticallyEqual(typed.Value, decoded.Value) {
				t.Fatal("canonical changed typed fields")
			}

			if _, err := EncodeVBSVGCSRECONFIGURECanonical(decoded.Value); err == nil {
				t.Fatal("canonical requires the caller extent")
			} else {
				var extent *runtime.ExtentError
				if !errors.As(err, &extent) || !extent.Required {
					t.Fatalf("default canonical: %v", err)
				}
			}
			requested := (row.Bits + 7) / 8
			out, canonicalErr := EncodeVBSVGCSRECONFIGURECanonicalAtLength(decoded.Value, requested)
			if decoded.Value.Alternatives.NoString != nil {
				var extent *runtime.ExtentError
				natural := (canonicalBits + 7) / 8
				if len(out) != 0 || !errors.As(canonicalErr, &extent) || extent.Required || extent.Actual != natural || extent.Minimum != requested || extent.Maximum != requested {
					t.Fatalf("unfilled target: bytes=%x error=%v extent=%+v", out, canonicalErr, extent)
				}
				// The named API succeeds only when its requested byte extent
				// can actually be filled and the typed fields survive decoding.
				shortest, err := EncodeVBSVGCSRECONFIGURECanonicalAtLength(decoded.Value, natural)
				if err != nil || len(shortest) != natural {
					t.Fatalf("natural canonical %x: %v", shortest, err)
				}
				again, err := DecodeVBSVGCSRECONFIGURE(shortest)
				if err != nil || !runtime.SemanticallyEqual(decoded.Value, again.Value) {
					t.Fatalf("natural canonical typed value: %v", err)
				}
			} else {
				var bound *runtime.BoundError
				if len(out) != 0 || !errors.As(canonicalErr, &bound) || bound.Kind != runtime.CanonicalTarget {
					t.Fatalf("non-octet null target %x: %v", out, canonicalErr)
				}
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
				if err := EncodeVBSVGCSRECONFIGURETo(positioned, fresh); err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				at, err := positioned.FinishBits(runtime.BitString{})
				if err != nil || at.BitLength != canonicalBits+phase {
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
				reconfigureCheckFields(t, reread.(runtime.Decoded[VBSVGCSRECONFIGURE]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeVBSVGCSRECONFIGURETo(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !reconfigureEqualBits(edit.Bytes, want, row.Bits) {
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
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 6 {
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

func TestVBSVGCSRECONFIGUREInputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		t.Fatal(err)
	}
	rows := reconfigureVectors(t)
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

func FuzzVBSVGCSRECONFIGURE(f *testing.F) {
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range reconfigureVectors(f) {
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
		decoded, ok := raw.(runtime.Decoded[VBSVGCSRECONFIGURE])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeVBSVGCSRECONFIGURETo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !reconfigureEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestVBSVGCSRECONFIGUREFieldBounds(t *testing.T) {
	row := reconfigureVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[VBSVGCSRECONFIGURE]).Value
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
		if err := EncodeVBSVGCSRECONFIGURETo(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}

// TS 44.018 V19.0.0 §10.5.2.21 p16717 and TS 24.008 V20.1.0
// §10.5.1.4 p20956 require at least one LV value octet in these messages.
func TestVBSVGCSRECONFIGUREZeroLV(t *testing.T) {
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Hex  string
		Bits int
	}
	if err := json.Unmarshal([]byte(`[{"Hex":"1812345680000000","Bits":59},{"Hex":"181234568000000a6a6a6a60","Bits":92},{"Hex":"1812345680000020","Bits":59},{"Hex":"181234568000002a6a6a6a60","Bits":92},{"Hex":"18123456803fffc0","Bits":59},{"Hex":"18123456803fffca6a6a6a60","Bits":92},{"Hex":"18123456803fffe0","Bits":59},{"Hex":"18123456803fffea6a6a6a60","Bits":92}]`), &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.DecodeBits(wire, row.Bits); err == nil {
			t.Fatalf("accepted zero-length LV %x", wire)
		}
	}
}

func FuzzNewGroupChannelDescription(f *testing.F) {
	d, err := LookupClause("9.1.21h", "new Group Channel Description")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range reconfigureVectors(f) {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		raw, err := d.Decode(data)
		if err != nil {
			return
		}
		if _, ok := raw.(runtime.Decoded[NewGroupChannelDescription]); !ok {
			t.Fatalf("typed helper %T", raw)
		}
		out, err := d.Encode(raw)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("helper replay %x: %v", out, err)
		}
	})
}
