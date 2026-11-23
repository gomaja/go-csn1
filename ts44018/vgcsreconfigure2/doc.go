// Package vgcsreconfigure2 contains source-selected CSN.1 codecs.
//
// TS 44.018 V19.0.0 §9.1.21i defines VBS/VGCS reconfigure2. The caller supplies the extent; the default canonical ceiling is 251 octets (TS 44.006 V19.0.0 §8.8.5 p2466). EncodeVBSVGCSRECONFIGURE2CanonicalAtLengthUnacknowledged applies an upper bound of 23 octets (§8.8.3 p2456/p2458 and §8.8.5 p2467). DescriptorVBSVGCSRECONFIGURE2ForUnacknowledgedMode returns the descriptor with the same 23-octet upper bound. No sender padding is added.
package vgcsreconfigure2
