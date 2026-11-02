package celldata_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/celldata"
	"github.com/gomaja/go-csn1/ts44060/padding"
)

// TS 44.060 V19.0.0 §11.2.17c pp16739–16743, 16749–16750;
// The reviewed reading assigns only complete remaining octets to CONTAINER_DATA.
func TestClause11217cWholeOctetsAtEveryPhase(t *testing.T) {
	for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		for remaining := 0; remaining <= 71; remaining++ {
			t.Run(fmt.Sprintf("phase%d/remaining%d", phase, remaining), func(t *testing.T) {
				prefix := fmt.Sprintf("%0*b", phase, 0)
				if phase == 0 {
					prefix = ""
				}
				prefix += "10"
				count := remaining / 8
				want := make([]byte, count)
				for i := range want {
					want[i] = byte(i*37 + 0xa5)
					prefix += fmt.Sprintf("%08b", want[i])
				}
				end := phase + 2 + remaining
				bits := servingSIPad(prefix, end) + "101"
				r, old := servingSIReader(t, bits, phase, 2+remaining)
				v, err := celldata.DecodePacketServingCellSIMessageContentFrom(r)
				if phase%8 != 6 {
					var phaseError *celldata.ContentPhaseError
					if !errors.As(err, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || r.Position() != phase {
						t.Fatalf("reader phase %d changed cursor or error: %v", phase, err)
					}
					target := runtime.NewWriter()
					if e := target.WriteUint(0, phase); e != nil {
						t.Fatal(e)
					}
					writerErr := celldata.EncodePacketServingCellSIMessageContentTo(target, v)
					if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || target.Position() != phase {
						t.Fatalf("writer phase %d changed cursor or error: %v", phase, writerErr)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				known := v.PAGEMODEGroupFallback.Known
				if known == nil || known.PAGEMODE != 2 || !bytes.Equal(known.CONTAINERDATA, want) {
					t.Fatalf("whole octets %+v, want %x", known, want)
				}
				if r.Position() != end {
					t.Fatalf("end %d want %d", r.Position(), end)
				}
				if err = r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				if x, e := r.ReadUint(3); e != nil || x != 5 {
					t.Fatalf("sentinel %d: %v", x, e)
				}
				raw := bits[:end]
				wire, err := runtime.SealBits(v, runtime.BitString{Bytes: servingSIBits(raw), BitLength: end}, end, runtime.BitString{}, r.Wire())
				if err != nil {
					t.Fatal(err)
				}
				w := runtime.NewWriter()
				w.WithWire(wire)
				if err = w.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				if err = celldata.EncodePacketServingCellSIMessageContentTo(w, v); err != nil {
					t.Fatal(err)
				}
				servingSIFinish(t, w, raw)
				fresh := runtime.Canonical(v)
				w = runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, end).Wire)
				if _, err = w.PushLimit(end); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				if err = celldata.EncodePacketServingCellSIMessageContentTo(w, fresh); err != nil {
					t.Fatal(err)
				}
				servingSIFinish(t, w, raw)
			})
		}
	}
}

