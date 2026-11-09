// Package egprsmode2 contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §12.48a.1. Standalone callers supply the exact containing bit extent. The wrapper imports the four assignment IEs from their owning publication packages, without copying their definitions. Receiver error designations terminate the containing description, preserve its typed prefix and received remainder, and replay exactly. Fresh and canonical senders refuse those designations. Application procedures, multiple-TBF capability checks and prior assignment state remain with callers.
package egprsmode2
