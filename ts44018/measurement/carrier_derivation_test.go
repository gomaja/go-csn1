package measurement

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 figure 9.1.55.1 and table 9.1.55.1 admit four
// full derivations before applying the 96-cell and redundant-zero boundary
// rule from §3.4.1.2.1.3. Enumerate every possible entry boundary, excluding
// derivations with ignored remainder or spare padding. The public decoder
// selects only the wholly redundant zero tail, ending in Rel-8 null.
func TestEnhancedMeasurementReportCompleteCarrierDerivations(t *testing.T) {
	wire := absentLegacyCarrier()
	var derivations []int
	for count := 96; count <= 153; count++ {
		r := runtime.NewReader(wire)
		if err := r.Expect("000100000000001"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < count; i++ {
			entry, err := decodeEnhancedMeasurementReportREPORTINGQUANTITYListValueEntrySelector(r)
			if err != nil || entry != nil {
				t.Fatalf("count %d entry %d: %v", count, i, err)
			}
		}
		suffix, err := decodeEnhancedMeasurementReportBITMAPLENGTHChoice(r)
		if err != nil || r.Position() != 168 {
			continue
		}
		ignored := false
		for _, span := range r.Wire().Padding {
			if span.BitLength != 0 {
				ignored = true
			}
		}
		if ignored {
			continue
		}
		w := runtime.NewWriter()
		if err := w.WriteLiteral("000100000000001"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < count; i++ {
			if err := encodeEnhancedMeasurementReportREPORTINGQUANTITYListValueEntrySelector(w, nil); err != nil {
				t.Fatal(err)
			}
		}
		if err := encodeEnhancedMeasurementReportBITMAPLENGTHChoice(w, suffix); err != nil {
			t.Fatal(err)
		}
		out, err := w.Finish(runtime.BitString{})
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("count %d: %x: %v", count, out, err)
		}
		derivations = append(derivations, count)
		rel8 := suffix.BITMAPLENGTH
		if count == 153 {
			if suffix.AltUnlabeled == nil {
				t.Fatal("missing Rel-8 null")
			}
		} else {
			if rel8 == nil || rel8.BITMAPLENGTH != 0 || len(rel8.REPORTINGQUANTITYList) != 1 || rel8.REPORTINGQUANTITYList[0] != nil || rel8.EUTRANMeasurementReport != nil {
				t.Fatalf("count %d: %+v", count, suffix)
			}
			rel9 := rel8.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
			switch count {
			case 143:
				if rel8.UTRANCSGMeasurementReportChoice.AltUnlabeled == nil {
					t.Fatal("missing Rel-9 null")
				}
			case 141:
				if rel9 == nil || rel9.UTRANCSGMeasurementReport != nil || rel9.SI23BAUSEDChoice.AltUnlabeled == nil {
					t.Fatal("missing Rel-11 null")
				}
			case 139:
				if rel9 == nil || rel9.UTRANCSGMeasurementReport != nil || rel9.SI23BAUSEDChoice.SI23BAUSED == nil || rel9.SI23BAUSEDChoice.SI23BAUSED.SI23BAUSED != 0 {
					t.Fatal("missing SI23_BA_USED zero")
				}
			}
		}
		t.Logf("legacy=%d boundary=%d suffix=%+v", count, 15+count, suffix)
	}
	if !reflect.DeepEqual(derivations, []int{139, 141, 143, 153}) {
		t.Fatalf("full derivations %v", derivations)
	}
	decoded, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil || len(*decoded.Value.REPORTINGQUANTITYList) != 153 || decoded.Value.BITMAPLENGTHChoice.AltUnlabeled == nil {
		t.Fatalf("selected %+v: %v", decoded, err)
	}
}
