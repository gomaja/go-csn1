package dtmhandover

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"os"
	goruntime "runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

type clauseVector struct {
	Name, Bits   string
	N, M         int
	Ms           []int
	Downlinks    int
	OracleCounts bool `json:"oracle_counts"`
}

func vectors(t testing.TB) []clauseVector {
	t.Helper()
	data, e := os.ReadFile("testdata/handover_vectors.json")
	if e != nil {
		t.Fatal(e)
	}
	var v []clauseVector
	if e = json.Unmarshal(data, &v); e != nil {
		t.Fatal(e)
	}
	return v
}
func packed(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}
func written(t testing.TB, v DTMHandoverPSRadioResourcesIE, phase, extent int, wire runtime.WireInfo) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(0, phase); e != nil {
		t.Fatal(e)
	}
	if e := EncodeDTMHandoverPSRadioResourcesIETo(w, v); e != nil {
		t.Fatal(e)
	}
	b, e := w.FinishBits(runtime.BitString{})
	if e != nil || b.BitLength != extent {
		t.Fatalf("encoded %d want %d: %v", b.BitLength, extent, e)
	}
	return b.Bytes
}
func assignments(v DTMHandoverPSRadioResourcesIE) []UplinkTBFAssignmentStruct {
	var a []UplinkTBFAssignmentStruct
	if g := v.GPRSModeChoice.GPRSMode; g != nil {
		if global := g.GPRSMode.GlobalTimeslotDescriptionGroup; global != nil {
			for _, x := range global.UplinkAssignmentList {
				a = append(a, x.UplinkAssignment)
			}
		}
	}
	if g := v.GPRSModeChoice.EGPRSMode; g != nil {
		if global := g.EGPRSMode.Group.GlobalTimeslotDescriptionGroup; global != nil {
			for _, x := range global.UplinkAssignmentList {
				a = append(a, x.UplinkAssignment)
			}
		}
	}
	return a
}

func downlinkCount(v DTMHandoverPSRadioResourcesIE) int {
	if g := v.GPRSModeChoice.GPRSMode; g != nil {
		return len(g.GPRSMode.DownlinkAssignmentList)
	}
	if g := v.GPRSModeChoice.EGPRSMode; g != nil && g.EGPRSMode.EGPRSWindowSizeGroupGroup != nil {
		return len(g.EGPRSMode.EGPRSWindowSizeGroupGroup.DownlinkAssignmentList)
	}
	return 0
}

