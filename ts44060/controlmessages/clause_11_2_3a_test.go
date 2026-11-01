package controlmessages_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func notification3GTarget(detail bool) string {
	if !detail {
		return "00" + fmt.Sprintf("%06b", 45)
	}
	return fmt.Sprintf("1%014b1%03b%09b1%014b1%03b%07b1%06b", 1234, 5, 301, 4321, 3, 85, 45)
}
func notificationEUTRANReport(count int) string {
	out := fmt.Sprintf("1%02b", count)
	for i := 0; i <= count; i++ {
		out += fmt.Sprintf("%03b%09b%06b", i, 301+i, 41+i)
	}
	return out
}
func notificationTarget(which int, detail bool) string {
	band := "0"
	if detail {
		band = "1101"
	}
	switch which {
	case 0:
		return "0" + fmt.Sprintf("%010b%06b", 511, 42)
	case 1:
		return "10" + notification3GTarget(detail)
	case 2:
		return "1100000"
	case 3:
		return "1101" + fmt.Sprintf("%010b%06b", 511, 42) + "000"
	case 4:
		return "11001" + notification3GTarget(detail) + "00"
	case 5:
		return "110001" + fmt.Sprintf("%016b", 45678) + band + fmt.Sprintf("%09b%06b", 321, 45) + "0"
	case 6:
		return "1100001" + notificationEUTRANReport(3)
	case 7, 8:
		plmn := "0"
		if detail {
			plmn = fmt.Sprintf("1%012b%012b", 0x123, 0x456)
		}
		if which == 7 {
			return "11100" + fmt.Sprintf("%028b", 0x1234567) + plmn + fmt.Sprintf("%027b1%06b", 0x1234567, 45) + "0"
		}
		return "11101" + fmt.Sprintf("%028b%016b", 0xabcdef1, 0xbeef) + plmn + fmt.Sprintf("%027b1%06b", 0x1234567, 45) + "0"
	case 9:
		return "11100" + fmt.Sprintf("%028b0%027b1%06b", 0x1234567, 0x1234567, 45) + "1" + notificationEUTRANReport(0)
	}
	panic("invalid synthetic target")
}
func notificationCommon(count int, detail bool) string {
	out := "01" + "11" + fmt.Sprintf("%06b0%03b", 41, count)
	if detail {
		out = "110" + "11" + fmt.Sprintf("%06b0%03b", 41, count)
	}
	for i := 0; i < count; i++ {
		out += fmt.Sprintf("%06b", i+4)
		if detail && i == 0 {
			out += fmt.Sprintf("1%06b", i+5)
		} else {
			out += "0"
		}
		out += fmt.Sprintf("%06b", i+7)
	}
	return out
}
func notificationRelease(count int, detail bool) string {
	out := "1"
	if detail {
		out += "11"
	} else {
		out += "0"
	}
	out += fmt.Sprintf("%03b", count)
	for i := 0; i <= count; i++ {
		out += fmt.Sprintf("%07b%06b", i+7, i+31)
	}
	out += "11" + "1"
	if detail {
		out += "1" + fmt.Sprintf("%018b1%03b%09b%06b", 0x2abcd, 5, 211, 45)
	} else {
		out += "0"
	}
	out += "1"
	if detail {
		out += "1" + "101"
	} else {
		out += "0"
	}
	return out
}

