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
	serving.Count = MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeUnavailable, Unavailable: &MeasurementResultsUnavailableArm{NONCELLM: 7}}
	v := MeasurementResultsContents{Content: serving}
	_, err := EncodeMeasurementResultsContents(v)
	return v, err
}
func resultsBuildReports(serving MeasurementResultsFields, reports []NCELLReportStruct, csg *ies.UTRANCSGMeasurementReportIE) (MeasurementResultsContents, error) {
	if len(reports) == 0 {
		reports = nil
	}
	arm := &MeasurementResultsReportsArm{NONCELLM: uint8(len(reports)), NCELLReports: reports}
	if csg != nil {
		arm.Suffix = MeasurementResultsReportsSuffix{Alternative: MeasurementResultsReportsSuffixAlternativeCSG, CSG: &MeasurementResultsCSGArm{Report: *csg}}
	} else {
		arm.Suffix = MeasurementResultsReportsSuffix{Alternative: MeasurementResultsReportsSuffixAlternativeNull, Null: &struct{}{}}
	}
	serving.Count = MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports, Reports: arm}
	v := MeasurementResultsContents{Content: serving}
	_, err := EncodeMeasurementResultsContents(v)
	return v, err
}
