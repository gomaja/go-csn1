package controlmessages_test

import (
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

func TestClause11211Content(t *testing.T) {
	t.Run("page0-timeslots0", func(t *testing.T) {
		simpleExercise(t, 35, false, "00100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 0)
		}, "PAGEMODE")
	})
	t.Run("page0-timeslots165", func(t *testing.T) {
		simpleExercise(t, 35, false, "00110100101", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 165)
		}, "PAGEMODE")
	})
	t.Run("page0-timeslots255", func(t *testing.T) {
		simpleExercise(t, 35, false, "00111111111", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 0)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 255)
		}, "PAGEMODE")
	})
	t.Run("page1-timeslots0", func(t *testing.T) {
		simpleExercise(t, 35, false, "01100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 0)
		}, "PAGEMODE")
	})
	t.Run("page1-timeslots165", func(t *testing.T) {
		simpleExercise(t, 35, false, "01110100101", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 165)
		}, "PAGEMODE")
	})
	t.Run("page1-timeslots255", func(t *testing.T) {
		simpleExercise(t, 35, false, "01111111111", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 1)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 255)
		}, "PAGEMODE")
	})
	t.Run("page2-timeslots0", func(t *testing.T) {
		simpleExercise(t, 35, false, "10100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 0)
		}, "PAGEMODE")
	})
	t.Run("page2-timeslots165", func(t *testing.T) {
		simpleExercise(t, 35, false, "10110100101", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 165)
		}, "PAGEMODE")
	})
	t.Run("page2-timeslots255", func(t *testing.T) {
		simpleExercise(t, 35, false, "10111111111", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 2)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 255)
		}, "PAGEMODE")
	})
	t.Run("page3-timeslots0", func(t *testing.T) {
		simpleExercise(t, 35, false, "11100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 0)
		}, "PAGEMODE")
	})
	t.Run("page3-timeslots165", func(t *testing.T) {
		simpleExercise(t, 35, false, "11110100101", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 165)
		}, "PAGEMODE")
	})
	t.Run("page3-timeslots255", func(t *testing.T) {
		simpleExercise(t, 35, false, "11111111111", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {
			simpleUint(t, v, "PAGEMODE", 3)
			simpleUint(t, v, "TIMESLOTSAVAILABLE", 255)
		}, "PAGEMODE")
	})
}

func FuzzClause11211Content(f *testing.F) {
	simpleFuzz(f, 35, false, "00100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo)
}

// Truncated content keeps the production block origin and a narrower explicit
// containing bound. TS 44.060 §§11.1.2–11.1.3 selects the nearest printed region.
func TestClause11211Truncation(t *testing.T) {
	simpleTruncation(t, 35, false, "00100000000", 1, runtime.DistributionPartError, 14, controlmessages.DecodePacketPDCHReleaseMessageContentFrom, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {})
	simpleTruncation(t, 35, false, "00100000000", 10, runtime.DistributionPartError, 14, controlmessages.DecodePacketPDCHReleaseMessageContentFrom, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {})
}

func TestClause11211Ignore(t *testing.T) {
	simpleIgnore(t, 35, false, "00100000000", controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) { simpleUint(t, v, "PAGEMODE", 0) })
}

func TestClause11211MalformedSelector(t *testing.T) {
	b := simpleBlock(t, 35, "00000000000")
	simpleRegionReplay(t, b, 35, false, runtime.DistributionPartError, 14, controlmessages.DecodePacketPDCHReleaseMessageContentFrom, controlmessages.EncodePacketPDCHReleaseMessageContentTo, func(t testing.TB, v controlmessages.PacketPDCHReleaseMessageContent) {})
}

// Spare-padding bits are retained on receipt. Canonical sending restores the
// positional pattern; only the printed leading 1 selects the Ignore arm.
