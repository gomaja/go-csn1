// Package ies contains RR value-part codecs from TS 44.018 V19.0.0.
// Restriction Timer is specified in §10.5.2.61 paragraphs 21336–21338;
// its sender clears spare bits while replay preserves received bits.
// EC Request Reference Description is specified in §10.5.2.83 paragraphs
// 22163–22165, with a ten-bit frame and three random bits.
//
// BA List Pref is specified in §10.5.2.1c paragraphs 13985–13992. Its syntax includes the length octet, terminated range and frequency lists, and spare padding. Relating that length field to the total extent is unresolved; canonical explicit lengths are capped at 247 root octets (TS 44.006 V19.0.0 §8.8.5; CHANNEL RELEASE header and IEI excluded).
//
// Packet Channel Description is specified in §10.5.2.25a paragraphs 17795–17811. Its value occupies 24 bits within the four-octet IE; the containing assignment supplies the packet-channel context. Non-hopping, indirect and direct hopping encodings remain distinct.
//
// MBMS Session Identity value part is specified in §10.5.2.62 paragraphs 21367–21369. Its value occupies one octet within the two-octet type 3 IE (§10.5.2.62 p21347); canonical explicit lengths are capped at one octet.
//
// EC Packet Channel Description Type 1 is specified in §10.5.2.84 paragraphs 22184–22191. Its value occupies 16 bits and retains the Type 1 uplink coverage field.
//
// EC Packet Channel Description Type 2 is specified in §10.5.2.85 paragraphs 22247–22253. Its value occupies 15 bits; callers provide the enclosing message bit extent.
//
// EC Packet Channel Description Type 3 is specified in §10.5.2.88 paragraphs 22347–22353. Its value occupies 15 bits and retains the extended uplink coverage field independently of Type 2; callers provide the enclosing message bit extent.
//
// TS 44.018 V19.0.0 §10.5.2.11a defines DTM Information Details value part. Canonical encoding requires the explicit containing value length, at most 255 octets; the H extension terminates in a spare-bit run.
//
// TS 44.018 V19.0.0 §10.5.2.11b DOCX paragraphs 15140–15149. The root includes the length octet, a terminated list of 31-bit entries and terminal octet spare bits. The two local entry-name corrections have authenticated correction receipts. The phase A §10.5.2.1e type-4 minimum precedent accounts for the whole IE stated at p15136: 6-1=5 octets for this LV root. HANDOVER COMMAND table 9.1.15.1 pp6317–6322 permits at most 34 whole-IE octets, so this root has at most 33. Oversize received images retain exact replay; explicit canonical lengths obey the carrier cap. The p15136 reference to TS 44.060 is retained without replacement.
// TS 44.018 V19.0.0 §10.5.2.14d, grammar pp16001–16004, delegates typed components to TS 44.060 through the authenticated standard-only quotations; the containing IE supplies the value-part extent. TS 44.060 §12.24 gives at most 98 cell-option bits; §12.9a supplies 19 power-control bits. Under p15981 the value-part maximum is ceil((98+19)/8)=15 octets.
// TS 44.018 V19.0.0 §10.5.2.1d pp14000–14007; the reviewed reading places the primitive types inside their field brackets without changing any bit. The root includes its length octet and two terminated 14-bit frequency lists; IEI is excluded. CHANNEL RELEASE table 9.1.7.1 pp4880–4902,4933–4938 and TS 44.006 V19.0.0 §8.8.5 p2466 bound the root to 251-3-1=247 octets including LENGTH and excluding IEI.
// TS 44.018 V19.0.0 §10.5.2.58, grammar pp21213–21215, delegates typed components to TS 44.060 through the authenticated standard-only quotations; the containing IE supplies the value-part extent. The type-4 value excludes IEI and LENGTH and cannot exceed 255 octets (TS 24.007 V20.0.0 §11.2.1.1.4 pp3025,3043). A selected actual carrier can impose a smaller limit.
// TS 44.018 V19.0.0 §10.5.2.58a, grammar pp21244–21246, delegates typed components to TS 44.060 through the authenticated standard-only quotations; the containing IE supplies the value-part extent. The type-4 value excludes IEI and LENGTH and cannot exceed 255 octets (TS 24.007 V20.0.0 §11.2.1.1.4 pp3025,3043). A selected actual carrier can impose a smaller limit. The p21223 four-octet whole-IE minimum leaves at least two value octets, following the phase A §10.5.2.1e precedent. The p21224 MBMS ANNOUNCEMENT restriction (eight whole-IE octets and specific entries) requires that explicit carrier context.
// TS 44.018 V19.0.0 §10.5.2.60, grammar pp21308–21310, delegates typed components to TS 44.060 through the authenticated standard-only quotations; the containing IE supplies the value-part extent. The type-4 value excludes IEI and LENGTH and cannot exceed 255 octets (TS 24.007 V20.0.0 §11.2.1.1.4 pp3025,3043). A selected actual carrier can impose a smaller limit.
// TS 44.018 V19.0.0 §10.5.2.75 DOCX paragraphs 21785–21825. The value includes delete/provide, 3G and E-UTRAN priority lists, T3230 and the Rel-11 extended-EARFCN branch. All helper references resolve within this clause.
//
// TS 44.018 V19.0.0 §10.5.2.8b pp14819–14843. Typed Channel Request Description 2 value; the type 4 value excludes IEI and length and has a generic 255-octet ceiling. Its actual carrier may be smaller.
package ies
