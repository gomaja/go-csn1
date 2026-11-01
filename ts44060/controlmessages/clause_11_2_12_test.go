package controlmessages_test

import (
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.12: complete synthetic CS-1 control blocks.
func TestClause11212Content(t *testing.T) {
	t.Run("global-uplink-ack0-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "100010011010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("global-uplink-ack0-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1000100110110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("global-uplink-ack0-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1000100110111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("global-uplink-ack1-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "100010011110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("global-uplink-ack1-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1000100111110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("global-uplink-ack1-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1000100111111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack0-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "100110011010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack0-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1001100110110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack0-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1001100110111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack1-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "100110011110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack1-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1001100111110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("global-downlink-ack1-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1001100111111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack0-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "101000010000001000000011000001000000010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack0-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1010000100000010000000110000010000000110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack0-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1010000100000010000000110000010000000111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack1-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "101000010000001000000011000001000000110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack1-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1010000100000010000000110000010000001110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("tlli-ack1-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1010000100000010000000110000010000001111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TLLIGRNTI", 270544960)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack0-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "101101010010101011010010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TQI", 42330)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack0-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1011010100101010110100110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TQI", 42330)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack0-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1011010100101010110100111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 0)
			simpleUint(t, v, "TQI", 42330)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack1-extensionNone", func(t *testing.T) {
		simpleExercise(t, 4, false, "101101010010101011010110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TQI", 42330)
			simpleOptional(t, v, "GRNTIExtension", nil)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack1-extension0", func(t *testing.T) {
		simpleExercise(t, 4, false, "1011010100101010110101110000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TQI", 42330)
			simpleUint(t, v, "GRNTIExtension", 0)
		}, "PAGEMODE")
	})
	t.Run("tqi-ack1-extension15", func(t *testing.T) {
		simpleExercise(t, 4, false, "1011010100101010110101111111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TYPEOFACK", 1)
			simpleUint(t, v, "TQI", 42330)
			simpleUint(t, v, "GRNTIExtension", 15)
		}, "PAGEMODE")
	})
}

func FuzzClause11212Content(f *testing.F) {
	simpleFuzz(f, 4, false, "100010011010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo)
}

func TestClause11212Truncation(t *testing.T) {
	simpleTruncation(t, 4, false, "100010011010", 1, runtime.DistributionPartError, 14, controlmessages.DecodePacketPollingRequestMessageContentFrom, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {})
	simpleTruncation(t, 4, false, "100010011010", 3, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketPollingRequestMessageContentFrom, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
		simpleUint(t, v, "PAGEMODE", 2)
	})
	simpleTruncation(t, 4, false, "100010011010", 9, runtime.NonDistributionPartError, 23, controlmessages.DecodePacketPollingRequestMessageContentFrom, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
		simpleUint(t, v, "PAGEMODE", 2)
	})
}

// TS 44.060 V19.0.0 §11.2.12: null and earlier-release receiver alternatives.
func TestClause11212EarlierRelease(t *testing.T) {
	t.Run("global-uplink-ack0-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1000100110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 0)
		})
	})
	t.Run("global-uplink-ack1-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1000100111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 1)
		})
	})
	t.Run("global-downlink-ack0-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1001100110", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 0)
		})
	})
	t.Run("global-downlink-ack1-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1001100111", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 1)
		})
	})
	t.Run("tlli-ack0-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1010000100000010000000110000010000000", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 0)
		})
	})
	t.Run("tlli-ack1-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1010000100000010000000110000010000001", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 1)
		})
	})
	t.Run("tqi-ack0-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1011010100101010110100", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 0)
		})
	})
	t.Run("tqi-ack1-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 4, false, "1011010100101010110101", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
			simpleUint(t, v, "TYPEOFACK", 1)
		})
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.
func TestClause11212Ignore(t *testing.T) {
	simpleIgnore(t, 4, false, "100010011010", controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
		simpleUint(t, v, "PAGEMODE", 2)
	})
}

func TestClause11212MalformedSelector(t *testing.T) {
	b := simpleBlock(t, 4, "101110011010")
	simpleRegionReplay(t, b, 4, false, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketPollingRequestMessageContentFrom, controlmessages.EncodePacketPollingRequestMessageContentTo, func(t testing.TB, v controlmessages.PacketPollingRequestMessageContent) {
		simpleUint(t, v, "PAGEMODE", 2)
	})
}
