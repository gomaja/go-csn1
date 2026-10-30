// Package ies contains the RLC/MAC information elements of TS 44.060 V19.0.0.
//
// TS 44.060 V19.0.0 §12.5.1 table 12.5.1.2 paragraphs 22203–22205
// identifies the BEP measurements of §12.5.3 and the timeslot measurements of
// §12.5.4 as the typed components of EGPRS Channel Quality Report. The report
// retains its six-bit C_VALUE between those components. Measurement selection
// still depends on the containing procedure's most recent measurement mode.
// Global TFI (§12.10 paragraphs 22935–22937) retains the direction selector
// and its five-bit uplink or downlink identity.
package ies
