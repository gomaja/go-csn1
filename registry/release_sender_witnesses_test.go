package registry_test

import (
	"encoding/hex"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/packetcsreleaseindication"
)

// TS 44.060 V19.0.0 §11.2.37 pp20193–20207 derives the TBF bitmap
// from the preceding global timeslot description in this synthetic source arm.
func releaseAllocationSenderSeed(t *testing.T) packetcsreleaseindication.PacketCSReleaseMessageContent {
	t.Helper()
	wire, err := hex.DecodeString("00010220a80608e2")
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewChoiceReaderBits(wire, 64, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(50); err != nil {
		t.Fatal(err)
	}
	v, err := packetcsreleaseindication.DecodePacketCSReleaseMessageContentFrom(r)
	if err != nil || r.Position() != 64 {
		t.Fatalf("typed release allocation seed: position %d, %v", r.Position(), err)
	}
	return runtime.CanonicalTo(v, 64)
}

func encodeReleaseAllocationWitness(v packetcsreleaseindication.PacketCSReleaseMessageContent) error {
	w := runtime.NewWriter()
	w.WithWire(v.Wire)
	if err := w.WriteUint(0, 14); err != nil {
		return err
	}
	if _, err := w.PushLimit(50); err != nil {
		return err
	}
	if err := packetcsreleaseindication.EncodePacketCSReleaseMessageContentTo(w, v); err != nil {
		return err
	}
	_, err := w.FinishBits(runtime.BitString{})
	return err
}

func releaseAllocationSenderWitnesses(t *testing.T, record func(string)) {
	t.Helper()
	v := releaseAllocationSenderSeed(t)
	if err := encodeReleaseAllocationWitness(v); err != nil {
		t.Fatalf("valid release source allocation control: %v", err)
	}
	group := v.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known.PaddingBitsChoice.FrequencyParameters.FrequencyParametersChoice.CHANNELCODINGCOMMAND.FrequencyParametersGroupFallback.Known.CHANNELCODINGCOMMANDGroup.MultipleUplinkAssignment.GlobalTimeslotDescriptionGroup
	if group == nil || len(group.UplinkTBFAssignmentList) != 1 || group.UplinkTBFAssignmentList[0].UplinkTBFAssignment.TBFTIMESLOTALLOCATION == nil {
		t.Fatal("seed lacks its typed TBF subset")
	}
	u := group.UplinkTBFAssignmentList[0].UplinkTBFAssignment
	w := runtime.NewWriter()
	w.Set("N", 8)
	w.Set("M", 8)
	err := packetcsreleaseindication.EncodeUplinkTBFAssignmentStructTo(w, u)
	requireSenderWitnessError(t, err, "11.2.37", "source allocation or repeated scope is absent", true)
	record("TS 44.060 V19.0.0 §11.2.37 pp20193–20207: source allocation or repeated scope is absent: %w")
	group.GlobalTimeslotDescription.MSTIMESLOTALLOCATIONChoice.MSTIMESLOTALLOCATION.MSTIMESLOTALLOCATION = 0
	err = encodeReleaseAllocationWitness(v)
	requireSenderWitnessError(t, err, "11.2.37", "allocation population is below its source minimum", false)
	record("TS 44.060 V19.0.0 §11.2.37 pp20193–20207: allocation population is below its source minimum")
	v = releaseAllocationSenderSeed(t)
	group = v.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known.PaddingBitsChoice.FrequencyParameters.FrequencyParametersChoice.CHANNELCODINGCOMMAND.FrequencyParametersGroupFallback.Known.CHANNELCODINGCOMMANDGroup.MultipleUplinkAssignment.GlobalTimeslotDescriptionGroup
	subset := group.UplinkTBFAssignmentList[0].UplinkTBFAssignment.TBFTIMESLOTALLOCATION
	subset.Bytes = make([]byte, len(subset.Bytes))
	err = encodeReleaseAllocationWitness(v)
	requireSenderWitnessError(t, err, "11.2.37", "TBF allocation must assign at least one timeslot", false)
	record("TS 44.060 V19.0.0 §11.2.37 pp20193–20207: TBF allocation must assign at least one timeslot")
}

func TestReleaseAllocationSenderWitnesses(t *testing.T) {
	observed := map[string]bool{}
	releaseAllocationSenderWitnesses(t, func(message string) { observed[message] = true })
	if len(observed) != 3 {
		t.Fatalf("got %d release allocation witnesses, want 3", len(observed))
	}
}
