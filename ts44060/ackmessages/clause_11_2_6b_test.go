package ackmessages

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.1.1.3 and 11.2.6b pp14540–14551.
// The root owns selector 000010. TCH uses 8+128 bits; DCCH uses 4+8.
// The reviewed reading repairs only the two single-dash explanatory annotations.
func TestClause1126bDBPSCHAcknowledgement(t *testing.T) {
	flo := false
	for _, channel := range []runtime.RLCMACChannel{runtime.RLCMACFACCH, runtime.RLCMACSACCH, runtime.RLCMACSDCCH} {
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACDBPSCH, Channel: channel, FLO: &flo}
		if err := ctx.Validate(); err != nil {
			t.Fatal(err)
		}
		for rb := range 32 {
			for mode := range 2 {
				for present := range 2 {
					bits := fmt.Sprintf("000010%05b%01b%01b", rb, mode, present)
					if present == 1 {
						if mode == 0 {
							bits += fmt.Sprintf("%08b", 173) + strings.Repeat("10100101", 16)
						} else {
							bits += fmt.Sprintf("%04b%08b", 13, 0xa5)
						}
					}
					minimum := len(bits)
					bits = acknowledgementPadding(bits, 184)
					checkAcknowledgement(t, bits, 0, minimum, 10, DecodePacketDBPSCHDownlinkAckNackMessageFrom, EncodePacketDBPSCHDownlinkAckNackMessageTo, func(v PacketDBPSCHDownlinkAckNackMessage) {
						if v.MESSAGETYPE != 2 || v.RBId != uint8(rb) || v.STARTINGSEQUENCENUMBERChoice.Alternative != PacketDBPSCHDownlinkAckNackMessageSTARTINGSEQUENCENUMBERChoiceAlternative(mode) {
							t.Fatal("selector, bearer or mode differs")
						}
						if mode == 0 {
							branch := v.STARTINGSEQUENCENUMBERChoice.RECEIVEDBLOCKBITMAP
							if branch == nil || (branch.STARTINGSEQUENCENUMBERGroup != nil) != (present == 1) {
								t.Fatal("TCH acknowledgement arm differs")
							}
							if present == 1 {
								g := branch.STARTINGSEQUENCENUMBERGroup
								if g.STARTINGSEQUENCENUMBER != 173 || g.RECEIVEDBLOCKBITMAP.BitLength != 128 || !bytes.Equal(g.RECEIVEDBLOCKBITMAP.Bytes, bytes.Repeat([]byte{0xa5}, 16)) {
									t.Fatal("128-bit TCH bitmap differs")
								}
							}
						} else {
							branch := v.STARTINGSEQUENCENUMBERChoice.STARTINGSEQUENCENUMBER
							if branch == nil || (branch.STARTINGSEQUENCENUMBERGroup != nil) != (present == 1) {
								t.Fatal("DCCH acknowledgement arm differs")
							}
							if present == 1 {
								g := branch.STARTINGSEQUENCENUMBERGroup
								if g.STARTINGSEQUENCENUMBER != 13 || g.RECEIVEDBLOCKBITMAP != 0xa5 {
									t.Fatal("8-bit DCCH bitmap differs")
								}
							}
						}
					}, func(v *PacketDBPSCHDownlinkAckNackMessage) { v.RBId ^= 1 }, true)
				}
			}
		}
	}
	for tag := range 64 {
		if tag == 2 {
			continue
		}
		bits := acknowledgementPadding(fmt.Sprintf("%06b0000000", tag), 184)
		if _, err := DecodePacketDBPSCHDownlinkAckNackMessage(acknowledgementBits(bits)); err == nil {
			t.Fatalf("accepted message type %d", tag)
		}
	}
}

func FuzzClause1126bDBPSCHReplay(f *testing.F) {
	f.Add(acknowledgementBits(acknowledgementPadding("0000101010100", 184)))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		v, err := DecodePacketDBPSCHDownlinkAckNackMessage(raw)
		if err != nil {
			return
		}
		out, err := EncodePacketDBPSCHDownlinkAckNackMessage(v.Value)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("replay differs: %v", err)
		}
	})
}

func TestClause1126bNamedCanonicalAndInvalidFields(t *testing.T) {
	raw := acknowledgementBits(acknowledgementPadding("0000101010100", 184))
	decoded, err := DecodePacketDBPSCHDownlinkAckNackMessage(raw)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := EncodePacketDBPSCHDownlinkAckNackMessageCanonicalAtLength(decoded.Value, 23)
	if err != nil || !bytes.Equal(canonical, raw) {
		t.Fatalf("named canonical: %v", err)
	}
	for _, kind := range []string{"tag", "rb", "choice", "missing"} {
		v := runtime.Canonical(decoded.Value)
		switch kind {
		case "tag":
			v.MESSAGETYPE = 3
		case "rb":
			v.RBId = 32
		case "choice":
			v.STARTINGSEQUENCENUMBERChoice.Alternative = 2
		case "missing":
			v.STARTINGSEQUENCENUMBERChoice.RECEIVEDBLOCKBITMAP = nil
		}
		if _, err := EncodePacketDBPSCHDownlinkAckNackMessageCanonicalAtLength(v, 23); err == nil {
			t.Fatalf("accepted invalid %s", kind)
		}
	}
}
