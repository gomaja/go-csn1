package registry_test

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/pshandover"
)

// TS 44.060 V19.0.0 §12.42 pp23778–23792: N is the current
// global allocation's population; M is the current nonempty TBF subset.
func handoverAllocationSenderWitnesses(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../ts44060/pshandover/testdata/handover_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		Name, Bits string
		N, M       int
	}
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	var bits string
	for _, v := range vectors {
		if v.N == 2 && v.M == 2 && strings.Contains(v.Name, "mode0-bitmap129-power0-subset3-usfs1-rich0") {
			bits = v.Bits
			break
		}
	}
	if bits == "" {
		t.Fatal("missing source allocation witness")
	}
	wire := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	seed := func() pshandover.PSHandoverRadioResourcesIE {
		r, e := runtime.NewReaderBits(wire, len(bits), nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.PushLimit(len(bits)); e != nil {
			t.Fatal(e)
		}
		v, e := pshandover.DecodePSHandoverRadioResourcesIEFrom(r)
		if e != nil {
			t.Fatal(e)
		}
		return runtime.Canonical(v)
	}
	encode := func(v pshandover.PSHandoverRadioResourcesIE) error {
		return pshandover.EncodePSHandoverRadioResourcesIETo(runtime.NewWriter(), v)
	}
	v := seed()
	if err = encode(v); err != nil {
		t.Fatalf("valid source witness: %v", err)
	}
	group := v.GPRSModeChoice.GPRSMode.GPRSMode.GlobalTimeslotDescriptionGroup
	if group == nil || len(group.UplinkAssignmentList) == 0 {
		t.Fatal("missing typed allocation")
	}
	assignment := group.UplinkAssignmentList[0].UplinkAssignment
	err = pshandover.EncodeUplinkTBFAssignmentStructTo(runtime.NewWriter(), assignment)
	requireSenderWitnessError(t, err, "12.42", "source allocation or repeated scope is absent", true)
	if _, err = pshandover.EncodeUplinkTBFAssignmentStructCanonicalAtLength(assignment, 8); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatalf("standalone canonical context: %v", err)
	}
	group.GlobalTimeslotDescription.MSTIMESLOTALLOCATIONChoice.MSTIMESLOTALLOCATION.MSTIMESLOTALLOCATION = 0
	requireSenderWitnessError(t, encode(v), "12.42", "allocation population is below its source minimum", false)
	v = seed()
	subset := v.GPRSModeChoice.GPRSMode.GPRSMode.GlobalTimeslotDescriptionGroup.UplinkAssignmentList[0].UplinkAssignment.TBFTIMESLOTALLOCATION
	if subset == nil {
		t.Fatal("missing source TBF subset")
	}
	subset.Bytes = make([]byte, len(subset.Bytes))
	requireSenderWitnessError(t, encode(v), "12.42", "TBF allocation must assign at least one timeslot", false)
}
