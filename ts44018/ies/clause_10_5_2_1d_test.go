package ies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.1d p14001–14007.
// Both lists use one presence bit plus fourteen frequency bits per entry,
// then a zero terminator. All vectors are synthetic and independently decoded.
func TestUTRANFreqListPublication(t *testing.T) {
	d, err := LookupClause("10.5.2.1d", "UTRAN Freq List")
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name, wire, canonical string
		semantic              int
		fdd, tdd              []uint64
	}{
		{"fdd-0-tdd-0", "013f", "0100", 10, []uint64{}, []uint64{}},
		{"fdd-1-tdd-0", "0380067f", "03800600", 25, []uint64{3}, []uint64{}},
		{"fdd-0-tdd-1", "037fff7f", "037fff00", 25, []uint64{}, []uint64{16383}},
		{"fdd-1-tdd-1", "048006fffe", "048006fffe", 40, []uint64{3}, []uint64{16383}},
		{"fdd-8-tdd-0", "108007013204ac0de824f05c20dcc2023f", "108007013204ac0de824f05c20dcc20200", 130, []uint64{3, 76, 149, 222, 295, 368, 441, 514}, []uint64{}},
		{"fdd-0-tdd-8", "107fff80b502d6088416b838d088613e3f", "107fff80b502d6088416b838d088613e00", 130, []uint64{}, []uint64{16383, 90, 181, 272, 363, 454, 545, 636}},
		{"fdd-16-tdd-16", "3d8007013204ac0de824f05c20dcc20284970a5216ec32686df0ee2200c44a7fff80b502d6088416b838d088613e42d786650e361f44443893d13e62aa3f", "3d8007013204ac0de824f05c20dcc20284970a5216ec32686df0ee2200c44a7fff80b502d6088416b838d088613e42d786650e361f44443893d13e62aa00", 490, []uint64{3, 76, 149, 222, 295, 368, 441, 514, 587, 660, 733, 806, 879, 952, 1025, 1098}, []uint64{16383, 90, 181, 272, 363, 454, 545, 636, 727, 818, 909, 1000, 1091, 1182, 1273, 1364}},
		{"fdd-63-tdd-63", "ed8007013204ac0de824f05c20dcc20284970a5216ec32686df0ee2200c44a89271372292c56e8b6f1802324c6928db71c923b6c7b68fff2122448c8da924725b24dac9fe948f2a4256ccb2296d72ed25fecc46991f3362690cd6a9b6737f2722ce8e9daf3c827b4cfb29ff74112846d0d6a23f45a28d8bfffc05a816b04420b5c1c6844309f216bc332871b0fa2221c49e89f315522d7c60a8ccb1b0238dc7768fa320b2443c8e2927b26624f9ca4e95532c125afcbba982b31c2665cd269b03377271bce929ddb3d227d1cffea0b342d2887d16aa38b488293dd2d6a6634e329f3d442a93b53e2aa9d5aeac12f", "ed8007013204ac0de824f05c20dcc20284970a5216ec32686df0ee2200c44a89271372292c56e8b6f1802324c6928db71c923b6c7b68fff2122448c8da924725b24dac9fe948f2a4256ccb2296d72ed25fecc46991f3362690cd6a9b6737f2722ce8e9daf3c827b4cfb29ff74112846d0d6a23f45a28d8bfffc05a816b04420b5c1c6844309f216bc332871b0fa2221c49e89f315522d7c60a8ccb1b0238dc7768fa320b2443c8e2927b26624f9ca4e95532c125afcbba982b31c2665cd269b03377271bce929ddb3d227d1cffea0b342d2887d16aa38b488293dd2d6a6634e329f3d442a93b53e2aa9d5aeac120", 1900, []uint64{3, 76, 149, 222, 295, 368, 441, 514, 587, 660, 733, 806, 879, 952, 1025, 1098, 1171, 1244, 1317, 1390, 1463, 1536, 1609, 1682, 1755, 1828, 1901, 1974, 2047, 2120, 2193, 2266, 2339, 2412, 2485, 2558, 2631, 2704, 2777, 2850, 2923, 2996, 3069, 3142, 3215, 3288, 3361, 3434, 3507, 3580, 3653, 3726, 3799, 3872, 3945, 4018, 4091, 4164, 4237, 4310, 4383, 4456, 4529}, []uint64{16383, 90, 181, 272, 363, 454, 545, 636, 727, 818, 909, 1000, 1091, 1182, 1273, 1364, 1455, 1546, 1637, 1728, 1819, 1910, 2001, 2092, 2183, 2274, 2365, 2456, 2547, 2638, 2729, 2820, 2911, 3002, 3093, 3184, 3275, 3366, 3457, 3548, 3639, 3730, 3821, 3912, 4003, 4094, 4185, 4276, 4367, 4458, 4549, 4640, 4731, 4822, 4913, 5004, 5095, 5186, 5277, 5368, 5459, 5550, 5641}},
	}
	var numbers func(reflect.Value, string) []uint64
	numbers = func(v reflect.Value, name string) []uint64 {
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
		}
		var out []uint64
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				f := v.Type().Field(i)
				if f.Name == "Wire" {
					continue
				}
				x := v.Field(i)
				if f.Name == name && x.CanUint() {
					out = append(out, x.Uint())
				} else {
					out = append(out, numbers(x, name)...)
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				out = append(out, numbers(v.Index(i), name)...)
			}
		}
		return out
	}
	equal := func(a, b []uint64) bool { return reflect.DeepEqual(append([]uint64{}, a...), append([]uint64{}, b...)) }
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			wire, e := hex.DecodeString(row.wire)
			if e != nil {
				t.Fatal(e)
			}
			value, e := d.Decode(wire)
			if e != nil {
				t.Fatal(e)
			}
			check := func(v any) {
				t.Helper()
				rv := reflect.ValueOf(v)
				if rv.Type().Name() != "UTRANFreqList" {
					rv = rv.FieldByName("Value")
				}
				if rv.Type().Name() != "UTRANFreqList" {
					t.Fatalf("untyped %T", v)
				}
				if !equal(numbers(rv, "FDDARFCN"), row.fdd) || !equal(numbers(rv, "TDDARFCN"), row.tdd) {
					t.Fatalf("typed frequency mismatch: %+v", v)
				}
			}
			check(value)
			replay, e := d.Encode(value)
			if e != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x %v", replay, e)
			}
			canonical, e := d.CanonicalAtLength(value, len(wire))
			if e != nil || hex.EncodeToString(canonical) != row.canonical {
				t.Fatalf("canonical %x %v", canonical, e)
			}
			v, e := d.Decode(canonical)
			if e != nil {
				t.Fatal(e)
			}
			check(v)
			for phase := 0; phase < 8; phase++ {
				data := make([]byte, (phase+len(wire)*8+8+7)/8)
				for i := range len(wire) * 8 {
					if wire[i/8]&(1<<uint(7-i%8)) != 0 {
						at := phase + i
						data[at/8] |= 1 << uint(7-at%8)
					}
				}
				for i := 0; i < 8; i++ {
					if 0xa5&(1<<uint(7-i)) != 0 {
						at := phase + len(wire)*8 + i
						data[at/8] |= 1 << uint(7-at%8)
					}
				}
				reader := runtime.NewReader(data)
				if _, e = reader.ReadUint(phase); e != nil {
					t.Fatal(e)
				}
				old, e := reader.PushLimit(len(wire) * 8)
				if e != nil {
					t.Fatal(e)
				}
				v, e := d.DecodeFrom(reader)
				if e != nil {
					t.Fatal(e)
				}
				check(v)
				if e = reader.PopLimit(old); e != nil {
					t.Fatal(e)
				}
				sentinel, e := reader.ReadUint(8)
				if e != nil || sentinel != 0xa5 {
					t.Fatalf("phase %d sentinel %x %v", phase, sentinel, e)
				}
			}
			for cut := 0; cut < row.semantic; cut++ {
				reader := runtime.NewReader(wire)
				if _, e = reader.PushLimit(cut); e != nil {
					t.Fatal(e)
				}
				if _, e = d.DecodeFrom(reader); e == nil {
					t.Fatalf("accepted truncated prefix %d", cut)
				}
				if reader.Position() > cut {
					t.Fatalf("crossed %d", cut)
				}
			}
			mutable := reflect.New(reflect.TypeOf(value)).Elem()
			mutable.Set(reflect.ValueOf(value))
			var edit func(reflect.Value) bool
			edit = func(v reflect.Value) bool {
				for v.Kind() == reflect.Pointer {
					if v.IsNil() {
						return false
					}
					v = v.Elem()
				}
				switch v.Kind() {
				case reflect.Struct:
					for i := 0; i < v.NumField(); i++ {
						if v.Type().Field(i).Name == "Wire" {
							continue
						}
						if (v.Type().Field(i).Name == "FDDARFCN" || v.Type().Field(i).Name == "TDDARFCN") && v.Field(i).CanSet() {
							v.Field(i).SetUint(v.Field(i).Uint() ^ 1)
							return true
						}
						if edit(v.Field(i)) {
							return true
						}
					}
				case reflect.Slice:
					for i := 0; i < v.Len(); i++ {
						if edit(v.Index(i)) {
							return true
						}
					}
				}
				return false
			}
			if edit(mutable.FieldByName("Value")) {
				out, e := d.Encode(mutable.Interface())
				if e != nil || bytes.Equal(out, wire) {
					t.Fatalf("lost edit %x %v", out, e)
				}
				changed, e := d.Decode(out)
				if e != nil {
					t.Fatal(e)
				}
				if equal(numbers(reflect.ValueOf(changed), "FDDARFCN"), row.fdd) && equal(numbers(reflect.ValueOf(changed), "TDDARFCN"), row.tdd) {
					t.Fatal("edited frequency not decoded")
				}
			}
			allocations := testing.AllocsPerRun(1, func() {
				if _, e := d.Decode(wire); e != nil {
					t.Fatal(e)
				}
			})
			if !raceEnabled && allocations > float64(32*len(wire)+128) {
				t.Fatalf("nonlinear allocations: %g at %d bytes", allocations, len(wire))
			}
		})
	}
	// CHANNEL RELEASE has three mandatory L3 octets and the separate UTRAN IEI:
	// TS 44.018 V19.0.0 §9.1.7 p4880–4902, p4933–4938;
	// TS 44.006 V19.0.0 §8.8.5 p2466 gives 251-3-1=247 root octets.
	boundary := make([]byte, 247)
	boundary[0] = 246
	v, e := d.Decode(boundary)
	if e != nil {
		t.Fatal(e)
	}
	if out, e := d.CanonicalAtLength(v, 247); e != nil || len(out) != 247 {
		t.Fatalf("at containing bound %x %v", out, e)
	}
	out, e := d.CanonicalAtLength(v, 248)
	var extent *runtime.ExtentError
	if out != nil || !errors.As(e, &extent) || extent.Maximum != 247 {
		t.Fatalf("past containing bound %x %v", out, e)
	}
}

func FuzzUTRANFreqListReplay(f *testing.F) {
	d, err := LookupClause("10.5.2.1d", "UTRAN Freq List")
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte{1, 0})
	f.Add([]byte{3, 128, 7, 128})
	f.Add([]byte{0xff, 0xff})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 512 {
			return
		}
		v, e := d.Decode(wire)
		if e != nil {
			return
		}
		out, e := d.Encode(v)
		if e != nil || !bytes.Equal(out, wire) {
			t.Fatalf("replay %x %v", out, e)
		}
	})
}
