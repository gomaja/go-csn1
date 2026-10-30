package ies

import (
	"bytes"
	"fmt"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var clause1232Codec = clauseCodec[FLOAckNackDescriptionIE]{DecodeFLOAckNackDescriptionIE, DecodeFLOAckNackDescriptionIEFrom, EncodeFLOAckNackDescriptionIE, EncodeFLOAckNackDescriptionIETo, EncodeFLOAckNackDescriptionIECanonicalAtLength}
var clause1232Vectors = []clauseVector[FLOAckNackDescriptionIE]{
	{name: "flo-compressed-0", hex: "000000", bits: 21, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 0, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 0, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 0, COMPRESSEDBITMAPSTARTINGCOLORCODE: 0, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{}, BitLength: 0}}}}}},
	{name: "flo-compressed-1", hex: "bff014", bits: 22, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 1, COMPRESSEDBITMAPSTARTINGCOLORCODE: 0, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{128}, BitLength: 1}}}}}},
	{name: "flo-compressed-7", hex: "fff07da0", bits: 28, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 1, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 7, COMPRESSEDBITMAPSTARTINGCOLORCODE: 1, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180}, BitLength: 7}}}}}},
	{name: "flo-compressed-8", hex: "3ff085a0", bits: 29, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 0, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 8, COMPRESSEDBITMAPSTARTINGCOLORCODE: 0, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180}, BitLength: 8}}}}}},
	{name: "flo-compressed-63", hex: "fff3fda5a5a5a5a5a5a5a0", bits: 84, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 1, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 63, COMPRESSEDBITMAPSTARTINGCOLORCODE: 1, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180, 180, 180, 180, 180, 180, 180, 180}, BitLength: 63}}}}}},
	{name: "flo-compressed-64", hex: "3ff405a5a5a5a5a5a5a5a0", bits: 85, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 0, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 64, COMPRESSEDBITMAPSTARTINGCOLORCODE: 0, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180, 180, 180, 180, 180, 180, 180, 180}, BitLength: 64}}}}}},
	{name: "flo-compressed-65", hex: "bff415a5a5a5a5a5a5a5a4", bits: 86, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 65, COMPRESSEDBITMAPSTARTINGCOLORCODE: 0, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180, 180, 180, 180, 180, 180, 180, 180, 128}, BitLength: 65}}}}}},
	{name: "flo-compressed-127", hex: "fff7fda5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a0", bits: 148, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 1, STARTINGSEQUENCENUMBER: 1023, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 0, COMPRESSEDBITMAPLENGTH: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceCOMPRESSEDBITMAPLENGTH{COMPRESSEDBITMAPLENGTH: 127, COMPRESSEDBITMAPSTARTINGCOLORCODE: 1, COMPRESSEDRECEIVEDBLOCKBITMAP: runtime.BitString{Bytes: []byte{180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180, 180}, BitLength: 127}}}}}},
	{name: "flo-uncompressed-0", hex: "9558", bits: 13, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: nil}}}}},
	{name: "flo-uncompressed-1", hex: "955c", bits: 14, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1}}}}}},
	{name: "flo-uncompressed-3", hex: "955d", bits: 16, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1}}}}}},
	{name: "flo-uncompressed-7", hex: "955da0", bits: 20, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1, 1, 0, 1, 0}}}}}},
	{name: "flo-uncompressed-8", hex: "955da0", bits: 21, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1, 1, 0, 1, 0, 0}}}}}},
	{name: "flo-uncompressed-9", hex: "955da4", bits: 22, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1, 1, 0, 1, 0, 0, 1}}}}}},
	{name: "flo-uncompressed-64", hex: "955da5a5a5a5a5a5a5a0", bits: 77, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0}}}}}},
	{name: "flo-uncompressed-127", hex: "955da5a5a5a5a5a5a5a5a5a5a5a5a5a5a5a0", bits: 140, value: FLOAckNackDescriptionIE{FLOAckNackDescriptionStruct: FLOAckNackDescriptionStruct{BEGINNINGOFWINDOW: 1, ENDOFWINDOW: 0, STARTINGSEQUENCENUMBER: 341, COMPRESSEDBITMAPLENGTHChoice: FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoice{Alternative: 1, UNCOMPRESSEDRECEIVEDBLOCKBITMAP: &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 0}}}}}},
}

// TS 44.060 V19.0.0 §12.3.2: exact printed field widths and all selectors.
func TestClause12_3_2(t *testing.T) {
	for _, vector := range clause1232Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			if vector.value.FLOAckNackDescriptionStruct.COMPRESSEDBITMAPLENGTHChoice.Alternative == 1 {
				minimum = 13
			}
			checkClauseVector(t, clause1232Codec, vector, minimum)
		})
	}
}

