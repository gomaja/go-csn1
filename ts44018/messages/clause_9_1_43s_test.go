package messages

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.43s; all values are synthetic.
var ecSystemInformationType4Cases = []ecSourceCase{
	{"empty", "97132b2b", 13},
	{"plmn-0", "971c22b12b2b", 31},
	{"plmn-1", "971c891a2b0b2b2b", 43},
	{"plmn-2", "971c22b6952b2b", 39},
	{"plmn-3", "971c891a2b694b2b2b", 51},
	{"plmn-4", "971c22b3572b2b", 39},
	{"plmn-5", "971c891a2b356b2b2b", 51},
	{"plmn-6", "971c22b697572b2b", 47},
	{"plmn-7", "971c891a2b69756b2b2b", 59},
	{"plmn-count-max", "971ffffffffffffffffffffffffffffffffffffffffffffb2b2b", 188},
}

func TestECSystemInformationType4Source(t *testing.T) {
	checkECSourceCases(t, "9.1.43s", "EC System Information Type 4", 4, ecSystemInformationType4Cases)
}
func FuzzECSystemInformationType4(f *testing.F) {
	fuzzECSource(f, "9.1.43s", "EC System Information Type 4", ecSystemInformationType4Cases)
}

func TestECSystemInformationType4TypedNetworkSharing(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.43s paragraphs 10129–10136. This is the
	// clause-local EC layout, distinct from the SI22 network-sharing structure.
	for _, c := range ecSystemInformationType4Cases {
		t.Run(c.name, func(t *testing.T) {
			wire, err := hex.DecodeString(c.wire)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeECSystemInformationType4(wire)
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.Value
			sharing := v.NetworkSharingInformation
			if c.name == "empty" {
				if sharing != nil {
					t.Fatal("unexpected network sharing")
				}
				return
			}
			if sharing == nil || sharing.CommonPLMNAllowed != 1 {
				t.Fatal("EC network-sharing structure lost")
			}
			flags := 7
			wantCount := 4
			if c.name != "plmn-count-max" {
				flags, err = strconv.Atoi(strings.TrimPrefix(c.name, "plmn-"))
				if err != nil {
					t.Fatal(err)
				}
				wantCount = 1
			}
			if int(sharing.NbAdditionalPLMNs) != wantCount-1 || len(sharing.MCCGroupList) != wantCount {
				t.Fatal("PLMN count lost")
			}
			for _, entry := range sharing.MCCGroupList {
				if (entry.MCC != nil) != (flags&1 != 0) || (entry.NCCPermitted != nil) != (flags&2 != 0) || (entry.ECAccessControlClassGroup != nil) != (flags&4 != 0) {
					t.Fatal("PLMN selector lost")
				}
				mcc, mnc, ncc, access := uint16(0x123), uint16(0x456), uint8(0xa5), uint8(0x55)
				if wantCount == 4 {
					mcc, mnc, ncc, access = 0xfff, 0xfff, 0xff, 0x7f
				}
				if entry.MNC != mnc || (entry.MCC != nil && *entry.MCC != mcc) || (entry.NCCPermitted != nil && *entry.NCCPermitted != ncc) {
					t.Fatal("PLMN identity or NCC lost")
				}
				if entry.ECAccessControlClassGroup != nil && (entry.ECAccessControlClassGroup.ECAccessControlClass != access || entry.ECAccessControlClassGroup.ExceptionReportStatus != 1) {
					t.Fatal("EC access classes lost")
				}
			}
			sharing.NbAdditionalPLMNs ^= 1
			if _, err := EncodeECSystemInformationType4CanonicalAtLength(v, len(wire)); err == nil {
				t.Fatal("inconsistent PLMN count encoded")
			}
		})
	}
	var sharing NetworkSharingInformationStruct
	for _, n := range []int{0, 5} {
		if err := sharing.SetMCCGroupList(make([]NetworkSharingInformationStructMCCGroupListEntry, n)); err == nil {
			t.Fatalf("accepted %d PLMNs", n)
		}
	}
	for _, n := range []int{1, 4} {
		if err := sharing.SetMCCGroupList(make([]NetworkSharingInformationStructMCCGroupListEntry, n)); err != nil || int(sharing.NbAdditionalPLMNs) != n-1 {
			t.Fatalf("PLMN setter: %v", err)
		}
	}
}

func TestECSystemInformationType4TypedCanonical(t *testing.T) {
	wire, err := EncodeECSystemInformationType4CanonicalAtLength(ECSystemInformationType4{MessageType: 4}, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x80, 0x03, 0x2b, 0x2b}) {
		t.Fatalf("fresh typed SI4 %x: %v", wire, err)
	}
	wire, err = EncodeECSystemInformationType4CanonicalAtLength(ECSystemInformationType4{MessageType: 4, ECSI4INDEX: 3, ECSI4COUNT: 3, ECSICHANGEMARK: 31}, 4)
	if err != nil || !bytes.Equal(wire, []byte{0x9f, 0xf3, 0x2b, 0x2b}) {
		t.Fatalf("boundary typed SI4 %x: %v", wire, err)
	}
}
