package registry

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// Received components can exceed their sender's containing carrier and still
// replay exactly. Only a source-recorded maximum may justify canonical refusal;
// an unfilled in-range extent or a wrapped decode error never qualifies.
func canonicalContainingExtentRefusal(d runtime.Descriptor, err error) bool {
	maximum, ok := canonicalContainingProbeMaxima[d.Standard+"/"+d.Clause+"/"+d.Name]
	if !ok {
		return false
	}
	var extent *runtime.ExtentError
	var decode *runtime.DecodeError
	return errors.As(err, &extent) && !errors.As(err, &decode) && extent.Maximum == maximum && extent.Actual > maximum
}

var canonicalContainingProbeMaxima = map[string]int{
	// TS 44.018 V19.0.0 §10.5.2.16 p16145 (IA Rest Octets is 0..11 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.16/EGPRS Packet Uplink Assignment": 11,
	// TS 44.018 V19.0.0 §10.5.2.16 p16145 (IA Rest Octets is 0..11 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.16/Access Technologies Request struct": 11,
	// TS 44.018 V19.0.0 §10.5.2.16 p16145 (IA Rest Octets is 0..11 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.16/Frequency Parameters, before time": 11,
	// TS 44.018 V19.0.0 §10.5.2.16 p16145 (IA Rest Octets is 0..11 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.16/Multiple Blocks Packet Downlink Assignment": 11,
	// TS 44.018 V19.0.0 §10.5.2.37e p19873 (SI16/SI17 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37e/LSA Parameters": 20,
	// TS 44.018 V19.0.0 §10.5.2.37e p19873 (SI16/SI17 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37e/LSA ID information": 20,
	// TS 44.018 V19.0.0 §10.5.2.37m p20180 (SI21 Rest Octets is 20 octets; nested component uses the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37m/Network Sharing EAB Information struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37n p20245 (SI22 Rest Octets is 20 octets; nested component uses the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37n/Network Sharing Information struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/IRAT Cell Reselection Information struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/Priority and UTRAN Parameters Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/UTRAN FDD/TDD Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/Repeated UTRAN FDD/TDD Neighbour Frequency and Priority struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/Priority and E-UTRAN Parameters Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37o p20299 (SI23 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37o/Repeated E-UTRAN Neighbour Frequency and Priority struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.44 p20742, p20745–20759 (SI10 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.44/neighbour information": 20,
	// TS 44.018 V19.0.0 §10.5.2.44 p20742, p20745–20759 (SI10 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.44/info field": 20,
	// TS 44.018 V19.0.0 §10.5.2.71 p21657–21663; §9.1.50b p10660–10665 (SI10ter rest-octet carrier is bit(32))
	"TS 44.018/10.5.2.71/SI10ter Rest Octets": 4,
	// TS 44.018 V19.0.0 §10.5.2.37a p19638 (SI9 Rest Octets is 17 octets; nested Scheduling info uses the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37a/Scheduling info": 17,
	// TS 44.018 V19.0.0 §10.5.2.37a p19650–19657 (Position: 4-bit modulus + at most 16 relative-position bits + 1 BCCH bit = 21 bits; Positions: two Positions and one selector = 43 bits; rounded to octets)
	"TS 44.018/10.5.2.37a/Positions": 6,
	// TS 44.018 V19.0.0 §10.5.2.37a p19650–19657 (Position: 4-bit modulus + at most 16 relative-position bits + 1 BCCH bit = 21 bits; Positions: two Positions and one selector = 43 bits; rounded to octets)
	"TS 44.018/10.5.2.37a/Position": 3,
	// TS 44.018 V19.0.0 §10.5.2.35 p19465, p19489–19491; §§10.5.2.36–10.5.2.37 p19629, p19633 (SI4 is at most 10 octets but these components also occur in the 20-octet SI7/SI8; use the largest containing maximum)
	"TS 44.018/10.5.2.35/SI4 Rest Octets_S": 20,
	// TS 44.018 V19.0.0 §10.5.2.35 p19465, p19489–19491; §§10.5.2.36–10.5.2.37 p19629, p19633 (SI4 is at most 10 octets but these components also occur in the 20-octet SI7/SI8; use the largest containing maximum)
	"TS 44.018/10.5.2.35/LSA ID information": 20,
	// TS 44.018 V19.0.0 §10.5.2.37l p20158 (SI13alt Rest Octets is 20 octets; nested component uses the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37l/PBCCH Description 2 struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.24 p17567 (P2 Rest Octets is 1..11 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.24/MBMS Information": 11,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/3G Neighbour Cell Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/UTRAN FDD Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated UTRAN FDD Neighbour Cells struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/UTRAN TDD Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated UTRAN TDD Neighbour Cells struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/GPRS Real Time Difference Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/RTD6 Struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/RTD12 Struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/GPRS BSIC Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p19054 (7-bit count + at most 127 one-bit entries = 134 bits, rounded to 17 octets)
	"TS 44.018/10.5.2.33b/GPRS REPORT PRIORITY Description struct": 17,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956, p18972–18974 (8-bit extension length + 1 permits 256 bits, but the containing SI2quater is only 20 octets; use its total maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/SI2q Extension Information": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p19081–19083 (7-bit count + at most 127 one-bit entries = 134 bits, rounded to 17 octets)
	"TS 44.018/10.5.2.33b/CCN Support Description struct": 17,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Priority and E-UTRAN Parameters Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/3G Priority Parameters Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated UTRAN Priority Parameters struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/E-UTRAN Parameters Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated E-UTRAN Neighbour Cells struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated E-UTRAN Not Allowed Cells struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Repeated E-UTRAN PCID to TA mapping struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/3G CSG Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/E-UTRAN CSG Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Extended EARFCNs Description struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.33b p18956 (SI2quater Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.33b/Extended EARFCNs Description for CSG Cells struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37b p19682 (SI13 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37b/PEO IMM Cell Group Definition struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37b p19682 (SI13 Rest Octets is 20 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37b/PEO IMM Cell Group Specific Parameters struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37g p19911 (SI19 Rest Octets is 20 octets; nested component uses the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.37g/COMPACT Neighbour Cell params struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.37h p19990, p20005–20008, p20022–20032 (SI18 is 20 octets; the continuation code consumes only the remaining portion of one instance)
	"TS 44.018/10.5.2.37h/Non-GSM Message struct": 20,
	// TS 44.018 V19.0.0 §10.5.2.78 p21896 (IPA Rest Octets is 19 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.78/IPA Uplink Assignment struct": 19,
	// TS 44.018 V19.0.0 §10.5.2.78 p21896 (IPA Rest Octets is 19 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.78/IPA Downlink Assignment struct": 19,
	// TS 44.018 V19.0.0 §10.5.2.78 p21896 (IPA Rest Octets is 19 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.78/IPA Single Block Uplink Assignment struct": 19,
	// TS 44.018 V19.0.0 §9.1.21a p7306–7308, p7322–7335, p7337–7345; §10.5.2.21 p16717; TS 24.008 V20.1.0 §10.5.1.4 p20956 (Mobile Allocation including length: 9 octets; Group Channel Description: 24+2+72=98 bits; Group Call information: 36+1+98=135 bits; shared bit string also carries up to 10 Mobile Identity octets including length)
	"TS 44.018/9.1.21a/Group Call information": 17,
	// TS 44.018 V19.0.0 §9.1.21a p7306–7308, p7322–7335, p7337–7345; §10.5.2.21 p16717; TS 24.008 V20.1.0 §10.5.1.4 p20956 (Mobile Allocation including length: 9 octets; Group Channel Description: 24+2+72=98 bits; Group Call information: 36+1+98=135 bits; shared bit string also carries up to 10 Mobile Identity octets including length)
	"TS 44.018/9.1.21a/Group Channel Description": 13,
	// TS 44.018 V19.0.0 §9.1.21a p7335, p7345; §10.5.2.70 p21621, p21633, p21647–21648; §9.1.50a p10650 (shared LV bit string also carries Frequency List inside the 160-bit SI10bis carrier; use its total 20-octet maximum because other fields vary)
	"TS 44.018/9.1.21a/bit string": 20,
	// TS 44.018 V19.0.0 §10.5.2.22c p17308 (NT/N Rest Octets is 20 octets; nested lists use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.22c/List of Group Call NCH information": 20,
	// TS 44.018 V19.0.0 §10.5.2.22c p17308 (NT/N Rest Octets is 20 octets; nested lists use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.22c/List of Emergency information": 20,
	// TS 44.018 V19.0.0 §10.5.2.22c p17308 (NT/N Rest Octets is 20 octets; nested lists use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.22c/List of VSTK_RAND information": 20,
	// TS 44.018 V19.0.0 §10.5.2.22c p17308 (NT/N Rest Octets is 20 octets; nested lists use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.22c/List of Reduced GCR": 20,
	// TS 44.018 V19.0.0 §10.5.2.23 p17388 (P1 Rest Octets is 0..17 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.23/MBMS Information": 17,
	// TS 44.018 V19.0.0 §10.5.2.23 p17388 (P1 Rest Octets is 0..17 octets; nested components use the entire containing maximum because the other fields vary)
	"TS 44.018/10.5.2.23/ETWS Primary Notification struct": 17,
	// TS 44.018 V19.0.0 §10.5.2.70 p21609–21635; §9.1.50a p10646–10650 (SI10bis rest-octet carrier is bit(160); neighbour component uses the entire containing maximum)
	"TS 44.018/10.5.2.70/SI10bis Neighbour Cell Info": 20,
	// TS 44.018 V19.0.0 §10.5.2.1e p14011, p14024 (type 4 with one-octet value length); TS 24.007 V20.0.0 §11.2.1.1.4 p3025, p3043 (type 4 value at most 255 octets)
	"TS 44.018/10.5.2.1e/Cell Selection Indicator after release of all TCH and SDCCH value part": 255,
	// TS 44.018 V19.0.0 §10.5.2.1e p14011, p14024 (type 4 with one-octet value length); TS 24.007 V20.0.0 §11.2.1.1.4 p3025, p3043 (type 4 value at most 255 octets)
	"TS 44.018/10.5.2.1e/UTRAN FDD Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.1e p14011, p14024 (type 4 with one-octet value length); TS 24.007 V20.0.0 §11.2.1.1.4 p3025, p3043 (type 4 value at most 255 octets)
	"TS 44.018/10.5.2.1e/UTRAN TDD Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.1e p14011, p14024 (type 4 with one-octet value length); TS 24.007 V20.0.0 §11.2.1.1.4 p3025, p3043 (type 4 value at most 255 octets)
	"TS 44.018/10.5.2.1e/E-UTRAN Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.11a p15093; TS 24.007 V20.0.0 §11.2.1.1.4 p3025, p3043 (type 4 value at most 255 octets)
	"TS 44.018/10.5.2.11a/DTM Information Details value part": 255,
	// TS 44.018 V19.0.0 §10.5.2.61 p21318, p21329–21332 (two-octet type 3 IE minus the IEI gives one value octet)
	"TS 44.018/10.5.2.61/Restriction Timer value part": 1,
	// TS 44.060 V19.0.0 §12.24 p23343–23356 (17 fixed + 10 optional PAN + 71 optional extension bits = 98 bits, rounded to 13 octets)
	"TS 44.060/12.24/GPRS Cell Options IE": 13,
	// TS 44.060 V19.0.0 §12.24 p23354–23356 (6-bit extension length + 1 = 64 bits)
	"TS 44.060/12.24/Extension Information": 8,
	// TS 44.060 V19.0.0 §12.8 p22739–22743 (6+6+4+8*(15+3) = 160 bits)
	"TS 44.060/12.8/Direct encoding 2 struct": 20,
	// TS 44.060 V19.0.0 §12.29 p23514–23520 (7-bit length + 127 bitmap bits + 1 selector + 6 MAIO bits = 141 bits, rounded to 18 octets)
	"TS 44.060/12.29/COMPACT reduced MA IE": 18,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/Individual priorities": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/3G Individual Priority Parameters Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/Repeated Individual UTRAN Priority Parameters struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/E-UTRAN Individual Priority Parameters Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/Repeated Individual E-UTRAN Priority Parameters Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/E-UTRAN IPP with extended EARFCNs Description struct": 255,
	// TS 44.018 V19.0.0 §10.5.2.75 p21765–21785 (figure places the Individual priorities value length in octet 2; at most 255 value octets; nested components use the whole containing value maximum because other fields vary)
	"TS 44.018/10.5.2.75/Repeated Individual E-UTRAN PP with extended EARFCNs Description struct": 255,
}

func TestCanonicalContainingExtentRefusalScope(t *testing.T) {
	d := runtime.Descriptor{Standard: "TS 44.018", Clause: "10.5.2.71", Name: "SI10ter Rest Octets"}
	if !canonicalContainingExtentRefusal(d, &runtime.ExtentError{Actual: 5, Maximum: 4}) {
		t.Fatal("source cap refusal not recognized")
	}
	for _, err := range []error{nil, &runtime.ExtentError{Actual: 3, Minimum: 4, Maximum: 4}, &runtime.ExtentError{Actual: 5, Maximum: 6}} {
		if canonicalContainingExtentRefusal(d, err) {
			t.Fatal("unrelated extent error accepted", err)
		}
	}
	d.Name = "unknown"
	if canonicalContainingExtentRefusal(d, &runtime.ExtentError{Actual: 5, Maximum: 4}) {
		t.Fatal("unrecorded cap accepted")
	}
}
