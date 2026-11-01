// Package singleuplink contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §12.48a.3. Standalone callers provide the exact containing bit extent; an unbounded reader refuses before consuming a selector. A reserved PDCH-pair designation reports the received containing remainder and preserves the accepted typed prefix. Received values replay exactly; fresh and canonical senders refuse that designation (§§11.1.2–11.1.3.2). Application procedures and prior-assignment state remain with callers. The nested §12.5.5 region ends the enclosing assignment before Dynamic Allocation 2. The §12.48a.4 component uses its printed explicit counts; no inferred N/M projection is selected.
package singleuplink
