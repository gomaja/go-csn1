package controlmessages_test

import (
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.14: complete synthetic CS-1 control blocks.
func TestClause11214Content(t *testing.T) {
	t.Run("page0-persistenceFalse", func(t *testing.T) {
		simpleExercise(t, 36, false, "00101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", nil)
		}, "PAGEMODE")
	})
	t.Run("page0-persistenceTrue", func(t *testing.T) {
		simpleExercise(t, 36, false, "001010010101011010000110110010110110000010110101111", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", []uint8{0, 5, 10, 15})
		}, "PAGEMODE")
	})
	t.Run("page1-persistenceFalse", func(t *testing.T) {
		simpleExercise(t, 36, false, "01101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", nil)
		}, "PAGEMODE")
	})
	t.Run("page1-persistenceTrue", func(t *testing.T) {
		simpleExercise(t, 36, false, "011010010101011010000110110010110110000010110101111", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", []uint8{0, 5, 10, 15})
		}, "PAGEMODE")
	})
	t.Run("page2-persistenceFalse", func(t *testing.T) {
		simpleExercise(t, 36, false, "10101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", nil)
		}, "PAGEMODE")
	})
	t.Run("page2-persistenceTrue", func(t *testing.T) {
		simpleExercise(t, 36, false, "101010010101011010000110110010110110000010110101111", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", []uint8{0, 5, 10, 15})
		}, "PAGEMODE")
	})
	t.Run("page3-persistenceFalse", func(t *testing.T) {
		simpleExercise(t, 36, false, "11101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", nil)
		}, "PAGEMODE")
	})
	t.Run("page3-persistenceTrue", func(t *testing.T) {
		simpleExercise(t, 36, false, "111010010101011010000110110010110110000010110101111", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "ACCCONTRCLASS", 42330)
			simpleUint(t, v, "S", 2)
			simpleUint(t, v, "TXINT", 13)
			simpleList(t, v, "MAXRETRANSList", []uint8{0, 1, 2, 3})
			simpleList(t, v, "PERSISTENCELEVELList", []uint8{0, 5, 10, 15})
		}, "PAGEMODE")
	})
}

func FuzzClause11214Content(f *testing.F) {
	simpleFuzz(f, 36, false, "00101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo)
}

func TestClause11214Truncation(t *testing.T) {
	simpleTruncation(t, 36, false, "00101001010101101000011011001011010", 1, runtime.DistributionPartError, 14, controlmessages.DecodePacketPRACHParametersMessageContentFrom, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {})
	simpleTruncation(t, 36, false, "00101001010101101000011011001011010", 20, runtime.DistributionPartError, 14, controlmessages.DecodePacketPRACHParametersMessageContentFrom, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.
func TestClause11214Ignore(t *testing.T) {
	simpleIgnore(t, 36, false, "00101001010101101000011011001011010", controlmessages.DecodePacketPRACHParametersMessageContentFrom, controlmessages.EncodePacketPRACHParametersMessageContentTo, func(t testing.TB, v controlmessages.PacketPRACHParametersMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}
