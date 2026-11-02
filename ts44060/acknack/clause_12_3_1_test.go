package acknack

import (
	"bytes"
	"fmt"
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

// TS 44.060 V19.0.0 §12.3.1, table 12.3.1.1, paragraphs 22093–22106:
// 3 flags + 11 SSN bits + one CRBB selector; the optional CRBB uses
// 7 length bits, one colour bit and Lc bits. URBB consumes the containing bound.
func TestClause12_3_1(t *testing.T) {
	vectors := []struct {
		name, bits string
		bounded    bool
		compressed int
		urbb       string
	}{
		{"bound-False-compressed--1-urbb-0", "0101101010101010", false, -1, ""},
		{"bound-False-compressed--1-urbb-1", "01011010101010101", false, -1, "1"},
		{"bound-False-compressed--1-urbb-8", "010110101010101010010110", false, -1, "10010110"},
		{"bound-False-compressed--1-urbb-31", "01011010101010101001011010010110100101101001011", false, -1, "1001011010010110100101101001011"},
		{"bound-False-compressed-0-urbb-0", "010110101010101100000001", false, 0, ""},
		{"bound-False-compressed-0-urbb-1", "0101101010101011000000011", false, 0, "1"},
		{"bound-False-compressed-0-urbb-8", "01011010101010110000000110010110", false, 0, "10010110"},
		{"bound-False-compressed-0-urbb-31", "0101101010101011000000011001011010010110100101101001011", false, 0, "1001011010010110100101101001011"},
		{"bound-False-compressed-1-urbb-0", "0101101010101011000000111", false, 1, ""},
		{"bound-False-compressed-1-urbb-1", "01011010101010110000001111", false, 1, "1"},
		{"bound-False-compressed-1-urbb-8", "010110101010101100000011110010110", false, 1, "10010110"},
		{"bound-False-compressed-1-urbb-31", "01011010101010110000001111001011010010110100101101001011", false, 1, "1001011010010110100101101001011"},
		{"bound-False-compressed-7-urbb-0", "0101101010101011000011111011001", false, 7, ""},
		{"bound-False-compressed-7-urbb-1", "01011010101010110000111110110011", false, 7, "1"},
		{"bound-False-compressed-7-urbb-8", "010110101010101100001111101100110010110", false, 7, "10010110"},
		{"bound-False-compressed-7-urbb-31", "01011010101010110000111110110011001011010010110100101101001011", false, 7, "1001011010010110100101101001011"},
		{"bound-False-compressed-8-urbb-0", "01011010101010110001000110110010", false, 8, ""},
		{"bound-False-compressed-8-urbb-1", "010110101010101100010001101100101", false, 8, "1"},
		{"bound-False-compressed-8-urbb-8", "0101101010101011000100011011001010010110", false, 8, "10010110"},
		{"bound-False-compressed-8-urbb-31", "010110101010101100010001101100101001011010010110100101101001011", false, 8, "1001011010010110100101101001011"},
		{"bound-False-compressed-127-urbb-0", "0101101010101011111111111011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001", false, 127, ""},
		{"bound-False-compressed-127-urbb-1", "01011010101010111111111110110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110011", false, 127, "1"},
		{"bound-False-compressed-127-urbb-8", "010110101010101111111111101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100110010110", false, 127, "10010110"},
		{"bound-False-compressed-127-urbb-31", "01011010101010111111111110110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110011001011010010110100101101001011", false, 127, "1001011010010110100101101001011"},
		{"bound-True-compressed--1-urbb-0", "100001111101101010101010", true, -1, ""},
		{"bound-True-compressed--1-urbb-1", "1000100001011010101010101", true, -1, "1"},
		{"bound-True-compressed--1-urbb-8", "10001011110110101010101010010110", true, -1, "10010110"},
		{"bound-True-compressed-0-urbb-0", "10001011110110101010101100000001", true, 0, ""},
		{"bound-True-compressed-0-urbb-1", "100011000101101010101011000000011", true, 0, "1"},
		{"bound-True-compressed-0-urbb-8", "1000111111011010101010110000000110010110", true, 0, "10010110"},
		{"bound-True-compressed-1-urbb-0", "100011000101101010101011000000111", true, 1, ""},
		{"bound-True-compressed-1-urbb-1", "1000110011011010101010110000001111", true, 1, "1"},
		{"bound-True-compressed-1-urbb-8", "10010000010110101010101100000011110010110", true, 1, "10010110"},
		{"bound-True-compressed-7-urbb-0", "100011110101101010101011000011111011001", true, 7, ""},
		{"bound-True-compressed-7-urbb-1", "1000111111011010101010110000111110110011", true, 7, "1"},
		{"bound-True-compressed-7-urbb-8", "10010011010110101010101100001111101100110010110", true, 7, "10010110"},
		{"bound-True-compressed-8-urbb-0", "1000111111011010101010110001000110110010", true, 8, ""},
		{"bound-True-compressed-8-urbb-1", "10010000010110101010101100010001101100101", true, 8, "1"},
		{"bound-True-compressed-8-urbb-8", "100100111101101010101011000100011011001010010110", true, 8, "10010110"},
		{"bound-True-compressed-127-urbb-0", "110010110101101010101011111111111011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001", true, 127, ""},
		{"bound-True-compressed-127-urbb-1", "1100101111011010101010111111111110110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110011", true, 127, "1"},
		{"bound-True-compressed-127-urbb-8", "11001111010110101010101111111111101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100101011001010110010101100110010110", true, 127, "10010110"},
	}
	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			r := clauseReader(t, v.bits)
			got, err := DecodeEGPRSAckNackDescriptionIEFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if r.Position() != len(v.bits) {
				t.Fatalf("consumed %d want %d", r.Position(), len(v.bits))
			}
			choice := got.EGPRSAckNackDescriptionStructChoice
			var body EGPRSAckNackDescriptionStruct
			if v.bounded {
				if choice.Alternative != 1 || choice.LengthL == nil || int(choice.LengthL.LengthL) != len(v.bits)-9 {
					t.Fatalf("wrong bounded arm: %+v", choice)
				}
				body = choice.LengthL.Content
			} else {
				if choice.Alternative != 0 || choice.EGPRSAckNackDescriptionStruct == nil {
					t.Fatalf("wrong remaining arm: %+v", choice)
				}
				body = choice.EGPRSAckNackDescriptionStruct.EGPRSAckNackDescriptionStruct
			}
			if body.FINALACKINDICATION != 1 || body.BEGINNINGOFWINDOW != 0 || body.ENDOFWINDOW != 1 || body.STARTINGSEQUENCENUMBER != 1365 {
				t.Fatalf("wrong typed header: %+v", body)
			}
			if v.compressed < 0 {
				if body.COMPRESSEDBITMAPLENGTHGroup != nil {
					t.Fatal("unexpected compressed bitmap")
				}
			} else {
				c := body.COMPRESSEDBITMAPLENGTHGroup
				if c == nil || int(c.COMPRESSEDBITMAPLENGTH) != v.compressed || c.COMPRESSEDBITMAPSTARTINGCOLORCODE != 1 || c.COMPRESSEDRECEIVEDBLOCKBITMAP.BitLength != v.compressed {
					t.Fatalf("wrong typed compressed bitmap: %+v", c)
				}
				want := strings.Repeat("10110010", (v.compressed+7)/8)[:v.compressed]
				if !bytes.Equal(c.COMPRESSEDRECEIVEDBLOCKBITMAP.Bytes, clausePack(want)) {
					t.Fatal("wrong compressed bitmap bits")
				}
			}
			var urbb strings.Builder
			for _, b := range body.UNCOMPRESSEDRECEIVEDBLOCKBITMAP {
				if b > 1 {
					t.Fatal("invalid bitmap bit")
				}
				urbb.WriteByte('0' + b)
			}
			if urbb.String() != v.urbb {
				t.Fatalf("URBB %q want %q", urbb.String(), v.urbb)
			}
			w := runtime.NewWriter()
			if err := EncodeEGPRSAckNackDescriptionIETo(w, got); err != nil {
				t.Fatal(err)
			}
			encoded, err := w.FinishBits(runtime.BitString{})
			if err != nil || encoded.BitLength != len(v.bits) || !bytes.Equal(encoded.Bytes, clausePack(v.bits)) {
				t.Fatalf("typed encode %x/%d: %v", encoded.Bytes, encoded.BitLength, err)
			}
			if len(v.bits)%8 == 0 {
				received, err := DecodeEGPRSAckNackDescriptionIE(clausePack(v.bits))
				if err != nil {
					t.Fatal(err)
				}
				replay, err := EncodeEGPRSAckNackDescriptionIE(received.Value)
				if err != nil || !bytes.Equal(replay, clausePack(v.bits)) {
					t.Fatalf("replay %x: %v", replay, err)
				}
				canonical, err := EncodeEGPRSAckNackDescriptionIECanonicalAtLength(received.Value, len(v.bits)/8)
				if err != nil || !bytes.Equal(canonical, clausePack(v.bits)) {
					t.Fatalf("canonical %x: %v", canonical, err)
				}
				edited := runtime.Canonical(received.Value)
				if v.bounded {
					edited.EGPRSAckNackDescriptionStructChoice.LengthL.Content.STARTINGSEQUENCENUMBER = 2047
				} else {
					edited.EGPRSAckNackDescriptionStructChoice.EGPRSAckNackDescriptionStruct.EGPRSAckNackDescriptionStruct.STARTINGSEQUENCENUMBER = 2047
				}
				changed, err := EncodeEGPRSAckNackDescriptionIECanonicalAtLength(edited, len(v.bits)/8)
				if err != nil {
					t.Fatal(err)
				}
				header := 1
				if v.bounded {
					header = 9
				}
				expected := v.bits[:header+3] + strings.Repeat("1", 11) + v.bits[header+14:]
				if !bytes.Equal(changed, clausePack(expected)) {
					t.Fatalf("edited %x want %x", changed, clausePack(expected))
				}
			}
			// A bounded IE may not read the following component, at any bit phase.
			for phase := 0; phase < 8; phase++ {
				bits := strings.Repeat("1", phase) + v.bits + "10101"
				framed := clauseReader(t, bits)
				if _, err := framed.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := framed.PushLimit(len(v.bits))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeEGPRSAckNackDescriptionIEFrom(framed); err != nil {
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
			minimum := 16
			if v.bounded {
				minimum = len(v.bits)
			} else if v.compressed >= 0 {
				minimum += 8 + v.compressed
			}
			for n := 0; n < minimum; n++ {
				if _, err := DecodeEGPRSAckNackDescriptionIEFrom(clauseReader(t, v.bits[:n])); err == nil {
					t.Fatalf("accepted prefix %d/%d", n, len(v.bits))
				}
			}
		})
	}
}

