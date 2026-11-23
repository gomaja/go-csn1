package vgcsreconfigure2

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §9.1.21i includes the short header. TS 44.006
// V19.0.0 §8.8.5 p2466 limits acknowledged L3 to 251 octets;
// §8.8.3 p2456/p2458 and §8.8.5 p2467 limit unacknowledged Bter to 23.
func TestReconfigureCarrierModeCeilings(t *testing.T) {
	d, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil {
		t.Fatal(err)
	}
	u := DescriptorVBSVGCSRECONFIGURE2ForUnacknowledgedMode()
	if d.MaxOctets != 251 || u.MaxOctets != 23 || u.Extent != d.Extent {
		t.Fatalf("mode metadata: %+v %+v", d, u)
	}
	roots := 0
	for _, entry := range Descriptors() {
		if entry.Name == d.Name {
			roots++
			if entry.MaxOctets != 251 {
				t.Fatal("mode replaced catalogue default")
			}
		}
	}
	if roots != 1 {
		t.Fatalf("catalogue duplicates: %d", roots)
	}
	defaultEncoders := []func(VBSVGCSRECONFIGURE2, int) ([]byte, error){EncodeVBSVGCSRECONFIGURE2CanonicalAtLength, func(v VBSVGCSRECONFIGURE2, n int) ([]byte, error) { return d.CanonicalAtLength(v, n) }}
	unacknowledgedEncoders := []func(VBSVGCSRECONFIGURE2, int) ([]byte, error){EncodeVBSVGCSRECONFIGURE2CanonicalAtLengthUnacknowledged, func(v VBSVGCSRECONFIGURE2, n int) ([]byte, error) { return u.CanonicalAtLength(v, n) }}
	for _, tc := range []struct {
		encoders []func(VBSVGCSRECONFIGURE2, int) ([]byte, error)
		maximum  int
	}{{defaultEncoders, 251}, {unacknowledgedEncoders, 23}} {
		for _, encode := range tc.encoders {
			for _, n := range []int{19, 20} {
				wire := make([]byte, n)
				wire[0] = 0x1c
				decoded, err := DecodeVBSVGCSRECONFIGURE2(wire)
				if err != nil {
					t.Fatal(err)
				}
				out, err := encode(decoded.Value, n)
				if err != nil || len(out) != n {
					t.Fatalf("existing success %d: %v", n, err)
				}
			}
			wire := make([]byte, 20)
			wire[0] = 0x1c
			decoded, err := DecodeVBSVGCSRECONFIGURE2(wire)
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range []int{23, 251} {
				if n > tc.maximum {
					continue
				}
				out, err := encode(decoded.Value, n)
				var unfilled *runtime.ExtentError
				if out != nil || !errors.As(err, &unfilled) || unfilled.Actual != 20 || unfilled.Minimum != n || unfilled.Maximum != n {
					t.Fatalf("existing unfilled target %d: %+v %v", n, unfilled, err)
				}
			}
			nullWire := make([]byte, 19)
			nullWire[0] = 0x1c
			nullValue, err := DecodeVBSVGCSRECONFIGURE2(nullWire)
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range []int{23, 251} {
				if n > tc.maximum {
					continue
				}
				out, err := encode(nullValue.Value, n)
				var bound *runtime.BoundError
				if out != nil || !errors.As(err, &bound) || bound.Kind != runtime.CanonicalTarget {
					t.Fatalf("existing null-target refusal %d: %v", n, err)
				}
			}
			// Invalid typed content proves the ceiling precedes the encoder.
			out, err := encode(VBSVGCSRECONFIGURE2{}, tc.maximum+1)
			var e *runtime.ExtentError
			if out != nil || !errors.As(err, &e) || e.Required || e.Maximum != tc.maximum || e.Actual != tc.maximum+1 || e.Minimum != 0 {
				t.Fatalf("ceiling preflight: %+v %v", e, err)
			}
			for _, n := range []int{-1, 0, 1 << 20} {
				if _, err := encode(VBSVGCSRECONFIGURE2{}, n); err == nil {
					t.Fatalf("accepted invalid extent %d", n)
				}
			}
		}
	}
	// Both mode descriptors preserve decoder and received-image replay behavior.
	for _, n := range []int{252, 300} {
		wire := make([]byte, n)
		wire[0] = 0x1c
		value, err := DecodeVBSVGCSRECONFIGURE2(wire)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := EncodeVBSVGCSRECONFIGURE2(value.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("typed replay %d: %v", n, err)
		}
		for _, entry := range []runtime.Descriptor{d, u} {
			decoded, err := entry.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := entry.Encode(decoded)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("descriptor replay %d: %v", n, err)
			}
			_, report, err := entry.DecodeWithReport(wire)
			if err != nil {
				t.Fatalf("report %d: %v %v", n, report, err)
			}
			out, err := entry.CanonicalAtLength(decoded, entry.MaxOctets+1)
			var e *runtime.ExtentError
			if out != nil || !errors.As(err, &e) || e.Maximum != entry.MaxOctets {
				t.Fatalf("decoded descriptor ceiling: %v", err)
			}
		}
	}
	after, err := LookupClause("9.1.21i", "VBS/VGCS RECONFIGURE2")
	if err != nil || after.MaxOctets != 251 {
		t.Fatal("getter mutated default")
	}
	for _, entry := range []runtime.Descriptor{d, u} {
		if _, err := entry.CanonicalAtLength(struct{}{}, 7); err == nil {
			t.Fatal("wrong type accepted")
		}
	}
}
