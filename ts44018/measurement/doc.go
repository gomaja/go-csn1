// Package measurement contains typed CSN.1 codecs for TS 44.018 measurement
// reports.
//
// The pre-Rel-8 Enhanced Measurement Report reporting bitmap has no on-wire
// length. DecodeEnhancedMeasurementReportWithContext takes the count of cells
// in the serving cell's Neighbour Cell list, derived from SI2, SI2bis, SI2ter,
// SI2quater, Measurement Information, or SI5-family information. It reads that
// many bitmap entries before the release additions and spare padding.
// DecodeEnhancedMeasurementReport returns ErrNeighbourCellCountRequired when
// this bitmap is present without that context. The Rel-8 bitmap carries its
// own BITMAP_LENGTH and works through either API.
//
// TS 44.018 V19.0.0 §§3.4.1.2.1.3, 9.1.55 defines the Neighbour Cell list
// and the two reporting bitmap layouts. A transmitted pre-Rel-8 bitmap has
// at least 96 positions when it fits; smaller counts are useful for isolated
// codec tests but are not complete over-the-air reports.
package measurement
