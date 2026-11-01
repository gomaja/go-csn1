package ies

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §12.66 pp25777–25835; the p25778 delimiter
// correction has an authenticated receipt. No §12.3.3 acknowledgement is part of this IE.
func dlmcBits(n, mask, outer int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%06b", (n+mask)&63)
	if outer == 0 {
		b.WriteByte('0')
	} else {
		b.WriteByte('1')
		for i := range 8 {
			if mask&(1<<(i%4)) == 0 {
				b.WriteByte('0')
			} else {
				fmt.Fprintf(&b, "1%04b", (i+mask)&15)
			}
		}
	}
	for i := range n {
		b.WriteByte('1')
		for j := range 4 {
			if mask&(1<<j) == 0 {
				b.WriteByte('0')
			} else {
				fmt.Fprintf(&b, "1%05b%03b", (i+j)&31, (i+mask)&7)
			}
		}
		if i%2 == 0 {
			b.WriteByte('0')
		} else {
			b.WriteByte('1')
			for j := range 8 {
				if mask&(1<<(j%4)) == 0 {
					b.WriteByte('0')
				} else {
					fmt.Fprintf(&b, "1%02b%04b", i%4, (i+j)&15)
				}
			}
		}
	}
	b.WriteByte('0')
	return b.String()
}
func checkDLMC(t *testing.T, v DLMCChannelQualityReportIE, n, mask, outer int) {
	t.Helper()
	if v.CVALUE != uint8((n+mask)&63) || len(v.DLMCBEPLinkQualityMeasurementsGroupList) != n || (v.DLMCInterferenceMeasurements != nil) != (outer == 1) {
		t.Fatal("root values lost")
	}
	if x := v.DLMCInterferenceMeasurements; x != nil {
		fields := reflect.ValueOf(x).Elem()
		for j := range 8 {
			f := fields.Field(j)
			present := mask&(1<<(j%4)) != 0
			if f.IsNil() == present {
				t.Fatal("interference presence lost")
			}
			if present && f.Elem().Uint() != uint64((j+mask)&15) {
				t.Fatal("interference value lost")
			}
		}
	}
	for i, x := range v.DLMCBEPLinkQualityMeasurementsGroupList {
		fields := reflect.ValueOf(x.DLMCBEPLinkQualityMeasurements)
		for j := range 4 {
			f := fields.Field(j)
			present := mask&(1<<j) != 0
			if f.IsNil() == present {
				t.Fatal("BEP presence lost")
			}
			if present && (f.Elem().Field(0).Uint() != uint64((i+j)&31) || f.Elem().Field(1).Uint() != uint64((i+mask)&7)) {
				t.Fatal("BEP values lost")
			}
		}
		if (x.DLMCBEPMEASUREMENTS != nil) != (i%2 == 1) {
			t.Fatal("timeslot presence lost")
		}
		if x.DLMCBEPMEASUREMENTS != nil {
			fields := reflect.ValueOf(x.DLMCBEPMEASUREMENTS).Elem()
			for j := range 8 {
				f := fields.Field(j)
				present := mask&(1<<(j%4)) != 0
				if f.IsNil() == present {
					t.Fatal("timeslot presence lost")
				}
				if present && (f.Elem().Field(0).Uint() != uint64(i%4) || f.Elem().Field(1).Uint() != uint64((i+j)&15)) {
					t.Fatal("timeslot values lost")
				}
			}
		}
	}
}
func TestDLMCFields(t *testing.T) {
	for _, n := range []int{0, 1, 2, 8, 32} {
		for mask := range 16 {
			for outer := range 2 {
				bits := dlmcBits(n, mask, outer)
				wire := dlmcPack(bits)
				d, err := DecodeDLMCChannelQualityReportIE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.BitsConsumed != len(bits) {
					t.Fatalf("extent %d/%d", d.BitsConsumed, len(bits))
				}
				checkDLMC(t, d.Value, n, mask, outer)
				replay, err := EncodeDLMCChannelQualityReportIE(d.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x %v", replay, err)
				}
				fresh, err := EncodeDLMCChannelQualityReportIE(runtime.Canonical(d.Value))
				if err != nil || !bytes.Equal(fresh, wire) {
					t.Fatalf("fresh %x %v", fresh, err)
				}
				canonical, err := EncodeDLMCChannelQualityReportIECanonicalAtLength(d.Value, len(wire))
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x %v", canonical, err)
				}
				d.Value.CVALUE ^= 63
				changed, err := EncodeDLMCChannelQualityReportIE(d.Value)
				if err != nil {
					t.Fatal(err)
				}
				scalar, err := DecodeDLMCChannelQualityReportIE(changed)
				if err != nil || !runtime.SemanticallyEqual(d.Value, scalar.Value) || bytes.Equal(changed, wire) {
					t.Fatalf("scalar edit %+v %v", scalar, err)
				}
				d.Value.DLMCInterferenceMeasurements = nil
				d.Value.DLMCBEPLinkQualityMeasurementsGroupList = nil
				if n != 0 || outer != 0 {
					if _, err := EncodeDLMCChannelQualityReportIE(d.Value); err == nil {
						t.Fatal("changed extent retained a sealed received layout")
					}
				}
				edited, err := EncodeDLMCChannelQualityReportIE(runtime.Canonical(d.Value))
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeDLMCChannelQualityReportIE(edited)
				if err != nil || again.BitsConsumed != 8 || again.Value.CVALUE != uint8(((n+mask)&63)^63) || again.Value.DLMCInterferenceMeasurements != nil || len(again.Value.DLMCBEPLinkQualityMeasurementsGroupList) != 0 {
					t.Fatalf("edit %+v %v", again, err)
				}
			}
		}
	}
}
func TestDLMCBounds(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		for _, mask := range []int{0, 1, 15} {
			for outer := range 2 {
				bits := dlmcBits(n, mask, outer)
				for prefix := range 8 {
					r := runtime.NewReader(dlmcPack(strings.Repeat("1", prefix) + bits + "10101"))
					if _, err := r.ReadUint(prefix); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(len(bits))
					if err != nil {
						t.Fatal(err)
					}
					v, err := DecodeDLMCChannelQualityReportIEFrom(r)
					if err != nil {
						t.Fatal(err)
					}
					checkDLMC(t, v, n, mask, outer)
					if r.Position() != prefix+len(bits) {
						t.Fatal("wrong limit")
					}
					if err = r.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					got, err := r.ReadUint(5)
					if err != nil || got != 21 {
						t.Fatalf("sentinel %d %v", got, err)
					}
				}
				for cut := range len(bits) {
					r := runtime.NewReader(dlmcPack(bits))
					if _, err := r.PushLimit(cut); err != nil {
						t.Fatal(err)
					}
					if _, err := DecodeDLMCChannelQualityReportIEFrom(r); err == nil {
						t.Fatalf("accepted %d/%d", cut, len(bits))
					}
					if r.Position() > cut {
						t.Fatal("escaped truncated bound")
					}
				}
			}
		}
	}
	if _, err := EncodeDLMCChannelQualityReportIE(DLMCChannelQualityReportIE{CVALUE: 64}); err == nil {
		t.Fatal("C_VALUE overflow accepted")
	}
	v := DLMCChannelQualityReportIE{DLMCInterferenceMeasurements: &DLMCInterferenceMeasurementReportStruct{ILEVELTN0: new(uint8)}}
	*v.DLMCInterferenceMeasurements.ILEVELTN0 = 16
	if _, err := EncodeDLMCChannelQualityReportIE(v); err == nil {
		t.Fatal("I_LEVEL overflow accepted")
	}
}
func TestDLMCGrowth(t *testing.T) {
	var previous float64
	for _, n := range []int{8, 16, 32, 64} {
		bits := dlmcBits(n, 15, 1)
		wire := dlmcPack(bits)
		if len(bits) != 48+66*n {
			t.Fatalf("size %d", len(bits))
		}
		allocations := testing.AllocsPerRun(3, func() {
			d, err := DecodeDLMCChannelQualityReportIE(wire)
			if err != nil {
				panic(err)
			}
			if len(d.Value.DLMCBEPLinkQualityMeasurementsGroupList) != n {
				panic("count")
			}
		})
		if !raceEnabled && previous > 0 && allocations > 2*previous+8 {
			t.Fatalf("superlinear allocations %g -> %g", previous, allocations)
		}
		previous = allocations
	}
}
func FuzzDLMCChannelQuality(f *testing.F) {
	for _, n := range []int{0, 1, 2} {
		f.Add(dlmcPack(dlmcBits(n, 15, 1)))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		d, err := DecodeDLMCChannelQualityReportIE(data)
		if err != nil {
			return
		}
		b, err := EncodeDLMCChannelQualityReportIE(d.Value)
		if err != nil || !bytes.Equal(b, data) {
			t.Fatalf("replay %x %x %v", data, b, err)
		}
	})
}

