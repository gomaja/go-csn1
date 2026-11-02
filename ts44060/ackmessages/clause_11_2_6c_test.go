package ackmessages

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.1.1.3 and 11.2.6c pp14577–14588.
// The same owned selector 000010 requires the FLO/ADCH context. UDCH has
// absent, compressed and uncompressed FLO reports; CDCH has absent or 4+8 bits.
func TestClause1126cDBPSCHType2Acknowledgement(t *testing.T) {
	flo := true
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACDBPSCH, Channel: runtime.RLCMACADCH, FLO: &flo}
	if err := ctx.Validate(); err != nil {
		t.Fatal(err)
	}
	for rb := range 32 {
		for _, arm := range []string{"udch-all", "udch-compressed", "udch-uncompressed", "cdch-all", "cdch-bitmap"} {
			bits := fmt.Sprintf("000010%05b00%06b00", rb, 37)
			minimum := 0
			sendable := true
			switch arm {
			case "udch-all":
				bits += "00"
			case "udch-compressed":
				bits += "01" + "10" + fmt.Sprintf("%010b", 341) + "0" + fmt.Sprintf("%07b", 9) + "1" + "101001011"
			case "udch-uncompressed":
				bits += "01" + "10" + fmt.Sprintf("%010b", 341) + "1"
				minimum = len(bits)
				bits += strings.Repeat("10", (184-len(bits)+1)/2)
				bits = bits[:184]
			case "cdch-all":
				bits += "10"
			case "cdch-bitmap":
				bits += "11" + fmt.Sprintf("%04b%08b", 13, 0xa5)
			}
			if minimum == 0 {
				minimum = len(bits)
			}
			bits = acknowledgementPadding(bits, 184)
			checkAcknowledgement(t, bits, 0, minimum, 10, DecodePacketDBPSCHDownlinkAckNackType2MessageFrom, EncodePacketDBPSCHDownlinkAckNackType2MessageTo, func(v PacketDBPSCHDownlinkAckNackType2Message) {
				if v.MESSAGETYPE != 2 || v.RBId != uint8(rb) || v.EGPRSChannelQualityReport.CVALUE != 37 {
					t.Fatal("selector, RB or quality differs")
				}
				choice := v.FLOAckNackDescriptionChoice
				if strings.HasPrefix(arm, "udch") {
					if choice.Alternative != 0 || choice.FLOAckNackDescription == nil {
						t.Fatal("UDCH arm differs")
					}
					d := choice.FLOAckNackDescription.FLOAckNackDescription
					if (d == nil) != (arm == "udch-all") {
						t.Fatal("FLO report presence differs")
					}
					if d != nil {
						s := d.FLOAckNackDescriptionStruct
						if s.BEGINNINGOFWINDOW != 1 || s.ENDOFWINDOW != 0 || s.STARTINGSEQUENCENUMBER != 341 {
							t.Fatal("FLO fields differ")
						}
						if arm == "udch-compressed" {
							c := s.COMPRESSEDBITMAPLENGTHChoice.COMPRESSEDBITMAPLENGTH
							if c == nil || c.COMPRESSEDBITMAPLENGTH != 9 || c.COMPRESSEDBITMAPSTARTINGCOLORCODE != 1 || c.COMPRESSEDRECEIVEDBLOCKBITMAP.BitLength != 9 {
								t.Fatal("compressed FLO bitmap differs")
							}
						} else {
							c := s.COMPRESSEDBITMAPLENGTHChoice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP
							if c == nil || len(c.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != 184-minimum {
								t.Fatal("FLO remainder extent differs")
							}
						}
					}
				} else {
					if choice.Alternative != 1 || choice.STARTINGSEQUENCENUMBER == nil {
						t.Fatal("CDCH arm differs")
					}
					g := choice.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup
					if (g == nil) != (arm == "cdch-all") {
						t.Fatal("CDCH bitmap presence differs")
					}
					if g != nil && (g.STARTINGSEQUENCENUMBER != 13 || g.RECEIVEDBLOCKBITMAP != 0xa5) {
						t.Fatal("CDCH bitmap differs")
					}
				}
			}, func(v *PacketDBPSCHDownlinkAckNackType2Message) { v.RBId ^= 1 }, sendable)
		}
	}
}

