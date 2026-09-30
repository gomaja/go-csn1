package cellselection

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestCellSelectionAfterReleaseRATBranches(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.1e table 10.5.2.1e.1:
	// 3-bit RAT choice, 1-prefixed entries and a 0 terminator.
	fddBandwidth, tddBandwidth, eBandwidth, target := uint8(3), uint8(2), uint8(4), uint16(9)
	_ = fddBandwidth
	_ = tddBandwidth
	for _, tc := range []struct {
		name    string
		value   CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart
		entries int
		want    []byte
	}{
		{"GSM", CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{GSMDescriptionChoice: CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
			Alternative:    CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeGSMDescription,
			GSMDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceGSMDescription{GSMDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceGSMDescriptionGSMDescriptionListEntry{{GSMDescription: GSMDescriptionStruct{BandIndicator: 1, ARFCN: 1, BSIC: 1}}}},
		}}, 1, []byte{0x18, 0x02, 0x08}},
		{"UTRAN FDD", CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{GSMDescriptionChoice: CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
			Alternative:         CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeUTRANFDDDescription,
			UTRANFDDDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANFDDDescription{UTRANFDDDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANFDDDescriptionUTRANFDDDescriptionListEntry{{UTRANFDDDescription: UTRANFDDDescriptionStruct{BandwidthFDD: &fddBandwidth, FDDARFCN: 42, FDDIndic0Group: &UTRANFDDDescriptionStructFDDIndic0Group{FDDIndic0: 1, NROFFDDCELLS: 1, FDDCELLINFORMATIONField: runtime.BitString{Bytes: []byte{0xff, 0xc0}, BitLength: 10}}}}}},
		}}, 1, []byte{0x3b, 0x00, 0xab, 0x0f, 0xfe}},
		{"UTRAN TDD", CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{GSMDescriptionChoice: CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
			Alternative:         CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeUTRANTDDDescription,
			UTRANTDDDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANTDDDescription{UTRANTDDDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANTDDDescriptionUTRANTDDDescriptionListEntry{{UTRANTDDDescription: UTRANTDDDescriptionStruct{BandwidthTDD: &tddBandwidth, TDDARFCN: 42, TDDIndic0Group: &UTRANTDDDescriptionStructTDDIndic0Group{TDDIndic0: 1, NROFTDDCELLS: 1, TDDCELLINFORMATIONField: runtime.BitString{Bytes: []byte{0xff, 0x80}, BitLength: 9}}}}}},
		}}, 1, []byte{0x5a, 0x00, 0xab, 0x0f, 0xfc}},
		{"E-UTRAN", CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{GSMDescriptionChoice: CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
			Alternative:       CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeEUTRANDescription,
			EUTRANDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceEUTRANDescription{EUTRANDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceEUTRANDescriptionEUTRANDescriptionListEntry{{EUTRANDescription: EUTRANDescriptionStruct{EARFCN: 123, MeasurementBandwidth: &eBandwidth, TARGETPCID: &target}}}},
		}}, 1, []byte{0x70, 0x07, 0xbc, 0x41, 0x20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := EncodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(wire, tc.want) {
				t.Fatalf("encoded %x, want pycrate %x", wire, tc.want)
			}
			d, err := DecodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(wire)
			if err != nil {
				t.Fatalf("decode %x: %v", wire, err)
			}
			if d.Value.GSMDescriptionChoice.Alternative != tc.value.GSMDescriptionChoice.Alternative || d.BitsConsumed > len(wire)*8 {
				t.Fatalf("wrong RAT or boundary: %+v", d)
			}
			reencoded, err := EncodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(d.Value)
			if err != nil || !bytes.Equal(reencoded, wire) {
				t.Fatalf("round trip %x -> %x: %v", wire, reencoded, err)
			}
			t.Logf("%s: %x", tc.name, wire)
		})
	}
}

