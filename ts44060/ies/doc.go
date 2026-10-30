// Package ies contains the published CSN.1 definitions from TS 44.060.
//
// TS 44.060 V19.0.0 EC Packet Timing Advance (§12.74 paragraphs 25986–25987) retains its six-bit timing-advance value.
//
// TS 44.060 V19.0.0 §12.32 defines the UTRAN TDD Target cell information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 §12.48a.4 defines Dynamic Allocation 2 IE, extracted from the printed source grammar. Canonical encoding takes an explicit containing length.
//
// TS 44.060 V19.0.0 §12.5a.2 table 12.5a.2.1 defines the EGPRS BEP Link Quality Measurements Type 2 IE. Seven independent selectors retain the GMSK, 8PSK, QPSK, 16QAM NSR, 32QAM NSR, 16QAM HSR and 32QAM HSR five-bit mean and three-bit variation measurements. The containing procedure supplies the applicable measurement mode and carrier context.
//
// TS 44.060 V19.0.0 §12.65 defines Dynamic Allocation 3 IE, extracted from the printed source grammar. Canonical encoding takes an explicit containing length.
//
// TS 44.060 V19.0.0 §12.68 defines the E-UTRAN Target Cell with extended EARFCN information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 §12.8.2 defines Dual Carrier Frequency Parameters IE, extracted from the printed source grammar. Its direct encoding 1 delegates to the GPRS Mobile Allocation IE in §12.10a; direct encoding 2 retains the printed variable octet-string frequency-list contents. Canonical encoding takes an explicit containing length.
//
// TS 44.060 V19.0.0 §12.69 defines the E-UTRAN IPP with extended EARFCNs information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 §12.49 defines the E-UTRAN Target Cell information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 EC Primary Ack/Nack Description (§12.73 paragraphs 25978–25980) retains its five-bit starting sequence number and distinct fixed eight-bit bitmap.
//
// TS 44.060 V19.0.0 EC Channel Quality Report Higher CC (§12.76 paragraphs 26013–26016) retains its optional five-bit GMSK mean and three-bit variation coefficient followed by the six-bit C_VALUE.
//
// TS 44.060 V19.0.0 Ack/Nack Description (§12.3 paragraphs 22072–22075) retains its final-ack indicator, seven-bit starting sequence number and fixed 64-bit received-block bitmap.
//
// TS 44.060 V19.0.0 EC Ack/Nack Description (§12.72 paragraphs 25970–25972) retains its five-bit starting sequence number and fixed 16-bit bitmap.
//
// TS 44.060 V19.0.0 §12.31 defines the UTRAN FDD Target cell information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 §12.70 defines the E-UTRAN NC with extended EARFCNs information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// TS 44.060 V19.0.0 §12.8.4 defines DLMC Frequency Parameters IE, extracted from the printed source grammar. Its direct encoding 1 delegates to the GPRS Mobile Allocation IE in §12.10a; direct encoding 2 retains the printed variable octet-string frequency-list contents. Canonical encoding takes an explicit containing length.
//
// TS 44.060 V19.0.0 FLO Ack/Nack Description (§12.3.2 paragraphs 22132–22142) retains both bitmap selectors, the seven-bit compressed length and its exact bitmap width. The uncompressed bitmap consumes the enclosing value extent; callers embedding it supply that bit limit.
//
// TS 44.060 V19.0.0 §12.50 defines the Individual Priorities information element. Its typed codec preserves the printed field widths, selectors and repeated structures.
//
// The Packet Request Reference IE codec implements TS 44.060 V19.0.0 §12.11 using its printed CSN.1 grammar.
//
// The Power Control Parameters IE codec implements TS 44.060 V19.0.0 §12.13 using its printed CSN.1 grammar.
//
// The PRACH Control Parameters IE codec implements TS 44.060 V19.0.0 §12.14 using its printed CSN.1 grammar. It preserves four retransmission values and the optional group of four persistence levels.
//
// The G-RNTI IE codec implements TS 44.060 V19.0.0 §12.16a using its printed CSN.1 grammar.
//
// The Cell Identification IE codec implements TS 44.060 V19.0.0 §12.23 using its printed CSN.1 grammar. Location Area Identification and Cell Identity retain the printed fixed octet fields.
//
// The PCCCH Organization Parameters IE codec implements TS 44.060 V19.0.0 §12.25 using its printed CSN.1 grammar.
//
// The MBMS In-band Signalling Indicator IE codec implements TS 44.060 V19.0.0 §12.45 using its printed CSN.1 grammar.
//
// The Channel Request Description IE codec implements TS 44.060 V19.0.0 §12.7 using its printed CSN.1 grammar.
//
// The Iu mode Channel Request Description IE codec implements TS 44.060 V19.0.0 §12.7a using its printed CSN.1 grammar. Its recursive entries apply only in the containing Multiple TBF request procedure.
//
// The Extended Channel Request Description IE codec implements TS 44.060 V19.0.0 §12.7b using its printed CSN.1 grammar. Its recursive entries apply only in the containing Multiple TBF request procedure.
//
// The Global Power Control Parameters IE codec implements TS 44.060 V19.0.0 §12.9 using its printed CSN.1 grammar. It enforces the printed zero following PC_MEAS_CHAN; raw parameter values retain their received coding.
//
// TS 44.060 V19.0.0 §12.29 defines COMPACT reduced MA IE. Its six-bit length retains the exact bitmap width, including zero, before the optional six-bit MAIO2 and terminal spare bits. Canonical encoding requires an explicit containing length.
//
// TS 44.060 V19.0.0 §12.53 defines E-UTRAN Parameters IE, with the printed
// measurement, neighbour-cell, not-allowed-cell and PCID-to-TA mapping helpers.
// Its PCID Group IE references are defined in §12.57.

package ies