// TS 44.060 V19.0.0 §12.46 pp24357–24451. Each standalone IE is
// bounded by its caller. Its unpublished containing message supplies no
// numerical standalone maximum, and the origin retains the bit phase.
func TestClause1246Standalone(t *testing.T) {
	for _, tc := range vectors(t) {
		for phase := 0; phase < 8; phase++ {
			t.Run(fmt.Sprintf("%s/phase%d", tc.Name, phase), func(t *testing.T) {
				bits := strings.Repeat("0", phase) + tc.Bits
				extent := len(bits)
				raw := packed(bits)
				full := packed(bits + "101")
				collector := runtime.NewReceiverCollector()
				rd, e := runtime.NewChoiceReaderBits(full, extent+3, collector)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = rd.ReadUint(phase); e != nil {
					t.Fatal(e)
				}
				old, e := rd.PushLimit(len(tc.Bits))
				if e != nil {
					t.Fatal(e)
				}
				v, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rd)
				if e != nil {
					t.Fatal(e)
				}
				if rd.Position() != extent {
					t.Fatalf("consumed %d want %d", rd.Position(), extent)
				}
				if v.RLCRESET != 1 || v.EXTENDEDDYNAMICALLOCATION != 1 {
					t.Fatalf("typed header %+v", v)
				}
				if count := downlinkCount(v); count != tc.Downlinks {
					t.Fatalf("downlink TBFs %d want %d", count, tc.Downlinks)
				}
				if tc.OracleCounts {
					a := assignments(v)
					if len(a) != 2 {
						t.Fatalf("TBF count %d", len(a))
					}
					for index, x := range a {
						wantM := tc.M
						if len(tc.Ms) > 0 {
							wantM = tc.Ms[index]
						}
						if x.RLCMODE != 1 || x.TFIAssignment != 21 || x.USFGRANULARITY != 1 {
							t.Fatalf("typed TBF %+v", x)
						}
						if x.TBFTIMESLOTALLOCATION != nil && x.TBFTIMESLOTALLOCATION.BitLength != tc.N {
							t.Fatalf("N=%d want %d", x.TBFTIMESLOTALLOCATION.BitLength, tc.N)
						}
						if usf := x.USFALLOCATIONChoice.Alt1; usf != nil {
							if usf.USFALLOCATION != 5 || len(usf.USFALLOCATIONList) != wantM-1 {
								t.Fatalf("M=%d want %d", len(usf.USFALLOCATIONList)+1, wantM)
							}
						} else if x.USFALLOCATIONChoice.USFALLOCATION == nil || x.USFALLOCATIONChoice.USFALLOCATION.USFALLOCATION != 5 {
							t.Fatal("lost USF")
						}
					}
				}
				wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, rd.Wire())
				if e != nil {
					t.Fatal(e)
				}
				got := written(t, v, phase, extent, wire)
				if !bytes.Equal(got, raw) {
					t.Fatalf("replay %x want %x", got, raw)
				}
				cv := runtime.Canonical(v)
				cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire
				got = written(t, cv, phase, extent, cw)
				if !bytes.Equal(got, raw) {
					t.Fatalf("canonical %x want %x", got, raw)
				}
				cv.MAXLAPDm = 3
				edited := written(t, cv, phase, extent, cw)
				er, _ := runtime.NewChoiceReaderBits(edited, extent, nil)
				_, _ = er.ReadUint(phase)
				ev, e := DecodeDTMHandoverPSRadioResourcesIEFrom(er)
				if e != nil || ev.MAXLAPDm != 3 {
					t.Fatalf("edit %+v %v", ev, e)
				}
				if e = rd.PopLimit(old); e != nil {
					t.Fatal(e)
				}
				sentinel, e := rd.ReadUint(3)
				if e != nil || sentinel != 5 {
					t.Fatalf("sentinel %d: %v", sentinel, e)
				}
				if report := collector.TakeReceiverReport(); len(report.Regions) != 0 || len(report.ZeroFill) != 0 {
					t.Fatalf("unexpected report %+v", report)
				}
			})
		}
	}
}
func FuzzClause1246(f *testing.F) {
	seeds := vectors(f)
	for i, tc := range seeds {
		if i%17 != 0 {
			continue
		}
		for phase := 0; phase < 8; phase++ {
			bits := strings.Repeat("0", phase) + tc.Bits
			f.Add(packed(bits), uint16(len(bits)), uint8(phase))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, bitlen uint16, origin uint8) {
		if len(data) > 4096 {
			return
		}
		n := int(bitlen)
		if n > len(data)*8 {
			return
		}
		phase := int(origin % 8)
		if phase > n {
			return
		}
		rd, e := runtime.NewChoiceReaderBits(data, n, nil)
		if e != nil {
			return
		}
		prefix, e := rd.ReadUint(phase)
		if e != nil {
			return
		}
		v, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rd)
		if e != nil {
			return
		}
		used := rd.Position()
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: used}, used, runtime.BitString{}, rd.Wire())
		if e != nil {
			t.Fatal(e)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if _, e = w.PushLimit(n); e != nil {
			t.Fatal(e)
		}
		if e = w.WriteUint(prefix, phase); e != nil {
			t.Fatal(e)
		}
		if e = EncodeDTMHandoverPSRadioResourcesIETo(w, v); e != nil {
			t.Fatal(e)
		}
		got, e := w.FinishBits(runtime.BitString{})
		if e != nil || got.BitLength != used {
			t.Fatalf("replay length %d want %d: %v", got.BitLength, used, e)
		}
		want := bytes.Clone(data[:(used+7)/8])
		if used%8 != 0 {
			want[len(want)-1] &= byte(0xff << uint(8-used%8))
		}
		if !bytes.Equal(got.Bytes, want) {
			t.Fatalf("replay %x want %x", got.Bytes, want)
		}
		c := runtime.NewReceiverCollector()
		rr, _ := runtime.NewChoiceReaderBits(data, n, c)
		_, _ = rr.ReadUint(phase)
		reported, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rr)
		if e != nil {
			t.Fatalf("report parity: %v", e)
		}
		plainJSON, _ := json.Marshal(v)
		reportJSON, _ := json.Marshal(reported)
		if !bytes.Equal(plainJSON, reportJSON) || rr.Position() != used {
			t.Fatal("report-enabled decoder changed typed values or extent")
		}
	})
}

