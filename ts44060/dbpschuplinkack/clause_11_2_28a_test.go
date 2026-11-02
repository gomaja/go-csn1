package dbpschuplinkack_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	ack "github.com/gomaja/go-csn1/ts44060/dbpschuplinkack"
)

// TS 44.060 V19.0.0 §11.2.28a paragraphs 18024-18038. These are
// complete DBPSCH message descriptions (§11.1.1.3), with an internal selector.
// The synthetic caller supplies the extent; no physical DBPSCH carrier is inferred.
func dbpschMessage(rb, identity, mode, bitmap int) (string, string) {
	s := fmt.Sprintf("001001%05b", rb)
	switch identity {
	case 0:
		s += "0"
	case 1:
		s += "1" + fmt.Sprintf("%032b", uint32(0x10203040)) + "0"
	case 2:
		s += "1" + fmt.Sprintf("%032b", uint32(0x10203040)) + "1" + "1010"
	}
	s += fmt.Sprintf("%01b%01b", mode, bitmap)
	if bitmap != 0 {
		if mode == 0 {
			s += "11001001" + strings.Repeat("10100101", 16)
		} else {
			s += "1001" + "10100101"
		}
	}
	return s, fmt.Sprintf("rb%d-identity%d-mode%d-bitmap%d", rb, identity, mode, bitmap)
}

