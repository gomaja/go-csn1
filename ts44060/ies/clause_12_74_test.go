package ies

import (
	"testing"
)

var clause1274Codec = clauseCodec[ECPacketTimingAdvanceIE]{DecodeECPacketTimingAdvanceIE, DecodeECPacketTimingAdvanceIEFrom, EncodeECPacketTimingAdvanceIE, EncodeECPacketTimingAdvanceIETo, EncodeECPacketTimingAdvanceIECanonicalAtLength}
var clause1274Vectors = []clauseVector[ECPacketTimingAdvanceIE]{
	{name: "timing-0", hex: "00", bits: 6, value: ECPacketTimingAdvanceIE{TIMINGADVANCEVALUE: 0}},
	{name: "timing-1", hex: "04", bits: 6, value: ECPacketTimingAdvanceIE{TIMINGADVANCEVALUE: 1}},
	{name: "timing-42", hex: "a8", bits: 6, value: ECPacketTimingAdvanceIE{TIMINGADVANCEVALUE: 42}},
	{name: "timing-63", hex: "fc", bits: 6, value: ECPacketTimingAdvanceIE{TIMINGADVANCEVALUE: 63}},
}

// TS 44.060 V19.0.0 §12.74: exact printed field widths and all selectors.
func TestClause12_74(t *testing.T) {
	for _, vector := range clause1274Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			checkClauseVector(t, clause1274Codec, vector, minimum)
		})
	}
	checkClauseInvalid(t, clause1274Codec, ECPacketTimingAdvanceIE{TIMINGADVANCEVALUE: 64}, 1)
}
