package packetenhancedmeasurement

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/csgreport"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/padding"
)

// TS 44.060 V19.0.0 §11.2.9d pp16070–16103, §11 and §12.60.
type PaddingBits = padding.PaddingBits
type PaddingBitsIgnoreChoice = padding.PaddingBitsIgnoreChoice
type PaddingBitsIgnoreChoiceIgnore = padding.PaddingBitsIgnoreChoiceIgnore
type PaddingBitsIgnoreChoiceIgnoreKnown = padding.PaddingBitsIgnoreChoiceIgnoreKnown

const PaddingBitsIgnoreChoiceAlternativeIgnore = padding.PaddingBitsIgnoreChoiceAlternativeIgnore
const PaddingBitsIgnoreChoiceIgnoreAlternativeKnown = padding.PaddingBitsIgnoreChoiceIgnoreAlternativeKnown

type EUTRANCSGMeasurementReportIE = csgreport.EUTRANCSGMeasurementReportIE
type message = PACKETENHANCEDMEASUREMENTREPORTMessageContent

func fresh() message {
	return PACKETENHANCEDMEASUREMENTREPORTMessageContent{NCMeasurementReport: NCMeasurementReportStruct{BAUSEDChoice: NCMeasurementReportStructBAUSEDChoice{Alternative: NCMeasurementReportStructBAUSEDChoiceAlternativeBAUSED, BAUSED: &NCMeasurementReportStructBAUSEDChoiceBAUSED{}}}, GRNTIExtensionChoice: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
}
func reports(v message) *[]*uint8             { return v.NCMeasurementReport.REPORTINGQUANTITYList }
func setReports(v *message, entries []*uint8) { v.NCMeasurementReport.REPORTINGQUANTITYList = &entries }
func release(v *message) {
	v.GRNTIExtensionChoice = PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceAlternativeGRNTIExtension, GRNTIExtension: &PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtension{BITMAPLENGTHChoice: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceAlternativeBITMAPLENGTH, BITMAPLENGTH: &PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTH{REPORTINGQUANTITYList: make([]*uint8, 1), UTRANCSGMeasurementReportChoice: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}}}}
}
func congested(v *message) {
	v.NCMeasurementReport.RepeatedInvalidBSICInformationList = make([]NCMeasurementReportStructRepeatedInvalidBSICInformationListEntry, 2)
}
func checkReports(t *testing.T, v message, index int, quantity uint8, minimum int) {
	t.Helper()
	p := reports(v)
	if p == nil || len(*p) < minimum {
		t.Fatalf("bitmap length %v, want >=%d", p, minimum)
	}
	for i, q := range *p {
		if i == index {
			if q == nil || *q != quantity {
				t.Fatalf("position %d quantity %v want %d", i, q, quantity)
			}
		} else if q != nil {
			t.Fatalf("unexpected position %d", i)
		}
	}
}
func checkValue(t *testing.T, v message, index int, quantity uint8, minimum int) {
	t.Helper()
	for _, encode := range []func(message) ([]byte, error){EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent, EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical, func(v message) ([]byte, error) {
		return EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonicalAtLength(v, 22)
	}} {
		wire, err := encode(v)
		if err != nil {
			t.Fatal(err)
		}
		if len(wire) != 22 {
			t.Fatalf("wire extent %d", len(wire))
		}
		got, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
		if err != nil {
			t.Fatal(err)
		}
		checkReports(t, got.Value, index, quantity, minimum)
		if minimum == 96 {
			want := 128
			if index >= 0 {
				want -= 6
			}
			if len(*reports(got.Value)) != want {
				t.Fatalf("bitmap positions %d want %d", len(*reports(got.Value)), want)
			}
		}
		replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(got.Value)
		if err != nil || !bytes.Equal(wire, replay) {
			t.Fatalf("replay %x != %x: %v", replay, wire, err)
		}
	}
}
func TestEveryBitmapPosition(t *testing.T) {
	for i := 0; i < 96; i++ {
		for _, q := range []uint8{0, 63} {
			t.Run(fmt.Sprintf("%d/%d", i, q), func(t *testing.T) {
				v := fresh()
				entries := make([]*uint8, 96)
				quantity := q
				entries[i] = &quantity
				setReports(&v, entries)
				checkValue(t, v, i, q, 96)
			})
		}
	}
}
func TestAbsentPresentAndRedundantPositions(t *testing.T) {
	v := fresh()
	wire, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
	if err != nil {
		t.Fatal(err)
	}
	absent, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
	if err != nil || reports(absent.Value) != nil {
		t.Fatalf("absent bitmap: %v", err)
	}
	for _, n := range []int{0, 1, 96, 97} {
		v := fresh()
		setReports(&v, make([]*uint8, n))
		checkValue(t, v, -1, 0, 96)
		wire, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
		if err != nil {
			t.Fatal(err)
		}
		if len(*reports(got.Value)) != 128 {
			t.Fatalf("legacy zero tail positions %d want %d", len(*reports(got.Value)), 128)
		}
		// Unused backing bits are retained for byte replay but are never bitmap positions.
		if 170%8 != 0 {
			wire[len(wire)-1] |= 0x3f
			got, err = DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
			if err != nil || len(*reports(got.Value)) != 128 {
				t.Fatalf("completion counted as bitmap: %v", err)
			}
			replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(got.Value)
			if err != nil || !bytes.Equal(wire, replay) {
				t.Fatalf("completion replay: %v", err)
			}
		}
	}
	v = fresh()
	entries := make([]*uint8, 96)
	for i := range entries {
		q := uint8(i % 64)
		entries[i] = &q
	}
	setReports(&v, entries)
	if _, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v); err == nil {
		t.Fatal("all 96 quantities fit a carrier too small for them")
	}
	v = fresh()
	entries = make([]*uint8, 97)
	q := uint8(0)
	entries[96] = &q
	setReports(&v, entries)
	if _, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v); err == nil {
		t.Fatal("present position 96 accepted")
	}
	v = fresh()
	setReports(&v, make([]*uint8, 0))
	congested(&v)
	checkValue(t, v, -1, 0, 92)
}
func TestExplicitRelease(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		v := fresh()
		if legacy {
			setReports(&v, make([]*uint8, 96))
		}
		release(&v)
		for _, encode := range []func(message) ([]byte, error){EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent, EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical} {
			wire, err := encode(v)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
			if err != nil {
				t.Fatal(err)
			}
			if legacy && len(*reports(got.Value)) != 96 {
				t.Fatalf("release swallowed by bitmap: %d", len(*reports(got.Value)))
			}
			if got.Value.GRNTIExtensionChoice.GRNTIExtension == nil || got.Value.GRNTIExtensionChoice.GRNTIExtension.BITMAPLENGTHChoice.BITMAPLENGTH == nil {
				t.Fatal("release fields lost")
			}
			replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(got.Value)
			if err != nil || !bytes.Equal(wire, replay) {
				t.Fatalf("release replay: %v", err)
			}
		}
	}
}
func TestFollowingSentinel(t *testing.T) {
	for _, canonical := range []bool{false, true} {
		v := fresh()
		setReports(&v, make([]*uint8, 96))
		if canonical {
			v = runtime.Canonical(v)
		}
		w := runtime.NewWriter()
		if err := w.WriteUint(0, 14); err != nil {
			t.Fatal(err)
		}
		if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		if w.Position() != 14+170 {
			t.Fatalf("carrier ends at %d want %d", w.Position(), 14+170)
		}
		if err := w.WriteUint(5, 3); err != nil {
			t.Fatal(err)
		}
		wire, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		r, err := runtime.NewReaderBits(wire.Bytes, wire.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(170)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		checkReports(t, got, -1, 0, 96)
		if len(*reports(got)) != 128 {
			t.Fatalf("physical bitmap positions %d", len(*reports(got)))
		}
		if err := r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(3)
		if err != nil || sentinel != 5 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
	}
}
func FuzzMeasurementCarrier(f *testing.F) {
	f.Add(make([]byte, 22))
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		v, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
		if err != nil {
			return
		}
		replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v.Value)
		if err != nil || !bytes.Equal(wire, replay) {
			t.Fatalf("replay %x != %x: %v", wire, replay, err)
		}
	})
}

