// Package vgcsreconfigure contains source-selected CSN.1 codecs.
//
// TS 44.018 V19.0.0 §9.1.21h paragraphs 7638–7645 and 7650–7654. The receiver no-string suffix replays exactly; a fresh writer emits only its zero selector. The named canonical API requires a caller extent and reports an unfilled target rather than returning shortened bytes. The caller supplies the extent; the default canonical ceiling is 251 octets (TS 44.006 V19.0.0 §8.8.5 p2466). EncodeVBSVGCSRECONFIGURECanonicalAtLengthUnacknowledged applies an upper bound of 23 octets (§8.8.3 p2456/p2458 and §8.8.5 p2467). DescriptorVBSVGCSRECONFIGUREForUnacknowledgedMode returns the descriptor with the same 23-octet upper bound. No sender padding is added.
package vgcsreconfigure
