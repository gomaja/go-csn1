package controlmessages_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func dbpschAssignmentPrefix(address, network int, detail, persistence bool) string {
	out := "01000000001110" + "10"
	if persistence {
		out += "1" + fmt.Sprintf("%04b%04b%04b%04b", 0, 5, 10, 15)
	} else {
		out += "0"
	}
	switch address {
	case 0:
		out += "0" + "010101"
	case 1:
		out += "10" + fmt.Sprintf("%032b", uint32(0x12345678))
	case 2:
		out += "11" + fmt.Sprintf("%011b%016b", 0x515, 0xabcd)
	}
	out += "0" + fmt.Sprintf("%06b%08b", 25, 0xa5)
	if detail {
		out += "1" + fmt.Sprintf("%08b", 0x5a) + "11011" + "11" + fmt.Sprintf("%08b", 0x81) + "1" + fmt.Sprintf("%08b", 0x44)
	} else {
		out += "000" + fmt.Sprintf("%08b", 0x81) + "0"
	}
	out += "011"
	if detail {
		out += "1" + fmt.Sprintf("%010b", 789) + "1" + fmt.Sprintf("%08b", 165)
	} else {
		out += "0" + fmt.Sprintf("%06b%06b", 17, 33) + "0"
	}
	switch network {
	case 0:
		out += "010"
	case 1:
		out += "1010" + fmt.Sprintf("%06b%07b%07b", 37, 65, 99)
	case 2:
		out += "1011" + fmt.Sprintf("%06b%07b%07b", 37, 65, 99)
	}
	return out
}

// TS 44.060 V19.0.0 §11.2.5b paragraphs 14289–14340. This assignment
// is an ordinary downlink PCCCH/PACCH content root (paragraph 14284),
// selected by external tag 001110 (§11.2.0.1 paragraph 13361).
func TestClause1125bDBPSCHAssignment(t *testing.T) {
	for address := 0; address < 3; address++ {
		for network := 0; network < 3; network++ {
			for _, detail := range []bool{false, true} {
				for _, persistence := range []bool{false, true} {
					raw := simpleControlBits(dbpschAssignmentPrefix(address, network, detail, persistence), 184)
					r, _ := contentReader(t, raw, 184, runtime.RLCMACDownlink, 14)
					v, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(r)
					if e != nil || r.Position() != 184 {
						t.Fatalf("address %d network %d detail %t persistence %t at %d: %v", address, network, detail, persistence, r.Position(), e)
					}
					simpleControlWant(t, v, "PAGEMODE", 2)
					simpleControlWant(t, v, "CHANNELTYPE", 25)
					simpleControlWant(t, v, "DOWNLINKTIMESLOTALLOCATION", 0xa5)
					simpleControlWant(t, v, "POWERLEVEL", 0x81)
					simpleControlWant(t, v, "TSC", 3)
					if address == 0 {
						simpleControlWant(t, v, "UPLINKTFI", 21)
					}
					if address == 1 {
						simpleControlWant(t, v, "GRNTI", 0x12345678)
					}
					if address == 2 {
						simpleControlWant(t, v, "RANDOMACCESSINFORMATIONValue", 0x515)
						simpleControlWant(t, v, "FRAMENUMBER", 0xabcd)
					}
					if detail {
						simpleControlWant(t, v, "UPLINKTIMESLOTALLOCATION", 0x5a)
						simpleControlWant(t, v, "USF", 5)
						simpleControlWant(t, v, "FPCEPC", 1)
						simpleControlWant(t, v, "CHANNELMODE", 0x44)
						simpleControlWant(t, v, "ARFCN", 789)
						simpleControlWant(t, v, "TIMINGADVANCEVALUE", 165)
					} else {
						simpleControlWant(t, v, "MAIO", 17)
						simpleControlWant(t, v, "HSN", 33)
						simpleControlWant(t, v, "TIMINGADVANCEVALUE")
					}
					if network == 0 {
						simpleControlWant(t, v, "TRMINSDCCH", 1)
						simpleControlWant(t, v, "TRESPSDCCH", 0)
					} else {
						simpleControlWant(t, v, "TRMINSACCH", 0)
						simpleControlWant(t, v, "TRESPSACCH", 1)
						simpleControlWant(t, v, "TRMIN", 37)
						simpleControlWant(t, v, "TRESPMACDTM", 65)
						simpleControlWant(t, v, "TRESPMACDedicated", 99)
					}
					simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, v)
					})
					cv := runtime.Canonical(v)
					out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, cv)
					})
					if !bytes.Equal(out.Bytes, raw) {
						t.Fatal("canonical changed assignment")
					}
					cv.PAGEMODEGroupFallback.Known.PAGEMODE = 1
					edited := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
						return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, cv)
					})
					er, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACDownlink, 14)
					ev, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(er)
					if e != nil {
						t.Fatal(e)
					}
					simpleControlWant(t, ev, "PAGEMODE", 1)
					simpleControlWant(t, ev, "CHANNELTYPE", 25)
					simpleControlVector(t, "11.2.5b", fmt.Sprintf("address%d-network%d-detail%t-persistence%t", address, network, detail, persistence), raw)
				}
			}
		}
	}
}

