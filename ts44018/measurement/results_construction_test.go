package measurement

import (
	"github.com/gomaja/go-csn1/ts44060/ies"
	"reflect"
)

func resultsSameServing(a, b MeasurementResultsFields) bool {
	a.Count = MeasurementResultsCountChoice{}
	b.Count = MeasurementResultsCountChoice{}
	return reflect.DeepEqual(a, b)
}
func resultsBuildUnavailable(serving MeasurementResultsFields) (MeasurementResultsContents, error) {
	serving.Count = MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeUnavailable, Unavailable: &MeasurementResultsUnavailableArm{NONCELLM: 7, ZeroPadding: make([]struct{}, 102)}}
	v := MeasurementResultsContents{Content: serving}
	_, err := EncodeMeasurementResultsContents(v)
	return v, err
}
func resultsBuildReports(serving MeasurementResultsFields, reports []NCELLReportStruct, csg *ies.UTRANCSGMeasurementReportIE) (MeasurementResultsContents, error) {
	end := 26 + 17*len(reports)
	arm := &MeasurementResultsReportsArm{NONCELLM: uint8(len(reports)), NCELLReports: reports}
	if csg != nil {
		end += 64
		if csg.PLMNID != nil {
			end += 24
		}
		nested := &MeasurementResultsCSGArm{Report: *csg}
		if end >= 128 {
			nested.Suffix = MeasurementResultsCSGSuffix{Alternative: MeasurementResultsCSGSuffixAlternativeNull, Null: &struct{}{}}
		} else {
			zeroes := make([]struct{}, 128-end)
			nested.Suffix = MeasurementResultsCSGSuffix{Alternative: MeasurementResultsCSGSuffixAlternativeZeroPadding, ZeroPadding: &zeroes}
		}
		arm.Suffix = MeasurementResultsReportsSuffix{Alternative: MeasurementResultsReportsSuffixAlternativeCSG, CSG: nested}
	} else if end >= 128 {
		arm.Suffix = MeasurementResultsReportsSuffix{Alternative: MeasurementResultsReportsSuffixAlternativeNull, Null: &struct{}{}}
	} else {
		zeroes := make([]struct{}, 128-end)
		arm.Suffix = MeasurementResultsReportsSuffix{Alternative: MeasurementResultsReportsSuffixAlternativeZeroPadding, ZeroPadding: &zeroes}
	}
	serving.Count = MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports, Reports: arm}
	v := MeasurementResultsContents{Content: serving}
	_, err := EncodeMeasurementResultsContents(v)
	return v, err
}
