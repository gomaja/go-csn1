// Package pdchpairs contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §12.5.5 pp22497–22525. Single- and dual-carrier PDCH-pair descriptions require a caller-supplied containing bit extent. Reserved designation 11 selects a reported PDCH pairs configuration error region and consumes the remaining containing extent (§§11.1–11.1.3.2); fresh and canonical senders refuse it, while received data replays exactly. Application error-handling procedures belong to the caller (§11.1.2 p13020).
package pdchpairs
