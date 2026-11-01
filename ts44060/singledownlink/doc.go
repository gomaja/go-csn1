// Package singledownlink contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §12.48a.2. Standalone callers provide the exact containing bit extent; an unbounded reader refuses before consuming a selector. A reserved PDCH-pair designation reports the received containing remainder and preserves the accepted typed prefix. Received values replay exactly; fresh and canonical senders refuse that designation (§§11.1.2–11.1.3.2). Application procedures and prior-assignment state remain with callers.
package singledownlink
