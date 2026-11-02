// Package messages contains RR message-content codecs from TS 44.018 V19.0.0.
// EC-Dummy (§9.1.62 paragraphs 11859–11861) carries the used downlink coverage
// class and padding. Its canonical encoding needs a caller-supplied length,
// at most 11 octets (TS 45.003 V19.0.0 §5.2b.1).
// Its message-type assignment is in §10.4 table 10.4.4 paragraph 13251.
// The caller supplies the channel and procedure context for these contents.
//
// EC System Information Type 3 (§9.1.43r, paragraphs 10039–10080) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; the authenticated EC-BCCH carrier maximum is 23 octets (TS 45.003 V19.0.0 §5.2a.1). Channel framing and SI instance assembly remain with the caller.
//
// EC System Information Type 1 (§9.1.43p, paragraphs 9743–9770) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; the authenticated EC-BCCH carrier maximum is 23 octets (TS 45.003 V19.0.0 §5.2a.1). Channel framing and SI instance assembly remain with the caller.
//
// EC System Information Type 4 (§9.1.43s, paragraphs 10122–10136) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; the authenticated EC-BCCH carrier maximum is 23 octets (TS 45.003 V19.0.0 §5.2a.1). Channel framing and SI instance assembly remain with the caller.
//
// TS 44.018 V19.0.0 §§9.1.50–9.1.50b define System Information Type 10, 10bis and 10ter. The 160-bit, 160-bit and 32-bit rest-octet carriers delegate to the typed components in §§10.5.2.44, 10.5.2.70 and 10.5.2.71. Each complete message occupies 21 octets; canonical encoding enforces that extent.
//
// TS 44.018 V19.0.0 §9.1.58 defines NOTIFY APPLICATION DATA with a 72-bit application-data field, an eight-bit data identity and an optional length-delimited MSISDN. Terminal spare padding requires an explicit containing extent; the pinned main-DCCH Bter maximum is 23 octets (TS 44.006 V19.0.0 §8.8.3).

// TS 44.018 V19.0.0 §9.1.47 defines Uplink free. The containing message supplies the exact bit extent; no additional numerical carrier cap is assigned by this intake.
//
// TS 44.018 V19.0.0 §9.1.49b defines VGCS SMS Information. The containing message supplies the exact bit extent; no additional numerical carrier cap is assigned by this intake.
// TS 44.018 V19.0.0 §9.1.57 defines VGCS Neighbour Cell Information message. The containing message supplies the exact bit extent; no additional numerical carrier cap is assigned by this intake. TS 44.018 V19.0.0 §9.1.57 p11507 assigns the imperative-message-part-error diagnosis for an all-zero Validity bitmap to the receiving mobile station application. The codec accepts and replays every printed three-bit value.
//
// TS 44.018 V19.0.0 §9.1.49a pp10567–10579. Complete short-header VGCS Additional Information root, including the first header octet; main-DCCH Bter permits at most 23 octets, while SACCH uses 21. The assigned message code is 01000.
// TS 44.018 V19.0.0 §9.1.61 paragraphs 11817–11827. The complete EC RR message includes its printed four-bit discriminator; TS 45.003 V19.0.0 §5.2b.1 provides 88 information bits.
// TS 44.018 V19.0.0 §9.1.63 paragraphs 11878–11895. The complete EC RR message includes its printed four-bit discriminator; TS 45.003 V19.0.0 §5.2b.1 provides 88 information bits.
// TS 44.018 V19.0.0 §9.1.69 paragraphs 12806–12808. The complete EC RR message includes its printed four-bit discriminator; TS 45.003 V19.0.0 §5.2b.1 provides 88 information bits.
package messages
