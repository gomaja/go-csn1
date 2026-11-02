package psi_test

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/padding"
	"github.com/gomaja/go-csn1/ts44060/psi"
)

// TS 44.060 V19.0.0 §11.2.25a; exact extracted paragraph span is recorded in the source intake.
// TS 44.060 V19.0.0 §11.2.0.1 paragraph 13381: external selector 111010.
var psi14Codec = psiCodec{tag: 58, mandatory: 2,
	decode: func(r *runtime.Reader) (any, error) { return psi.DecodePSI14MessageContentFrom(r) },
	encode: func(w *runtime.Writer, v any) error {
		return psi.EncodePSI14MessageContentTo(w, v.(psi.PSI14MessageContent))
	},
	canonical: func(v any) any { return runtime.Canonical(v.(psi.PSI14MessageContent)) },
	canonicalAt: func(v any, n int) ([]byte, error) {
		return psi.EncodePSI14MessageContentCanonicalAtLength(v.(psi.PSI14MessageContent), n)
	},
	page: func(v any) (uint8, bool) {
		p := v.(psi.PSI14MessageContent).PAGEMODEGroupFallback.Known
		if p == nil {
			return 0, false
		}
		return p.PAGEMODE, true
	},
	edit: func(v any, page uint8) any {
		p := v.(psi.PSI14MessageContent)
		p.PAGEMODEGroupFallback.Known.PAGEMODE = page
		return p
	},
}
var psi14Vectors = []psiVector{
	{name: "False-null", body: "100101110010101010110110010100111001010010001110101110011110011", extent: 77, canonicalExtent: 77, padded: false, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "sgsnr": {1}, "si13changemark": {2}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "False-zero", body: "1001011100101010101101100101001110010100100011101011100111100110", extent: 184, canonicalExtent: 78, padded: true, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "sgsnr": {1}, "si13changemark": {2}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "False-addition-null", body: "10010111001010101011011001010011100101001000111010111001111001110101001011", extent: 88, canonicalExtent: 88, padded: false, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "rac": {165}, "sgsnr": {1}, "si13changemark": {2}, "sistatusind": {1}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "False-addition-zero", body: "100101110010101010110110010100111001010010001110101110011110011101010010110", extent: 184, canonicalExtent: 89, padded: true, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "rac": {165}, "sgsnr": {1}, "si13changemark": {2}, "sistatusind": {1}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "False-r6-no-power", body: "100101110010101010110110010100111001010010001110101110011110011101010010111010", extent: 92, canonicalExtent: 92, padded: false, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "rac": {165}, "sgsnr": {1}, "si13changemark": {2}, "si2nsupport": {2}, "sistatusind": {1}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "False-r6-power", body: "10010111001010101011011001010011100101001000111010111001111001110101001011111101110", extent: 97, canonicalExtent: 97, padded: false, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "hsn": {21}, "lbmstxpwrmaxcch": {27}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "rac": {165}, "sgsnr": {1}, "si13changemark": {2}, "si2nsupport": {2}, "sistatusind": {1}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
	{name: "True-null", body: "10110010111011101001011011011", extent: 184, canonicalExtent: 44, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "tn": {3}, "tsc": {5}}},
	{name: "True-zero", body: "101100101110111010010110110110", extent: 184, canonicalExtent: 44, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "tn": {3}, "tsc": {5}}},
	{name: "True-addition-null", body: "10110010111011101001011011011111", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "psichangedind": {1}, "tn": {3}, "tsc": {5}}},
	{name: "True-addition-zero", body: "10110010111011101001011011011111", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "psichangedind": {1}, "tn": {3}, "tsc": {5}}},
	{name: "True-r6-no-power", body: "10110010111011101001011011011111", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "psichangedind": {1}, "tn": {3}, "tsc": {5}}},
	{name: "True-r6-power", body: "10110010111011101001011011011111", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"arfcn": {731}, "pagemode": {2}, "pb": {7}, "psi1repeatperiod": {9}, "psichangedind": {1}, "tn": {3}, "tsc": {5}}},
	{name: "CCCH-without-mobile-allocation", body: "100101011011001010011100101001000111010111001111001110101001011111101110", extent: 86, canonicalExtent: 86, padded: false, expected: map[string][]uint64{"accessbursttype": {1}, "alpha": {7}, "bcchchangemark": {5}, "bscvmax": {9}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "navgi": {9}, "networkcontrolorder": {2}, "nmo": {1}, "pagemode": {2}, "pcmeaschan": {1}, "priorityaccessthr": {5}, "rac": {165}, "sgsnr": {1}, "si2nsupport": {2}, "sistatusind": {1}, "spgcccchsup": {1}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}}},
}

func TestClause11225aTypedDistribution(t *testing.T) {
	psiCheckVectors(t, psi14Codec, psi14Vectors)
	psiCheckMandatoryCuts(t, psi14Codec, psi14Vectors[0])
}
func FuzzPSI14Distribution(f *testing.F) { psiFuzz(f, psi14Codec, psi14Vectors[len(psi14Vectors)-1]) }

func TestPSI14AbsentMobileAllocation(t *testing.T) {
	var row psiVector
	for _, candidate := range psi14Vectors {
		if candidate.name == "False-r6-power" {
			row = candidate
		}
	}
	row.name = "CCCH-without-mobile-allocation"
	row.body = row.body[:6] + "0" + row.body[18:]
	row.extent = 14 + len(row.body)
	row.canonicalExtent = row.extent
	expected := map[string][]uint64{}
	for k, v := range row.expected {
		if k != "si13changemark" && k != "hsn" {
			expected[k] = v
		}
	}
	row.expected = expected
	psiCheckVectors(t, psi14Codec, []psiVector{row})
}

// TS 44.060 V19.0.0 §11.1.3.1 paragraphs 13026–13031 puts
// padding after distribution contents. In §11.2.25a pp17755–17781,
// the CCCH arm ends without its own padding, unlike the PBCCH arm.
func TestPSI14CCCHCompleteDistributionBlock(t *testing.T) {
	row := psi14Vectors[len(psi14Vectors)-1]
	raw := psiImage(psi14Codec.tag, row.body, 184, true)
	rd, c := psiRead(t, raw, 184, psi14Codec.tag)
	v, e := psi.DecodePSI14MessageContentFrom(rd)
	if e != nil {
		t.Fatal(e)
	}
	if v.PAGEMODEGroupFallback.Known == nil || rd.Position() != 14+len(row.body) {
		t.Fatalf("CCCH content at %d", rd.Position())
	}
	tail, e := padding.DecodePaddingBitsFrom(rd)
	if e != nil || rd.Position() != 184 {
		t.Fatalf("distribution padding at %d: %v", rd.Position(), e)
	}
	if len(c.TakeReceiverReport().Regions) != 0 {
		t.Fatal("valid distribution block has an ignored region")
	}
	framed := psi14Codec
	framed.encode = func(w *runtime.Writer, value any) error {
		if e := psi14Codec.encode(w, value); e != nil {
			return e
		}
		return padding.EncodePaddingBitsTo(w, tail)
	}
	psiEqualBits(t, psiEncode(t, framed, v, raw, 184, rd), raw, 184)
	tail = runtime.Canonical(tail)
	psiEqualBits(t, psiEncode(t, framed, runtime.Canonical(v), raw, 184, nil), raw, 184)
}
