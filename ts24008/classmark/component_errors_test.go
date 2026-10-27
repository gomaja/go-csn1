package classmark

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
		{"Classmark3ValuePartMultibandSupportedChoiceA5Bits", "invalid truncation point", runtime.InvalidValue, func(w *runtime.Writer) error {
			w.WithWire(runtime.WireInfo{TruncatedAt: map[string][]int{"Classmark3ValuePartMultibandSupportedChoiceA5Bits": {99}}})
			return encodeClassmark3ValuePartMultibandSupportedChoiceA5Bits(w, Classmark3ValuePartMultibandSupportedChoiceA5Bits{})
		}},
		{"HSCSDMultiSlotCapabilityHSCSDMultiSlotClass", "value reserved by source table", runtime.InvalidValue, func(w *runtime.Writer) error {
			w.WithWire(runtime.Canonical(HSCSDMultiSlotCapability{}).Wire)
			return encodeHSCSDMultiSlotCapabilityHSCSDMultiSlotClass(w, 0)
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
