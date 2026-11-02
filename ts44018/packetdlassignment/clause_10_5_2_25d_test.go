package packetdlassignment

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.25d pp18158–18236: the source vectors
// cover all base option combinations, every release boundary through Rel-10,
// both receiver termination arms, and empty/populated sentinel lists.
// Imported timing, power, window and level fields retain their typed IEs.
type rrAssignmentVector struct {
	name, bits                string
	mandatory, depth, options int
	terminal                  string
	populated                 bool
	segments                  [][2]int
}

var rrAssignmentVectors = []rrAssignmentVector{
	{name: "base-options-00", bits: "10110100101000000", mandatory: 17, depth: -1, options: 0, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-00", bits: "1011010010100000010110110110", mandatory: 17, depth: 0, options: 0, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-01", bits: "101101001011100101110011010000", mandatory: 30, depth: -1, options: 1, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-01", bits: "10110100101110010111001101000010110110110", mandatory: 30, depth: 0, options: 1, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-02", bits: "10110100101001101001000", mandatory: 23, depth: -1, options: 2, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-02", bits: "1011010010100110100100010110110110", mandatory: 23, depth: 0, options: 2, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-03", bits: "101101001011100101110011011101001000", mandatory: 36, depth: -1, options: 3, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-03", bits: "10110100101110010111001101110100100010110110110", mandatory: 36, depth: 0, options: 3, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-04", bits: "101101001010001100110011100000011011100", mandatory: 39, depth: -1, options: 4, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-04", bits: "10110100101000110011001110000001101110010110110110", mandatory: 39, depth: 0, options: 4, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-05", bits: "1011010010111001011100110101100110011100000011011100", mandatory: 52, depth: -1, options: 5, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-05", bits: "101101001011100101110011010110011001110000001101110010110110110", mandatory: 52, depth: 0, options: 5, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-06", bits: "101101001010011010011100110011100000011011100", mandatory: 45, depth: -1, options: 6, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-06", bits: "10110100101001101001110011001110000001101110010110110110", mandatory: 45, depth: 0, options: 6, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-07", bits: "1011010010111001011100110111010011100110011100000011011100", mandatory: 58, depth: -1, options: 7, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-07", bits: "101101001011100101110011011101001110011001110000001101110010110110110", mandatory: 58, depth: 0, options: 7, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-08", bits: "1011010010100001100110", mandatory: 22, depth: -1, options: 8, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-08", bits: "101101001010000110011010110110110", mandatory: 22, depth: 0, options: 8, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-09", bits: "10110100101110010111001101001100110", mandatory: 35, depth: -1, options: 9, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-09", bits: "1011010010111001011100110100110011010110110110", mandatory: 35, depth: 0, options: 9, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-10", bits: "1011010010100110100101100110", mandatory: 28, depth: -1, options: 10, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-10", bits: "101101001010011010010110011010110110110", mandatory: 28, depth: 0, options: 10, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-11", bits: "10110100101110010111001101110100101100110", mandatory: 41, depth: -1, options: 11, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-11", bits: "1011010010111001011100110111010010110011010110110110", mandatory: 41, depth: 0, options: 11, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-12", bits: "10110100101000110011001110000001101111100110", mandatory: 44, depth: -1, options: 12, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-12", bits: "1011010010100011001100111000000110111110011010110110110", mandatory: 44, depth: 0, options: 12, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-13", bits: "101101001011100101110011010110011001110000001101111100110", mandatory: 57, depth: -1, options: 13, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-13", bits: "10110100101110010111001101011001100111000000110111110011010110110110", mandatory: 57, depth: 0, options: 13, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-14", bits: "10110100101001101001110011001110000001101111100110", mandatory: 50, depth: -1, options: 14, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-14", bits: "1011010010100110100111001100111000000110111110011010110110110", mandatory: 50, depth: 0, options: 14, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "base-options-15", bits: "101101001011100101110011011101001110011001110000001101111100110", mandatory: 63, depth: -1, options: 15, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "r99-options-15", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110", mandatory: 63, depth: 0, options: 15, terminal: "null", populated: false, segments: [][2]int{}},
	{name: "rel6-0-null", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011010", mandatory: 63, depth: 1, options: 15, terminal: "null", populated: false, segments: [][2]int{{74, 76}}},
	{name: "rel6-0-error-region", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011010010101", mandatory: 63, depth: 1, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}}},
	{name: "rel6-1-null", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000", mandatory: 63, depth: 1, options: 15, terminal: "null", populated: true, segments: [][2]int{{74, 127}}},
	{name: "rel6-1-error-region", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000010101", mandatory: 63, depth: 1, options: 15, terminal: "error-region", populated: true, segments: [][2]int{{74, 127}}},
	{name: "rel7-0-null", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010", mandatory: 63, depth: 2, options: 15, terminal: "null", populated: false, segments: [][2]int{{74, 76}, {76, 81}}},
	{name: "rel7-0-error-region", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010010101", mandatory: 63, depth: 2, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}}},
	{name: "rel7-1-null", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110111101100000011011110111000101011001000100110001100001111101011100010", mandatory: 63, depth: 2, options: 15, terminal: "null", populated: true, segments: [][2]int{{74, 127}, {127, 143}}},
	{name: "rel7-1-error-region", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110111101100000011011110111000101011001000100110001100001111101011100010010101", mandatory: 63, depth: 2, options: 15, terminal: "error-region", populated: true, segments: [][2]int{{74, 127}, {127, 143}}},
	{name: "rel8-0-null", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010100", mandatory: 63, depth: 3, options: 15, terminal: "null", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}}},
	{name: "rel8-0-error-region", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010100010101", mandatory: 63, depth: 3, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}}},
	{name: "rel8-1-null", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000111110101110001011111011011001101001110011100", mandatory: 63, depth: 3, options: 15, terminal: "null", populated: true, segments: [][2]int{{74, 127}, {127, 143}, {143, 172}}},
	{name: "rel8-1-error-region", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000111110101110001011111011011001101001110011100010101", mandatory: 63, depth: 3, options: 15, terminal: "error-region", populated: true, segments: [][2]int{{74, 127}, {127, 143}, {143, 172}}},
	{name: "rel9-0-null", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110101101010010", mandatory: 63, depth: 4, options: 15, terminal: "null", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}, {84, 86}}},
	{name: "rel9-0-error-region", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110101101010010010101", mandatory: 63, depth: 4, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}, {84, 86}}},
	{name: "rel9-1-null", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000111110101110001011111011011001101001110011100110011111010011000111101011100011101", mandatory: 63, depth: 4, options: 15, terminal: "null", populated: true, segments: [][2]int{{74, 127}, {127, 143}, {143, 172}, {172, 208}}},
	{name: "rel9-1-error-region", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000111110101110001011111011011001101001110011100110011111010011000111101011100011101010101", mandatory: 63, depth: 4, options: 15, terminal: "error-region", populated: true, segments: [][2]int{{74, 127}, {127, 143}, {143, 172}, {172, 208}}},
	{name: "rel10-0-spare", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110101101010010100010101", mandatory: 63, depth: 5, options: 15, terminal: "spare", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}, {84, 86}, {86, 90}}},
	{name: "rel10-1-spare", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011011110110000001101111011100010101100100010011000110000111110101110001011111011011001101001110011100110011111010011000111101011100011101110001111101100011011001000000111001001010001000101001101110110010011010101", mandatory: 63, depth: 5, options: 15, terminal: "spare", populated: true, segments: [][2]int{{74, 127}, {127, 143}, {143, 172}, {172, 208}, {208, 278}}},
	{name: "terminal-0-error", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110010101", mandatory: 63, depth: 0, options: 15, terminal: "error-region", populated: false, segments: [][2]int{}},
	{name: "terminal-1-error", bits: "1011010010111001011100110111010011100110011100000011011111001101011011011010010101", mandatory: 63, depth: 1, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}}},
	{name: "terminal-2-error", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010010101", mandatory: 63, depth: 2, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}}},
	{name: "terminal-3-error", bits: "101101001011100101110011011101001110011001110000001101111100110101101101101011010100010101", mandatory: 63, depth: 3, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}}},
	{name: "terminal-4-error", bits: "10110100101110010111001101110100111001100111000000110111110011010110110110101101010010010101", mandatory: 63, depth: 4, options: 15, terminal: "error-region", populated: false, segments: [][2]int{{74, 76}, {76, 81}, {81, 84}, {84, 86}}},
}

func rrAssignmentPack(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

func checkRRArm(t *testing.T, terminal string, null, noString, child bool) {
	t.Helper()
	if terminal == "null" && (!null || noString || child) || terminal == "error-region" && (null || !noString || child) || terminal == "child" && (null || noString || !child) {
		t.Fatalf("release arm %s: null=%v noString=%v child=%v", terminal, null, noString, child)
	}
}

func checkRRAssignmentFields(t *testing.T, v RRPacketDownlinkAssignmentValuePart, x rrAssignmentVector) {
	t.Helper()
	if v.MACMODE != 2 || v.RLCMODE != 1 || v.TIMESLOTALLOCATION != 0xa5 {
		t.Fatalf("base fields %+v", v)
	}
	ta := v.PacketTimingAdvance
	if x.options&1 != 0 {
		if ta.TIMINGADVANCEVALUE == nil || *ta.TIMINGADVANCEVALUE != 37 || ta.TIMINGADVANCEINDEXGroup == nil || ta.TIMINGADVANCEINDEXGroup.TIMINGADVANCEINDEX != 9 || ta.TIMINGADVANCEINDEXGroup.TIMINGADVANCETIMESLOTNUMBER != 5 {
			t.Fatalf("timing IE %+v", ta)
		}
	} else if ta.TIMINGADVANCEVALUE != nil || ta.TIMINGADVANCEINDEXGroup != nil {
		t.Fatal("absent timing fields acquired values")
	}
	if x.options&2 != 0 {
		if v.P0Group == nil || v.P0Group.P0 != 10 || v.P0Group.PRMODE != 1 {
			t.Fatal("P0 group differs")
		}
	} else if v.P0Group != nil {
		t.Fatal("unexpected P0 group")
	}
	if x.options&4 != 0 {
		p := v.PowerControlParameters
		if p == nil || p.ALPHA != 9 || p.GAMMATN0 == nil || *p.GAMMATN0 != 7 || p.GAMMATN7 == nil || *p.GAMMATN7 != 23 || p.GAMMATN1 != nil || p.GAMMATN2 != nil || p.GAMMATN3 != nil || p.GAMMATN4 != nil || p.GAMMATN5 != nil || p.GAMMATN6 != nil {
			t.Fatalf("power IE %+v", p)
		}
	} else if v.PowerControlParameters != nil {
		t.Fatal("unexpected power IE")
	}
	if x.options&8 != 0 {
		if v.DOWNLINKTFIASSIGNMENT == nil || *v.DOWNLINKTFIASSIGNMENT != 19 {
			t.Fatal("base downlink TFI differs")
		}
	} else if v.DOWNLINKTFIASSIGNMENT != nil {
		t.Fatal("unexpected downlink TFI")
	}
	if x.depth < 0 {
		if v.EGPRSWindowSizeChoice.AltUnlabeled == nil || v.EGPRSWindowSizeChoice.EGPRSWindowSize != nil {
			t.Fatal("R99 null arm differs")
		}
		return
	}
	r99 := v.EGPRSWindowSizeChoice.EGPRSWindowSize
	if r99 == nil || r99.EGPRSWindowSizeGroup == nil || r99.EGPRSWindowSizeGroup.EGPRSWindowSize.EGPRSWindowSize != 13 || r99.EGPRSWindowSizeGroup.LINKQUALITYMEASUREMENTMODE != 2 || r99.PacketExtendedTimingAdvance == nil || *r99.PacketExtendedTimingAdvance != 2 {
		t.Fatal("R99 typed window/timing fields differ")
	}
	c6 := r99.MultipleDownlinkTBFAssignmentChoice
	if x.depth == 0 {
		checkRRArm(t, x.terminal, c6.AltUnlabeled != nil, c6.NoString != nil, c6.MultipleDownlinkTBFAssignment != nil)
		return
	}
	checkRRArm(t, "child", c6.AltUnlabeled != nil, c6.NoString != nil, c6.MultipleDownlinkTBFAssignment != nil)
	r6 := c6.MultipleDownlinkTBFAssignment
	if x.populated {
		if len(r6.MultipleDownlinkTBFAssignmentList) != 1 {
			t.Fatal("missing Rel-6 assignment")
		}
		m := r6.MultipleDownlinkTBFAssignmentList[0].MultipleDownlinkTBFAssignment
		if m.UplinkControlTimeslot == nil || *m.UplinkControlTimeslot != 5 || m.TIMESLOTALLOCATION != 0x81 || len(m.DownlinkTBFAssignmentList) != 2 {
			t.Fatal("multiple-TBF typed carrier differs")
		}
		a, b := m.DownlinkTBFAssignmentList[0].DownlinkTBFAssignment, m.DownlinkTBFAssignmentList[1].DownlinkTBFAssignment
		if a.PFI != 61 || a.RLCMODE != 1 || a.DOWNLINKTFIASSIGNMENT != 17 || a.CONTROLACK != 0 || a.EGPRSWindowSize == nil || a.EGPRSWindowSize.EGPRSWindowSize != 12 || b.PFI != 9 || b.RLCMODE != 1 || b.DOWNLINKTFIASSIGNMENT != 3 || b.EGPRSWindowSize != nil {
			t.Fatal("nested downlink assignments differ")
		}
	} else if len(r6.MultipleDownlinkTBFAssignmentList) != 0 {
		t.Fatal("unexpected Rel-6 assignment")
	}
	c7 := r6.FANRChoice
	if x.depth == 1 {
		checkRRArm(t, x.terminal, c7.AltUnlabeled != nil, c7.NoString != nil, c7.FANR != nil)
		return
	}
	checkRRArm(t, "child", c7.AltUnlabeled != nil, c7.NoString != nil, c7.FANR != nil)
	r7 := c7.FANR
	if r7.FANR != 1 || r7.DownlinkEGPRSLevel.EGPRSLevel != 2 {
		t.Fatal("Rel-7 FANR or typed EGPRS level differs")
	}
	if x.populated {
		if len(r7.NPMTransferTimeGroupList) != 2 || r7.NPMTransferTimeGroupList[0].NPMTransferTime == nil || *r7.NPMTransferTimeGroupList[0].NPMTransferTime != 21 || r7.NPMTransferTimeGroupList[0].EVENTBASEDFANR != 1 || r7.NPMTransferTimeGroupList[1].NPMTransferTime != nil || r7.NPMTransferTimeGroupList[1].EVENTBASEDFANR != 0 {
			t.Fatal("Rel-7 NPM/event list differs")
		}
	} else if len(r7.NPMTransferTimeGroupList) != 0 {
		t.Fatal("unexpected NPM list")
	}
	c8 := r7.MeasurementControlEUTRANChoice
	if x.depth == 2 {
		checkRRArm(t, x.terminal, c8.AltUnlabeled != nil, c8.NoString != nil, c8.MeasurementControlEUTRAN != nil)
		return
	}
	checkRRArm(t, "child", c8.AltUnlabeled != nil, c8.NoString != nil, c8.MeasurementControlEUTRAN != nil)
	r8 := c8.MeasurementControlEUTRAN
	if x.populated {
		if r8.MeasurementControlEUTRANGroupList == nil || r8.MeasurementControlUTRANGroupList == nil || len(*r8.MeasurementControlEUTRANGroupList) != 1 || len(*r8.MeasurementControlUTRANGroupList) != 1 {
			t.Fatal("missing Rel-8 measurement lists")
		}
		e, u := (*r8.MeasurementControlEUTRANGroupList)[0], (*r8.MeasurementControlUTRANGroupList)[0]
		if e.MeasurementControlEUTRAN != 1 || e.EUTRANFREQUENCYINDEX != 5 || len(e.EUTRANFREQUENCYINDEXList) != 1 || e.EUTRANFREQUENCYINDEXList[0].EUTRANFREQUENCYINDEX != 3 || u.MeasurementControlUTRAN != 0 || u.UTRANFREQUENCYINDEX != 19 || len(u.UTRANFREQUENCYINDEXList) != 1 || u.UTRANFREQUENCYINDEXList[0].UTRANFREQUENCYINDEX != 7 {
			t.Fatal("nested Rel-8 frequency lists differ")
		}
	} else if r8.MeasurementControlEUTRANGroupList != nil || r8.MeasurementControlUTRANGroupList != nil {
		t.Fatal("unexpected measurement lists")
	}
	c9 := r8.RLCEntity2Choice
	if x.depth == 3 {
		checkRRArm(t, x.terminal, c9.AltUnlabeled != nil, c9.NoString != nil, c9.RLCEntity2 != nil)
		return
	}
	checkRRArm(t, "child", c9.AltUnlabeled != nil, c9.NoString != nil, c9.RLCEntity2 != nil)
	r9 := c9.RLCEntity2
	if x.populated {
		if r9.RLCEntity2Group == nil || r9.RLCEntity2Group.RLCEntity3 == nil {
			t.Fatal("missing typed EMST RLC entities")
		}
		a, b := r9.RLCEntity2Group.RLCEntity2, *r9.RLCEntity2Group.RLCEntity3
		if a.DOWNLINKTFIASSIGNMENT != 7 || a.RLCMODE != 1 || a.PFI != 71 || a.EGPRSWindowSize == nil || a.EGPRSWindowSize.EGPRSWindowSize != 9 || b.DOWNLINKTFIASSIGNMENT != 11 || b.RLCMODE != 1 || b.PFI != 29 || b.EGPRSWindowSize != nil {
			t.Fatal("typed RLC entity fields differ")
		}
	} else if r9.RLCEntity2Group != nil {
		t.Fatal("unexpected RLC entities")
	}
	c10 := r9.EMSRAdditionalPFCs1Choice
	if x.depth == 4 {
		checkRRArm(t, x.terminal, c10.AltUnlabeled != nil, c10.NoString != nil, c10.EMSRAdditionalPFCs1 != nil)
		return
	}
	checkRRArm(t, "child", c10.AltUnlabeled != nil, c10.NoString != nil, c10.EMSRAdditionalPFCs1 != nil)
	r10 := c10.EMSRAdditionalPFCs1
	if x.populated {
		if len(r10.EMSRAdditionalPFCs1List) != 2 || len(r10.EMSRAdditionalPFCs2List) != 1 || len(r10.EMSRAdditionalPFCs3List) != 1 {
			t.Fatal("missing Rel-10 PFC lists")
		}
		a, b, c, d := r10.EMSRAdditionalPFCs1List[0].EMSRAdditionalPFCs1, r10.EMSRAdditionalPFCs1List[1].EMSRAdditionalPFCs1, r10.EMSRAdditionalPFCs2List[0].EMSRAdditionalPFCs2, r10.EMSRAdditionalPFCs3List[0].EMSRAdditionalPFCs3
		if a.DOWNLINKTFIASSIGNMENT != 3 || a.PFI != 13 || a.NPMTransferTime == nil || *a.NPMTransferTime != 27 || b.DOWNLINKTFIASSIGNMENT != 4 || b.PFI != 14 || b.NPMTransferTime != nil || c.DOWNLINKTFIASSIGNMENT != 5 || c.PFI != 17 || c.NPMTransferTime != nil || d.DOWNLINKTFIASSIGNMENT != 6 || d.PFI != 19 || d.NPMTransferTime == nil || *d.NPMTransferTime != 27 {
			t.Fatal("typed Rel-10 PFC values differ")
		}
	} else if len(r10.EMSRAdditionalPFCs1List)+len(r10.EMSRAdditionalPFCs2List)+len(r10.EMSRAdditionalPFCs3List) != 0 {
		t.Fatal("unexpected additional PFCs")
	}
}

func TestRRPacketDownlinkAssignmentReleaseVectors(t *testing.T) {
	for _, x := range rrAssignmentVectors {
		for phase := 0; phase < 8; phase++ {
			t.Run(fmt.Sprintf("%s/phase%d", x.name, phase), func(t *testing.T) {
				prefix := strings.Repeat("1", phase)
				raw := prefix + x.bits
				r := runtime.NewReader(rrAssignmentPack(raw + "10101"))
				if _, err := r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(x.bits))
				if err != nil {
					t.Fatal(err)
				}
				v, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				checkRRAssignmentFields(t, v, x)
				if r.Position() != len(raw) {
					t.Fatalf("position %d want %d", r.Position(), len(raw))
				}
				layout, err := runtime.SealBits(v, runtime.BitString{Bytes: rrAssignmentPack(raw), BitLength: len(raw)}, len(raw), runtime.BitString{}, r.Wire())
				if err != nil {
					t.Fatal(err)
				}
				if err := r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				sentinel, err := r.ReadUint(5)
				if err != nil || sentinel != 21 {
					t.Fatal("following sentinel consumed")
				}
				w := runtime.NewWriter()
				w.WithWire(layout)
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				limit, err := w.PushLimit(len(x.bits))
				if err != nil {
					t.Fatal(err)
				}
				if err := EncodeRRPacketDownlinkAssignmentValuePartTo(w, v); err != nil {
					t.Fatal(err)
				}
				if err := w.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				replay, err := w.FinishBits(runtime.BitString{})
				if err != nil || replay.BitLength != len(raw) || !bytes.Equal(replay.Bytes, rrAssignmentPack(raw)) {
					t.Fatalf("bit replay %x/%d: %v", replay.Bytes, replay.BitLength, err)
				}
				fresh := runtime.Canonical(v)
				canonicalBits := len(raw)
				if x.terminal == "error-region" {
					// The ignored tail is received layout, not fresh semantic content.
					canonicalBits -= 5
				}
				w = runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(fresh, canonicalBits).Wire)
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				limit, err = w.PushLimit(canonicalBits - phase)
				if err != nil {
					t.Fatal(err)
				}
				if err := EncodeRRPacketDownlinkAssignmentValuePartTo(w, fresh); err != nil {
					t.Fatal(err)
				}
				if err := w.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				canonical, err := w.FinishBits(runtime.BitString{})
				expected := raw
				switch x.terminal {
				case "error-region":
					expected = raw[:len(raw)-5]
				case "spare":
					expected = raw[:len(raw)-5] + "00000"
				}
				if err != nil || canonical.BitLength != canonicalBits || !bytes.Equal(canonical.Bytes, rrAssignmentPack(expected)) {
					t.Fatalf("canonical %x/%d want %x: %v", canonical.Bytes, canonical.BitLength, rrAssignmentPack(expected), err)
				}
				again, err := runtime.NewReaderBits(canonical.Bytes, canonicalBits, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := again.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				decoded, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(again)
				if err != nil || !runtime.SemanticallyEqual(v, decoded) {
					t.Fatalf("canonical changed typed fields: %v", err)
				}
				// A changed base field must be encoded even when the received layout is retained.
				changed := v
				changed.TIMESLOTALLOCATION ^= 0xff
				w = runtime.NewWriter()
				w.WithWire(layout)
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				if _, err := w.PushLimit(len(x.bits)); err != nil {
					t.Fatal(err)
				}
				if err := EncodeRRPacketDownlinkAssignmentValuePartTo(w, changed); err != nil {
					t.Fatal(err)
				}
				edited, err := w.FinishBits(runtime.BitString{})
				wanted := prefix + x.bits[:3] + fmt.Sprintf("%08b", uint8(0xa5)^0xff) + x.bits[11:]
				if err != nil || !bytes.Equal(edited.Bytes, rrAssignmentPack(wanted)) {
					t.Fatalf("base field edit %x want %x: %v", edited.Bytes, rrAssignmentPack(wanted), err)
				}
			})
		}
	}
}

func TestRRPacketDownlinkAssignmentTruncationAndLiterals(t *testing.T) {
	for _, x := range rrAssignmentVectors {
		for cut := 0; cut < x.mandatory; cut++ {
			r, err := runtime.NewReaderBits(rrAssignmentPack(x.bits), cut, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(r); err == nil {
				t.Fatalf("%s accepted mandatory truncation%d", x.name, cut)
			}
			if r.Position() > cut {
				t.Fatal("decoder crossed supplied bit extent")
			}
		}
		for _, segment := range x.segments {
			for cut := segment[0] + 1; cut < segment[1]; cut++ {
				r, err := runtime.NewReaderBits(rrAssignmentPack(x.bits), cut, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(r); err == nil {
					t.Fatalf("%s accepted incomplete release at%d", x.name, cut)
				}
			}
		}
		wrong := x.bits[:x.mandatory-1] + "1" + x.bits[x.mandatory:]
		r, err := runtime.NewReaderBits(rrAssignmentPack(wrong), len(wrong), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(r); err == nil {
			t.Fatalf("%s accepted reserved base literal", x.name)
		}
	}
}

func TestRRPacketDownlinkAssignmentByteBoundsAndGrowth(t *testing.T) {
	var x rrAssignmentVector
	for _, candidate := range rrAssignmentVectors {
		if candidate.depth == 5 && candidate.populated {
			x = candidate
			break
		}
	}
	// Rel-10 ends in spare bits: byte completion and growth to the actual containing value are legal.
	wire := rrAssignmentPack(x.bits)
	got, err := DecodeRRPacketDownlinkAssignmentValuePart(wire)
	if err != nil {
		t.Fatal(err)
	}
	checkRRAssignmentFields(t, got.Value, x)
	replay, err := EncodeRRPacketDownlinkAssignmentValuePart(got.Value)
	if err != nil || !bytes.Equal(replay, wire) {
		t.Fatalf("byte replay %x: %v", replay, err)
	}
	for _, n := range []int{len(wire), len(wire) + 1, 255} {
		encoded, err := EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(got.Value, n)
		if err != nil || len(encoded) != n {
			t.Fatalf("extent%d %x: %v", n, encoded, err)
		}
		again, err := DecodeRRPacketDownlinkAssignmentValuePart(encoded)
		if err != nil || !runtime.SemanticallyEqual(got.Value, again.Value) {
			t.Fatalf("extent%d changed typed fields: %v", n, err)
		}
	}
	for _, n := range []int{-1, 0, len(wire) - 2, 256, 65536} {
		if _, err := EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(got.Value, n); err == nil {
			t.Fatalf("accepted invalid extent%d", n)
		}
	}
	_, err = EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(got.Value, 256)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Maximum != 255 {
		t.Fatalf("missing type4 value ceiling: %v", err)
	}
	_, err = EncodeRRPacketDownlinkAssignmentValuePartCanonical(got.Value)
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("missing containing extent requirement: %v", err)
	}
	for _, change := range []func(*RRPacketDownlinkAssignmentValuePart){
		func(v *RRPacketDownlinkAssignmentValuePart) { v.MACMODE = 4 },
		func(v *RRPacketDownlinkAssignmentValuePart) { v.RLCMODE = 2 },
		func(v *RRPacketDownlinkAssignmentValuePart) { *v.PacketTimingAdvance.TIMINGADVANCEVALUE = 64 },
		func(v *RRPacketDownlinkAssignmentValuePart) { v.P0Group.P0 = 16 },
		func(v *RRPacketDownlinkAssignmentValuePart) { v.PowerControlParameters.ALPHA = 16 },
		func(v *RRPacketDownlinkAssignmentValuePart) { *v.DOWNLINKTFIASSIGNMENT = 32 },
		func(v *RRPacketDownlinkAssignmentValuePart) {
			v.EGPRSWindowSizeChoice.EGPRSWindowSize.EGPRSWindowSizeGroup.EGPRSWindowSize.EGPRSWindowSize = 31
		},
	} {
		v := runtime.Canonical(got.Value)
		change(&v)
		if _, err := EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(v, len(wire)); err == nil {
			t.Fatal("invalid local or imported typed field accepted")
		}
	}
	grown := got.Value
	r10 := grown.EGPRSWindowSizeChoice.EGPRSWindowSize.MultipleDownlinkTBFAssignmentChoice.MultipleDownlinkTBFAssignment.FANRChoice.FANR.MeasurementControlEUTRANChoice.MeasurementControlEUTRAN.RLCEntity2Choice.RLCEntity2.EMSRAdditionalPFCs1Choice.EMSRAdditionalPFCs1
	r10.EMSRAdditionalPFCs1List = append(r10.EMSRAdditionalPFCs1List, r10.EMSRAdditionalPFCs1List[0])
	if _, err := EncodeRRPacketDownlinkAssignmentValuePart(grown); err == nil {
		t.Fatal("plain encoding silently changed a received list boundary")
	}
	encoded, err := EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(grown, len(wire)+4)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeRRPacketDownlinkAssignmentValuePart(encoded)
	if err != nil {
		t.Fatal(err)
	}
	list := again.Value.EGPRSWindowSizeChoice.EGPRSWindowSize.MultipleDownlinkTBFAssignmentChoice.MultipleDownlinkTBFAssignment.FANRChoice.FANR.MeasurementControlEUTRANChoice.MeasurementControlEUTRAN.RLCEntity2Choice.RLCEntity2.EMSRAdditionalPFCs1Choice.EMSRAdditionalPFCs1.EMSRAdditionalPFCs1List
	if len(list) != 3 || list[2].EMSRAdditionalPFCs1.PFI != 13 {
		t.Fatal("canonical list growth lost typed PFC")
	}
}

func FuzzRRPacketDownlinkAssignment(f *testing.F) {
	for _, x := range rrAssignmentVectors {
		f.Add(rrAssignmentPack(x.bits))
	}
	f.Add([]byte{})
	f.Add([]byte{0xff})
	f.Add([]byte{0, 0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		got, err := DecodeRRPacketDownlinkAssignmentValuePart(data)
		if err != nil {
			return
		}
		encoded, err := EncodeRRPacketDownlinkAssignmentValuePart(got.Value)
		if err != nil || !bytes.Equal(encoded, data) {
			t.Fatalf("accepted input failed replay: %x %v", encoded, err)
		}
		if len(data) <= 255 {
			canonical, err := EncodeRRPacketDownlinkAssignmentValuePartCanonicalAtLength(got.Value, len(data))
			if err != nil {
				return
			}
			again, err := DecodeRRPacketDownlinkAssignmentValuePart(canonical)
			if err != nil || !runtime.SemanticallyEqual(got.Value, again.Value) {
				t.Fatalf("canonical typed fields changed: %v", err)
			}
		}
	})
}

func rrAssignmentExactBits(t *testing.T, bits string) RRPacketDownlinkAssignmentValuePart {
	t.Helper()
	r, err := runtime.NewReaderBits(rrAssignmentPack(bits), len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeRRPacketDownlinkAssignmentValuePartFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position() != len(bits) {
		t.Fatal("wrong exact-bit consumption")
	}
	layout, err := runtime.SealBits(v, runtime.BitString{Bytes: rrAssignmentPack(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	for _, canonical := range []bool{false, true} {
		value := v
		w := runtime.NewWriter()
		w.WithWire(layout)
		if canonical {
			value = runtime.Canonical(v)
			w.WithWire(runtime.CanonicalTo(value, len(bits)).Wire)
		}
		old, err := w.PushLimit(len(bits))
		if err != nil {
			t.Fatal(err)
		}
		if err := EncodeRRPacketDownlinkAssignmentValuePartTo(w, value); err != nil {
			t.Fatal(err)
		}
		if err := w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		encoded, err := w.FinishBits(runtime.BitString{})
		if err != nil || encoded.BitLength != len(bits) || !bytes.Equal(encoded.Bytes, rrAssignmentPack(bits)) {
			t.Fatalf("canonical%v exact bits differ: %x/%d %v", canonical, encoded.Bytes, encoded.BitLength, err)
		}
	}
	return v
}

func TestRRPacketDownlinkAssignmentImportedFieldArms(t *testing.T) {
	// TS 44.060 V19.0.0 §§12.12 and 12.13: each Timing Advance
	// field and every per-timeslot GAMMA has its own independent presence bit.
	for timing := 0; timing < 4; timing++ {
		for mask := 0; mask < 256; mask++ {
			bits := "10110100101"
			if timing&1 != 0 {
				bits += "1" + fmt.Sprintf("%06b", 37)
			} else {
				bits += "0"
			}
			if timing&2 != 0 {
				bits += "1" + "1001" + "101"
			} else {
				bits += "0"
			}
			bits += "0" + "1" + "1001"
			for tn := 0; tn < 8; tn++ {
				if mask&(1<<tn) != 0 {
					bits += "1" + fmt.Sprintf("%05b", (7+3*tn)%32)
				} else {
					bits += "0"
				}
			}
			bits += "00"
			v := rrAssignmentExactBits(t, bits)
			ta := v.PacketTimingAdvance
			if (ta.TIMINGADVANCEVALUE != nil) != (timing&1 != 0) || (ta.TIMINGADVANCEINDEXGroup != nil) != (timing&2 != 0) {
				t.Fatalf("timing presence%d differs", timing)
			}
			if ta.TIMINGADVANCEVALUE != nil && *ta.TIMINGADVANCEVALUE != 37 || ta.TIMINGADVANCEINDEXGroup != nil && (ta.TIMINGADVANCEINDEXGroup.TIMINGADVANCEINDEX != 9 || ta.TIMINGADVANCEINDEXGroup.TIMINGADVANCETIMESLOTNUMBER != 5) {
				t.Fatal("imported timing values differ")
			}
			p := v.PowerControlParameters
			if p == nil || p.ALPHA != 9 {
				t.Fatal("missing typed power IE")
			}
			gamma := []*uint8{p.GAMMATN0, p.GAMMATN1, p.GAMMATN2, p.GAMMATN3, p.GAMMATN4, p.GAMMATN5, p.GAMMATN6, p.GAMMATN7}
			for tn, value := range gamma {
				if (value != nil) != (mask&(1<<tn) != 0) || value != nil && *value != uint8((7+3*tn)%32) {
					t.Fatalf("timing%d mask%x TN%d differs", timing, mask, tn)
				}
			}
		}
	}
}

func TestRRPacketDownlinkAssignmentIndependentReleaseOptions(t *testing.T) {
	const base = "10110100101000000"
	for window := 0; window < 2; window++ {
		for timing := 0; timing < 2; timing++ {
			bits := base
			if window != 0 {
				bits += "1" + "01101" + "10"
			} else {
				bits += "0"
			}
			if timing != 0 {
				bits += "1" + "10"
			} else {
				bits += "0"
			}
			v := rrAssignmentExactBits(t, bits)
			r99 := v.EGPRSWindowSizeChoice.EGPRSWindowSize
			if r99 == nil || (r99.EGPRSWindowSizeGroup != nil) != (window != 0) || (r99.PacketExtendedTimingAdvance != nil) != (timing != 0) || r99.MultipleDownlinkTBFAssignmentChoice.AltUnlabeled == nil {
				t.Fatal("independent R99 options differ")
			}
		}
	}
	// A present Multiple Downlink TBF Assignment can itself have an empty inner list.
	v := rrAssignmentExactBits(t, base+"00"+"1"+"1"+"0"+"10000001"+"0"+"0")
	list := v.EGPRSWindowSizeChoice.EGPRSWindowSize.MultipleDownlinkTBFAssignmentChoice.MultipleDownlinkTBFAssignment.MultipleDownlinkTBFAssignmentList
	if len(list) != 1 || list[0].MultipleDownlinkTBFAssignment.UplinkControlTimeslot != nil || list[0].MultipleDownlinkTBFAssignment.TIMESLOTALLOCATION != 0x81 || len(list[0].MultipleDownlinkTBFAssignment.DownlinkTBFAssignmentList) != 0 {
		t.Fatal("nested empty assignment list differs")
	}
	// EMST can carry RLC Entity 2 while omitting the independently optional Entity 3.
	for _, x := range rrAssignmentVectors {
		if x.depth == 4 && x.populated && x.terminal == "null" {
			bits := x.bits[:len(x.bits)-15] + "0"
			v := rrAssignmentExactBits(t, bits)
			r9 := v.EGPRSWindowSizeChoice.EGPRSWindowSize.MultipleDownlinkTBFAssignmentChoice.MultipleDownlinkTBFAssignment.FANRChoice.FANR.MeasurementControlEUTRANChoice.MeasurementControlEUTRAN.RLCEntity2Choice.RLCEntity2
			if r9.RLCEntity2Group == nil || r9.RLCEntity2Group.RLCEntity3 != nil || r9.RLCEntity2Group.RLCEntity2.PFI != 71 {
				t.Fatal("independent RLC Entity 3 absence differs")
			}
		}
	}
}
