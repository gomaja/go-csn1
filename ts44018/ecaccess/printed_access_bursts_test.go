package ecaccess_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/ecaccess"
)

type printedAccessCase struct {
	name, context  string
	bits           int
	prefix, suffix string
	bodyFields     int
	widths         []int
	run            func(*testing.T, []uint64)
	fuzz           func(*testing.T, []byte)
}

// TS 44.018 V19.0.0 table 9.1.65.2 pp12116–12245 supplies these
// contexts and field widths; table 9.1.65.2.1 pp12248–12255 supplies
// the nineteen continuation bits. RACH table 9.1.65.4 is excluded.
func printedAccessCases() []printedAccessCase {
	return []printedAccessCase{
		makePrintedAccessCase("EC PACKET CHANNEL REQUEST message content", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"3 (CC1)\",\"prefix\":\"\",\"suffix\":\"EGPRS Capability\"}", "", "", 4, []int{3, 1, 3, 3, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContentFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContentTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContentCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent {
				return ecaccess.Table9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent{ECNumberOfBlocks: uint8(v[0]), ECPriority: uint8(v[1]), RandomBits: uint8(v[2]), SelectedDLCoverageClass: uint8(v[3]), EGPRSCapability: uint8(v[4])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent) []uint64 {
				return []uint64{uint64(v.ECNumberOfBlocks), uint64(v.ECPriority), uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass), uint64(v.EGPRSCapability)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN3Cc1PrefixNSuffixEgprsCapabilityECPACKETCHANNELREQUESTMessageContent) {
				v.ECNumberOfBlocks ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (RLC Data Block method or Page Response for Positioning Event)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"4 (CC1)\",\"prefix\":\"00\",\"suffix\":\"EGPRS Capability\"}", "00", "", 4, []int{3, 3, 1, 1, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent {
				return ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent{RandomBits: uint8(v[0]), SelectedDLCoverageClass: uint8(v[1]), AccessDiscriminator: uint8(v[2]), FinalMTAAccess: uint8(v[3]), EGPRSCapability: uint8(v[4])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) []uint64 {
				return []uint64{uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass), uint64(v.AccessDiscriminator), uint64(v.FinalMTAAccess), uint64(v.EGPRSCapability)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN00SuffixEgprsCapabilityECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) {
				v.RandomBits ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Access Burst method)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"4 (CC1)\",\"prefix\":\"01\",\"suffix\":\"Spare\"}", "01", "", 1, []int{8, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethodCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod {
				return ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod{ShortID: uint8(v[0]), Spare: uint8(v[1])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) []uint64 {
				return []uint64{uint64(v.ShortID), uint64(v.Spare)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN01SuffixSpareECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) {
				v.ShortID ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Extended Access Burst Method – part 1)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"4 (CC1)\",\"prefix\":\"10\",\"suffix\":\"0\"}", "10", "0", 2, []int{4, 4},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1From, ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1To, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1Canonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1 {
				return ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1{RandomIDLow: uint8(v[0]), MSTransmissionOffset: uint8(v[1])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1) []uint64 {
				return []uint64{uint64(v.RandomIDLow), uint64(v.MSTransmissionOffset)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN4Cc1PrefixN10SuffixN0ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1) {
				v.RandomIDLow ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Extended Access Burst Method – part 2)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"4 (CC1)\\nNote 1\",\"prefix\":\"10\",\"suffix\":\"1\"}", "10", "1", 2, []int{4, 4, 12, 3, 1, 3},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2From, ecaccess.DecodeTable9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2To, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2, ecaccess.EncodeTable9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2Canonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2 {
				return ecaccess.Table9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2{RandomIDLow: uint8(v[0]), MSSyncAccuracy: uint8(v[1]), RandomIDHigh: uint16(v[2]), ShortBSSID: uint8(v[3]), FinalMTAAccess: uint8(v[4]), Spare: uint8(v[5])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2) []uint64 {
				return []uint64{uint64(v.RandomIDLow), uint64(v.MSSyncAccuracy), uint64(v.RandomIDHigh), uint64(v.ShortBSSID), uint64(v.FinalMTAAccess), uint64(v.Spare)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN4Cc1Note1PrefixN10SuffixN1ECMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2) {
				v.RandomIDLow ^= 1
			}),
		makePrintedAccessCase("EC PACKET CHANNEL REQUEST message content", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"5 (CC2)\",\"prefix\":\"\",\"suffix\":\"0\"}", "", "0", 4, []int{3, 1, 3, 3},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent {
				return ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent{ECNumberOfBlocks: uint8(v[0]), ECPriority: uint8(v[1]), RandomBits: uint8(v[2]), SelectedDLCoverageClass: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) []uint64 {
				return []uint64{uint64(v.ECNumberOfBlocks), uint64(v.ECPriority), uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) {
				v.ECNumberOfBlocks ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (RLC Data Block method or Page Response for Positioning Event)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"5 (CC2)\",\"prefix\":\"00\",\"suffix\":\"1\"}", "00", "1", 4, []int{3, 3, 1, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent {
				return ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent{RandomBits: uint8(v[0]), SelectedDLCoverageClass: uint8(v[1]), AccessDiscriminator: uint8(v[2]), FinalMTAAccess: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) []uint64 {
				return []uint64{uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass), uint64(v.AccessDiscriminator), uint64(v.FinalMTAAccess)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) {
				v.RandomBits ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Access Burst method)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"5 (CC2)\",\"prefix\":\"01\",\"suffix\":\"1\"}", "01", "1", 1, []int{8},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod {
				return ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod{ShortID: uint8(v[0])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) []uint64 {
				return []uint64{uint64(v.ShortID)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN5Cc2PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) {
				v.ShortID ^= 1
			}),
		makePrintedAccessCase("EC PACKET CHANNEL REQUEST message content", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"6 (CC3)\",\"prefix\":\"\",\"suffix\":\"0\"}", "", "0", 4, []int{3, 1, 3, 3},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent {
				return ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent{ECNumberOfBlocks: uint8(v[0]), ECPriority: uint8(v[1]), RandomBits: uint8(v[2]), SelectedDLCoverageClass: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) []uint64 {
				return []uint64{uint64(v.ECNumberOfBlocks), uint64(v.ECPriority), uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) {
				v.ECNumberOfBlocks ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (RLC Data Block method or Page Response for Positioning Event)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"6 (CC3)\",\"prefix\":\"00\",\"suffix\":\"1\"}", "00", "1", 4, []int{3, 3, 1, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent {
				return ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent{RandomBits: uint8(v[0]), SelectedDLCoverageClass: uint8(v[1]), AccessDiscriminator: uint8(v[2]), FinalMTAAccess: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) []uint64 {
				return []uint64{uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass), uint64(v.AccessDiscriminator), uint64(v.FinalMTAAccess)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) {
				v.RandomBits ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Access Burst method)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"6 (CC3)\",\"prefix\":\"01\",\"suffix\":\"1\"}", "01", "1", 1, []int{8},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod {
				return ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod{ShortID: uint8(v[0])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) []uint64 {
				return []uint64{uint64(v.ShortID)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN6Cc3PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) {
				v.ShortID ^= 1
			}),
		makePrintedAccessCase("EC PACKET CHANNEL REQUEST message content", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"7 (CC4) or 8 (CC5)\",\"prefix\":\"\",\"suffix\":\"0\"}", "", "0", 4, []int{3, 1, 3, 3},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContentCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent {
				return ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent{ECNumberOfBlocks: uint8(v[0]), ECPriority: uint8(v[1]), RandomBits: uint8(v[2]), SelectedDLCoverageClass: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) []uint64 {
				return []uint64{uint64(v.ECNumberOfBlocks), uint64(v.ECPriority), uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixNSuffixN0ECPACKETCHANNELREQUESTMessageContent) {
				v.ECNumberOfBlocks ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content RLC Data Block method or Page Response for Positioning Event)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"7 (CC4) or 8 (CC5)\",\"prefix\":\"00\",\"suffix\":\"1\"}", "00", "1", 4, []int{3, 3, 1, 1},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent {
				return ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent{RandomBits: uint8(v[0]), SelectedDLCoverageClass: uint8(v[1]), AccessDiscriminator: uint8(v[2]), FinalMTAAccess: uint8(v[3])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) []uint64 {
				return []uint64{uint64(v.RandomBits), uint64(v.SelectedDLCoverageClass), uint64(v.AccessDiscriminator), uint64(v.FinalMTAAccess)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN00SuffixN1ECMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEvent) {
				v.RandomBits ^= 1
			}),
		makePrintedAccessCase("EC MULTILATERATION REQUEST message content (Access Burst method)", "{\"table\":\"9.1.65.2\",\"training_sequence\":\"7 (CC4) or 8 (CC5)\",\"prefix\":\"01\",\"suffix\":\"1\"}", "01", "1", 1, []int{8},
			ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, ecaccess.DecodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod, ecaccess.EncodeTable9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethodCanonical,
			func(v []uint64) ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod {
				return ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod{ShortID: uint8(v[0])}
			},
			func(v ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) []uint64 {
				return []uint64{uint64(v.ShortID)}
			},
			func(v *ecaccess.Table9_1_65_2TrainingSequenceN7Cc4Or8Cc5PrefixN01SuffixN1ECMULTILATERATIONREQUESTMessageContentAccessBurstMethod) {
				v.ShortID ^= 1
			}),
	}
}

func makePrintedAccessCase[T any](name, context, prefix, suffix string, bodyFields int, widths []int,
	decodeFrom func(*runtime.Reader) (T, error), decode func([]byte) (runtime.Decoded[T], error),
	encodeTo func(*runtime.Writer, T) error, encode func(T) ([]byte, error), canonical func(T) ([]byte, error),
	construct func([]uint64) T, fields func(T) []uint64, edit func(*T)) printedAccessCase {
	c := printedAccessCase{name: name, context: context, prefix: prefix, suffix: suffix, bodyFields: bodyFields, widths: widths}
	c.bits = len(prefix) + len(suffix)
	for _, w := range widths {
		c.bits += w
	}
	c.run = func(t *testing.T, values []uint64) {
		t.Helper()
		input := printedAccessWire(c, values)
		d, err := ecaccess.LookupContext("9.1.65", name, context)
		if err != nil {
			t.Fatal(err)
		}
		if d.Context != context || d.Name != name || d.Extent.MinimumBits != c.bits || d.Extent.MaximumBits != c.bits {
			t.Fatal("printed identity or extent changed")
		}
		v, err := decode(input)
		if err != nil {
			t.Fatal(err)
		}
		if v.BitsConsumed != c.bits {
			t.Fatal("wrong consumed extent")
		}
		printedValuesEqual(t, fields(v.Value), values)
		dv, err := d.Decode(input)
		if err != nil {
			t.Fatal(err)
		}
		typed, ok := dv.(runtime.Decoded[T])
		if !ok {
			t.Fatal("descriptor context selects a different type")
		}
		printedValuesEqual(t, fields(typed.Value), values)
		w := runtime.NewWriter()
		if err := encodeTo(w, construct(values)); err != nil {
			t.Fatal(err)
		}
		if w.Position() != c.bits {
			t.Fatal("writer extent changed")
		}
		fresh, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(fresh.Bytes, input) {
			t.Fatal("fresh typed fields changed printed order")
		}
		replay, err := encode(v.Value)
		if err != nil || !bytes.Equal(replay, input) {
			t.Fatal("byte replay changed", err)
		}
		canon, err := canonical(v.Value)
		if err != nil || !bytes.Equal(canon, input) {
			t.Fatal("canonical field order changed", err)
		}
		changed := v.Value
		edit(&changed)
		edited, err := encode(changed)
		if err != nil {
			t.Fatal(err)
		}
		expected := append([]byte(nil), input...)
		bit := len(prefix) + widths[0] - 1
		expected[bit/8] ^= 1 << uint(7-bit%8)
		if !bytes.Equal(edited, expected) {
			t.Fatal("typed edit changed an unrelated bit")
		}
		again, err := decode(edited)
		if err != nil {
			t.Fatal(err)
		}
		want := append([]uint64(nil), values...)
		want[0] ^= 1
		printedValuesEqual(t, fields(again.Value), want)
		for n := 0; n < c.bits; n++ {
			r, e := runtime.NewReaderBits(input, n, nil)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := decodeFrom(r); e == nil {
				t.Fatalf("accepted truncation at %d bits", n)
			}
		}
		r, e := runtime.NewReaderBits(input, c.bits, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := decodeFrom(r); e != nil || r.Position() != c.bits {
			t.Fatal("exact reader extent", e)
		}
		badBits := make([]int, 0, len(prefix)+len(suffix))
		for i := range len(prefix) {
			badBits = append(badBits, i)
		}
		if suffix != "" {
			at := len(prefix)
			for _, width := range widths[:bodyFields] {
				at += width
			}
			badBits = append(badBits, at)
		}
		for _, bit := range badBits {
			bad := append([]byte(nil), input...)
			bad[bit/8] ^= 1 << uint(7-bit%8)
			r, e := runtime.NewReaderBits(bad, c.bits, nil)
			if e != nil {
				t.Fatal(e)
			}
			if _, e := decodeFrom(r); e == nil {
				t.Fatal("accepted changed literal bit")
			}
		}
	}
	c.fuzz = func(t *testing.T, input []byte) {
		t.Helper()
		v, err := decode(input)
		if err != nil {
			return
		}
		if v.BitsConsumed != c.bits {
			t.Fatal("wrong fuzz extent")
		}
		out, err := encode(v.Value)
		if err != nil || !bytes.Equal(out, input) {
			t.Fatal("fuzz replay changed", err)
		}
		out, err = canonical(v.Value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := decode(out)
		if err != nil {
			t.Fatal(err)
		}
		printedValuesEqual(t, fields(again.Value), fields(v.Value))
	}
	return c
}

func printedValuesEqual(t *testing.T, got, want []uint64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatal("typed field count")
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("typed field %d: got %d want %d", i, got[i], want[i])
		}
	}
}
func printedAccessWire(c printedAccessCase, values []uint64) []byte {
	var word uint32
	bits := 0
	appendValue := func(v uint64, width int) { word = word<<uint(width) | uint32(v); bits += width }
	for _, b := range c.prefix {
		appendValue(uint64(b-'0'), 1)
	}
	for i, width := range c.widths {
		if i == c.bodyFields {
			for _, b := range c.suffix {
				appendValue(uint64(b-'0'), 1)
			}
		}
		appendValue(values[i], width)
	}
	if len(c.widths) == c.bodyFields {
		for _, b := range c.suffix {
			appendValue(uint64(b-'0'), 1)
		}
	}
	out := make([]byte, (bits+7)/8)
	word <<= uint(len(out)*8 - bits)
	for i := range out {
		out[i] = byte(word >> uint((len(out)-1-i)*8))
	}
	return out
}
func printedAccessValues(c printedAccessCase, sample int) []uint64 {
	values := make([]uint64, len(c.widths))
	for i, width := range c.widths {
		max := uint64(1)<<uint(width) - 1
		switch sample {
		case 0:
			values[i] = 0
		case 1:
			values[i] = max
		default:
			values[i] = uint64(i*3+5) & max
		}
	}
	if c.bits == 30 && sample == 3 {
		copy(values, []uint64{5, 10, 0xabc, 5, 1, 3})
	}
	return values
}
func TestPrintedAccessBursts(t *testing.T) {
	cases := printedAccessCases()
	if len(cases) != 14 || len(ecaccess.Descriptors()) != 14 {
		t.Fatal("missing or merged printed contexts")
	}
	seen := map[string]bool{}
	for i, c := range cases {
		if seen[c.context] {
			t.Fatal("duplicate printed context")
		}
		seen[c.context] = true
		t.Run(fmt.Sprint(i)+"/"+c.name, func(t *testing.T) {
			for sample := 0; sample < 4; sample++ {
				c.run(t, printedAccessValues(c, sample))
			}
		})
	}
}
func FuzzPrintedAccessBursts(f *testing.F) {
	cases := printedAccessCases()
	for i, c := range cases {
		f.Add(uint8(i), printedAccessWire(c, printedAccessValues(c, 3)))
	}
	f.Fuzz(func(t *testing.T, index uint8, input []byte) {
		if len(input) > 8 {
			return
		}
		cases[int(index)%len(cases)].fuzz(t, input)
	})
}
