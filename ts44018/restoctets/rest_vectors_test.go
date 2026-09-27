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
		if _, err := d.Decode(make([]byte, 20)); !errors.Is(err, runtime.ErrContextRequired) {
			t.Fatalf("missing typed ACS context error: %v", err)
		}
		if d.DecodeWithContext == nil || d.EncodeWithContext == nil {
			t.Fatalf("%s lacks registry context API", tc.name)
		}
	}
}

func TestSI7SI8ExplicitACSLayouts(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.35, paragraph after the CSN.1 grammar:
	// ACS=1 includes both O and S; ACS=0 includes only S.
	for _, tc := range []struct {
		name   string
		decode func([]byte, runtime.SI4ACS) (any, error)
		encode func(any, runtime.SI4ACS) ([]byte, error)
	}{
		{"SI7", func(b []byte, c runtime.SI4ACS) (any, error) { return DecodeSI7RestOctetsWithContext(b, c) }, func(v any, c runtime.SI4ACS) ([]byte, error) {
			return EncodeSI7RestOctetsWithContext(v.(SI7RestOctets), c)
		}},
		{"SI8", func(b []byte, c runtime.SI4ACS) (any, error) { return DecodeSI8RestOctetsWithContext(b, c) }, func(v any, c runtime.SI4ACS) ([]byte, error) {
			return EncodeSI8RestOctetsWithContext(v.(SI8RestOctets), c)
		}},
	} {
		for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
			wire := bytes.Repeat([]byte{0x2b}, 20)
			got, err := tc.decode(wire, acs)
			if err != nil {
				t.Fatalf("%s ACS=%d: %v", tc.name, acs, err)
			}
			var value any
			switch d := got.(type) {
			case runtime.Decoded[SI7RestOctets]:
				value = d.Value
				if (d.Value.SI4RestOctetsO != nil) != (acs == runtime.SI4ACSOne) {
					t.Fatalf("SI7 wrong layout")
				}
			case runtime.Decoded[SI8RestOctets]:
				value = d.Value
				if (d.Value.SI4RestOctetsO != nil) != (acs == runtime.SI4ACSOne) {
					t.Fatalf("SI8 wrong layout")
				}
			}
			encoded, err := tc.encode(value, acs)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("%s ACS=%d round trip: %x %v", tc.name, acs, encoded, err)
			}
			switch v := value.(type) {
			case SI7RestOctets:
				v.Wire = runtime.WireInfo{}
				encoded, err = EncodeSI7RestOctetsWithContext(v, acs)
			case SI8RestOctets:
				v.Wire = runtime.WireInfo{}
				encoded, err = EncodeSI8RestOctetsWithContext(v, acs)
			}
			if err != nil || len(encoded) != 20 {
				t.Fatalf("%s ACS=%d new value: %x %v", tc.name, acs, encoded, err)
			}
			if _, err := tc.encode(value, 1-acs); err == nil {
				t.Fatalf("%s accepted mismatched ACS", tc.name)
			}
		}
		if _, err := tc.decode(make([]byte, 20), runtime.SI4ACS(2)); err == nil {
			t.Fatalf("%s accepted invalid ACS", tc.name)
		}
	}
}

