package runtime

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDescriptorSourceOriginCollectorBudgets(t *testing.T) {
	for _, bits := range []int{0, 1, 6, 170} {
		for _, longPath := range []bool{false, true} {
			t.Run(fmt.Sprintf("bits=%d/long_path=%t", bits, longPath), func(t *testing.T) {
				var wantReport Report
				var wantReceiver ReceiverReport
				for _, origin := range []int{0, 14, maxBits - bits} {
					data := make([]byte, (bits+7)/8+1) // Completion and sentinel are outside the extent.
					data[len(data)-1] = 0xff
					d := Descriptor{SourceOrigin: origin, DecodeFrom: func(r *Reader) (any, error) {
						value, err := r.ReadBitString(bits)
						if err != nil || !r.Collecting() {
							return value, err
						}
						c := r.collector.observationCollector()
						if c.eventLimit != int32(bits) || c.pathBudget != bits*128 {
							t.Errorf("origin %d: observation budgets = %d/%d, want %d/%d", origin, c.eventLimit, c.pathBudget, bits, bits*128)
						}
						path := "Field"
						visits := bits + 2
						if longPath {
							path = strings.Repeat("x", bits*32+1)
							visits = 2
							if err = r.Enter(path); err != nil {
								return nil, err
							}
							defer r.Leave()
						}
						if bits > 0 {
							for range visits {
								if err = r.Observe(origin, bits, 0, Source{}, path); err != nil {
									return nil, err
								}
							}
						}
						for range bits + 130 {
							if _, err = r.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
								return nil, err
							}
						}
						budget := r.collector.receiver.budget
						if budget.limit != bits+128 || budget.pathBudget != (bits+128)*128 {
							t.Errorf("origin %d: receiver budgets = %d/%d", origin, budget.limit, budget.pathBudget)
						}
						return value, nil
					}}
					plain, err := d.DecodeBits(data, bits)
					if err != nil {
						t.Fatal(err)
					}
					value, report, receiver, err := d.DecodeBitsWithReport(data, bits)
					if err != nil || !reflect.DeepEqual(value, plain) {
						t.Fatalf("origin %d: decode parity: %v", origin, err)
					}
					for i := range report.Observations {
						report.Observations[i].Location.BitOffset -= origin
					}
					for i := range receiver.Regions {
						receiver.Regions[i].Location.BitOffset -= origin
					}
					if origin == 0 {
						wantReport, wantReceiver = report, receiver
					} else if !reflect.DeepEqual(report, wantReport) || !reflect.DeepEqual(receiver, wantReceiver) {
						t.Errorf("origin %d changed bounded reports: observations %d/%d, regions %d/%d, dropped %d/%d versus %d/%d", origin, len(report.Observations), len(wantReport.Observations), len(receiver.Regions), len(wantReceiver.Regions), report.Dropped, receiver.Dropped, wantReport.Dropped, wantReceiver.Dropped)
					}
				}
			})
		}
	}
}
