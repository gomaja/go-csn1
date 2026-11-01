package ies

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.5a.1 pp22531–22541 delegates to
// §§12.5a.2 and 12.5a.3 with a six-bit C_VALUE between them.
func channelType2Vector(mask, outer, high, cvalue int) string {
	var s strings.Builder
	for i := range 7 {
		if mask&(1<<i) == 0 {
			s.WriteByte('0')
			continue
		}
		mean, cv := i, 0
		if high != 0 {
			mean, cv = 31-i, 7
		}
		fmt.Fprintf(&s, "1%05b%03b", mean, cv)
	}
	fmt.Fprintf(&s, "%06b", cvalue)
	s.WriteString(timeslotType2Vector(outer, (mask*73)&255, high*3))
	return s.String()
}

func TestChannelQualityType2Fields(t *testing.T) {
	for mask := range 128 {
		for outer := range 4 {
			for high := range 2 {
				cvalue := high * 63
				s := channelType2Vector(mask, outer, high, cvalue)
				wire := qualityPack(s)
				d, err := DecodeEGPRSChannelQualityReportType2IE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.BitsConsumed != len(s) || d.Value.CVALUE != uint8(cvalue) {
					t.Fatalf("wrong wrapper %+v", d)
				}
				component := d.Value.EGPRSBEPLinkQualityMeasurementsType2
				fields := reflect.ValueOf(component)
				if fields.Type() != reflect.TypeFor[EGPRSBEPLinkQualityMeasurementsType2IE]() {
					t.Fatal("wrong BEP component type")
				}
				for i := range 7 {
					group := fields.Field(i)
					present := mask&(1<<i) != 0
					if group.IsNil() == present {
						t.Fatalf("BEP group %d presence", i)
					}
					mean, cv := i, 0
					if high != 0 {
						mean, cv = 31-i, 7
					}
					if present && (group.Elem().Field(0).Uint() != uint64(mean) || group.Elem().Field(1).Uint() != uint64(cv)) {
						t.Fatalf("BEP group %d values", i)
					}
				}
				checkTimeslotType2(t, d.Value.EGPRSTimeslotLinkQualityMeasurementsType2, outer, (mask*73)&255, high*3)
				got, err := EncodeEGPRSChannelQualityReportType2IE(d.Value)
				if err != nil || !bytes.Equal(got, wire) {
					t.Fatalf("replay %x %v", got, err)
				}
				fresh := runtime.Canonical(d.Value)
				got, err = EncodeEGPRSChannelQualityReportType2IE(fresh)
				if err != nil || !bytes.Equal(got, wire) {
					t.Fatalf("fresh %x %v", got, err)
				}
				got, err = EncodeEGPRSChannelQualityReportType2IECanonical(fresh)
				if err != nil || !bytes.Equal(got, wire) {
					t.Fatalf("canonical %x %v", got, err)
				}
				d.Value.CVALUE ^= 63
				got, err = EncodeEGPRSChannelQualityReportType2IE(d.Value)
				want := qualityPack(channelType2Vector(mask, outer, high, cvalue^63))
				if err != nil || !bytes.Equal(got, want) {
					t.Fatalf("edit %x want %x: %v", got, want, err)
				}
			}
		}
	}
	for cvalue := range 64 {
		wire := qualityPack(channelType2Vector(0, 0, 0, cvalue))
		d, err := DecodeEGPRSChannelQualityReportType2IE(wire)
		if err != nil || d.Value.CVALUE != uint8(cvalue) {
			t.Fatalf("C_VALUE %d %+v %v", cvalue, d, err)
		}
	}
	if _, err := EncodeEGPRSChannelQualityReportType2IE(EGPRSChannelQualityReportType2IE{CVALUE: 64}); err == nil {
		t.Fatal("accepted six-bit overflow")
	}
}

func TestChannelQualityType2Bounds(t *testing.T) {
	for _, mask := range []int{0, 1, 127} {
		for outer := range 4 {
			s := channelType2Vector(mask, outer, 1, 63)
			wire := qualityPack(s)
			for cut := range len(s) {
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				_, err := DecodeEGPRSChannelQualityReportType2IEFrom(r)
				if err == nil {
					t.Fatalf("accepted truncated wrapper")
				}
				var located *runtime.OperationError
				if !errors.As(err, &located) || located.Location.Source.Standard != "TS 44.060" || (located.Location.Source.Clause != "12.5a.1" && located.Location.Source.Clause != "12.5a.2" && located.Location.Source.Clause != "12.5a.3") {
					t.Fatalf("component clause identity lost: %+v", err)
				}
				if r.Position() > cut {
					t.Fatal("escaped bound")
				}
			}
			r := runtime.NewReader(qualityPack(s + "10101"))
			old, err := r.PushLimit(len(s))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := DecodeEGPRSChannelQualityReportType2IEFrom(r); err != nil {
				t.Fatal(err)
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			got, err := r.ReadUint(5)
			if err != nil || got != 21 {
				t.Fatalf("sentinel %d %v", got, err)
			}
		}
	}
}

func FuzzChannelQualityType2(f *testing.F) {
	for outer := range 4 {
		for _, mask := range []int{0, 1, 127} {
			f.Add(qualityPack(channelType2Vector(mask, outer, 1, 63)))
		}
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		d, err := DecodeEGPRSChannelQualityReportType2IE(wire)
		if err != nil {
			return
		}
		got, err := EncodeEGPRSChannelQualityReportType2IE(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}
