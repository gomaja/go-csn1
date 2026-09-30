// Package measurement contains typed CSN.1 codecs for TS 44.018 measurement
// reports.
//
// The pre-Rel-8 Enhanced Measurement Report reporting bitmap has a nominal
// 96 positions. Each transmitted position is a typed entry, including
// redundant no-report positions beyond the serving cell's Neighbour Cell
// list. Trailing no-report positions may be omitted if they cannot fit in
// the message. The Rel-8 bitmap carries its own BITMAP_LENGTH.
//
// TS 44.018 V19.0.0 §§3.4.1.2.1.3, 9.1.55 defines the Neighbour Cell list
// and the two reporting bitmap layouts.
package measurement
