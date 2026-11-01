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

type smsInformationVector struct {
	Name, Hex                string
	Bits, Mandatory, Content int
	Numbers                  map[string]uint64
	Absent                   []string
	Strings                  map[string]struct {
		Hex  string
		Bits int
	}
}

// TS 44.018 V19.0.0 §9.1.49b, paragraphs 10596–10621; message assignment:
// §10.4 table 10.4.2. These synthetic values have an independent field reading.
const smsInformationVectorJSON = `[{"Name":"nonsegmented-0","Hex":"24002b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":21,"Numbers":{"SMSReferenceNumber":5,"Length":0},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"","Bits":0}},"Content":21},{"Name":"nonsegmented-1","Hex":"2400af2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":22,"Numbers":{"SMSReferenceNumber":5,"Length":1},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"80","Bits":1}},"Content":22},{"Name":"nonsegmented-7","Hex":"2403ad5b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":28,"Numbers":{"SMSReferenceNumber":5,"Length":7},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aa","Bits":7}},"Content":28},{"Name":"nonsegmented-8","Hex":"24042d532b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":29,"Numbers":{"SMSReferenceNumber":5,"Length":8},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aa","Bits":8}},"Content":29},{"Name":"nonsegmented-31","Hex":"240fad5555555b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":52,"Numbers":{"SMSReferenceNumber":5,"Length":31},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaa","Bits":31}},"Content":52},{"Name":"nonsegmented-151","Hex":"244bad5555555555555555555555555555555555555b2b","Bits":184,"Mandatory":172,"Numbers":{"SMSReferenceNumber":5,"Length":151},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":151}},"Content":172},{"Name":"nonsegmented-254","Hex":"247f2d5555555555555555555555555555555555555555555555555555555555555540","Bits":275,"Mandatory":275,"Numbers":{"SMSReferenceNumber":5,"Length":254},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa8","Bits":254}},"Content":275},{"Name":"nonsegmented-255","Hex":"247fad5555555555555555555555555555555555555555555555555555555555555550","Bits":276,"Mandatory":276,"Numbers":{"SMSReferenceNumber":5,"Length":255},"Absent":["SegmentNumber"],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":255}},"Content":276},{"Name":"final-segment-0-length-0","Hex":"24b8402b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":26,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":0},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"","Bits":0}},"Content":26},{"Name":"final-segment-0-length-1","Hex":"24b8406b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":27,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":1},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"80","Bits":1}},"Content":27},{"Name":"final-segment-0-length-8","Hex":"24b8422aab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":34,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":8},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aa","Bits":8}},"Content":34},{"Name":"final-segment-0-length-31","Hex":"24b847eaaaaaaaab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":57,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":31},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaa","Bits":31}},"Content":57},{"Name":"final-segment-0-length-151","Hex":"24b865eaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab","Bits":184,"Mandatory":177,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":151},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":151}},"Content":177},{"Name":"final-segment-0-length-254","Hex":"24b87faaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":280,"Mandatory":280,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":254},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa8","Bits":254}},"Content":280},{"Name":"final-segment-0-length-255","Hex":"24b87feaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa80","Bits":281,"Mandatory":281,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":0,"Length":255},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":255}},"Content":281},{"Name":"final-segment-15-length-0","Hex":"24bfc02b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":26,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":0},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"","Bits":0}},"Content":26},{"Name":"final-segment-15-length-1","Hex":"24bfc06b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":27,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":1},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"80","Bits":1}},"Content":27},{"Name":"final-segment-15-length-8","Hex":"24bfc22aab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":34,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":8},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aa","Bits":8}},"Content":34},{"Name":"final-segment-15-length-31","Hex":"24bfc7eaaaaaaaab2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b","Bits":184,"Mandatory":57,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":31},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaa","Bits":31}},"Content":57},{"Name":"final-segment-15-length-151","Hex":"24bfe5eaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab","Bits":184,"Mandatory":177,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":151},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":151}},"Content":177},{"Name":"final-segment-15-length-254","Hex":"24bfffaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":280,"Mandatory":280,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":254},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa8","Bits":254}},"Content":280},{"Name":"final-segment-15-length-255","Hex":"24bfffeaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa80","Bits":281,"Mandatory":281,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":15,"Length":255},"Absent":[],"Strings":{"SMSFinalContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":255}},"Content":281},{"Name":"nonfinal-segment-fills-block","Hex":"24b9aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Bits":184,"Mandatory":18,"Numbers":{"SMSReferenceNumber":7,"SegmentNumber":3},"Absent":[],"Strings":{"SMSContent":{"Hex":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa8","Bits":166}},"Content":184}]`

