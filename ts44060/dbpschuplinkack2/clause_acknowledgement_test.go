package dbpschuplinkack2_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	ack "github.com/gomaja/go-csn1/ts44060/dbpschuplinkack2"
)

type acknowledgementVector struct {
	kind                          string
	rb, ssn, bitmap               int
	beginning, end, length, color int
}

func acknowledgementCases(rb int) []acknowledgementVector {
	cases := []acknowledgementVector{
		{kind: "udch-ack", rb: rb},
		{kind: "cdch-ack", rb: rb},
		{kind: "cdch-bitmap", rb: rb, ssn: 9, bitmap: 0xa5},
		{kind: "udch-uncompressed", rb: rb, beginning: 1, ssn: 513},
	}
	for i, n := range []int{0, 1, 7, 127} {
		cases = append(cases, acknowledgementVector{kind: "udch-compressed", rb: rb, beginning: i % 2, end: (i / 2) % 2, ssn: []int{0, 1, 513, 1023}[i], color: i % 2, length: n})
	}
	return cases
}

// TS 44.060 V19.0.0 §11.2.28b paragraphs 18071-18082 and §12.3.2:
// selector 001001, RB Id, mode/presence selectors, then the selected fields.
// The FLO uncompressed bitmap occupies the containing remainder; padding may
// be the null string (§11.1.1.3). No physical ADCH block size is inferred.
func acknowledgementPrefix(v acknowledgementVector) string {
	s := fmt.Sprintf("001001%05b", v.rb)
	switch v.kind {
	case "udch-ack":
		return s + "00"
	case "cdch-ack":
		return s + "10"
	case "cdch-bitmap":
		return s + fmt.Sprintf("11%04b%08b", v.ssn, v.bitmap)
	case "udch-compressed":
		return s + fmt.Sprintf("01%01b%01b%010b0%07b%01b", v.beginning, v.end, v.ssn, v.length, v.color) + strings.Repeat("10100101", 16)[:v.length]
	case "udch-uncompressed":
		return s + fmt.Sprintf("01%01b%01b%010b1", v.beginning, v.end, v.ssn)
	default:
		panic("unknown acknowledgement test arm")
	}
}

