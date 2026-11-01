package ies

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.75 pp25996–26001:
// each optional BEP pair has 1+5+3 bits; C_VALUE always adds six bits.
func ecReportVector(mask, cvalue, high int) ([]byte, int) {
	var s strings.Builder
	for i := range 2 {
		if mask&(1<<i) == 0 {
			s.WriteByte('0')
			continue
		}
		mean, cv := i, 0
		if high != 0 {
			mean, cv = 31-i, 7-i
		}
		fmt.Fprintf(&s, "1%05b%03b", mean, cv)
	}
	fmt.Fprintf(&s, "%06b", cvalue)
	n := s.Len()
	wire := make([]byte, (n+7)/8)
	for i, b := range s.String() {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	return wire, n
}

func TestECChannelQualityReportFields(t *testing.T) {
	for mask := range 4 {
		for cvalue := range 64 {
			for high := range 2 {
				wire, n := ecReportVector(mask, cvalue, high)
				d, err := DecodeECChannelQualityReportIE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.BitsConsumed != n || d.Value.CVALUE != uint8(cvalue) {
					t.Fatalf("mask %d CV %d: %+v", mask, cvalue, d)
				}
				fresh := ECChannelQualityReportIE{CVALUE: uint8(cvalue)}
				if mask&1 != 0 {
					mean, cv := uint8(0), uint8(0)
					if high != 0 {
						mean, cv = 31, 7
					}
					fresh.GMSKMEANBEPGroup = &ECChannelQualityReportIEGMSKMEANBEPGroup{GMSKMEANBEP: mean, GMSKCVBEP: cv}
					if d.Value.GMSKMEANBEPGroup == nil || *d.Value.GMSKMEANBEPGroup != *fresh.GMSKMEANBEPGroup {
						t.Fatal("lost GMSK fields")
					}
				} else if d.Value.GMSKMEANBEPGroup != nil {
					t.Fatal("unexpected GMSK fields")
				}
				if mask&2 != 0 {
					mean, cv := uint8(1), uint8(0)
					if high != 0 {
						mean, cv = 30, 6
					}
					fresh.N8PSKMEANBEPGroup = &ECChannelQualityReportIEN8PSKMEANBEPGroup{N8PSKMEANBEP: mean, N8PSKCVBEP: cv}
					if d.Value.N8PSKMEANBEPGroup == nil || *d.Value.N8PSKMEANBEPGroup != *fresh.N8PSKMEANBEPGroup {
						t.Fatal("lost 8PSK fields")
					}
				} else if d.Value.N8PSKMEANBEPGroup != nil {
					t.Fatal("unexpected 8PSK fields")
				}
				replay, err := EncodeECChannelQualityReportIE(d.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x %v", replay, err)
				}
				encoded, err := EncodeECChannelQualityReportIE(fresh)
				if err != nil || !bytes.Equal(encoded, wire) {
					t.Fatalf("fresh %x %v", encoded, err)
				}
				canonical, err := EncodeECChannelQualityReportIECanonical(fresh)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x %v", canonical, err)
				}
				d.Value.CVALUE ^= 63
				edited, err := EncodeECChannelQualityReportIE(d.Value)
				want, _ := ecReportVector(mask, cvalue^63, high)
				if err != nil || !bytes.Equal(edited, want) {
					t.Fatalf("edit %x want %x: %v", edited, want, err)
				}
			}
		}
	}
}

