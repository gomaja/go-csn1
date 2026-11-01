package ies

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var globalTimingCodec = clauseCodec[GlobalPacketTimingAdvanceIE]{DecodeGlobalPacketTimingAdvanceIE, DecodeGlobalPacketTimingAdvanceIEFrom, EncodeGlobalPacketTimingAdvanceIE, EncodeGlobalPacketTimingAdvanceIETo, EncodeGlobalPacketTimingAdvanceIECanonicalAtLength}

// TS 44.060 V19.0.0 Table 12.12a.1, paragraphs 23156–23161:
// three independent optional fields, with widths 6, 4+3 and 4+3.
func TestGlobalPacketTimingAdvance(t *testing.T) {
	for mask := 0; mask < 8; mask++ {
		for _, value := range []uint8{0, 1, 21, 63} {
			t.Run(fmt.Sprintf("options-%d-value-%d", mask, value), func(t *testing.T) {
				var expected GlobalPacketTimingAdvanceIE
				var bits strings.Builder
				if mask&1 != 0 {
					expected.TIMINGADVANCEVALUE = &value
					fmt.Fprintf(&bits, "1%06b", value)
				} else {
					bits.WriteByte('0')
				}
				if mask&2 != 0 {
					expected.UPLINKTIMINGADVANCEINDEXGroup = &GlobalPacketTimingAdvanceIEUPLINKTIMINGADVANCEINDEXGroup{UPLINKTIMINGADVANCEINDEX: value & 15, UPLINKTIMINGADVANCETIMESLOTNUMBER: value & 7}
					fmt.Fprintf(&bits, "1%04b%03b", value&15, value&7)
				} else {
					bits.WriteByte('0')
				}
				if mask&4 != 0 {
					expected.DOWNLINKTIMINGADVANCEINDEXGroup = &GlobalPacketTimingAdvanceIEDOWNLINKTIMINGADVANCEINDEXGroup{DOWNLINKTIMINGADVANCEINDEX: (value + 1) & 15, DOWNLINKTIMINGADVANCETIMESLOTNUMBER: (value + 1) & 7}
					fmt.Fprintf(&bits, "1%04b%03b", (value+1)&15, (value+1)&7)
				} else {
					bits.WriteByte('0')
				}
				wire := clausePack(bits.String())
				checkClauseVector(t, globalTimingCodec, clauseVector[GlobalPacketTimingAdvanceIE]{hex: hex.EncodeToString(wire), bits: bits.Len(), value: expected}, bits.Len())
				decoded, err := DecodeGlobalPacketTimingAdvanceIE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if decoded.BitsConsumed != bits.Len() {
					t.Fatalf("consumed %d want %d", decoded.BitsConsumed, bits.Len())
				}
				canonical, err := EncodeGlobalPacketTimingAdvanceIECanonical(decoded.Value)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("source-bounded canonical %x want %x: %v", canonical, wire, err)
				}
				edited := decoded.Value
				replacement := uint8(42)
				edited.TIMINGADVANCEVALUE = &replacement
				editedBits := bits.Len()
				if mask&1 == 0 {
					editedBits += 6
				}
				output, err := EncodeGlobalPacketTimingAdvanceIECanonicalAtLength(edited, (editedBits+7)/8)
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeGlobalPacketTimingAdvanceIE(output)
				if err != nil || again.Value.TIMINGADVANCEVALUE == nil || *again.Value.TIMINGADVANCEVALUE != 42 {
					t.Fatalf("typed edit: %+v %v", again, err)
				}
			})
		}
	}
	tooLarge := uint8(64)
	checkClauseInvalid(t, globalTimingCodec, GlobalPacketTimingAdvanceIE{TIMINGADVANCEVALUE: &tooLarge}, 3)
	checkClauseInvalid(t, globalTimingCodec, GlobalPacketTimingAdvanceIE{UPLINKTIMINGADVANCEINDEXGroup: &GlobalPacketTimingAdvanceIEUPLINKTIMINGADVANCEINDEXGroup{UPLINKTIMINGADVANCEINDEX: 16}}, 3)
	checkClauseInvalid(t, globalTimingCodec, GlobalPacketTimingAdvanceIE{DOWNLINKTIMINGADVANCEINDEXGroup: &GlobalPacketTimingAdvanceIEDOWNLINKTIMINGADVANCEINDEXGroup{DOWNLINKTIMINGADVANCETIMESLOTNUMBER: 8}}, 3)
}

func FuzzBasicContextIEReplay(f *testing.F) {
	roots := []struct{ clause, name string }{{"12.12a", "Global Packet Timing Advance IE"}, {"12.26", "Extension Bits IE"}, {"12.52", "3G Priority Parameters IE"}, {"12.52", "Repeated UTRAN Priority Parameters struct"}}
	descriptors := make([]runtime.Descriptor, len(roots))
	for i, root := range roots {
		d, err := LookupClause(root.clause, root.name)
		if err != nil {
			f.Fatal(err)
		}
		descriptors[i] = d
		f.Add(uint8(i), []byte{0}, uint16(0))
		f.Add(uint8(i), []byte{0xff, 0xff, 0xff, 0xff, 0}, uint16(17))
	}
	_, priorityBits := priorityParametersVector(true, 7, 3)
	f.Add(uint8(2), clausePack(priorityBits), uint16(len(priorityBits)))
	f.Add(uint8(3), []byte{0, 0}, uint16(9))
	f.Fuzz(func(t *testing.T, index uint8, wire []byte, limit uint16) {
		if len(wire) > 512 {
			return
		}
		d := descriptors[int(index)%len(descriptors)]
		v, err := d.Decode(wire)
		if err == nil {
			out, e := d.Encode(v)
			if e != nil || !bytes.Equal(out, wire) {
				t.Fatalf("replay %x want %x: %v", out, wire, e)
			}
		}
		r := runtime.NewReader(wire)
		width := int(limit) % (len(wire)*8 + 1)
		if _, err := r.PushLimit(width); err != nil {
			t.Fatal(err)
		}
		_, _ = d.DecodeFrom(r)
		if r.Position() > width {
			t.Fatalf("crossed limit %d", width)
		}
	})
}