func TestCellSelectionAfterReleaseMinimumValueLength(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.1e: four octets for the whole
	// type 4 IE means at least two octets in this value-part API.
	if _, err := DecodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart([]byte{0}); err == nil {
		t.Fatal("accepted a one-octet value part")
	}
	value := CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{GSMDescriptionChoice: CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
		Alternative:    CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeGSMDescription,
		GSMDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceGSMDescription{},
	}}
	encoded, err := EncodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(value)
	if err != nil || !bytes.Equal(encoded, []byte{0, 0}) {
		t.Fatalf("fresh minimum value part = %x, %v; want 0000", encoded, err)
	}
	if _, err := DecodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart([]byte{0, 0}); err != nil {
		t.Fatalf("rejected a two-octet value part: %v", err)
	}
}

func FuzzCellSelectionAfterRelease(f *testing.F) {
	f.Add([]byte{0x10, 0, 0})
	f.Fuzz(func(t *testing.T, wire []byte) {
		d, err := DecodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(wire)
		if err != nil {
			return
		}
		encoded, err := EncodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("round trip %x -> %x: %v", wire, encoded, err)
		}
	})
}

func TestCellInformationWidths(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.1e delegates p(n) and q(n)
	// to §9.1.54 tables 9.1.54.1a/b, including reserved counts.
	for _, tc := range []struct {
		count    uint8
		fdd, tdd int
	}{
		{0, 0, 0}, {1, 10, 9}, {16, 122, 106},
		{17, 0, 111}, {20, 0, 126}, {21, 0, 0}, {31, 0, 0},
	} {
		for _, fdd := range []bool{true, false} {
			value := CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart{}
			want := tc.tdd
			if fdd {
				want = tc.fdd
				value.GSMDescriptionChoice = CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
					Alternative:         CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeUTRANFDDDescription,
					UTRANFDDDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANFDDDescription{UTRANFDDDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANFDDDescriptionUTRANFDDDescriptionListEntry{{UTRANFDDDescription: UTRANFDDDescriptionStruct{FDDIndic0Group: &UTRANFDDDescriptionStructFDDIndic0Group{NROFFDDCELLS: tc.count, FDDCELLINFORMATIONField: runtime.BitString{Bytes: make([]byte, (want+7)/8), BitLength: want}}}}}},
				}
			} else {
				value.GSMDescriptionChoice = CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoice{
					Alternative:         CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceAlternativeUTRANTDDDescription,
					UTRANTDDDescription: &CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANTDDDescription{UTRANTDDDescriptionList: []CellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePartGSMDescriptionChoiceUTRANTDDDescriptionUTRANTDDDescriptionListEntry{{UTRANTDDDescription: UTRANTDDDescriptionStruct{TDDIndic0Group: &UTRANTDDDescriptionStructTDDIndic0Group{NROFTDDCELLS: tc.count, TDDCELLINFORMATIONField: runtime.BitString{Bytes: make([]byte, (want+7)/8), BitLength: want}}}}}},
				}
			}
			wire, err := EncodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(value)
			if err != nil {
				t.Fatalf("count %d FDD %t: %v", tc.count, fdd, err)
			}
			d, err := DecodeCellSelectionIndicatorAfterReleaseOfAllTCHAndSDCCHValuePart(wire)
			if err != nil {
				t.Fatalf("count %d FDD %t decode %x: %v", tc.count, fdd, wire, err)
			}
			got := 0
			if fdd {
				got = d.Value.GSMDescriptionChoice.UTRANFDDDescription.UTRANFDDDescriptionList[0].UTRANFDDDescription.FDDIndic0Group.FDDCELLINFORMATIONField.BitLength
			} else {
				got = d.Value.GSMDescriptionChoice.UTRANTDDDescription.UTRANTDDDescriptionList[0].UTRANTDDDescription.TDDIndic0Group.TDDCELLINFORMATIONField.BitLength
			}
			if got != want {
				t.Fatalf("count %d FDD %t width = %d, want %d", tc.count, fdd, got, want)
			}
		}
	}
}
