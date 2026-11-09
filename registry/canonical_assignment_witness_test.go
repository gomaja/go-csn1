package registry

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/egprsmode2"
)

// TS 44.060 V19.0.0 §12.48a.1 p24650 delegates Single Uplink (§12.48a.3).
// Its Pulse Format (§12.8.3 pp22817–22819) contains the bounded bitmap
// 1, 01. These edits prove the two imported guards through the public wrapper.
func canonicalAssignmentImportedErrors(t *testing.T, d runtime.Descriptor, record func(string, error, int)) {
	t.Helper()
	wire, err := hex.DecodeString("139081ab02a0")
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewReaderBits(wire, 43, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(43); err != nil {
		t.Fatal(err)
	}
	original, err := egprsmode2.DecodeEGPRSMode2IEFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, ignored := range []bool{false, true} {
		v := runtime.Canonical(original)
		group := v.DownlinkAssignmentChoice.DownlinkAssignment.UplinkAssignment.PulseFormat.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.PulseFormatBitmapLength
		if group == nil || group.PulseFormatBitmapLength != 2 || group.PulseFormatBitmap.Known == nil {
			t.Fatal("witness did not select the typed three-bit bitmap")
		}
		group.PulseFormatBitmapLength = 0
		if ignored {
			group.PulseFormatBitmap.Alternative = 1
			group.PulseFormatBitmap.Known = nil
			group.PulseFormatBitmap.Ignored = &runtime.BitString{Bytes: []byte{0x80}, BitLength: 1}
		}
		_, err := d.CanonicalAtLength(v, 64)
		if ignored {
			var fallback *runtime.FallbackError
			if !errors.As(err, &fallback) {
				t.Errorf("imported known fallback: %v", err)
				continue
			}
		} else {
			var bound *runtime.BoundError
			if !errors.As(err, &bound) || bound.Kind != runtime.LengthBound {
				t.Errorf("imported bitmap length bound: %v", err)
				continue
			}
		}
		record(entryAtLength, err, 64)
	}
}