func TestSI18AndSI20ZeroLengthTerminator(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.37h table 10.5.2.37h.1 ends
	// the Non-GSM list when NR_OF_CONTAINER_OCTETS=0; §.37i reuses it.
	for _, tc := range []struct{ clause, name string }{{"10.5.2.37h", "SI 18 Rest Octets"}, {"10.5.2.37i", "SI 20 Rest Octets"}} {
		d, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Decode(nil); !errors.Is(err, runtime.ErrEmptyValue) {
			t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
		}
		wire := append([]byte{0x03, 0x00}, bytes.Repeat([]byte{0x2b}, 18)...)
		decoded, err := d.Decode(wire)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		encoded, err := d.Encode(decoded)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("%s round trip %x: %v", tc.name, encoded, err)
		}
		switch v := decoded.(type) {
		case runtime.Decoded[SI18RestOctets]:
			if len(v.Value.NonGSMMessageList) != 0 || len(v.Value.Wire.Terminal) != 1 || v.Value.Wire.Terminal[0].BitLength != 8 {
				t.Fatalf("SI18 terminator: %+v", v.Value)
			}
		case runtime.Decoded[SI20RestOctets]:
			if len(v.Value.NonGSMMessageList) != 0 || len(v.Value.Wire.Terminal) != 1 || v.Value.Wire.Terminal[0].BitLength != 8 {
				t.Fatalf("SI20 terminator: %+v", v.Value)
			}
		}
		// The discriminator bits of the zero-length stop record are wire
		// state. Editing a semantic field keeps that record byte-exact.
		altTerminal := append([]byte{0x03, 0x20}, bytes.Repeat([]byte{0x2b}, 18)...)
		alt, err := d.Decode(altTerminal)
		if err != nil {
			t.Fatalf("%s alternate terminal: %v", tc.name, err)
		}
		switch v := alt.(type) {
		case runtime.Decoded[SI18RestOctets]:
			v.Value.SI18INDEX = 1
			encoded, err = EncodeSI18RestOctets(v.Value)
		case runtime.Decoded[SI20RestOctets]:
			v.Value.SI18INDEX = 1
			encoded, err = EncodeSI20RestOctets(v.Value)
		}
		if err != nil || len(encoded) != 20 || encoded[1] != 0x20 {
			t.Fatalf("%s edited terminal: %x %v", tc.name, encoded, err)
		}
		if _, err := d.Decode(append([]byte{0x03, 0x21}, bytes.Repeat([]byte{0x2b}, 18)...)); err == nil {
			t.Fatalf("%s accepted missing terminator", tc.name)
		}
		// A record with 18 container octets fills the 20-octet IE, so the
		// second stop condition needs no zero-length terminator.
		full := append([]byte{0x00, 0x32}, bytes.Repeat([]byte{0xaa}, 18)...)
		reserved := append([]byte{0x00, 0x12}, bytes.Repeat([]byte{0xaa}, 18)...)
		if _, err := d.Decode(reserved); err == nil {
			t.Fatalf("%s accepted reserved non-GSM discriminator", tc.name)
		}
		decoded, err = d.Decode(full)
		if err != nil {
			t.Fatalf("%s full IE: %v", tc.name, err)
		}
		switch v := decoded.(type) {
		case runtime.Decoded[SI18RestOctets]:
			if len(v.Value.NonGSMMessageList) != 1 || v.Value.NonGSMMessageList[0].NonGSMProtocolDiscriminator != 1 || v.Value.NonGSMMessageList[0].NROFCONTAINEROCTETS != 18 || len(v.Value.NonGSMMessageList[0].CONTAINERList) != 18 {
				t.Fatalf("SI18 full message: %+v", v.Value)
			}
		case runtime.Decoded[SI20RestOctets]:
			if len(v.Value.NonGSMMessageList) != 1 || v.Value.NonGSMMessageList[0].NonGSMProtocolDiscriminator != 1 || v.Value.NonGSMMessageList[0].NROFCONTAINEROCTETS != 18 || len(v.Value.NonGSMMessageList[0].CONTAINERList) != 18 {
				t.Fatalf("SI20 full message: %+v", v.Value)
			}
		}
		encoded, err = d.Encode(decoded)
		if err != nil || !bytes.Equal(encoded, full) {
			t.Fatalf("%s full IE round trip: %x %v", tc.name, encoded, err)
		}
		messages := []NonGSMMessageStruct{{NonGSMProtocolDiscriminator: 1, NROFCONTAINEROCTETS: 1, CONTAINERList: []uint8{0xaa}}}
		var constructed any
		if tc.name == "SI 18 Rest Octets" {
			constructed = SI18RestOctets{NonGSMMessageList: messages}
		} else {
			constructed = SI20RestOctets{NonGSMMessageList: messages}
		}
		encoded, err = d.Encode(constructed)
		if err != nil || len(encoded) != 20 {
			t.Fatalf("%s constructed: %x %v", tc.name, encoded, err)
		}
		if _, err = d.Decode(encoded); err != nil {
			t.Fatalf("%s constructed decode: %v", tc.name, err)
		}
		messages[0].NonGSMProtocolDiscriminator = 0
		if tc.name == "SI 18 Rest Octets" {
			constructed = SI18RestOctets{NonGSMMessageList: messages}
		} else {
			constructed = SI20RestOctets{NonGSMMessageList: messages}
		}
		if _, err = d.Encode(constructed); err == nil {
			t.Fatalf("%s encoded reserved non-GSM discriminator", tc.name)
		}
	}
}

