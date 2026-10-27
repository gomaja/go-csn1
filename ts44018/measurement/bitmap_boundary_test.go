package measurement

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func absentLegacyCarrier() []byte {
	wire := make([]byte, 21)
	wire[0], wire[1] = 0x10, 0x02
	return wire
}

// TS 44.018 V19.0.0 §3.4.1.2.1.3 and table 9.1.55.1: after
// the 96 cell positions only a wholly zero remainder is redundant bitmap.
func TestEnhancedMeasurementReportZeroTailBoundary(t *testing.T) {
	for count := 0; count <= 153; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			wire := absentLegacyCarrier()
			r := runtime.NewReader(wire)
			if _, err := r.PushLimit(15 + count); err != nil {
				t.Fatal(err)
			}
			v, err := DecodeEnhancedMeasurementReportFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if v.REPORTINGQUANTITYList == nil || len(*v.REPORTINGQUANTITYList) != count || v.BITMAPLENGTHChoice.AltUnlabeled == nil || r.Position() != 15+count {
				t.Fatalf("count %d: bitmap=%v release=%+v end=%d", count, v.REPORTINGQUANTITYList, v.BITMAPLENGTHChoice, r.Position())
			}
		})
	}
	short := absentLegacyCarrier()[:14]
	d, err := DecodeEnhancedMeasurementReport(short)
	if err != nil || len(*d.Value.REPORTINGQUANTITYList) != 97 {
		t.Fatalf("short: %+v, %v", d, err)
	}
	out, err := EncodeEnhancedMeasurementReport(d.Value)
	if err != nil || !bytes.Equal(out, short) {
		t.Fatalf("short replay %x: %v", out, err)
	}
}