func TestDLMCCValueAndTimeslotCodes(t *testing.T) {
	for c := range 64 {
		bits := fmt.Sprintf("%06b00", c)
		wire := dlmcPack(bits)
		d, err := DecodeDLMCChannelQualityReportIE(wire)
		if err != nil || d.BitsConsumed != 8 || d.Value.CVALUE != uint8(c) {
			t.Fatalf("C_VALUE %d: %+v %v", c, d, err)
		}
		fresh, err := EncodeDLMCChannelQualityReportIECanonicalAtLength(d.Value, len(wire))
		if err != nil || !bytes.Equal(fresh, wire) {
			t.Fatalf("C_VALUE %d: %x %v", c, fresh, err)
		}
	}
	for mask := range 256 {
		for modulation := range 4 {
			bits := dlmcTimeslotBits(mask, modulation)
			wire := dlmcPack(bits)
			d, err := DecodeDLMCBEPMeasurementsStruct(wire)
			if err != nil || d.BitsConsumed != len(bits) {
				t.Fatalf("timeslot extent %+v %v", d, err)
			}
			fields := reflect.ValueOf(d.Value)
			for i := range 8 {
				x := fields.Field(i)
				present := mask&(1<<i) != 0
				if x.IsNil() == present {
					t.Fatal("timeslot presence")
				}
				if present && (x.Elem().Field(0).Uint() != uint64(modulation) || x.Elem().Field(1).Uint() != uint64((i*3+modulation)&15)) {
					t.Fatal("timeslot codes")
				}
			}
			b, err := EncodeDLMCBEPMeasurementsStructCanonical(d.Value)
			if err != nil || !bytes.Equal(b, wire) {
				t.Fatalf("timeslot encoding %x %v", b, err)
			}
		}
	}
	for group := range 4 {
		for field, invalid := range []uint64{32, 8} {
			d, err := DecodeDLMCChannelQualityReportIE(dlmcPack(dlmcBits(2, 15, 1)))
			if err != nil {
				t.Fatal(err)
			}
			v := runtime.Canonical(d.Value)
			reflect.ValueOf(&v.DLMCBEPLinkQualityMeasurementsGroupList[0].DLMCBEPLinkQualityMeasurements).Elem().Field(group).Elem().Field(field).SetUint(invalid)
			if _, err := EncodeDLMCChannelQualityReportIE(v); err == nil {
				t.Fatal("BEP overflow")
			}
		}
	}
	for group := range 8 {
		for field, invalid := range []uint64{4, 16} {
			d, err := DecodeDLMCChannelQualityReportIE(dlmcPack(dlmcBits(2, 15, 1)))
			if err != nil {
				t.Fatal(err)
			}
			v := runtime.Canonical(d.Value)
			reflect.ValueOf(v.DLMCBEPLinkQualityMeasurementsGroupList[1].DLMCBEPMEASUREMENTS).Elem().Field(group).Elem().Field(field).SetUint(invalid)
			if _, err := EncodeDLMCChannelQualityReportIE(v); err == nil {
				t.Fatal("timeslot overflow")
			}
		}
	}
}

func dlmcPack(s string) []byte {
	wire := make([]byte, (len(s)+7)/8)
	for i, b := range s {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	return wire
}

func dlmcTimeslotBits(mask, modulation int) string {
	var b strings.Builder
	for i := range 8 {
		if mask&(1<<i) == 0 {
			b.WriteByte('0')
		} else {
			fmt.Fprintf(&b, "1%02b%04b", modulation, (i*3+modulation)&15)
		}
	}
	return b.String()
}

func TestDLMCCanonicalExtentRequired(t *testing.T) {
	_, err := EncodeDLMCChannelQualityReportIECanonical(DLMCChannelQualityReportIE{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("missing containing extent: %v", err)
	}
}
