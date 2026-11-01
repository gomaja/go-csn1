// Package celldata contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §11.2.17b paragraphs 16669-16694; content with externally supplied bit extent.
//
// TS 44.060 V19.0.0 §11.2.17c pp16739–16743, pp16749–16750. Packet Serving Cell SI consumes every complete remaining octet inside the supplied containing extent; residual bits belong to padding. Canonical targets respect the active local limit.
package celldata
