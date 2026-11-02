package packetsysteminformationtype3_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	subject "github.com/gomaja/go-csn1/ts44060/packetsysteminformationtype3"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// TS 44.060 V19.0.0 §11.2.20, paragraphs 16942–17074; §11 padding;
// §§10.0b.2.1 and 10.3.1.1: CS-1 has 184 logical bits. §11.2.0.1
// paragraph 13373 assigns external selector 110011. The vectors are synthetic.
type psi3Vector struct {
	Name      string   `json:"name"`
	Bits      string   `json:"bits"`
	Extent    int      `json:"extent"`
	FullBlock bool     `json:"full_block"`
	Fields    []string `json:"fields"`
	Tail      string   `json:"tail"`
}

func psi3Vectors(t testing.TB) []psi3Vector {
	t.Helper()
	data, err := os.ReadFile("testdata/clause_11_2_20_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []psi3Vector
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	return vectors
}
func psi3Bytes(bits string) []byte {
	b := make([]byte, (len(bits)+7)/8)
	for i, v := range bits {
		if v == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func psi3Reader(t testing.TB, raw []byte, extent int, full bool) (*runtime.Reader, *runtime.Collector) {
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
	if err != nil || tag != 51 {
		t.Fatalf("selector %d: %v", tag, err)
	}
	return r, c
}
func psi3Fields(v any) []string {
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
func psi3Encode(t testing.TB, raw []byte, extent int, wire runtime.WireInfo, v subject.PSI3MessageContent) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err := w.WriteUint(uint64(raw[0]), 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(51, 6); err != nil {
		t.Fatal(err)
	}
	if err := subject.EncodePSI3MessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	if err != nil || got.BitLength != extent {
		t.Fatalf("encoded extent %d want %d: %v", got.BitLength, extent, err)
	}
	return got.Bytes
}
func TestClause11_2_20SourceVariants(t *testing.T) {
	for _, tc := range psi3Vectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			raw := psi3Bytes(tc.Bits)
			r, c := psi3Reader(t, raw, tc.Extent, tc.FullBlock)
			v, err := subject.DecodePSI3MessageContentFrom(r)
			if err != nil || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Ignored != nil {
				t.Fatalf("typed decode: %+v %v", v, err)
			}
			if r.Position() != tc.Extent {
				t.Fatalf("position %d want %d", r.Position(), tc.Extent)
			}
			if got := psi3Fields(v); !reflect.DeepEqual(got, tc.Fields) {
				t.Fatalf("typed fields\ngot %v\nwant%v", got, tc.Fields)
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("normal source arm entered error region: %+v", report)
			}
			sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			if got := psi3Encode(t, raw, tc.Extent, sealed, v); !bytes.Equal(got, raw) {
				t.Fatalf("replay %x want %x", got, raw)
			}
			if tc.Tail != "ignored" {
				cv := runtime.Canonical(v)
				target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire
				if got := psi3Encode(t, raw, tc.Extent, target, cv); !bytes.Equal(got, raw) {
					t.Fatalf("canonical %x want %x", got, raw)
				}
				cv.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := psi3Encode(t, raw, tc.Extent, target, cv)
				er, _ := psi3Reader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePSI3MessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != cv.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("typed edit: %+v %v", ev, e)
				}
			} else {
				// The printed compatibility arm has no sender padding. A fresh encoding
				// cannot manufacture received ignored bits to fill a larger target.
				_, ce := subject.EncodePSI3MessageContentCanonicalAtLength(v, 23)
				var underfill *runtime.ExtentError
				if !errors.As(ce, &underfill) || underfill.Required || underfill.Minimum != 23 || underfill.Maximum != 23 || underfill.Actual >= 23 {
					t.Fatalf("compatibility-arm canonical extent: %v", ce)
				}
				v.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
				edited := psi3Encode(t, raw, tc.Extent, sealed, v)
				er, _ := psi3Reader(t, edited, tc.Extent, tc.FullBlock)
				ev, e := subject.DecodePSI3MessageContentFrom(er)
				if e != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != v.PAGEMODEGroupFallback.Known.PAGEMODE {
					t.Fatalf("compatibility-arm edit: %+v %v", ev, e)
				}
			}
		})
	}
}
func TestClause11_2_20ReceiverAndBounds(t *testing.T) {
	tc := psi3Vectors(t)[0]
	raw := psi3Bytes(tc.Bits)
	for end := 14; end < 24; end++ {
		r, _ := psi3Reader(t, raw, end, false)
		v, err := subject.DecodePSI3MessageContentFrom(r)
		if err == nil && v.PAGEMODEGroupFallback.Ignored == nil {
			t.Fatalf("truncated distribution accepted without located error at %d: %+v", end, v)
		}
	}
	// The caller's exact bound excludes the following sentinel (Annex B §B.2.1).
	for _, extra := range []int{1, 2, 8, 64, 1024, 4096} {
		data := append(append([]byte(nil), raw...), bytes.Repeat([]byte{0xa5}, extra)...)
		r, err := runtime.NewChoiceReaderBits(data, len(data)*8, nil)
		if err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(184)
		if err != nil {
			t.Fatal(err)
		}
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
		if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(6); err != nil {
			t.Fatal(err)
		}
		v, err := subject.DecodePSI3MessageContentFrom(r)
		if err != nil || v.PAGEMODEGroupFallback.Known == nil || r.Position() != 184 {
			t.Fatalf("sentinel consumed: %d %v", r.Position(), err)
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(8)
		if err != nil || sentinel != 0xa5 {
			t.Fatalf("following sentinel %x: %v", sentinel, err)
		}
	}
	// A truncated distribution prefix chooses the printed Distribution part error;
	// arbitrary received error bits are retained for replay, never made typed data.
	for end := 14; end < 18; end++ {
		r, c := psi3Reader(t, raw, end, false)
		v, err := subject.DecodePSI3MessageContentFrom(r)
		if err == nil {
			report := c.TakeReceiverReport()
			if v.PAGEMODEGroupFallback.Ignored == nil || len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || r.Position() != end {
				t.Fatalf("missing located receiver region at %d: %+v", end, report)
			}
		}
	}
}
func FuzzClause11_2_20Replay(f *testing.F) {
	for _, v := range psi3Vectors(f) {
		if v.FullBlock {
			f.Add(psi3Bytes(v.Bits))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 || raw[0]>>6 != 1 || (raw[1]>>2) != 51 {
			return
		}
		r, _ := psi3Reader(t, raw, 184, true)
		v, err := subject.DecodePSI3MessageContentFrom(r)
		if err != nil {
			return
		}
		sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		if got := psi3Encode(t, raw, 184, sealed, v); !bytes.Equal(got, raw) {
			t.Fatalf("replay %x != %x", got, raw)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.20: the distribution error arm owns the
// malformed content after the external selector, through the exact block end.
func TestClause11_2_20MalformedControlBlock(t *testing.T) {
	raw := psi3Bytes(psi3Vectors(t)[0].Bits)
	raw[43/8] &^= 1 << uint(7-43%8)
	r, c := psi3Reader(t, raw, 184, true)
	v, err := subject.DecodePSI3MessageContentFrom(r)
	report := c.TakeReceiverReport()
	if err != nil || v.PAGEMODEGroupFallback.Known != nil || v.PAGEMODEGroupFallback.Ignored == nil {
		t.Fatalf("malformed content: %+v %v", v, err)
	}
	region := v.PAGEMODEGroupFallback.Ignored
	if r.Position() != 184 || region.Label != runtime.DistributionPartError || region.Location.BitOffset != 14 || region.Bits.BitLength != 170 || len(report.Regions) != 1 {
		t.Fatalf("distribution region %+v; report %+v; position %d", region, report, r.Position())
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if got := psi3Encode(t, raw, 184, wire, v); !bytes.Equal(got, raw) {
		t.Fatalf("error-region replay %x != %x", got, raw)
	}
	_, err = subject.EncodePSI3MessageContentCanonicalAtLength(v, 23)
	refused := false
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if op, ok := cause.(*runtime.OperationError); ok && op.Kind == runtime.InvalidValue {
			detail := op.Detail
			if op.Cause != nil {
				detail = op.Cause.Error()
			}
			if detail == "receiver error region has no canonical sender encoding" {
				refused = true
			}
		}
	}
	if !refused {
		t.Fatalf("fresh sender accepted a receiver error region: %v", err)
	}
}

func TestClause11_2_20NestedEdit(t *testing.T) {
	for _, tc := range psi3Vectors(t) {
		if tc.Name != "Cell Selection struct//0/maximum" {
			continue
		}
		raw := psi3Bytes(tc.Bits)
		r, _ := psi3Reader(t, raw, 184, true)
		v, err := subject.DecodePSI3MessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		cv := runtime.Canonical(v)
		k := cv.PAGEMODEGroupFallback.Known
		if k == nil || len(k.NeighbourCellParameters) != 1 {
			t.Fatal("missing typed neighbour")
		}
		cell := &k.NeighbourCellParameters[0].NeighbourCellParamsStruct.CellSelectionParams
		cell.BSIC ^= 1
		want := cell.BSIC
		target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
		encoded := psi3Encode(t, raw, 184, target, cv)
		if bytes.Equal(encoded, raw) {
			t.Fatal("nested edit did not change the wire")
		}
		er, _ := psi3Reader(t, encoded, 184, true)
		ev, err := subject.DecodePSI3MessageContentFrom(er)
		if err != nil {
			t.Fatal(err)
		}
		if ev.PAGEMODEGroupFallback.Known == nil || len(ev.PAGEMODEGroupFallback.Known.NeighbourCellParameters) != 1 || ev.PAGEMODEGroupFallback.Known.NeighbourCellParameters[0].NeighbourCellParamsStruct.CellSelectionParams.BSIC != want {
			t.Fatalf("nested BSIC edit: %+v", ev)
		}
		return
	}
	t.Fatal("missing nested-edit source vector")
}
