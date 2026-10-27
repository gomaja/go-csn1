package measurement_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/measurement"
)

func ambiguousReleaseValue() measurement.EnhancedMeasurementReport {
	entries := make([]*uint8, 96)
	for i := 0; i < 4; i++ {
		quantity := uint8(0)
		entries[i] = &quantity
	}
	return measurement.EnhancedMeasurementReport{
		MessageType:           4,
		ServingCellData:       &measurement.ServingCellDataStruct{},
		REPORTINGQUANTITYList: &entries,
		BITMAPLENGTHChoice: measurement.EnhancedMeasurementReportBITMAPLENGTHChoice{
			Alternative: measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeBITMAPLENGTH,
			BITMAPLENGTH: &measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTH{
				REPORTINGQUANTITYList: make([]*uint8, 1),
				UTRANCSGMeasurementReportChoice: measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{
					Alternative:  measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeAltUnlabeled,
					AltUnlabeled: &struct{}{},
				},
			},
		},
	}
}

func TestEnhancedMeasurementReportNonzeroReleaseTail(t *testing.T) {
	v := ambiguousReleaseValue()
	// At this enclosing offset the Rel-8 H selector is 1 instead of 0.
	// It is the sole 1 in the ten-bit suffix; all explicit fields survive.
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 3); err != nil {
		t.Fatal(err)
	}
	if err := measurement.EncodeEnhancedMeasurementReportTo(w, v); err != nil {
		t.Fatal(err)
	}
	wire, err := w.Finish(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	ones := 0
	for bit := 161; bit < 171; bit++ {
		if wire[bit/8]&(1<<uint(7-bit%8)) != 0 {
			ones++
		}
	}
	if ones != 1 {
		t.Fatalf("release tail has %d set bits: %x", ones, wire)
	}
	r := runtime.NewReader(wire)
	if _, err := r.ReadUint(3); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(168); err != nil {
		t.Fatal(err)
	}
	got, err := measurement.DecodeEnhancedMeasurementReportFrom(r)
	if err != nil || !runtime.SemanticallyEqual(runtime.Canonical(v), runtime.Canonical(got)) {
		t.Fatalf("one-bit release fields changed: %+v: %v", got.BITMAPLENGTHChoice, err)
	}
}

func TestEnhancedMeasurementReportReleaseTailWithPadding(t *testing.T) {
	v := ambiguousReleaseValue()
	v.ServingCellData = nil
	for _, encode := range []func(measurement.EnhancedMeasurementReport) ([]byte, error){measurement.EncodeEnhancedMeasurementReport, measurement.EncodeEnhancedMeasurementReportCanonical} {
		wire, err := encode(v)
		if err != nil {
			t.Fatal(err)
		}
		got, err := measurement.DecodeEnhancedMeasurementReport(wire)
		if err != nil || got.Value.BITMAPLENGTHChoice.BITMAPLENGTH == nil || len(*got.Value.REPORTINGQUANTITYList) != 96 {
			t.Fatalf("padded release fields lost: %+v: %v", got.Value.BITMAPLENGTHChoice, err)
		}
		wantRelease := runtime.Canonical(v.BITMAPLENGTHChoice.BITMAPLENGTH)
		gotRelease := runtime.Canonical(got.Value.BITMAPLENGTHChoice.BITMAPLENGTH)
		// Null and L both omit Rel-9 fields; spare padding emits L.
		gotRelease.UTRANCSGMeasurementReportChoice = wantRelease.UTRANCSGMeasurementReportChoice
		if got.Value.BITMAPLENGTHChoice.BITMAPLENGTH.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport != nil || !runtime.SemanticallyEqual(wantRelease, gotRelease) {
			t.Fatal("padded release fields changed")
		}
	}
}

func TestEnhancedMeasurementReportAmbiguousReceivedReplay(t *testing.T) {
	// The received all-zero suffix represents ten redundant positions.
	wire := []byte{0x10, 0x08, 0, 0, 0x06, 0x04, 0x08, 0x10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	got, err := measurement.DecodeEnhancedMeasurementReport(wire)
	if err != nil || len(*got.Value.REPORTINGQUANTITYList) != 106 || got.Value.BITMAPLENGTHChoice.BITMAPLENGTH != nil {
		t.Fatalf("received boundary: %+v: %v", got, err)
	}
	out, err := measurement.EncodeEnhancedMeasurementReport(got.Value)
	if err != nil || !bytes.Equal(out, wire) {
		t.Fatalf("received replay %x: %v", out, err)
	}
	v := ambiguousReleaseValue()
	v.BITMAPLENGTHChoice = got.Value.BITMAPLENGTHChoice
	for _, encode := range []func(measurement.EnhancedMeasurementReport) ([]byte, error){measurement.EncodeEnhancedMeasurementReport, measurement.EncodeEnhancedMeasurementReportCanonical} {
		out, err = encode(v)
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("fresh zero positions %x: %v", out, err)
		}
	}
}

// TS 44.018 V19.0.0 §3.4.1.2.1.3 and table 9.1.55.1, with the
// boundary rule approved 2026-10-27: an entirely zero tail after the
// 96th position denotes redundant legacy positions, never release fields.
func TestEnhancedMeasurementReportAmbiguousReleaseRefusal(t *testing.T) {
	v := ambiguousReleaseValue()
	for name, encode := range map[string]func(measurement.EnhancedMeasurementReport) ([]byte, error){
		"plain":     measurement.EncodeEnhancedMeasurementReport,
		"canonical": measurement.EncodeEnhancedMeasurementReportCanonical,
		"canonical-length": func(v measurement.EnhancedMeasurementReport) ([]byte, error) {
			return measurement.EncodeEnhancedMeasurementReportCanonicalAtLength(v, 21)
		},
	} {
		t.Run(name, func(t *testing.T) {
			wire, err := encode(v)
			var failure *runtime.OperationError
			if !errors.As(err, &failure) || failure.Kind != runtime.InvalidValue || failure.Location.Path != "EnhancedMeasurementReport" {
				t.Fatalf("ambiguous release encoded as %x: %v", wire, err)
			}
			cause := error(failure)
			for errors.Unwrap(cause) != nil {
				cause = errors.Unwrap(cause)
			}
			if cause.Error() != "release additions encode an all-zero tail indistinguishable from redundant legacy bitmap positions" {
				t.Fatalf("ambiguous release cause: %v", failure.Cause)
			}
		})
	}
}