// The first legacy selector follows 42 bits of printed header and fields.
// Every present quantity adds its selector and six bits; omitted positions are zero.
func TestClauseDerivedBitmapBits(t *testing.T) {
	for index := -1; index < 96; index++ {
		for _, quantity := range []uint8{0, 63} {
			v := fresh()
			entries := make([]*uint8, 96)
			if index >= 0 {
				q := quantity
				entries[index] = &q
			}
			setReports(&v, entries)
			wire, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
			if err != nil {
				t.Fatal(err)
			}
			expected := make([]byte, 22)
			expected[0] = 0
			set := func(pos int) { expected[pos/8] |= 1 << uint(7-pos%8) }
			set(42 - 1)
			if index >= 0 {
				set(42 + index)
				for i := 0; i < 6; i++ {
					if quantity&(1<<uint(5-i)) != 0 {
						set(42 + index + 1 + i)
					}
				}
			}
			for pos := 0; pos < 170; pos++ {
				mask := byte(1 << uint(7-pos%8))
				if wire[pos/8]&mask != expected[pos/8]&mask {
					t.Fatalf("position %d quantity %d: wire bit %d differs: %x want %x", index, quantity, pos, wire, expected)
				}
			}
		}
	}
}
func TestReplayedSignificantCarrier(t *testing.T) {
	v := fresh()
	setReports(&v, make([]*uint8, 96))
	wire, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
	if err != nil {
		t.Fatal(err)
	}
	received, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
	if err != nil {
		t.Fatal(err)
	}
	// Reinsert byte-backed content: completion is outside the radio-block interval.
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(w, received.Value); err != nil {
		t.Fatal(err)
	}
	if w.Position() != 14+170 {
		t.Fatalf("byte completion entered carrier: end %d want %d", w.Position(), 14+170)
	}
	if err := w.WriteUint(5, 3); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(170)
	if err != nil {
		t.Fatal(err)
	}
	var decoded runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]
	found := false
	for _, d := range Descriptors() {
		if d.Name == "PACKET ENHANCED MEASUREMENT REPORT message content" {
			value, err := d.DecodeFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			var ok bool
			decoded, ok = value.(runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent])
			if !ok {
				t.Fatalf("descriptor value %T", value)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("root descriptor absent")
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(3)
	if err != nil || sentinel != 5 {
		t.Fatalf("replayed sentinel %d: %v", sentinel, err)
	}
	replay := runtime.NewWriter()
	if err := replay.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(replay, decoded.Value); err != nil {
		t.Fatal(err)
	}
	if err := replay.WriteUint(5, 3); err != nil {
		t.Fatal(err)
	}
	again, err := replay.FinishBits(runtime.BitString{})
	if err != nil || again.BitLength != raw.BitLength || !bytes.Equal(again.Bytes, raw.Bytes) {
		t.Fatalf("significant replay %x/%d != %x/%d: %v", again.Bytes, again.BitLength, raw.Bytes, raw.BitLength, err)
	}
}

func TestPacketLaterReleaseFields(t *testing.T) {
	for _, kind := range []string{"quantity-zero", "identity-zero", "utran", "eutran", "si23"} {
		v := fresh()
		release(&v)
		r5 := v.GRNTIExtensionChoice.GRNTIExtension
		r8 := r5.BITMAPLENGTHChoice.BITMAPLENGTH
		q := uint8(0)
		switch kind {
		case "quantity-zero":
			r8.REPORTINGQUANTITYList = []*uint8{&q}
		case "identity-zero":
			r5.GRNTIExtension = &q
		case "utran", "eutran", "si23":
			r8.UTRANCSGMeasurementReportChoice = PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeUTRANCSGMeasurementReport, UTRANCSGMeasurementReport: &PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReport{SI23BAUSEDChoice: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}}
			r9 := r8.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
			switch kind {
			case "utran":
				r9.UTRANCSGMeasurementReport = &ies.UTRANCSGMeasurementReportIE{UTRANCGI: 123, CSGID: 456, REPORTINGQUANTITY: 63}
			case "eutran":
				r9.EUTRANCSGMeasurementReport = &EUTRANCSGMeasurementReportIE{EUTRANCI: 123, TrackingAreaCode: 456, CSGID: 789, REPORTINGQUANTITY: 63}
			case "si23":
				r9.SI23BAUSEDChoice = PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{Alternative: PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeSI23BAUSED, SI23BAUSED: &PACKETENHANCEDMEASUREMENTREPORTMessageContentGRNTIExtensionChoiceGRNTIExtensionBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceSI23BAUSED{SI23BAUSED: 1, PaddingBits: PaddingBits{IgnoreChoice: PaddingBitsIgnoreChoice{Alternative: PaddingBitsIgnoreChoiceAlternativeIgnore, Ignore: &PaddingBitsIgnoreChoiceIgnore{Alternative: PaddingBitsIgnoreChoiceIgnoreAlternativeKnown, Known: &PaddingBitsIgnoreChoiceIgnoreKnown{}}}}}}
			}
		}
		for _, encode := range []func(message) ([]byte, error){EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent, EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical} {
			wire, err := encode(v)
			if err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			got, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
			if err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			if !runtime.SemanticallyEqual(runtime.Canonical(v), runtime.Canonical(got.Value)) {
				a, b := runtime.Canonical(v), runtime.Canonical(got.Value)
				emrReleaseFields(&a)
				emrReleaseFields(&b)
				if !runtime.SemanticallyEqual(a, b) {
					t.Fatalf("%s fields changed", kind)
				}
			}
			replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(got.Value)
			if err != nil || !bytes.Equal(wire, replay) {
				t.Fatalf("%s replay: %v", kind, err)
			}
		}
	}
}

