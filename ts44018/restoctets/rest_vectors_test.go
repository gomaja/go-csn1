package restoctets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestEveryNamedRestOctetRejectsEmptyValueWithSentinel(t *testing.T) {
	for _, definition := range Descriptors() {
		if !strings.Contains(strings.ToLower(definition.Name), "rest octet") {
			continue
		}
		if _, err := definition.Decode(nil); !errors.Is(err, runtime.ErrEmptyValue) {
			t.Errorf("%s §%s: zero input = %v", definition.Name, definition.Clause, err)
		}
	}
}

func TestOtherRestOctetNamedVectors(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.32–35, .37a, .37e, .37j–k,
	// .37m–n and .44: L selects absent optional fields and 0x2b
	// supplies the specified L/H spare-padding pattern (TS 44.060 §11).
	for _, tc := range []struct {
		clause, name string
		length       int
		fill         byte
		empty        bool
	}{
		{"10.5.2.25", "P3 Rest Octets", 3, 0x2b, false},
		{"10.5.2.32", "SI1 Rest Octets", 1, 0x2b, false},
		{"10.5.2.33", "SI2bis Rest Octets", 1, 0x2b, false},
		{"10.5.2.33a", "SI2ter Rest Octets", 4, 0x2b, false},
		{"10.5.2.33c", "SI2n Rest Octets", 20, 0x2b, false},
		{"10.5.2.34", "SI3 Rest Octet", 4, 0x2b, false},
		{"10.5.2.35", "SI4 Rest Octets", 1, 0x2b, true},
		{"10.5.2.35a", "SI6 rest octets", 7, 0x2b, false},
		{"10.5.2.37a", "SI9 rest octets", 17, 0x2b, false},
		{"10.5.2.37b", "SI 13 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37e", "SI16 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37f", "SI17 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37j", "SI14 Rest Octets", 16, 0x2b, false},
		{"10.5.2.37k", "SI15 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37l", "SI 13alt Rest Octets", 20, 0x2b, false},
		{"10.5.2.37m", "SI 21 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37n", "SI 22 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37o", "SI 23 Rest Octets", 20, 0x2b, false},
		{"10.5.2.44", "SI10 rest octets", 20, 0x2b, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := LookupClause(tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := definition.Decode(nil); !errors.Is(err, runtime.ErrEmptyValue) {
				t.Fatalf("zero length policy: %v", err)
			}
			wire := bytes.Repeat([]byte{tc.fill}, tc.length)
			decoded, err := definition.Decode(wire)
			if err != nil {
				t.Fatalf("decode %x: %v", wire, err)
			}
			encoded, err := definition.Encode(decoded)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("descriptor round trip %x -> %x: %v", wire, encoded, err)
			}
			if tc.length > 1 && !tc.empty {
				if _, err := definition.Decode(wire[:tc.length-1]); err == nil {
					t.Fatal("short value accepted")
				}
			}
			if !tc.empty {
				if _, err := definition.Decode(append(wire, 0x2b)); err == nil {
					t.Fatal("overlength value accepted")
				}
			}
		})
	}
}

func TestSI13GPRSCellOptionsClosure(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.37b selects PBCCH absent and
	// TS 44.060 V19.0.0 §§12.24, 12.9a supply the two nested IEs.
	// Pycrate 0.7.11 decodes and re-encodes the complete 20 octets.
	wire := append([]byte{0x80, 0, 0, 0, 0, 0, 0, 0x03}, bytes.Repeat([]byte{0x2b}, 12)...)
	decoded, err := DecodeSI13RestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.BCCHCHANGEMARKGroup == nil || decoded.Value.BCCHCHANGEMARKGroup.RACChoice.RAC == nil {
		t.Fatalf("missing GPRS cell options: %+v", decoded.Value)
	}
	if decoded.Value.BCCHCHANGEMARKGroup.RACChoice.RAC.GPRSCellOptions.NMO != 0 {
		t.Fatalf("unexpected NMO: %+v", decoded.Value.BCCHCHANGEMARKGroup.RACChoice.RAC.GPRSCellOptions)
	}
	encoded, err := EncodeSI13RestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("SI13 round trip %x -> %x: %v", wire, encoded, err)
	}
}

func TestIPARestOctetsNamedVector(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.78 fixes the value at 19 octets.
	// Pycrate 0.7.11 accepts and reproduces the all-L padding vector.
	definition, err := LookupClause("10.5.2.78", "IPA Rest Octets")
	if err != nil {
		t.Fatal(err)
	}
	wire := bytes.Repeat([]byte{0x2b}, 19)
	decoded, err := definition.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := definition.Encode(decoded)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("IPA round trip %x -> %x: %v", wire, encoded, err)
	}
}

