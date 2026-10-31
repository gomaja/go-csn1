// Package controlblock implements control-header bit layouts and stateless
// message framing from TS 44.060 V19.0.0 §§9.1.12a–b, 10.0b, 10.3 and 10.4.
//
// Header entry points operate on the caller's reader/writer. They do not strip
// a non-octet prefix or reset L/H coordinates. EC-PACCH/D excludes the separately
// encoded USF. EC-PACCH/U has no MAC header. DecodeControlHeader exposes logical
// optional octets; the MCS-0 physical type-3 header and §10.0b.3.2 field mapping
// belong to that carrier and are not interpreted as CS-1 here.
//
// AccessBurst is the physical container, not a decoded request/acknowledgement.
// Message dispatch uses an injected source-backed resolver. Concrete generated
// bodies and the complete ordinary/EC/DBPSCH catalogue are supplied separately.
//
// Reassemble accepts one complete ordered transaction and retains final filler.
// NewReader retains the first-field bit origin; multipart EC body decoding
// refuses until its L/H coordinate convention has a source-backed decision.
// PSI/container continuation is not control-block segmentation. Applications
// own §9.1.12b's timers, parallel-message limits, PDCH changes, unknown-TFI
// admission, retransmissions and decisions to discard incomplete messages.
package controlblock
