// Package ies contains RR value-part codecs from TS 44.018 V19.0.0.
// Restriction Timer is specified in §10.5.2.61 paragraphs 21336–21338;
// its sender clears spare bits while replay preserves received bits.
// EC Request Reference Description is specified in §10.5.2.83 paragraphs
// 22163–22165, with a ten-bit frame and three random bits.
//
// BA List Pref is specified in §10.5.2.1c paragraphs 13985–13992. Its syntax includes the length octet, terminated range and frequency lists, and spare padding.
//
// Packet Channel Description is specified in §10.5.2.25a paragraphs 17795–17811. Its value occupies 24 bits within the four-octet IE; the containing assignment supplies the packet-channel context. Non-hopping, indirect and direct hopping encodings remain distinct.
//
// MBMS Session Identity value part is specified in §10.5.2.62 paragraphs 21367–21369. Its value occupies one octet within the two-octet type 3 IE.
//
// EC Packet Channel Description Type 1 is specified in §10.5.2.84 paragraphs 22184–22191. Its value occupies 16 bits and retains the Type 1 uplink coverage field.
//
// EC Packet Channel Description Type 2 is specified in §10.5.2.85 paragraphs 22247–22253. Its value occupies 15 bits; callers provide the enclosing message bit extent.
//
// EC Packet Channel Description Type 3 is specified in §10.5.2.88 paragraphs 22347–22353. Its value occupies 15 bits and retains the extended uplink coverage field independently of Type 2; callers provide the enclosing message bit extent.
//
// TS 44.018 V19.0.0 §10.5.2.11a defines DTM Information Details value part. Canonical encoding requires the explicit containing value length, at most 255 octets; the H extension terminates in a spare-bit run.
package ies