func TestSI2quaterMinimalVector(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.33b fixes the value at 20 octets.
	// Pycrate 0.7.11 decodes the zero-value optional branches.
	definition, err := LookupClause("10.5.2.33b", "SI2quater Rest Octets")
	if err != nil {
		t.Fatal(err)
	}
	wire := make([]byte, 20)
	decoded, err := definition.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := definition.Encode(decoded)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("SI2quater round trip %x -> %x: %v", wire, encoded, err)
	}
}

func TestGroupCallRestOctetEntryPoints(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.22c–23 and .70. Pycrate 0.7.11
	// decodes and reproduces these absent-list, all-L padding vectors.
	for _, tc := range []struct {
		clause, name string
		length       int
	}{
		{"10.5.2.22c", "NT/N Rest Octets", 20},
		{"10.5.2.23", "P1 Rest Octets", 17},
		{"10.5.2.70", "SI10bis Rest Octets", 1},
	} {
		definition, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		wire := bytes.Repeat([]byte{0x2b}, tc.length)
		decoded, err := definition.Decode(wire)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		encoded, err := definition.Encode(decoded)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("%s: %x -> %x: %v", tc.name, wire, encoded, err)
		}
	}
}

func TestP1GroupCallMobileAllocationLengthValue(t *testing.T) {
	// TS 44.018 V19.0.0 §§9.1.21a, 10.5.2.21 and .23. The
	// length, 36-bit reference, 24-bit channel and allocation byte
	// were independently decoded and encoded by pycrate 0.7.11.
	wire, _ := hex.DecodeString("3123456789b2a190c0300b2b2b2b2b2b2b")
	decoded, err := DecodeP1RestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	group := decoded.Value.GroupCallInformation
	if group == nil || group.GroupCallReference != 0x123456789 || group.GroupChannelDescription == nil || group.GroupChannelDescription.ChannelDescription != 0x654321 {
		t.Fatalf("group-call fields: %+v", group)
	}
	allocation := group.GroupChannelDescription.MobileAllocationChoice.MobileAllocation
	if allocation == nil || allocation.MobileAllocation.Length != 1 || allocation.MobileAllocation.Value.BitLength != 8 || allocation.MobileAllocation.Value.Bytes[0] != 0x80 {
		t.Fatalf("mobile allocation: %+v", allocation)
	}
	encoded, err := EncodeP1RestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("round trip: %x, %v", encoded, err)
	}
	allocation.MobileAllocation.Value.Bytes[0] = 0x81
	encoded, err = EncodeP1RestOctets(decoded.Value)
	want, _ := hex.DecodeString("3123456789b2a190c0302b2b2b2b2b2b2b")
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("edited allocation: %x, %v", encoded, err)
	}
}

func TestSI10bisNeighbourFrequencyListLengthValue(t *testing.T) {
	// TS 44.018 V19.0.0 §§9.1.21a and 10.5.2.70. Pycrate
	// 0.7.11 independently decodes and encodes both wire values.
	wire, _ := hex.DecodeString("011950c8580c002b")
	decoded, err := DecodeSI10bisRestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.SI10bisSequence != 0 || len(decoded.Value.PositionInSI5ListGroupList) != 1 {
		t.Fatalf("neighbour list: %+v", decoded.Value)
	}
	neighbour := decoded.Value.PositionInSI5ListGroupList[0].SI10bisNeighbourCellInfo
	if neighbour.GroupChannelDescription.ChannelDescription != 0x654321 {
		t.Fatalf("channel description: %+v", neighbour.GroupChannelDescription)
	}
	choice := neighbour.FrequencyShortListChoice
	if choice == nil || choice.FrequencyList == nil {
		t.Fatalf("frequency list: %+v", choice)
	}
	list := &choice.FrequencyList.FrequencyList
	if list.Length != 1 || list.Value.BitLength != 8 || list.Value.Bytes[0] != 0x80 {
		t.Fatalf("frequency list value: %+v", list)
	}
	encoded, err := EncodeSI10bisRestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("round trip: %x, %v", encoded, err)
	}
	list.Value.Bytes[0] = 0x81
	encoded, err = EncodeSI10bisRestOctets(decoded.Value)
	want, _ := hex.DecodeString("011950c8580c082b")
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("edited frequency list: %x, %v", encoded, err)
	}
}