func TestClause12_3_2InvalidFields(t *testing.T) {
	// TS 44.060 V19.0.0 §12.3.2 paragraphs 22136–22142.
	for _, field := range []string{"beginning", "end", "sequence", "selector", "missing-compressed", "length", "color", "bitmap-length", "missing-uncompressed", "bitmap-bit"} {
		t.Run(field, func(t *testing.T) {
			value := runtime.Canonical(clause1232Vectors[1].value)
			body := &value.FLOAckNackDescriptionStruct
			choice := &body.COMPRESSEDBITMAPLENGTHChoice
			switch field {
			case "beginning":
				body.BEGINNINGOFWINDOW = 2
			case "end":
				body.ENDOFWINDOW = 2
			case "sequence":
				body.STARTINGSEQUENCENUMBER = 1024
			case "selector":
				choice.Alternative = 2
			case "missing-compressed":
				choice.COMPRESSEDBITMAPLENGTH = nil
			case "length":
				choice.COMPRESSEDBITMAPLENGTH.COMPRESSEDBITMAPLENGTH = 128
			case "color":
				choice.COMPRESSEDBITMAPLENGTH.COMPRESSEDBITMAPSTARTINGCOLORCODE = 2
			case "bitmap-length":
				choice.COMPRESSEDBITMAPLENGTH.COMPRESSEDBITMAPLENGTH = 2
			case "missing-uncompressed":
				choice.Alternative = 1
				choice.COMPRESSEDBITMAPLENGTH = nil
			case "bitmap-bit":
				choice.Alternative = 1
				choice.COMPRESSEDBITMAPLENGTH = nil
				choice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP = &FLOAckNackDescriptionStructCOMPRESSEDBITMAPLENGTHChoiceUNCOMPRESSEDRECEIVEDBLOCKBITMAP{UNCOMPRESSEDRECEIVEDBLOCKBITMAP: []uint8{2}}
			}
			checkClauseInvalid(t, clause1232Codec, value, 3)
		})
	}
}

func TestClause12_3_2EnclosingBitmapExtent(t *testing.T) {
	// TS 44.060 V19.0.0 §12.3.2 paragraph 22142: bit** terminates
	// at the enclosing FLO extent, including zero-bit and long bitmaps.
	for _, n := range []int{0, 1, 7, 8, 127, 1024, 4096} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			bitmap := strings.Repeat("10", (n+1)/2)[:n]
			payload := "1001010101011" + bitmap
			framed := clausePack("101" + payload + "10101")
			r := runtime.NewReader(framed)
			if _, err := r.ReadUint(3); err != nil {
				t.Fatal(err)
			}
			outer, err := r.PushLimit(len(payload))
			if err != nil {
				t.Fatal(err)
			}
			value, err := DecodeFLOAckNackDescriptionIEFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			body := value.FLOAckNackDescriptionStruct
			if body.BEGINNINGOFWINDOW != 1 || body.ENDOFWINDOW != 0 || body.STARTINGSEQUENCENUMBER != 341 || body.COMPRESSEDBITMAPLENGTHChoice.Alternative != 1 {
				t.Fatal("wrong enclosing header")
			}
			got := body.COMPRESSEDBITMAPLENGTHChoice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP
			if got == nil || len(got.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != n {
				t.Fatalf("bitmap length: %#v want %d", got, n)
			}
			for i, b := range got.UNCOMPRESSEDRECEIVEDBLOCKBITMAP {
				if b != uint8(bitmap[i]-'0') {
					t.Fatalf("bitmap[%d]=%d", i, b)
				}
			}
			if r.Remaining() != 0 {
				t.Fatalf("enclosing bits left: %d", r.Remaining())
			}
			if err := r.PopLimit(outer); err != nil {
				t.Fatal(err)
			}
			marker, err := r.ReadUint(5)
			if err != nil || marker != 21 {
				t.Fatalf("enclosing sentinel %d: %v", marker, err)
			}
			w := runtime.NewWriter()
			if err := w.WriteUint(5, 3); err != nil {
				t.Fatal(err)
			}
			bound, err := w.PushLimit(len(payload))
			if err != nil {
				t.Fatal(err)
			}
			if err := EncodeFLOAckNackDescriptionIETo(w, runtime.Canonical(value)); err != nil {
				t.Fatal(err)
			}
			if err := w.PopLimit(bound); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(21, 5); err != nil {
				t.Fatal(err)
			}
			output, err := w.Finish(runtime.BitString{})
			if err != nil || !bytes.Equal(output, framed) {
				t.Fatalf("bounded canonical differs: %x: %v", output, err)
			}
			// The fixed empty enclosing extent is valid; extending it adds bitmap
			// bits, rather than treating them as padding or a following field.
			if n > 0 {
				short := runtime.NewReader(clausePack(payload))
				if _, err := short.PushLimit(len(payload) - 1); err != nil {
					t.Fatal(err)
				}
				decoded, err := DecodeFLOAckNackDescriptionIEFrom(short)
				if err != nil {
					t.Fatal(err)
				}
				if len(decoded.FLOAckNackDescriptionStruct.COMPRESSEDBITMAPLENGTHChoice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != n-1 {
					t.Fatal("short extent did not bound the bitmap")
				}
			}
		})
	}
}

func TestClause12_3_2BitmapAllocationGrowth(t *testing.T) {
	// The bit** list in TS 44.060 V19.0.0 §12.3.2 paragraph 22142
	// must stay within a linear allocation envelope as its enclosing extent grows.
	for _, n := range []int{128, 512, 2048, 8192} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			wire := clausePack("1001010101011" + strings.Repeat("1", n))
			goruntime.GC()
			var before, after goruntime.MemStats
			goruntime.ReadMemStats(&before)
			for range 16 {
				reader := runtime.NewReader(wire)
				if _, err := reader.PushLimit(13 + n); err != nil {
					t.Fatal(err)
				}
				value, err := DecodeFLOAckNackDescriptionIEFrom(reader)
				if err != nil {
					t.Fatal(err)
				}
				if len(value.FLOAckNackDescriptionStruct.COMPRESSEDBITMAPLENGTHChoice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != n {
					t.Fatal("incorrect growing bitmap length")
				}
				goruntime.KeepAlive(value)
			}
			goruntime.ReadMemStats(&after)
			perValue := (after.TotalAlloc - before.TotalAlloc) / 16
			if bound := uint64(65536 + 128*n); perValue > bound {
				t.Fatalf("allocation %d bytes/value exceeds linear bound %d for %d bits", perValue, bound, n)
			}
		})
	}
}
