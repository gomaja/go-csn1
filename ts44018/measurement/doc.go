// Package measurement contains typed CSN.1 codecs for TS 44.018 measurement
// reports.
//
// TS 44.018 V19.0.0 §9.1.55 requires at least 96 legacy bitmap positions,
// with redundant positions zero and capacity-limited omission of trailing
// no-report positions. Each transmitted position is a typed entry. The
// Rel-8 bitmap carries its own BITMAP_LENGTH. After cell position 96, an
// entirely zero remainder is redundant bitmap through the enclosing input
// limit; otherwise the release chain starts immediately after position 96
// (§3.4.1.2.1.3 and table 9.1.55.1). Fresh legacy bitmaps without release
// additions end in redundant zero positions; those with additions have
// exactly 96 positions before the release chain.
//
// Plain EncodeEnhancedMeasurementReport preserves a decoded value's received
// layout for byte-exact re-encoding, including accepted shorter images. It
// rejects edits that move a semantic boundary or conflict with retained wire
// state. Fresh complete messages and EncodeEnhancedMeasurementReportCanonical
// fill exactly 21 octets, including the short L2 header (TS 44.006 V19.0.0
// §§5.2, 6.4a, 8.8.3). CanonicalAtLength accepts only 21 for this carrier.
// Its descriptor advertises the fixed canonical 168-bit extent separately
// from the receiver's accepted-image behavior.
//
// Canonical encoding discards received layout throughout an independent
// copy, checks that the result decodes to equivalent typed fields, and leaves
// the caller's value unchanged. Redundant trailing no-report positions and
// null versus L no-string arms with no release fields are normalized; explicit
// H arms and their fields remain distinct (TS 44.018 V19.0.0 §9.1.55).
// The same Encode<Type>Canonical form is generated for every definition.
// Nested definitions without a standalone source maximum require
// Encode<Type>CanonicalAtLength and otherwise return runtime.ExtentError.
//
// TS 44.018 V19.0.0 §§3.4.1.2.1.3, 9.1.55 define the Neighbour Cell list
// and the two reporting bitmap layouts.
package measurement
