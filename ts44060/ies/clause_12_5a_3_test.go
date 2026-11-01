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

func qualityPack(s string) []byte {
	wire := make([]byte, (len(s)+7)/8)
	for i, b := range s {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	return wire
}

// TS 44.060 V19.0.0 §12.5a.3 pp22571–22618:
// eight optional two-bit-modulation/four-bit-mean pairs and eight I_LEVELs.
func timeslotType2Vector(outer, mask, mod int) string {
	var s strings.Builder
	if outer&1 == 0 {
		s.WriteByte('0')
	} else {
		s.WriteByte('1')
		for i := range 8 {
			if mask&(1<<i) == 0 {
				s.WriteByte('0')
			} else {
				fmt.Fprintf(&s, "1%02b%04b", mod, (i*3+mod)&15)
			}
		}
	}
	if outer&2 == 0 {
		s.WriteByte('0')
	} else {
		s.WriteByte('1')
		for i := range 8 {
			if mask&(1<<(7-i)) == 0 {
				s.WriteByte('0')
			} else {
				fmt.Fprintf(&s, "1%04b", (15-i*2+mod)&15)
			}
		}
	}
	return s.String()
}

func checkTimeslotType2(t *testing.T, v EGPRSTimeslotLinkQualityMeasurementsType2IE, outer, mask, mod int) {
	t.Helper()
	// Only §12.5a.3's helpers receive clause-qualified Go names.
	bep := v.BEPMEASUREMENTS
	interference := v.INTERFERENCEMEASUREMENTS
	if reflect.TypeOf(bep) != reflect.TypeFor[*Clause12_5a_3BEPMeasurementReportStruct]() || reflect.TypeOf(interference) != reflect.TypeFor[*Clause12_5a_3InterferenceMeasurementReportStruct]() {
		t.Fatal("wrong clause-owned helper types")
	}
	if (bep != nil) != (outer&1 != 0) || (interference != nil) != (outer&2 != 0) {
		t.Fatal("wrong outer presence")
	}
	if bep != nil {
		fields := reflect.ValueOf(bep).Elem()
		for i := range 8 {
			group := fields.Field(i)
			present := mask&(1<<i) != 0
			if group.IsNil() == present {
				t.Fatalf("BEP TN%d presence", i)
			}
			if present && (group.Elem().Field(0).Uint() != uint64(mod) || group.Elem().Field(1).Uint() != uint64((i*3+mod)&15)) {
				t.Fatalf("BEP TN%d lost typed values", i)
			}
		}
	}
	if interference != nil {
		fields := reflect.ValueOf(interference).Elem()
		for i := range 8 {
			group := fields.Field(i)
			present := mask&(1<<(7-i)) != 0
			if group.IsNil() == present {
				t.Fatalf("I_LEVEL TN%d presence", i)
			}
			if present && group.Elem().Uint() != uint64((15-i*2+mod)&15) {
				t.Fatalf("I_LEVEL TN%d lost typed value", i)
			}
		}
	}
}

func TestTimeslotType2Fields(t *testing.T) {
	for outer := range 4 {
		for mask := range 256 {
			for mod := range 4 {
				s := timeslotType2Vector(outer, mask, mod)
				wire := qualityPack(s)
				d, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.BitsConsumed != len(s) {
					t.Fatalf("consumed %d want %d", d.BitsConsumed, len(s))
				}
				checkTimeslotType2(t, d.Value, outer, mask, mod)
				replay, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(d.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x %v", replay, err)
				}
				fresh := runtime.Canonical(d.Value)
				encoded, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(fresh)
				if err != nil || !bytes.Equal(encoded, wire) {
					t.Fatalf("fresh %x %v", encoded, err)
				}
				canonical, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IECanonical(fresh)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x %v", canonical, err)
				}
				d.Value.BEPMEASUREMENTS = nil
				d.Value.INTERFERENCEMEASUREMENTS = nil
				// Removing optional groups changes the component extent. The received
				// layout remains strict; the edited value gets a fresh layout explicitly.
				if len(s) != 2 {
					if _, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(d.Value); err == nil {
						t.Fatal("changed extent retained a sealed received layout")
					}
				}
				edited, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(runtime.Canonical(d.Value))
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(edited)
				if err != nil || again.BitsConsumed != 2 || again.Value.BEPMEASUREMENTS != nil || again.Value.INTERFERENCEMEASUREMENTS != nil {
					t.Fatalf("edit %+v %v", again, err)
				}
			}
		}
	}
}

