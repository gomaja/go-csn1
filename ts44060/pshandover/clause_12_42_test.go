package pshandover

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"os"
	goruntime "runtime"
	"runtime/debug"
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
func written(t testing.TB, v PSHandoverRadioResourcesIE, phase, extent int, wire runtime.WireInfo) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(0, phase); e != nil {
		t.Fatal(e)
	}
	if e := EncodePSHandoverRadioResourcesIETo(w, v); e != nil {
		t.Fatal(e)
	}
	b, e := w.FinishBits(runtime.BitString{})
	if e != nil || b.BitLength != extent {
		t.Fatalf("encoded %d want %d: %v", b.BitLength, extent, e)
	}
	return b.Bytes
}
func assignments(v PSHandoverRadioResourcesIE) []UplinkTBFAssignmentStruct {
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

func downlinkCount(v PSHandoverRadioResourcesIE) int {
	if g := v.GPRSModeChoice.GPRSMode; g != nil {
		return len(g.GPRSMode.DownlinkAssignmentList)
	}
	if g := v.GPRSModeChoice.EGPRSMode; g != nil && g.EGPRSMode.EGPRSWindowSizeGroupGroup != nil {
		return len(g.EGPRSMode.EGPRSWindowSizeGroupGroup.DownlinkAssignmentList)
	}
	return 0
}

// TS 44.060 V19.0.0 §12.42 pp23723–23817. Each standalone IE is
// bounded by its caller. Its unpublished containing message supplies no
// numerical standalone maximum, and the origin retains the bit phase.
func TestClause1242Standalone(t *testing.T) {
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
				v, e := DecodePSHandoverRadioResourcesIEFrom(rd)
				if e != nil {
					t.Fatal(e)
				}
				if rd.Position() != extent {
					t.Fatalf("consumed %d want %d", rd.Position(), extent)
				}
				if v.ARFCN != 341 || v.SI != 2 || v.NCI != 1 || v.BSIC != 42 || v.NETWORKCONTROLORDER != 2 {
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
				cv.ARFCN = 342
				edited := written(t, cv, phase, extent, cw)
				er, _ := runtime.NewChoiceReaderBits(edited, extent, nil)
				_, _ = er.ReadUint(phase)
				ev, e := DecodePSHandoverRadioResourcesIEFrom(er)
				if e != nil || ev.ARFCN != 342 {
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
func FuzzClause1242(f *testing.F) {
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
		if _, e = rd.PushLimit(n - phase); e != nil {
			return
		}
		v, e := DecodePSHandoverRadioResourcesIEFrom(rd)
		if e != nil {
			return
		}
		used := rd.Position()
		w := runtime.NewWriter()
		w.WithWire(rd.Wire())
		if _, e = w.PushLimit(n); e != nil {
			t.Fatal(e)
		}
		if e = w.WriteUint(prefix, phase); e != nil {
			t.Fatal(e)
		}
		if e = EncodePSHandoverRadioResourcesIETo(w, v); e != nil {
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
		if _, e = rr.PushLimit(n - phase); e != nil {
			t.Fatal(e)
		}
		reported, e := DecodePSHandoverRadioResourcesIEFrom(rr)
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

// TS 44.060 V19.0.0 §12.42 defines N from the containing global
// description. Standalone helpers cannot authenticate an application scalar.
func TestClause1242AllocationContext(t *testing.T) {
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
func TestClause1242GrowingList(t *testing.T) {
	previous := uint64(0)
	for _, count := range []int{128, 512, 2048} {
		bits := strings.Repeat("0", 45) + "001010000000" + strings.Repeat("101101010001111101", count) + "00"
		raw := packed(bits)
		rd, e := runtime.NewChoiceReaderBits(raw, len(bits), nil)
		if e != nil {
			t.Fatal(e)
		}
		goruntime.GC()
		old := debug.SetGCPercent(-1)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		value, e := DecodePSHandoverRadioResourcesIEFrom(rd)
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

// TS 44.060 V19.0.0 §12.42 pp23723–23817 requires the complete
// selected fields. No trailing zero rule applies to this IE.
func TestClause1242TruncationAndChangedAllocation(t *testing.T) {
	minimal := vectors(t)[0].Bits
	for cut := 0; cut < len(minimal); cut++ {
		rd, _ := runtime.NewChoiceReaderBits(packed(minimal), cut, nil)
		if _, err := DecodePSHandoverRadioResourcesIEFrom(rd); err == nil {
			t.Fatalf("accepted incomplete minimum at %d", cut)
		}
	}
	for _, tc := range vectors(t) {
		if tc.N != 2 || tc.M != 2 || !strings.Contains(tc.Name, "mode0-bitmap129-power0-subset3-usfs1-rich0") {
			continue
		}
		rd, _ := runtime.NewChoiceReaderBits(packed(tc.Bits), len(tc.Bits), nil)
		value, err := DecodePSHandoverRadioResourcesIEFrom(rd)
		if err != nil {
			t.Fatal(err)
		}
		value = runtime.Canonical(value)
		global := value.GPRSModeChoice.GPRSMode.GPRSMode.GlobalTimeslotDescriptionGroup
		global.GlobalTimeslotDescription.MSTIMESLOTALLOCATIONChoice.MSTIMESLOTALLOCATION.MSTIMESLOTALLOCATION = 0xe0
		w := runtime.NewWriter()
		if _, err := w.PushLimit(len(tc.Bits) + 8); err != nil {
			t.Fatal(err)
		}
		w.Set("N", 2)
		w.Set("M", 2)
		if err := EncodePSHandoverRadioResourcesIETo(w, value); err == nil || !strings.Contains(err.Error(), "invalid bit string width") {
			t.Fatalf("changed global allocation did not derive a new subset width: %v", err)
		}
		return
	}
	t.Fatal("missing selected-allocation witness")
}

// TS 44.060 V19.0.0 §12.42 pp23768–23796: 0x81 assigns two
// timeslots; subset 01 assigns one. M-1 is zero, and PFI is absent.
func TestClause1242SparseAllocation(t *testing.T) {
	bits := "001010101011011010100001010011000010101001100001010000001101101010001101110100"
	rd, _ := runtime.NewChoiceReaderBits(packed(bits), len(bits), nil)
	v, err := DecodePSHandoverRadioResourcesIEFrom(rd)
	if err != nil {
		t.Fatal(err)
	}
	a := v.GPRSModeChoice.GPRSMode.GPRSMode.GlobalTimeslotDescriptionGroup.UplinkAssignmentList
	if len(a) != 1 {
		t.Fatal("TBF count")
	}
	u := a[0].UplinkAssignment
	if rd.Position() != len(bits) || u.PFI != nil || u.RLCMODE != 1 || u.TFIAssignment != 21 || u.TBFTIMESLOTALLOCATION == nil || u.TBFTIMESLOTALLOCATION.BitLength != 2 || u.TBFTIMESLOTALLOCATION.Bytes[0] != 0x40 || u.USFALLOCATIONChoice.Alt1 == nil || u.USFALLOCATIONChoice.Alt1.USFALLOCATION != 5 || len(u.USFALLOCATIONChoice.Alt1.USFALLOCATIONList) != 0 {
		t.Fatalf("typed sparse allocation: %+v", u)
	}
}
