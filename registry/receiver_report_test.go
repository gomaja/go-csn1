package registry

import (
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

func TestReceiverReportLookupFailure(t *testing.T) {
	_, reserved, receiver, err := DecodeBitsWithReport("missing", "definition", nil, 0)
	if err == nil || len(reserved.Observations) != 0 || len(receiver.Regions) != 0 || len(receiver.ZeroFill) != 0 {
		t.Fatal("lookup failure polluted report")
	}
}

func TestReceiverReportDescriptor(t *testing.T) {
	d := runtime.Descriptor{DecodeFrom: func(r *runtime.Reader) (any, error) {
		r.SetZeroExtension(true)
		return r.ReadUintWithFill(8, runtime.Source{Standard: "TS 44.060", Clause: "11.2.34"}, "Timing")
	}}
	value, reserved, report, err := d.DecodeBitsWithReport([]byte{0xff}, 3)
	if err != nil || value != uint64(224) || len(reserved.Observations) != 0 || len(report.ZeroFill) != 1 || report.ZeroFill[0].BitOffset != 3 || report.ZeroFill[0].Width != 5 {
		t.Fatalf("value=%v report=%+v error=%v", value, report, err)
	}
	if _, err = d.DecodeBits([]byte{0xff}, -1); err == nil {
		t.Fatal("invalid extent accepted")
	}
	if _, _, _, err = (runtime.Descriptor{}).DecodeBitsWithReport(nil, 0); err != runtime.ErrUnsupported {
		t.Fatalf("unsupported: %v", err)
	}
}
