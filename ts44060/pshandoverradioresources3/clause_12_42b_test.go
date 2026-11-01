package pshandoverradioresources3_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	subject "github.com/gomaja/go-csn1/ts44060/pshandoverradioresources3"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// TS 44.060 V19.0.0 §12.42b, authenticated paragraphs 23967–24102;
// §11.1 p12944: the reserved arm's explicit interpretation retains its remainder.
// All committed handoverResourcesVectors are synthetic source-derived inputs.
type handoverResourcesVector struct {
	Name      string
	Bits      string
	Extent    int
	Origin    int
	FullBlock bool `json:"full_block"`
	Fields    []string
	Tail      string
	Facts     []struct {
		Kind   string
		Label  string
		Value  uint64
		Offset int
		Width  int
		Clause string
	}
}

func handoverResourcesVectors(t testing.TB) []handoverResourcesVector {
	t.Helper()
	b, e := os.ReadFile("testdata/clause_vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var out []handoverResourcesVector
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func handoverResourcesBytes(s string) []byte {
	out := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func handoverResourcesFields(v any) []string {
	var out []string
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, name string) {
		if !v.IsValid() || !v.CanInterface() {
			return
		}
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if !v.IsNil() {
				walk(v.Elem(), name)
			}
			return
		}
		if _, ok := v.Interface().(runtime.ErrorRegion); ok {
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
				f := v.Type().Field(i)
				if !f.IsExported() || f.Name == "Wire" || f.Name == "Alternative" || f.Name == "PaddingBits" || (f.Name == "Ignored" && (f.Type == reflect.TypeOf(runtime.BitString{}) || f.Type == reflect.TypeOf((*runtime.BitString)(nil)))) {
					continue
				}
				n := strings.Map(func(r rune) rune {
					if unicode.IsLetter(r) || unicode.IsDigit(r) {
						return unicode.ToLower(r)
					}
					return -1
				}, strings.TrimSuffix(f.Name, "List"))
				if len(n) > 1 && n[0] == 'n' && n[1] >= '0' && n[1] <= '9' {
					n = n[1:]
				}
				if n == "downlinketfiassignment2" {
					n = "downlinketfiassignment"
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
func handoverResourcesReader(t testing.TB, tc handoverResourcesVector) (*runtime.Reader, *runtime.Collector, int) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, e := runtime.NewChoiceReaderBits(handoverResourcesBytes(tc.Bits+"110"), tc.Extent+3, c)
	if e != nil {
		t.Fatal(e)
	}
	if tc.FullBlock {
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
		old, e := r.PushLimit(tc.Extent)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); e != nil {
			t.Fatal(e)
		}
		tag, e := r.ReadUint(6)
		if e != nil || tag != 7 {
			t.Fatalf("downlink selector %d: %v", tag, e)
		}
		return r, c, old
	}
	if _, e = r.ReadUint(tc.Origin); e != nil {
		t.Fatal(e)
	}
	old, e := r.PushLimit(tc.Extent - tc.Origin)
	if e != nil {
		t.Fatal(e)
	}
	return r, c, old
}
func handoverResourcesEncode(tc handoverResourcesVector, wire runtime.WireInfo, v subject.PSHandoverRadioResources3IE) (runtime.BitString, error) {
	w := runtime.NewWriter()
	w.WithWire(wire)
	if false {
		raw := handoverResourcesBytes(tc.Bits)
		if e := w.WriteUint(uint64(raw[0]), 8); e != nil {
			return runtime.BitString{}, e
		}
		if e := w.WriteUint(7, 6); e != nil {
			return runtime.BitString{}, e
		}
	} else {
		if e := w.WriteUint(0, tc.Origin); e != nil {
			return runtime.BitString{}, e
		}
	}
	if e := subject.EncodePSHandoverRadioResources3IETo(w, v); e != nil {
		return runtime.BitString{}, e
	}
	return w.FinishBits(wire.Tail)
}
func TestClause12_42bSourceVariants(t *testing.T) {
	for _, tc := range handoverResourcesVectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			r, c, old := handoverResourcesReader(t, tc)
			v, e := subject.DecodePSHandoverRadioResources3IEFrom(r)
			if e != nil {
				t.Fatalf("typed source decode: %v", e)
			}
			if r.Position() != tc.Extent {
				t.Fatalf("extent %d want %d", r.Position(), tc.Extent)
			}
			got := handoverResourcesFields(v)
			ending := false
			for _, fact := range tc.Facts {
				if fact.Kind == "reserved" {
					ending = true
				}
			}
			match := reflect.DeepEqual(got, tc.Fields)
			if ending && len(got) >= len(tc.Fields) {
				match = reflect.DeepEqual(got[:len(tc.Fields)], tc.Fields)
				for _, rest := range got[len(tc.Fields):] {
					if !strings.HasSuffix(rest, "=0") {
						match = false
					}
				}
			}
			if !match {
				t.Fatalf("typed fields got %v want %v", got, tc.Fields)
			}
			report := c.TakeReport()
			regions := c.TakeReceiverReport()
			oi, ri := 0, 0
			for _, fact := range tc.Facts {
				switch fact.Kind {
				case "reserved":
					if oi >= len(report.Observations) {
						t.Fatal("reserved observation missing")
					}
					o := report.Observations[oi]
					oi++
					if o.Value != fact.Value || o.Location.BitOffset != fact.Offset || o.Location.Width != fact.Width || o.Location.Source.Clause != fact.Clause || o.Location.Path == "" {
						t.Fatalf("reserved extent %+v want %+v", o, fact)
					}
				case "region":
					if ri >= len(regions.Regions) {
						t.Fatal("printed receiver region missing")
					}
					o := regions.Regions[ri]
					ri++
					if string(o.Label) != fact.Label || o.Location.BitOffset != fact.Offset || o.Location.Width != fact.Width || o.Location.Source.Clause != fact.Clause {
						t.Fatalf("receiver extent %+v want %+v", o, fact)
					}
				}
			}
			if len(report.Observations) != oi || len(regions.Regions) != ri || report.Dropped != 0 || regions.Dropped != 0 {
				t.Fatalf("unexpected report %+v %+v", report, regions)
			}
			wire := r.Wire()
			if e = r.PopLimit(old); e != nil {
				t.Fatal(e)
			}
			sentinel, e := r.ReadUint(3)
			if e != nil || sentinel != 6 {
				t.Fatalf("containing extent ate sentinel %d: %v", sentinel, e)
			}
			raw := handoverResourcesBytes(tc.Bits)
			sealed, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, wire)
			if e != nil {
				t.Fatal(e)
			}
			out, e := handoverResourcesEncode(tc, sealed, v)
			if e != nil || out.BitLength != tc.Extent || !bytes.Equal(out.Bytes, raw) {
				t.Fatalf("received replay extent %d want %d: %v", out.BitLength, tc.Extent, e)
			}

			edited := v
			edited.BSIC ^= 1
			changed, err := handoverResourcesEncode(tc, sealed, edited)
			if err != nil {
				t.Fatal(err)
			}
			changedCase := tc
			var changedBits strings.Builder
			for i := 0; i < changed.BitLength; i++ {
				if changed.Bytes[i/8]&(1<<uint(7-i%8)) != 0 {
					changedBits.WriteByte('1')
				} else {
					changedBits.WriteByte('0')
				}
			}
			changedCase.Bits = changedBits.String()
			er, _, _ := handoverResourcesReader(t, changedCase)
			ev, err := subject.DecodePSHandoverRadioResources3IEFrom(er)
			if err != nil || ev.BSIC != edited.BSIC {
				t.Fatalf("accepted BSIC edit: %v", err)
			}
			if oi+ri > 0 {
				if _, e = handoverResourcesEncode(tc, runtime.Canonical(struct{ Wire runtime.WireInfo }{}).Wire, runtime.Canonical(v)); e == nil {
					t.Fatal("receiver-only arm sent canonically")
				}
				v.Wire = runtime.WireInfo{}
				if _, e = handoverResourcesEncode(tc, runtime.WireInfo{}, v); e == nil {
					t.Fatal("receiver-only arm sent fresh")
				}
			} else if tc.Tail != "ignored" {
				target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire
				out, e := handoverResourcesEncode(tc, target, runtime.Canonical(v))
				if e != nil || out.BitLength != tc.Extent || !bytes.Equal(out.Bytes, raw) {
					t.Fatalf("canonical source arm extent %d: %v", out.BitLength, e)
				}
			}
		})
	}
}
func FuzzClause12_42bReplay(f *testing.F) {
	for _, tc := range handoverResourcesVectors(f) {
		if tc.FullBlock || (!false && tc.Origin == 0) {
			f.Add(handoverResourcesBytes(tc.Bits), uint16(tc.Extent))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte, length uint16) {
		if len(raw) > 512 {
			return
		}
		bits := int(length)
		if bits > len(raw)*8 {
			return
		}
		if false && (bits != 184 || len(raw) != 23 || raw[0]>>6 != 1 || raw[1]>>2 != 7) {
			return
		}
		tc := handoverResourcesVector{Extent: bits, Origin: 0, FullBlock: false}
		if false {
			tc.Origin = 14
		}
		var s strings.Builder
		for i := 0; i < bits; i++ {
			if raw[i/8]&(1<<uint(7-i%8)) != 0 {
				s.WriteByte('1')
			} else {
				s.WriteByte('0')
			}
		}
		tc.Bits = s.String()
		r, _, _ := handoverResourcesReader(t, tc)
		v, e := subject.DecodePSHandoverRadioResources3IEFrom(r)
		if e != nil {
			return
		}
		// A normal IE may leave following containing fields unread; retain
		// those bits in the caller's tail when replaying the complete image.
		if r.Position() < tc.Origin || r.Position() > bits {
			t.Fatal("invalid typed extent")
		}
		if v.ReceiverRegionEnds() && r.Position() != bits {
			t.Fatal("receiver remainder left its containing extent")
		}
		consumed := r.Position()
		tail := runtime.BitString{Bytes: handoverResourcesBytes(tc.Bits[consumed:]), BitLength: bits - consumed}
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: handoverResourcesBytes(tc.Bits), BitLength: bits}, consumed, tail, r.Wire())
		if e != nil {
			t.Fatal(e)
		}
		out, e := handoverResourcesEncode(tc, wire, v)
		if e != nil || out.BitLength != bits || !bytes.Equal(out.Bytes, handoverResourcesBytes(tc.Bits)) {
			t.Fatalf("received replay failed: %v", e)
		}
	})
}

// TS 44.060 V19.0.0 §12.42b, SI table: code 11 is reserved
// without an alternative receiver interpretation (§11.1 p12944).
func TestHandoverResourcesReservedSI(t *testing.T) {
	tc := handoverResourcesVectors(t)[0]
	if tc.Origin != 0 || tc.Bits[0] != '0' {
		t.Fatal("expected source arm without Handover Reference")
	}
	raw := handoverResourcesBytes(tc.Bits)
	raw[0] |= 0x60
	for _, collector := range []*runtime.Collector{nil, runtime.NewReceiverCollector()} {
		r, err := runtime.NewReaderBits(raw, tc.Extent, collector)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = subject.DecodePSHandoverRadioResources3IEFrom(r); err == nil {
			t.Fatal("reserved SI decoded")
		}
	}
	r, _, _ := handoverResourcesReader(t, tc)
	v, err := subject.DecodePSHandoverRadioResources3IEFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	v.SI = 3
	if err = subject.EncodePSHandoverRadioResources3IETo(runtime.NewWriter(), runtime.Canonical(v)); err == nil {
		t.Fatal("reserved SI sent")
	}
}
