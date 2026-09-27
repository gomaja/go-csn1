package restoctets

import (
	"bytes"
	"errors"
	"reflect"
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
	if _, err := DecodeIARestOctets([]byte{0x2b}); err == nil || errors.Is(err, runtime.ErrEmptyValue) {
		t.Fatalf("nonempty truncated IA value: %v", err)
	}
	for _, wire := range [][]byte{
		{0x00},                   // LL, noncanonical received spare padding
		{0x20},                   // LL, compressed inter-RAT indication set
		{0x50, 0x00, 0x00, 0x0b}, // LH, multiple blocks packet downlink assignment
		{0x50, 0x00, 0x00, 0x80, 0x00, 0x00, 0x09},             // LH, TMGI IE
		{0x50, 0x00, 0x00, 0xc0, 0x00, 0x00, 0x00, 0x20, 0x0b}, // LH, packet timing advance IE
		{0x40, 0x20, 0x00, 0x00, 0x00, 0x09},                   // LH, EGPRS uplink TFI with TS 44.060 MCS/window IEs
		{0x80, 0x00},                                           // HL, zero-length frequency parameters
		{0x82, 0x00, 0x00, 0x00},                               // HL, two-octet frequency parameters
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
		decoded.Value.Wire.Original = nil // exercise direct encoder, not raw replay
		encoded, err := EncodeIARestOctets(decoded.Value)
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("direct re-encode %x -> %x: %v", wire, encoded, err)
		}
	}
}

func TestIACanonicalSparePadding(t *testing.T) {
	decoded, err := DecodeIARestOctets([]byte{0x00})
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.Wire = runtime.WireInfo{}
	encoded, err := EncodeIARestOctets(decoded.Value)
	if err != nil || !bytes.Equal(encoded, []byte{0x03}) {
		t.Fatalf("canonical L padding = %x, %v; want 03", encoded, err)
	}
}

func TestIAStructuralVectors(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.16 IA grammar: inspect fields within
	// each discriminator branch, not only its decoded byte count.
	ll, err := DecodeIARestOctets([]byte{0x20})
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
	egprs, err := DecodeIARestOctets([]byte{0x40, 0x20, 0, 0, 0, 0x09})
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
	hl, err := DecodeIARestOctets([]byte{0x82, 0, 0, 0})
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

func FuzzIADefinitions(f *testing.F) {
	for _, seed := range [][]byte{{}, {0x00}, {0x20}, {0x50, 0x00, 0x00, 0x0b}, {0x50, 0, 0, 0x80, 0, 0, 0x09}, {0x50, 0, 0, 0xc0, 0, 0, 0, 0x20, 0x0b}, {0x40, 0x20, 0, 0, 0, 0x09}, {0x80, 0x00}, {0x82, 0x00, 0x00, 0x00}, {0xd0, 0, 0, 0, 0, 0, 0x0b}, {0xe8, 0x2b}, {0xff}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, name := range Definitions() {
			definition, err := Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := definition.Decode(data)
			if err != nil {
				continue
			}
			value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			encoded, err := definition.Encode(value)
			if err != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("%s round trip %x -> %x: %v", name, data, encoded, err)
			}
		}
	})
}
