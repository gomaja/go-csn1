package registry_test

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44060/ies"
)

func TestPublicComponentErrorPaths(t *testing.T) {
	countMismatch := measurement.MeasurementResultsContents{Content: measurement.MeasurementResultsFields{Count: measurement.MeasurementResultsCountChoice{
		Alternative: measurement.MeasurementResultsCountChoiceAlternativeReports,
		Reports:     &measurement.MeasurementResultsReportsArm{NONCELLM: 1},
	}}}
	for _, tc := range []struct {
		name, path, detail string
		kind               runtime.ErrorKind
		call               func() error
	}{
		{"choice", "MBMSChannelParametersIE/MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice", "choice requires exactly one alternative", runtime.InvalidBranch, func() error { _, err := ies.EncodeMBMSChannelParametersIE(ies.MBMSChannelParametersIE{}); return err }},
		{"reports count", "MeasurementResultsContents/MeasurementResultsFields/MeasurementResultsReportsArm/MeasurementResultsNCELLReports", "repeat count mismatch", runtime.InvalidValue, func() error { _, err := measurement.EncodeMeasurementResultsContents(countMismatch); return err }},
		{"container count", "NonGSMMessageStruct/NonGSMMessageStructCONTAINERList", "repeat count mismatch", runtime.InvalidValue, func() error {
			_, err := restoctets.EncodeNonGSMMessageStruct(restoctets.NonGSMMessageStruct{NonGSMProtocolDiscriminator: 1, NROFCONTAINEROCTETS: 1})
			return err
		}},
		{"list element", "SI18RestOctets/SI18RestOctetsNonGSMMessageList[0]", "zero-length SI stop record belongs in Wire", runtime.InvalidValue, func() error {
			_, err := restoctets.EncodeSI18RestOctets(restoctets.SI18RestOctets{NonGSMMessageList: []restoctets.NonGSMMessageStruct{{}}})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			var op *runtime.OperationError
			if !errors.As(err, &op) || op.Operation != "encode" || op.Kind != tc.kind || op.Location.Path != tc.path {
				t.Fatalf("error: %v; want %s", err, tc.path)
			}
			if got := err.Error(); len(got) < len(tc.detail) || got[len(got)-len(tc.detail):] != tc.detail {
				t.Fatalf("detail changed: %v", err)
			}
		})
	}
}

func TestContinuationComponentPathParity(t *testing.T) {
	_, decodeErr := restoctets.DecodeNonGSMMessageStructFrom(runtime.NewReader([]byte{0x3f}))
	encodeErr := restoctets.EncodeNonGSMMessageStructTo(runtime.NewWriter(), restoctets.NonGSMMessageStruct{NonGSMProtocolDiscriminator: 1, NROFCONTAINEROCTETS: 31})
	for _, err := range []error{decodeErr, encodeErr} {
		var op *runtime.OperationError
		if !errors.As(err, &op) || op.Location.Path != "NonGSMMessageStruct/NonGSMMessageStructCONTAINERList" || op.Location.BitOffset != -1 || !errors.Is(err, runtime.ErrContextRequired) {
			t.Fatalf("continuation: %v", err)
		}
	}
}

func TestListComponentPathParity(t *testing.T) {
	wire := make([]byte, 20)
	wire[1] = 1
	_, decodeErr := restoctets.DecodeSI18RestOctets(wire)
	_, encodeErr := restoctets.EncodeSI18RestOctets(restoctets.SI18RestOctets{NonGSMMessageList: []restoctets.NonGSMMessageStruct{{NROFCONTAINEROCTETS: 1, CONTAINERList: []uint8{0}}}})
	for _, err := range []error{decodeErr, encodeErr} {
		var op *runtime.OperationError
		if !errors.As(err, &op) || op.Location.Path != "SI18RestOctets/SI18RestOctetsNonGSMMessageList[0]" || op.Kind != runtime.InvalidValue {
			t.Fatalf("list: %v", err)
		}
	}
}