func acknowledgementBits(v acknowledgementVector, origin, extent int) string {
	s := acknowledgementPrefix(v)
	if len(s) > extent {
		panic("test acknowledgement exceeds its containing extent")
	}
	if v.kind == "udch-uncompressed" {
		return s + strings.Repeat("10100101", (extent-len(s)+7)/8)[:extent-len(s)]
	}
	// §11 paragraph 12939: one zero then positional 00101011 spare bits.
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

func acknowledgementBytes(s string) []byte {
	b := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}

func acknowledgementWire(t testing.TB, r *runtime.Reader, bits string) runtime.WireInfo {
	t.Helper()
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: acknowledgementBytes(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

func acknowledgementFields(t testing.TB, value ack.PacketDBPSCHUplinkAckNackMessageContent, want acknowledgementVector, extent int) {
	t.Helper()
	k := value.MESSAGETYPEGroupFallback.Known
	if k == nil || value.MESSAGETYPEGroupFallback.Ignored != nil || k.MESSAGETYPE != 9 || int(k.RBId) != want.rb {
		t.Fatal("message lost its typed selector or RB identity")
	}
	c := k.FLOAckNackDescriptionChoice
	if strings.HasPrefix(want.kind, "cdch-") {
		if c.STARTINGSEQUENCENUMBER == nil || c.FLOAckNackDescription != nil {
			t.Fatal("CDCH mode lost its typed arm")
		}
		g := c.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup
		if want.kind == "cdch-ack" {
			if g != nil {
				t.Fatal("all-acknowledged CDCH contains a bitmap")
			}
		} else if g == nil || int(g.STARTINGSEQUENCENUMBER) != want.ssn || int(g.RECEIVEDBLOCKBITMAP) != want.bitmap {
			t.Fatal("CDCH sequence number or bitmap differs")
		}
		return
	}
	if c.FLOAckNackDescription == nil || c.STARTINGSEQUENCENUMBER != nil {
		t.Fatal("UDCH mode lost its typed arm")
	}
	if want.kind == "udch-ack" {
		if c.FLOAckNackDescription.FLOAckNackDescription != nil {
			t.Fatal("all-acknowledged UDCH contains a FLO description")
		}
		return
	}
	ie := c.FLOAckNackDescription.FLOAckNackDescription
	if ie == nil {
		t.Fatal("FLO description became opaque or absent")
	}
	flo := ie.FLOAckNackDescriptionStruct
	if int(flo.BEGINNINGOFWINDOW) != want.beginning || int(flo.ENDOFWINDOW) != want.end || int(flo.STARTINGSEQUENCENUMBER) != want.ssn {
		t.Fatal("FLO window flags or sequence number differs")
	}
	bitmap := flo.COMPRESSEDBITMAPLENGTHChoice
	if want.kind == "udch-compressed" {
		g := bitmap.COMPRESSEDBITMAPLENGTH
		bits := strings.Repeat("10100101", 16)[:want.length]
		if g == nil || bitmap.UNCOMPRESSEDRECEIVEDBLOCKBITMAP != nil || int(g.COMPRESSEDBITMAPLENGTH) != want.length || int(g.COMPRESSEDBITMAPSTARTINGCOLORCODE) != want.color || g.COMPRESSEDRECEIVEDBLOCKBITMAP.BitLength != want.length || !bytes.Equal(g.COMPRESSEDRECEIVEDBLOCKBITMAP.Bytes, acknowledgementBytes(bits)) {
			t.Fatal("compressed FLO bitmap length, color or typed bits differs")
		}
		return
	}
	g := bitmap.UNCOMPRESSEDRECEIVEDBLOCKBITMAP
	if g == nil || bitmap.COMPRESSEDBITMAPLENGTH != nil || len(g.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != extent-26 {
		t.Fatal("uncompressed FLO bitmap did not fill its exact containing remainder")
	}
	for i, bit := range g.UNCOMPRESSEDRECEIVEDBLOCKBITMAP {
		if bit != "10100101"[i%8]-'0' {
			t.Fatalf("uncompressed bitmap bit %d differs", i)
		}
	}
}

func TestAcknowledgementTypedArms(t *testing.T) {
	for _, rb := range []int{0, 17, 31} {
		for _, vector := range acknowledgementCases(rb) {
			t.Run(fmt.Sprintf("rb%d-%s-length%d", rb, vector.kind, vector.length), func(t *testing.T) {
				wire := acknowledgementBytes(acknowledgementBits(vector, 0, 184))
				d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(wire)
				if err != nil {
					t.Fatal(err)
				}
				acknowledgementFields(t, d.Value, vector, 184)
				if d.BitsConsumed != 184 {
					t.Fatalf("consumed %d bits", d.BitsConsumed)
				}
				replay, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay differs: %v", err)
				}
				canonical, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(d.Value, 23)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical differs: %v", err)
				}
				var extent *runtime.ExtentError
				if _, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonical(d.Value); !errors.As(err, &extent) || !extent.Required {
					t.Fatalf("missing required containing extent: %v", err)
				}
				known := *d.Value.MESSAGETYPEGroupFallback.Known
				d.Value.MESSAGETYPEGroupFallback.Known = &known
				vector.rb = (rb + 1) % 32
				known.RBId = uint8(vector.rb)
				want := acknowledgementBytes(acknowledgementBits(vector, 0, 184))
				for _, encode := range []func() ([]byte, error){
					func() ([]byte, error) { return ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value) },
					func() ([]byte, error) {
						return ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(d.Value, 23)
					},
				} {
					got, err := encode()
					if err != nil || !bytes.Equal(got, want) {
						t.Fatalf("RB edit did not reach wire: %v", err)
					}
					fresh, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(got)
					if err != nil {
						t.Fatal(err)
					}
					acknowledgementFields(t, fresh.Value, vector, 184)
				}
			})
		}
	}
}

func TestAcknowledgementBoundAndPhase(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		for _, n := range []int{184, 256, 512, 1024} {
			for _, vector := range acknowledgementCases(17) {
				bits := strings.Repeat("1", phase) + acknowledgementBits(vector, phase, n) + "10101"
				r, err := runtime.NewChoiceReaderBits(acknowledgementBytes(bits), len(bits), nil)
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
				value, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				acknowledgementFields(t, value, vector, n)
				if r.Position() != phase+n {
					t.Fatalf("phase %d extent %d position %d", phase, n, r.Position())
				}
				if err = r.PopLimit(limit); err != nil {
					t.Fatal(err)
				}
				if sentinel, err := r.ReadUint(5); err != nil || sentinel != 21 {
					t.Fatal("containing extent consumed the sentinel")
				}
				w := runtime.NewWriter()
				w.WithWire(acknowledgementWire(t, r, bits))
				if err = w.WriteUint((uint64(1)<<phase)-1, phase); err != nil {
					t.Fatal(err)
				}
				end, err := w.PushLimit(n)
				if err != nil {
					t.Fatal(err)
				}
				if err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentTo(w, value); err != nil {
					t.Fatal(err)
				}
				if err = w.PopLimit(end); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(21, 5); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != len(bits) || !bytes.Equal(out.Bytes, acknowledgementBytes(bits)) {
					t.Fatalf("phase %d extent %d %s replay: %v", phase, n, vector.kind, err)
				}
			}
		}
	}
}