func TestSI19PrintedRepeatAndInterpretedRanges(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.37g tables 10.5.2.37g.1–2:
	// printed repeat uses the raw 4-bit count, while the exposed count and
	// frequency difference width have table interpretations 1..16, 1..8.
	cell := COMPACTCellSelectionStruct{BCCChoice: COMPACTCellSelectionStructBCCChoice{
		Alternative: COMPACTCellSelectionStructBCCChoiceAlternativeBCC,
		BCC:         &COMPACTCellSelectionStructBCCChoiceBCC{},
	}}
	v := SI19RestOctets{COMPACTNeighbourCellParameters: COMPACTNeighbourCellParamsStruct{
		STARTFREQUENCYGroupList: []COMPACTNeighbourCellParamsStructSTARTFREQUENCYGroupListEntry{{
			STARTFREQUENCY: 1, COMPACTCellSelectionParams: cell,
			NROFREMAININGCELLS: 1, FREQDIFFLENGTH: 1,
		}},
	}}
	wire, err := EncodeSI19RestOctets(v)
	if err != nil || len(wire) != 20 {
		t.Fatalf("encode SI19: %x %v", wire, err)
	}
	decoded, err := DecodeSI19RestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	groups := decoded.Value.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList
	if len(groups) != 1 || groups[0].NROFREMAININGCELLS != 1 || groups[0].FREQDIFFLENGTH != 1 || len(groups[0].FREQUENCYDIFFGroupList) != 0 {
		t.Fatalf("interpreted count/width: %+v", groups)
	}
	again, err := EncodeSI19RestOctets(decoded.Value)
	if err != nil || !bytes.Equal(again, wire) {
		t.Fatalf("round trip %x: %v", again, err)
	}
	v.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList[0].NROFREMAININGCELLS = 0
	if _, err := EncodeSI19RestOctets(v); err == nil {
		t.Fatal("accepted count below range")
	}
	v.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList[0].NROFREMAININGCELLS = 17
	if _, err := EncodeSI19RestOctets(v); err == nil {
		t.Fatal("accepted count above range")
	}
	v.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList[0].NROFREMAININGCELLS = 2
	v.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList[0].FREQDIFFLENGTH = 2
	v.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList[0].FREQUENCYDIFFGroupList = []COMPACTNeighbourCellParamsStructSTARTFREQUENCYGroupListEntryFREQUENCYDIFFGroupListEntry{{
		FREQUENCYDIFF: runtime.BitString{Bytes: []byte{0x80}, BitLength: 2}, COMPACTCellSelectionStruct: cell,
	}}
	wire, err = EncodeSI19RestOctets(v)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = DecodeSI19RestOctets(wire)
	if err != nil {
		t.Fatal(err)
	}
	groups = decoded.Value.COMPACTNeighbourCellParameters.STARTFREQUENCYGroupList
	if len(groups) != 1 || groups[0].NROFREMAININGCELLS != 2 || groups[0].FREQDIFFLENGTH != 2 || len(groups[0].FREQUENCYDIFFGroupList) != 1 || groups[0].FREQUENCYDIFFGroupList[0].FREQUENCYDIFF.BitLength != 2 {
		t.Fatalf("one repeated cell: %+v", groups)
	}
}

func FuzzSI7SI8WithACS(f *testing.F) {
	f.Add(bytes.Repeat([]byte{0x2b}, 20))
	f.Add(bytes.Repeat([]byte{0}, 20))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
			if d, err := DecodeSI7RestOctetsWithContext(data, acs); err == nil {
				out, err := EncodeSI7RestOctetsWithContext(d.Value, acs)
				if err != nil || !bytes.Equal(out, data) {
					t.Fatalf("SI7 ACS=%d: %x -> %x: %v", acs, data, out, err)
				}
			}
			if d, err := DecodeSI8RestOctetsWithContext(data, acs); err == nil {
				out, err := EncodeSI8RestOctetsWithContext(d.Value, acs)
				if err != nil || !bytes.Equal(out, data) {
					t.Fatalf("SI8 ACS=%d: %x -> %x: %v", acs, data, out, err)
				}
			}
		}
	})
}
