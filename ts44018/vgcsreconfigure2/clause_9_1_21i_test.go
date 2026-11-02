package vgcsreconfigure2

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

type cellReconfigureVector struct {
	Name, Hex       string
	Bits, Mandatory int
	Numbers         map[string]uint64
	Absent          []string
	Strings         map[string]struct {
		Hex  string
		Bits int
	}
}

// TS 44.018 V19.0.0 §9.1.21i, paragraphs 7677–7689; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const cellReconfigureVectorJSON = `[{"Name":"desc-0-start-0-tail-0","Hex":"1c000000000000000000000000000000000000","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-0-start-0-tail-1","Hex":"1c0000000000000000000000000000000000005300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-0-start-0-tail-2","Hex":"1c000000000000000000000000000000000000535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-0-start-65535-tail-0","Hex":"1c00000000000000000000000000000000ffff","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-0-start-65535-tail-1","Hex":"1c00000000000000000000000000000000ffff5300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-0-start-65535-tail-2","Hex":"1c00000000000000000000000000000000ffff535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"00000000000000000000000000000000","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-0-tail-0","Hex":"1c0123456789abcdef0123456789abcdef0000","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-0-tail-1","Hex":"1c0123456789abcdef0123456789abcdef00005300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-0-tail-2","Hex":"1c0123456789abcdef0123456789abcdef0000535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-65535-tail-0","Hex":"1c0123456789abcdef0123456789abcdefffff","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-65535-tail-1","Hex":"1c0123456789abcdef0123456789abcdefffff5300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-123456789abcdef0123456789abcdef-start-65535-tail-2","Hex":"1c0123456789abcdef0123456789abcdefffff535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"0123456789abcdef0123456789abcdef","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-0-tail-0","Hex":"1cffffffffffffffffffffffffffffffff0000","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-0-tail-1","Hex":"1cffffffffffffffffffffffffffffffff00005300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-0-tail-2","Hex":"1cffffffffffffffffffffffffffffffff0000535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":0},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-65535-tail-0","Hex":"1cffffffffffffffffffffffffffffffffffff","Bits":152,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-65535-tail-1","Hex":"1cffffffffffffffffffffffffffffffffffff5300","Bits":161,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}},{"Name":"desc-ffffffffffffffffffffffffffffffff-start-65535-tail-2","Hex":"1cffffffffffffffffffffffffffffffffffff535300","Bits":169,"Mandatory":152,"Numbers":{"StartingTime":65535},"Absent":[],"Strings":{"NewCellChannelDescription":{"Hex":"ffffffffffffffffffffffffffffffff","Bits":128}}}]`

func cellReconfigureVectors(t testing.TB) []cellReconfigureVector {
	t.Helper()
	var rows []cellReconfigureVector
	if err := json.Unmarshal([]byte(cellReconfigureVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func cellReconfigureFields(value any) map[string][]reflect.Value {
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

func cellReconfigureCheckFields(t *testing.T, v VBSVGCSRECONFIGURE2, row cellReconfigureVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 7 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := cellReconfigureFields(v)
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
		if !ok || b.BitLength != want.Bits || hex.EncodeToString(b.Bytes) != want.Hex {
			t.Fatalf("%s: %#v want %#v", name, b, want)
		}
	}
}

func cellReconfigureEqualBits(a, b []byte, n int) bool {
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

func TestVBSVGCSRECONFIGURE2SourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range cellReconfigureVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[VBSVGCSRECONFIGURE2])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			cellReconfigureCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeVBSVGCSRECONFIGURE2To(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !cellReconfigureEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeVBSVGCSRECONFIGURE2To(canonicalWriter, fresh); err != nil {
				t.Fatal(err)
			}
			canonical, err := canonicalWriter.FinishBits(runtime.BitString{})
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
			typed := reread.(runtime.Decoded[VBSVGCSRECONFIGURE2])
			cellReconfigureCheckFields(t, typed.Value, row)
			if !runtime.SemanticallyEqual(typed.Value, decoded.Value) {
				t.Fatal("canonical changed typed fields")
			}

			// TS 44.018 V19.0.0 §9.1.21i p7683: ignored received bits
			// have no sender representation. The named byte API must fill
			// its requested extent or return the exact typed refusal.
			requested := (row.Bits + 7) / 8
			out, canonicalErr := EncodeVBSVGCSRECONFIGURE2CanonicalAtLength(decoded.Value, requested)
			if decoded.Value.Alternatives.NoString != nil {
				var extent *runtime.ExtentError
				natural := (canonicalBits + 7) / 8
				if len(out) != 0 || !errors.As(canonicalErr, &extent) || extent.Required || extent.Actual != natural || extent.Minimum != requested || extent.Maximum != requested {
					t.Fatalf("unfilled target: bytes=%x error=%v extent=%+v", out, canonicalErr, extent)
				}
				shortest, err := EncodeVBSVGCSRECONFIGURE2CanonicalAtLength(decoded.Value, natural)
				if err != nil || len(shortest) != natural {
					t.Fatalf("natural canonical %x: %v", shortest, err)
				}
				again, err := DecodeVBSVGCSRECONFIGURE2(shortest)
				if err != nil || !runtime.SemanticallyEqual(decoded.Value, again.Value) {
					t.Fatalf("natural canonical typed value: %v", err)
				}
			} else if canonicalErr != nil || len(out) != requested {
				t.Fatalf("complete null-tail target: %x: %v", out, canonicalErr)
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
				if err := EncodeVBSVGCSRECONFIGURE2To(positioned, fresh); err != nil {
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
				cellReconfigureCheckFields(t, reread.(runtime.Decoded[VBSVGCSRECONFIGURE2]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeVBSVGCSRECONFIGURE2To(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !cellReconfigureEqualBits(edit.Bytes, want, row.Bits) {
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
				cellReconfigureCheckFields(t, reread.(runtime.Decoded[VBSVGCSRECONFIGURE2]).Value, row)
				stable, err := d.CanonicalAtLength(reread, len(wire))
				if err != nil || !bytes.Equal(stable, canonicalBytes) {
					t.Fatalf("unstable canonical %x: %v", stable, err)
				}
			}
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 7 {
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

func TestVBSVGCSRECONFIGURE2InputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil {
		t.Fatal(err)
	}
	rows := cellReconfigureVectors(t)
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

func FuzzVBSVGCSRECONFIGURE2(f *testing.F) {
	d, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range cellReconfigureVectors(f) {
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
		decoded, ok := raw.(runtime.Decoded[VBSVGCSRECONFIGURE2])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeVBSVGCSRECONFIGURE2To(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !cellReconfigureEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestVBSVGCSRECONFIGURE2FieldBounds(t *testing.T) {
	row := cellReconfigureVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[VBSVGCSRECONFIGURE2]).Value
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
		if err := EncodeVBSVGCSRECONFIGURE2To(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}
