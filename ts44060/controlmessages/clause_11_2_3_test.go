package controlmessages_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func cellChangeFailureFields() string {
	return fmt.Sprintf("%032b%010b%06b%04b", uint32(0x10203040), 511, 42, 7)
}
func cellChangeFailureRelease(which int, detail bool) string {
	optional := func(n int, bits string) string {
		if which == n || which == 7 && n < 2 {
			return "1" + bits
		}
		return "0"
	}
	band := "0"
	if detail {
		band = "1101"
	}
	fdd := fmt.Sprintf("%014b1", 1234) + band + fmt.Sprintf("%09b", 301)
	tdd := fmt.Sprintf("%014b1", 4321) + band + fmt.Sprintf("%07b1", 85)
	eutran := fmt.Sprintf("%016b", 45678) + band + fmt.Sprintf("%09b", 321)
	plmn := "0"
	if detail {
		plmn = fmt.Sprintf("1%012b%012b", 0x123, 0x456)
	}
	utcsg := fmt.Sprintf("%028b", 0x1234567) + plmn
	etcsg := fmt.Sprintf("%028b%016b", 0xabcdef1, 0xbeef) + plmn
	ext := fmt.Sprintf("%018b", 0x2abcd) + band + fmt.Sprintf("%09b", 211)
	return "1" + optional(0, fdd) + optional(1, tdd) + "1" + optional(2, "1010") + "1" + optional(3, eutran) + "1" + optional(4, utcsg) + optional(5, etcsg) + "1" + optional(6, ext)
}

// TS 44.060 V19.0.0 §11.2.3 paragraphs 13604–13627;
// uplink selector 000000 is external (§11.2.0.2 paragraph 13424).
// §§12.31/32/49/62/63/68 define the target widths independently of this root.
func TestClause1123CellChangeFailure(t *testing.T) {
	for which := -1; which < 8; which++ {
		for _, detail := range []bool{false, true} {
			raw := simpleControlBits("01000000000000"+cellChangeFailureFields()+cellChangeFailureRelease(which, detail), 184)
			r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 0)
			v, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(r)
			if e != nil || v.TLLIGRNTI != 0x10203040 || v.ARFCN != 511 || v.BSIC != 42 || v.CAUSE != 7 || r.Position() != 184 {
				t.Fatalf("target %d detail %t typed %+v at %d: %v", which, detail, v, r.Position(), e)
			}
			for _, field := range []struct {
				index int
				name  string
				value uint64
			}{{0, "FDDARFCN", 1234}, {1, "TDDARFCN", 4321}, {2, "GRNTIExtension", 10}, {3, "EARFCN", 45678}, {4, "UTRANCI", 0x1234567}, {5, "EUTRANCI", 0xabcdef1}, {6, "EARFCNExtended", 0x2abcd}} {
				if field.index == which || which == 7 && field.index < 2 {
					simpleControlWant(t, v, field.name, field.value)
				} else {
					simpleControlWant(t, v, field.name)
				}
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, v)
			})
			cv := runtime.Canonical(v)
			out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, cv)
			})
			if !bytes.Equal(out.Bytes, raw) {
				t.Fatalf("canonical changed target %d", which)
			}
			cv.ARFCN ^= 1
			edited := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, cv)
			})
			er, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACUplink, 0)
			ev, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(er)
			if e != nil || ev.ARFCN != 510 || ev.TLLIGRNTI != v.TLLIGRNTI {
				t.Fatalf("edited target %+v: %v", ev, e)
			}
			simpleControlVector(t, "11.2.3", fmt.Sprintf("target%d-detail%t", which, detail), raw)
		}
	}
}
func TestClause1123ReleaseBoundaries(t *testing.T) {
	// The five release suffixes admit null at their exact containing boundary
	// and a zero followed by unspecified bits. Each is reached independently.
	for _, release := range []string{"", "100", "10010", "1001010", "1001010100"} {
		prefix := "01000000000000" + cellChangeFailureFields() + release
		end := len(prefix)
		raw := simpleControlBits(prefix, 184)
		r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 0)
		old, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(r)
		if e != nil || r.Position() != end {
			t.Fatalf("null release %q at %d: %v", release, r.Position(), e)
		}
		simpleControlReplay(t, raw, end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, v)
		})
		cv := runtime.Canonical(v)
		simpleControlCanonical(t, raw, 14, end, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, cv)
		})
		if e = r.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		sentinel, e := r.ReadUint(1)
		if e != nil || sentinel != 0 {
			t.Fatalf("sentinel after release %q: %d %v", release, sentinel, e)
		}
		ignored := simpleControlBits(prefix+"0"+strings.Repeat("1", 184-end-1), 184)
		ir, _ := contentReader(t, ignored, 184, runtime.RLCMACUplink, 0)
		iv, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(ir)
		if e != nil || iv.ARFCN != 511 || ir.Position() != 184 {
			t.Fatalf("ignored release %q at %d: %v", release, ir.Position(), e)
		}
		simpleControlReplay(t, ignored, 184, 14, ir, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, iv)
		})
	}
	raw := simpleControlBits("01000000000000"+cellChangeFailureFields()+"0", 184)
	for end := 14; end < 66; end++ {
		r, _ := runtime.NewChoiceReaderBits(raw, end, nil)
		_, _ = r.ReadUint(14)
		if _, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(r); e == nil {
			t.Fatalf("mandatory prefix truncated at %d", end)
		}
	}
}
func FuzzClause1123CellChangeFailure(f *testing.F) {
	f.Add(simpleControlBits("01000000000000"+cellChangeFailureFields()+cellChangeFailureRelease(7, true), 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, e := controlmessages.DecodePacketCellChangeFailureMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketCellChangeFailureMessageContentTo(w, v)
			})
		}
	})
}
