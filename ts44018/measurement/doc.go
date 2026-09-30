// Package measurement contains typed CSN.1 codecs for TS 44.018 measurement
// reports.
//
// The pre-Rel-8 Enhanced Measurement Report reporting bitmap has a nominal
// 96 positions. Each transmitted position is a typed entry, including
// redundant no-report positions beyond the serving cell's Neighbour Cell
// list. Trailing no-report positions may be omitted if they cannot fit in
// the message. The Rel-8 bitmap carries its own BITMAP_LENGTH.
//
// Plain EncodeEnhancedMeasurementReport preserves a decoded value's received
// layout for byte-exact re-encoding. It rejects edits that move a semantic
// boundary or conflict with retained wire state. To encode an edited value as
// a freshly constructed message, use EncodeEnhancedMeasurementReportCanonical.
// It discards received layout throughout an independent copy, checks that
// the result decodes to equivalent typed fields, and leaves the caller's value
// unchanged. The only permitted normalization is the source-defined redundant
// trailing no-report positions. The same Encode<Type>Canonical form is
// generated for every definition. Nested definitions without a standalone
// source maximum require Encode<Type>CanonicalAtLength and otherwise return
// runtime.ExtentError. For a fresh pre-Rel-8 report, two explicit absent
// positions expand to 96 transmitted no-report positions; decoding exposes
// all 96 as typed nil entries. The complete EMR, including its short L2
// header, is limited to 21 octets in the Bter SACCH frame (TS 44.006
// V19.0.0 §§5.1, 5.2, 8.8.3); when redundant trailing no-report bits
// cannot fit, canonical encoding omits them as TS 44.018 §9.1.55 permits.
//
// TS 44.018 V19.0.0 §§3.4.1.2.1.3, 9.1.55 defines the Neighbour Cell list
// and the two reporting bitmap layouts.
package measurement
