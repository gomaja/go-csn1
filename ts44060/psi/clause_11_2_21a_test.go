package psi_test

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/psi"
)

// TS 44.060 V19.0.0 §11.2.21a; exact extracted paragraph span is recorded in the source intake.
// TS 44.060 V19.0.0 §11.2.0.1 paragraph 13382: external selector 111100.
var psi3terCodec = psiCodec{tag: 60, mandatory: 12,
	decode: func(r *runtime.Reader) (any, error) { return psi.DecodePSI3TerMessageContentFrom(r) },
	encode: func(w *runtime.Writer, v any) error {
		return psi.EncodePSI3TerMessageContentTo(w, v.(psi.PSI3TerMessageContent))
	},
	canonical: func(v any) any { return runtime.Canonical(v.(psi.PSI3TerMessageContent)) },
	canonicalAt: func(v any, n int) ([]byte, error) {
		return psi.EncodePSI3TerMessageContentCanonicalAtLength(v.(psi.PSI3TerMessageContent), n)
	},
	page: func(v any) (uint8, bool) {
		p := v.(psi.PSI3TerMessageContent).PAGEMODEGroupFallback.Known
		if p == nil {
			return 0, false
		}
		return p.PAGEMODE, true
	},
	edit: func(v any, page uint8) any {
		p := v.(psi.PSI3TerMessageContent)
		p.PAGEMODEGroupFallback.Known.PAGEMODE = page
		return p
	},
}
var psi3terVectors = []psiVector{
	{name: "absent-cells-None", body: "11101001110000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
	{name: "absent-cells-0", body: "111010011100010000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {0}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
	{name: "absent-cells-1", body: "1110100111000100000011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {1}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1}}},
	{name: "absent-cells-7", body: "1110100111000100001111010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {7}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1}}},
	{name: "absent-cells-127", body: "1110100111000111111111010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {127}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}}},
	{name: "empty-cells-None", body: "1110100111001000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
	{name: "empty-cells-0", body: "11101001110010010000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {0}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
	{name: "empty-cells-1", body: "111010011100100100000011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {1}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1}}},
	{name: "empty-cells-7", body: "111010011100100100001111010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {7}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1}}},
	{name: "empty-cells-127", body: "111010011100100111111111010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {127}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}}},
	{name: "six-cells-None", body: "11101001110011010100010001111111100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {17, 63}}},
	{name: "six-cells-0", body: "111010011100110101000100011111111010000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {0}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {17, 63}}},
	{name: "six-cells-1", body: "1110100111001101010001000111111110100000011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {1}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1}, "rtd": {17, 63}}},
	{name: "six-cells-7", body: "1110100111001101010001000111111110100001111010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {7}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1}, "rtd": {17, 63}}},
	{name: "six-cells-127", body: "1110100111001101010001000111111110111111111010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {127}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1}, "rtd": {17, 63}}},
	{name: "twelve-cells-None", body: "11101001110010101101010101011000111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {2731, 4095}}},
	{name: "twelve-cells-0", body: "111010011100101011010101010110001111111111111110000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {0}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {2731, 4095}}},
	{name: "twelve-cells-1", body: "1110100111001010110101010101100011111111111111100000011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {1}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1}, "rtd": {2731, 4095}}},
	{name: "twelve-cells-7", body: "1110100111001010110101010101100011111111111111100001111010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"numbercells": {7}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1}, "rtd": {2731, 4095}}},
	{name: "both-cells-None", body: "11101001110011101001011010001000111111111101001011101010101011000111111111111110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"cellindexstartrtd": {37, 37}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {17, 63, 2731, 4095}}},
	{name: "both-cells-0", body: "111010011100111010010110100010001111111111010010111010101010110001111111111111110000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"cellindexstartrtd": {37, 37}, "numbercells": {0}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "rtd": {17, 63, 2731, 4095}}},
	{name: "both-cells-1", body: "1110100111001110100101101000100011111111110100101110101010101100011111111111111100000011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"cellindexstartrtd": {37, 37}, "numbercells": {1}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1}, "rtd": {17, 63, 2731, 4095}}},
	{name: "both-cells-7", body: "1110100111001110100101101000100011111111110100101110101010101100011111111111111100001111010101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"cellindexstartrtd": {37, 37}, "numbercells": {7}, "pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}, "reppriority": {1, 0, 1, 0, 1, 0, 1}, "rtd": {17, 63, 2731, 4095}}},
	{name: "single-absent-RTD-1100100", body: "1110100111001100100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
	{name: "single-absent-RTD-1010010", body: "1110100111001010010", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3tercount": {12}, "psi3terindex": {9}}},
}

func TestClause11221aTypedDistribution(t *testing.T) {
	psiCheckVectors(t, psi3terCodec, psi3terVectors)
	psiCheckMandatoryCuts(t, psi3terCodec, psi3terVectors[0])
}
func FuzzPSI3TerDistribution(f *testing.F) {
	psiFuzz(f, psi3terCodec, psi3terVectors[len(psi3terVectors)-1])
}

func TestPSI3TerSingleAbsentRTD(t *testing.T) {
	for _, body := range []string{"1100100", "1010010"} {
		row := psiVector{name: body, body: psi3terVectors[0].body[:12] + body, extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {3}, "psi3changemark": {2}, "psi3terindex": {9}, "psi3tercount": {12}}}
		psiCheckVectors(t, psi3terCodec, []psiVector{row})
		raw := psiImage(psi3terCodec.tag, row.body, 184, true)
		rd, _ := psiRead(t, raw, 184, psi3terCodec.tag)
		v, e := psi.DecodePSI3TerMessageContentFrom(rd)
		if e != nil {
			t.Fatal(e)
		}
		d := v.PAGEMODEGroupFallback.Known.Group.RealTimeDifferenceDescription
		if d == nil {
			t.Fatal("selected RTD description lost")
		}
		if body == "1100100" {
			if d.CellIndexStartRTDGroup == nil || d.CellIndexStartRTDGroup.RTDStruct.RTD != nil || len(d.CellIndexStartRTDGroup.RTDStructList) != 0 {
				t.Fatal("RTD6 presence or empty repetition")
			}
		} else {
			if d.CellIndexStartRTDGroup2 == nil || d.CellIndexStartRTDGroup2.RTDStruct.RTD != nil || len(d.CellIndexStartRTDGroup2.RTDStructList) != 0 {
				t.Fatal("RTD12 presence or empty repetition")
			}
		}
	}
}