// TS 44.060 V19.0.0 §12.3.1 table 12.3.1.2, p22110: L is 15..255.
func TestClause12_3_1SourceOwnership(t *testing.T) {
	descriptors := Descriptors()
	if len(descriptors) != 2 {
		t.Fatalf("descriptor count %d want the root and its local struct", len(descriptors))
	}
	for _, d := range descriptors {
		if d.Standard != "TS 44.060" || d.Clause != "12.3.1" || strings.Contains(d.Name, "DLMC") {
			t.Fatalf("foreign component selected: %+v", d)
		}
	}
}

func TestClause12_3_1BoundaryValues(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		for _, ssn := range []int{0, 2047} {
			bits := fmt.Sprintf("0%03b%011b0", mask, ssn)
			got, err := DecodeEGPRSAckNackDescriptionIEFrom(clauseReader(t, bits))
			if err != nil {
				t.Fatal(err)
			}
			body := got.EGPRSAckNackDescriptionStructChoice.EGPRSAckNackDescriptionStruct.EGPRSAckNackDescriptionStruct
			if int(body.FINALACKINDICATION) != mask>>2 || int(body.BEGINNINGOFWINDOW) != (mask>>1)&1 || int(body.ENDOFWINDOW) != mask&1 || int(body.STARTINGSEQUENCENUMBER) != ssn {
				t.Fatalf("flags/SSN: %+v", body)
			}
		}
	}
	// 14 fixed header bits + 1 selector + 7 length + 1 colour + 127 CRBB
	// bits + 105 URBB bits = 255 bits, preceded by selector 1 and 8-bit L.
	bits := "1" + strings.Repeat("1", 8) + strings.Repeat("0", 14) + "1" + strings.Repeat("1", 7) + "0" + strings.Repeat("0", 127) + strings.Repeat("1", 105)
	got, err := DecodeEGPRSAckNackDescriptionIE(clausePack(bits))
	if err != nil {
		t.Fatal(err)
	}
	bounded := got.Value.EGPRSAckNackDescriptionStructChoice.LengthL
	if bounded == nil || bounded.LengthL != 255 || len(bounded.Content.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != 105 || bounded.Content.COMPRESSEDBITMAPLENGTHGroup.COMPRESSEDBITMAPSTARTINGCOLORCODE != 0 {
		t.Fatal("wrong maximum-length decomposition")
	}
	canonical, err := EncodeEGPRSAckNackDescriptionIECanonicalAtLength(got.Value, 33)
	if err != nil || !bytes.Equal(canonical, clausePack(bits)) {
		t.Fatalf("maximum canonical %x: %v", canonical, err)
	}
	for length := 0; length < 15; length++ {
		invalid := "1" + fmt.Sprintf("%08b", length) + strings.Repeat("0", length)
		if _, err := DecodeEGPRSAckNackDescriptionIEFrom(clauseReader(t, invalid)); err == nil {
			t.Fatalf("accepted L=%d below fixed header", length)
		}
	}
}

