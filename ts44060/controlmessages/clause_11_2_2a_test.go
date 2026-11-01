package controlmessages_test

import (
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.2a: complete synthetic CS-1 control blocks.
func TestClause1122aContent(t *testing.T) {
	t.Run("page0-tfi0-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "0000100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page0-tfi0-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "0000100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page0-tfi1-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "0001100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page0-tfi1-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "0001100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page1-tfi0-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "0100100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page1-tfi0-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "0100100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page1-tfi1-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "0101100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page1-tfi1-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "0101100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page2-tfi0-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "1000100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page2-tfi0-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "1000100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page2-tfi1-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "1001100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page2-tfi1-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "1001100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page3-tfi0-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "1100100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page3-tfi0-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "1100100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
	t.Run("page3-tfi1-cell0", func(t *testing.T) {
		simpleExercise(t, 11, false, "1101100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
		}, "PAGEMODE")
	})
	t.Run("page3-tfi1-cell1", func(t *testing.T) {
		simpleExercise(t, 11, false, "1101100111101010000110110111", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "ARFCN", 673)
			simpleUint(t, v, "BSIC", 45)
			simpleUint(t, v, "CONTAINERID", 3)
		}, "PAGEMODE")
	})
}

func FuzzClause1122aContent(f *testing.F) {
	simpleFuzz(f, 11, false, "0000100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo)
}

func TestClause1122aTruncation(t *testing.T) {
	simpleTruncation(t, 11, false, "0000100111101010000110110111", 1, runtime.DistributionPartError, 14, controlmessages.DecodePacketCellChangeContinueMessageContentFrom, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {})
	simpleTruncation(t, 11, false, "0000100111101010000110110111", 3, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketCellChangeContinueMessageContentFrom, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
	simpleTruncation(t, 11, false, "0000100111101010000110110111", 10, runtime.NonDistributionPartError, 23, controlmessages.DecodePacketCellChangeContinueMessageContentFrom, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.
func TestClause1122aIgnore(t *testing.T) {
	simpleIgnore(t, 11, false, "0000100110", controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}

func TestClause1122aMalformedSelector(t *testing.T) {
	b := simpleBlock(t, 11, "0010100110")
	simpleRegionReplay(t, b, 11, false, runtime.AddressInformationPartError, 16, controlmessages.DecodePacketCellChangeContinueMessageContentFrom, controlmessages.EncodePacketCellChangeContinueMessageContentTo, func(t testing.TB, v controlmessages.PacketCellChangeContinueMessageContent) {
		simpleUint(t, v, "PAGEMODE", 0)
	})
}