// TS 44.060 V19.0.0 §12.46 p24378: only
// the authenticated Rel-12 suffix infers absent zeros. A received value
// replays the transmitted prefix; a fresh sender includes the inferred fields.
func TestClause1246ZeroFill(t *testing.T) {
	prefix := strings.Repeat("0", 110) + "1100" + "0000"
	suffix := "11110111110"
	for phase := 0; phase < 8; phase++ {
		for cut := 1; cut <= len(suffix); cut++ {
			t.Run(fmt.Sprintf("phase%d/cut%d", phase, cut), func(t *testing.T) {
				bits := strings.Repeat("0", phase) + prefix + suffix[:cut]
				raw := packed(bits)
				c := runtime.NewReceiverCollector()
				rd, e := runtime.NewChoiceReaderBits(raw, len(bits), c)
				if e != nil {
					t.Fatal(e)
				}
				_, _ = rd.ReadUint(phase)
				v, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rd)
				if e != nil {
					t.Fatal(e)
				}
				fills := 0
				for _, loc := range c.TakeReceiverReport().ZeroFill {
					fills += loc.Width
					if loc.Source.Clause != "12.46" {
						t.Fatalf("fill source %+v", loc)
					}
				}
				want := 11 - cut
				if cut == 1 {
					want = 2
				} else if cut <= 6 {
					want = 7 - cut
				}
				if fills != want || rd.Wire().ImplicitZeros != want {
					t.Fatalf("filled %d wire %d want %d", fills, rd.Wire().ImplicitZeros, want)
				}
				wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: len(bits)}, len(bits), runtime.BitString{}, rd.Wire())
				if e != nil {
					t.Fatal(e)
				}
				replay := written(t, v, phase, len(bits), wire)
				if !bytes.Equal(replay, raw) {
					t.Fatal("truncated replay changed")
				}
				fresh := runtime.Canonical(v)
				total := len(bits) + want
				cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, total).Wire
				sent := written(t, fresh, phase, total, cw)
				check, _ := runtime.NewChoiceReaderBits(sent, total, runtime.NewReceiverCollector())
				_, _ = check.ReadUint(phase)
				if _, e = DecodeDTMHandoverPSRadioResourcesIEFrom(check); e != nil || check.Wire().ImplicitZeros != 0 {
					t.Fatalf("fresh sender omitted inferred fields: %v", e)
				}
			})
		}
	}
	for cut := 0; cut < 118; cut++ {
		rd, _ := runtime.NewChoiceReaderBits(packed(prefix), cut, runtime.NewReceiverCollector())
		if _, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rd); e == nil {
			t.Fatalf("filled outside suffix at bit %d", cut)
		}
	}
}

// TS 44.060 V19.0.0 §12.46 defines N from the containing global
// description. Standalone helpers cannot authenticate an application scalar.
func TestClause1246AllocationContext(t *testing.T) {
	bits := "01101010001111101"
	rd, _ := runtime.NewChoiceReaderBits(packed(bits), len(bits), nil)
	rd.Set("N", 8)
	rd.Set("M", 8)
	if _, e := DecodeUplinkTBFAssignmentStructFrom(rd); !errors.Is(e, runtime.ErrContextRequired) {
		t.Fatalf("ambient N/M admitted: %v", e)
	}
}

