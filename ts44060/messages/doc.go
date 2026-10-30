// Package messages contains access-burst payload codecs from TS 44.060 V19.0.0.
// Handover Access (§11.2.33 paragraphs 19930–19931) carries an eight-bit
// handover reference for Iu-mode access. It has no control-block message-type
// prefix. The caller supplies the physical-channel context and burst repetition
// required by §11.2.33 paragraph 19927.
//
// Handover Access (TS 44.060 V19.0.0 §11.2.33 paragraphs 19930–19931) carries an eight-bit handover reference on the Iu-mode access burst. The caller supplies the physical-channel context and burst repetition required by paragraph 19927.
package messages
