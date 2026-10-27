package measurement_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
	"github.com/gomaja/go-csn1/ts44018/measurement"
)

func resultsWithPadding(count int) measurement.MeasurementResultsContents {
	zeros := make([]struct{}, count)
	return measurement.MeasurementResultsContents{Content: measurement.MeasurementResultsFields{Count: measurement.MeasurementResultsCountChoice{
		Alternative: measurement.MeasurementResultsCountChoiceAlternativeReports,
		Reports:     &measurement.MeasurementResultsReportsArm{NONCELLM: 1, NCELLReports: []measurement.NCELLReportStruct{{}}, Suffix: measurement.MeasurementResultsReportsSuffix{Alternative: measurement.MeasurementResultsReportsSuffixAlternativeZeroPadding, ZeroPadding: &zeros}},
	}}}
}

func TestLocatedErrorPaths(t *testing.T) {
	emr := measurement.EnhancedMeasurementReport{MessageType: 4, RepeatedInvalidBSICInformationList: make([]measurement.EnhancedMeasurementReportRepeatedInvalidBSICInformationListEntry, 2)}
	emr.RepeatedInvalidBSICInformationList[1].RepeatedInvalidBSICInformation.BCCHFREQNCELL = 32
	invalid := resultsWithPadding(85)
	invalid.Content.Count.Reports.NCELLReports[0].RXLEVNCELL = 64
	listPath := "EnhancedMeasurementReport/EnhancedMeasurementReportRepeatedInvalidBSICInformationList[1]/EnhancedMeasurementReportRepeatedInvalidBSICInformationListEntry/RepeatedInvalidBSICInformationStruct/RepeatedInvalidBSICInformationStructBCCHFREQNCELL"
	cases := []struct {
		name          string
		call          func() error
		operation     string
		kind          runtime.ErrorKind
		offset, width int
		path, detail  string
		sentinel      error
	}{
		{"list encode", func() error { _, err := measurement.EncodeEnhancedMeasurementReport(emr); return err }, "encode", runtime.InvalidValue, 32, 5, listPath, "value outside 5-bit field", nil},
		{"list decode", func() error {
			_, err := measurement.DecodeEnhancedMeasurementReport([]byte{0x10, 0x04, 0, 1})
			return err
		}, "decode", runtime.Truncated, 32, 5, listPath, "required bit absent", nil},
		{"serving encode", func() error {
			_, err := measurement.EncodeEnhancedMeasurementReport(measurement.EnhancedMeasurementReport{MessageType: 4, ServingCellData: &measurement.ServingCellDataStruct{RXLEVVAL: 64}})
			return err
		}, "encode", runtime.InvalidValue, 14, 6, "EnhancedMeasurementReport/ServingCellDataStruct/ServingCellDataStructRXLEVVAL", "value outside 6-bit field", nil},
		{"reports encode", func() error { _, err := measurement.EncodeMeasurementResultsContents(invalid); return err }, "encode", runtime.InvalidValue, 26, 6, "MeasurementResultsContents/MeasurementResultsFields/MeasurementResultsReportsArm/MeasurementResultsNCELLReports[0]/NCELLReportStruct/NCELLReportStructRXLEVNCELL", "value outside 6-bit field", nil},
		{"short bound", func() error {
			_, err := measurement.EncodeMeasurementResultsContents(resultsWithPadding(84))
			return err
		}, "encode", runtime.InvalidValue, 127, 1, "MeasurementResultsContents", "length bound (limit 128): content ends before the length-delimited value", runtime.ErrBoundConflict},
		{"long bound", func() error {
			_, err := measurement.EncodeMeasurementResultsContents(resultsWithPadding(86))
			return err
		}, "encode", runtime.InvalidValue, 128, 0, "MeasurementResultsContents/MeasurementResultsFields/MeasurementResultsReportsArm/MeasurementResultsSuffixZeroPadding[85]", "length bound (limit 128): encoded bit exceeds the length-delimited value", runtime.ErrBoundConflict},
		{"import decode", func() error { _, err := uecapability.DecodeGERANCS([]byte{0x33, 3, 0, 0, 0, 255, 255}); return err }, "decode", runtime.InvalidBranch, 41, 0, "GERANCS/Classmark3/Classmark3ValuePart", "choice has 0 matching alternatives", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			var op *runtime.OperationError
			if !errors.As(err, &op) {
				t.Fatal(err)
			}
			if op.Operation != tc.operation || op.Kind != tc.kind || op.Location.Path != tc.path || op.Location.BitOffset != tc.offset || op.Location.Width != tc.width {
				t.Fatalf("operation: %+v", op)
			}
			want := fmt.Sprintf("CSN.1 %s %s at bit %d in %s: %s", tc.operation, tc.kind, tc.offset, tc.path, tc.detail)
			t.Logf("MESSAGE\t%s\t%s", tc.name, err)
			if err.Error() != want {
				t.Fatalf("got %q want %q", err, want)
			}
			if tc.sentinel != nil && !errors.Is(err, tc.sentinel) {
				t.Fatal("sentinel lost")
			}
			if op.Cause == nil || !errors.Is(err, op.Cause) {
				t.Fatal("cause identity lost")
			}
			if tc.operation == "decode" {
				var de *runtime.DecodeError
				if !errors.As(err, &de) || de.Offset != tc.offset || de.Kind != tc.kind {
					t.Fatal("decode cause changed")
				}
			}
		})
	}
}
