package ies

import (
	"testing"
)

var clause1276Codec = clauseCodec[ECChannelQualityReportHigherCCIE]{DecodeECChannelQualityReportHigherCCIE, DecodeECChannelQualityReportHigherCCIEFrom, EncodeECChannelQualityReportHigherCCIE, EncodeECChannelQualityReportHigherCCIETo, EncodeECChannelQualityReportHigherCCIECanonicalAtLength}
var clause1276Vectors = []clauseVector[ECChannelQualityReportHigherCCIE]{
	{name: "quality-0-0", hex: "00", bits: 7, value: ECChannelQualityReportHigherCCIE{CVALUE: 0, GMSKMEANBEPGroup: nil}},
	{name: "quality-0-63", hex: "7e", bits: 7, value: ECChannelQualityReportHigherCCIE{CVALUE: 63, GMSKMEANBEPGroup: nil}},
	{name: "quality-1-0", hex: "8000", bits: 15, value: ECChannelQualityReportHigherCCIE{CVALUE: 0, GMSKMEANBEPGroup: &ECChannelQualityReportHigherCCIEGMSKMEANBEPGroup{GMSKMEANBEP: 0, GMSKCVBEP: 0}}},
	{name: "quality-1-63", hex: "fffe", bits: 15, value: ECChannelQualityReportHigherCCIE{CVALUE: 63, GMSKMEANBEPGroup: &ECChannelQualityReportHigherCCIEGMSKMEANBEPGroup{GMSKMEANBEP: 31, GMSKCVBEP: 7}}},
	{name: "quality-1-42", hex: "d6d4", bits: 15, value: ECChannelQualityReportHigherCCIE{CVALUE: 42, GMSKMEANBEPGroup: &ECChannelQualityReportHigherCCIEGMSKMEANBEPGroup{GMSKMEANBEP: 21, GMSKCVBEP: 5}}},
}

// TS 44.060 V19.0.0 §12.76: exact printed field widths and all selectors.
func TestClause12_76(t *testing.T) {
	for _, vector := range clause1276Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			checkClauseVector(t, clause1276Codec, vector, minimum)
		})
	}
	checkClauseInvalid(t, clause1276Codec, ECChannelQualityReportHigherCCIE{CVALUE: 64}, (15+7)/8)
	checkClauseInvalid(t, clause1276Codec, ECChannelQualityReportHigherCCIE{GMSKMEANBEPGroup: &ECChannelQualityReportHigherCCIEGMSKMEANBEPGroup{GMSKMEANBEP: 32}}, (15+7)/8)
	checkClauseInvalid(t, clause1276Codec, ECChannelQualityReportHigherCCIE{GMSKMEANBEPGroup: &ECChannelQualityReportHigherCCIEGMSKMEANBEPGroup{GMSKCVBEP: 8}}, (15+7)/8)
}
