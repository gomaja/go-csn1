package registry

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/pshandover2"
)

// TS 44.060 V19.0.0 §12.42a: Extension Length + 1 is the bound.
// A present MTTI assignment needs its selector and value, hence two bits.
func canonicalHandoverExtensionBound(t *testing.T, d runtime.Descriptor, record func(string, error, int)) {
	t.Helper()
	w := runtime.NewWriter()
	if err := w.WriteLiteral("00000000001011000001000010000000000000000010100000"); err != nil {
		t.Fatal(err)
	}
	bits, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := pshandover2.DecodePSHandoverRadioResources2IEFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	one := uint8(1)
	v.ExtensionLengthGroup = &pshandover2.PSHandoverRadioResources2IEExtensionLengthGroup{}
	v.ExtensionLengthGroup.Content.Known = &pshandover2.ExtensionInformation{MTTIDOWNLINKASSIGNMENTC1: &one}
	_, err = d.CanonicalAtLength(v, 64)
	var bound *runtime.BoundError
	if !errors.As(err, &bound) || bound.Kind != runtime.LengthBound {
		t.Fatalf("nonzero MTTI in a one-bit description: %v", err)
	}
	record(entryAtLength, err, 64)
}

func TestCanonicalHandoverExtensionBound(t *testing.T) {
	d, err := LookupClause("TS 44.060", "12.42a", "PS Handover Radio Resources 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	canonicalHandoverExtensionBound(t, d, func(string, error, int) {})
}