func TestClause12_3_1GrowingBitmap(t *testing.T) {
	for _, n := range []int{0, 1, 8, 255, 1024, 8192, 65536} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			bits := "0101101010101010" + strings.Repeat("1", n)
			got, err := DecodeEGPRSAckNackDescriptionIEFrom(clauseReader(t, bits))
			if err != nil {
				t.Fatal(err)
			}
			b := got.EGPRSAckNackDescriptionStructChoice.EGPRSAckNackDescriptionStruct.EGPRSAckNackDescriptionStruct.UNCOMPRESSEDRECEIVEDBLOCKBITMAP
			if len(b) != n || cap(b) > max(128, 2*n) {
				t.Fatalf("length/capacity %d/%d want %d", len(b), cap(b), n)
			}
		})
	}
}
func TestClause12_3_1InvalidValue(t *testing.T) {
	for _, name := range []string{"final", "beginning", "end", "sequence", "bitmap-bit", "compressed-length", "colour", "bitmap-width", "selector", "missing-arm"} {
		t.Run(name, func(t *testing.T) {
			got, err := DecodeEGPRSAckNackDescriptionIEFrom(clauseReader(t, "0101101010101010"))
			if err != nil {
				t.Fatal(err)
			}
			body := &got.EGPRSAckNackDescriptionStructChoice.EGPRSAckNackDescriptionStruct.EGPRSAckNackDescriptionStruct
			switch name {
			case "final":
				body.FINALACKINDICATION = 2
			case "beginning":
				body.BEGINNINGOFWINDOW = 2
			case "end":
				body.ENDOFWINDOW = 2
			case "sequence":
				body.STARTINGSEQUENCENUMBER = 2048
			case "bitmap-bit":
				body.UNCOMPRESSEDRECEIVEDBLOCKBITMAP = []uint8{2}
			case "compressed-length":
				body.COMPRESSEDBITMAPLENGTHGroup = &EGPRSAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHGroup{COMPRESSEDBITMAPLENGTH: 128}
			case "colour":
				body.COMPRESSEDBITMAPLENGTHGroup = &EGPRSAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHGroup{COMPRESSEDBITMAPSTARTINGCOLORCODE: 2}
			case "bitmap-width":
				body.COMPRESSEDBITMAPLENGTHGroup = &EGPRSAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHGroup{COMPRESSEDBITMAPLENGTH: 1}
			case "selector":
				got.EGPRSAckNackDescriptionStructChoice.Alternative = 2
			case "missing-arm":
				got.EGPRSAckNackDescriptionStructChoice.EGPRSAckNackDescriptionStruct = nil
			}
			if err := EncodeEGPRSAckNackDescriptionIETo(runtime.NewWriter(), got); err == nil {
				t.Fatal("accepted invalid typed field")
			}
		})
	}
}
func FuzzClause12_3_1(f *testing.F) {
	for _, s := range []string{"0101101010101010", "100001111101101010101010", "01011010101010101000000111"} {
		f.Add(clausePack(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := DecodeEGPRSAckNackDescriptionIE(data)
		if err == nil {
			replay, err := EncodeEGPRSAckNackDescriptionIE(got.Value)
			if err != nil || !bytes.Equal(data, replay) {
				t.Fatalf("replay %x: %v", replay, err)
			}
		}
		component, componentErr := DecodeEGPRSAckNackDescriptionStruct(data)
		if componentErr == nil {
			replay, encodeErr := EncodeEGPRSAckNackDescriptionStruct(component.Value)
			if encodeErr != nil || !bytes.Equal(data, replay) {
				t.Fatalf("component replay %x: %v", replay, encodeErr)
			}
		}
	})
}