// §11 paragraph 12939: one zero followed by positional 00101011 spare bits.
func dbpschPad(s string, origin, extent int) string {
	if len(s) < extent {
		s += "0"
	}
	for len(s) < extent {
		if (byte(0x2b)>>uint(7-(origin+len(s))%8))&1 != 0 {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}
func dbpschBytes(s string) []byte {
	b := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func dbpschFields(t testing.TB, v ack.PacketDBPSCHUplinkAckNackMessageContent, rb, identity, mode, bitmap int) {
	t.Helper()
	k := v.MESSAGETYPEGroupFallback.Known
	if k == nil || v.MESSAGETYPEGroupFallback.Ignored != nil {
		t.Fatal("valid message lost its typed body")
	}
	if k.MESSAGETYPE != 9 || int(k.RBId) != rb {
		t.Fatalf("type=%d RB=%d", k.MESSAGETYPE, k.RBId)
	}
	if identity == 0 {
		if k.CONTENTIONRESOLUTIONTLLIGroup != nil {
			t.Fatal("unexpected contention identity")
		}
	} else {
		g := k.CONTENTIONRESOLUTIONTLLIGroup
		if g == nil || g.CONTENTIONRESOLUTIONTLLI != 0x10203040 {
			t.Fatal("contention identity differs")
		}
		if identity == 1 {
			if g.GRNTIExtension != nil {
				t.Fatal("unexpected extension")
			}
		} else if g.GRNTIExtension == nil || *g.GRNTIExtension != 10 {
			t.Fatal("extension differs")
		}
	}
	c := k.STARTINGSEQUENCENUMBERChoice
	if mode == 0 {
		if c.RECEIVEDBLOCKBITMAP == nil || c.STARTINGSEQUENCENUMBER != nil {
			t.Fatal("TCH mode lost")
		}
		g := c.RECEIVEDBLOCKBITMAP.STARTINGSEQUENCENUMBERGroup
		if bitmap == 0 {
			if g != nil {
				t.Fatal("unexpected TCH bitmap")
			}
		} else if g == nil || g.STARTINGSEQUENCENUMBER != 201 || g.RECEIVEDBLOCKBITMAP.BitLength != 128 || !bytes.Equal(g.RECEIVEDBLOCKBITMAP.Bytes, bytes.Repeat([]byte{0xa5}, 16)) {
			t.Fatal("TCH bitmap or SSN differs")
		}
	} else {
		if c.STARTINGSEQUENCENUMBER == nil || c.RECEIVEDBLOCKBITMAP != nil {
			t.Fatal("DCCH mode lost")
		}
		g := c.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup
		if bitmap == 0 {
			if g != nil {
				t.Fatal("unexpected DCCH bitmap")
			}
		} else if g == nil || g.STARTINGSEQUENCENUMBER != 9 || g.RECEIVEDBLOCKBITMAP != 0xa5 {
			t.Fatal("DCCH bitmap or SSN differs")
		}
	}
}
func TestClause11228aTypedMessages(t *testing.T) {
	for _, rb := range []int{0, 17, 31} {
		for identity := 0; identity < 3; identity++ {
			for mode := 0; mode < 2; mode++ {
				for bitmap := 0; bitmap < 2; bitmap++ {
					body, name := dbpschMessage(rb, identity, mode, bitmap)
					t.Run(name, func(t *testing.T) {
						// All arms fit this synthetic 256-bit containing message description.
						data := dbpschBytes(dbpschPad(body, 0, 256))
						d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(data)
						if err != nil {
							t.Fatal(err)
						}
						dbpschFields(t, d.Value, rb, identity, mode, bitmap)
						if d.BitsConsumed != 256 {
							t.Fatalf("consumed=%d", d.BitsConsumed)
						}
						replay, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
						if err != nil || !bytes.Equal(replay, data) {
							t.Fatalf("replay differs: %v", err)
						}
						canonical, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(d.Value, 32)
						if err != nil || !bytes.Equal(canonical, data) {
							t.Fatalf("canonical differs: %v", err)
						}
						var extent *runtime.ExtentError
						if _, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonical(d.Value); !errors.As(err, &extent) || !extent.Required {
							t.Fatalf("missing containing extent: %v", err)
						}
						edited := d.Value
						known := *edited.MESSAGETYPEGroupFallback.Known
						edited.MESSAGETYPEGroupFallback.Known = &known
						known.RBId = uint8((rb + 1) % 32)
						wire, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(edited)
						if err != nil {
							t.Fatal(err)
						}
						want, _ := dbpschMessage((rb+1)%32, identity, mode, bitmap)
						if !bytes.Equal(wire, dbpschBytes(dbpschPad(want, 0, 256))) {
							t.Fatal("same-width edit did not reach wire")
						}
						fresh, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(wire)
						if err != nil {
							t.Fatal(err)
						}
						dbpschFields(t, fresh.Value, (rb+1)%32, identity, mode, bitmap)
					})
				}
			}
		}
	}
}
func TestClause11228aBoundAndPhase(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		for _, n := range []int{192, 256, 512, 1024} {
			body, _ := dbpschMessage(17, 1, 0, 1)
			message := dbpschPad(body, phase, n)
			bits := strings.Repeat("1", phase) + message + "10101"
			r, err := runtime.NewChoiceReaderBits(dbpschBytes(bits), len(bits), nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			limit, err := r.PushLimit(n)
			if err != nil {
				t.Fatal(err)
			}
			v, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			dbpschFields(t, v, 17, 1, 0, 1)
			if r.Position() != phase+n {
				t.Fatalf("position=%d want %d", r.Position(), phase+n)
			}
			if err = r.PopLimit(limit); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(5)
			if err != nil || sentinel != 21 {
				t.Fatal("containing limit consumed sentinel")
			}
			w := runtime.NewWriter()
			image := dbpschBytes(bits)
			received, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: image, BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
			if e != nil {
				t.Fatal(e)
			}
			w.WithWire(received)
			if err = w.WriteUint((uint64(1)<<phase)-1, phase); err != nil {
				t.Fatal(err)
			}
			end, err := w.PushLimit(n)
			if err != nil {
				t.Fatal(err)
			}
			if err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentTo(w, v); err != nil {
				t.Fatal(err)
			}
			if err = w.PopLimit(end); err != nil {
				t.Fatal(err)
			}
			if err = w.WriteUint(21, 5); err != nil {
				t.Fatal(err)
			}
			out, err := w.FinishBits(runtime.BitString{})
			if err != nil || out.BitLength != len(bits) || !bytes.Equal(out.Bytes, dbpschBytes(bits)) {
				t.Fatalf("phase %d extent %d replay: %v", phase, n, err)
			}
		}
	}
}

type dbpschResolver struct{}

// TS 44.060 V19.0.0 §11.2.28a paragraph 18019 restricts the carrier.
func (dbpschResolver) Resolve(c runtime.RLCMACContext, selector uint64) (controlblock.Binding, error) {
	switch c.Channel {
	case runtime.RLCMACFACCH, runtime.RLCMACSACCH, runtime.RLCMACSDCCH:
	default:
		return controlblock.Binding{}, fmt.Errorf("§11.2.28a requires FACCH, SACCH or SDCCH")
	}
	if c.Mode != runtime.RLCMACDBPSCH || c.Direction != runtime.RLCMACDownlink || selector != 9 || c.FLO == nil || *c.FLO {
		return controlblock.Binding{}, fmt.Errorf("§11.2.28a requires the non-FLO downlink DBPSCH context")
	}
	return controlblock.Binding{Kind: controlblock.KnownMessage, Ownership: controlblock.InternalSelector, DecodeFrom: func(r *runtime.Reader, _ runtime.RLCMACContext) (any, error) {
		return ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
	}, EncodeTo: func(w *runtime.Writer, v any, _ runtime.RLCMACContext) error {
		return ack.EncodePacketDBPSCHUplinkAckNackMessageContentTo(w, v.(ack.PacketDBPSCHUplinkAckNackMessageContent))
	}}, nil
}
func TestClause11228aDBPSCHContext(t *testing.T) {
	flo := false
	body, _ := dbpschMessage(17, 2, 1, 1)
	wire := dbpschBytes(dbpschPad(body, 0, 184))
	for _, channel := range []runtime.RLCMACChannel{runtime.RLCMACFACCH, runtime.RLCMACSACCH, runtime.RLCMACSDCCH} {
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACDBPSCH, Channel: channel, FLO: &flo}
		r, err := runtime.NewChoiceReaderBits(wire, 184, nil)
		if err != nil {
			t.Fatal(err)
		}
		m, err := controlblock.DecodeMessageFrom(r, ctx, dbpschResolver{})
		if err != nil {
			t.Fatal(err)
		}
		if m.Selector != 9 || r.Position() != 184 {
			t.Fatal("internal selector was consumed twice")
		}
		v, ok := m.Body.(ack.PacketDBPSCHUplinkAckNackMessageContent)
		if !ok {
			t.Fatal("untyped body")
		}
		dbpschFields(t, v, 17, 2, 1, 1)
		w := runtime.NewWriter()
		received, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: wire, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if e != nil {
			t.Fatal(e)
		}
		w.WithWire(received)
		_, err = w.PushLimit(184)
		if err != nil {
			t.Fatal(err)
		}
		if err = controlblock.EncodeMessageTo(w, ctx, m, dbpschResolver{}); err != nil {
			t.Fatal(err)
		}
		b, err := w.Bytes()
		if err != nil || !bytes.Equal(b, wire) {
			t.Fatalf("DBPSCH replay: %v", err)
		}
	}
}
func TestClause11228aContextRefusals(t *testing.T) {
	flo := false
	yes := true
	body, _ := dbpschMessage(17, 2, 1, 1)
	wire := dbpschBytes(dbpschPad(body, 0, 184))
	base := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACDBPSCH, Channel: runtime.RLCMACFACCH, FLO: &flo}
	for _, which := range []string{"pacch", "adch", "uplink", "flo", "unknown-flo", "ordinary"} {
		ctx := base
		switch which {
		case "pacch":
			ctx.Channel = runtime.RLCMACPACCH
		case "adch":
			ctx.Channel = runtime.RLCMACADCH
		case "uplink":
			ctx.Direction = runtime.RLCMACUplink
		case "flo":
			ctx.FLO = &yes
		case "unknown-flo":
			ctx.FLO = nil
		case "ordinary":
			ctx.Mode = runtime.RLCMACOrdinary
		}
		r, err := runtime.NewChoiceReaderBits(wire, 184, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = controlblock.DecodeMessageFrom(r, ctx, dbpschResolver{}); err == nil {
			t.Fatalf("%s context selected this message", which)
		}
		if r.Position() != 0 {
			t.Fatalf("%s refusal consumed the selector", which)
		}
	}
}

func TestClause11228aMalformedAndTruncated(t *testing.T) {
	body, _ := dbpschMessage(17, 0, 0, 0)
	for cut := 0; cut < 14; cut++ {
		collector := runtime.NewReceiverCollector()
		r, err := runtime.NewChoiceReaderBits(dbpschBytes(body), cut, collector)
		if err != nil {
			t.Fatal(err)
		}
		v, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
		if err != nil || v.MESSAGETYPEGroupFallback.Known != nil || v.MESSAGETYPEGroupFallback.Ignored == nil {
			t.Fatalf("cut %d did not select receiver error: %v", cut, err)
		}
		report := collector.TakeReceiverReport()
		if len(report.Regions) != 1 || report.Regions[0].Label != runtime.ErrorLabel("DBPSCH message part error") || report.Regions[0].Location.BitOffset != 0 || report.Regions[0].Location.Width != cut {
			t.Fatalf("cut %d receiver report differs", cut)
		}
	}
	bad := dbpschBytes(dbpschPad("111111"+body[6:], 0, 184))
	r, err := runtime.NewChoiceReaderBits(bad, 184, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
	if err != nil || v.MESSAGETYPEGroupFallback.Ignored == nil || v.MESSAGETYPEGroupFallback.Ignored.Label != runtime.ErrorLabel("DBPSCH message part error") {
		t.Fatalf("wrong message type lost receiver label: %v", err)
	}
	if !v.ReceiverRegionEnds() {
		t.Fatal("receiver region must end its containing description")
	}
	if _, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(v, 23); err == nil {
		t.Fatal("receiver-only error arm was sent canonically")
	}
	valid, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(dbpschBytes(dbpschPad(body, 0, 184)))
	if err != nil {
		t.Fatal(err)
	}
	valid.Value.MESSAGETYPEGroupFallback.Known.MESSAGETYPE = 8
	if _, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContent(valid.Value); err == nil {
		t.Fatal("invalid selector accepted for sending")
	}
}
func FuzzClause11228aMessage(f *testing.F) {
	for mode := 0; mode < 2; mode++ {
		for bitmap := 0; bitmap < 2; bitmap++ {
			s, _ := dbpschMessage(17, 1, mode, bitmap)
			f.Add(dbpschBytes(dbpschPad(s, 0, 256)))
		}
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1024 {
			t.Skip()
		}
		d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(b)
		if err != nil {
			return
		}
		wire, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
		if err != nil || !bytes.Equal(wire, b) {
			t.Fatalf("successful decode did not replay: %v", err)
		}
	})
}

func TestClause11228aFieldLimits(t *testing.T) {
	body, _ := dbpschMessage(17, 2, 1, 1)
	for _, tlli := range []uint32{0, ^uint32(0)} {
		for _, extension := range []uint8{0, 15} {
			for _, ssn := range []uint8{0, 15} {
				for _, bitmap := range []uint8{0, 255} {
					d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(dbpschBytes(dbpschPad(body, 0, 184)))
					if err != nil {
						t.Fatal(err)
					}
					k := d.Value.MESSAGETYPEGroupFallback.Known
					k.CONTENTIONRESOLUTIONTLLIGroup.CONTENTIONRESOLUTIONTLLI = tlli
					k.CONTENTIONRESOLUTIONTLLIGroup.GRNTIExtension = &extension
					g := k.STARTINGSEQUENCENUMBERChoice.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup
					g.STARTINGSEQUENCENUMBER = ssn
					g.RECEIVEDBLOCKBITMAP = bitmap
					b, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
					if err != nil {
						t.Fatal(err)
					}
					want := fmt.Sprintf("001001%05b1%032b1%04b11%04b%08b", 17, tlli, extension, ssn, bitmap)
					if !bytes.Equal(b, dbpschBytes(dbpschPad(want, 0, 184))) {
						t.Fatal("edited boundary field differs from clause bit calculation")
					}
				}
			}
		}
	}
	for _, field := range []string{"rb", "extension", "ssn"} {
		d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(dbpschBytes(dbpschPad(body, 0, 184)))
		if err != nil {
			t.Fatal(err)
		}
		k := d.Value.MESSAGETYPEGroupFallback.Known
		switch field {
		case "rb":
			k.RBId = 32
		case "extension":
			v := uint8(16)
			k.CONTENTIONRESOLUTIONTLLIGroup.GRNTIExtension = &v
		case "ssn":
			k.STARTINGSEQUENCENUMBERChoice.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup.STARTINGSEQUENCENUMBER = 16
		}
		if _, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value); err == nil {
			t.Fatalf("out-of-width %s sent", field)
		}
	}
}