// §11.2.0.1 p13386 supplies the external selector 100000. At origin 14,
// PAGE_MODE leaves 168 bits: exactly 21 CONTAINER_DATA octets and no padding.
func TestClause11217cCompleteBlocksAndEdits(t *testing.T) {
	for page := 0; page < 4; page++ {
		bits := "01000000" + "100000" + fmt.Sprintf("%02b", page)
		for i := 0; i < 21; i++ {
			bits += fmt.Sprintf("%08b", byte(i*11+3))
		}
		r, _ := servingSIReader(t, bits, 14, 170)
		v, err := celldata.DecodePacketServingCellSIMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		if v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Known.PAGEMODE != uint8(page) || len(v.PAGEMODEGroupFallback.Known.CONTAINERDATA) != 21 {
			t.Fatal("complete-block fields lost")
		}
		fresh := runtime.Canonical(v)
		fresh.PAGEMODEGroupFallback.Known.CONTAINERDATA[20] = 0xfe
		w := runtime.NewWriter()
		if _, err = w.PushLimit(184); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(0x1020, 14); err != nil {
			t.Fatal(err)
		}
		if err = celldata.EncodePacketServingCellSIMessageContentTo(w, fresh); err != nil {
			t.Fatal(err)
		}
		servingSIFinish(t, w, bits[:176]+"11111110")
		// The byte API translates its 22-octet extent to origin 14, leaving 21
		// data octets and six padding bits, never a 22nd data octet.
		fresh.PAGEMODEGroupFallback.Known.PaddingBits = padding.PaddingBits{IgnoreChoice: padding.PaddingBitsIgnoreChoice{
			Alternative: padding.PaddingBitsIgnoreChoiceAlternativeIgnore,
			Ignore:      &padding.PaddingBitsIgnoreChoiceIgnore{Alternative: padding.PaddingBitsIgnoreChoiceIgnoreAlternativeKnown, Known: &padding.PaddingBitsIgnoreChoiceIgnoreKnown{}},
		}}
		canonical, err := celldata.EncodePacketServingCellSIMessageContentCanonicalAtLength(fresh, 22)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := celldata.DecodePacketServingCellSIMessageContent(canonical)
		if err != nil {
			t.Fatal(err)
		}
		if len(decoded.Value.PAGEMODEGroupFallback.Known.CONTAINERDATA) != 21 {
			t.Fatal("byte completion counted as data")
		}
		replay, err := celldata.EncodePacketServingCellSIMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(replay, canonical) {
			t.Fatal("byte API replay mismatch")
		}
		if _, err = celldata.EncodePacketServingCellSIMessageContentCanonical(fresh); err == nil {
			t.Fatal("missing containing extent accepted")
		}
		fresh.PAGEMODEGroupFallback.Known.CONTAINERDATA = append(fresh.PAGEMODEGroupFallback.Known.CONTAINERDATA, 0)
		if _, err = celldata.EncodePacketServingCellSIMessageContentCanonicalAtLength(fresh, 22); err == nil {
			t.Fatal("partial final octet accepted as data")
		}
	}
}
func TestClause11217cMissingExtentAndGrowingInput(t *testing.T) {
	unbounded := runtime.NewChoiceReader([]byte{0, 0, 0}, nil)
	if _, err := unbounded.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	if _, err := celldata.DecodePacketServingCellSIMessageContentFrom(unbounded); err == nil {
		t.Fatal("missing containing extent accepted")
	}
	for _, n := range []int{0, 1} {
		r, err := runtime.NewChoiceReaderBits([]byte{0, 0}, 14+n, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		v, err := celldata.DecodePacketServingCellSIMessageContentFrom(r)
		if err != nil || v.PAGEMODEGroupFallback.Ignored == nil || v.PAGEMODEGroupFallback.Ignored.Label != runtime.DistributionPartError || v.PAGEMODEGroupFallback.Ignored.Bits.BitLength != n {
			t.Fatalf("truncated PAGE_MODE %d lost its printed receiver arm: %v", n, err)
		}
	}
	for _, n := range []int{8, 64, 512, 4096} {
		data := make([]byte, n)
		decoded, err := celldata.DecodePacketServingCellSIMessageContent(data)
		if err != nil {
			t.Fatal(err)
		}
		known := decoded.Value.PAGEMODEGroupFallback.Known
		if known == nil || len(known.CONTAINERDATA) != n-1 {
			t.Fatalf("data count at %d bytes", n)
		}
		replay, err := celldata.EncodePacketServingCellSIMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(replay, data) {
			t.Fatalf("growing replay %d: %v", n, err)
		}
		allocs := testing.AllocsPerRun(1, func() {
			if _, err := celldata.DecodePacketServingCellSIMessageContent(data); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && allocs > float64(n*8+512) {
			t.Fatalf("nonlinear allocation count at %d: %f", n, allocs)
		}
	}
}
func FuzzClause11217cContent(f *testing.F) {
	f.Add([]byte{0, 0, 0}, uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, unused uint8) {
		if len(data) > 4096 {
			return
		}
		n := len(data) * 8
		if n > 0 {
			n -= int(unused % 8)
		}
		shifted := make([]byte, (14+n+7)/8)
		for i := 0; i < n; i++ {
			shifted[(14+i)/8] |= ((data[i/8] >> uint(7-i%8)) & 1) << uint(7-(14+i)%8)
		}
		data = shifted
		n += 14
		r, err := runtime.NewChoiceReaderBits(data, n, nil)
		if err != nil {
			return
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		v, err := celldata.DecodePacketServingCellSIMessageContentFrom(r)
		if err != nil {
			return
		}
		if r.Position() != n {
			t.Fatal("remainder extent")
		}
		if v.PAGEMODEGroupFallback.Known == nil {
			return
		}
		wire, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: n}, n, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err = w.WriteUint(0, 14); err != nil {
			t.Fatal(err)
		}
		if err = celldata.EncodePacketServingCellSIMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		if out.BitLength != n {
			t.Fatal("replay extent")
		}
		for i := 0; i < n; i++ {
			if (out.Bytes[i/8]>>uint(7-i%8))&1 != (data[i/8]>>uint(7-i%8))&1 {
				t.Fatal("replay bits")
			}
		}
	})
}

// §11.2.17c p16743 and §11 p12938 retain receiver error labels;
// §§11.1.2–11.1.3 select them from wire syntax; collectors add diagnostics.
func TestClause11217cReceiverRegions(t *testing.T) {
	for _, body := range []string{"1", "10" + "10100101" + "11111"} {
		prefix := "01000000" + "100000"
		bits := prefix + body
		strict, _ := servingSIReader(t, bits, 14, len(body))
		plain, err := celldata.DecodePacketServingCellSIMessageContentFrom(strict)
		if err != nil {
			t.Fatal(err)
		}
		if len(body) == 1 && plain.PAGEMODEGroupFallback.Ignored == nil {
			t.Fatal("unobserved distribution arm lost")
		}
		if len(body) != 1 && plain.PAGEMODEGroupFallback.Known == nil {
			t.Fatal("unobserved typed prefix lost")
		}
		collector := runtime.NewReceiverCollector()
		r, err := runtime.NewChoiceReaderBits(servingSIBits(bits+"101"), len(bits)+3, collector)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(len(body))
		if err != nil {
			t.Fatal(err)
		}
		v, err := celldata.DecodePacketServingCellSIMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		report := collector.TakeReceiverReport()
		if len(report.Regions) != 1 {
			t.Fatalf("receiver report %+v", report)
		}
		if len(body) == 1 {
			if v.PAGEMODEGroupFallback.Ignored == nil || report.Regions[0].Label != runtime.ErrorLabel("Distribution part error") || report.Regions[0].Location.BitOffset != 14 || report.Regions[0].Location.Width != 1 {
				t.Fatal("distribution region lost")
			}
		} else {
			known := v.PAGEMODEGroupFallback.Known
			if known == nil || len(known.CONTAINERDATA) != 1 || known.CONTAINERDATA[0] != 0xa5 || known.PaddingBits.IgnoreChoice.Ignore == nil || known.PaddingBits.IgnoreChoice.Ignore.Ignored == nil || report.Regions[0].Label != runtime.Ignore || report.Regions[0].Location.BitOffset != 25 || report.Regions[0].Location.Width != 4 {
				t.Fatal("padding Ignore region lost")
			}
		}
		if r.Position() != len(bits) {
			t.Fatal("region extent")
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if x, e := r.ReadUint(3); e != nil || x != 5 {
			t.Fatal("region consumed sentinel")
		}
		wire, err := runtime.SealBits(v, runtime.BitString{Bytes: servingSIBits(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err = w.WriteUint(0x1020, 14); err != nil {
			t.Fatal(err)
		}
		if err = celldata.EncodePacketServingCellSIMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		servingSIFinish(t, w, bits)
		w = runtime.NewWriter()
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(bits)).Wire)
		if _, err = w.PushLimit(len(bits)); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(0x1020, 14); err != nil {
			t.Fatal(err)
		}
		if err = celldata.EncodePacketServingCellSIMessageContentTo(w, runtime.Canonical(v)); err == nil {
			t.Fatal("receiver region accepted as a canonical sender value")
		}
	}
}

func servingSIBits(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, c := range bits {
		if c == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

// TS 44.060 V19.0.0 §11 pp12937–12940; TS 24.007 V20.0.0
// Annex B §B.1.2.2: positional spare padding repeats 00101011.
func servingSIPad(prefix string, end int) string {
	if len(prefix) == end {
		return prefix
	}
	prefix += "0"
	for len(prefix) < end {
		prefix += string("00101011"[len(prefix)%8])
	}
	return prefix
}
func servingSIReader(t *testing.T, bits string, origin, extent int) (*runtime.Reader, int) {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(servingSIBits(bits), len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(extent)
	if err != nil {
		t.Fatal(err)
	}
	return r, old
}
func servingSIFinish(t *testing.T, w *runtime.Writer, want string) {
	t.Helper()
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != len(want) || !bytes.Equal(out.Bytes, servingSIBits(want)) {
		t.Fatalf("wire %x/%d want %x/%d: %v", out.Bytes, out.BitLength, servingSIBits(want), len(want), err)
	}
}
