package packetmeasurementorder_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	subject "github.com/gomaja/go-csn1/ts44060/packetmeasurementorder"
)

// TS 44.060 V19.0.0 §11.2.9b, paragraphs 15520–15761; §11 padding;
// §§10.0b.2.1 and 10.3.1.1: CS-1 has 184 logical bits. §11.2.0.1
// paragraph 13347 assigns external selector 000011. The vectors are synthetic.
type measurementOrderVector struct {
	Name      string   `json:"name"`
	Bits      string   `json:"bits"`
	Extent    int      `json:"extent"`
	FullBlock bool     `json:"full_block"`
	Fields    []string `json:"fields"`
	Tail      string   `json:"tail"`
}

func measurementOrderVectors(t testing.TB) []measurementOrderVector {
	t.Helper()
	data, err := os.ReadFile("testdata/clause_11_2_9b_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []measurementOrderVector
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}
func measurementOrderBytes(bits string) []byte {
	b := make([]byte, (len(bits)+7)/8)
	for i, v := range bits {
		if v == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func measurementOrderReader(t testing.TB, raw []byte, extent int, full bool) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(raw, extent, c)
	if err != nil {
		t.Fatal(err)
	}
	if full {
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
		if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
			t.Fatal(err)
		}
	} else if _, err = r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	tag, err := r.ReadUint(6)
	if err != nil || tag != 3 {
		t.Fatalf("selector %d: %v", tag, err)
	}
	return r, c
}
func measurementOrderFields(v any) []string {
	var out []string
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, name string) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if !v.IsNil() {
				walk(v.Elem(), name)
			}
			return
		}
		if b, ok := v.Interface().(runtime.BitString); ok {
			var s strings.Builder
			for i := 0; i < b.BitLength; i++ {
				if b.Bytes[i/8]&(1<<uint(7-i%8)) != 0 {
					s.WriteByte('1')
				} else {
					s.WriteByte('0')
				}
			}
			out = append(out, name+"=bits:"+s.String())
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				field := v.Type().Field(i)
				if field.Name == "Wire" || field.Name == "Alternative" || field.Name == "PaddingBits" {
					continue
				}
				n := strings.Map(func(r rune) rune {
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						return unicode.ToLower(r)
					}
					return -1
				}, strings.TrimSuffix(field.Name, "List"))
				if len(n) > 1 && n[0] == 'n' && n[1] >= '0' && n[1] <= '9' && strings.Trim(n[1:], "0123456789") != "" {
					n = n[1:]
				}
				walk(v.Field(i), n)
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), name)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			out = append(out, fmt.Sprintf("%s=%d", name, v.Uint()))
		}
	}
	walk(reflect.ValueOf(v), "")
	return out
}
func measurementOrderEncode(t testing.TB, raw []byte, extent int, wire runtime.WireInfo, v subject.PacketMeasurementOrderMessageContent) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err := w.WriteUint(uint64(raw[0]), 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(3, 6); err != nil {
		t.Fatal(err)
	}
	if err := subject.EncodePacketMeasurementOrderMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	if err != nil || got.BitLength != extent {
		t.Fatalf("encoded extent %d want %d: %v", got.BitLength, extent, err)
	}
	return got.Bytes
}
func TestClause11_2_9bSourceVariants(t *testing.T) {
	for _, tc := range measurementOrderVectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			raw := measurementOrderBytes(tc.Bits)
			r, c := measurementOrderReader(t, raw, tc.Extent, tc.FullBlock)
			v, err := subject.DecodePacketMeasurementOrderMessageContentFrom(r)
			if err != nil || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Ignored != nil {
				t.Fatalf("typed decode: %+v %v", v, err)
			}
			if r.Position() != tc.Extent {
				t.Fatalf("position %d want %d", r.Position(), tc.Extent)
			}
			if got := measurementOrderFields(v); !reflect.DeepEqual(got, tc.Fields) {
				t.Fatalf("typed fields\ngot %v\nwant%v", got, tc.Fields)
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("normal source arm entered error region: %+v", report)
			}
			sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			if got := measurementOrderEncode(t, raw, tc.Extent, sealed, v); !bytes.Equal(got, raw) {
				t.Fatalf("replay %x want %x", got, raw)
			}
			if tc.Tail != "ignored" {
				cv := runtime.Canonical(v)
				target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire
				if got := measurementOrderEncode(t, raw, tc.Extent, target, cv); !bytes.Equal(got, raw) {
					t.Fatalf("canonical %x want %x", got, raw)
				}
				cv.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := measurementOrderEncode(t, raw, tc.Extent, target, cv)
				er, _ := measurementOrderReader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePacketMeasurementOrderMessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != cv.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("typed edit: %+v %v", ev, e)
				}
			} else {
				// The printed compatibility arm has no sender padding. A fresh encoding
				// cannot manufacture received ignored bits to fill a larger target.
				cw := runtime.NewWriter()
				cw.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire)
				if err := cw.WriteUint(uint64(raw[0]), 8); err != nil {
					t.Fatal(err)
				}
				if err := cw.WriteUint(3, 6); err != nil {
					t.Fatal(err)
				}
				ce := subject.EncodePacketMeasurementOrderMessageContentTo(cw, runtime.Canonical(v))
				if ce == nil {
					short, err := cw.FinishBits(runtime.BitString{})
					if err != nil || short.BitLength >= tc.Extent {
						t.Fatalf("compatibility arm invented ignored bits: %d/%d %v", short.BitLength, tc.Extent, err)
					}
				} else {
					var underfill *runtime.ExtentError
					if !errors.As(ce, &underfill) || underfill.Required {
						t.Fatalf("compatibility-arm canonical extent: %v", ce)
					}
				}
				v.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := measurementOrderEncode(t, raw, tc.Extent, sealed, v)
				er, _ := measurementOrderReader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePacketMeasurementOrderMessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != v.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("compatibility-arm edit: %+v %v", ev, e)
				}
			}
		})
	}
}
func TestClause11_2_9bReceiverAndBounds(t *testing.T) {
	tc := measurementOrderVectors(t)[0]
	raw := measurementOrderBytes(tc.Bits)
	for end := 14; end < 16; end++ {
		r, _ := measurementOrderReader(t, raw, end, false)
		v, err := subject.DecodePacketMeasurementOrderMessageContentFrom(r)
		if err == nil && v.PAGEMODEGroupFallback.Ignored == nil {
			t.Fatalf("truncated distribution accepted without located error at %d: %+v", end, v)
		}
	}
	// The caller's exact bound excludes the following sentinel (Annex B §B.2.1).
	for _, extra := range []int{1, 2, 8, 64, 1024, 4096} {
		data := append(append([]byte(nil), raw...), bytes.Repeat([]byte{0xa5}, extra)...)
		r, _ := measurementOrderReader(t, data, 184, true)
		v, err := subject.DecodePacketMeasurementOrderMessageContentFrom(r)
		if err != nil || v.PAGEMODEGroupFallback.Known == nil || r.Position() != 184 {
			t.Fatalf("sentinel consumed: %d %v", r.Position(), err)
		}
	}
	// Malformed PAGE_MODE prefix chooses the printed Distribution part error;
	// arbitrary received error bits are retained for replay, never made typed data.
	for end := 14; end < 16; end++ {
		r, c := measurementOrderReader(t, raw, end, false)
		v, err := subject.DecodePacketMeasurementOrderMessageContentFrom(r)
		if err == nil {
			report := c.TakeReceiverReport()
			if v.PAGEMODEGroupFallback.Ignored == nil || len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || r.Position() != end {
				t.Fatalf("missing located receiver region at %d: %+v", end, report)
			}
		}
	}
}
func FuzzClause11_2_9bReplay(f *testing.F) {
	for _, v := range measurementOrderVectors(f) {
		if v.FullBlock {
			f.Add(measurementOrderBytes(v.Bits))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 || raw[0]>>6 != 1 || (raw[1]>>2) != 3 {
			return
		}
		r, _ := measurementOrderReader(t, raw, 184, true)
		v, err := subject.DecodePacketMeasurementOrderMessageContentFrom(r)
		if err != nil {
			return
		}
		sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		if got := measurementOrderEncode(t, raw, 184, sealed, v); !bytes.Equal(got, raw) {
			t.Fatalf("replay %x != %x", got, raw)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.9b pp15587–15591: the raw count
// NR_OF_REMOVED_FREQ encodes one fewer than the ordered six-bit entries.
func TestClause11_2_9bRemovedFrequencyOrder(t *testing.T) {
	bits := "1" + "00001" + "000001" + "000011" + "0"
	raw := measurementOrderBytes(bits)
	r, err := runtime.NewReaderBits(raw, len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := subject.DecodeNCFrequencyListStructFrom(r)
	if err != nil || v.NROFREMOVEDFREQGroup == nil {
		t.Fatalf("removed list: %+v %v", v, err)
	}
	group := v.NROFREMOVEDFREQGroup
	if group.NROFREMOVEDFREQ != 1 || !reflect.DeepEqual(group.REMOVEDFREQINDEXList, []uint8{1, 3}) || r.Position() != 19 {
		t.Fatalf("ordered entries: %+v at %d", group, r.Position())
	}
	group.REMOVEDFREQINDEXList[1] = 63
	w := runtime.NewWriter()
	if err := subject.EncodeNCFrequencyListStructTo(w, v); err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	want := measurementOrderBytes("1" + "00001" + "000001" + "111111" + "0")
	if err != nil || got.BitLength != 19 || !bytes.Equal(got.Bytes, want) {
		t.Fatalf("ordered edit: %v", err)
	}
}
