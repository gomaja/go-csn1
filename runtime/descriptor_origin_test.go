package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

// TS 24.007 V20.0.0 Annex B §B.1.2.2 positions L/H relative to the
// physical block; TS 44.060 V19.0.0 §11 p12940 uses octet-aligned 0x2b.
func TestDescriptorSourceOriginPadding(t *testing.T) {
	for _, tc := range []struct {
		origin int
		data   byte
	}{{0, 0xab}, {14, 0x8a}} {
		t.Run(fmt.Sprint(tc.origin), func(t *testing.T) {
			d := Descriptor{DecodeFrom: func(r *Reader) (any, error) {
				if r.Position() != tc.origin || r.BoundedRemaining() != 8 {
					return nil, fmt.Errorf("origin/extent = %d/%d", r.Position(), r.BoundedRemaining())
				}
				if err := r.Expect("10LLLLLL"); err != nil {
					return nil, err
				}
				return r.Position(), nil
			}}
			setDescriptorOrigin(t, &d, tc.origin)
			a, err := d.DecodeBits([]byte{tc.data}, 8)
			if err != nil || a != tc.origin+8 {
				t.Fatalf("plain: %v %v", a, err)
			}
			b, report, receiver, err := d.DecodeBitsWithReport([]byte{tc.data}, 8)
			if err != nil || !reflect.DeepEqual(a, b) || len(report.Observations) != 0 || len(receiver.Regions) != 0 {
				t.Fatalf("report: %v %+v %+v %v", b, report, receiver, err)
			}
		})
	}
}

func setDescriptorOrigin(t testing.TB, d *Descriptor, origin int) {
	t.Helper()
	d.SourceOrigin = origin
}

func TestDescriptorSourceOriginExactExtent(t *testing.T) {
	// Packet EMR's carrier has 170 significant content bits after block bit 14
	// (TS 44.060 V19.0.0 §§10.3.2,11.2.0.2,11.2.9d). Its six byte-completion
	// bits and the following sentinel must never reach DecodeFrom.
	for origin := 0; origin < 24; origin++ {
		for completion := 0; completion < 64; completion++ {
			input := bytes.Repeat([]byte{0xa5}, 23)
			input[21] = 0x80 | byte(completion)
			input[22] = 0xff
			d := Descriptor{DecodeFrom: func(r *Reader) (any, error) {
				if r.Position() != origin || r.BoundedRemaining() != 170 {
					return nil, fmt.Errorf("origin/extent %d/%d", r.Position(), r.BoundedRemaining())
				}
				value, err := r.ReadBitString(170)
				if err != nil {
					return nil, err
				}
				if r.Position() != origin+170 || r.BoundedRemaining() != 0 {
					return nil, fmt.Errorf("rounded end %d/%d", r.Position(), r.BoundedRemaining())
				}
				if _, err = r.ReadUint(1); err == nil {
					return nil, errors.New("completion or sentinel consumed")
				}
				return value, nil
			}}
			setDescriptorOrigin(t, &d, origin)
			want := append([]byte(nil), input[:22]...)
			want[21] &= 0xc0
			a, err := d.DecodeBits(input, 170)
			if err != nil || !reflect.DeepEqual(a, BitString{Bytes: want, BitLength: 170}) {
				t.Fatalf("origin %d completion %d: %v", origin, completion, err)
			}
			b, _, _, err := d.DecodeBitsWithReport(input, 170)
			if err != nil || !reflect.DeepEqual(a, b) {
				t.Fatalf("report origin %d completion %d: %v", origin, completion, err)
			}
			if input[21] != (0x80|byte(completion)) || input[22] != 0xff {
				t.Fatal("input modified")
			}
		}
	}
}

