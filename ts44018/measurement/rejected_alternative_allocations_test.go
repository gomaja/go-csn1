//go:build !race

package measurement

import (
	"encoding/hex"
	"testing"
)

func TestMeasurementResultsReportAllocations(t *testing.T) {
	want := []float64{8, 10, 13}
	for i, report := range measurementReports {
		t.Run(report.name, func(t *testing.T) {
			data, err := hex.DecodeString(report.hex)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeMeasurementResultsContents(data); err != nil {
				t.Fatal(err)
			}
			got := testing.AllocsPerRun(100, func() {
				if _, err := DecodeMeasurementResultsContents(data); err != nil {
					panic(err)
				}
			})
			if got != want[i] {
				t.Fatalf("allocations = %g, want %g", got, want[i])
			}
		})
	}
}
