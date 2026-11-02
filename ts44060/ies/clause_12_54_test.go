package ies

import (
	"bytes"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

var clause1254Codec = clauseCodec[N3GCSGDescriptionIE]{DecodeN3GCSGDescriptionIE, DecodeN3GCSGDescriptionIEFrom, EncodeN3GCSGDescriptionIE, EncodeN3GCSGDescriptionIETo, EncodeN3GCSGDescriptionIECanonicalAtLength}
var clause1254Vectors = []clauseVector[N3GCSGDescriptionIE]{
	{name: "empty", hex: "00", bits: 1, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: nil}},
	{name: "empty-helper", hex: "80", bits: 5, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: nil}, UTRANFREQUENCYINDEXList: nil}}}},
	{name: "codes", hex: "c01ff907e0", bits: 37, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: []PSCGroupIEPSCListEntry{{PSC: 0}, {PSC: 511}}, PSCPatternLengthGroupList: nil}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 0}, {UTRANFREQUENCYINDEX: 31}}}}}},
	{name: "pattern-0-0", hex: "a24200", bits: 17, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 0, PSCPattern: runtime.BitString{Bytes: []byte{128}, BitLength: 1}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-0-1", hex: "a34200", bits: 17, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 0, PSCPattern: runtime.BitString{Bytes: []byte{128}, BitLength: 1}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-1-0", hex: "a72100", bits: 18, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 1, PSCPattern: runtime.BitString{Bytes: []byte{192}, BitLength: 2}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-1-1", hex: "a7a100", bits: 18, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 1, PSCPattern: runtime.BitString{Bytes: []byte{192}, BitLength: 2}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-2-0", hex: "ab9080", bits: 19, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 2, PSCPattern: runtime.BitString{Bytes: []byte{224}, BitLength: 3}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-2-1", hex: "abd080", bits: 19, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 2, PSCPattern: runtime.BitString{Bytes: []byte{224}, BitLength: 3}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-3-0", hex: "afc840", bits: 20, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 3, PSCPattern: runtime.BitString{Bytes: []byte{240}, BitLength: 4}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-3-1", hex: "afe840", bits: 20, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 3, PSCPattern: runtime.BitString{Bytes: []byte{240}, BitLength: 4}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-4-0", hex: "b3e420", bits: 21, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 4, PSCPattern: runtime.BitString{Bytes: []byte{248}, BitLength: 5}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-4-1", hex: "b3f420", bits: 21, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 4, PSCPattern: runtime.BitString{Bytes: []byte{248}, BitLength: 5}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-5-0", hex: "b7f210", bits: 22, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 5, PSCPattern: runtime.BitString{Bytes: []byte{252}, BitLength: 6}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-5-1", hex: "b7fa10", bits: 22, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 5, PSCPattern: runtime.BitString{Bytes: []byte{252}, BitLength: 6}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-6-0", hex: "bbf908", bits: 23, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 6, PSCPattern: runtime.BitString{Bytes: []byte{254}, BitLength: 7}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-6-1", hex: "bbfd08", bits: 23, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 6, PSCPattern: runtime.BitString{Bytes: []byte{254}, BitLength: 7}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-7-0", hex: "bffc84", bits: 24, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 7, PSCPattern: runtime.BitString{Bytes: []byte{255}, BitLength: 8}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "pattern-7-1", hex: "bffe84", bits: 24, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: nil, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 7, PSCPattern: runtime.BitString{Bytes: []byte{255}, BitLength: 8}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 1}}}}}},
	{name: "two-groups", hex: "c54ab3ffc02410fe00", bits: 65, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCList: []PSCGroupIEPSCListEntry{{PSC: 42}}, PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 2, PSCPattern: runtime.BitString{Bytes: []byte{160}, BitLength: 3}, PSCPatternSense: 1}}}, UTRANFREQUENCYINDEXList: nil}, {CSGPSCSPLIT: PSCGroupIE{PSCList: []PSCGroupIEPSCListEntry{{PSC: 511}, {PSC: 1}}, PSCPatternLengthGroupList: nil}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 0}, {UTRANFREQUENCYINDEX: 1}, {UTRANFREQUENCYINDEX: 31}}}}}},
}