type acknowledgementResolver struct{}

// TS 44.060 V19.0.0 §11.2.28b paragraphs 18065-18069 selects downlink
// ADCH with FLO; §11.1.1.3 keeps the six-bit selector inside the description.
func (acknowledgementResolver) Resolve(c runtime.RLCMACContext, selector uint64) (controlblock.Binding, error) {
	if c.Mode != runtime.RLCMACDBPSCH || c.Direction != runtime.RLCMACDownlink || c.Channel != runtime.RLCMACADCH || c.FLO == nil || !*c.FLO || selector != 9 {
		return controlblock.Binding{}, fmt.Errorf("§11.2.28b requires ADCH in the FLO downlink DBPSCH context")
	}
	return controlblock.Binding{Kind: controlblock.KnownMessage, Ownership: controlblock.InternalSelector, DecodeFrom: func(r *runtime.Reader, _ runtime.RLCMACContext) (any, error) {
		return ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
	}, EncodeTo: func(w *runtime.Writer, v any, _ runtime.RLCMACContext) error {
		return ack.EncodePacketDBPSCHUplinkAckNackMessageContentTo(w, v.(ack.PacketDBPSCHUplinkAckNackMessageContent))
	}}, nil
}

func TestAcknowledgementDBPSCHContext(t *testing.T) {
	flo := true
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACDBPSCH, Channel: runtime.RLCMACADCH, FLO: &flo}
	for _, vector := range acknowledgementCases(17) {
		wire := acknowledgementBytes(acknowledgementBits(vector, 0, 184))
		r, err := runtime.NewChoiceReaderBits(wire, 184, nil)
		if err != nil {
			t.Fatal(err)
		}
		message, err := controlblock.DecodeMessageFrom(r, ctx, acknowledgementResolver{})
		if err != nil {
			t.Fatal(err)
		}
		if message.Selector != 9 || r.Position() != 184 {
			t.Fatal("internal DBPSCH selector was consumed twice")
		}
		value, ok := message.Body.(ack.PacketDBPSCHUplinkAckNackMessageContent)
		if !ok {
			t.Fatal("DBPSCH body lost its concrete type")
		}
		acknowledgementFields(t, value, vector, 184)
		w := runtime.NewWriter()
		w.WithWire(acknowledgementWire(t, r, acknowledgementBits(vector, 0, 184)))
		if _, err = w.PushLimit(184); err != nil {
			t.Fatal(err)
		}
		if err = controlblock.EncodeMessageTo(w, ctx, message, acknowledgementResolver{}); err != nil {
			t.Fatal(err)
		}
		out, err := w.Bytes()
		if err != nil || !bytes.Equal(out, wire) {
			t.Fatalf("complete DBPSCH replay: %v", err)
		}
	}
}

func TestAcknowledgementContextRefusals(t *testing.T) {
	yes, no := true, false
	base := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACDBPSCH, Channel: runtime.RLCMACADCH, FLO: &yes}
	vector := acknowledgementVector{kind: "cdch-bitmap", rb: 17, ssn: 9, bitmap: 0xa5}
	wire := acknowledgementBytes(acknowledgementBits(vector, 0, 184))
	for _, which := range []string{"pacch", "facch", "sacch", "sdcch", "uplink", "non-flo", "unknown-flo", "ordinary", "wrong-selector"} {
		ctx := base
		data := wire
		switch which {
		case "pacch":
			ctx.Channel = runtime.RLCMACPACCH
		case "facch":
			ctx.Channel = runtime.RLCMACFACCH
		case "sacch":
			ctx.Channel = runtime.RLCMACSACCH
		case "sdcch":
			ctx.Channel = runtime.RLCMACSDCCH
		case "uplink":
			ctx.Direction = runtime.RLCMACUplink
		case "non-flo":
			ctx.FLO = &no
		case "unknown-flo":
			ctx.FLO = nil
		case "ordinary":
			ctx.Mode = runtime.RLCMACOrdinary
		case "wrong-selector":
			data = bytes.Clone(wire)
			data[0] ^= 1 << 2
		}
		r, err := runtime.NewChoiceReaderBits(data, 184, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = controlblock.DecodeMessageFrom(r, ctx, acknowledgementResolver{}); err == nil {
			t.Fatalf("%s context selected Type 2", which)
		}
		if r.Position() != 0 {
			t.Fatalf("%s refusal consumed the selector", which)
		}
	}
}

