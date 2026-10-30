package messages

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.43r; all values are synthetic.
var ecSystemInformationType3Cases = []ecSourceCase{
	{"empty", "77182b2b", 16},
	{"serving-0", "771ffe2b2b2b", 26},
	{"serving-1", "771ffff92b2b", 31},
	{"frequency-0", "771a0aa32b2b", 29},
	{"frequency-31", "771bfaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa32b2b", 277},
	{"neighbour-absent", "7719022b2b", 24},
	{"neighbour-six", "771905fd2b2b", 31},
	{"neighbour-nine", "771907ffab2b2b", 34},
	{"reselection-0", "771907ffc32b2b", 38},
	{"reselection-1", "771907fff82b2b", 40},
	{"reselection-2", "771907ffdffe2b2b2b", 49},
	{"reselection-3", "771907ffffff8b2b2b", 51},
	{"reselection-4", "771907ffcfeb2b2b", 44},
	{"reselection-5", "771907fffbfb2b2b", 46},
	{"reselection-6", "771907ffdffffd2b2b", 55},
	{"reselection-7", "771907ffffffff2b2b2b", 57},
	{"reselection-8", "771907ffc72b2b", 40},
	{"reselection-9", "771907fff9eb2b2b", 42},
	{"reselection-10", "771907ffdffeeb2b2b", 51},
	{"reselection-11", "771907ffffffbb2b2b", 53},
	{"reselection-12", "771907ffcfff2b2b", 46},
	{"reselection-13", "771907fffbff2b2b", 48},
	{"reselection-14", "771907ffdfffffab2b2b", 57},
	{"reselection-15", "771907ffffffffeb2b2b", 59},
	{"neighbour-count-max", "7719fa4924924924924924924924932b2b", 117},
}

func TestECSystemInformationType3Source(t *testing.T) {
	checkECSourceCases(t, "9.1.43r", "EC System Information Type 3", 3, ecSystemInformationType3Cases)
}
func FuzzECSystemInformationType3(f *testing.F) {
	fuzzECSource(f, "9.1.43r", "EC System Information Type 3", ecSystemInformationType3Cases)
}