// TS 44.060 V19.0.0 §12.54 table 12.54.1, DOCX paragraphs 25411–25413.
// Synthetic field concatenations follow the printed widths and selectors.
func TestClause12_54(t *testing.T) {
	for _, vector := range clause1254Vectors {
		t.Run(vector.name, func(t *testing.T) {
			checkClauseVector(t, clause1254Codec, vector, vector.bits)
			d, err := clause1254Codec.decode(clauseBytes(t, vector.hex))
			if err != nil || d.BitsConsumed != vector.bits {
				t.Fatalf("consumed %d want %d: %v", d.BitsConsumed, vector.bits, err)
			}
		})
	}
	checkClauseInvalid(t, clause1254Codec, N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 32}}}}}, 32)
}

// An edit changes exactly the cited scalar bit and retains the received tail.
func TestClause12_54Edit(t *testing.T) {
	vector := clause1254Vectors[2]
	wire := clauseBytes(t, vector.hex)
	d, err := clause1254Codec.decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	d.Value.CSGPSCSPLITGroupList[0].UTRANFREQUENCYINDEXList[0].UTRANFREQUENCYINDEX ^= 1
	want := bytes.Clone(wire)
	want[28/8] ^= 1 << uint(7-28%8)
	got, err := clause1254Codec.encode(d.Value)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("edit %x want %x: %v", got, want, err)
	}
	fresh, err := clause1254Codec.decode(got)
	if err != nil || !runtime.SemanticallyEqual(fresh.Value, d.Value) {
		t.Fatalf("edited typed value lost: %v", err)
	}
}

// Empty helper and index lists give 4 bits per group plus one terminator
// (TS 44.060 §§12.54, 12.58). Storage allocation count is bounded linearly.
func TestClause12_54Growth(t *testing.T) {
	for _, n := range []int{1, 16, 64, 256} {
		value := N3GCSGDescriptionIE{CSGPSCSPLITGroupList: make([]N3GCSGDescriptionIECSGPSCSPLITGroupListEntry, n)}
		wire, err := clause1254Codec.encode(value)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := clause1254Codec.decode(wire)
		if err != nil || len(decoded.Value.CSGPSCSPLITGroupList) != n || decoded.BitsConsumed != 4*n+1 {
			t.Fatalf("growth n=%d bits=%d: %v", n, decoded.BitsConsumed, err)
		}
		if len(wire) != (4*n+8)/8 {
			t.Fatal("nonlinear encoded length")
		}
		allocations := testing.AllocsPerRun(3, func() {
			if _, err := clause1254Codec.decode(wire); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && allocations > float64(32*n+128) {
			t.Fatalf("allocations %.0f exceed linear bound for %d entries", allocations, n)
		}
	}
}
func FuzzClause12_54Replay(f *testing.F) {
	for _, v := range clause1254Vectors {
		f.Add(clauseBytes(f, v.hex))
	}
	f.Add(clauseBytes(f, "a1fa520fc0"))
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		d, err := clause1254Codec.decode(wire)
		if err != nil {
			return
		}
		got, err := clause1254Codec.encode(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}

// TS 44.060 V19.0.0 §12.54 and §12.58: each repeated pattern owns its width.
func TestClause12_54RepeatedPatterns(t *testing.T) {
	vector := clauseVector[N3GCSGDescriptionIE]{name: "independent-pattern-widths", hex: "a1fa520fc0", bits: 36, value: N3GCSGDescriptionIE{CSGPSCSPLITGroupList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntry{{CSGPSCSPLIT: PSCGroupIE{PSCPatternLengthGroupList: []PSCGroupIEPSCPatternLengthGroupListEntry{{PSCPatternLength: 0, PSCPattern: runtime.BitString{Bytes: []byte{0}, BitLength: 1}, PSCPatternSense: 1}, {PSCPatternLength: 7, PSCPattern: runtime.BitString{Bytes: []byte{0xa5}, BitLength: 8}, PSCPatternSense: 0}}}, UTRANFREQUENCYINDEXList: []N3GCSGDescriptionIECSGPSCSPLITGroupListEntryUTRANFREQUENCYINDEXListEntry{{UTRANFREQUENCYINDEX: 0}, {UTRANFREQUENCYINDEX: 31}}}}}}
	checkClauseVector(t, clause1254Codec, vector, vector.bits)
}
