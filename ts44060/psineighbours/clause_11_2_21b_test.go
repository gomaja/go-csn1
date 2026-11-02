package psineighbours_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	subject "github.com/gomaja/go-csn1/ts44060/psineighbours"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// TS 44.060 V19.0.0 §11.2.21b, paragraphs 17397–17451; §11 padding;
// §§10.0b.2.1 and 10.3.1.1: CS-1 has 184 logical bits. §11.2.0.1
// paragraph 13383 assigns external selector 111101. The vectors are synthetic.
type psi3quaterVector struct {
	Name      string   `json:"name"`
	Bits      string   `json:"bits"`
	Extent    int      `json:"extent"`
	FullBlock bool     `json:"full_block"`
	Fields    []string `json:"fields"`
	Tail      string   `json:"tail"`
}

func psi3quaterVectors(t testing.TB) []psi3quaterVector {
	t.Helper()
	data, err := os.ReadFile("testdata/clause_11_2_21b_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []psi3quaterVector
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}
func psi3quaterBytes(bits string) []byte {
	b := make([]byte, (len(bits)+7)/8)
	for i, v := range bits {
		if v == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func psi3quaterReader(t testing.TB, raw []byte, extent int, full bool) (*runtime.Reader, *runtime.Collector) {
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
	if err != nil || tag != 61 {
		t.Fatalf("selector %d: %v", tag, err)
	}
	return r, c
}
func psi3quaterFields(v any) []string {
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
				if len(n) > 1 && n[0] == 'n' && n[1] >= '0' && n[1] <= '9' {
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
func psi3quaterEncode(t testing.TB, raw []byte, extent int, wire runtime.WireInfo, v subject.PSI3QuaterMessageContent) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err := w.WriteUint(uint64(raw[0]), 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(61, 6); err != nil {
		t.Fatal(err)
	}
	if err := subject.EncodePSI3QuaterMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	if err != nil || got.BitLength != extent {
		t.Fatalf("encoded extent %d want %d: %v", got.BitLength, extent, err)
	}
	return got.Bytes
}
func TestClause11_2_21bSourceVariants(t *testing.T) {
	for _, tc := range psi3quaterVectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			raw := psi3quaterBytes(tc.Bits)
			r, c := psi3quaterReader(t, raw, tc.Extent, tc.FullBlock)
			v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
			if err != nil || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Ignored != nil {
				t.Fatalf("typed decode: %+v %v", v, err)
			}
			if r.Position() != tc.Extent {
				t.Fatalf("position %d want %d", r.Position(), tc.Extent)
			}
			if got := psi3quaterFields(v); !reflect.DeepEqual(got, tc.Fields) {
				t.Fatalf("typed fields\ngot %v\nwant%v", got, tc.Fields)
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("normal source arm entered error region: %+v", report)
			}
			sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			if got := psi3quaterEncode(t, raw, tc.Extent, sealed, v); !bytes.Equal(got, raw) {
				t.Fatalf("replay %x want %x", got, raw)
			}
			if tc.Tail != "ignored" {
				cv := runtime.Canonical(v)
				target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire
				if got := psi3quaterEncode(t, raw, tc.Extent, target, cv); !bytes.Equal(got, raw) {
					t.Fatalf("canonical %x want %x", got, raw)
				}
				cv.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := psi3quaterEncode(t, raw, tc.Extent, target, cv)
				er, _ := psi3quaterReader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePSI3QuaterMessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != cv.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("typed edit: %+v %v", ev, e)
				}
			} else {
				// The printed compatibility arm has no sender padding. A fresh encoding
				// cannot manufacture received ignored bits to fill a larger target.
				_, ce := subject.EncodePSI3QuaterMessageContentCanonicalAtLength(v, 23)
				var underfill *runtime.ExtentError
				if !errors.As(ce, &underfill) || underfill.Required || underfill.Minimum != 23 || underfill.Maximum != 23 || underfill.Actual >= 23 {
					t.Fatalf("compatibility-arm canonical extent: %v", ce)
				}
				v.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := psi3quaterEncode(t, raw, tc.Extent, sealed, v)
				er, _ := psi3quaterReader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePSI3QuaterMessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != v.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("compatibility-arm edit: %+v %v", ev, e)
				}
			}
		})
	}
}
func TestClause11_2_21bReceiverAndBounds(t *testing.T) {
	tc := psi3quaterVectors(t)[0]
	raw := psi3quaterBytes(tc.Bits)
	for end := 14; end < 24; end++ {
		r, _ := psi3quaterReader(t, raw, end, false)
		v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
		if err == nil && v.PAGEMODEGroupFallback.Ignored == nil {
			t.Fatalf("truncated distribution accepted without located error at %d: %+v", end, v)
		}
	}
	// The caller's exact bound excludes the following sentinel (Annex B §B.2.1).
	for _, extra := range []int{1, 2, 8, 64, 1024, 4096} {
		data := append(append([]byte(nil), raw...), bytes.Repeat([]byte{0xa5}, extra)...)
		r, _ := psi3quaterReader(t, data, 184, true)
		v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
		if err != nil || v.PAGEMODEGroupFallback.Known == nil || r.Position() != 184 {
			t.Fatalf("sentinel consumed: %d %v", r.Position(), err)
		}
	}
	// Malformed PAGE_MODE prefix chooses the printed Distribution part error;
	// arbitrary received error bits are retained for replay, never made typed data.
	for end := 14; end < 18; end++ {
		r, c := psi3quaterReader(t, raw, end, false)
		v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
		if err == nil {
			report := c.TakeReceiverReport()
			if v.PAGEMODEGroupFallback.Ignored == nil || len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || r.Position() != end {
				t.Fatalf("missing located receiver region at %d: %+v", end, report)
			}
		}
	}
}
func FuzzClause11_2_21bReplay(f *testing.F) {
	for _, v := range psi3quaterVectors(f) {
		if v.FullBlock {
			f.Add(psi3quaterBytes(v.Bits))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 || raw[0]>>6 != 1 || (raw[1]>>2) != 61 {
			return
		}
		r, _ := psi3quaterReader(t, raw, 184, true)
		v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
		if err != nil {
			return
		}
		sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		if got := psi3quaterEncode(t, raw, 184, sealed, v); !bytes.Equal(got, raw) {
			t.Fatalf("replay %x != %x", got, raw)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.21b pp17426–17428: the leading
// FDD designation is zero. In this minimal full-block vector it follows
// 14 framing/type bits, 12 header bits, and seven presence/list bits.
func TestClause11_2_21bReservedFDDDesignation(t *testing.T) {
	var vector psi3quaterVector
	for _, candidate := range psi3quaterVectors(t) {
		if candidate.Name == "NR_OF_FDD_CELLS table index 1" {
			vector = candidate
			break
		}
	}
	if !vector.FullBlock || vector.Bits[33] != '0' {
		t.Fatal("missing reserved-designation vector")
	}
	raw := psi3quaterBytes(vector.Bits)
	raw[33/8] ^= 1 << uint(7-33%8)
	r, c := psi3quaterReader(t, raw, 184, true)
	v, err := subject.DecodePSI3QuaterMessageContentFrom(r)
	report := c.TakeReceiverReport()
	if err != nil || v.PAGEMODEGroupFallback.Ignored == nil || len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || r.Position() != 184 {
		t.Fatalf("malformed FDD designation: offset %d, report %+v, error %v", r.Position(), report, err)
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if got := psi3quaterEncode(t, raw, 184, wire, v); !bytes.Equal(got, raw) {
		t.Fatal("receiver error replay differs")
	}
}