// TS 44.060 V19.0.0 §11.2.3a paragraphs 13653–13731; the reviewed reading
// removes only the premature brace at paragraph 13658. Uplink selector
// 001100 is external (§11.2.0.2 paragraph 13438). Counts at paragraphs
// 13713/13726 retain their printed val(N + 1) meaning.
func TestClause1123aCellChangeNotification(t *testing.T) {
	for target := 0; target < 10; target++ {
		for _, detail := range []bool{false, true} {
			// Large CSG and dual-3G targets use the short normal-form release tail.
			releaseDetail := detail && target < 2
			prefix := "01000000001100" + "010101" + notificationTarget(target, detail) + notificationCommon(0, detail) + notificationRelease(0, releaseDetail)
			if len(prefix) >= 184 {
				t.Fatalf("target %d prefix %d", target, len(prefix))
			}
			raw := simpleControlBits(prefix, 184)
			r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
			v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
			if e != nil || r.Position() != 184 {
				t.Fatalf("target %d detail %t at %d: %v", target, detail, r.Position(), e)
			}
			if v.PMOUSED != 1 || v.PCCNSENDING != 1 || v.CCNMeasurementReport.RXLEVSERVINGCELL != 41 || v.CCNMeasurementReport.NUMBEROFNCMEASUREMENTS != 0 {
				t.Fatalf("typed common %+v", v)
			}
			simpleControlWant(t, v, "UPLINKTFI", 21)
			simpleControlWant(t, v, "N3G", 0)
			simpleControlWant(t, v, "N3GCELLLISTINDEX", 7)
			if target == 0 || target == 3 {
				simpleControlWant(t, v, "ARFCN", 511)
				simpleControlWant(t, v, "BSIC", 42)
			}
			if target == 1 || target == 4 {
				simpleControlWant(t, v, "REPORTINGQUANTITY", func() []uint64 {
					if releaseDetail {
						return []uint64{45, 31, 45}
					}
					return []uint64{45, 31}
				}()...)
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
			})
			cv := runtime.Canonical(v)
			out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, cv)
			})
			if !bytes.Equal(out.Bytes, raw) {
				t.Fatal("canonical changed notification fields")
			}
			cv.PMOUSED = 0
			edited := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, cv)
			})
			er, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACUplink, 12)
			ev, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(er)
			if e != nil || ev.PMOUSED != 0 || ev.PCCNSENDING != 1 {
				t.Fatalf("edited %+v: %v", ev, e)
			}
			simpleControlVector(t, "11.2.3a", fmt.Sprintf("target%d-detail%t", target, detail), raw)
		}
	}
	for _, count := range []int{0, 1, 7} {
		prefix := "01000000001100" + "010101" + notificationTarget(0, false) + notificationCommon(count, true) + notificationRelease(0, false)
		raw := simpleControlBits(prefix, 184)
		if len(prefix) >= 184 {
			t.Fatalf("NC prefix %d", len(prefix))
		}
		r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
		v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
		if e != nil || int(v.CCNMeasurementReport.NUMBEROFNCMEASUREMENTS) != count || len(v.CCNMeasurementReport.FREQUENCYNGroupList) != count {
			t.Fatalf("NC count %d %+v: %v", count, v.CCNMeasurementReport, e)
		}
		simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
		})
		simpleControlVector(t, "11.2.3a", fmt.Sprintf("nc-count%d", count), raw)
	}
	for _, count := range []int{0, 1, 7} {
		prefix := "01000000001100" + "010101" + notificationTarget(0, false) + notificationCommon(0, false) + notificationRelease(count, false)
		raw := simpleControlBits(prefix, 184)
		if len(prefix) >= 184 {
			t.Fatalf("3G prefix %d", len(prefix))
		}
		r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
		v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
		if e != nil {
			t.Fatalf("3G count %d: %v", count, e)
		}
		simpleControlWant(t, v, "N3G", uint64(count))
		want := []uint64{}
		for i := 0; i <= count; i++ {
			want = append(want, uint64(i+7))
		}
		simpleControlWant(t, v, "N3GCELLLISTINDEX", want...)
		simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
		})
		simpleControlVector(t, "11.2.3a", fmt.Sprintf("3g-count%d", count), raw)
	}
}
func TestClause1123aReleaseAndEscape(t *testing.T) {
	base := "01000000001100" + "010101" + notificationTarget(0, false) + notificationCommon(0, false)
	// All four release extensions independently expose null and zero+ignored.
	tails := []string{"", "10" + "000" + fmt.Sprintf("%07b%06b", 7, 31), "10000" + fmt.Sprintf("%07b%06b", 7, 31) + "11", "10000" + fmt.Sprintf("%07b%06b", 7, 31) + "1110"}
	for _, tail := range tails {
		prefix := base + tail
		end := len(prefix)
		raw := simpleControlBits(prefix, 184)
		r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
		_, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
		if e != nil || r.Position() != end {
			t.Fatalf("null suffix %q at %d: %v", tail, r.Position(), e)
		}
		simpleControlReplay(t, raw, end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
		})
		ignored := simpleControlBits(prefix+"0"+strings.Repeat("1", 183-end), 184)
		ir, _ := contentReader(t, ignored, 184, runtime.RLCMACUplink, 12)
		iv, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(ir)
		if e != nil || ir.Position() != 184 {
			t.Fatalf("ignored suffix %q: %v", tail, e)
		}
		simpleControlReplay(t, ignored, 184, 14, ir, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, iv)
		})
	}
	raw := simpleControlBits("01000000001100"+"0101011111"+strings.Repeat("1", 160), 184)
	r, c := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
	v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
	if e != nil {
		t.Fatal(e)
	}
	simpleControlWant(t, v, "UPLINKTFI", 21)
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.MessageEscape || v.PMOUSED != 0 || v.PCCNSENDING != 0 || r.Position() != 184 {
		t.Fatalf("escape value %+v report %+v", v, report)
	}
	simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
		return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
	})
}
func FuzzClause1123aCellChangeNotification(f *testing.F) {
	f.Add(simpleControlBits("01000000001100"+"010101"+notificationTarget(0, true)+notificationCommon(0, true)+notificationRelease(0, true), 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
			})
		}
	})
}