func TestAcknowledgementReceiverRegion(t *testing.T) {
	for _, vector := range acknowledgementCases(17) {
		bits := acknowledgementBits(vector, 0, 184)
		for cut := 0; cut < len(acknowledgementPrefix(vector)); cut++ {
			collector := runtime.NewReceiverCollector()
			r, err := runtime.NewChoiceReaderBits(acknowledgementBytes(bits), cut, collector)
			if err != nil {
				t.Fatal(err)
			}
			value, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
			g := value.MESSAGETYPEGroupFallback
			if err != nil || g.Known != nil || g.Ignored == nil || g.Ignored.Label != runtime.DBPSCHMessagePartError || !value.ReceiverRegionEnds() {
				t.Fatalf("%s cut %d lost receiver region: %v", vector.kind, cut, err)
			}
			report := collector.TakeReceiverReport()
			if len(report.Regions) != 1 || report.Regions[0].Label != runtime.DBPSCHMessagePartError || report.Regions[0].Location.BitOffset != 0 || report.Regions[0].Location.Width != cut || r.Position() != cut {
				t.Fatalf("%s cut %d receiver extent differs", vector.kind, cut)
			}
			w := runtime.NewWriter()
			w.WithWire(acknowledgementWire(t, r, bits[:cut]))
			if _, err = w.PushLimit(cut); err != nil {
				t.Fatal(err)
			}
			if err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentTo(w, value); err != nil {
				t.Fatal(err)
			}
			out, err := w.FinishBits(runtime.BitString{})
			if err != nil || out.BitLength != cut || !bytes.Equal(out.Bytes, acknowledgementBytes(bits[:cut])) {
				t.Fatalf("%s cut %d receiver replay differs: %v", vector.kind, cut, err)
			}
		}
	}
	vector := acknowledgementVector{kind: "cdch-ack", rb: 17}
	bad := "111111" + acknowledgementBits(vector, 0, 184)[6:]
	collector := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(acknowledgementBytes(bad+"10101"), 189, collector)
	if err != nil {
		t.Fatal(err)
	}
	limit, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	value, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContentFrom(r)
	if err != nil || value.MESSAGETYPEGroupFallback.Known != nil || value.MESSAGETYPEGroupFallback.Ignored == nil || value.MESSAGETYPEGroupFallback.Ignored.Label != runtime.DBPSCHMessagePartError {
		t.Fatalf("bad selector lost receiver label: %v", err)
	}
	if err = r.PopLimit(limit); err != nil {
		t.Fatal(err)
	}
	if sentinel, err := r.ReadUint(5); err != nil || sentinel != 21 {
		t.Fatal("error region crossed the containing boundary")
	}
	if _, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(value, 23); err == nil {
		t.Fatal("receiver-only arm was sent canonically")
	}
}