func TestReleaseRejectsLongLegacyBitmap(t *testing.T) {
	v := fresh()
	setReports(&v, make([]*uint8, 97))
	release(&v)
	for _, encode := range []func(message) ([]byte, error){EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent, EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical} {
		if _, err := encode(v); err == nil {
			t.Fatal("long legacy bitmap accepted with explicit release")
		}
	}
}
func TestReceivedQuantityEdit(t *testing.T) {
	v := fresh()
	entries := make([]*uint8, 96)
	q := uint8(0)
	entries[0] = &q
	setReports(&v, entries)
	wire, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
	if err != nil {
		t.Fatal(err)
	}
	received, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(wire)
	if err != nil {
		t.Fatal(err)
	}
	*(*reports(received.Value))[0] = 63
	edited, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(received.Value)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(edited, wire) {
		t.Fatal("quantity edit did not change the bits")
	}
	again, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(edited)
	if err != nil {
		t.Fatal(err)
	}
	checkReports(t, again.Value, 0, 63, 96)
}

func TestPacketInputExtent(t *testing.T) {
	for n := 0; n <= 5; n++ {
		if _, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(make([]byte, n)); err == nil {
			t.Fatalf("short input %d accepted", n)
		}
	}
	for n := 23; n <= 65536; n *= 2 {
		if _, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(make([]byte, n)); err == nil {
			t.Fatalf("oversized input %d accepted", n)
		}
	}
}

