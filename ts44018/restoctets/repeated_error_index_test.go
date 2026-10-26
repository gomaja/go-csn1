package restoctets

import (
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestAccessTechnologiesSecondEntryErrorHasIndex(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16: one four-bit entry and continuation,
	// followed by only two bits of the second four-bit entry.
	r := runtime.NewReader([]byte{0x08})
	if _, err := r.PushLimit(7); err != nil {
		t.Fatal(err)
	}
	_, err := DecodeAccessTechnologiesRequestStructFrom(r)
	var op *runtime.OperationError
	if !errors.As(err, &op) || !strings.Contains(op.Location.Path, "AccessTechnologiesRequestStruct[1]/") {
		t.Fatalf("second access technology entry location: %v", err)
	}
}

func TestAccessTechnologiesSecondContinuationErrorHasIndex(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16: two complete entries; the
	// continuation bit belonging to the second entry is missing.
	r := runtime.NewReader([]byte{0x08, 0x00})
	if _, err := r.PushLimit(9); err != nil {
		t.Fatal(err)
	}
	_, err := DecodeAccessTechnologiesRequestStructFrom(r)
	var op *runtime.OperationError
	if !errors.As(err, &op) || !strings.Contains(op.Location.Path, "AccessTechnologiesRequestStruct[1]/") {
		t.Fatalf("second continuation location: %v", err)
	}
}
