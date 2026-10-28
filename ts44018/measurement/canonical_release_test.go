package measurement

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestCanonicalEnhancedMeasurementReportReleaseEmptyArms(t *testing.T) {
	// TS 44.018 V19.0.0 figure 9.1.55.1 has null and L/no-string
	// alternatives before Rel-9 and Rel-11. Neither carries release fields.
	// TS 24.007 V20.0.0 §§B.1.1.2, B.1.1.4, B.1.2.2 distinguish the
	// empty string, the choice and position-dependent padding.
	for _, count := range []int{1, 2, 3, 8, 32, 128} {
		legacyCounts := []int{0, 2}
		if count == 128 {
			legacyCounts = legacyCounts[:1]
		}
		for _, legacy := range legacyCounts {
			for _, reports := range []string{"none", "first-zero", "last-max"} {
				for _, release := range []int{9, 11} {
					t.Run(fmt.Sprintf("entries=%d/legacy=%d/reports=%s/release=%d", count, legacy, reports, release), func(t *testing.T) {
						v := legacyWithRelease(legacy)
						if count == 128 {
							v.REPORTINGQUANTITYList = nil
						}
						rel8 := v.BITMAPLENGTHChoice.BITMAPLENGTH
						rel8.BITMAPLENGTH = uint8(count - 1)
						rel8.REPORTINGQUANTITYList = make([]*uint8, count)
						switch reports {
						case "first-zero":
							q := uint8(0)
							rel8.REPORTINGQUANTITYList[0] = &q
						case "last-max":
							q := uint8(63)
							rel8.REPORTINGQUANTITYList[count-1] = &q
						}
						rel8.UTRANCSGMeasurementReportChoice = EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{
							Alternative:  EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeAltUnlabeled,
							AltUnlabeled: &struct{}{},
						}
						if release == 11 {
							rel8.UTRANCSGMeasurementReportChoice = EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{
								Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeUTRANCSGMeasurementReport,
								UTRANCSGMeasurementReport: &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReport{
									SI23BAUSEDChoice: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{
										Alternative:  EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeAltUnlabeled,
										AltUnlabeled: &struct{}{},
									},
								},
							}
						}
						before := runtime.Canonical(v)
						wire, err := EncodeEnhancedMeasurementReportCanonical(v)
						if err != nil {
							t.Fatal(err)
						}
						d, err := DecodeEnhancedMeasurementReport(wire)
						if err != nil {
							t.Fatal(err)
						}
						got := d.Value.BITMAPLENGTHChoice.BITMAPLENGTH
						if got == nil || got.BITMAPLENGTH != rel8.BITMAPLENGTH || !reflect.DeepEqual(got.REPORTINGQUANTITYList, rel8.REPORTINGQUANTITYList) {
							t.Fatal("canonical bytes changed the counted reporting bitmap")
						}
						if !equivalentCanonicalEMR(v, d.Value) {
							t.Fatal("empty release arms compare unequal")
						}
						if !runtime.SemanticallyEqual(v, before) {
							t.Fatal("comparison changed the caller's value")
						}
						replay, err := EncodeEnhancedMeasurementReport(d.Value)
						if err != nil || !bytes.Equal(wire, replay) {
							t.Fatalf("replay: %v", err)
						}
						framed, err := EncodeEnhancedMeasurementReportCanonicalAtLength(v, 21)
						if err != nil || !bytes.Equal(wire, framed) {
							t.Fatalf("framed canonical: %v", err)
						}
					})
				}
			}
		}
	}
}

func TestCanonicalEnhancedMeasurementReportPreservesReleaseFields(t *testing.T) {
	// TS 44.018 V19.0.0 table 9.1.55.1: the Rel-8 bitmap has exactly
	// BITMAP_LENGTH+1 positions; a zero quantity is a report, not absence.
	for count := 1; count <= 128; count++ {
		v := legacyWithRelease(0)
		rel8 := v.BITMAPLENGTHChoice.BITMAPLENGTH
		rel8.BITMAPLENGTH = uint8(count - 1)
		rel8.REPORTINGQUANTITYList = make([]*uint8, count)
		for _, change := range []string{"quantity", "count", "release"} {
			b := runtime.Canonical(v)
			other := b.BITMAPLENGTHChoice.BITMAPLENGTH
			switch change {
			case "quantity":
				q := uint8(0)
				other.REPORTINGQUANTITYList[count-1] = &q
			case "count":
				if count == 128 {
					other.BITMAPLENGTH--
					other.REPORTINGQUANTITYList = other.REPORTINGQUANTITYList[:count-1]
				} else {
					other.BITMAPLENGTH++
					other.REPORTINGQUANTITYList = append(other.REPORTINGQUANTITYList, nil)
				}
			case "release":
				other.UTRANCSGMeasurementReportChoice = EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{
					Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeUTRANCSGMeasurementReport,
					UTRANCSGMeasurementReport: &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReport{
						SI23BAUSEDChoice: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{
							Alternative: EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeSI23BAUSED,
							SI23BAUSED:  &EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceSI23BAUSED{SI23BAUSED: 1},
						},
					},
				}
			}
			if equivalentCanonicalEMR(v, b) {
				t.Fatalf("%d entries: ignored changed %s", count, change)
			}
		}
	}
}
