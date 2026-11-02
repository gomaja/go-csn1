package registry

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.2.31,12.42b and §11.1 p12944:
// only an observed, source-selected remainder permits this sender refusal.
func canonicalReservedRemainderRefusal(d runtime.Descriptor, wire []byte, value any, err error) bool {
	if d.Standard != "TS 44.060" || (d.Clause != "11.2.31" && d.Clause != "12.42b" && d.Clause != "11.2.29") {
		return false
	}
	switch d.Name {
	case "UFPS struct", "Carrier Specific Info struct", "DLMC UL Carrier Info struct", "Packet Timeslot Reconfigure message content", "PS Handover Radio Resources 3 IE", "Packet Uplink Assignment message content":
	default:
		return false
	}
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		return false
	}
	_, report, de := d.DecodeWithReport(wire)
	if de != nil || len(report.Observations) != 1 {
		return false
	}
	o := report.Observations[0]
	if o.Value != 3 || o.Location.Width < 2 || o.Location.Source.Standard != d.Standard || o.Location.Source.Version != d.Version || o.Location.Source.Clause != d.Clause {
		return false
	}
	_, refusal := d.CanonicalAtLength(value, len(wire))
	var operation *runtime.OperationError
	if !errors.As(refusal, &operation) || operation.Kind != runtime.InvalidValue {
		return false
	}
	for refusal != nil {
		if refusal.Error() == "reserved remainder has no sender encoding" {
			return true
		}
		refusal = errors.Unwrap(refusal)
	}
	return false
}

func TestCanonicalReservedRemainderProbe(t *testing.T) {
	for _, clause := range []string{"11.2.31", "12.42b"} {
		d, e := LookupClause("TS 44.060", clause, "UFPS struct")
		if e != nil {
			t.Fatal(e)
		}
		if ok, e := probeCanonical(d, []byte{0xff}); !ok || e != nil {
			t.Fatalf("source remainder: %t %v", ok, e)
		}
		for name, change := range map[string]func(*runtime.Descriptor){"unrelated definition": func(d *runtime.Descriptor) { d.Name = "unrelated" }, "unrelated refusal": func(d *runtime.Descriptor) {
			d.CanonicalAtLength = func(any, int) ([]byte, error) { return nil, errors.New("unexpected") }
		}, "missing observation": func(d *runtime.Descriptor) {
			decode := d.Decode
			d.DecodeWithCollector = func(b []byte, _ *runtime.Collector) (any, error) { return decode(b) }
		}} {
			t.Run(clause+"/"+name, func(t *testing.T) {
				changed := d
				change(&changed)
				if _, e := probeCanonical(changed, []byte{0xff}); e == nil {
					t.Fatal("unproved refusal accepted")
				}
			})
		}
	}
}
