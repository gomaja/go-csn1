package controlmessages_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func powerTimingParameters(detail bool) (string, string) {
	timing := "000"
	power := "1001" + strings.Repeat("0", 8)
	if detail {
		timing = fmt.Sprintf("1%06b1%04b%03b1%04b%03b", 37, 9, 5, 6, 3)
		power = "1001"
		for i := 1; i <= 8; i++ {
			power += fmt.Sprintf("1%05b", i)
		}
	}
	return timing, power
}
func powerTimingPrefix(global bool, action int, detail bool) string {
	out := "01000000000101" + "100" + "010101" + "0"
	if global {
		out += fmt.Sprintf("1%04b%05b%05b%04b10%04b", 9, 17, 18, 11, 7)
	} else {
		out += "0"
	}
	timing, power := powerTimingParameters(detail)
	switch action {
	case 0:
		out += "0" + timing + power
	case 1:
		out += "10" + timing
	case 2:
		out += "11" + power
	}
	return out
}
func powerTimingRelease(count int) string {
	return "1110" + "1101" + fmt.Sprintf("11%04b", count) + strings.Repeat("1", count+1)
}

// TS 44.060 V19.0.0 §11.2.13 paragraphs 16392–16415, including the
// decision-113 correction at paragraph 16410. Downlink selector 000101
// is external (§11.2.0.1 paragraph 13351); §§12.9/12.12a/12.13 own the IEs.
func TestClause11213PowerTiming(t *testing.T) {
	for _, global := range []bool{false, true} {
		for action := 0; action < 3; action++ {
			for _, detail := range []bool{false, true} {
				for _, count := range []int{0, 3, 15} {
					prefix := powerTimingPrefix(global, action, detail) + powerTimingRelease(count)
					if len(prefix) >= 184 {
						t.Fatalf("prefix %d", len(prefix))
					}
					raw := simpleControlBits(prefix, 184)
					r, _ := contentReader(t, raw, 184, runtime.RLCMACDownlink, 5)
					v, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(r)
					if e != nil || r.Position() != 184 {
						t.Fatalf("global %t action %d detail %t count %d at %d: %v", global, action, detail, count, r.Position(), e)
					}
					simpleControlWant(t, v, "PAGEMODE", 2)
					simpleControlWant(t, v, "UPLINKTFI", 21)
					simpleControlWant(t, v, "CarrierCount", uint64(count))
					simpleControlWant(t, v, "PacketExtendedTimingAdvance", 2)
					simpleControlWant(t, v, "CarrierIdentification", 1)
					if detail && action != 2 {
						simpleControlWant(t, v, "TIMINGADVANCEVALUE", 37)
					} else {
						simpleControlWant(t, v, "TIMINGADVANCEVALUE")
					}
					if detail && action != 1 {
						simpleControlWant(t, v, "GAMMATN7", 8)
					} else {
						simpleControlWant(t, v, "GAMMATN7")
					}
					simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, v)
					})
					cv := runtime.Canonical(v)
					out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, cv)
					})
					if !bytes.Equal(out.Bytes, raw) {
						t.Fatal("canonical changed power/timing fields")
					}
					cv.PAGEMODEGroupFallback.Known.PAGEMODE = 1
					edited := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, cv)
					})
					er, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACDownlink, 5)
					ev, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(er)
					if e != nil {
						t.Fatal(e)
					}
					simpleControlWant(t, ev, "PAGEMODE", 1)
					simpleControlWant(t, ev, "CarrierCount", uint64(count))
					simpleControlVector(t, "11.2.13", fmt.Sprintf("global%t-action%d-detail%t-count%d", global, action, detail, count), raw)
				}
			}
		}
	}
}
func TestClause11213ReleaseAndReceiverRegions(t *testing.T) {
	for _, suffix := range []string{"", "10", "1010"} {
		prefix := powerTimingPrefix(false, 1, false) + suffix
		raw := simpleControlBits(prefix, 184)
		end := len(prefix)
		r, _ := contentReader(t, raw, 184, runtime.RLCMACDownlink, 5)
		old, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(r)
		if e != nil || r.Position() != end {
			t.Fatalf("null suffix %q at %d: %v", suffix, r.Position(), e)
		}
		simpleControlWant(t, v, "PAGEMODE", 2)
		simpleControlReplay(t, raw, end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, v)
		})
		if e = r.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		if sentinel, e := r.ReadUint(1); e != nil || sentinel != 0 {
			t.Fatalf("sentinel %d: %v", sentinel, e)
		}
		ignored := simpleControlBits(prefix+"0"+strings.Repeat("1", 183-end), 184)
		ir, _ := contentReader(t, ignored, 184, runtime.RLCMACDownlink, 5)
		iv, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(ir)
		if e != nil || ir.Position() != 184 {
			t.Fatalf("ignored suffix %q: %v", suffix, e)
		}
		simpleControlWant(t, iv, "PAGEMODE", 2)
		simpleControlReplay(t, ignored, 184, 14, ir, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, iv)
		})
	}
	for _, tc := range []struct{ bits, label string }{{"101", "Address information part error"}, {"1000101011", "Message escape"}, {"1000101010" + "1" + fmt.Sprintf("%04b%05b%05b%04b11%04b", 9, 17, 18, 11, 7), "Non-distribution part error"}} {
		raw := simpleControlBits("01000000000101"+tc.bits+strings.Repeat("1", 170-len(tc.bits)), 184)
		r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 5)
		v, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(r)
		if e != nil {
			t.Fatalf("region %s: %v", tc.label, e)
		}
		simpleControlWant(t, v, "PAGEMODE", 2)
		report := c.TakeReceiverReport()
		if len(report.Regions) != 1 || string(report.Regions[0].Label) != tc.label {
			t.Fatalf("region %s report %+v", tc.label, report)
		}
		simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, v)
		})
	}
}
func FuzzClause11213PowerTiming(f *testing.F) {
	f.Add(simpleControlBits(powerTimingPrefix(true, 0, true)+powerTimingRelease(15), 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, e := controlmessages.DecodePacketPowerControlTimingAdvanceMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketPowerControlTimingAdvanceMessageContentTo(w, v)
			})
		}
	})
}
