package restoctets

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"
)

func TestPacketUplinkAssignmentUSF(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16, Packet Uplink Assignment and
	// table 10.5.2.16.1. These synthetic encodings and every field below
	// were independently checked with pycrate 0.7.11 ia_rest_octets.
	pointer := func(v uint8) *uint8 { return &v }
	start := uint16(0x1234)
	for _, tc := range []struct {
		name, wire     string
		usf            uint8
		branch         PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENT
		extendedRA     *uint8
		pfi            *uint8
		raAlternative  PacketUplinkAssignmentExtendedRAChoiceAlternative
		pfiAlternative PacketUplinkAssignmentPFIChoiceAlternative
	}{
		{name: "usf5", wire: "80a000", usf: 5},
		{name: "granularity1", wire: "80f544", usf: 7, branch: PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENT{USFGRANULARITY: 1, CHANNELCODINGCOMMAND: 2, TLLIBLOCKCHANNELCODING: 1, GAMMA: 17}},
		{name: "p0-and-all-optionals", wire: "b66cede3a891a0", usf: 3, branch: PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENT{TFIASSIGNMENT: 13, POLLING: 1, P0Group: &PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENTP0Group{P0: 9, PRMODE: 1}, CHANNELCODINGCOMMAND: 2, TLLIBLOCKCHANNELCODING: 1, ALPHA: pointer(7), GAMMA: 17, TIMINGADVANCEINDEX: pointer(10), TBFSTARTINGTIME: &start}, raAlternative: PacketUplinkAssignmentExtendedRAChoiceAlternativeAltL, pfiAlternative: PacketUplinkAssignmentPFIChoiceAlternativePFI},
		{name: "extended-ra", wire: "804000eb", usf: 2, extendedRA: pointer(21), raAlternative: PacketUplinkAssignmentExtendedRAChoiceAlternativeExtendedRA, pfiAlternative: PacketUplinkAssignmentPFIChoiceAlternativeAltL},
		{name: "pfi", wire: "8080007540", usf: 4, pfi: pointer(85), raAlternative: PacketUplinkAssignmentExtendedRAChoiceAlternativeAltL, pfiAlternative: PacketUplinkAssignmentPFIChoiceAlternativePFI},
		{name: "both-extensions-and-p0", wire: "80db00071648", usf: 6, branch: PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENT{USFGRANULARITY: 1, P0Group: &PacketUplinkAssignmentTFIASSIGNMENTChoiceTFIASSIGNMENTP0Group{P0: 6}}, extendedRA: pointer(17), pfi: pointer(73), raAlternative: PacketUplinkAssignmentExtendedRAChoiceAlternativeExtendedRA, pfiAlternative: PacketUplinkAssignmentPFIChoiceAlternativePFI},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := hex.DecodeString(tc.wire)
			if err != nil {
				t.Fatal(err)
			}
			d, err := DecodePacketUplinkAssignment(wire)
			if err != nil {
				t.Fatal(err)
			}
			branch := d.Value.TFIASSIGNMENTChoice.TFIASSIGNMENT
			if branch == nil {
				t.Fatal("TFI assignment branch missing")
			}
			// Reflection keeps the regression test executable if a source
			// mutation removes USF from the generated type entirely.
			gotUSF := reflect.ValueOf(*branch).FieldByName("USF")
			if !gotUSF.IsValid() || gotUSF.Kind() != reflect.Uint8 || gotUSF.Uint() != uint64(tc.usf) {
				t.Fatalf("USF missing or incorrect: want %d, branch %+v", tc.usf, branch)
			}
			reflect.ValueOf(&tc.branch).Elem().FieldByName("USF").SetUint(uint64(tc.usf))
			if !reflect.DeepEqual(*branch, tc.branch) {
				t.Fatalf("assignment fields = %+v, want %+v", *branch, tc.branch)
			}
			// Build both extension choices only from the oracle expectations,
			// including absent, L and H-with-absent-payload alternatives.
			wantRA := PacketUplinkAssignmentExtendedRAChoice{Alternative: tc.raAlternative}
			switch tc.raAlternative {
			case PacketUplinkAssignmentExtendedRAChoiceAlternativeAltUnlabeled:
				wantRA.AltUnlabeled = &struct{}{}
			case PacketUplinkAssignmentExtendedRAChoiceAlternativeAltL:
				wantRA.AltL = &struct{}{}
			case PacketUplinkAssignmentExtendedRAChoiceAlternativeExtendedRA:
				wantRA.ExtendedRA = &PacketUplinkAssignmentExtendedRAChoiceExtendedRA{ExtendedRA: tc.extendedRA}
			}
			wantPFI := PacketUplinkAssignmentPFIChoice{Alternative: tc.pfiAlternative}
			switch tc.pfiAlternative {
			case PacketUplinkAssignmentPFIChoiceAlternativeAltUnlabeled:
				wantPFI.AltUnlabeled = &struct{}{}
			case PacketUplinkAssignmentPFIChoiceAlternativeAltL:
				wantPFI.AltL = &struct{}{}
			case PacketUplinkAssignmentPFIChoiceAlternativePFI:
				wantPFI.PFI = &PacketUplinkAssignmentPFIChoicePFI{PFI: tc.pfi}
			}
			if !reflect.DeepEqual(d.Value.ExtendedRAChoice, wantRA) || !reflect.DeepEqual(d.Value.PFIChoice, wantPFI) {
				t.Fatalf("extension choices = %+v, %+v; want %+v, %+v", d.Value.ExtendedRAChoice, d.Value.PFIChoice, wantRA, wantPFI)
			}
			encoded, err := EncodePacketUplinkAssignment(d.Value)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("round trip %x -> %x: %v", wire, encoded, err)
			}
			// Construct a fresh assignment without received wire state.
			fresh := PacketUplinkAssignment{
				TFIASSIGNMENTChoice: PacketUplinkAssignmentTFIASSIGNMENTChoice{Alternative: PacketUplinkAssignmentTFIASSIGNMENTChoiceAlternativeTFIASSIGNMENT, TFIASSIGNMENT: &tc.branch},
				ExtendedRAChoice:    wantRA,
				PFIChoice:           wantPFI,
			}
			encoded, err = EncodePacketUplinkAssignment(fresh)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("fresh encoding %x -> %x: %v", wire, encoded, err)
			}
		})
	}
}

func FuzzPacketUplinkAssignmentUSF(f *testing.F) {
	for _, wire := range []string{"80a000", "80f544", "b66cede3a891a0", "804000eb", "8080007540", "80db00071648"} {
		data, err := hex.DecodeString(wire)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		d, err := DecodePacketUplinkAssignment(data)
		if err != nil {
			return
		}
		encoded, err := EncodePacketUplinkAssignment(d.Value)
		if err != nil || !bytes.Equal(encoded, data) {
			t.Fatalf("round trip %x -> %x: %v", data, encoded, err)
		}
	})
}