func TestPacketContentOrigin(t *testing.T) {
	for _, canonical := range []bool{false, true} {
		v := fresh()
		var content []byte
		var err error
		if canonical {
			content, err = EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical(v)
			v = runtime.Canonical(v)
		} else {
			content, err = EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(v)
		}
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		if err := w.WriteUint(0x100a, 14); err != nil {
			t.Fatal(err)
		}
		if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		block, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		for bit := 0; bit < 170; bit++ {
			a := (content[bit/8] >> uint(7-bit%8)) & 1
			b := (block.Bytes[(bit+14)/8] >> uint(7-(bit+14)%8)) & 1
			if a != b {
				t.Fatalf("Packet content origin lost at bit %d (canonical=%t): byte API %x, radio block %x", bit, canonical, content, block.Bytes)
			}
		}
	}
}

// TS 44.060 V19.0.0 sections 10.3.2, 11.2.0.2 and 11.2.9d:
// the 184-bit block contains 14 framing bits and 170 significant content bits.
func TestSignificantExtentAndCompletion(t *testing.T) {
	content, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(fresh())
	if err != nil {
		t.Fatal(err)
	}
	d, err := Lookup("PACKET ENHANCED MEASUREMENT REPORT message content")
	if err != nil {
		t.Fatal(err)
	}
	if SourceOrigin(d) != 14 {
		t.Fatal("descriptor lost source origin")
	}
	for completion := byte(0); completion < 64; completion++ {
		input := append([]byte(nil), content...)
		input[len(input)-1] = input[len(input)-1]&0xc0 | completion
		decoded, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(input)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(replay, input) {
			t.Fatalf("completion %d not retained: %v", completion, err)
		}
		w := runtime.NewWriter()
		if err := w.WriteUint(0, 14); err != nil {
			t.Fatal(err)
		}
		if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(w, decoded.Value); err != nil {
			t.Fatal(err)
		}
		if w.Position() != 184 {
			t.Fatalf("completion %d transmitted as significant content: %d", completion, w.Position())
		}
		if err := w.WriteUint(5, 3); err != nil {
			t.Fatal(err)
		}
		raw, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(170)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContentFrom(r); err != nil {
			t.Fatal(err)
		}
		if err := r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if sentinel, err := r.ReadUint(3); err != nil || sentinel != 5 {
			t.Fatalf("completion %d changed sentinel: %d %v", completion, sentinel, err)
		}
	}
	for bits := 170; bits <= 176; bits++ {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, 14); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteBitString(runtime.BitString{Bytes: content, BitLength: 176}, 176); err != nil {
			t.Fatal(err)
		}
		raw, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		r, err := runtime.NewReaderBits(raw.Bytes, 14+bits, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		_, err = DecodePACKETENHANCEDMEASUREMENTREPORTMessageContentFrom(r)
		if bits == 170 {
			if err != nil || r.Position() != 184 {
				t.Fatalf("significant bound: %v", err)
			}
		} else {
			var de *runtime.DecodeError
			if !errors.As(err, &de) || de.Kind != runtime.Limit || r.Position() != 14 {
				t.Fatalf("partial backing completion %d: %v", bits, err)
			}
		}
	}
}