// Positive-count counterparts let an independent decoder exercise every RAT
// choice even when it cannot represent the printed zero-NC-measurement case.
func TestClause1123aPositiveMeasurementOracle(t *testing.T) {
	for target := 0; target < 10; target++ {
		for _, detail := range []bool{false, true} {
			prefix := "01000000001100" + "010101" + notificationTarget(target, detail) + notificationCommon(1, false) + "0"
			if len(prefix) > 184 {
				t.Fatalf("target %d positive NC prefix %d", target, len(prefix))
			}
			raw := simpleControlBits(prefix, 184)
			r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
			v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
			if e != nil || r.Position() != 184 || v.CCNMeasurementReport.NUMBEROFNCMEASUREMENTS != 1 || len(v.CCNMeasurementReport.FREQUENCYNGroupList) != 1 {
				t.Fatalf("positive target %d detail %t: %v", target, detail, e)
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
			})
			simpleControlVector(t, "11.2.3a", fmt.Sprintf("positive-target%d-detail%t", target, detail), raw)
		}
	}
	for _, count := range []int{0, 1, 7} {
		prefix := "01000000001100" + "010101" + notificationTarget(0, false) + notificationCommon(1, false) + notificationRelease(count, false)
		if len(prefix) > 184 {
			t.Fatalf("positive 3G count %d prefix %d", count, len(prefix))
		}
		raw := simpleControlBits(prefix, 184)
		r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 12)
		v, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r)
		if e != nil {
			t.Fatal(e)
		}
		simpleControlWant(t, v, "N3G", uint64(count))
		simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeNotificationMessageContentTo(w, v)
		})
		simpleControlVector(t, "11.2.3a", fmt.Sprintf("positive-3g-count%d", count), raw)
	}
}

func TestClause1123aMalformedCommonFields(t *testing.T) {
	prefix := "01000000001100" + "010101" + notificationTarget(0, false) + notificationCommon(0, false)
	raw := simpleControlBits(prefix, 184)
	for end := 14; end < len(prefix); end++ {
		r, _ := runtime.NewChoiceReaderBits(raw, end, nil)
		_, _ = r.ReadUint(14)
		if _, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r); e == nil {
			t.Fatalf("truncated common prefix accepted at %d", end)
		}
	}
	// Paragraph 13694 fixes the bit after RXLEV_SERVING_CELL to zero.
	invalid := append([]byte(nil), raw...)
	at := 14 + 6 + 17 + 4 + 6
	invalid[at/8] ^= 1 << uint(7-at%8)
	r, _ := runtime.NewChoiceReaderBits(invalid, 184, nil)
	_, _ = r.ReadUint(14)
	if _, e := controlmessages.DecodePacketCellChangeNotificationMessageContentFrom(r); e == nil {
		t.Fatal("allocated-in-earlier-version literal accepted as zero")
	}
}