func TestECChannelQualityReportExtent(t *testing.T) {
	for mask := range 4 {
		wire, n := ecReportVector(mask, 63, 1)
		r := runtime.NewReader(append(append([]byte(nil), wire...), 0xa5))
		old, err := r.PushLimit(n)
		if err != nil {
			t.Fatal(err)
		}
		v, err := DecodeECChannelQualityReportIEFrom(r)
		if err != nil || r.Position() != n || v.CVALUE != 63 {
			t.Fatalf("extent %d %+v %v", r.Position(), v, err)
		}
		if err := r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(8)
		if err != nil || sentinel != 0xa5 {
			t.Fatalf("sentinel %x %v", sentinel, err)
		}
		for cut := range n {
			short := runtime.NewReader(wire)
			if _, err := short.PushLimit(cut); err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeECChannelQualityReportIEFrom(short); err == nil {
				t.Fatalf("accepted %d/%d bits", cut, n)
			}
			if short.Position() > cut {
				t.Fatal("escaped bound")
			}
		}
		if _, err := DecodeECChannelQualityReportIE(wire[:len(wire)-1]); err == nil {
			t.Fatal("accepted truncated octet")
		}
		withTail := append(append([]byte(nil), wire...), 0x7e, 0xa5)
		d, err := DecodeECChannelQualityReportIE(withTail)
		if err != nil {
			t.Fatal(err)
		}
		got, err := EncodeECChannelQualityReportIE(d.Value)
		if err != nil || !bytes.Equal(got, withTail) {
			t.Fatalf("tail replay %x %v", got, err)
		}
	}
	for _, v := range []ECChannelQualityReportIE{
		{CVALUE: 64},
		{GMSKMEANBEPGroup: &ECChannelQualityReportIEGMSKMEANBEPGroup{GMSKMEANBEP: 32}},
		{GMSKMEANBEPGroup: &ECChannelQualityReportIEGMSKMEANBEPGroup{GMSKCVBEP: 8}},
		{N8PSKMEANBEPGroup: &ECChannelQualityReportIEN8PSKMEANBEPGroup{N8PSKMEANBEP: 32}},
		{N8PSKMEANBEPGroup: &ECChannelQualityReportIEN8PSKMEANBEPGroup{N8PSKCVBEP: 8}},
	} {
		if _, err := EncodeECChannelQualityReportIE(v); err == nil {
			t.Fatal("accepted field overflow")
		}
		if _, err := EncodeECChannelQualityReportIECanonical(v); err == nil {
			t.Fatal("canonical accepted field overflow")
		}
	}
}

func TestECChannelQualityReportScalarRanges(t *testing.T) {
	// TS 44.060 V19.0.0 table 12.75.1 pp25997–26001:
	// exercise every code of each five-bit mean and three-bit CV separately.
	for modulation := range 2 {
		for mean := range 32 {
			for cv := range 8 {
				v := ECChannelQualityReportIE{CVALUE: 37}
				var bits string
				if modulation == 0 {
					v.GMSKMEANBEPGroup = &ECChannelQualityReportIEGMSKMEANBEPGroup{GMSKMEANBEP: uint8(mean), GMSKCVBEP: uint8(cv)}
					bits = fmt.Sprintf("1%05b%03b0%06b", mean, cv, 37)
				} else {
					v.N8PSKMEANBEPGroup = &ECChannelQualityReportIEN8PSKMEANBEPGroup{N8PSKMEANBEP: uint8(mean), N8PSKCVBEP: uint8(cv)}
					bits = fmt.Sprintf("01%05b%03b%06b", mean, cv, 37)
				}
				want := make([]byte, 2)
				for i, bit := range bits {
					if bit == '1' {
						want[i/8] |= 1 << uint(7-i%8)
					}
				}
				wire, err := EncodeECChannelQualityReportIE(v)
				if err != nil || !bytes.Equal(wire, want) {
					t.Fatalf("modulation %d mean %d CV %d: %x want %x: %v", modulation, mean, cv, wire, want, err)
				}
				d, err := DecodeECChannelQualityReportIE(want)
				if err != nil || d.BitsConsumed != 16 || !runtime.SemanticallyEqual(d.Value, v) {
					t.Fatalf("modulation %d mean %d CV %d: %+v %v", modulation, mean, cv, d, err)
				}
			}
		}
	}
}

func FuzzECChannelQualityReport(f *testing.F) {
	for mask := range 4 {
		wire, _ := ecReportVector(mask, 63, 1)
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		d, err := DecodeECChannelQualityReportIE(wire)
		if err != nil {
			return
		}
		got, err := EncodeECChannelQualityReportIE(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}
