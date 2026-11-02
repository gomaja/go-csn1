package pulseformat

import (
	"bytes"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"strings"
	"testing"
)

func clausePack(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << (7 - uint(i%8))
		}
	}
	return out
}
func clauseReader(t *testing.T, bits string) *runtime.Reader {
	t.Helper()
	r, err := runtime.NewReaderBits(clausePack(bits), len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// TS 44.060 V19.0.0 §12.8.3 table 12.8.3.1, paragraphs 22812–22821:
// coding 1 has three bits; coding 2 selects a length-minus-one bitmap or
// one of the prefix codewords 1, 01, 00. Incomplete bitmap tails use the
// printed error region, retaining the received bits for replay.
func TestClause12_8_3(t *testing.T) {
	vectors := []struct{ name, bits string }{
		{"coding1-0", "0000"},
		{"coding1-1", "0001"},
		{"coding1-2", "0010"},
		{"coding1-3", "0011"},
		{"coding1-4", "0100"},
		{"coding1-5", "0101"},
		{"coding1-6", "0110"},
		{"coding1-7", "0111"},
		{"nonhopping-1", "111"},
		{"nonhopping-01", "1101"},
		{"nonhopping-00", "1100"},
		{"bitmap-1-1", "1000000001"},
		{"bitmap-2-01", "10000000101"},
		{"bitmap-2-00", "10000000100"},
		{"bitmap-5-10100", "10000010010100"},
		{"bitmap-6-100101", "100000101100101"},
		{"bitmap-1-0", "1000000000"},
		{"bitmap-2-10", "10000000110"},
		{"bitmap-128-000000000000", "10111111100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{"bitmap-128-111111111111", "10111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			r := clauseReader(t, v.bits)
			got, err := DecodePulseFormatIEFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if r.Position() != len(v.bits) {
				t.Fatalf("consumed %d want %d", r.Position(), len(v.bits))
			}
			choice := got.PulseFormatCoding1Choice
			if v.bits[0] == '0' {
				want := uint8(0)
				for _, b := range v.bits[1:] {
					want = want*2 + uint8(b-'0')
				}
				if choice.Alternative != 0 || choice.PulseFormatCoding1 == nil || choice.PulseFormatCoding1.PulseFormatCoding1 != want {
					t.Fatalf("wrong coding 1: %+v", choice)
				}
				edited := got
				copyArm := *choice.PulseFormatCoding1
				copyArm.PulseFormatCoding1 ^= 7
				edited.PulseFormatCoding1Choice.PulseFormatCoding1 = &copyArm
				w := runtime.NewWriter()
				if err := EncodePulseFormatIETo(w, edited); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != 4 || out.Bytes[0] != (want^7)<<4 {
					t.Fatalf("edited %x: %v", out.Bytes, err)
				}
			} else {
				if choice.Alternative != 1 || choice.PulseFormatCoding2 == nil {
					t.Fatal("missing coding 2")
				}
				inner := choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice
				if v.bits[1] == '1' {
					if inner.Alternative != 1 || inner.NonHoppingCarrierPulseFormat == nil {
						t.Fatal("missing nonhopping arm")
					}
					code := inner.NonHoppingCarrierPulseFormat.NonHoppingCarrierPulseFormat
					want := 0
					switch v.bits[2:] {
					case "01":
						want = 1
					case "00":
						want = 2
					}
					if int(code.Alternative) != want {
						t.Fatalf("codeword selector %d want %d", code.Alternative, want)
					}
				} else {
					if inner.Alternative != 0 || inner.PulseFormatBitmapLength == nil {
						t.Fatal("missing bitmap arm")
					}
					b := inner.PulseFormatBitmapLength
					if int(b.PulseFormatBitmapLength)+1 != len(v.bits)-9 {
						t.Fatal("wrong length-minus-one")
					}
					bitmap := v.bits[9:]
					want := []int{}
					for len(bitmap) > 0 {
						if bitmap[0] == '1' {
							want = append(want, 0)
							bitmap = bitmap[1:]
						} else if len(bitmap) > 1 {
							if bitmap[1] == '1' {
								want = append(want, 1)
							} else {
								want = append(want, 2)
							}
							bitmap = bitmap[2:]
						} else {
							break
						}
					}
					if len(bitmap) == 0 {
						if b.PulseFormatBitmap.Alternative != 0 || b.PulseFormatBitmap.Known == nil {
							t.Fatal("valid codewords not typed")
						}
						entries := *b.PulseFormatBitmap.Known
						if len(entries) != len(want) {
							t.Fatalf("entries %d want %d", len(entries), len(want))
						}
						for i, e := range entries {
							if int(e.Alternative) != want[i] {
								t.Fatalf("entry %d selector %d want %d", i, e.Alternative, want[i])
							}
						}
					} else if b.PulseFormatBitmap.Alternative != 1 || b.PulseFormatBitmap.Ignored == nil {
						t.Fatal("incomplete codeword not retained as error region")
					}
				}
			}
			w := runtime.NewWriter()
			err = EncodePulseFormatIETo(w, got)
			if err != nil { // The printed receiver error region has no sender language.
				if !strings.Contains(v.name, "bitmap-1-0") && !strings.Contains(v.name, "bitmap-2-10") {
					t.Fatal(err)
				}
			} else {
				encoded, e := w.FinishBits(runtime.BitString{})
				if e != nil || encoded.BitLength != len(v.bits) || !bytes.Equal(encoded.Bytes, clausePack(v.bits)) {
					t.Fatalf("encode %x/%d: %v", encoded.Bytes, encoded.BitLength, e)
				}
			}
			received, err := DecodePulseFormatIE(clausePack(v.bits))
			if err != nil {
				t.Fatal(err)
			}
			replay, err := EncodePulseFormatIE(received.Value)
			if err != nil || !bytes.Equal(replay, clausePack(v.bits)) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			if v.bits[0] == '0' || v.bits[1] == '1' {
				canonical, err := EncodePulseFormatIECanonicalAtLength(received.Value, 1)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodePulseFormatIE(canonical); err != nil {
					t.Fatal(err)
				}
			}
			for phase := 0; phase < 8; phase++ {
				framed := clauseReader(t, strings.Repeat("1", phase)+v.bits+"10101")
				if _, err := framed.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := framed.PushLimit(len(v.bits))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodePulseFormatIEFrom(framed); err != nil {
					t.Fatal(err)
				}
				if err := framed.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				sentinel, err := framed.ReadUint(5)
				if err != nil || sentinel != 21 {
					t.Fatalf("sentinel %d: %v", sentinel, err)
				}
			}
			for n := 0; n < len(v.bits); n++ {
				if _, err := DecodePulseFormatIEFrom(clauseReader(t, v.bits[:n])); err == nil {
					t.Fatalf("accepted prefix %d/%d", n, len(v.bits))
				}
			}
		})
	}
}
func TestClause12_8_3InvalidValue(t *testing.T) {
	for _, name := range []string{"coding-width", "selector", "missing-arm"} {
		t.Run(name, func(t *testing.T) {
			got, err := DecodePulseFormatIEFrom(clauseReader(t, "0000"))
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "coding-width":
				got.PulseFormatCoding1Choice.PulseFormatCoding1.PulseFormatCoding1 = 8
			case "selector":
				got.PulseFormatCoding1Choice.Alternative = 2
			case "missing-arm":
				got.PulseFormatCoding1Choice.PulseFormatCoding1 = nil
			}
			if err := EncodePulseFormatIETo(runtime.NewWriter(), got); err == nil {
				t.Fatal("accepted invalid typed field")
			}
		})
	}
}

