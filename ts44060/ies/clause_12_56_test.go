package ies

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.56 pp25436–25444: presence, one-bit control,
// first frequency index, then 1-prefixed indices terminated by 0.
func measurementControlBits(outer, count, high int) string {
	var b strings.Builder
	for j, width := range []int{3, 5} {
		if outer&(1<<j) == 0 {
			b.WriteByte('0')
			continue
		}
		first := 0
		if high != 0 {
			first = (1 << width) - 1
		}
		fmt.Fprintf(&b, "1%d%0*b", high, width, first)
		for i := range count {
			fmt.Fprintf(&b, "1%0*b", width, (i*3+high)&((1<<width)-1))
		}
		b.WriteByte('0')
	}
	return b.String()
}

func checkMeasurementControl(t *testing.T, v MeasurementControlParametersDescription, outer, count, high int) {
	t.Helper()
	if (v.MeasurementControlEUTRANGroup != nil) != (outer&1 != 0) || (v.MeasurementControlUTRANGroup != nil) != (outer&2 != 0) {
		t.Fatal("presence lost")
	}
	if g := v.MeasurementControlEUTRANGroup; g != nil {
		first := uint8(0)
		if high != 0 {
			first = 7
		}
		if g.MeasurementControlEUTRAN != uint8(high) || g.EUTRANFREQUENCYINDEX != first || len(g.EUTRANFREQUENCYINDEXList) != count {
			t.Fatal("E-UTRAN values lost")
		}
		for i, x := range g.EUTRANFREQUENCYINDEXList {
			if x.EUTRANFREQUENCYINDEX != uint8((i*3+high)&7) {
				t.Fatal("E-UTRAN list value lost")
			}
		}
	}
	if g := v.MeasurementControlUTRANGroup; g != nil {
		first := uint8(0)
		if high != 0 {
			first = 31
		}
		if g.MeasurementControlUTRAN != uint8(high) || g.UTRANFREQUENCYINDEX != first || len(g.UTRANFREQUENCYINDEXList) != count {
			t.Fatal("UTRAN values lost")
		}
		for i, x := range g.UTRANFREQUENCYINDEXList {
			if x.UTRANFREQUENCYINDEX != uint8((i*3+high)&31) {
				t.Fatal("UTRAN list value lost")
			}
		}
	}
}

func TestMeasurementControlFields(t *testing.T) {
	for outer := range 4 {
		for _, n := range []int{0, 1, 2, 8, 32} {
			for high := range 2 {
				bits := measurementControlBits(outer, n, high)
				wire := measurementControlPack(bits)
				d, err := DecodeMeasurementControlParametersDescription(wire)
				if err != nil {
					t.Fatal(err)
				}
				if d.BitsConsumed != len(bits) {
					t.Fatalf("extent %d want %d", d.BitsConsumed, len(bits))
				}
				checkMeasurementControl(t, d.Value, outer, n, high)
				replay, err := EncodeMeasurementControlParametersDescription(d.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x %v", replay, err)
				}
				fresh := runtime.Canonical(d.Value)
				encoded, err := EncodeMeasurementControlParametersDescription(fresh)
				if err != nil || !bytes.Equal(encoded, wire) {
					t.Fatalf("fresh %x %v", encoded, err)
				}
				canonical, err := EncodeMeasurementControlParametersDescriptionCanonicalAtLength(d.Value, len(wire))
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x %v", canonical, err)
				}
				if outer != 0 {
					if g := d.Value.MeasurementControlEUTRANGroup; g != nil {
						g.MeasurementControlEUTRAN ^= 1
					} else {
						d.Value.MeasurementControlUTRANGroup.MeasurementControlUTRAN ^= 1
					}
					changed, err := EncodeMeasurementControlParametersDescription(d.Value)
					if err != nil {
						t.Fatal(err)
					}
					again, err := DecodeMeasurementControlParametersDescription(changed)
					if err != nil || !runtime.SemanticallyEqual(d.Value, again.Value) || bytes.Equal(changed, wire) {
						t.Fatalf("scalar edit %+v %v", again, err)
					}
				}
				d.Value.MeasurementControlEUTRANGroup = nil
				d.Value.MeasurementControlUTRANGroup = nil
				// A new presence layout has a new semantic extent. Preserve the strict
				// received boundary and explicitly create a fresh edited value.
				if outer != 0 {
					if _, err := EncodeMeasurementControlParametersDescription(d.Value); err == nil {
						t.Fatal("changed extent retained a sealed received layout")
					}
				}
				edited, err := EncodeMeasurementControlParametersDescription(runtime.Canonical(d.Value))
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeMeasurementControlParametersDescription(edited)
				if err != nil || again.BitsConsumed != 2 || again.Value.MeasurementControlEUTRANGroup != nil || again.Value.MeasurementControlUTRANGroup != nil {
					t.Fatalf("edit %+v %v", again, err)
				}
			}
		}
	}
}

