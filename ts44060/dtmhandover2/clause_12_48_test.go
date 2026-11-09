package dtmhandover2

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.48 supplies the grammar and field widths.
// §§11.1.2–11.1.3 define the reported receiver regions; their remainder
// belongs to the exact containing description, including an empty remainder.
type handoverVector struct {
	Name        string            `json:"name"`
	Bits        string            `json:"bits"`
	ErrorOffset int               `json:"error_offset"`
	ErrorLabel  string            `json:"error_label"`
	Fields      map[string]uint64 `json:"fields"`
}

func handoverVectors(t testing.TB) []handoverVector {
	t.Helper()
	data, err := os.ReadFile("testdata/handover_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []handoverVector
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("empty source vector set")
	}
	return rows
}
func handoverBits(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func handoverFields(v reflect.Value, out map[string][]uint64) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			handoverFields(v.Elem(), out)
		}
		return
	}
	switch v.Kind() {
	case reflect.Struct:
		typ := v.Type()
		if typ == reflect.TypeFor[runtime.WireInfo]() {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			field := typ.Field(i)
			if !field.IsExported() || field.Name == "Wire" {
				continue
			}
			x := v.Field(i)
			for x.Kind() == reflect.Pointer && !x.IsNil() {
				x = x.Elem()
			}
			switch x.Kind() {
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				out[field.Name] = append(out[field.Name], x.Uint())
			default:
				handoverFields(x, out)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			handoverFields(v.Index(i), out)
		}
	}
}
func TestClause12_48(t *testing.T) {
	d, err := Lookup("DTM Handover PS Radio Resources 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range handoverVectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7} {
				prefix := strings.Repeat("0", phase)
				input := handoverBits(prefix + tc.Bits + "101")
				c := runtime.NewReceiverCollector()
				r, err := runtime.NewChoiceReaderBits(input, phase+len(tc.Bits)+3, c)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(tc.Bits))
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := d.DecodeFrom(r)
				if err != nil {
					t.Fatalf("phase %d: %v", phase, err)
				}
				value, ok := decoded.(runtime.Decoded[DTMHandoverPSRadioResources2IE])
				if !ok {
					t.Fatalf("untyped root: %T", decoded)
				}
				fields := map[string][]uint64{}
				handoverFields(reflect.ValueOf(value.Value), fields)
				for name, want := range tc.Fields {
					if len(fields[name]) == 0 || fields[name][0] != want {
						t.Fatalf("typed %s=%v, want %d", name, fields[name], want)
					}
				}
				if r.Position() != phase+len(tc.Bits) {
					t.Fatalf("consumed %d want %d", r.Position(), phase+len(tc.Bits))
				}
				report := c.TakeReceiverReport()
				if tc.ErrorOffset >= 0 {
					if len(report.Regions) != 1 {
						t.Fatalf("region count %d", len(report.Regions))
					}
					region := report.Regions[0]
					if string(region.Label) != tc.ErrorLabel || region.Location.BitOffset != phase+tc.ErrorOffset || region.Location.Width != len(tc.Bits)-tc.ErrorOffset {
						t.Fatalf("region label %q offset %d width %d", region.Label, region.Location.BitOffset, region.Location.Width)
					}
				} else if len(report.Regions) != 0 {
					t.Fatalf("ordinary vector selected %q", report.Regions[0].Label)
				}
				if err := r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				if n, err := r.ReadUint(3); err != nil || n != 5 {
					t.Fatal("outside sentinel was consumed")
				}
				w := runtime.NewWriter()
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				limit, err := w.PushLimit(len(tc.Bits))
				if err != nil {
					t.Fatal(err)
				}
				if err := EncodeDTMHandoverPSRadioResources2IETo(w, value.Value); err != nil {
					t.Fatalf("replay: %v", err)
				}
				if err := w.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				replay, err := w.FinishBits(runtime.BitString{})
				if err != nil || replay.BitLength != phase+len(tc.Bits) || !bytes.Equal(replay.Bytes, handoverBits(prefix+tc.Bits)) {
					t.Fatalf("received replay extent %d: %v", replay.BitLength, err)
				}
				canonical := runtime.Canonical(value.Value)
				fresh := runtime.NewWriter()
				fresh.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, phase+len(tc.Bits)).Wire)
				if err := fresh.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				if _, err := fresh.PushLimit(len(tc.Bits)); err != nil {
					t.Fatal(err)
				}
				err = EncodeDTMHandoverPSRadioResources2IETo(fresh, canonical)
				if tc.ErrorOffset >= 0 {
					var op *runtime.OperationError
					if !errors.As(err, &op) || op.Kind != runtime.InvalidValue {
						t.Fatalf("reserved sender: %v", err)
					}
				} else if err != nil {
					t.Fatalf("fresh sender: %v", err)
				}
				plain, err := runtime.NewReaderBits(input, phase+len(tc.Bits)+3, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := plain.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				if _, err := plain.PushLimit(len(tc.Bits)); err != nil {
					t.Fatal(err)
				}
				direct, err := DecodeDTMHandoverPSRadioResources2IEFrom(plain)
				if err != nil || !runtime.SemanticallyEqual(value.Value, direct) {
					t.Fatalf("typed and descriptor parity: %v", err)
				}
			}
		})
	}
}
func TestClause12_48Truncations(t *testing.T) {
	rows := handoverVectors(t)
	checked := 0
	for _, tc := range rows {
		if tc.ErrorOffset >= 0 || len(tc.Bits) > 256 {
			continue
		}
		if checked == 12 {
			break
		}
		checked++
		for n := 0; n < len(tc.Bits); n++ {
			for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7} {
				r, err := runtime.NewChoiceReaderBits(handoverBits(strings.Repeat("0", phase)+tc.Bits[:n]+"101"), phase+n+3, runtime.NewReceiverCollector())
				if err != nil {
					t.Fatal(err)
				}
				if _, err := r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				limit, err := r.PushLimit(n)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = DecodeDTMHandoverPSRadioResources2IEFrom(r)
				if r.Position() > phase+n {
					t.Fatal("truncation crossed containing limit")
				}
				if r.Position() < phase+n {
					if _, err := r.ReadBitString(phase + n - r.Position()); err != nil {
						t.Fatal(err)
					}
				}
				if err := r.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				if x, err := r.ReadUint(3); err != nil || x != 5 {
					t.Fatal("truncation consumed outside sentinel")
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no ordinary truncation seed")
	}
}
func FuzzClause12_48(f *testing.F) {
	for _, tc := range handoverVectors(f) {
		if len(tc.Bits) < 200 {
			f.Add(handoverBits(tc.Bits), uint16(len(tc.Bits)))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, n uint16) {
		if len(data) > 1024 {
			return
		}
		length := min(int(n), len(data)*8)
		const phase = 0
		bits := make([]byte, (phase+length+7)/8)
		for i := 0; i < length; i++ {
			if data[i/8]&(1<<uint(7-i%8)) != 0 {
				bits[(phase+i)/8] |= 1 << uint(7-(phase+i)%8)
			}
		}
		decode := func(collector *runtime.Collector) (DTMHandoverPSRadioResources2IE, *runtime.Reader, error) {
			r, err := runtime.NewChoiceReaderBits(bits, phase+length, collector)
			if err != nil {
				return DTMHandoverPSRadioResources2IE{}, r, err
			}
			if _, err := r.ReadUint(phase); err != nil {
				return DTMHandoverPSRadioResources2IE{}, r, err
			}
			v, err := DecodeDTMHandoverPSRadioResources2IEFrom(r)
			return v, r, err
		}
		ordinary, r, err := decode(nil)
		reported, _, re := decode(runtime.NewReceiverCollector())
		if fmt.Sprint(err) != fmt.Sprint(re) || !reflect.DeepEqual(ordinary, reported) {
			t.Fatalf("collector parity: %v / %v", err, re)
		}
		if err != nil {
			return
		}
		used := r.Position()
		raw := append([]byte(nil), bits[:(used+7)/8]...)
		if used%8 != 0 {
			raw[len(raw)-1] &= 0xff << uint(8-used%8)
		}
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: used}, used, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err := w.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		if _, err := w.PushLimit(length); err != nil {
			t.Fatal(err)
		}
		if err := EncodeDTMHandoverPSRadioResources2IETo(w, ordinary); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil || out.BitLength != used || !bytes.Equal(out.Bytes, raw) {
			t.Fatalf("bounded replay length %d want %d: %v", out.BitLength, used, err)
		}
	})
}

// The ordinary minimal codeword supplies an editable typed value; changing
// one field preserves its surrounding selector and containing extent.
func TestClause12_48Edit(t *testing.T) {
	tc := handoverVectors(t)[0]
	r, err := runtime.NewReaderBits(handoverBits(tc.Bits), len(tc.Bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	value, err := DecodeDTMHandoverPSRadioResources2IEFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	edited := runtime.Canonical(value)
	edited.RLCRESET ^= 1
	edited = runtime.CanonicalTo(edited, len(tc.Bits))
	w := runtime.NewWriter()
	w.WithWire(edited.Wire)
	if _, err := w.PushLimit(len(tc.Bits)); err != nil {
		t.Fatal(err)
	}
	if err := EncodeDTMHandoverPSRadioResources2IETo(w, edited); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	if out.BitLength != len(tc.Bits) || bytes.Equal(out.Bytes, handoverBits(tc.Bits)) {
		t.Fatal("edit lost or changed extent")
	}
	reader, err := runtime.NewReaderBits(out.Bytes, out.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeDTMHandoverPSRadioResources2IEFrom(reader)
	if err != nil || decoded.RLCRESET != edited.RLCRESET {
		t.Fatalf("typed edit: %v", err)
	}
}

// TS 44.060 V19.0.0 §§11.1.2–11.1.3: a labelled receiver region
// owns the containing remainder, including an empty or growing suffix.
func TestClause12_48ReceiverRemainder(t *testing.T) {
	for _, tc := range handoverVectors(t) {
		if tc.ErrorLabel != "TBF mode error" {
			continue
		}
		for _, length := range []int{0, 8, 256, 4096} {
			body := tc.Bits[:tc.ErrorOffset] + strings.Repeat("1", length)
			d, err := Lookup("DTM Handover PS Radio Resources 2 IE")
			if err != nil {
				t.Fatal(err)
			}
			value, _, report, err := d.DecodeBitsWithReport(handoverBits(body), len(body))
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Regions) != 1 || string(report.Regions[0].Label) != "TBF mode error" || report.Regions[0].Location.Width != length {
				t.Fatalf("remainder %d: %+v", length, report)
			}
			typed := value.(runtime.Decoded[DTMHandoverPSRadioResources2IE])
			w := runtime.NewWriter()
			if err := EncodeDTMHandoverPSRadioResources2IETo(w, typed.Value); err != nil {
				t.Fatal(err)
			}
			out, err := w.FinishBits(runtime.BitString{})
			if err != nil || out.BitLength != len(body) || !bytes.Equal(out.Bytes, handoverBits(body)) {
				t.Fatalf("remainder replay %d: %v", length, err)
			}
		}
		return
	}
	t.Fatal("no printed receiver-region seed")
}

// TS 44.060 V19.0.0 §12.48: the Extension Information // retains
// the complete earlier-release group when later release groups are absent.
func TestReleasedExtensionPrefix(t *testing.T) {
	bits := "00000000000000000000000000000000000000000000000000000000000000000010000100000000000000000000000000000000000000100000010001100111000"
	d, err := Lookup("DTM Handover PS Radio Resources 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	value, err := d.DecodeBits(handoverBits(bits), len(bits))
	if err != nil {
		t.Fatal(err)
	}
	typed := value.(runtime.Decoded[DTMHandoverPSRadioResources2IE])
	fields := map[string][]uint64{}
	handoverFields(reflect.ValueOf(typed.Value), fields)
	if len(fields["MTTIDOWNLINKASSIGNMENTC1"]) != 1 || fields["MTTIDOWNLINKASSIGNMENTC1"][0] != 1 {
		t.Fatal("typed release prefix lost")
	}
	w := runtime.NewWriter()
	if err := EncodeDTMHandoverPSRadioResources2IETo(w, typed.Value); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != len(bits) || !bytes.Equal(out.Bytes, handoverBits(bits)) {
		t.Fatalf("prefix replay: %v", err)
	}
	edited := runtime.Canonical(typed.Value)
	edited.ExtensionLengthGroup.ExtensionLength = 63
	edited = runtime.CanonicalTo(edited, 188)
	fresh := runtime.NewWriter()
	fresh.WithWire(edited.Wire)
	if _, err := fresh.PushLimit(188); err != nil {
		t.Fatal(err)
	}
	if err := EncodeDTMHandoverPSRadioResources2IETo(fresh, edited); err != nil {
		t.Fatal(err)
	}
	complete, err := fresh.FinishBits(runtime.BitString{})
	if err != nil || complete.BitLength != 188 {
		t.Fatalf("complete fresh extension: %v", err)
	}
	reader, err := runtime.NewReaderBits(complete.Bytes, complete.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeDTMHandoverPSRadioResources2IEFrom(reader)
	if err != nil || !runtime.SemanticallyEqual(edited, again) {
		t.Fatalf("complete extension semantics: %v", err)
	}
}