func TestSI10terMinimalVector(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.71: the printed 14-bit
	// mandatory prefix followed by two aligned spare-padding bits.
	wire := []byte{0, 3}
	decoded, err := DecodeSI10terRestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.SI10bisSequence != 0 || decoded.Value.BSIC != nil || decoded.Value.NCHPosition != 0 {
		t.Fatalf("SI10ter fields: %+v", decoded.Value)
	}
	encoded, err := EncodeSI10terRestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, wire) {
		t.Fatalf("SI10ter round trip: %x, %v", encoded, err)
	}
}

func TestP2RestOctetsLengthsAndAbsentFields(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.24 permits 1–11 octets. The
	// all-L pattern is accepted by pycrate 0.7.11; within a full
	// Paging Request Type 2, tshark 4.6.8 identifies the P2 rest
	// octets and the three absent Priority fields.
	for length := 1; length <= 11; length++ {
		wire := bytes.Repeat([]byte{0x2b}, length)
		decoded, err := DecodeP2RestOctets(wire)
		if err != nil {
			t.Fatalf("%d octets: %v", length, err)
		}
		if decoded.Value.CN3 != nil || decoded.Value.Priority1 != nil || decoded.Value.Priority2 != nil || decoded.Value.Priority3 != nil {
			t.Fatalf("%d octets: absent fields became present", length)
		}
		encoded, err := EncodeP2RestOctets(decoded.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("%d octets: %x -> %x, %v", length, wire, encoded, err)
		}
	}
	for _, wire := range [][]byte{nil, bytes.Repeat([]byte{0x2b}, 12)} {
		if _, err := DecodeP2RestOctets(wire); err == nil {
			t.Fatalf("accepted %d octets", len(wire))
		}
	}
	// The Rel-6 branches use the TS 44.060 §12.36 MBMS Channel
	// Parameters IE. These vectors also round trip in pycrate 0.7.11.
	for _, tc := range []struct {
		wire []byte
		mbms bool
	}{
		{[]byte{0x28, 0x20}, false},
		{[]byte{0x29, 0x28}, false},
		{[]byte{0x29, 0xcb}, true},
	} {
		decoded, err := DecodeP2RestOctets(tc.wire)
		if err != nil {
			t.Fatalf("Rel-6 %x: %v", tc.wire, err)
		}
		branch := decoded.Value.MBMSNotification3Choice.MBMSNotification3
		if branch == nil || (branch.MBMSNotification3Group != nil && branch.MBMSNotification3Group.MBMSNotification3 != nil) != tc.mbms {
			t.Fatalf("Rel-6 fields %x: %+v", tc.wire, branch)
		}
		encoded, err := EncodeP2RestOctets(decoded.Value)
		if err != nil || !bytes.Equal(encoded, tc.wire) {
			t.Fatalf("Rel-6 round trip %x -> %x: %v", tc.wire, encoded, err)
		}
	}
}

func TestSI7AndSI8RequireACSContext(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.35 selects the SI7/SI8 grammar
	// from ACS in the containing SI4 message, outside these 20 octets.
	for _, tc := range []struct{ clause, name string }{{"10.5.2.36", "SI7 Rest Octets"}, {"10.5.2.37", "SI8 Rest Octets"}} {
		d, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
		}
		if _, err := d.Decode(make([]byte, 20)); !errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("missing ACS conflict: %v", err)
		}
	}
}

func TestSI18AndSI20FailClosedOnTerminatorConflict(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.37h–i: the source's
	// zero-length Non-GSM Message terminator is disputed by the
	// independent pycrate decoder; the standalone values stay closed.
	for _, tc := range []struct{ clause, name string }{{"10.5.2.37h", "SI 18 Rest Octets"}, {"10.5.2.37i", "SI 20 Rest Octets"}} {
		d, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
		}
		wire := append([]byte{0x03, 0x00}, bytes.Repeat([]byte{0x2b}, 18)...)
		if _, err := d.Decode(wire); !errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("missing terminator conflict: %v", err)
		}
	}
}

func TestSI19FailsClosedOnCountConflict(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.37g gives contradictory
	// NR_OF_REMAINING_CELLS count and range text.
	d, err := LookupClause("10.5.2.37g", "SI 19 Rest Octets")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
		t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
	}
	if _, err := d.Decode(bytes.Repeat([]byte{0x2b}, 20)); !errors.Is(err, runtime.ErrUnsupported) {
		t.Fatalf("missing count conflict: %v", err)
	}
}
