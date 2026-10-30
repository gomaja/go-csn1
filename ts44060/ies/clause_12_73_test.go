package ies

import (
	"testing"
)

var clause1273Codec = clauseCodec[ECPrimaryAckNackDescriptionIE]{DecodeECPrimaryAckNackDescriptionIE, DecodeECPrimaryAckNackDescriptionIEFrom, EncodeECPrimaryAckNackDescriptionIE, EncodeECPrimaryAckNackDescriptionIETo, EncodeECPrimaryAckNackDescriptionIECanonicalAtLength}
var clause1273Vectors = []clauseVector[ECPrimaryAckNackDescriptionIE]{
	{name: "ec-12.73-0", hex: "0000", bits: 13, value: ECPrimaryAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 0, RECEIVEDBLOCKBITMAP: 0}},
	{name: "ec-12.73-31", hex: "fff8", bits: 13, value: ECPrimaryAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 31, RECEIVEDBLOCKBITMAP: 255}},
	{name: "ec-12.73-21", hex: "ad28", bits: 13, value: ECPrimaryAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 21, RECEIVEDBLOCKBITMAP: 165}},
	{name: "ec-12.73-1", hex: "0808", bits: 13, value: ECPrimaryAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 1, RECEIVEDBLOCKBITMAP: 1}},
}

// TS 44.060 V19.0.0 §12.73: exact printed field widths and all selectors.
func TestClause12_73(t *testing.T) {
	for _, vector := range clause1273Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			checkClauseVector(t, clause1273Codec, vector, minimum)
		})
	}
	checkClauseInvalid(t, clause1273Codec, ECPrimaryAckNackDescriptionIE{STARTINGSEQUENCENUMBER: 32}, 2)
}
