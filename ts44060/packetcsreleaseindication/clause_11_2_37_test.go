package packetcsreleaseindication

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

// TS 44.060 V19.0.0 §11.2.37 supplies the grammar and field widths.
// §§11.1.2–11.1.3 define the reported receiver regions; their remainder
// belongs to the exact containing description, including an empty remainder.
type releaseVector struct {
	Name          string            `json:"name"`
	Bits          string            `json:"bits"`
	ErrorOffset   int               `json:"error_offset"`
	ErrorLabel    string            `json:"error_label"`
	Fields        map[string]uint64 `json:"fields"`
	PaddingPhase  *int              `json:"padding_phase"`
	PaddingOffset *int              `json:"padding_offset"`
}

func releaseVectors(t testing.TB) []releaseVector {
	t.Helper()
	data, err := os.ReadFile("testdata/release_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []releaseVector
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("empty source vector set")
	}
	return rows
}
func releaseBits(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func releaseFields(v reflect.Value, out map[string][]uint64) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			releaseFields(v.Elem(), out)
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
			switch x.Kind() {
			case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				out[field.Name] = append(out[field.Name], x.Uint())
			case reflect.Pointer:
				if !x.IsNil() && x.Elem().Kind() == reflect.Uint8 {
					out[field.Name] = append(out[field.Name], x.Elem().Uint())
				} else {
					releaseFields(x, out)
				}
			default:
				releaseFields(x, out)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			releaseFields(v.Index(i), out)
		}
	}
}
func TestClause11_2_37(t *testing.T) {
	d, err := Lookup("Packet CS Release message content")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range releaseVectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			for _, phase := range []int{14, 22, 30} {
				prefix := strings.Repeat("0", phase)
				input := releaseBits(prefix + tc.Bits + "101")
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
				value, ok := decoded.(runtime.Decoded[PacketCSReleaseMessageContent])
				if !ok {
					t.Fatalf("untyped root: %T", decoded)
				}
				fields := map[string][]uint64{}
				releaseFields(reflect.ValueOf(value.Value), fields)
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
				if err := EncodePacketCSReleaseMessageContentTo(w, value.Value); err != nil {
					t.Fatalf("replay: %v", err)
				}
				if err := w.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteLiteral("101"); err != nil {
					t.Fatal(err)
				}
				replay, err := w.FinishBits(runtime.BitString{})
				if err != nil || replay.BitLength != phase+len(tc.Bits)+3 || !bytes.Equal(replay.Bytes, input) {
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
				err = EncodePacketCSReleaseMessageContentTo(fresh, canonical)
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
				direct, err := DecodePacketCSReleaseMessageContentFrom(plain)
				if err != nil || !runtime.SemanticallyEqual(value.Value, direct) {
					t.Fatalf("typed and descriptor parity: %v", err)
				}
			}
		})
	}
}
func TestClause11_2_37Truncations(t *testing.T) {
	rows := releaseVectors(t)
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
			for _, phase := range []int{14} {
				r, err := runtime.NewChoiceReaderBits(releaseBits(strings.Repeat("0", phase)+tc.Bits[:n]+"101"), phase+n+3, runtime.NewReceiverCollector())
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
				_, _ = DecodePacketCSReleaseMessageContentFrom(r)
				if r.Position() > phase+n {
					t.Fatal("truncation crossed containing limit")
				}
				if err := r.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				if r.Position() < phase+n {
					if _, err := r.ReadBitString(phase + n - r.Position()); err != nil {
						t.Fatal(err)
					}
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
func FuzzClause11_2_37(f *testing.F) {
	for _, tc := range releaseVectors(f) {
		if len(tc.Bits) < 200 {
			f.Add(releaseBits(tc.Bits), uint16(len(tc.Bits)))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, n uint16) {
		if len(data) > 1024 {
			return
		}
		length := min(int(n), len(data)*8)
		const phase = 14
		bits := make([]byte, (phase+length+7)/8)
		for i := 0; i < length; i++ {
			if data[i/8]&(1<<uint(7-i%8)) != 0 {
				bits[(phase+i)/8] |= 1 << uint(7-(phase+i)%8)
			}
		}
		decode := func(collector *runtime.Collector) (PacketCSReleaseMessageContent, *runtime.Reader, error) {
			r, err := runtime.NewChoiceReaderBits(bits, phase+length, collector)
			if err != nil {
				return PacketCSReleaseMessageContent{}, r, err
			}
			if _, err := r.ReadUint(phase); err != nil {
				return PacketCSReleaseMessageContent{}, r, err
			}
			v, err := DecodePacketCSReleaseMessageContentFrom(r)
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
		if err := EncodePacketCSReleaseMessageContentTo(w, ordinary); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil || out.BitLength != used || !bytes.Equal(out.Bytes, raw) {
			t.Fatalf("bounded replay length %d want %d: %v", out.BitLength, used, err)
		}
	})
}

// TS 44.060 V19.0.0 §§10.3.1, 11.2.0.1 p13388 and 11.2.37:
// one MAC octet and selector 101001 place this downlink content at bit 14.
// Paragraph 12938 puts a zero selector before the §11 spare-padding sequence.
func TestClause11_2_37CompleteControlBlocks(t *testing.T) {
	phases := map[int]bool{}
	checked := 0
	for _, tc := range releaseVectors(t) {
		if tc.ErrorOffset >= 0 || len(tc.Bits) != 170 || tc.PaddingPhase == nil {
			continue
		}
		t.Run(tc.Name, func(t *testing.T) {
			input := releaseBits("01000000" + "101001" + tc.Bits)
			c := runtime.NewReceiverCollector()
			r, err := runtime.NewChoiceReaderBits(input, 184, c)
			if err != nil {
				t.Fatal(err)
			}
			h, err := controlblock.DecodeDownlinkHeader(r)
			if err != nil || h.MAC.PayloadType != 1 || r.Position() != 8 {
				t.Fatalf("MAC header: %v", err)
			}
			selector, err := r.ReadUint(6)
			if err != nil || selector != 41 || r.Position() != 14 {
				t.Fatal("printed downlink selector differs")
			}
			value, err := DecodePacketCSReleaseMessageContentFrom(r)
			if err != nil || r.Position() != 184 {
				t.Fatalf("complete content at bit %d: %v", r.Position(), err)
			}
			report := c.TakeReceiverReport()
			if len(report.Regions) != 0 {
				t.Fatalf("ordinary carrier selected %q", report.Regions[0].Label)
			}
			wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: input, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			w := runtime.NewWriter()
			w.WithWire(wire)
			if _, err := w.PushLimit(184); err != nil {
				t.Fatal(err)
			}
			if err := controlblock.EncodeDownlinkHeader(w, h); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(41, 6); err != nil {
				t.Fatal(err)
			}
			if err := EncodePacketCSReleaseMessageContentTo(w, value); err != nil {
				t.Fatal(err)
			}
			got, err := w.FinishBits(runtime.BitString{})
			if err != nil || got.BitLength != 184 || !bytes.Equal(got.Bytes, input) {
				t.Fatalf("complete carrier replay: %v", err)
			}
			decoded, err := DecodePacketCSReleaseMessageContent(releaseBits(tc.Bits))
			if err != nil || !runtime.SemanticallyEqual(value, decoded.Value) {
				t.Fatalf("byte function source coordinate: %v", err)
			}
			replay, err := EncodePacketCSReleaseMessageContent(decoded.Value)
			if err != nil || !bytes.Equal(replay, releaseBits(tc.Bits)) {
				t.Fatalf("byte replay: %v", err)
			}
		})
		phases[*tc.PaddingPhase] = true
		checked++
	}
	if checked == 0 || len(phases) != 8 {
		t.Fatalf("complete carrier coverage: %d cases, %d padding phases", checked, len(phases))
	}
	d, err := Lookup("Packet CS Release message content")
	if err != nil || SourceOrigin(d) != 14 {
		t.Fatalf("source origin: %v", err)
	}
	for phase := 0; phase < 8; phase++ {
		if phase == 6 {
			continue
		}
		r, err := runtime.NewChoiceReaderBits(make([]byte, 24), 184, runtime.NewReceiverCollector())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		_, err = DecodePacketCSReleaseMessageContentFrom(r)
		var mismatch *ContentPhaseError
		if !errors.As(err, &mismatch) || r.Position() != phase {
			t.Fatalf("phase %d advanced to %d: %v", phase, r.Position(), err)
		}
	}
}

func TestClause11_2_37TypedEdit(t *testing.T) {
	for _, tc := range releaseVectors(t) {
		if tc.Name != "arm-0" {
			continue
		}
		input := releaseBits(tc.Bits)
		decoded, err := DecodePacketCSReleaseMessageContent(input)
		if err != nil {
			t.Fatal(err)
		}
		edited := decoded.Value
		known := *edited.PAGEMODEGroupFallback.Known
		known.PAGEMODE = (known.PAGEMODE + 1) & 3
		edited.PAGEMODEGroupFallback.Known = &known
		expected := bytes.Clone(input)
		expected[0] = (expected[0] & 0x3f) | (known.PAGEMODE << 6)
		out, err := EncodePacketCSReleaseMessageContent(edited)
		if err != nil || !bytes.Equal(out, expected) {
			t.Fatalf("typed page-mode edit: %v", err)
		}
		result, err := DecodePacketCSReleaseMessageContent(out)
		if err != nil || result.Value.PAGEMODEGroupFallback.Known.PAGEMODE != known.PAGEMODE {
			t.Fatalf("edited typed value: %v", err)
		}
		unchanged, err := EncodePacketCSReleaseMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(unchanged, input) {
			t.Fatalf("edit changed the received value: %v", err)
		}
		return
	}
	t.Fatal("missing complete source-derived edit seed")
}

func TestClause11_2_37IgnoredRemainderGrowth(t *testing.T) {
	var seed *releaseVector
	rows := releaseVectors(t)
	for i := range rows {
		if rows[i].ErrorLabel == "Frequency parameters error" {
			seed = &rows[i]
			break
		}
	}
	if seed == nil {
		t.Fatal("missing source frequency-error seed")
	}
	var previous uint64
	for _, octets := range []int{128, 512, 2048} {
		content := seed.Bits[:seed.ErrorOffset] + strings.Repeat("1", octets*8)
		data := releaseBits(strings.Repeat("0", 14) + content)
		goruntime.GC()
		old := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		collector := runtime.NewReceiverCollector()
		r, err := runtime.NewChoiceReaderBits(data, 14+len(content), collector)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		_, err = DecodePacketCSReleaseMessageContentFrom(r)
		goruntime.ReadMemStats(&after)
		debug.SetGCPercent(old)
		if err != nil || r.Position() != 14+len(content) {
			t.Fatalf("%d-octet receiver remainder: %v", octets, err)
		}
		report := collector.TakeReceiverReport()
		if len(report.Regions) != 1 || report.Regions[0].Location.Width != octets*8 {
			t.Fatalf("%d-octet region extent", octets)
		}
		allocated := after.TotalAlloc - before.TotalAlloc
		if !raceEnabled && (after.NumGC != before.NumGC || allocated > uint64(octets)*256+65536 || previous > 0 && allocated > previous*5+65536) {
			t.Fatalf("nonlinear receiver remainder: %d octets, %d bytes after %d", octets, allocated, previous)
		}
		t.Logf("%d-octet remainder: %d allocated bytes", octets, allocated)
		previous = allocated
	}
}
