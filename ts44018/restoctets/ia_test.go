package restoctets

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestIAEmptyValueAndBranches(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16 allows a zero-octet IE length in
	// the containing message, while its printed value grammar requires
	// two discriminator bits.
	if _, err := DecodeIARestOctets(nil); !errors.Is(err, runtime.ErrEmptyValue) {
		t.Fatalf("empty IA value: %v", err)
	}
	if _, err := DecodeIARestOctets([]byte{0}); err == nil || errors.Is(err, runtime.ErrEmptyValue) {
		t.Fatalf("nonempty truncated IA value: %v", err)
	}
	for _, wire := range [][]byte{
		{0x2b},                   // LL, canonical spare padding
		{0x0b},                   // LL, compressed inter-RAT indication set (H)
		{0x50, 0x00, 0x00, 0x0b}, // LH, multiple blocks packet downlink assignment
		{0x50, 0x00, 0x00, 0x80, 0x00, 0x00, 0x2b},             // LH, TMGI IE
		{0x50, 0x00, 0x00, 0xc0, 0x00, 0x00, 0x00, 0x20, 0x0b}, // LH, packet timing advance IE
		{0x40, 0x20, 0x00, 0x00, 0x00, 0x09, 0x2b},             // LH, EGPRS uplink TFI with TS 44.060 MCS/window IEs
		{0x80, 0x08},                                           // HL, zero-length frequency parameters
		{0x82, 0x00, 0x00, 0x08},                               // HL, two-octet frequency parameters
		{0xd0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b},             // HH, EGPRS Level IE
		{0xe8, 0x2b}, // HH, second part packet assignment
	} {
		decoded, err := DecodeIARestOctets(wire)
		if err != nil {
			t.Fatalf("decode %x: %v", wire, err)
		}
		if decoded.BitsConsumed != len(wire)*8 || decoded.Tail.BitLength != 0 {
			t.Fatalf("%x consumed %d bits with tail %d", wire, decoded.BitsConsumed, decoded.Tail.BitLength)
		}
		encoded, err := EncodeIARestOctets(decoded.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("direct re-encode %x -> %x: %v", wire, encoded, err)
		}
	}
}

func TestIACanonicalSparePadding(t *testing.T) {
	decoded, err := DecodeIARestOctets([]byte{0x2b})
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.Wire = runtime.WireInfo{}
	encoded, err := EncodeIARestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, []byte{0x2b}) {
		t.Fatalf("canonical L padding = %x, %v; want 2b", encoded, err)
	}
}

func TestLateRestOctetAdditionsAbsentInPadding(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.16–18: the Rel-14/15 optional
	// selectors are L/H relative to the 0x2b spare-padding pattern.
	for _, tc := range []struct {
		name   string
		wire   []byte
		decode func([]byte) (bool, error)
	}{
		{"IA", bytes.Repeat([]byte{0x2b}, 11), func(b []byte) (bool, error) {
			d, err := DecodeIARestOctets(b)
			return err == nil && d.Value.MultilaterationInformationRequest == nil && d.Value.PEOIMMCellGroupDetails == nil, err
		}},
		{"IAR", bytes.Repeat([]byte{0x2b}, 3), func(b []byte) (bool, error) {
			d, err := DecodeIARRestOctets(b)
			return err == nil && d.Value.PEOIMMCellGroupDetails == nil, err
		}},
		{"IAX", bytes.Repeat([]byte{0x2b}, 4), func(b []byte) (bool, error) {
			d, err := DecodeIAXRestOctets(b)
			return err == nil && d.Value.PEOIMMCellGroupDetails == nil, err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := tc.decode(tc.wire)
			if err != nil || !ok {
				t.Fatalf("late addition present in padding %x: %v", tc.wire, err)
			}
		})
	}
}

func TestCompressedInterRATHOIndUsesLHMeaning(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16 table 10.5.2.16.1 and
	// §10.5.2.18 define L as not used and H as used.
	for _, tc := range []struct {
		first byte
		want  uint8
	}{{0x2b, 0}, {0x0b, 1}} {
		wire := bytes.Repeat([]byte{0x2b}, 11)
		wire[0] = tc.first
		d, err := DecodeIARestOctets(wire)
		if err != nil {
			t.Fatal(err)
		}
		branch := d.Value.CompressedInterRATHOINFOINDChoice.RCC
		if branch == nil || branch.CompressedInterRATHOINFOIND != tc.want {
			t.Fatalf("IA %x: %+v", wire, branch)
		}
		encoded, err := EncodeIARestOctets(d.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("IA round trip %x -> %x: %v", wire, encoded, err)
		}
	}
}

