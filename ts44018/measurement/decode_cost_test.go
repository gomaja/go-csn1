package measurement

import (
	"encoding/hex"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 table 10.5.2.20.0: two neighbour reports
// followed by literal-zero padding in the 16-octet contents.
var measurementCostInput, _ = hex.DecodeString("a826269e1daa1cc00000000000000000")

func BenchmarkDecodeMeasurementResultsContents(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := DecodeMeasurementResultsContents(measurementCostInput); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeMeasurementResultsContents(b *testing.B) {
	d, err := DecodeMeasurementResultsContents(measurementCostInput)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := EncodeMeasurementResultsContents(d.Value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementFieldBinding(b *testing.B) {
	for _, name := range []string{"RXLEV_FULL_SERVING_CELL", "rxlev_full_serving_cell"} {
		b.Run(name, func(b *testing.B) {
			r := runtime.NewReader(nil)
			b.ReportAllocs()
			for b.Loop() {
				r.Set(name, 42)
			}
		})
	}
}

func BenchmarkMeasurementConstantFieldDecode(b *testing.B) {
	data := []byte{0xa8}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeNCELLReportStructRXLEVNCELL(runtime.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementConstantFieldEncode(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if err := encodeNCELLReportStructRXLEVNCELL(runtime.NewWriter(), 42); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementCountedReportsDecode(b *testing.B) {
	data := make([]byte, 5)
	b.ReportAllocs()
	for b.Loop() {
		r := runtime.NewReader(data)
		r.Set("NO_NCELL_M", 2)
		if _, err := decodeMeasurementResultsNCELLReports(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementCountedReportsEncode(b *testing.B) {
	value := make([]NCELLReportStruct, 2)
	b.ReportAllocs()
	for b.Loop() {
		w := runtime.NewWriter()
		w.Set("NO_NCELL_M", 2)
		if err := encodeMeasurementResultsNCELLReports(w, value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementZeroPaddingDecode(b *testing.B) {
	data := make([]byte, 16)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeMeasurementResultsSuffixNull(runtime.NewReader(data)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMeasurementZeroPaddingEncode(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		w := runtime.NewWriter()
		if _, err := w.PushLimit(128); err != nil {
			b.Fatal(err)
		}
		if err := encodeMeasurementResultsSuffixNull(w, struct{}{}); err != nil {
			b.Fatal(err)
		}
	}
}

// From/To entry points share expression bindings with their caller. A field
// remains observable even when the generated grammar never reads it itself.
func TestMeasurementFieldBindingsRemainObservable(t *testing.T) {
	r := runtime.NewReader(measurementCostInput)
	v, err := DecodeMeasurementResultsContentsFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	if err := EncodeMeasurementResultsContentsTo(w, v); err != nil {
		t.Fatal(err)
	}
	for _, eval := range []func(string) (int, error){r.Eval, w.Eval} {
		for _, name := range []string{"BA_USED", "DTX_USED", "RXLEV_FULL_SERVING_CELL", "RXLEV-NCELL", "NO_NCELL_M"} {
			if _, err := eval("val(" + name + ")"); err != nil {
				t.Fatalf("lost %s binding: %v", name, err)
			}
		}
	}
}