func TestAcknowledgementFieldEditsAndLimits(t *testing.T) {
	for _, ssn := range []int{0, 1023} {
		vector := acknowledgementCases(17)[6]
		d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(acknowledgementBytes(acknowledgementBits(vector, 0, 184)))
		if err != nil {
			t.Fatal(err)
		}
		flo := &d.Value.MESSAGETYPEGroupFallback.Known.FLOAckNackDescriptionChoice.FLOAckNackDescription.FLOAckNackDescription.FLOAckNackDescriptionStruct
		flo.STARTINGSEQUENCENUMBER = uint16(ssn)
		flo.BEGINNINGOFWINDOW, flo.ENDOFWINDOW = 1, 1
		flo.COMPRESSEDBITMAPLENGTHChoice.COMPRESSEDBITMAPLENGTH.COMPRESSEDBITMAPSTARTINGCOLORCODE = 0
		vector.ssn, vector.beginning, vector.end, vector.color = ssn, 1, 1, 0
		out, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
		if err != nil || !bytes.Equal(out, acknowledgementBytes(acknowledgementBits(vector, 0, 184))) {
			t.Fatalf("FLO endpoint edit differs: %v", err)
		}
	}
	for _, ssn := range []int{0, 15} {
		for _, bitmap := range []int{0, 255} {
			vector := acknowledgementVector{kind: "cdch-bitmap", rb: 17, ssn: 9, bitmap: 0xa5}
			d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(acknowledgementBytes(acknowledgementBits(vector, 0, 184)))
			if err != nil {
				t.Fatal(err)
			}
			g := d.Value.MESSAGETYPEGroupFallback.Known.FLOAckNackDescriptionChoice.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup
			g.STARTINGSEQUENCENUMBER, g.RECEIVEDBLOCKBITMAP = uint8(ssn), uint8(bitmap)
			vector.ssn, vector.bitmap = ssn, bitmap
			out, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
			if err != nil || !bytes.Equal(out, acknowledgementBytes(acknowledgementBits(vector, 0, 184))) {
				t.Fatalf("CDCH endpoint edit differs: %v", err)
			}
		}
	}
	for _, field := range []string{"selector", "rb", "cdch-ssn", "flo-ssn", "beginning", "end", "color", "length", "bitmap-length"} {
		vector := acknowledgementCases(17)[6]
		if field == "cdch-ssn" {
			vector = acknowledgementVector{kind: "cdch-bitmap", rb: 17, ssn: 9, bitmap: 0xa5}
		}
		d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(acknowledgementBytes(acknowledgementBits(vector, 0, 184)))
		if err != nil {
			t.Fatal(err)
		}
		k := d.Value.MESSAGETYPEGroupFallback.Known
		switch field {
		case "selector":
			k.MESSAGETYPE = 8
		case "rb":
			k.RBId = 32
		case "cdch-ssn":
			k.FLOAckNackDescriptionChoice.STARTINGSEQUENCENUMBER.STARTINGSEQUENCENUMBERGroup.STARTINGSEQUENCENUMBER = 16
		default:
			flo := &k.FLOAckNackDescriptionChoice.FLOAckNackDescription.FLOAckNackDescription.FLOAckNackDescriptionStruct
			compressed := flo.COMPRESSEDBITMAPLENGTHChoice.COMPRESSEDBITMAPLENGTH
			switch field {
			case "flo-ssn":
				flo.STARTINGSEQUENCENUMBER = 1024
			case "beginning":
				flo.BEGINNINGOFWINDOW = 2
			case "end":
				flo.ENDOFWINDOW = 2
			case "color":
				compressed.COMPRESSEDBITMAPSTARTINGCOLORCODE = 2
			case "length":
				compressed.COMPRESSEDBITMAPLENGTH = 128
			case "bitmap-length":
				compressed.COMPRESSEDRECEIVEDBLOCKBITMAP.BitLength--
			}
		}
		if _, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value); err == nil {
			t.Fatalf("invalid %s sent", field)
		}
	}
}

func FuzzAcknowledgementReplay(f *testing.F) {
	for _, vector := range acknowledgementCases(17) {
		f.Add(acknowledgementBytes(acknowledgementBits(vector, 0, 184)))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024 {
			t.Skip()
		}
		d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(data)
		if err != nil {
			return
		}
		out, err := ack.EncodePacketDBPSCHUplinkAckNackMessageContent(d.Value)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("successful decode lost replay: %v", err)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.28b requires thirteen content bits even
// for the shortest known sender arm; an eight-bit target cannot contain it.
func TestAcknowledgementCanonicalTargetWitness(t *testing.T) {
	wire := acknowledgementBytes(acknowledgementBits(acknowledgementVector{kind: "udch-ack"}, 0, 16))
	d, err := ack.DecodePacketDBPSCHUplinkAckNackMessageContent(wire)
	if err != nil || d.Value.MESSAGETYPEGroupFallback.Known == nil {
		t.Fatalf("short known arm: %v", err)
	}
	_, err = ack.EncodePacketDBPSCHUplinkAckNackMessageContentCanonicalAtLength(d.Value, 1)
	var bound *runtime.BoundError
	if !errors.As(err, &bound) || bound.Kind != runtime.CanonicalTarget {
		t.Fatalf("short canonical target: %v", err)
	}
}