func FuzzClause1126cDBPSCHReplay(f *testing.F) {
	f.Add(acknowledgementBits(acknowledgementPadding("00001010101001001010000", 184)))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		v, err := DecodePacketDBPSCHDownlinkAckNackType2Message(raw)
		if err != nil {
			return
		}
		out, err := EncodePacketDBPSCHDownlinkAckNackType2Message(v.Value)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("replay differs: %v", err)
		}
	})
}

func TestClause1126cNamedCanonicalAndInvalidFields(t *testing.T) {
	raw := acknowledgementBits(acknowledgementPadding("00001010101001001010000", 184))
	decoded, err := DecodePacketDBPSCHDownlinkAckNackType2Message(raw)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := EncodePacketDBPSCHDownlinkAckNackType2MessageCanonicalAtLength(decoded.Value, 23)
	if err != nil || !bytes.Equal(canonical, raw) {
		t.Fatalf("named canonical: %v", err)
	}
	for tag := range 64 {
		if tag == 2 {
			continue
		}
		bits := acknowledgementPadding(fmt.Sprintf("%06b", tag)+"10101001001010000", 184)
		if _, err := DecodePacketDBPSCHDownlinkAckNackType2Message(acknowledgementBits(bits)); err == nil {
			t.Fatalf("accepted message type %d", tag)
		}
	}
	for _, kind := range []string{"tag", "rb", "quality", "choice", "missing"} {
		v := runtime.Canonical(decoded.Value)
		switch kind {
		case "tag":
			v.MESSAGETYPE = 3
		case "rb":
			v.RBId = 32
		case "quality":
			v.EGPRSChannelQualityReport.CVALUE = 64
		case "choice":
			v.FLOAckNackDescriptionChoice.Alternative = 2
		case "missing":
			v.FLOAckNackDescriptionChoice.FLOAckNackDescription = nil
		}
		if _, err := EncodePacketDBPSCHDownlinkAckNackType2MessageCanonicalAtLength(v, 23); err == nil {
			t.Fatalf("accepted invalid %s", kind)
		}
	}
}

// This is component growth under an explicit bound, not physical ADCH capacity.
// TS 44.060 V19.0.0 §12.3.2 p22142 makes the uncompressed bitmap bit**.
func TestClause1126cUncompressedBoundGrowth(t *testing.T) {
	for _, n := range []int{0, 1, 7, 8, 127, 1024, 4096} {
		prefix := fmt.Sprintf("000010%05b00%06b00", 21, 37) + "01" + "10" + fmt.Sprintf("%010b", 341) + "1"
		bits := prefix + strings.Repeat("1", n)
		r := acknowledgementReader(t, acknowledgementBits(bits+"10101"), len(bits)+5, 0)
		old, err := r.PushLimit(len(bits))
		if err != nil {
			t.Fatal(err)
		}
		v, err := DecodePacketDBPSCHDownlinkAckNackType2MessageFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		bitmap := v.FLOAckNackDescriptionChoice.FLOAckNackDescription.FLOAckNackDescription.FLOAckNackDescriptionStruct.COMPRESSEDBITMAPLENGTHChoice.UNCOMPRESSEDRECEIVEDBLOCKBITMAP.UNCOMPRESSEDRECEIVEDBLOCKBITMAP
		if len(bitmap) != n || cap(bitmap) > 2*n+16 || r.Position() != len(bits) {
			t.Fatalf("bitmap length %d capacity %d position %d", len(bitmap), cap(bitmap), r.Position())
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(5)
		if err != nil || sentinel != 21 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
	}
}
