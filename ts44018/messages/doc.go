// Package messages contains RR message-content codecs from TS 44.018 V19.0.0.
// EC-Dummy (§9.1.62 paragraphs 11859–11861) carries the used downlink coverage
// class and padding. Its canonical encoding needs a caller-supplied length.
// Its message-type assignment is in §10.4 table 10.4.4 paragraph 13251.
// The caller supplies the channel and procedure context for these contents.
//
// EC System Information Type 3 (§9.1.43r, paragraphs 10039–10080) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; no numerical EC-BCCH carrier maximum is established in the permitted catalogue (§10.1 p13004). This unresolved carrier limit leaves channel framing and SI instance assembly to the caller.
//
// EC System Information Type 1 (§9.1.43p, paragraphs 9743–9770) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; no numerical EC-BCCH carrier maximum is established in the permitted catalogue (§10.1 p13004). This unresolved carrier limit leaves channel framing and SI instance assembly to the caller.
//
// EC System Information Type 4 (§9.1.43s, paragraphs 10122–10136) provides EC-BCCH message contents. Canonical padding requires a caller-supplied extent; no numerical EC-BCCH carrier maximum is established in the permitted catalogue (§10.1 p13004). This unresolved carrier limit leaves channel framing and SI instance assembly to the caller.
//
// TS 44.018 V19.0.0 §§9.1.50–9.1.50b define System Information Type 10, 10bis and 10ter. The 160-bit, 160-bit and 32-bit rest-octet carriers delegate to the typed components in §§10.5.2.44, 10.5.2.70 and 10.5.2.71. Each complete message occupies 21 octets; canonical encoding enforces that extent.
//
// TS 44.018 V19.0.0 §9.1.58 defines NOTIFY APPLICATION DATA with a 72-bit application-data field, an eight-bit data identity and an optional length-delimited MSISDN. Terminal spare padding requires an explicit containing extent; no numerical maximum is recorded from the pinned source.

package messages