func TestIAStructuralVectors(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16 IA grammar: inspect fields within
	// each discriminator branch, not only its decoded byte count.
	ll, err := DecodeIARestOctets([]byte{0x0b})
	if err != nil || ll.Value.CompressedInterRATHOINFOINDChoice.RCC == nil || ll.Value.CompressedInterRATHOINFOINDChoice.RCC.CompressedInterRATHOINFOIND != 1 {
		t.Fatalf("LL compressed indication: %+v, %v", ll.Value, err)
	}
	lh, err := DecodeIARestOctets([]byte{0x50, 0, 0, 0x0b})
	if err != nil {
		t.Fatal(err)
	}
	lhBranch := lh.Value.CompressedInterRATHOINFOINDChoice.EGPRSPacketUplinkAssignment
	if lhBranch == nil || lhBranch.EGPRSPacketUplinkAssignmentChoice.MultipleBlocksPacketDownlinkAssignment == nil || lhBranch.EGPRSPacketUplinkAssignmentChoice.MultipleBlocksPacketDownlinkAssignment.MultipleBlocksPacketDownlinkAssignment.NUMBEROFALLOCATEDBLOCKS != 0 {
		t.Fatalf("LH multiple blocks path: %+v", lhBranch)
	}
	egprs, err := DecodeIARestOctets([]byte{0x40, 0x20, 0, 0, 0, 0x09, 0x2b})
	if err != nil {
		t.Fatal(err)
	}
	egprsBranch := egprs.Value.CompressedInterRATHOINFOINDChoice.EGPRSPacketUplinkAssignment
	if egprsBranch == nil || egprsBranch.EGPRSPacketUplinkAssignmentChoice.EGPRSPacketUplinkAssignment == nil {
		t.Fatalf("LH EGPRS uplink path: %+v", egprsBranch)
	}
	tfi := egprsBranch.EGPRSPacketUplinkAssignmentChoice.EGPRSPacketUplinkAssignment.EGPRSPacketUplinkAssignment.TFIASSIGNMENTChoice.TFIASSIGNMENT
	if tfi == nil || tfi.EGPRSCHANNELCODINGCOMMAND.EGPRSModulationAndCodingScheme != 0 || tfi.EGPRSWindowSize.EGPRSWindowSize != 0 {
		t.Fatalf("LH EGPRS MCS/window path: %+v", tfi)
	}
	hl, err := DecodeIARestOctets([]byte{0x82, 0, 0, 0x08})
	if err != nil {
		t.Fatal(err)
	}
	hlBranch := hl.Value.CompressedInterRATHOINFOINDChoice.LengthOfFrequencyParameters
	if hlBranch == nil || hlBranch.LengthOfFrequencyParameters != 2 || hlBranch.FrequencyParametersBeforeTime.MAIOChoice.MAIO == nil || hlBranch.FrequencyParametersBeforeTime.MAIOChoice.MAIO.MAIO != 0 {
		t.Fatalf("HL frequency parameters path: %+v", hlBranch)
	}
	hh, err := DecodeIARestOctets([]byte{0xe8, 0x2b})
	if err != nil {
		t.Fatal(err)
	}
	hhBranch := hh.Value.CompressedInterRATHOINFOINDChoice.PacketUplinkAssignment
	if hhBranch == nil || hhBranch.PacketUplinkAssignmentChoice.SecondPartPacketAssignment == nil || hhBranch.PacketUplinkAssignmentChoice.SecondPartPacketAssignment.SecondPartPacketAssignment.ExtendedRAChoice.AltL == nil {
		t.Fatalf("HH second part path: %+v", hhBranch)
	}
}

