package runtime

import (
	"bytes"
	"testing"
)

// TS 44.060 V19.0.0 §12.5.5 pp22504/22514, §11.2.37 p20100
// and §12.42a p23904: these printed labels identify reported receiver regions.
// The application implements the associated procedure (§11.1.2 p13020).
func TestDesignationErrorLabels(t *testing.T) {
	for _, name := range []string{"PDCH pairs configuration error", "Frequency parameters error", "Frequency Parameters error"} {
		t.Run(name, func(t *testing.T) {
			c := NewReceiverCollector()
			r, err := NewChoiceReaderBits([]byte{0xd4}, 6, c)
			if err != nil {
				t.Fatal(err)
			}
			region, err := r.ReadErrorRegion(ErrorLabel(name), -1, Source{Standard: "TS 44.060", Version: "19.0.0", Clause: "11.1.2"})
			if err != nil {
				t.Fatal(err)
			}
			report := c.TakeReceiverReport()
			if len(report.Regions) != 1 || report.Regions[0].Label != ErrorLabel(name) || report.Regions[0].Location.Width != 6 {
				t.Fatalf("report: %+v", report)
			}
			w := NewWriter()
			if err := w.WriteErrorRegion(region); err != nil {
				t.Fatal(err)
			}
			out, err := w.FinishBits(BitString{})
			if err != nil || out.BitLength != 6 || !bytes.Equal(out.Bytes, []byte{0xd4}) {
				t.Fatalf("replay %+v %v", out, err)
			}
		})
	}
	for _, name := range []string{"", "Unknown error", "PDCH pairs configuration Error", "Frequency Parameters Error"} {
		if _, err := NewReader(nil).ReadErrorRegion(ErrorLabel(name), 0, Source{}); err == nil {
			t.Fatalf("unregistered label %q accepted", name)
		}
	}
}
