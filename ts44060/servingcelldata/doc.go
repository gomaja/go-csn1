// Package servingcelldata contains source-selected CSN.1 codecs.
//
// TS 44.060 V19.0.0 §11.2.17a: decode all syntactically valid containers and report reserved PD 2–7 and finite CD_LENGTH 19–30. Fresh and canonical sends refuse reserved values; received values replay exactly. The printed action, "the contents of the container shall be discarded", and SI/PSI reassembly belong to applications.
package servingcelldata
