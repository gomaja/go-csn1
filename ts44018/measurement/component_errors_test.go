package measurement

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
		{"MeasurementResultsUnavailableArmAlt0List", "literal-zero padding requires enclosing length bound", runtime.InvalidValue, func(w *runtime.Writer) error { return encodeMeasurementResultsUnavailableArmAlt0List(w, struct{}{}) }},
		{"EnhancedMeasurementReportREPORTINGQUANTITYListValue", "pre-Rel-8 reporting bitmap exceeds 96 positions", runtime.Limit, func(w *runtime.Writer) error {
			return encodeEnhancedMeasurementReportREPORTINGQUANTITYListValue(w, make([]*uint8, 97))
		}},
		{"MeasurementResultsNCELLReports", "repeat count mismatch", runtime.InvalidValue, func(w *runtime.Writer) error {
			w.Set("NO_NCELL_M", 1)
			return encodeMeasurementResultsNCELLReports(w, nil)
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
