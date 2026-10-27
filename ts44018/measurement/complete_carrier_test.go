package measurement

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.006 V19.0.0 §§5.2, 6.4a and 8.8.3 require all 21 SACCH
// Bter octets, including the short L2 header. This sender extent is
// separate from the receiver's retained shorter grammar images.
func TestEnhancedMeasurementReportCompleteCarrier(t *testing.T) {
	decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	for _, fresh := range []EnhancedMeasurementReport{decoded.Value, runtime.Canonical(decoded.Value)} {
		canonical, err := EncodeEnhancedMeasurementReportCanonical(fresh)
		if err != nil || len(canonical) != 21 {
			t.Fatalf("canonical length %d: %v", len(canonical), err)
		}
		want := append([]byte{0x10, 0x01}, bytes.Repeat([]byte{0x2b}, 19)...)
		if !bytes.Equal(canonical, want) {
			t.Fatalf("carrier %x, want %x", canonical, want)
		}
		for _, extent := range []int{0, 1, 2, 20, 22} {
			_, err := EncodeEnhancedMeasurementReportCanonicalAtLength(fresh, extent)
			var failure *runtime.ExtentError
			if !errors.As(err, &failure) {
				t.Fatalf("accepted extent %d: %v", extent, err)
			}
		}
	}
	fresh, err := EncodeEnhancedMeasurementReport(runtime.Canonical(decoded.Value))
	if err != nil || len(fresh) != 21 {
		t.Fatalf("fresh length %d: %v", len(fresh), err)
	}
	for _, input := range [][]byte{{0x10, 0x01}, {0x10, 0x01, 0xff}, {0x10, 0x02}} {
		decoded, err := DecodeEnhancedMeasurementReport(input)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := EncodeEnhancedMeasurementReport(decoded.Value)
		if err != nil || !bytes.Equal(replay, input) {
			t.Fatalf("short replay %x -> %x: %v", input, replay, err)
		}
	}
	for _, d := range Descriptors() {
		if d.Name == "Enhanced Measurement report" {
			if d.MaxOctets != 21 || d.Extent.MinimumBits != 168 || d.Extent.MaximumBits != 168 || !d.Extent.Fixed || !d.Extent.Finite {
				t.Fatalf("extent %+v", d.Extent)
			}
			return
		}
	}
	t.Fatal("missing descriptor")
}

