package pshandover2

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.42a pp23920–23927 permits a Release 9 prefix
// ending before the Release 11 field. The omitted field is inferred as zero.
func TestReleasedExtensionPrefix(t *testing.T) {
	bits := "0000000000101100000100001000000000000000001010000100010011000"
	d, err := Lookup("PS Handover Radio Resources 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	for phase := 0; phase < 8; phase++ {
		prefix := strings.Repeat("0", phase)
		r, err := runtime.NewChoiceReaderBits(handoverBits(prefix+bits+"101"), phase+len(bits)+3, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		limit, err := r.PushLimit(len(bits))
		if err != nil {
			t.Fatal(err)
		}
		value, err := DecodePSHandoverRadioResources2IEFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		fields := map[string][]uint64{}
		handoverFields(reflect.ValueOf(value), fields)
		if got := fields["MTTIDOWNLINKASSIGNMENTC1"]; len(got) != 1 || got[0] != 1 {
			t.Fatalf("phase %d typed release prefix lost: %v", phase, got)
		}
		if got := fields["EGPRSPacketDownlinkAckNackType3Support"]; len(got) != 1 || got[0] != 0 {
			t.Fatalf("phase %d omitted release field: %v", phase, got)
		}
		if err := r.PopLimit(limit); err != nil {
			t.Fatal(err)
		}
		if sentinel, err := r.ReadUint(3); err != nil || sentinel != 5 {
			t.Fatalf("phase %d consumed outside sentinel: %v", phase, err)
		}
		for _, canonical := range []bool{false, true} {
			w := runtime.NewWriter()
			if err := w.WriteLiteral(prefix); err != nil {
				t.Fatal(err)
			}
			bound, err := w.PushLimit(len(bits))
			if err != nil {
				t.Fatal(err)
			}
			v := value
			if canonical {
				v = runtime.Canonical(v)
			}
			if err := EncodePSHandoverRadioResources2IETo(w, v); err != nil {
				t.Fatalf("phase %d canonical %v: %v", phase, canonical, err)
			}
			if err := w.PopLimit(bound); err != nil {
				t.Fatal(err)
			}
			replay, err := w.FinishBits(runtime.BitString{})
			if err != nil || replay.BitLength != phase+len(bits) || !bytes.Equal(replay.Bytes, handoverBits(prefix+bits)) {
				t.Fatalf("phase %d canonical %v exact prefix replay: %v", phase, canonical, err)
			}
		}
	}
	// Also exercise the owned descriptor entry point used by the source witness.
	if _, err := d.DecodeBits(handoverBits(bits), len(bits)); err != nil {
		t.Fatal(err)
	}
}
