package pshandover

import (
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.42 pp23778–23792 derives N from the current
// global allocation and M from the current nonempty TBF subset. These tests
// edit an accepted source value; no application N/M is supplied.
func TestAllocationSenderRefusals(t *testing.T) {
	var witness clauseVector
	for _, v := range vectors(t) {
		if v.N == 2 && v.M == 2 && strings.Contains(v.Name, "mode0-bitmap129-power0-subset3-usfs1-rich0") {
			witness = v
			break
		}
	}
	if witness.Bits == "" {
		t.Fatal("missing two-timeslot source witness")
	}
	for _, tc := range []struct{ name, want string }{
		{"unscoped helper", "source allocation or repeated scope is absent"},
		{"empty global allocation", "allocation population is below its source minimum"},
		{"empty TBF allocation", "TBF allocation must assign at least one timeslot"},
		{"oversized TBF allocation", "invalid bit string width"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := runtime.NewReaderBits(packed(witness.Bits), len(witness.Bits), nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(len(witness.Bits)); err != nil {
				t.Fatal(err)
			}
			v, err := DecodePSHandoverRadioResourcesIEFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			v = runtime.Canonical(v)
			global := v.GPRSModeChoice.GPRSMode.GPRSMode.GlobalTimeslotDescriptionGroup
			assignment := &global.UplinkAssignmentList[0].UplinkAssignment
			w := runtime.NewWriter()
			switch tc.name {
			case "unscoped helper":
				err = EncodeUplinkTBFAssignmentStructTo(w, *assignment)
				if !errors.Is(err, runtime.ErrContextRequired) {
					t.Fatalf("standalone helper omitted missing allocation: %v", err)
				}
				if out, canonicalErr := EncodeUplinkTBFAssignmentStructCanonicalAtLength(*assignment, 8); out != nil || !errors.Is(canonicalErr, runtime.ErrContextRequired) {
					t.Fatalf("standalone canonical helper: output=%v error=%v", out, canonicalErr)
				}
			case "empty global allocation":
				global.GlobalTimeslotDescription.MSTIMESLOTALLOCATIONChoice.MSTIMESLOTALLOCATION.MSTIMESLOTALLOCATION = 0
				err = EncodePSHandoverRadioResourcesIETo(w, v)
			case "empty TBF allocation":
				assignment.TBFTIMESLOTALLOCATION = &runtime.BitString{Bytes: []byte{0}, BitLength: 2}
				err = EncodePSHandoverRadioResourcesIETo(w, v)
			case "oversized TBF allocation":
				assignment.TBFTIMESLOTALLOCATION = &runtime.BitString{Bytes: make([]byte, 9), BitLength: 65}
				err = EncodePSHandoverRadioResourcesIETo(w, v)
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("sender refusal: %v, want %q", err, tc.want)
			}
		})
	}
}