// The reviewed reading and paragraph 14299 authorize fill throughout the channel,
// timing and network-response suffix, including cuts inside each scalar.
func TestClause1125bZeroFillScope(t *testing.T) {
	prefix := dbpschAssignmentPrefix(0, 2, true, false)
	raw := simpleControlBits(prefix, 184)
	for end := 25; end <= len(prefix); end++ {
		r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 14)
		old, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(r)
		if e != nil {
			t.Fatalf("cut %d: %v", end, e)
		}
		report := c.TakeReceiverReport()
		filledBits := 0
		for _, fill := range report.ZeroFill {
			filledBits += fill.Width
			if fill.BitOffset < end || fill.Source.Clause != "11.2.5b" {
				t.Fatalf("cut %d fill %+v", end, fill)
			}
		}
		if filledBits != r.VirtualBits() {
			t.Fatalf("cut %d report %d inferred %d", end, filledBits, r.VirtualBits())
		}
		if end < len(prefix) && len(report.ZeroFill) == 0 {
			t.Fatalf("cut %d missing fill report", end)
		}
		simpleControlWant(t, v, "PAGEMODE", 2)
		simpleControlWant(t, v, "UPLINKTFI", 21)
		simpleControlReplay(t, raw, end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, v)
		})
		if e = r.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		got, e := r.ReadUint(1)
		want := uint64((raw[end/8] >> uint(7-end%8)) & 1)
		if e != nil || got != want {
			t.Fatalf("sentinel at %d: %d want %d: %v", end, got, want, e)
		}
		cv := runtime.Canonical(v)
		out := simpleControlFresh(t, raw, 14, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, cv)
		})
		fr, _ := runtime.NewChoiceReaderBits(out.Bytes, out.BitLength, nil)
		_, _ = fr.ReadUint(14)
		fv, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(fr)
		if e != nil || !runtime.SemanticallyEqual(v, fv) {
			t.Fatalf("canonical cut %d changed typed fields: %v", end, e)
		}
	}
	for end := 14; end < 25; end++ {
		r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 14)
		_, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(r)
		if report := c.TakeReceiverReport(); len(report.ZeroFill) != 0 {
			t.Fatalf("fill before authorized suffix at %d: %+v", end, report)
		}
	}
}
func FuzzClause1125bDBPSCHAssignment(f *testing.F) {
	f.Add(simpleControlBits(dbpschAssignmentPrefix(0, 2, true, true), 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, v)
			})
		}
	})
}

func TestClause1125bReceiverRegions(t *testing.T) {
	for _, tc := range []struct {
		end    int
		prefix string
		label  runtime.ErrorLabel
		page   bool
	}{{15, "10", runtime.DistributionPartError, false}, {20, "1000010101", runtime.AddressInformationPartError, true}, {184, "10000101011", runtime.MessageEscape, true}} {
		raw := simpleControlBits("01000000001110"+tc.prefix, 184)
		r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 14)
		_, e := r.PushLimit(tc.end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketDBPSCHAssignmentMessageContentFrom(r)
		if e != nil {
			t.Fatalf("region %s: %v", tc.label, e)
		}
		report := c.TakeReceiverReport()
		if len(report.Regions) != 1 || report.Regions[0].Label != tc.label {
			t.Fatalf("region %s got %+v", tc.label, report)
		}
		if tc.page {
			simpleControlWant(t, v, "PAGEMODE", 2)
		}
		simpleControlReplay(t, raw, tc.end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketDBPSCHAssignmentMessageContentTo(w, v)
		})
	}
}
