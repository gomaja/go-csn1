package access_test

import (
	"testing"

	"github.com/gomaja/go-csn1/ts44060/access"
)

// TS 44.060 V19.0.0 §11.2.5 p14032 fixes the physical 8/11-bit
// lengths; tables 11.2.5.1–2 pp14036–14070 give these exact arms.
func TestClause1125ChannelRequests(t *testing.T) {
	eleven := []accessArm{
		{"0", "OnePhaseAccessRequest", []string{"MultislotClass", "Priority", "RandomBits"}, []int{5, 2, 3}},
		{"100", "ShortAccessRequest", []string{"NumberOfBlocks", "Priority", "RandomBits"}, []int{3, 2, 3}},
		{"110000", "TwoPhaseAccessRequest", []string{"Priority", "RandomBits"}, []int{2, 3}},
	}
	for i, name := range []string{"PageResponse", "CellUpdate", "MMProcedure", "SingleBlockWithoutTBFEstablishment", "OnePhaseAccessRequestInRLCUnackMode", "DedicatedChannelRequest", "EmergencyCall", "SingleBlockMBMSAccess"} {
		eleven = append(eleven, accessArm{prefix: binaryPrefix(49+i, 6), name: name, fields: []string{"RandomBits"}, widths: []int{5}})
	}
	checkAccessChoices(t, 11, eleven, access.DecodePacketChannelRequest11BitMessageContentFrom, access.EncodePacketChannelRequest11BitMessageContentTo, access.EncodePacketChannelRequest11BitMessageContentCanonical)
	eight := []accessArm{
		{"1", "OnePhaseAccessRequest", []string{"MultislotClass", "RandomBits"}, []int{5, 2}},
		{"00", "ShortAccessRequest", []string{"NumberOfBlocks", "RandomBits"}, []int{3, 3}},
	}
	for i, name := range []string{"TwoPhaseAccessRequest", "PageResponse", "CellUpdate", "MMProcedure", "SingleBlockWithoutTBFEstablishment"} {
		eight = append(eight, accessArm{prefix: binaryPrefix(8+i, 5), name: name, fields: []string{"RandomBits"}, widths: []int{3}})
	}
	for i, name := range []string{"OnePhaseAccessRequestInRLCUnackMode", "DedicatedChannelRequest", "EmergencyCall"} {
		eight = append(eight, accessArm{prefix: binaryPrefix(26+i, 6), name: name, fields: []string{"RandomBits"}, widths: []int{2}})
	}
	eight = append(eight, accessArm{"01111", "SingleBlockMBMSAccess", []string{"RandomBits"}, []int{3}})
	checkAccessChoices(t, 8, eight, access.DecodePacketChannelRequest8BitMessageContentFrom, access.EncodePacketChannelRequest8BitMessageContentTo, access.EncodePacketChannelRequest8BitMessageContentCanonical)
}
func binaryPrefix(v, n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '0' + byte((v>>uint(n-1-i))&1)
	}
	return string(out)
}