func TestDescriptorSourceOriginReports(t *testing.T) {
	for origin := 0; origin < 24; origin++ {
		for _, fail := range []bool{false, true} {
			source := Source{Standard: "synthetic", Clause: "1"}
			decode := func(r *Reader) (any, error) {
				if err := r.Enter("Body"); err != nil {
					return nil, err
				}
				defer r.Leave()
				start := r.Position()
				value, err := r.ReadUint(2)
				if err != nil {
					return nil, err
				}
				if err = r.Observe(start, 2, value, source); err != nil {
					return nil, err
				}
				region, err := r.ReadErrorRegion(Ignore, 3, source)
				if err != nil {
					return nil, err
				}
				if fail {
					return region, r.Expect("1")
				}
				return region, nil
			}
			d := Descriptor{DecodeFrom: decode}
			setDescriptorOrigin(t, &d, origin)
			a, plainErr := d.DecodeBits([]byte{0xa0}, 6)
			b, reserved, receiver, reportErr := d.DecodeBitsWithReport([]byte{0xa0}, 6)
			if !reflect.DeepEqual(a, b) || fmt.Sprint(plainErr) != fmt.Sprint(reportErr) {
				t.Fatalf("origin %d report path differs: %v / %v", origin, plainErr, reportErr)
			}
			want := Location{Path: "Body", BitOffset: origin, Width: 2, Source: source}
			if len(reserved.Observations) != 1 || reserved.Observations[0].Location != want || reserved.Observations[0].Value != 2 || reserved.Dropped != 0 {
				t.Fatalf("origin %d reserved: %+v", origin, reserved)
			}
			want.BitOffset = origin + 2
			want.Width = 3
			if len(receiver.Regions) != 1 || receiver.Regions[0].Location != want || receiver.Regions[0].Label != Ignore || receiver.Dropped != 0 {
				t.Fatalf("origin %d receiver: %+v", origin, receiver)
			}
			if fail {
				var operation *OperationError
				if !errors.As(reportErr, &operation) || operation.Location.BitOffset != origin+5 || operation.Location.Width != 1 || operation.Location.Path != "Body" {
					t.Fatalf("origin %d diagnostic: %v", origin, reportErr)
				}
			} else if reportErr != nil {
				t.Fatal(reportErr)
			}
		}
	}
}

func TestDescriptorSourceOriginZeroFill(t *testing.T) {
	for origin := 0; origin < 24; origin++ {
		d := Descriptor{DecodeFrom: func(r *Reader) (any, error) {
			r.SetZeroExtension(true)
			return r.ReadUintWithFill(5, fillSource, "Tail")
		}}
		setDescriptorOrigin(t, &d, origin)
		a, err := d.DecodeBits([]byte{0xc0}, 2)
		if err != nil || a != uint64(24) {
			t.Fatalf("plain fill: %v %v", a, err)
		}
		b, reserved, receiver, err := d.DecodeBitsWithReport([]byte{0xc0}, 2)
		if err != nil || !reflect.DeepEqual(a, b) || len(reserved.Observations) != 0 || len(receiver.ZeroFill) != 1 || receiver.ZeroFill[0] != (Location{Path: "Tail", BitOffset: origin + 2, Width: 3, Source: fillSource}) {
			t.Fatalf("origin %d fill: %v %+v %+v %v", origin, b, reserved, receiver, err)
		}
	}
}