func TestECSystemInformationType3TypedReselection(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.43r paragraphs 10051–10080. CE_AUTH_OFFSET
	// at paragraph 10078 is tested from the source; upstream omits that field.
	for _, c := range ecSystemInformationType3Cases {
		t.Run(c.name, func(t *testing.T) {
			wire, err := hex.DecodeString(c.wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeECSystemInformationType3(wire)
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.Value
			if v.ECSI4Indicator != 1 {
				t.Fatal("SI4 indicator lost")
			}
			serving := v.ECCellReselectionParameters
			if strings.HasPrefix(c.name, "serving-") {
				if serving == nil || serving.CELLRESELECTHYSTERESIS != 7 || serving.CELLRESELECTOFFSET != 63 {
					t.Fatal("serving parameters lost")
				}
				present := strings.HasSuffix(c.name, "-1")
				if (serving.C1DELTAMINGroup != nil) != present {
					t.Fatal("C1 selector lost")
				}
				if present && (serving.C1DELTAMINGroup.C1DELTAMIN != 3 || serving.C1DELTAMINGroup.C1DELTAMAX != 7) {
					t.Fatal("C1 bounds lost")
				}
			} else if serving != nil {
				t.Fatal("unexpected serving parameters")
			}
			desc := v.ECNeighbourCellDescription
			if strings.HasPrefix(c.name, "frequency-") {
				n, err := strconv.Atoi(strings.TrimPrefix(c.name, "frequency-"))
				if err != nil {
					t.Fatal(err)
				}
				if desc == nil || int(desc.NumberOfOctets) != n || desc.NeighbourFrequencyListInformation.BitLength != 8*(n+1) || !bytes.Equal(desc.NeighbourFrequencyListInformation.Bytes, bytes.Repeat([]byte{0xaa}, n+1)) {
					t.Fatal("neighbour frequency information lost")
				}
				desc.NeighbourFrequencyListInformation.BitLength--
				if _, err := EncodeECSystemInformationType3CanonicalAtLength(v, len(wire)); err == nil {
					t.Fatal("invalid neighbour frequency width encoded")
				}
			} else if desc != nil {
				t.Fatal("unexpected neighbour frequency description")
			}
			neighbours := v.ECNeighbourCellReselectionParameters
			if !strings.HasPrefix(c.name, "neighbour-") && !strings.HasPrefix(c.name, "reselection-") {
				if neighbours != nil {
					t.Fatal("unexpected neighbours")
				}
				return
			}
			if neighbours == nil {
				t.Fatal("neighbours absent")
			}
			wantCount := 1
			if c.name == "neighbour-count-max" {
				wantCount = 32
			}
			if int(neighbours.NbNCELL) != wantCount-1 || len(neighbours.BSICGroupList) != wantCount {
				t.Fatal("neighbour count lost")
			}
			for _, entry := range neighbours.BSICGroupList {
				if entry.CELLTYPE != 1 {
					t.Fatal("cell type lost")
				}
				switch c.name {
				case "neighbour-six":
					if entry.BSICChoice == nil || entry.BSICChoice.BSIC == nil || entry.BSICChoice.BSIC.BSIC != 63 || entry.BSICChoice.Alt1 != nil {
						t.Fatal("six-bit BSIC lost")
					}
				case "neighbour-nine":
					if entry.BSICChoice == nil || entry.BSICChoice.Alt1 == nil || entry.BSICChoice.Alt1.BSIC != 511 || entry.BSICChoice.BSIC != nil {
						t.Fatal("nine-bit BSIC lost")
					}
				default:
					if !strings.HasPrefix(c.name, "reselection-") && entry.BSICChoice != nil {
						t.Fatal("unexpected BSIC")
					}
				}
				group := entry.CELLBARACCESSGroupGroup
				if !strings.HasPrefix(c.name, "reselection-") {
					if group != nil {
						t.Fatal("unexpected indicated parameters")
					}
					continue
				}
				flags, err := strconv.Atoi(strings.TrimPrefix(c.name, "reselection-"))
				if err != nil {
					t.Fatal(err)
				}
				if group == nil || entry.BSICChoice == nil || entry.BSICChoice.Alt1 == nil || entry.BSICChoice.Alt1.BSIC != 511 {
					t.Fatal("indicated parameters or BSIC lost")
				}
				if (group.CELLBARACCESSGroup != nil) != (flags&1 != 0) || (group.ECRXLEVACCESSMINGroup != nil) != (flags&2 != 0) || (group.CELLRESELECTOFFSET != nil) != (flags&4 != 0) || (group.CEAUTHOFFSET != nil) != (flags&8 != 0) {
					t.Fatal("reselection selector lost")
				}
				if group.CELLBARACCESSGroup != nil && (group.CELLBARACCESSGroup.CELLBARACCESS != 1 || group.CELLBARACCESSGroup.SAMERAASSERVINGCELL != 1) {
					t.Fatal("access parameters lost")
				}
				if group.ECRXLEVACCESSMINGroup != nil && (group.ECRXLEVACCESSMINGroup.ECRXLEVACCESSMIN != 63 || group.ECRXLEVACCESSMINGroup.MSTXPWRMAXCCH != 31) {
					t.Fatal("access minimum/power lost")
				}
				if group.CELLRESELECTOFFSET != nil && *group.CELLRESELECTOFFSET != 63 {
					t.Fatal("reselect offset lost")
				}
				if group.CEAUTHOFFSET != nil && *group.CEAUTHOFFSET != 3 {
					t.Fatal("CE authorization offset lost")
				}
			}
			neighbours.NbNCELL ^= 1
			if _, err := EncodeECSystemInformationType3CanonicalAtLength(v, len(wire)); err == nil {
				t.Fatal("inconsistent neighbour count encoded")
			}
		})
	}
	var value ECNeighbourCellReselectionParametersStruct
	for _, n := range []int{0, 33} {
		if err := value.SetBSICGroupList(make([]ECNeighbourCellReselectionParametersStructBSICGroupListEntry, n)); err == nil {
			t.Fatalf("accepted %d neighbours", n)
		}
	}
	for _, n := range []int{1, 32} {
		if err := value.SetBSICGroupList(make([]ECNeighbourCellReselectionParametersStructBSICGroupListEntry, n)); err != nil || int(value.NbNCELL) != n-1 {
			t.Fatalf("neighbour setter: %v", err)
		}
	}
}

func TestECSystemInformationType3TypedCanonical(t *testing.T) {
	wire, err := EncodeECSystemInformationType3CanonicalAtLength(ECSystemInformationType3{MessageType: 3}, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x60, 0x00, 0x2b, 0x2b}) {
		t.Fatalf("fresh typed SI3 %x: %v", wire, err)
	}
	wire, err = EncodeECSystemInformationType3CanonicalAtLength(ECSystemInformationType3{MessageType: 3, ECSI3INDEX: 3, ECSI3COUNT: 3, ECSICHANGEMARK: 31, ECSI4Indicator: 1}, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x7f, 0xf8, 0x2b, 0x2b}) {
		t.Fatalf("boundary typed SI3 %x: %v", wire, err)
	}
}