// The printed terminated TBF list has no count ceiling in this IE. Measure
// growing synthetic lists with nil receiver collectors and no collections.
func TestClause1246GrowingList(t *testing.T) {
	previous := uint64(0)
	for _, count := range []int{128, 512, 2048} {
		bits := strings.Repeat("0", 110) + "1100001010000000" + strings.Repeat("101101010001111101", count) + "00100"
		raw := packed(bits)
		rd, e := runtime.NewChoiceReaderBits(raw, len(bits), nil)
		if e != nil {
			t.Fatal(e)
		}
		goruntime.GC()
		old := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		value, e := DecodeDTMHandoverPSRadioResourcesIEFrom(rd)
		goruntime.ReadMemStats(&after)
		debug.SetGCPercent(old)
		if e != nil || len(assignments(value)) != count || rd.Position() != len(bits) {
			t.Fatalf("%d entries: %v", count, e)
		}
		if !raceEnabled && after.NumGC != before.NumGC {
			t.Fatal("collection inside measurement")
		}
		allocated := after.TotalAlloc - before.TotalAlloc
		if !raceEnabled && (allocated > uint64(count)*8192+1048576 || previous > 0 && allocated > previous*5+1048576) {
			t.Fatalf("nonlinear list storage: %d entries, %d bytes after %d", count, allocated, previous)
		}
		t.Logf("%d entries: %d bytes", count, allocated)
		previous = allocated
	}
}

// TS 44.060 V19.0.0 §§12.46 p24378, 11 p12938 and
// 11.1.3.2 pp13080–13081: received padding may select Ignore.
func TestClause1246ReceivedErrorReplay(t *testing.T) {
	corpus, err := os.ReadFile("testdata/fuzz/FuzzClause1246/8a7101e44137a83b")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(corpus)), "\n")
	if len(lines) != 4 || lines[0] != "go test fuzz v1" {
		t.Fatal("unexpected synthetic seed format")
	}
	quoted := strings.TrimSuffix(strings.TrimPrefix(lines[1], "[]byte("), ")")
	input, err := strconv.Unquote(quoted)
	if err != nil {
		t.Fatal("invalid seed encoding")
	}
	data := []byte(input)
	if len(data) != 22 || lines[2] != "uint16(169)" || lines[3] != "byte('\\x06')" {
		t.Fatalf("unexpected seed metadata: octets=%d", len(data))
	}
	const n, phase = 169, 6
	collector := runtime.NewReceiverCollector()
	rd, err := runtime.NewChoiceReaderBits(data, n, collector)
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := rd.ReadUint(phase)
	if err != nil {
		t.Fatal(err)
	}
	value, err := DecodeDTMHandoverPSRadioResourcesIEFrom(rd)
	if err != nil {
		t.Fatal(err)
	}
	used := rd.Position()
	report := collector.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore || used != n {
		t.Fatalf("received region/extent mismatch: regions=%d used=%d", len(report.Regions), used)
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: used}, used, runtime.BitString{}, rd.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, err = w.PushLimit(n); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(prefix, phase); err != nil {
		t.Fatal(err)
	}
	if err = EncodeDTMHandoverPSRadioResourcesIETo(w, value); err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	if err != nil || got.BitLength != used {
		t.Fatalf("sealed replay extent=%d want=%d: %v", got.BitLength, used, err)
	}
	want := bytes.Clone(data[:(used+7)/8])
	want[len(want)-1] &= byte(0xff << uint(8-used%8))
	if !bytes.Equal(got.Bytes, want) {
		t.Fatal("sealed replay changed consumed received bits")
	}
	t.Logf("sealed From/To replay: input octets=%d significant bits=%d phase=%d regions=%d", len(data), n, phase, len(report.Regions))
	// The byte entry starts at the IE, so remove the containing phase prefix.
	// Octet completion becomes part of the byte entry's received Ignore extent.
	t.Run("byte_entry_payload", func(t *testing.T) {
		payload := make([]byte, (n-phase+7)/8)
		for i := phase; i < n; i++ {
			payload[(i-phase)/8] |= ((data[i/8] >> uint(7-i%8)) & 1) << uint(7-(i-phase)%8)
		}
		decoded, err := DecodeDTMHandoverPSRadioResourcesIE(payload)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := EncodeDTMHandoverPSRadioResourcesIE(decoded.Value)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, payload) {
			t.Fatal("byte replay changed seed payload at byte-entry origin")
		}
		t.Logf("byte replay of origin-normalized payload: octets=%d payload bits=%d consumed bits=%d tail bits=%d", len(payload), n-phase, decoded.BitsConsumed, decoded.Tail.BitLength)
	})
}