func TestDescriptorSourceOriginInvalidInput(t *testing.T) {
	called := false
	for _, tc := range []struct {
		origin, bits int
		kind         ErrorKind
	}{
		{-1, 0, InvalidSchema}, {maxBits + 1, 0, InvalidSchema}, {int(^uint(0) >> 1), 0, InvalidSchema},
		{maxBits, 1, Limit}, {maxBits - 1, 2, Limit}, {14, -1, InvalidValue}, {14, 9, InvalidValue},
	} {
		d := Descriptor{DecodeFrom: func(*Reader) (any, error) { called = true; return nil, nil }}
		setDescriptorOrigin(t, &d, tc.origin)
		_, err := d.DecodeBits([]byte{0}, tc.bits)
		var de *DecodeError
		if !errors.As(err, &de) || de.Kind != tc.kind {
			t.Fatalf("%+v plain: %v", tc, err)
		}
		_, reserved, receiver, reportErr := d.DecodeBitsWithReport([]byte{0}, tc.bits)
		if fmt.Sprint(err) != fmt.Sprint(reportErr) || len(reserved.Observations) != 0 || len(receiver.Regions) != 0 || len(receiver.ZeroFill) != 0 {
			t.Fatalf("%+v report: %v", tc, reportErr)
		}
	}
	if called {
		t.Fatal("invalid input reached decoder")
	}
	d := Descriptor{}
	setDescriptorOrigin(t, &d, 14)
	if _, err := d.DecodeBits(nil, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
	if _, _, _, err := d.DecodeBitsWithReport(nil, 0); !errors.Is(err, ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestDescriptorSourceOriginZeroAndCallerOwnership(t *testing.T) {
	decode := func(r *Reader) (any, error) { return r.ReadUint(3) }
	d := Descriptor{DecodeFrom: decode}
	// The zero value retains both the reader and nil-collector allocation path.
	data := []byte{0xa5}
	old := testing.AllocsPerRun(100, func() {
		r, err := NewChoiceReaderBits(data, 7, nil)
		if err != nil {
			panic(err)
		}
		_, _ = decode(r)
	})
	owned := testing.AllocsPerRun(100, func() { _, _ = d.DecodeBits(data, 7) })
	if owned != old {
		t.Fatalf("zero origin allocations %g, original %g", owned, old)
	}
	for phase := 0; phase < 8; phase++ {
		c := NewReceiverCollector()
		r, err := NewChoiceReaderBits([]byte{0xff, 0xff}, 16, c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		r.Set("context", 7)
		steps := r.steps
		budget := *steps
		setDescriptorOrigin(t, &d, 14)
		if _, err = d.DecodeFrom(r); err != nil {
			t.Fatal(err)
		}
		if r.Position() != phase+3 || r.collector != c || r.steps != steps || *steps != budget || r.vars["context"] != 7 || r.BoundedRemaining() != 13-phase {
			t.Fatalf("caller state changed at phase %d", phase)
		}
	}
}

func TestDescriptorSourceOriginStorageBound(t *testing.T) {
	for _, origin := range []int{1, 14, 8192, maxBits} {
		for _, bits := range []int{0, 1, 8, 170, 8192} {
			if bits > maxBits-origin {
				continue
			}
			d := Descriptor{DecodeFrom: func(r *Reader) (any, error) {
				if r.Check() != nil || len(r.data) != (origin+bits+7)/8 || r.Position() != origin || r.BoundedRemaining() != bits {
					return nil, errors.New("source storage exceeds exact positioned extent")
				}
				return nil, nil
			}}
			setDescriptorOrigin(t, &d, origin)
			if _, err := d.DecodeBits(make([]byte, (bits+7)/8), bits); err != nil {
				t.Fatalf("origin %d extent %d: %v", origin, bits, err)
			}
		}
	}
}

func FuzzDescriptorSourceOrigin(f *testing.F) {
	f.Add([]byte{0x8a}, 8, 14)
	f.Add([]byte{0xff, 0xa5}, 11, 3)
	f.Add([]byte{}, 0, 0)
	f.Add([]byte{0}, -1, -1)
	f.Fuzz(func(t *testing.T, data []byte, bits, origin int) {
		if len(data) > 4096 || origin > 4096 && origin <= maxBits {
			t.Skip()
		}
		d := Descriptor{DecodeFrom: func(r *Reader) (any, error) {
			if r.Position() != origin || r.BoundedRemaining() != bits {
				return nil, errors.New("wrong origin/extent")
			}
			value, err := r.ReadBitString(bits)
			if err != nil {
				return nil, err
			}
			if r.Position() != origin+bits || r.BoundedRemaining() != 0 {
				return nil, errors.New("rounded content")
			}
			if _, err = r.ReadUint(1); err == nil {
				return nil, errors.New("completion consumed")
			}
			return value, nil
		}}
		setDescriptorOrigin(t, &d, origin)
		a, err := d.DecodeBits(data, bits)
		b, _, _, reportErr := d.DecodeBitsWithReport(data, bits)
		valid := bits >= 0 && bits <= len(data)*8 && origin >= 0 && origin <= maxBits && bits <= maxBits-origin
		if (err == nil) != valid || fmt.Sprint(err) != fmt.Sprint(reportErr) || !reflect.DeepEqual(a, b) {
			t.Fatalf("valid %v errors %v / %v", valid, err, reportErr)
		}
		if !valid {
			return
		}
		want := append([]byte{}, data[:(bits+7)/8]...)
		if bits%8 != 0 {
			want[len(want)-1] &= 0xff << uint(8-bits%8)
		}
		if !reflect.DeepEqual(a, BitString{Bytes: want, BitLength: bits}) {
			t.Fatal("content changed")
		}
	})
}

func TestDescriptorSourceOriginPositionedSymbols(t *testing.T) {
	for origin := 0; origin < 24; origin++ {
		// Compute the transmitted literal from the standard's octet pattern,
		// independently of the reader's literalBit implementation.
		input := []byte{0}
		for bit := 0; bit < 8; bit++ {
			value := (byte(0x2b) >> uint(7-(origin+bit)%8)) & 1
			if bit%2 != 0 {
				value ^= 1
			}
			input[0] |= value << uint(7-bit)
		}
		d := Descriptor{SourceOrigin: origin, DecodeFrom: func(r *Reader) (any, error) { return nil, r.Expect("LHLHLHLH") }}
		if _, err := d.DecodeBits(input, 8); err != nil {
			t.Fatalf("origin %d plain: %v", origin, err)
		}
		if _, _, _, err := d.DecodeBitsWithReport(input, 8); err != nil {
			t.Fatalf("origin %d report: %v", origin, err)
		}
	}
}