// TS 44.060 V19.0.0 §12.8.3 p22817–22819: the bitmap's length
// intersection and no-string fallback must also hold for freshly edited values.
func TestClause12_8_3CanonicalBitmapErrors(t *testing.T) {
	for _, name := range []string{"length", "known-fallback"} {
		t.Run(name, func(t *testing.T) {
			value, err := DecodePulseFormatIEFrom(clauseReader(t, "1000000001"))
			if err != nil {
				t.Fatal(err)
			}
			bitmap := value.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.PulseFormatBitmapLength
			if name == "length" {
				bitmap.PulseFormatBitmapLength = 1
				_, err = EncodePulseFormatIECanonicalAtLength(value, 2)
				var bound *runtime.BoundError
				if !errors.As(err, &bound) || bound.Kind != runtime.LengthBound {
					t.Fatalf("length error: %v", err)
				}
			} else {
				bitmap.PulseFormatBitmap.Alternative = 1
				bitmap.PulseFormatBitmap.Known = nil
				bitmap.PulseFormatBitmap.Ignored = &runtime.BitString{Bytes: []byte{128}, BitLength: 1}
				_, err = EncodePulseFormatIECanonicalAtLength(value, 2)
				if !errors.Is(err, runtime.ErrFallbackKnown) {
					t.Fatalf("known-fallback error: %v", err)
				}
			}
		})
	}
}