func TestCallerPhaseBeforeInput(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteBitString(runtime.BitString{Bytes: make([]byte, 22), BitLength: 170}, 170); err != nil {
			t.Fatal(err)
		}
		raw, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		c := runtime.NewReceiverCollector()
		r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		r.Set("marker", 19)
		_, err = DecodePACKETENHANCEDMEASUREMENTREPORTMessageContentFrom(r)
		target := runtime.NewWriter()
		if err := target.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		writerErr := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(target, fresh())
		if phase == 6 {
			if err != nil || writerErr != nil {
				t.Fatalf("congruent phase: %v / %v", err, writerErr)
			}
			continue
		}
		var pe *ContentPhaseError
		if !errors.As(err, &pe) || pe.BitPosition() != phase || pe.SourceOrigin() != 14 || r.Position() != phase {
			t.Fatalf("reader phase %d: %v", phase, err)
		}
		if !errors.As(writerErr, &pe) || target.Position() != phase {
			t.Fatalf("writer phase %d: %v", phase, writerErr)
		}
		if marker, err := r.Eval("val(marker)"); err != nil || marker != 19 {
			t.Fatal("caller context changed")
		}
		if len(c.TakeReport().Observations) != 0 || len(c.TakeReceiverReport().Regions) != 0 {
			t.Fatal("phase refusal changed reports")
		}
	}
}

func TestReceivedPrefixKeepsCanonicalCarrierExtent(t *testing.T) {
	input := make([]byte, 20)
	decoded, err := DecodePACKETENHANCEDMEASUREMENTREPORTMessageContent(input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContent(decoded.Value)
	if err != nil || !bytes.Equal(replay, input) {
		t.Fatalf("received prefix replay: %v", err)
	}
	canonical, err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentCanonical(decoded.Value)
	if err != nil || len(canonical) != 22 {
		t.Fatalf("canonical source carrier: %d octets, %v", len(canonical), err)
	}
}
