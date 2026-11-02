package controlmessages_test

import (
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.15: complete synthetic CS-1 control blocks.
func TestClause11215Content(t *testing.T) {
	t.Run("page0", func(t *testing.T) {
		simpleExercise(t, 6, false, "001111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "RANDOMACCESSINFORMATIONValue", 1443)
			simpleUint(t, v, "FRAMENUMBER", 46442)
			simpleUint(t, v, "TQI", 42330)
		}, "PAGEMODE")
	})
	t.Run("page1", func(t *testing.T) {
		simpleExercise(t, 6, false, "011111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "RANDOMACCESSINFORMATIONValue", 1443)
			simpleUint(t, v, "FRAMENUMBER", 46442)
			simpleUint(t, v, "TQI", 42330)
		}, "PAGEMODE")
	})
	t.Run("page2", func(t *testing.T) {
		simpleExercise(t, 6, false, "101111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "RANDOMACCESSINFORMATIONValue", 1443)
			simpleUint(t, v, "FRAMENUMBER", 46442)
			simpleUint(t, v, "TQI", 42330)
		}, "PAGEMODE")
	})
	t.Run("page3", func(t *testing.T) {
		simpleExercise(t, 6, false, "111111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "RANDOMACCESSINFORMATIONValue", 1443)
			simpleUint(t, v, "FRAMENUMBER", 46442)
			simpleUint(t, v, "TQI", 42330)
		}, "PAGEMODE")
	})
}

func FuzzClause11215Content(f *testing.F) {
	simpleFuzz(f, 6, false, "001111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo)
}

func TestClause11215Truncation(t *testing.T) {
	simpleTruncation(t, 6, false, "001111011010001110110101011010101010010101011010", 1, runtime.DistributionPartError, 14, controlmessages.DecodePacketQueueingNotificationMessageContentFrom, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {})
	simpleTruncation(t, 6, false, "001111011010001110110101011010101010010101011010", 4, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketQueueingNotificationMessageContentFrom, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
	simpleTruncation(t, 6, false, "001111011010001110110101011010101010010101011010", 33, runtime.NonDistributionPartError, 46, controlmessages.DecodePacketQueueingNotificationMessageContentFrom, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.
func TestClause11215Ignore(t *testing.T) {
	simpleIgnore(t, 6, false, "001111011010001110110101011010101010010101011010", controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}

func TestClause11215MalformedSelector(t *testing.T) {
	b := simpleBlock(t, 6, "001101011010001110110101011010101010010101011010")
	simpleRegionReplay(t, b, 6, false, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketQueueingNotificationMessageContentFrom, controlmessages.EncodePacketQueueingNotificationMessageContentTo, func(t testing.TB, v controlmessages.PacketQueueingNotificationMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}
