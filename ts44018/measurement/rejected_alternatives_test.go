package measurement

import (
	"encoding/hex"
	"testing"
)

// TS 44.018 V19.0.0 table 10.5.2.20.0: synthetic unavailable,
// three-neighbour and six-neighbour Measurement Results contents.
var measurementReports = []struct{ name, hex string }{
	{"NoCells", "a82627c0000000000000000000000000"},
	{"ThreeCells", "a826269e1daa1cc00000000000000000"},
	{"SixCells", "a8262699dda9d1c87b0a4d8e3b1f2c40"},
}

func BenchmarkMeasurementResultsReports(b *testing.B) {
	for _, report := range measurementReports {
		b.Run(report.name, func(b *testing.B) {
			data, err := hex.DecodeString(report.hex)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			for range b.N {
				if _, err := DecodeMeasurementResultsContents(data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