func TestEnhancedMeasurementReportNestedCompleteCarrier(t *testing.T) {
	decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	for _, received := range []bool{false, true} {
		w := runtime.NewWriter()
		if err := w.WriteUint(5, 3); err != nil {
			t.Fatal(err)
		}
		v := decoded.Value
		wantBits := 16
		if !received {
			v = runtime.Canonical(v)
			wantBits = 168
		}
		if err := EncodeEnhancedMeasurementReportTo(w, v); err != nil {
			t.Fatal(err)
		}
		if w.Position() != 3+wantBits {
			t.Fatalf("received=%t end=%d", received, w.Position())
		}
		if err := w.WriteUint(3, 2); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnhancedMeasurementReportCompleteCarrierServingFields(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55, Serving cell data struct. The complete
	// synthetic carrier is also decoded by an independent field oracle.
	v := EnhancedMeasurementReport{MessageType: 4, BAUSED: 1, BSICSeen: 1,
		ServingCellData:    &ServingCellDataStruct{DTXUSED: 1, RXLEVVAL: 37, RXQUALFULL: 5, MEANBEP: 17, CVBEP: 3, NBRRCVDBLOCKS: 19},
		BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeNoString, NoString: &EnhancedMeasurementReportBITMAPLENGTHChoiceNoString{}}}
	out, err := EncodeEnhancedMeasurementReportCanonical(v)
	want := append([]byte{0x10, 0xae, 0x5b, 0x17, 0x33}, bytes.Repeat([]byte{0x2b}, 16)...)
	if err != nil || !bytes.Equal(out, want) {
		t.Fatalf("serving carrier %x: %v", out, err)
	}
	decoded, err := DecodeEnhancedMeasurementReport(out)
	if err != nil || decoded.Value.ServingCellData == nil || !runtime.SemanticallyEqual(*decoded.Value.ServingCellData, *v.ServingCellData) {
		t.Fatalf("serving fields %+v: %v", decoded.Value.ServingCellData, err)
	}
}

func TestEnhancedMeasurementReportCompleteCarrierNullReleaseBoundaries(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.55: null at an exact received boundary
	// and L in a longer carrier both omit the release fields. Explicit
	// Rel-8/Rel-9 H arms remain present when their following null gains fill.
	for _, rel11 := range []bool{false, true} {
		decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0, 0, 0})
		if err != nil {
			t.Fatal(err)
		}
		v := runtime.Canonical(decoded.Value)
		rel8 := v.BITMAPLENGTHChoice.BITMAPLENGTH
		count := 8
		rel8.UTRANCSGMeasurementReportChoice = EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}
		if rel11 {
			count = 6
			rel8.UTRANCSGMeasurementReportChoice = EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeUTRANCSGMeasurementReport, UTRANCSGMeasurementReport: &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReport{SI23BAUSEDChoice: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}}
		}
		rel8.BITMAPLENGTH = uint8(count - 1)
		rel8.REPORTINGQUANTITYList = make([]*uint8, count)
		out, err := EncodeEnhancedMeasurementReportCanonical(v)
		if err != nil || len(out) != 21 {
			t.Fatalf("Rel-11=%t: %x, %v", rel11, out, err)
		}
		again, err := DecodeEnhancedMeasurementReport(out)
		if err != nil {
			t.Fatal(err)
		}
		got := again.Value.BITMAPLENGTHChoice.BITMAPLENGTH
		if got == nil || got.BITMAPLENGTH != uint8(count-1) || len(got.REPORTINGQUANTITYList) != count {
			t.Fatalf("Rel-8 fields: %+v", got)
		}
		if rel11 {
			rel9 := got.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
			if rel9 == nil || rel9.UTRANCSGMeasurementReport != nil || rel9.SI23BAUSEDChoice.SI23BAUSED != nil {
				t.Fatalf("release fields: %+v", rel9)
			}
			if rel8.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport.SI23BAUSEDChoice.AltUnlabeled == nil {
				t.Fatal("modified caller's Rel-11 boundary")
			}
		} else {
			if got.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport != nil {
				t.Fatal("introduced Rel-9 fields")
			}
			if rel8.UTRANCSGMeasurementReportChoice.AltUnlabeled == nil {
				t.Fatal("modified caller's Rel-9 boundary")
			}
		}
	}
}

func TestEnhancedMeasurementReportNearFullNestedCarrier(t *testing.T) {
	decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	v := runtime.Canonical(decoded.Value)
	entries := make([]*uint8, 96)
	for i := 0; i < 9; i++ {
		zero := uint8(0)
		entries[i] = &zero
	}
	v.REPORTINGQUANTITYList = &entries
	for _, offset := range []int{0, 1, 2, 3, 4, 5, 6, 7} {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, offset); err != nil {
			t.Fatal(err)
		}
		if err := EncodeEnhancedMeasurementReportTo(w, v); err != nil {
			t.Fatalf("offset %d: %v", offset, err)
		}
		if w.Position() != offset+168 {
			t.Fatalf("offset %d end %d", offset, w.Position())
		}
		if err := w.WriteUint(3, 2); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnhancedMeasurementReportFreshCarrierInsideReceivedWriter(t *testing.T) {
	decoded, err := DecodeEnhancedMeasurementReport([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	for _, extent := range []int{160, 168, 200} {
		w := runtime.NewWriter()
		w.WithWire(decoded.Value.Wire)
		priorBits, priorReceived := w.ReceivedBits()
		if _, err := w.PushLimit(extent); err != nil {
			t.Fatal(err)
		}
		err := EncodeEnhancedMeasurementReportTo(w, runtime.Canonical(decoded.Value))
		if extent < 168 {
			if err == nil {
				t.Fatal("fresh carrier fit a 160-bit containing bound")
			}
			continue
		}
		if err != nil || w.Position() != 168 || w.RemainingLimit() != extent-168 {
			t.Fatalf("extent=%d end=%d remaining=%d: %v", extent, w.Position(), w.RemainingLimit(), err)
		}
		bits, received := w.ReceivedBits()
		if bits != priorBits || received != priorReceived {
			t.Fatal("fresh carrier replaced parent's received layout")
		}
	}
}