func TestClause12_8_3Edits(t *testing.T) {
	// TS 44.060 V19.0.0 §12.8.3 p22820: edit all three non-hopping codes.
	codes := []string{"1", "01", "00"}
	for alternative, code := range codes {
		received, err := DecodePulseFormatIE(clausePack("11" + code))
		if err != nil {
			t.Fatal(err)
		}
		edited := runtime.Canonical(received.Value)
		field := &edited.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.NonHoppingCarrierPulseFormat.NonHoppingCarrierPulseFormat
		next := (alternative + 1) % 3
		field.Alternative = PulseFormatCoding2StructPulseFormatBitmapLengthChoiceNonHoppingCarrierPulseFormatNonHoppingCarrierPulseFormatAlternative(next)
		field.Alt1 = nil
		field.Alt01 = nil
		field.Alt00 = nil
		switch next {
		case 0:
			field.Alt1 = &struct{}{}
		case 1:
			field.Alt01 = &struct{}{}
		case 2:
			field.Alt00 = &struct{}{}
		}
		changed, err := EncodePulseFormatIECanonicalAtLength(edited, 1)
		if err != nil || !bytes.Equal(changed, clausePack("11"+codes[next])) {
			t.Fatalf("non-hopping edit %s: %x %v", code, changed, err)
		}
	}
	// Replace the typed bitmap group, retaining the enclosing root choice.
	// The lengths are hand-calculated from the printed length-minus-one rule.
	for _, pair := range [][2]string{{"1000000001", "10000000101"}, {"10000000101", "10000000100"}, {"10000000100", "1000000001"}} {
		received, err := DecodePulseFormatIE(clausePack(pair[0]))
		if err != nil {
			t.Fatal(err)
		}
		replacement, err := DecodePulseFormatIEFrom(clauseReader(t, pair[1]))
		if err != nil {
			t.Fatal(err)
		}
		edited := runtime.Canonical(received.Value)
		edited.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.PulseFormatBitmapLength = replacement.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.PulseFormatBitmapLength
		changed, err := EncodePulseFormatIECanonicalAtLength(edited, 2)
		if err != nil || !bytes.Equal(changed, clausePack(pair[1])) {
			t.Fatalf("bitmap edit %x: %v", changed, err)
		}
	}
}

func FuzzClause12_8_3(f *testing.F) {
	for _, s := range []string{"0000", "111", "1101", "1100", "1000000001", "10000000110"} {
		f.Add(clausePack(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := DecodePulseFormatIE(data)
		if err == nil {
			replay, err := EncodePulseFormatIE(got.Value)
			if err != nil || !bytes.Equal(replay, data) {
				t.Fatalf("replay %x: %v", replay, err)
			}
		}
		component, componentErr := DecodePulseFormatCoding2Struct(data)
		if componentErr == nil {
			replay, encodeErr := EncodePulseFormatCoding2Struct(component.Value)
			if encodeErr != nil || !bytes.Equal(data, replay) {
				t.Fatalf("component replay %x: %v", replay, encodeErr)
			}
		}
	})
}
