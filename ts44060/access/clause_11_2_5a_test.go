package access_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
)

// TS 44.060 V19.0.0 §11.2.5a table 11.2.5a.2 pp14098–14118
// defines eleven-bit non-PEO access requests; p14116 uses decision
// 113's exact colon repair. Training-sequence choice is physical context.
func TestClause1125aEGPRSRequests(t *testing.T) {
	arms := []accessArm{
		{"0", "OnePhaseAccessRequest", []string{"MultislotClass", "Priority", "RandomBits"}, []int{5, 2, 3}},
		{"100", "ShortAccessRequest", []string{"NumberOfBlocks", "Priority", "RandomBits"}, []int{3, 2, 3}},
		{"101", "OnePhaseAccessRequestByReducedLatencyMS", []string{"MultislotClassGroup", "Priority", "RandomBits"}, []int{3, 2, 3}},
		{"110000", "TwoPhaseAccessRequest", []string{"Priority", "RandomBits"}, []int{2, 3}},
		{"110011", "Signalling", []string{"RandomBits"}, []int{5}},
		{"110101", "OnePhaseAccessRequestInRLCUnackMode", []string{"RandomBits"}, []int{5}},
		{"110110", "DedicatedChannelRequest", []string{"RandomBits"}, []int{5}},
		{"110111", "EmergencyCall", []string{"RandomBits"}, []int{5}},
		{"111000", "TwoPhaseAccessRequestByIPACapableMS", []string{"Priority", "RandomBits"}, []int{2, 3}},
		{"111001", "SignallingByIPACapableMS", []string{"RandomBits"}, []int{5}},
	}
	checkAccessChoices(t, 11, arms, access.DecodeEGPRSPacketChannelRequestMessageContentFrom, access.EncodeEGPRSPacketChannelRequestMessageContentTo, access.EncodeEGPRSPacketChannelRequestMessageContentCanonical)
}

// Table 11.2.5a.4 pp14132–14187 supplies three selector bits before
// each eight-bit body. The part-2 body here covers bits 4–11 only;
// table 11.2.5a.4.1's nineteen further bits require a separate carrier.
func TestClause1125aPEOBodies(t *testing.T) {
	checkPEOBody(t, "000", []string{"AccessCause", "PEOPriority", "RandomBits", "EGPRSCapability", "Spare"}, []int{2, 1, 3, 1, 1}, access.DecodeEGPRSPACKETCHANNELREQUESTMessageContentForPEOOnePhaseAccessRequestFrom, access.EncodeEGPRSPACKETCHANNELREQUESTMessageContentForPEOOnePhaseAccessRequestTo)
	checkPEOBody(t, "001", []string{"RandomBits", "AccessDiscriminator", "FinalMTAAccess", "Spare"}, []int{3, 1, 1, 3}, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo)
	checkPEOBody(t, "010", []string{"ShortID"}, []int{8}, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo)
	checkPEOBody(t, "011", []string{"RandomIDLow", "MSTransmissionOffset"}, []int{4, 4}, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1From, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1To)
	checkPEOBody(t, "100", []string{"RandomIDLow", "MSSyncAccuracy"}, []int{4, 4}, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2From, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2To)
}
func checkPEOBody[T any](t *testing.T, prefix string, fields []string, widths []int, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error) {
	t.Helper()
	for value := 0; value < 256; value++ {
		bits := fmt.Sprintf("%08b", value)
		full := prefix + bits
		r := accessReader(t, full+"10101101", nil)
		if _, err := r.ReadUint(3); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(8)
		if err != nil {
			t.Fatal(err)
		}
		v, err := decode(r)
		if err != nil || r.Position() != 11 {
			t.Fatalf("%s: %v", full, err)
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(8)
		if err != nil || sentinel != 0xad {
			t.Fatal("sentinel consumed")
		}
		rv := reflect.ValueOf(v)
		offset := 0
		for i, name := range fields {
			want := uint64(0)
			for _, b := range bits[offset : offset+widths[i]] {
				want = want*2 + uint64(b-'0')
			}
			offset += widths[i]
			if rv.FieldByName(name).Uint() != want {
				t.Fatalf("%s typed %s lost", full, name)
			}
		}
		w := runtime.NewWriter()
		if err = w.WriteUint(uint64(prefix[0]-'0')*4+uint64(prefix[1]-'0')*2+uint64(prefix[2]-'0'), 3); err != nil {
			t.Fatal(err)
		}
		if err = encode(w, runtime.Canonical(v)); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		accessEqual(t, out, full)
		accessTruncations(t, bits, decode)
		edited := reflect.ValueOf(&v).Elem().FieldByName(fields[len(fields)-1])
		edited.SetUint(edited.Uint() ^ 1)
		accessEqual(t, accessEncode(t, v, encode), fmt.Sprintf("%08b", value^1))
	}
}
