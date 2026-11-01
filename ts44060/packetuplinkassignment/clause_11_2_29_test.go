package packetuplinkassignment_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	subject "github.com/gomaja/go-csn1/ts44060/packetuplinkassignment"
)

// TS 44.060 V19.0.0 §11.2.29, authenticated paragraphs 18110–18406;
// §11.1 p12944: the reserved arm's explicit interpretation retains its remainder.
// All committed vectors are synthetic source-derived inputs.
type sourceVector struct {
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

func vectors(t testing.TB) []sourceVector {
	t.Helper()
	b, e := os.ReadFile("testdata/clause_vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var out []sourceVector
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func pack(s string) []byte {
	out := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func sourceFields(v any) []string {
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
func reader(t testing.TB, tc sourceVector) (*runtime.Reader, *runtime.Collector, int) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, e := runtime.NewChoiceReaderBits(pack(tc.Bits+"110"), tc.Extent+3, c)
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
		if e != nil || tag != 10 {
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
func encode(tc sourceVector, wire runtime.WireInfo, v subject.PacketUplinkAssignmentMessageContent) (runtime.BitString, error) {
	w := runtime.NewWriter()
	w.WithWire(wire)
	if true {
		raw := pack(tc.Bits)
		if e := w.WriteUint(uint64(raw[0]), 8); e != nil {
			return runtime.BitString{}, e
		}
		if e := w.WriteUint(10, 6); e != nil {
			return runtime.BitString{}, e
		}
	} else {
		if e := w.WriteUint(0, tc.Origin); e != nil {
			return runtime.BitString{}, e
		}
	}
	if e := subject.EncodePacketUplinkAssignmentMessageContentTo(w, v); e != nil {
		return runtime.BitString{}, e
	}
	return w.FinishBits(runtime.BitString{})
}
func TestSourceVariants(t *testing.T) {
	for _, tc := range vectors(t) {
		t.Run(tc.Name, func(t *testing.T) {
			r, c, old := reader(t, tc)
			v, e := subject.DecodePacketUplinkAssignmentMessageContentFrom(r)
			if e != nil {
				t.Fatalf("typed source decode: %v", e)
			}
			if r.Position() != tc.Extent {
				t.Fatalf("extent %d want %d", r.Position(), tc.Extent)
			}
			got := sourceFields(v)
			ending := false
			for _, fact := range tc.Facts {
				if fact.Kind == "reserved" || fact.Kind == "region" {
					ending = true
				}
			}
			match := slices.Equal(got, tc.Fields)
			if ending && len(got) >= len(tc.Fields) {
				match = slices.Equal(got[:len(tc.Fields)], tc.Fields)
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
			raw := pack(tc.Bits)
			sealed, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, wire)
			if e != nil {
				t.Fatal(e)
			}
			out, e := encode(tc, sealed, v)
			if e != nil || out.BitLength != tc.Extent || !bytes.Equal(out.Bytes, raw) {
				t.Fatalf("received replay extent %d want %d: %v", out.BitLength, tc.Extent, e)
			}
			if oi+ri > 0 {
				if _, e = encode(tc, runtime.Canonical(struct{ Wire runtime.WireInfo }{}).Wire, runtime.Canonical(v)); e == nil {
					t.Fatal("receiver-only arm sent canonically")
				}
				v.Wire = runtime.WireInfo{}
				if _, e = encode(tc, runtime.WireInfo{}, v); e == nil {
					t.Fatal("receiver-only arm sent fresh")
				}
			} else if tc.Tail != "ignored" {
				target := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, tc.Extent).Wire
				out, e := encode(tc, target, runtime.Canonical(v))
				if e != nil || out.BitLength != tc.Extent || !bytes.Equal(out.Bytes, raw) {
					t.Fatalf("canonical source arm extent %d: %v", out.BitLength, e)
				}
			}
		})
	}
}
func FuzzSourceReplay(f *testing.F) {
	for _, tc := range vectors(f) {
		f.Add(pack(tc.Bits), uint16(tc.Extent))
	}
	f.Fuzz(func(t *testing.T, raw []byte, length uint16) {
		if len(raw) > 512 {
			return
		}
		bits := int(length)
		if bits > len(raw)*8 {
			return
		}
		if true && (bits < 14 || bits > 184 || len(raw) < 2 || raw[0] != 0x40 || raw[1]>>2 != 10) {
			return
		}
		tc := sourceVector{Extent: bits, Origin: 0, FullBlock: true}
		if true {
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
		r, e := runtime.NewChoiceReaderBits(pack(tc.Bits), bits, nil)
		if e != nil {
			return
		}
		if _, e = r.ReadUint(14); e != nil {
			return
		}
		if _, e = r.PushLimit(bits - 14); e != nil {
			return
		}
		v, e := subject.DecodePacketUplinkAssignmentMessageContentFrom(r)
		if e != nil {
			return
		}
		if r.Position() != bits {
			t.Fatalf("accepted receiver extent %d/%d", r.Position(), bits)
		}
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: pack(tc.Bits), BitLength: bits}, bits, runtime.BitString{}, r.Wire())
		if e != nil {
			t.Fatal(e)
		}
		out, e := encode(tc, wire, v)
		if e != nil || out.BitLength != bits || !bytes.Equal(out.Bytes, pack(tc.Bits)) {
			t.Fatalf("received replay failed: %v", e)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.29 pp18110–18113: PAGE_MODE is
// the first two content bits, after the separately owned six-bit selector.
func TestPacketUplinkAssignmentEditsAndPrefixes(t *testing.T) {
	for _, tc := range vectors(t) {
		if !tc.FullBlock {
			continue
		}
		r, _, _ := reader(t, tc)
		v, err := subject.DecodePacketUplinkAssignmentMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		known := v.PAGEMODEGroupFallback.Known
		if known == nil {
			t.Fatal("synthetic PAGE_MODE prefix selected a receiver error region")
		}
		raw := pack(tc.Bits)
		sealed, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: tc.Extent}, tc.Extent, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		changed := *known
		changed.PAGEMODE = (changed.PAGEMODE + 1) % 4
		v.PAGEMODEGroupFallback.Known = &changed
		out, err := encode(tc, sealed, v)
		expected := append([]byte(nil), raw...)
		expected[1] = (expected[1] & 0xfc) | changed.PAGEMODE
		if err != nil || out.BitLength != tc.Extent || !bytes.Equal(out.Bytes, expected) {
			t.Fatalf("PAGE_MODE edit %s: %v", tc.Name, err)
		}
		short, err := runtime.NewChoiceReaderBits(raw, 15, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = short.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		if _, err = short.PushLimit(1); err != nil {
			t.Fatal(err)
		}
		partial, err := subject.DecodePacketUplinkAssignmentMessageContentFrom(short)
		if err == nil && partial.PAGEMODEGroupFallback.Known != nil {
			t.Fatalf("cut inside PAGE_MODE populated the known arm: %s", tc.Name)
		}
	}
}

// TS 44.060 V19.0.0 §11.2.29 defines Access Technologies Request
// as four identifier bits and a continuation bit per entry. The standalone
// helper has no printed numerical carrier ceiling; callers bound its input.
func TestAssignmentAccessTechnologyGrowth(t *testing.T) {
	for _, count := range []int{8, 64, 512, 4096} {
		w := runtime.NewWriter()
		for i := 0; i < count; i++ {
			if err := w.WriteUint(uint64(i%16), 4); err != nil {
				t.Fatal(err)
			}
			more := uint64(0)
			if i+1 < count {
				more = 1
			}
			if err := w.WriteUint(more, 1); err != nil {
				t.Fatal(err)
			}
		}
		raw, err := w.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := subject.DecodeAccessTechnologiesRequestStruct(raw)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.BitsConsumed != count*5 || len(decoded.Value.Entries) != count {
			t.Fatalf("%d-entry extent or count changed", count)
		}
		for i, v := range decoded.Value.Entries {
			if v.AccessTechnologyType != uint8(i%16) {
				t.Fatalf("entry %d changed", i)
			}
		}
		out, err := subject.EncodeAccessTechnologiesRequestStruct(decoded.Value)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("%d-entry replay: %v", count, err)
		}
		out, err = subject.EncodeAccessTechnologiesRequestStructCanonicalAtLength(runtime.Canonical(decoded.Value), len(raw))
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("%d-entry canonical: %v", count, err)
		}
		allocations := testing.AllocsPerRun(1, func() {
			if _, err := subject.DecodeAccessTechnologiesRequestStruct(raw); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && allocations > float64(128+count*16) {
			t.Fatalf("%d entries allocated %.0f times, exceeding the linear bound", count, allocations)
		}
	}
}