func TestIARAndIAXRestOctets(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.17 fixes IAR at three octets;
	// §10.5.2.18 permits an absent IAX IE with zero length.
	if _, err := DecodeIARRestOctets(nil); !errors.Is(err, runtime.ErrEmptyValue) {
		t.Fatalf("IAR empty input: %v", err)
	}
	if _, err := DecodeIAXRestOctets(nil); !errors.Is(err, runtime.ErrEmptyValue) {
		t.Fatalf("IAX empty input: %v", err)
	}
	if _, err := DecodeIARRestOctets([]byte{0x0b, 0x2b, 0x2b, 0x2b}); err == nil {
		t.Fatal("four-octet IAR value accepted")
	}
	if _, err := DecodeIAXRestOctets([]byte{0x0b, 0x2b, 0x2b, 0x2b, 0x2b}); err == nil {
		t.Fatal("five-octet IAX value accepted")
	}
	for _, tc := range []struct {
		name string
		wire []byte
	}{
		{"IAR Rest Octets", []byte{0x0b, 0x2b, 0x2b}},
		{"IAX Rest Octets", []byte{0x0b}},
		{"IAX Rest Octets", []byte{0x0b, 0x2b, 0x2b, 0x2b}},
	} {
		definition, err := Lookup(tc.name)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := definition.Decode(tc.wire)
		if err != nil {
			t.Fatal(err)
		}
		value := reflect.ValueOf(decoded)
		if value.FieldByName("BitsConsumed").Int() != int64(len(tc.wire)*8) {
			t.Fatalf("%s consumed wrong length", tc.name)
		}
		encoded, err := definition.Encode(value.FieldByName("Value").Interface())
		if err != nil || !bytes.Equal(encoded, tc.wire) {
			t.Fatalf("%s round trip %x -> %x: %v", tc.name, tc.wire, encoded, err)
		}
	}
	if _, err := Lookup("PEO IMM Cell Group Details struct"); err == nil || !strings.Contains(err.Error(), "10.5.2.16") || !strings.Contains(err.Error(), "10.5.2.17") || !strings.Contains(err.Error(), "10.5.2.18") {
		t.Fatalf("duplicate clause lookup: %v", err)
	}
	for _, clause := range []string{"10.5.2.16", "10.5.2.17", "10.5.2.18"} {
		if _, err := LookupClause(clause, "PEO IMM Cell Group Details struct"); err != nil {
			t.Fatal(err)
		}
	}
	iar, err := DecodeIARRestOctets([]byte{0x0b, 0x2b, 0x2b})
	if err != nil || iar.Value.ExtendedRA1 != nil || iar.Value.ExtendedRA2 != nil || iar.Value.ExtendedRA3 != nil || iar.Value.ExtendedRA4 != nil || iar.Value.RCCChoice.AltL == nil {
		t.Fatalf("IAR field paths: %+v, %v", iar.Value, err)
	}
	iax, err := DecodeIAXRestOctets([]byte{0x0b, 0x2b, 0x2b, 0x2b})
	if err != nil || iax.Value.CompressedInterRATHOINFOIND != 0 || iax.Value.RCCChoice.AltL == nil {
		t.Fatalf("IAX field paths: %+v, %v", iax.Value, err)
	}
}

func FuzzIADefinitions(f *testing.F) {
	// TS 44.018 V19.0.0 §§10.5.2.37g–i: nonempty SI18/SI19/SI20
	// seeds reach their repeat termination and interpreted-count branches.
	f.Add(append([]byte{0, 0x32}, bytes.Repeat([]byte{0xaa}, 18)...))
	f.Add(append([]byte{0, 0x3f}, bytes.Repeat([]byte{0xaa}, 18)...))
	f.Add(append([]byte{0, 0x21, 0xaa, 0x3f}, bytes.Repeat([]byte{0xbb}, 16)...))
	f.Add(append([]byte{0x03, 0}, bytes.Repeat([]byte{0x2b}, 18)...))
	f.Add(append([]byte{0x02, 0x00, 0x80, 0x00, 0x01}, bytes.Repeat([]byte{0x2b}, 15)...))
	for _, seed := range [][]byte{{}, {0x00}, {0x20}, {0x0b}, {0x0b, 0x2b, 0x2b}, {0x50, 0x00, 0x00, 0x0b}, {0x50, 0, 0, 0x80, 0, 0, 0x09}, {0x50, 0, 0, 0xc0, 0, 0, 0, 0x20, 0x0b}, {0x40, 0x20, 0, 0, 0, 0x09}, {0x80, 0x00}, {0x82, 0x00, 0x00, 0x00}, {0xd0, 0, 0, 0, 0, 0, 0x0b}, {0xe8, 0x2b}, {0xff}} {
		f.Add(seed)
	}
	for _, length := range []int{1, 3, 4, 7, 11, 16, 17, 20} {
		f.Add(bytes.Repeat([]byte{0x2b}, length))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, definition := range Descriptors() {
			decoded, err := definition.Decode(data)
			if err != nil {
				continue
			}
			value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			encoded, err := definition.Encode(value)
			if err != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("%s §%s round trip %x -> %x: %v", definition.Name, definition.Clause, data, encoded, err)
			}
		}
	})
}
