// Package radioaccesscapability contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §12.30 pp23543–23547 selects the typed MS RA capability value part struct of TS 24.008 V20.1.0 §10.5.5.12a. All spare bits, including those within the declared access-capability and additional-technology lengths, are suppressed on fresh and canonical sends. Received values retain all spare bits for exact replay.
package radioaccesscapability
