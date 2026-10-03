package restoctets

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestSI6EqualityFieldsHoldLHMeaning(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.35a: L means DTM unsupported
	// and selects the 1800 band even when the raw padding bit is one.
	wire := bytes.Repeat([]byte{0x2b}, 7)
	d, err := DecodeSI6RestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	choice := d.Value.DTMSupportChoice
	if choice.DTMSupport == nil || *choice.DTMSupport != 0 {
		t.Fatalf("DTM L choice has value %+v", choice)
	}
	band := d.Value.BandIndicatorClause105235a.BANDINDICATORChoice
	if band.BANDINDICATOR == nil || *band.BANDINDICATOR != 0 {
		t.Fatalf("band L choice has value %+v", band)
	}
	got, err := EncodeSI6RestOctets(d.Value)
	if err != nil || !bytes.Equal(got, wire) {
		t.Fatalf("round trip=%x err=%v", got, err)
	}
}

func TestStandaloneEqualityFieldsAtInvertedPaddingBit(t *testing.T) {
	// TS 24.007 V20.0.0 Annex B §B.1.2.2 fixes the L pattern at
	// 0x2b. At bit position 2, transmitted L is one and H is zero.
	tests := []struct {
		name   string
		decode func(*runtime.Reader, uint8) (uint8, error)
	}{
		{"SI1 band", func(r *runtime.Reader, want uint8) (uint8, error) {
			v, err := DecodeBandIndicatorClause105232From(r)
			if err != nil {
				return 0, err
			}
			if want == 0 && v.BANDINDICATORChoice.BANDINDICATOR != nil {
				return *v.BANDINDICATORChoice.BANDINDICATOR, nil
			}
			if want == 1 && v.BANDINDICATORChoice.AltUnlabeled != nil {
				return *v.BANDINDICATORChoice.AltUnlabeled, nil
			}
			return 0, fmt.Errorf("wrong band alternative")
		}},
		{"SI6 band", func(r *runtime.Reader, want uint8) (uint8, error) {
			v, err := DecodeBandIndicatorClause105235aFrom(r)
			if err != nil {
				return 0, err
			}
			if want == 0 && v.BANDINDICATORChoice.BANDINDICATOR != nil {
				return *v.BANDINDICATORChoice.BANDINDICATOR, nil
			}
			if want == 1 && v.BANDINDICATORChoice.AltUnlabeled != nil {
				return *v.BANDINDICATORChoice.AltUnlabeled, nil
			}
			return 0, fmt.Errorf("wrong band alternative")
		}},
		{"emergency", func(r *runtime.Reader, want uint8) (uint8, error) {
			v, err := DecodeEmergencyIndFrom(r)
			if err != nil {
				return 0, err
			}
			if want == 0 && v.EmergencyIndChoice.EmergencyInd != nil {
				return *v.EmergencyIndChoice.EmergencyInd, nil
			}
			if want == 1 && v.EmergencyIndChoice.AltUnlabeled != nil {
				return *v.EmergencyIndChoice.AltUnlabeled, nil
			}
			return 0, fmt.Errorf("wrong emergency alternative")
		}},
	}
	for _, tc := range tests {
		for _, sample := range []struct {
			wire byte
			want uint8
		}{{0x2b, 0}, {0x0b, 1}} {
			t.Run(fmt.Sprintf("%s/%02x", tc.name, sample.wire), func(t *testing.T) {
				r := runtime.NewReader([]byte{sample.wire})
				if _, err := r.ReadUint(2); err != nil {
					t.Fatal(err)
				}
				got, err := tc.decode(r, sample.want)
				if err != nil || got != sample.want {
					t.Fatalf("L/H value=%d want=%d err=%v", got, sample.want, err)
				}
			})
		}
	}
}