func TestEnhancedMeasurementReportNonzeroTailBoundary(t *testing.T) {
	for bit := 111; bit < 168; bit++ {
		wire := absentLegacyCarrier()
		wire[bit/8] |= 1 << uint(7-bit%8)
		r := runtime.NewReader(wire)
		if err := r.Expect("000100000000001"); err != nil {
			t.Fatal(err)
		}
		v, err := decodeEnhancedMeasurementReportREPORTINGQUANTITYListValue(r)
		if err != nil || len(v) != 96 || r.Position() != 111 {
			t.Fatalf("bit %d: entries=%d end=%d: %v", bit, len(v), r.Position(), err)
		}
	}
	// A one outside the enclosing limit cannot select release additions.
	wire := absentLegacyCarrier()
	wire[20] = 1
	r := runtime.NewReader(wire)
	old, err := r.PushLimit(160)
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeEnhancedMeasurementReportFrom(r)
	if err != nil || len(*v.REPORTINGQUANTITYList) != 145 || v.BITMAPLENGTHChoice.AltUnlabeled == nil {
		t.Fatalf("enclosing boundary: %+v: %v", v, err)
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	last, err := r.ReadUint(8)
	if err != nil || last != 1 {
		t.Fatalf("following value %d: %v", last, err)
	}
}

func TestEnhancedMeasurementReportPaddedBoundary(t *testing.T) {
	wire := append([]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, bytes.Repeat([]byte{0x2b}, 7)...)
	d, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil || d.Value.REPORTINGQUANTITYList == nil || len(*d.Value.REPORTINGQUANTITYList) != 96 || d.Value.BITMAPLENGTHChoice.NoString == nil {
		t.Fatalf("padded boundary: %+v: %v", d, err)
	}
	out, err := EncodeEnhancedMeasurementReport(d.Value)
	if err != nil || !bytes.Equal(out, wire) {
		t.Fatalf("padded replay %x: %v", out, err)
	}
}

func TestEnhancedMeasurementReportFreshRedundantPositions(t *testing.T) {
	for _, count := range []int{0, 1, 95, 96, 97, 98, 99, 104, 128, 153} {
		entries := make([]*uint8, count)
		v := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries, BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
		for _, encode := range []func(EnhancedMeasurementReport) ([]byte, error){EncodeEnhancedMeasurementReport, EncodeEnhancedMeasurementReportCanonical} {
			out, err := encode(v)
			if err != nil || !bytes.Equal(out, absentLegacyCarrier()) {
				t.Fatalf("count %d: %x: %v", count, out, err)
			}
		}
	}
	entries := make([]*uint8, 97)
	present := uint8(1)
	entries[96] = &present
	v := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries, BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
	for _, encode := range []func(EnhancedMeasurementReport) ([]byte, error){EncodeEnhancedMeasurementReport, EncodeEnhancedMeasurementReportCanonical} {
		_, err := encode(v)
		var located *runtime.OperationError
		if !errors.As(err, &located) {
			t.Fatalf("present position 97: %v", err)
		}
	}
}

func TestEnhancedMeasurementReportNilBitmapConstruction(t *testing.T) {
	// TS 44.018 V19.0.0 table 9.1.55.1: a present empty bitmap and
	// explicit trailing absent positions have the same reported quantities.
	var entries []*uint8
	v := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries,
		BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
	for _, encode := range []func(EnhancedMeasurementReport) ([]byte, error){EncodeEnhancedMeasurementReport, EncodeEnhancedMeasurementReportCanonical} {
		out, err := encode(v)
		if err != nil || !bytes.Equal(out, absentLegacyCarrier()) {
			t.Fatalf("nil bitmap: %x: %v", out, err)
		}
	}
	out, err := EncodeEnhancedMeasurementReportCanonicalAtLength(v, 21)
	if err != nil || !bytes.Equal(out, absentLegacyCarrier()) {
		t.Fatalf("nil bitmap at length: %x: %v", out, err)
	}
	r := runtime.NewReader(absentLegacyCarrier())
	if _, err := r.PushLimit(15); err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeEnhancedMeasurementReportFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	out, err = EncodeEnhancedMeasurementReportCanonical(decoded)
	if err != nil || !bytes.Equal(out, absentLegacyCarrier()) {
		t.Fatalf("received zero-position bitmap: %x: %v", out, err)
	}
	withRelease := legacyWithRelease(0)
	withRelease.REPORTINGQUANTITYList = &entries
	out, err = EncodeEnhancedMeasurementReportCanonical(withRelease)
	if err != nil {
		t.Fatalf("nil bitmap with additions: %v", err)
	}
	again, err := DecodeEnhancedMeasurementReport(out)
	if err != nil || again.Value.REPORTINGQUANTITYList == nil || len(*again.Value.REPORTINGQUANTITYList) != 96 || again.Value.BITMAPLENGTHChoice.BITMAPLENGTH == nil {
		t.Fatalf("nil bitmap release fields: %+v: %v", again, err)
	}
	if entries != nil {
		t.Fatal("construction changed the caller's nil slice")
	}
}

func legacyWithRelease(count int) EnhancedMeasurementReport {
	entries := make([]*uint8, count)
	return EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries,
		BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{
			Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeBITMAPLENGTH,
			BITMAPLENGTH: &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTH{
				REPORTINGQUANTITYList: make([]*uint8, 1),
				UTRANCSGMeasurementReportChoice: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{
					Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeNoString,
					NoString:    &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceNoString{},
				},
			},
		}}
}

func TestEnhancedMeasurementReportFreshReleaseBoundary(t *testing.T) {
	for _, count := range []int{0, 1, 95, 96, 97, 98, 99, 104, 128, 153} {
		v := legacyWithRelease(count)
		for _, encode := range []func(EnhancedMeasurementReport) ([]byte, error){EncodeEnhancedMeasurementReport, EncodeEnhancedMeasurementReportCanonical} {
			wire, err := encode(v)
			if count > 96 {
				var located *runtime.OperationError
				if !errors.As(err, &located) {
					t.Fatalf("count %d additions accepted: %v", count, err)
				}
				continue
			}
			if err != nil {
				t.Fatalf("count %d: %v", count, err)
			}
			d, err := DecodeEnhancedMeasurementReport(wire)
			if err != nil || len(*d.Value.REPORTINGQUANTITYList) != 96 || d.Value.BITMAPLENGTHChoice.BITMAPLENGTH == nil {
				t.Fatalf("count %d: %+v: %v", count, d, err)
			}
		}
	}
	// The independent Rel-8 seven-bit count still permits 128 entries.
	v := legacyWithRelease(0)
	v.REPORTINGQUANTITYList = nil
	v.BITMAPLENGTHChoice.BITMAPLENGTH.BITMAPLENGTH = 127
	v.BITMAPLENGTHChoice.BITMAPLENGTH.REPORTINGQUANTITYList = make([]*uint8, 128)
	wire, err := EncodeEnhancedMeasurementReportCanonical(v)
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil || len(d.Value.BITMAPLENGTHChoice.BITMAPLENGTH.REPORTINGQUANTITYList) != 128 {
		t.Fatalf("Rel-8 128: %+v: %v", d, err)
	}
}

