package restoctets

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
		{"NonGSMMessageStructCONTAINERList", "reserved container length code", runtime.InvalidValue, func(w *runtime.Writer) error {
			w.Set("NR_OF_CONTAINER_OCTETS", 19)
			return encodeNonGSMMessageStructCONTAINERList(w, nil)
		}},
		{"NonGSMMessageStructCONTAINERList", "CSN.1 context required: continuation count requires enclosing SI value", runtime.InvalidValue, func(w *runtime.Writer) error {
			w.Set("NR_OF_CONTAINER_OCTETS", 31)
			return encodeNonGSMMessageStructCONTAINERList(w, nil)
		}},
		{"SI18RestOctetsNonGSMMessageList", "SI message list exceeds fixed IE", runtime.InvalidValue, func(w *runtime.Writer) error { return encodeSI18RestOctetsNonGSMMessageList(w, nil) }},
		{"IARestOctetsCompressedInterRATHOINFOINDChoiceRCCCompressedInterRATHOINFOIND", "L/H field outside 0..1", runtime.InvalidValue, func(w *runtime.Writer) error {
			return encodeIARestOctetsCompressedInterRATHOINFOINDChoiceRCCCompressedInterRATHOINFOIND(w, 2)
		}},
		{"COMPACTNeighbourCellParamsStructSTARTFREQUENCYGroupListEntryFREQDIFFLENGTH", "interpreted field below 1", runtime.InvalidValue, func(w *runtime.Writer) error {
			return encodeCOMPACTNeighbourCellParamsStructSTARTFREQUENCYGroupListEntryFREQDIFFLENGTH(w, 0)
		}},
		{"IARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentEGPRSPacketUplinkAssignmentChoiceAlt1", "reserved CSN.1 alternative", runtime.InvalidValue, func(w *runtime.Writer) error {
			return encodeIARestOctetsCompressedInterRATHOINFOINDChoiceEGPRSPacketUplinkAssignmentEGPRSPacketUplinkAssignmentChoiceAlt1(w, struct{}{})
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
