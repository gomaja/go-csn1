package controlmessages_test

import (
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.9c: complete synthetic CS-1 control blocks.
func TestClause1129cContent(t *testing.T) {
	t.Run("tfi0-cause0-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause0-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause0-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause0-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause0-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause0-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011000110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause1-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011001110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause2-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011010110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause3-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011011110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause4-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011100110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause5-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011101110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause6-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011110110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi0-cause7-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "010011111110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "UPLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause0-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011000110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause1-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011001110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause2-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011010110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause3-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011011110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause4-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011100110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause5-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011101110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause6-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011110110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-statusNone-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-statusNone-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111011000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 0)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-statusNone-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111011111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "DownlinkETFI", 7)
			simpleOptional(t, v, "STATUSMESSAGETYPE", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-status35-extensionNone", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111110001110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleOptional(t, v, "DownlinkETFI", nil)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-status35-extension0", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111110001111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 0)
		}, "TBFCAUSE")
	})
	t.Run("tfi1-cause7-status35-extension7", func(t *testing.T) {
		simpleExercise(t, 6, true, "110011111110001111111", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
			simpleUint(t, v, "DOWNLINKTFI", 19)
			simpleUint(t, v, "STATUSMESSAGETYPE", 35)
			simpleUint(t, v, "DownlinkETFI", 7)
		}, "TBFCAUSE")
	})
}

func FuzzClause1129cContent(f *testing.F) {
	simpleFuzz(f, 6, true, "010011000010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo)
}

// TS 44.060 V19.0.0 §11.2.9c: null and earlier-release receiver alternatives.
func TestClause1129cEarlierRelease(t *testing.T) {
	t.Run("tfi0-cause0-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
		})
	})
	t.Run("tfi0-cause0-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110001100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
		})
	})
	t.Run("tfi0-cause1-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
		})
	})
	t.Run("tfi0-cause1-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110011100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
		})
	})
	t.Run("tfi0-cause2-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110100", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
		})
	})
	t.Run("tfi0-cause2-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110101100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
		})
	})
	t.Run("tfi0-cause3-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
		})
	})
	t.Run("tfi0-cause3-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100110111100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
		})
	})
	t.Run("tfi0-cause4-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
		})
	})
	t.Run("tfi0-cause4-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111001100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
		})
	})
	t.Run("tfi0-cause5-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
		})
	})
	t.Run("tfi0-cause5-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111011100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
		})
	})
	t.Run("tfi0-cause6-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111100", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
		})
	})
	t.Run("tfi0-cause6-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111101100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
		})
	})
	t.Run("tfi0-cause7-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
		})
	})
	t.Run("tfi0-cause7-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "0100111111100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
		})
	})
	t.Run("tfi1-cause0-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
		})
	})
	t.Run("tfi1-cause0-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110001100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 0)
		})
	})
	t.Run("tfi1-cause1-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
		})
	})
	t.Run("tfi1-cause1-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110011100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 1)
		})
	})
	t.Run("tfi1-cause2-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110100", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
		})
	})
	t.Run("tfi1-cause2-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110101100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 2)
		})
	})
	t.Run("tfi1-cause3-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
		})
	})
	t.Run("tfi1-cause3-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100110111100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 3)
		})
	})
	t.Run("tfi1-cause4-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111000", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
		})
	})
	t.Run("tfi1-cause4-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111001100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 4)
		})
	})
	t.Run("tfi1-cause5-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
		})
	})
	t.Run("tfi1-cause5-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111011100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 5)
		})
	})
	t.Run("tfi1-cause6-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111100", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
		})
	})
	t.Run("tfi1-cause6-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111101100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 6)
		})
	})
	t.Run("tfi1-cause7-statusNone-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111110", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
		})
	})
	t.Run("tfi1-cause7-status35-extensionNone", func(t *testing.T) {
		simpleLegacy(t, 6, true, "1100111111100011", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
			simpleUint(t, v, "TBFCAUSE", 7)
		})
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.
func TestClause1129cIgnore(t *testing.T) {
	simpleIgnore(t, 6, true, "010011000010", controlmessages.DecodePacketMobileTBFStatusMessageContentFrom, controlmessages.EncodePacketMobileTBFStatusMessageContentTo, func(t testing.TB, v controlmessages.PacketMobileTBFStatusMessageContent) {
		simpleUint(t, v, "TBFCAUSE", 0)
	})
}

// TS 44.060 V19.0.0 §11.2.9c paragraphs 16050–16052 requires
// the Global TFI, cause and status-presence flag before the nullable suffix.
func TestClause1129cMandatoryTruncation(t *testing.T) {
	for _, bits := range []int{0, 1, 5, 6, 8, 9} {
		b := simpleBlock(t, 6, "0100110000")
		r, c := simpleReader(t, b, 6, true)
		if _, err := r.PushLimit(bits); err != nil {
			t.Fatal(err)
		}
		_, err := controlmessages.DecodePacketMobileTBFStatusMessageContentFrom(r)
		var located *runtime.DecodeError
		if !errors.As(err, &located) || r.Position() > 14+bits || len(c.TakeReceiverReport().Regions) != 0 {
			t.Fatalf("mandatory truncation at %d: %T %v", bits, err, err)
		}
	}
}