func TestEnhancedMeasurementReportCannotFitMinimum(t *testing.T) {
	// 21 present quantities consume 147 bits; only six absent positions
	// fit after the 15-bit prefix. The source permits this shorter bitmap.
	entries := make([]*uint8, 21)
	for i := range entries {
		q := uint8(i)
		entries[i] = &q
	}
	v := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries,
		BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
	wire, err := EncodeEnhancedMeasurementReportCanonical(v)
	if err != nil {
		t.Fatal(err)
	}
	d, err := DecodeEnhancedMeasurementReport(wire)
	if err != nil || len(*d.Value.REPORTINGQUANTITYList) != 27 || d.Value.BITMAPLENGTHChoice.AltUnlabeled == nil {
		t.Fatalf("cannot-fit: %+v: %v", d, err)
	}
	if !equivalentCanonicalEMR(v, d.Value) {
		t.Fatal("lost reported quantities")
	}
	q := uint8(1)
	entries = append(entries, &q)
	if _, err := EncodeEnhancedMeasurementReportCanonical(v); err == nil {
		t.Fatal("22 present quantities fit the carrier")
	}
}

func TestEnhancedMeasurementReportBoundaryGrowth(t *testing.T) {
	// Every possible enclosing extent is exercised, rather than a sample
	// of counts. Each absent position consumes exactly one input bit.
	for bits := 15; bits <= 168; bits++ {
		r := runtime.NewReader(absentLegacyCarrier())
		if _, err := r.PushLimit(bits); err != nil {
			t.Fatal(err)
		}
		v, err := DecodeEnhancedMeasurementReportFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		if got := len(*v.REPORTINGQUANTITYList); got != bits-15 || got > r.Position() || r.Position() != bits {
			t.Fatalf("bits %d entries %d end %d", bits, got, r.Position())
		}
	}
}

func TestEnhancedMeasurementReportEveryReleaseTruncation(t *testing.T) {
	v := legacyWithRelease(96)
	base, err := EncodeEnhancedMeasurementReport(v)
	if err != nil {
		t.Fatal(err)
	}
	for bits := 111; bits <= 168; bits++ {
		r := runtime.NewReader(base)
		if _, err := r.PushLimit(bits); err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeEnhancedMeasurementReportFrom(r)
		zero := true
		for bit := 111; bit < bits; bit++ {
			if base[bit/8]&(1<<uint(7-bit%8)) != 0 {
				zero = false
			}
		}
		if zero {
			if err != nil || decoded.REPORTINGQUANTITYList == nil || len(*decoded.REPORTINGQUANTITYList) != bits-15 || decoded.BITMAPLENGTHChoice.AltUnlabeled == nil {
				t.Fatalf("zero suffix at %d: %+v: %v", bits, decoded, err)
			}
			continue
		}
		// Parse the independently fixed release boundary for comparison.
		suffixReader := runtime.NewReader(base)
		if _, e := suffixReader.PushLimit(bits); e != nil {
			t.Fatal(e)
		}
		if _, e := suffixReader.ReadBitString(111); e != nil {
			t.Fatal(e)
		}
		suffix, suffixErr := decodeEnhancedMeasurementReportBITMAPLENGTHChoice(suffixReader)
		if (err == nil) != (suffixErr == nil) {
			t.Fatalf("suffix at %d: root=%v suffix=%v", bits, err, suffixErr)
		}
		if err == nil && (len(*decoded.REPORTINGQUANTITYList) != 96 || !runtime.SemanticallyEqual(suffix, decoded.BITMAPLENGTHChoice)) {
			t.Fatalf("release fields at %d", bits)
		}
	}
}

func TestEnhancedMeasurementReportEveryRedundantPositionRefusal(t *testing.T) {
	for index := 96; index < 153; index++ {
		entries := make([]*uint8, index+1)
		q := uint8(0)
		entries[index] = &q
		v := EnhancedMeasurementReport{MessageType: 4, REPORTINGQUANTITYList: &entries, BITMAPLENGTHChoice: EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}}
		for _, encode := range []func(EnhancedMeasurementReport) ([]byte, error){EncodeEnhancedMeasurementReport, EncodeEnhancedMeasurementReportCanonical} {
			_, err := encode(v)
			var located *runtime.OperationError
			if !errors.As(err, &located) {
				t.Fatalf("present position %d: %v", index+1, err)
			}
		}
	}
}