func TestTimeslotType2BoundsAndOffsets(t *testing.T) {
	for outer := range 4 {
		for _, mask := range []int{0, 1, 128, 255} {
			s := timeslotType2Vector(outer, mask, 3)
			for prefix := range 8 {
				wire := qualityPack(strings.Repeat("1", prefix) + s + "10101")
				r := runtime.NewReader(wire)
				if _, err := r.ReadUint(prefix); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(s))
				if err != nil {
					t.Fatal(err)
				}
				v, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IEFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				checkTimeslotType2(t, v, outer, mask, 3)
				if r.Position() != prefix+len(s) {
					t.Fatal("wrong bounded position")
				}
				if err := r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				got, err := r.ReadUint(5)
				if err != nil || got != 21 {
					t.Fatalf("sentinel %d %v", got, err)
				}
			}
			for cut := range len(s) {
				r := runtime.NewReader(qualityPack(s))
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				_, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IEFrom(r)
				if err == nil {
					t.Fatalf("accepted %d/%d bits", cut, len(s))
				}
				var located *runtime.OperationError
				if !errors.As(err, &located) || located.Location.Source.Standard != "TS 44.060" || located.Location.Source.Clause != "12.5a.3" {
					t.Fatalf("clause identity lost: %+v", err)
				}
				if r.Position() > cut {
					t.Fatal("escaped bound")
				}
			}
		}
	}
	for _, size := range []int{0, 1, 16, 256, 4096} {
		wire := append(qualityPack(timeslotType2Vector(3, 255, 3)), bytes.Repeat([]byte{0xa5}, size)...)
		d, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(wire)
		if err != nil || d.BitsConsumed != 98 {
			t.Fatalf("tail %d consumed %d: %v", size, d.BitsConsumed, err)
		}
		got, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("tail replay %d %v", size, err)
		}
	}
	wire := qualityPack(timeslotType2Vector(3, 255, 3))
	for group := range 8 {
		for kind, invalid := range []uint64{4, 16, 16} {
			d, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(wire)
			if err != nil {
				t.Fatal(err)
			}
			v := runtime.Canonical(d.Value)
			if kind < 2 {
				reflect.ValueOf(v.BEPMEASUREMENTS).Elem().Field(group).Elem().Field(kind).SetUint(invalid)
			} else {
				reflect.ValueOf(v.INTERFERENCEMEASUREMENTS).Elem().Field(group).Elem().SetUint(invalid)
			}
			if _, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(v); err == nil {
				t.Fatalf("accepted group %d field %d overflow", group, kind)
			}
		}
	}
}

func FuzzTimeslotType2Measurements(f *testing.F) {
	for outer := range 4 {
		for _, mask := range []int{0, 1, 128, 255} {
			f.Add(qualityPack(timeslotType2Vector(outer, mask, 3)))
		}
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		d, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(wire)
		if err != nil {
			return
		}
		got, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}

func TestTimeslotType2ScalarEdits(t *testing.T) {
	for outer := 1; outer < 4; outer++ {
		for _, mask := range []int{1, 128, 255} {
			for mod := range 4 {
				wire := qualityPack(timeslotType2Vector(outer, mask, mod))
				d, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.Value.BEPMEASUREMENTS != nil {
					fields := reflect.ValueOf(d.Value.BEPMEASUREMENTS).Elem()
					for i := range 8 {
						f := fields.Field(i)
						if !f.IsNil() {
							v := f.Elem().Field(1)
							v.SetUint(v.Uint() ^ 15)
							break
						}
					}
				} else {
					fields := reflect.ValueOf(d.Value.INTERFERENCEMEASUREMENTS).Elem()
					for i := range 8 {
						f := fields.Field(i)
						if !f.IsNil() {
							v := f.Elem()
							v.SetUint(v.Uint() ^ 15)
							break
						}
					}
				}
				b, err := EncodeEGPRSTimeslotLinkQualityMeasurementsType2IE(d.Value)
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeEGPRSTimeslotLinkQualityMeasurementsType2IE(b)
				if err != nil || !runtime.SemanticallyEqual(d.Value, again.Value) || bytes.Equal(b, wire) {
					t.Fatalf("scalar edit %+v %v", again, err)
				}
			}
		}
	}
}
