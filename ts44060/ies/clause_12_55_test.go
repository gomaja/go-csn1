package ies

import (
	"bytes"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

var clause1255Codec = clauseCodec[EUTRANCSGDescriptionIE]{DecodeEUTRANCSGDescriptionIE, DecodeEUTRANCSGDescriptionIEFrom, EncodeEUTRANCSGDescriptionIE, EncodeEUTRANCSGDescriptionIETo, EncodeEUTRANCSGDescriptionIECanonicalAtLength}
var clause1255Vectors = []clauseVector[EUTRANCSGDescriptionIE]{
	{name: "empty", hex: "00", bits: 1, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: nil}},
	{name: "empty-helper", hex: "80", bits: 6, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: nil}, EUTRANFREQUENCYINDEXList: nil}}}},
	{name: "codes", hex: "c01ffbfa3c", bits: 40, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: []PCIDGroupIEPCIDListEntry{{PCID: 0}, {PCID: 511}}, PCIDPatternLengthGroupList: nil, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 0}, {EUTRANFREQUENCYINDEX: 7}}}}}},
	{name: "pattern-0-0", hex: "a04490", bits: 22, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 0, PCIDPattern: runtime.BitString{Bytes: []byte{128}, BitLength: 1}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-0-1", hex: "bfc690", bits: 22, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 0, PCIDPattern: runtime.BitString{Bytes: []byte{128}, BitLength: 1}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-1-0", hex: "a04e48", bits: 23, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 1, PCIDPattern: runtime.BitString{Bytes: []byte{192}, BitLength: 2}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-1-1", hex: "bfcf48", bits: 23, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 1, PCIDPattern: runtime.BitString{Bytes: []byte{192}, BitLength: 2}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-2-0", hex: "a05724", bits: 24, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 2, PCIDPattern: runtime.BitString{Bytes: []byte{224}, BitLength: 3}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-2-1", hex: "bfd7a4", bits: 24, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 2, PCIDPattern: runtime.BitString{Bytes: []byte{224}, BitLength: 3}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-3-0", hex: "a05f9200", bits: 25, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 3, PCIDPattern: runtime.BitString{Bytes: []byte{240}, BitLength: 4}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-3-1", hex: "bfdfd200", bits: 25, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 3, PCIDPattern: runtime.BitString{Bytes: []byte{240}, BitLength: 4}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-4-0", hex: "a067c900", bits: 26, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 4, PCIDPattern: runtime.BitString{Bytes: []byte{248}, BitLength: 5}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-4-1", hex: "bfe7e900", bits: 26, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 4, PCIDPattern: runtime.BitString{Bytes: []byte{248}, BitLength: 5}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-5-0", hex: "a06fe480", bits: 27, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 5, PCIDPattern: runtime.BitString{Bytes: []byte{252}, BitLength: 6}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-5-1", hex: "bfeff480", bits: 27, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 5, PCIDPattern: runtime.BitString{Bytes: []byte{252}, BitLength: 6}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-6-0", hex: "a077f240", bits: 28, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 6, PCIDPattern: runtime.BitString{Bytes: []byte{254}, BitLength: 7}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-6-1", hex: "bff7fa40", bits: 28, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 6, PCIDPattern: runtime.BitString{Bytes: []byte{254}, BitLength: 7}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-7-0", hex: "a07ff920", bits: 29, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 7, PCIDPattern: runtime.BitString{Bytes: []byte{255}, BitLength: 8}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(0)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-7-1", hex: "bffffd20", bits: 29, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: nil, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 7, PCIDPattern: runtime.BitString{Bytes: []byte{255}, BitLength: 8}, PCIDPatternSense: 1}}, PCIDBITMAPGROUP: csgBitmapGroup(63)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 1}}}}}},
	{name: "two-groups", hex: "c54559ffe015544f80", bits: 67, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDList: []PCIDGroupIEPCIDListEntry{{PCID: 42}}, PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 2, PCIDPattern: runtime.BitString{Bytes: []byte{160}, BitLength: 3}, PCIDPatternSense: 1}}}, EUTRANFREQUENCYINDEXList: nil}, {CSGPCISPLIT: PCIDGroupIE{PCIDList: []PCIDGroupIEPCIDListEntry{{PCID: 511}, {PCID: 1}}, PCIDPatternLengthGroupList: nil, PCIDBITMAPGROUP: csgBitmapGroup(21)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 0}, {EUTRANFREQUENCYINDEX: 1}, {EUTRANFREQUENCYINDEX: 7}}}}}},
}