func TestMeasurementControlBounds(t *testing.T) {
	for outer := range 4 {
		for _, n := range []int{0, 1, 8} {
			bits := measurementControlBits(outer, n, 1)
			for prefix := range 8 {
				r := runtime.NewReader(measurementControlPack(strings.Repeat("1", prefix) + bits + "10101"))
				if _, err := r.ReadUint(prefix); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(bits))
				if err != nil {
					t.Fatal(err)
				}
				v, err := DecodeMeasurementControlParametersDescriptionFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				checkMeasurementControl(t, v, outer, n, 1)
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
				r := runtime.NewReader(measurementControlPack(bits))
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeMeasurementControlParametersDescriptionFrom(r); err == nil {
					t.Fatalf("accepted %d/%d bits", cut, len(bits))
				}
				if r.Position() > cut {
					t.Fatal("escaped truncated bound")
				}
			}
		}
	}
	v := MeasurementControlParametersDescription{MeasurementControlEUTRANGroup: &MeasurementControlParametersDescriptionMeasurementControlEUTRANGroup{EUTRANFREQUENCYINDEX: 8}}
	if _, err := EncodeMeasurementControlParametersDescription(v); err == nil {
		t.Fatal("three-bit overflow accepted")
	}
	v = MeasurementControlParametersDescription{MeasurementControlUTRANGroup: &MeasurementControlParametersDescriptionMeasurementControlUTRANGroup{UTRANFREQUENCYINDEX: 32}}
	if _, err := EncodeMeasurementControlParametersDescription(v); err == nil {
		t.Fatal("five-bit overflow accepted")
	}
}

func TestMeasurementControlGrowth(t *testing.T) {
	var previous float64
	for _, n := range []int{16, 32, 64, 128} {
		bits := measurementControlBits(3, n, 1)
		wire := measurementControlPack(bits)
		if len(bits) != 14+10*n {
			t.Fatalf("size %d", len(bits))
		}
		allocations := testing.AllocsPerRun(3, func() {
			d, err := DecodeMeasurementControlParametersDescription(wire)
			if err != nil {
				panic(err)
			}
			if len(d.Value.MeasurementControlUTRANGroup.UTRANFREQUENCYINDEXList) != n {
				panic("count")
			}
		})
		if !raceEnabled && previous > 0 && allocations > 2*previous+8 {
			t.Fatalf("superlinear allocations %g -> %g", previous, allocations)
		}
		previous = allocations
	}
}

func FuzzMeasurementControl(f *testing.F) {
	for outer := range 4 {
		f.Add(measurementControlPack(measurementControlBits(outer, 2, 1)))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		d, err := DecodeMeasurementControlParametersDescription(data)
		if err != nil {
			return
		}
		b, err := EncodeMeasurementControlParametersDescription(d.Value)
		if err != nil || !bytes.Equal(b, data) {
			t.Fatalf("replay %x %x %v", data, b, err)
		}
	})
}

func measurementControlPack(s string) []byte {
	wire := make([]byte, (len(s)+7)/8)
	for i, b := range s {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	return wire
}

func TestMeasurementControlCanonicalExtentRequired(t *testing.T) {
	_, err := EncodeMeasurementControlParametersDescriptionCanonical(MeasurementControlParametersDescription{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("missing containing extent: %v", err)
	}
}
