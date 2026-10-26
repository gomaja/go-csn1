package ies

import (
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestPSCGroupSecondEntryErrorHasIndex(t *testing.T) {
	// TS 44.060 V19.0.0 §12.58: one complete PSC entry followed by
	// a second entry with an incomplete nine-bit PSC.
	r := runtime.NewReader([]byte{0x80, 0x20})
	if _, err := r.PushLimit(15); err != nil {
		t.Fatal(err)
	}
	_, err := DecodePSCGroupIEFrom(r)
	var op *runtime.OperationError
	if !errors.As(err, &op) || !strings.Contains(op.Location.Path, "PSCGroupIEPSCList[1]/") {
		t.Fatalf("second PSC entry location: %v", err)
	}
}
