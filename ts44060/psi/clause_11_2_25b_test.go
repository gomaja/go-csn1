package psi_test

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/psi"
)

// TS 44.060 V19.0.0 §11.2.25b; exact extracted paragraph span is recorded in the source intake.
// TS 44.060 V19.0.0 §11.2.0.1 paragraph 13384: external selector 111110.
var psi15Codec = psiCodec{tag: 62, mandatory: 2,
	decode: func(r *runtime.Reader) (any, error) { return psi.DecodePSI15MessageContentFrom(r) },
	encode: func(w *runtime.Writer, v any) error {
		return psi.EncodePSI15MessageContentTo(w, v.(psi.PSI15MessageContent))
	},
	canonical: func(v any) any { return runtime.Canonical(v.(psi.PSI15MessageContent)) },
	canonicalAt: func(v any, n int) ([]byte, error) {
		return psi.EncodePSI15MessageContentCanonicalAtLength(v.(psi.PSI15MessageContent), n)
	},
	page: func(v any) (uint8, bool) {
		p := v.(psi.PSI15MessageContent).PAGEMODEGroupFallback.Known
		if p == nil {
			return 0, false
		}
		return p.PAGEMODE, true
	},
	edit: func(v any, page uint8) any {
		p := v.(psi.PSI15MessageContent)
		p.PAGEMODEGroupFallback.Known.PAGEMODE = page
		return p
	},
}
var psi15Vectors = []psiVector{
	{name: "page-0-fdd-None-tdd-None", body: "000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {0}}},
	{name: "page-0-fdd-[]-tdd-[]", body: "00100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {0}}},
	{name: "page-0-fdd-[0]-tdd-[]", body: "00110000000000000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0}, "pagemode": {0}}},
	{name: "page-0-fdd-[]-tdd-[16383]", body: "00101111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {0}, "tddarfcn": {16383}}},
	{name: "page-0-fdd-[1, 16383]-tdd-[0, 12345]", body: "00110000000000000111111111111111101000000000000001110000001110010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {1, 16383}, "pagemode": {0}, "tddarfcn": {0, 12345}}},
	{name: "page-0-fdd-[0, 1, 2, 3, 4]-tdd-[12, 8192, 16383]", body: "00110000000000000010000000000000110000000000001010000000000001110000000000010001000000000011001100000000000001111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0, 1, 2, 3, 4}, "pagemode": {0}, "tddarfcn": {12, 8192, 16383}}},
	{name: "page-1-fdd-None-tdd-None", body: "010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {1}}},
	{name: "page-1-fdd-[]-tdd-[]", body: "01100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {1}}},
	{name: "page-1-fdd-[0]-tdd-[]", body: "01110000000000000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0}, "pagemode": {1}}},
	{name: "page-1-fdd-[]-tdd-[16383]", body: "01101111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {1}, "tddarfcn": {16383}}},
	{name: "page-1-fdd-[1, 16383]-tdd-[0, 12345]", body: "01110000000000000111111111111111101000000000000001110000001110010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {1, 16383}, "pagemode": {1}, "tddarfcn": {0, 12345}}},
	{name: "page-1-fdd-[0, 1, 2, 3, 4]-tdd-[12, 8192, 16383]", body: "01110000000000000010000000000000110000000000001010000000000001110000000000010001000000000011001100000000000001111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0, 1, 2, 3, 4}, "pagemode": {1}, "tddarfcn": {12, 8192, 16383}}},
	{name: "page-2-fdd-None-tdd-None", body: "100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}}},
	{name: "page-2-fdd-[]-tdd-[]", body: "10100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}}},
	{name: "page-2-fdd-[0]-tdd-[]", body: "10110000000000000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0}, "pagemode": {2}}},
	{name: "page-2-fdd-[]-tdd-[16383]", body: "10101111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "tddarfcn": {16383}}},
	{name: "page-2-fdd-[1, 16383]-tdd-[0, 12345]", body: "10110000000000000111111111111111101000000000000001110000001110010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {1, 16383}, "pagemode": {2}, "tddarfcn": {0, 12345}}},
	{name: "page-2-fdd-[0, 1, 2, 3, 4]-tdd-[12, 8192, 16383]", body: "10110000000000000010000000000000110000000000001010000000000001110000000000010001000000000011001100000000000001111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0, 1, 2, 3, 4}, "pagemode": {2}, "tddarfcn": {12, 8192, 16383}}},
	{name: "page-3-fdd-None-tdd-None", body: "110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}}},
	{name: "page-3-fdd-[]-tdd-[]", body: "11100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}}},
	{name: "page-3-fdd-[0]-tdd-[]", body: "11110000000000000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0}, "pagemode": {3}}},
	{name: "page-3-fdd-[]-tdd-[16383]", body: "11101111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "tddarfcn": {16383}}},
	{name: "page-3-fdd-[1, 16383]-tdd-[0, 12345]", body: "11110000000000000111111111111111101000000000000001110000001110010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {1, 16383}, "pagemode": {3}, "tddarfcn": {0, 12345}}},
	{name: "page-3-fdd-[0, 1, 2, 3, 4]-tdd-[12, 8192, 16383]", body: "11110000000000000010000000000000110000000000001010000000000001110000000000010001000000000011001100000000000001111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"fddarfcn": {0, 1, 2, 3, 4}, "pagemode": {3}, "tddarfcn": {12, 8192, 16383}}},
}

func TestClause11225bTypedDistribution(t *testing.T) {
	psiCheckVectors(t, psi15Codec, psi15Vectors)
	psiCheckMandatoryCuts(t, psi15Codec, psi15Vectors[0])
}
func FuzzPSI15Distribution(f *testing.F) { psiFuzz(f, psi15Codec, psi15Vectors[len(psi15Vectors)-1]) }

// TS 44.060 V19.0.0 §11.2.25b: an absent frequency list differs
// from a present structure with both printed terminated lists empty.
func TestPSI15PresentEmptyLists(t *testing.T) {
	for i, row := range psi15Vectors[:2] {
		raw := psiImage(psi15Codec.tag, row.body, 184, true)
		rd, _ := psiRead(t, raw, 184, psi15Codec.tag)
		v, e := psi.DecodePSI15MessageContentFrom(rd)
		if e != nil {
			t.Fatal(e)
		}
		list := v.PAGEMODEGroupFallback.Known.UTRANFrequencyList
		if (list == nil) != (i == 0) {
			t.Fatalf("optional list %v", list)
		}
		if list != nil && (len(list.FDDARFCNList) != 0 || len(list.TDDARFCNList) != 0) {
			t.Fatal("nonempty list")
		}
	}
}

// The helper has a caller-supplied containing extent; these growing
// structural lists are separate from the 184-bit physical control block.
func TestUTRANFrequencyListLinearGrowth(t *testing.T) {
	for _, n := range []int{1, 16, 64, 256} {
		bits := ""
		for i := 0; i < n; i++ {
			bits += "100000000000000"
		}
		bits += "00"
		raw := psiPack(bits)
		rd, e := runtime.NewChoiceReaderBits(raw, len(bits), nil)
		if e != nil {
			t.Fatal(e)
		}
		v, e := psi.DecodeUTRANFrequencyListDescriptionStructFrom(rd)
		if e != nil || len(v.FDDARFCNList) != n || len(v.TDDARFCNList) != 0 || rd.Position() != len(bits) {
			t.Fatalf("list size %d: %v", n, e)
		}
		allocs := testing.AllocsPerRun(3, func() {
			r, e := runtime.NewChoiceReaderBits(raw, len(bits), nil)
			if e != nil {
				panic(e)
			}
			if _, e = psi.DecodeUTRANFrequencyListDescriptionStructFrom(r); e != nil {
				panic(e)
			}
		})
		if !raceEnabled && allocs > float64(32*n+100) {
			t.Fatalf("%d entries allocate %.0f objects", n, allocs)
		}
	}
}