// TS 44.060 V19.0.0 §12.55 table 12.55.1, DOCX paragraphs 25423–25425.
// Synthetic field concatenations follow the printed widths and selectors.
func TestClause12_55(t *testing.T) {
	for _, vector := range clause1255Vectors {
		t.Run(vector.name, func(t *testing.T) {
			checkClauseVector(t, clause1255Codec, vector, vector.bits)
			d, err := clause1255Codec.decode(clauseBytes(t, vector.hex))
			if err != nil || d.BitsConsumed != vector.bits {
				t.Fatalf("consumed %d want %d: %v", d.BitsConsumed, vector.bits, err)
			}
		})
	}
	checkClauseInvalid(t, clause1255Codec, EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 8}}}}}, 32)
}

// An edit changes exactly the cited scalar bit and retains the received tail.
func TestClause12_55Edit(t *testing.T) {
	vector := clause1255Vectors[2]
	wire := clauseBytes(t, vector.hex)
	d, err := clause1255Codec.decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	d.Value.CSGPCISPLITGroupList[0].EUTRANFREQUENCYINDEXList[0].EUTRANFREQUENCYINDEX ^= 1
	want := bytes.Clone(wire)
	want[33/8] ^= 1 << uint(7-33%8)
	got, err := clause1255Codec.encode(d.Value)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("edit %x want %x: %v", got, want, err)
	}
	fresh, err := clause1255Codec.decode(got)
	if err != nil || !runtime.SemanticallyEqual(fresh.Value, d.Value) {
		t.Fatalf("edited typed value lost: %v", err)
	}
}

// Empty helper and index lists give 5 bits per group plus one terminator
// (TS 44.060 §§12.55, 12.57). Storage allocation count is bounded linearly.
func TestClause12_55Growth(t *testing.T) {
	for _, n := range []int{1, 16, 64, 256} {
		value := EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: make([]EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry, n)}
		wire, err := clause1255Codec.encode(value)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := clause1255Codec.decode(wire)
		if err != nil || len(decoded.Value.CSGPCISPLITGroupList) != n || decoded.BitsConsumed != 5*n+1 {
			t.Fatalf("growth n=%d bits=%d: %v", n, decoded.BitsConsumed, err)
		}
		if len(wire) != (5*n+8)/8 {
			t.Fatal("nonlinear encoded length")
		}
		allocations := testing.AllocsPerRun(3, func() {
			if _, err := clause1255Codec.decode(wire); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && allocations > float64(32*n+128) {
			t.Fatalf("allocations %.0f exceed linear bound for %d entries", allocations, n)
		}
	}
}
func FuzzClause12_55Replay(f *testing.F) {
	for _, v := range clause1255Vectors {
		f.Add(clauseBytes(f, v.hex))
	}
	f.Add(clauseBytes(f, "afc3f4a478"))
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		d, err := clause1255Codec.decode(wire)
		if err != nil {
			return
		}
		got, err := clause1255Codec.encode(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}

func csgBitmapGroup(value uint8) *uint8 { return &value }

// TS 44.060 V19.0.0 §12.55 and §12.57: each repeated pattern owns its width.
func TestClause12_55RepeatedPatterns(t *testing.T) {
	vector := clauseVector[EUTRANCSGDescriptionIE]{name: "independent-pattern-widths", hex: "afc3f4a478", bits: 39, value: EUTRANCSGDescriptionIE{CSGPCISPLITGroupList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntry{{CSGPCISPLIT: PCIDGroupIE{PCIDPatternLengthGroupList: []PCIDGroupIEPCIDPatternLengthGroupListEntry{{PCIDPatternLength: 0, PCIDPattern: runtime.BitString{Bytes: []byte{0}, BitLength: 1}, PCIDPatternSense: 1}, {PCIDPatternLength: 7, PCIDPattern: runtime.BitString{Bytes: []byte{0xa5}, BitLength: 8}, PCIDPatternSense: 0}}, PCIDBITMAPGROUP: csgBitmapGroup(31)}, EUTRANFREQUENCYINDEXList: []EUTRANCSGDescriptionIECSGPCISPLITGroupListEntryEUTRANFREQUENCYINDEXListEntry{{EUTRANFREQUENCYINDEX: 0}, {EUTRANFREQUENCYINDEX: 7}}}}}}
	checkClauseVector(t, clause1255Codec, vector, vector.bits)
}
