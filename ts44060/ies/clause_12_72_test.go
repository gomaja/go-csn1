package ies

import (
	"testing"
)

var clause1272Codec = clauseCodec[ECAckNackDescriptionIE]{DecodeECAckNackDescriptionIE, DecodeECAckNackDescriptionIEFrom, EncodeECAckNackDescriptionIE, EncodeECAckNackDescriptionIETo, EncodeECAckNackDescriptionIECanonicalAtLength}
var clause1272Vectors = []clauseVector[ECAckNackDescriptionIE]{
	{name: "ec-12.72-0", hex: "000000", bits: 21, value: ECAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 0, RECEIVEDBLOCKBITMAP: 0}},
	{name: "ec-12.72-31", hex: "fffff8", bits: 21, value: ECAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 31, RECEIVEDBLOCKBITMAP: 65535}},
	{name: "ec-12.72-21", hex: "ad2ad0", bits: 21, value: ECAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 21, RECEIVEDBLOCKBITMAP: 42330}},
	{name: "ec-12.72-1", hex: "080008", bits: 21, value: ECAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 1, RECEIVEDBLOCKBITMAP: 1}},
}

// TS 44.060 V19.0.0 §12.72: exact printed field widths and all selectors.
func TestClause12_72(t *testing.T) {
	for _, vector := range clause1272Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			checkClauseVector(t, clause1272Codec, vector, minimum)
		})
	}
	checkClauseInvalid(t, clause1272Codec, ECAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 32}, 3)
}
