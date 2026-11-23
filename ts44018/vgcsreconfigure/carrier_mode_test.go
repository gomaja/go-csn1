package vgcsreconfigure

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// TS 44.018 V19.0.0 §9.1.21h includes the short header. TS 44.006
// V19.0.0 §8.8.5 p2466 limits acknowledged L3 to 251 octets;
// §8.8.3 p2456/p2458 and §8.8.5 p2467 limit unacknowledged Bter to 23.
func TestReconfigureCarrierModeCeilings(t *testing.T) {
	d, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil {
		t.Fatal(err)
	}
	u := DescriptorVBSVGCSRECONFIGUREForUnacknowledgedMode()
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
	defaultEncoders := []func(VBSVGCSRECONFIGURE, int) ([]byte, error){EncodeVBSVGCSRECONFIGURECanonicalAtLength, func(v VBSVGCSRECONFIGURE, n int) ([]byte, error) { return d.CanonicalAtLength(v, n) }}
	unacknowledgedEncoders := []func(VBSVGCSRECONFIGURE, int) ([]byte, error){EncodeVBSVGCSRECONFIGURECanonicalAtLengthUnacknowledged, func(v VBSVGCSRECONFIGURE, n int) ([]byte, error) { return u.CanonicalAtLength(v, n) }}
	for _, tc := range []struct {
		encoders []func(VBSVGCSRECONFIGURE, int) ([]byte, error)
		maximum  int
	}{{defaultEncoders, 251}, {unacknowledgedEncoders, 23}} {
		for _, encode := range tc.encoders {
			for _, n := range []int{7, tc.maximum} {
				v := carrierSizedValue(n)
				out, err := encode(v, n)
				if err != nil || len(out) != n {
					t.Fatalf("valid upper bound %d: %v", n, err)
				}
				decoded, err := DecodeVBSVGCSRECONFIGURE(out)
				if err != nil || !runtime.SemanticallyEqual(decoded.Value, v) {
					t.Fatalf("typed value at %d: %v", n, err)
				}
			}
			out, err := encode(carrierSizedValue(tc.maximum+1), tc.maximum+1)
			var ceiling *runtime.ExtentError
			if out != nil || !errors.As(err, &ceiling) || ceiling.Maximum != tc.maximum || ceiling.Actual != tc.maximum+1 {
				t.Fatalf("sized above ceiling: %v", err)
			}
			// Invalid typed content proves the ceiling precedes the encoder.
			out, err = encode(VBSVGCSRECONFIGURE{}, tc.maximum+1)
			var e *runtime.ExtentError
			if out != nil || !errors.As(err, &e) || e.Required || e.Maximum != tc.maximum || e.Actual != tc.maximum+1 || e.Minimum != 0 {
				t.Fatalf("ceiling preflight: %+v %v", e, err)
			}
			for _, n := range []int{-1, 0, 1 << 20} {
				if _, err := encode(VBSVGCSRECONFIGURE{}, n); err == nil {
					t.Fatalf("accepted invalid extent %d", n)
				}
			}
		}
	}
	// Both mode descriptors preserve decoder and received-image replay behavior.
	for _, n := range []int{252, 300} {
		wire := make([]byte, n)
		wire[0] = 0x18
		value, err := DecodeVBSVGCSRECONFIGURE(wire)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := EncodeVBSVGCSRECONFIGURE(value.Value)
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
	after, err := LookupClause("9.1.21h", "VBS/VGCS RECONFIGURE")
	if err != nil || after.MaxOctets != 251 {
		t.Fatal("getter mutated default")
	}
	for _, entry := range []runtime.Descriptor{d, u} {
		if _, err := entry.CanonicalAtLength(struct{}{}, 7); err == nil {
			t.Fatal("wrong type accepted")
		}
	}
}

func carrierSizedValue(octets int) VBSVGCSRECONFIGURE {
	v := VBSVGCSRECONFIGURE{MessageType: 6, Alternatives: VBSVGCSRECONFIGUREAlternatives{Alternative: VBSVGCSRECONFIGUREAlternativesAlternativeNoString, NoString: &VBSVGCSRECONFIGUREAlternativesNoString{}}}
	if octets > 7 {
		// §9.1.21h p7651–7654: header/selector/LV and trailing fields
		// use 60 bits, rounded to eight octets, plus the LV value octets.
		length := octets - 8
		v.NewGroupChannelDescription.MobileAllocationChoice = &NewGroupChannelDescriptionMobileAllocationChoiceValue{Alternative: NewGroupChannelDescriptionMobileAllocationChoiceValueAlternativeMobileAllocation, MobileAllocation: &NewGroupChannelDescriptionMobileAllocationChoiceValueMobileAllocation{MobileAllocation: restoctets.BitString{Length: uint8(length), Value: runtime.BitString{Bytes: make([]byte, length), BitLength: length * 8}}}}
	}
	return v
}
