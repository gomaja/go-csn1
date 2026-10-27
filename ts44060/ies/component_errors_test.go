package ies

import (
	"encoding/json"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"strings"
	"testing"
)

func TestGeneratedComponentErrorFamilies(t *testing.T) {
	for _, tc := range []struct {
		component, detail string
		kind              runtime.ErrorKind
		call              func(*runtime.Writer) error
	}{
		{"MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice", "choice requires exactly one alternative", runtime.InvalidBranch, func(w *runtime.Writer) error {
			return encodeMBMSChannelParametersIEMBMSPTMChannelDescriptionChoice(w, MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice{})
		}},
		{"MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice", "choice alternative mismatch", runtime.InvalidBranch, func(w *runtime.Writer) error {
			return encodeMBMSChannelParametersIEMBMSPTMChannelDescriptionChoice(w, MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice{MPRACHDescription: &MBMSChannelParametersIEMBMSPTMChannelDescriptionChoiceMPRACHDescription{}})
		}},
		{"MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice", "invalid choice alternative", runtime.InvalidBranch, func(w *runtime.Writer) error {
			return encodeMBMSChannelParametersIEMBMSPTMChannelDescriptionChoice(w, MBMSChannelParametersIEMBMSPTMChannelDescriptionChoice{Alternative: 255, MPRACHDescription: &MBMSChannelParametersIEMBMSPTMChannelDescriptionChoiceMPRACHDescription{}})
		}},
		{"GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback", "extension fallback requires exactly one arm", runtime.InvalidValue, func(w *runtime.Writer) error {
			return encodeGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback(w, GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{})
		}},
		{"GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback", "fallback arm mismatch", runtime.InvalidBranch, func(w *runtime.Writer) error {
			return encodeGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback(w, GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{Ignored: &runtime.BitString{}})
		}},
		{"GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback", "invalid fallback arm", runtime.InvalidBranch, func(w *runtime.Writer) error {
			return encodeGPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback(w, GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{Alternative: 255, Ignored: &runtime.BitString{}})
		}},
	} {
		t.Run(tc.detail, func(t *testing.T) {
			for _, parent := range []string{"First", "Second/Nested"} {
				w := runtime.NewWriter()
				if err := w.Enter(parent); err != nil {
					t.Fatal(err)
				}
				err := tc.call(w)
				var op *runtime.OperationError
				if !errors.As(err, &op) {
					t.Fatal(err)
				}
				facts := struct {
					Parent, Component, Operation string
					Kind                         runtime.ErrorKind
					Location                     runtime.Location
					Message, Cause               string
				}{parent, tc.component, op.Operation, op.Kind, op.Location, err.Error(), errors.Unwrap(err).Error()}
				encoded, marshalErr := json.Marshal(facts)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Logf("FACTS\t%s", encoded)
				want := parent + "/" + tc.component
				if !errors.As(err, &op) || op.Operation != "encode" || op.Kind != tc.kind || op.Location.Path != want || !strings.HasSuffix(err.Error(), ": "+tc.detail) {
					t.Errorf("error: %v; want %s: %s", err, want, tc.detail)
				}
				if op.Cause == nil || !errors.Is(err, op.Cause) {
					t.Fatal("cause lost")
				}
			}
		})
	}
}