func smsInformationVectors(t testing.TB) []smsInformationVector {
	t.Helper()
	var rows []smsInformationVector
	if err := json.Unmarshal([]byte(smsInformationVectorJSON), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

func smsInformationFields(value any) map[string][]reflect.Value {
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

func smsInformationCheckFields(t *testing.T, v VGCSSMSINFORMATION, row smsInformationVector) {
	t.Helper()
	if v.RRShortPD != 0 || v.MessageType != 9 || v.ShortLayer2Header != 0 {
		t.Fatalf("header: %#v", v)
	}
	fields := smsInformationFields(v)
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
		if name == "SMSContent" {
			if len(got) != want.Bits {
				t.Fatalf("%s: %d bits, want %d", name, len(got), want.Bits)
			}
			packed, err := hex.DecodeString(want.Hex)
			if err != nil {
				t.Fatal(err)
			}
			for i, bit := range got {
				if bit.Uint() != uint64(packed[i/8]>>uint(7-i%8)&1) {
					t.Fatalf("%s bit %d", name, i)
				}
			}
			continue
		}
		if len(got) != 1 {
			t.Fatalf("%s: %d bit strings", name, len(got))
		}
		b, ok := got[0].Interface().(runtime.BitString)
		if !ok || b.BitLength != want.Bits || hex.EncodeToString(b.Bytes) != want.Hex {
			t.Fatalf("%s: %#v want %#v", name, b, want)
		}
	}
}

func smsInformationEqualBits(a, b []byte, n int) bool {
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

func TestVGCSSMSINFORMATIONSourceVectors(t *testing.T) {
	d, err := LookupClause("9.1.49b", "VGCS SMS INFORMATION")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range smsInformationVectors(t) {
		t.Run(row.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(row.Hex)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := d.DecodeBits(wire, row.Bits)
			if err != nil {
				t.Fatal(err)
			}
			decoded, ok := raw.(runtime.Decoded[VGCSSMSINFORMATION])
			if !ok {
				t.Fatalf("typed result %T", raw)
			}
			if decoded.BitsConsumed != row.Bits || decoded.Tail.BitLength != 0 {
				t.Fatalf("consumed %d of %d; tail %d", decoded.BitsConsumed, row.Bits, decoded.Tail.BitLength)
			}
			smsInformationCheckFields(t, decoded.Value, row)
			w := runtime.NewWriter()
			if err := EncodeVGCSSMSINFORMATIONTo(w, decoded.Value); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(decoded.Tail)
			if err != nil || replay.BitLength != row.Bits || !smsInformationEqualBits(replay.Bytes, wire, row.Bits) {
				t.Fatalf("replay %#v: %v", replay, err)
			}
			canonicalWriter := runtime.NewWriter()
			fresh := runtime.CanonicalTo(decoded.Value, row.Bits)
			canonicalWriter.WithWire(fresh.Wire)
			if err := EncodeVGCSSMSINFORMATIONTo(canonicalWriter, fresh); err != nil {
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
			typed := reread.(runtime.Decoded[VGCSSMSINFORMATION])
			smsInformationCheckFields(t, typed.Value, row)
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
				if err := EncodeVGCSSMSINFORMATIONTo(positioned, fresh); err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				at, err := positioned.FinishBits(runtime.BitString{})
				if err != nil || at.BitLength != row.Bits+phase {
					t.Fatalf("phase %d extent %#v: %v", phase, at, err)
				}

				// TS 24.007 V20.0.0 Annex B §B.1.2.2: terminal spare
				// padding follows absolute 0x2B phase, independently of fields.
				expected := make([]byte, (row.Bits+phase+7)/8)
				for bit := 0; bit < row.Bits; bit++ {
					value := wire[bit/8] >> uint(7-bit%8) & 1
					if bit >= row.Content {
						value ^= byte(0x2b>>uint(7-bit%8)&1) ^ byte(0x2b>>uint(7-(bit+phase)%8)&1)
					}
					expected[(bit+phase)/8] |= value << uint(7-(bit+phase)%8)
				}
				if !bytes.Equal(at.Bytes, expected) {
					t.Fatalf("absolute phase %d: %x want %x", phase, at.Bytes, expected)
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
				smsInformationCheckFields(t, reread.(runtime.Decoded[VGCSSMSINFORMATION]).Value, row)
			}
			edited := decoded.Value
			edited.RRShortPD = 1
			ew := runtime.NewWriter()
			if err := EncodeVGCSSMSINFORMATIONTo(ew, edited); err != nil {
				t.Fatal(err)
			}
			edit, err := ew.FinishBits(decoded.Tail)
			want := bytes.Clone(wire)
			want[0] ^= 0x80
			if err != nil || edit.BitLength != row.Bits || !smsInformationEqualBits(edit.Bytes, want, row.Bits) {
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
				smsInformationCheckFields(t, reread.(runtime.Decoded[VGCSSMSINFORMATION]).Value, row)
				stable, err := d.CanonicalAtLength(reread, len(wire))
				if err != nil || !bytes.Equal(stable, canonicalBytes) {
					t.Fatalf("unstable canonical %x: %v", stable, err)
				}
			}
			for candidate := uint8(0); candidate < 32; candidate++ {
				if candidate == 9 {
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

func TestVGCSSMSINFORMATIONInputGrowth(t *testing.T) {
	d, err := LookupClause("9.1.49b", "VGCS SMS INFORMATION")
	if err != nil {
		t.Fatal(err)
	}
	rows := smsInformationVectors(t)
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

func FuzzVGCSSMSINFORMATION(f *testing.F) {
	d, err := LookupClause("9.1.49b", "VGCS SMS INFORMATION")
	if err != nil {
		f.Fatal(err)
	}
	for _, row := range smsInformationVectors(f) {
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
		decoded, ok := raw.(runtime.Decoded[VGCSSMSINFORMATION])
		if !ok {
			t.Fatalf("typed decode %T", raw)
		}
		w := runtime.NewWriter()
		if err := EncodeVGCSSMSINFORMATIONTo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(decoded.Tail)
		if err != nil || out.BitLength != bits || !smsInformationEqualBits(out.Bytes, data, bits) {
			t.Fatalf("accepted input replay %#v: %v", out, err)
		}
	})
}

func TestVGCSSMSINFORMATIONFieldBounds(t *testing.T) {
	row := smsInformationVectors(t)[0]
	wire, err := hex.DecodeString(row.Hex)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LookupClause("9.1.49b", "VGCS SMS INFORMATION")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := d.DecodeBits(wire, row.Bits)
	if err != nil {
		t.Fatal(err)
	}
	original := raw.(runtime.Decoded[VGCSSMSINFORMATION]).Value
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
		if err := EncodeVGCSSMSINFORMATIONTo(runtime.NewWriter(), v); err == nil {
			t.Fatalf("accepted invalid %s", field)
		}
	}
}

// TS 44.018 V19.0.0 §9.1.49b pp10604–10613: the four-bit reference
// and segment fields remain editable independently of the content length.
func TestVGCSSMSINFORMATIONBodyEdits(t *testing.T) {
	for _, row := range smsInformationVectors(t) {
		wire, err := hex.DecodeString(row.Hex)
		if err != nil {
			t.Fatal(err)
		}
		d, err := LookupClause("9.1.49b", "VGCS SMS INFORMATION")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := d.DecodeBits(wire, row.Bits)
		if err != nil {
			t.Fatal(err)
		}
		v := raw.(runtime.Decoded[VGCSSMSINFORMATION]).Value
		var ref *uint8
		if body := v.NonSegmentedSMSDescriptionChoice.NonSegmentedSMSDescription; body != nil {
			ref = &body.NonSegmentedSMSDescription.SMSReferenceNumber
		} else {
			ref = &v.NonSegmentedSMSDescriptionChoice.SegmentedSMSDescription.SegmentedSMSDescription.SMSReferenceNumber
		}
		*ref = 9
		w := runtime.NewWriter()
		if err := EncodeVGCSSMSINFORMATIONTo(w, v); err != nil {
			t.Fatal(err)
		}
		edited, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		reread, err := d.DecodeBits(edited.Bytes, row.Bits)
		if err != nil {
			t.Fatal(err)
		}
		got := smsInformationFields(reread.(runtime.Decoded[VGCSSMSINFORMATION]).Value)["SMSReferenceNumber"]
		if len(got) != 1 || got[0].Uint() != 9 {
			t.Fatalf("reference edit: %v", got)
		}
		*ref = 16
		if err := EncodeVGCSSMSINFORMATIONTo(runtime.NewWriter(), v); err == nil {
			t.Fatal("accepted five-bit reference")
		}
	}
}

func smsComponentFuzz[T any](f *testing.F, name string) {
	d, err := LookupClause("9.1.49b", name)
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{0, 0})
	f.Add([]byte{0xff, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		raw, err := d.Decode(data)
		if err != nil {
			return
		}
		if _, ok := raw.(runtime.Decoded[T]); !ok {
			t.Fatalf("typed component %T", raw)
		}
		out, err := d.Encode(raw)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("component replay %x: %v", out, err)
		}
	})
}
func FuzzNonSegmentedSMSDescription(f *testing.F) {
	smsComponentFuzz[NonSegmentedSMSDescriptionStruct](f, "Non-Segmented_SMS Description struct")
}
func FuzzSegmentedSMSDescription(f *testing.F) {
	smsComponentFuzz[SegmentedSMSDescriptionStruct](f, "Segmented_SMS Description struct")
}
