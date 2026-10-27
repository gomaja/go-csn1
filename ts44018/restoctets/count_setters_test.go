package restoctets_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// TS 24.007 V20.0.0 Annex B §B.2.1; the source clauses are also
// recorded on every generated setter. All positions count, including absent
// optional elements in the EMR REPORTING_QUANTITY list (TS 44.018 §9.1.55).
func TestEveryCountSetterLength(t *testing.T) {
	for _, tc := range []struct {
		value                   any
		method, count           string
		minimum, maximum, extra int
	}{
		{measurement.MeasurementResultsReportsArm{}, "SetNCELLReports", "NONCELLM", 0, 6, 0},
		{measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTH{}, "SetREPORTINGQUANTITYList", "BITMAPLENGTH", 1, 128, 1},
		{measurement.EUTRANMeasurementReportStruct{}, "SetEUTRANFREQUENCYINDEXGroupList", "NEUTRAN", 1, 4, 1},
		{restoctets.CCNSupportDescriptionStruct{}, "SetCCNSUPPORTEDList", "NumberCells", 0, 127, 0},
		{restoctets.COMPACTNeighbourCellParamsStructSTARTFREQUENCYGroupListEntry{}, "SetFREQUENCYDIFFGroupList", "NROFREMAININGCELLS", 0, 15, 0},
		{restoctets.GPRSBSICDescriptionStruct{}, "SetFrequencyScrollingGroupList", "NumberRemainingBSIC", 0, 127, 0},
		{restoctets.GPRSREPORTPRIORITYDescriptionStruct{}, "SetREPPRIORITYList", "NumberCells", 0, 127, 0},
		{restoctets.IRATCellReselectionInformationStruct{}, "SetPLMNIndexGroupList", "NbAdditionalPLMNs", 1, 4, 1},
		{restoctets.NetworkSharingEABInformationStruct{}, "SetAdditionalEABAuthorizationMaskGroupList", "NbAdditionalPLMNs", 1, 4, 1},
		{restoctets.NetworkSharingInformationStruct{}, "SetMCCGroupList", "NbAdditionalPLMNs", 1, 4, 1},
		{restoctets.NonGSMMessageStruct{}, "SetCONTAINERList", "NROFCONTAINEROCTETS", 0, 18, 0},
		{restoctets.PEOIMMCellGroupSpecificParametersStruct{}, "SetBSICGroupList", "NbNCELL", 1, 16, 1},
	} {
		t.Run(reflect.TypeOf(tc.value).Name(), func(t *testing.T) {
			limit := tc.maximum + 1
			if tc.method == "SetCONTAINERList" {
				limit = 32
			}
			for length := 0; length <= limit; length++ {
				pointer := reflect.New(reflect.TypeOf(tc.value))
				method := pointer.MethodByName(tc.method)
				if !method.IsValid() {
					t.Fatal("setter missing")
				}
				items := reflect.MakeSlice(method.Type().In(0), length, length)
				before := pointer.Elem().Interface()
				got := method.Call([]reflect.Value{items})[0]
				valid := length >= tc.minimum && length <= tc.maximum
				if got.IsNil() != valid {
					t.Fatalf("length %d: %v", length, got.Interface())
				}
				if !valid {
					if !reflect.DeepEqual(pointer.Elem().Interface(), before) {
						t.Fatal("rejected length changed receiver")
					}
					continue
				}
				if int(pointer.Elem().FieldByName(tc.count).Uint()) != length-tc.extra {
					t.Fatalf("length %d count not derived", length)
				}
				list := pointer.Elem().FieldByName(strings.TrimPrefix(tc.method, "Set"))
				if list.Len() != length || length == 0 && !list.IsNil() || length > 0 && list.Pointer() != items.Pointer() {
					t.Fatal("setter did not store supplied slice")
				}
			}
			nilPointer := reflect.Zero(reflect.PointerTo(reflect.TypeOf(tc.value)))
			method := nilPointer.MethodByName(tc.method)
			if got := method.Call([]reflect.Value{reflect.Zero(method.Type().In(0))})[0]; got.IsNil() {
				t.Fatal("nil receiver accepted")
			}
		})
	}
}

func TestCountSetterStoresWithoutAllocation(t *testing.T) {
	for _, length := range []int{0, 1, 32, 127} {
		items := make([]uint8, length)
		var value restoctets.CCNSupportDescriptionStruct
		if count := testing.AllocsPerRun(10, func() {
			if err := value.SetCCNSUPPORTEDList(items); err != nil {
				panic(err)
			}
		}); count != 0 {
			t.Fatalf("length %d allocated %g", length, count)
		}
	}
}
