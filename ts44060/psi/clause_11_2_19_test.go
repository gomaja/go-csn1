package psi_test

import (
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/psi"
)

// TS 44.060 V19.0.0 §11.2.19; exact extracted paragraph span is recorded in the source intake.
// TS 44.060 V19.0.0 §11.2.0.1 paragraph 13372: external selector 110010.
var psi2Codec = psiCodec{tag: 50, mandatory: 10,
	decode: func(r *runtime.Reader) (any, error) { return psi.DecodePSI2MessageContentFrom(r) },
	encode: func(w *runtime.Writer, v any) error {
		return psi.EncodePSI2MessageContentTo(w, v.(psi.PSI2MessageContent))
	},
	canonical: func(v any) any { return runtime.Canonical(v.(psi.PSI2MessageContent)) },
	canonicalAt: func(v any, n int) ([]byte, error) {
		return psi.EncodePSI2MessageContentCanonicalAtLength(v.(psi.PSI2MessageContent), n)
	},
	page: func(v any) (uint8, bool) {
		p := v.(psi.PSI2MessageContent).PAGEMODEGroupFallback.Known
		if p == nil {
			return 0, false
		}
		return p.PAGEMODE, true
	},
	edit: func(v any, page uint8) any {
		p := v.(psi.PSI2MessageContent)
		p.PAGEMODEGroupFallback.Known.PAGEMODE = page
		return p
	},
}
var psi2Vectors = []psiVector{
	{name: "empty-null", body: "1001011101000000", extent: 30, canonicalExtent: 30, padded: false, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "empty-zero", body: "10010111010000000", extent: 184, canonicalExtent: 31, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "empty-r99-empty", body: "1001011101000000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "empty-compact", body: "1001011101000000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "empty-additional", body: "100101110100000010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}}},
	{name: "empty-both", body: "100101110100000011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}}},
	{name: "identification-null", body: "10010111011101010101010101010101010101010101010101010010011010101011100001100000", extent: 94, canonicalExtent: 94, padded: false, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rac": {147}}},
	{name: "identification-zero", body: "100101110111010101010101010101010101010101010101010100100110101010111000011000000", extent: 184, canonicalExtent: 95, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rac": {147}}},
	{name: "identification-r99-empty", body: "10010111011101010101010101010101010101010101010101010010011010101011100001100000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rac": {147}}},
	{name: "identification-compact", body: "10010111011101010101010101010101010101010101010101010010011010101011100001100000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rac": {147}}},
	{name: "identification-additional", body: "1001011101110101010101010101010101010101010101010101001001101010101110000110000010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rac": {147}}},
	{name: "identification-both", body: "1001011101110101010101010101010101010101010101010101001001101010101110000110000011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rac": {147}}},
	{name: "nongprs-null", body: "100101110101101001100101101010110100111000100000", extent: 62, canonicalExtent: 62, padded: false, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "neci": {1}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "nongprs-zero", body: "1001011101011010011001011010101101001110001000000", extent: 184, canonicalExtent: 63, padded: true, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "neci": {1}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "nongprs-r99-empty", body: "100101110101101001100101101010110100111000100000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "neci": {1}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "nongprs-compact", body: "100101110101101001100101101010110100111000100000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "largecelloperation": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "nccchnh": {11}, "neci": {1}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "nongprs-additional", body: "10010111010110100110010110101011010011100010000010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "neci": {1}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "nongprs-both", body: "10010111010110100110010110101011010011100010000011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"att": {1}, "bsagblksres": {3}, "bspamfrms": {5}, "ccchconf": {2}, "dtx": {1}, "ec": {1}, "largecelloperation": {1}, "maxretrans": {2}, "mstxpwrmaxccch": {17}, "nccchnh": {11}, "neci": {1}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "pwrc": {0}, "radiolinktimeout": {9}, "txinteger": {9}}},
	{name: "rfl-null", body: "1001011101001100100001010010100111100100101100000", extent: 63, canonicalExtent: 63, padded: false, expected: map[string][]uint64{"lengthofrflcontents": {0}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {9}}},
	{name: "rfl-zero", body: "10010111010011001000010100101001111001001011000000", extent: 184, canonicalExtent: 64, padded: true, expected: map[string][]uint64{"lengthofrflcontents": {0}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {9}}},
	{name: "rfl-r99-empty", body: "1001011101001100100001010010100111100100101100000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"lengthofrflcontents": {0}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {9}}},
	{name: "rfl-compact", body: "1001011101001100100001010010100111100100101100000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "lengthofrflcontents": {0}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {9}}},
	{name: "rfl-additional", body: "100101110100110010000101001010011110010010110000010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"lengthofrflcontents": {0}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rflnumber": {9}}},
	{name: "rfl-both", body: "100101110100110010000101001010011110010010110000011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "lengthofrflcontents": {0}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rflnumber": {9}}},
	{name: "cell-list-null", body: "1001011101000100001011111111000", extent: 45, canonicalExtent: 45, padded: false, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {0, 7, 15}}},
	{name: "cell-list-zero", body: "10010111010001000010111111110000", extent: 184, canonicalExtent: 46, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {0, 7, 15}}},
	{name: "cell-list-r99-empty", body: "1001011101000100001011111111000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {0, 7, 15}}},
	{name: "cell-list-compact", body: "1001011101000100001011111111000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "rflnumber": {0, 7, 15}}},
	{name: "cell-list-additional", body: "100101110100010000101111111100010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rflnumber": {0, 7, 15}}},
	{name: "cell-list-both", body: "100101110100010000101111111100011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "rflnumber": {0, 7, 15}}},
	{name: "mobile-list-null", body: "100101110100001110101010101000", extent: 44, canonicalExtent: 44, padded: false, expected: map[string][]uint64{"hsn": {21}, "manumber": {13}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "mobile-list-zero", body: "1001011101000011101010101010000", extent: 184, canonicalExtent: 45, padded: true, expected: map[string][]uint64{"hsn": {21}, "manumber": {13}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "mobile-list-r99-empty", body: "100101110100001110101010101000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"hsn": {21}, "manumber": {13}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "mobile-list-compact", body: "100101110100001110101010101000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"hsn": {21}, "largecelloperation": {1}, "manumber": {13}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}}},
	{name: "mobile-list-additional", body: "10010111010000111010101010100010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"hsn": {21}, "manumber": {13}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}}},
	{name: "mobile-list-both", body: "10010111010000111010101010100011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"hsn": {21}, "largecelloperation": {1}, "manumber": {13}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}}},
	{name: "nonhopping-empty-null", body: "1001011101000001101000", extent: 36, canonicalExtent: 36, padded: false, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "nonhopping-empty-zero", body: "10010111010000011010000", extent: 184, canonicalExtent: 37, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "nonhopping-empty-r99-empty", body: "1001011101000001101000100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "nonhopping-empty-compact", body: "1001011101000001101000111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "nonhopping-empty-additional", body: "100101110100000110100010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "tsc": {5}}},
	{name: "nonhopping-empty-both", body: "100101110100000110100011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "tsc": {5}}},
	{name: "nonhopping-null", body: "100101110100000110101010100000110010110110110100011001011100", extent: 74, canonicalExtent: 74, padded: false, expected: map[string][]uint64{"arfcn": {321, 721}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "nonhopping-zero", body: "1001011101000001101010101000001100101101101101000110010111000", extent: 184, canonicalExtent: 75, padded: true, expected: map[string][]uint64{"arfcn": {321, 721}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "nonhopping-r99-empty", body: "100101110100000110101010100000110010110110110100011001011100100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {321, 721}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "nonhopping-compact", body: "100101110100000110101010100000110010110110110100011001011100111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {321, 721}, "largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "nonhopping-additional", body: "10010111010000011010101010000011001011011011010001100101110010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {321, 721}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "nonhopping-both", body: "10010111010000011010101010000011001011011011010001100101110011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {321, 721}, "largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-empty-null", body: "10010111010000011011100100", extent: 40, canonicalExtent: 40, padded: false, expected: map[string][]uint64{"manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "hopping-empty-zero", body: "100101110100000110111001000", extent: 184, canonicalExtent: 41, padded: true, expected: map[string][]uint64{"manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "hopping-empty-r99-empty", body: "10010111010000011011100100100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "hopping-empty-compact", body: "10010111010000011011100100111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "manumber": {9}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "tsc": {5}}},
	{name: "hopping-empty-additional", body: "1001011101000001101110010010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"manumber": {9}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "tsc": {5}}},
	{name: "hopping-empty-both", body: "1001011101000001101110010011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "manumber": {9}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "tsc": {5}}},
	{name: "hopping-null", body: "10010111010000011011100110100011001011011010001001011100", extent: 70, canonicalExtent: 70, padded: false, expected: map[string][]uint64{"maio": {17, 40}, "manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-zero", body: "100101110100000110111001101000110010110110100010010111000", extent: 184, canonicalExtent: 71, padded: true, expected: map[string][]uint64{"maio": {17, 40}, "manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-r99-empty", body: "10010111010000011011100110100011001011011010001001011100100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"maio": {17, 40}, "manumber": {9}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-compact", body: "10010111010000011011100110100011001011011010001001011100111110001100111011111111110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "maio": {17, 40}, "manumber": {9}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-additional", body: "1001011101000001101110011010001100101101101000100101110010111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"maio": {17, 40}, "manumber": {9}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "hopping-both", body: "1001011101000001101110011010001100101101101000100101110011111000110011101111111111011111101", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "maio": {17, 40}, "manumber": {9}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {3}, "nibccch2": {7}, "nibccch3": {15}, "nongsminformation": {3}, "pagemode": {2}, "psi2changemark": {1}, "psi2count": {5}, "psi2index": {3}, "psi3quaterbroadcast": {1}, "psi3terbroadcast": {0}, "psi8broadcast": {1}, "timeslotallocation": {150, 151}, "tsc": {5}}},
	{name: "compact-without-idle-or-nccch", body: "1001011101000000110000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {0}, "pagemode": {2}}},
	{name: "idle-mask-A", body: "10010111010000001111000000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "pagemode": {2}}},
	{name: "idle-mask-B", body: "1001011101000000111110001000110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "pagemode": {2}}},
	{name: "idle-mask-C", body: "100101110100000011110101010000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch1": {5}, "pagemode": {2}}},
	{name: "idle-mask-D", body: "10010111010000001111100011010100110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {5}, "pagemode": {2}}},
	{name: "idle-mask-E", body: "100101110100000011110011001000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch2": {9}, "pagemode": {2}}},
	{name: "idle-mask-F", body: "10010111010000001111100010110010110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch2": {9}, "pagemode": {2}}},
	{name: "idle-mask-G", body: "1001011101000000111101010111001000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch1": {5}, "nibccch2": {9}, "pagemode": {2}}},
	{name: "idle-mask-H", body: "100101110100000011111000110101110010110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {5}, "nibccch2": {9}, "pagemode": {2}}},
	{name: "idle-mask-I", body: "100101110100000011110001110100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-J", body: "10010111010000001111100010011101110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-K", body: "1001011101000000111101010101110100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch1": {5}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-L", body: "100101110100000011111000110101011101110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {5}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-M", body: "1001011101000000111100110011110100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch2": {9}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-N", body: "100101110100000011111000101100111101110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch2": {9}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-O", body: "10010111010000001111010101110011110100", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nibccch1": {5}, "nibccch2": {9}, "nibccch3": {13}, "pagemode": {2}}},
	{name: "idle-mask-P", body: "1001011101000000111110001101011100111101110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"largecelloperation": {1}, "nccchnh": {11}, "nibccch0": {1}, "nibccch1": {5}, "nibccch2": {9}, "nibccch3": {13}, "pagemode": {2}}},
}

func TestClause11219TypedDistribution(t *testing.T) {
	psiCheckVectors(t, psi2Codec, psi2Vectors)
	psiCheckMandatoryCuts(t, psi2Codec, psi2Vectors[0])
}
func FuzzPSI2Distribution(f *testing.F) { psiFuzz(f, psi2Codec, psi2Vectors[len(psi2Vectors)-1]) }

func TestPSI2CompactOptionalArms(t *testing.T) {
	rows := []psiVector{{name: "compact-without-idle-or-nccch", body: psi2Vectors[0].body[:16] + "110000", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"pagemode": {2}, "largecelloperation": {0}}}}
	for mask := 0; mask < 16; mask++ {
		body := psi2Vectors[0].body[:16] + "1111"
		expected := map[string][]uint64{"pagemode": {2}, "largecelloperation": {1}}
		for i := 0; i < 4; i++ {
			if mask&(1<<i) == 0 {
				body += "0"
			} else {
				body += "1"
				value := uint64(1 + i*4)
				for bit := 3; bit >= 0; bit-- {
					if value&(1<<uint(bit)) == 0 {
						body += "0"
					} else {
						body += "1"
					}
				}
				expected["nibccch"+string(rune('0'+i))] = []uint64{value}
			}
		}
		if mask%2 == 0 {
			body += "0"
		} else {
			body += "11011"
			expected["nccchnh"] = []uint64{11}
		}
		body += "0"
		rows = append(rows, psiVector{name: "idle-mask-" + string(rune('A'+mask)), body: body, extent: 184, canonicalExtent: 184, padded: true, expected: expected})
	}
	psiCheckVectors(t, psi2Codec, rows)
	raw := psiImage(psi2Codec.tag, rows[1].body, 184, true)
	rd, _ := psiRead(t, raw, 184, psi2Codec.tag)
	v, e := psi.DecodePSI2MessageContentFrom(rd)
	if e != nil {
		t.Fatal(e)
	}
	compact := v.PAGEMODEGroupFallback.Known.Group.COMPACTControlInformationChoice.COMPACTControlInformation.COMPACTControlInformation
	if compact == nil || compact.NumberOfIdleBlocks == nil {
		t.Fatal("present empty idle-block struct was lost")
	}
}

// TS 44.060 V19.0.0 §11.2.19 paragraphs 16845–16849: count code
// zero carries three complete octets before the list's terminator.
func TestPSI2TypedReferenceFrequencyContents(t *testing.T) {
	for _, row := range psi2Vectors {
		if row.name != "rfl-r99-empty" {
			continue
		}
		raw := psiImage(psi2Codec.tag, row.body, 184, true)
		rd, _ := psiRead(t, raw, 184, psi2Codec.tag)
		v, e := psi.DecodePSI2MessageContentFrom(rd)
		if e != nil {
			t.Fatal(e)
		}
		list := v.PAGEMODEGroupFallback.Known.Group.ReferenceFrequencyLists.ReferenceFrequencyStructList
		if len(list) != 1 || list[0].ReferenceFrequencyStruct.RFLContents.BitLength != 24 {
			t.Fatalf("typed RFL %+v", list)
		}
		b := list[0].ReferenceFrequencyStruct.RFLContents.Bytes
		if len(b) != 3 || b[0] != 0xa5 || b[1] != 0x3c || b[2] != 0x96 {
			t.Fatalf("RFL contents %x", b)
		}
	}
}

// TS 44.060 V19.0.0 §11.2.19 paragraphs 16847–16849:
// a four-bit code n carries n+3 octets, hence 24–144 bits of contents.
func TestReferenceFrequencyBoundAndSentinel(t *testing.T) {
	for _, code := range []int{0, 1, 14, 15} {
		body := fmt.Sprintf("1101%04b", code)
		for i := 0; i < code+3; i++ {
			body += "10100101"
		}
		extent := len(body)
		raw := psiPack(body + "101")
		rd, e := runtime.NewChoiceReaderBits(raw, extent+3, nil)
		if e != nil {
			t.Fatal(e)
		}
		old, e := rd.PushLimit(extent)
		if e != nil {
			t.Fatal(e)
		}
		v, e := psi.DecodeReferenceFrequencyStructFrom(rd)
		if e != nil || v.RFLNUMBER != 13 || int(v.LengthOfRFLContents) != code || v.RFLContents.BitLength != 8*(code+3) || rd.Position() != extent {
			t.Fatalf("code %d: %v at %d", code, e, rd.Position())
		}
		for _, b := range v.RFLContents.Bytes {
			if b != 0xa5 {
				t.Fatal("octet contents changed")
			}
		}
		if e = rd.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		sentinel, e := rd.ReadUint(3)
		if e != nil || sentinel != 5 {
			t.Fatalf("sentinel %d: %v", sentinel, e)
		}
		// One missing contents bit remains a failure inside the caller's bound.
		short := psiPack(body[:extent-1] + "101")
		bad, e := runtime.NewChoiceReaderBits(short, extent+2, nil)
		if e != nil {
			t.Fatal(e)
		}
		prior, e := bad.PushLimit(extent - 1)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = psi.DecodeReferenceFrequencyStructFrom(bad); e == nil {
			t.Fatal("short RFL contents accepted")
		}
		if bad.Position() != 8 {
			t.Fatalf("failed field consumed to %d", bad.Position())
		}
		if _, e = bad.ReadBitString(extent - 9); e != nil {
			t.Fatal(e)
		}
		if e = bad.PopLimit(prior); e != nil {
			t.Fatal(e)
		}
		sentinel, e = bad.ReadUint(3)
		if e != nil || sentinel != 5 {
			t.Fatal("RFL crossed its bound")
		}
	}
}
