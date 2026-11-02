package celldata_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/celldata"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

func cellBits(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, c := range bits {
		if c == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

// TS 44.060 V19.0.0 §11 pp12937–12940; TS 24.007 V20.0.0
// Annex B §B.1.2.2: positional spare padding repeats 00101011.
func cellPad(prefix string, end int) string {
	if len(prefix) == end {
		return prefix
	}
	prefix += "0"
	for len(prefix) < end {
		prefix += string("00101011"[len(prefix)%8])
	}
	return prefix
}
func cellReader(t *testing.T, bits string, origin, extent int) (*runtime.Reader, int) {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(cellBits(bits), len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(extent)
	if err != nil {
		t.Fatal(err)
	}
	return r, old
}
func cellFinish(t *testing.T, w *runtime.Writer, want string) {
	t.Helper()
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != len(want) || !bytes.Equal(out.Bytes, cellBits(want)) {
		t.Fatalf("wire %x/%d want %x/%d: %v", out.Bytes, out.BitLength, cellBits(want), len(want), err)
	}
}
func cellReadStatus(t *testing.T, bits string, extent int) (celldata.PacketSIStatusMessageContent, *runtime.Reader) {
	t.Helper()
	var r *runtime.Reader
	var old int
	if extent == 170 && len(bits) >= 184 {
		// TS 44.060 V19.0.0 §§10.0b.2.1, 10.3.2 and 11.2.0.2:
		// the production CS-1 header leaves the content at absolute bit 14.
		var err error
		r, err = runtime.NewChoiceReaderBits(cellBits(bits), len(bits), nil)
		if err != nil {
			t.Fatal(err)
		}
		old, err = r.PushLimit(184)
		if err != nil {
			t.Fatal(err)
		}
		ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
		if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
			t.Fatal(err)
		}
		tag, err := r.ReadUint(6)
		if err != nil || tag != 13 || r.Position() != 14 {
			t.Fatalf("selector %d origin %d: %v", tag, r.Position(), err)
		}
		t.Logf("input=%x", cellBits(bits[:184]))
	} else {
		r, old = cellReader(t, bits, 14, extent)
	}
	v, err := celldata.DecodePacketSIStatusMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position() != 14+extent {
		t.Fatalf("content ended at %d want %d", r.Position(), 14+extent)
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	return v, r
}

// TS 44.060 V19.0.0 §11.2.17b pp16669–16694: each list entry has
// its printed prefix, eight-bit type, and one of the four MESS_REC values.
// §11.2.0.2 p13439 assigns 001101; §10.3 supplies the eight-bit MAC header.
func TestClause11217bReceptionStates(t *testing.T) {
	for direction := 0; direction < 2; direction++ {
		for state := 0; state < 4; state++ {
			for _, count := range []int{0, 15} {
				for flags := 0; flags < 4; flags++ {
					support, request := uint8(flags/2), uint8(flags%2)
					t.Run(fmt.Sprintf("direction%d/state%d/count%d/flags%d", direction, state, count, flags), func(t *testing.T) {
						prefix := "01000000" + "001101" + fmt.Sprintf("%01b%05b%03b", direction, 23, 5) + "1" + "00110101" + fmt.Sprintf("%02b", state)
						if state == 2 {
							prefix += "101" + fmt.Sprintf("%04b", count) + strings.Repeat("1", count+1)
						}
						if state == 3 {
							prefix += "101"
						}
						prefix += "01"                                       // supported list terminator and additional flag
						prefix += "1" + "10100101" + "1" + "01011010" + "01" // two unknown types, terminator and additional flag
						prefix += fmt.Sprintf("1%01b%01b", support, request) // release 6, PSCSI support, release request
						bits := cellPad(prefix, 184) + "101"
						v, r := cellReadStatus(t, bits, 170)
						if x, err := r.ReadUint(3); err != nil || x != 5 {
							t.Fatalf("sentinel %d: %v", x, err)
						}
						if v.BCCHCHANGEMARK != 5 {
							t.Fatalf("change mark %d", v.BCCHCHANGEMARK)
						}
						g := v.GLOBALTFI.UPLINKTFIChoice
						if direction == 0 && (g.UPLINKTFI == nil || g.UPLINKTFI.UPLINKTFI != 23) {
							t.Fatal("uplink TFI lost")
						}
						if direction == 1 && (g.DOWNLINKTFI == nil || g.DOWNLINKTFI.DOWNLINKTFI != 23) {
							t.Fatal("downlink TFI lost")
						}
						known := v.ReceivedSIMessageList
						unknown := v.ReceivedUnknownSIMessageList
						if len(known.SIMESSAGETYPEGroupList) != 1 || known.ADDITIONALMSGTYPE != 1 || known.SIMESSAGETYPEGroupList[0].SIMESSAGETYPE != 0x35 {
							t.Fatalf("supported list %+v", known)
						}
						choice := known.SIMESSAGETYPEGroupList[0].MESSRECChoice
						switch state {
						case 0:
							if choice.MESSREC == nil || choice.MESSREC.MESSREC != 0 {
								t.Fatal("not-received arm lost")
							}
						case 1:
							if choice.AltUnlabeled == nil || choice.AltUnlabeled.MESSREC != 1 {
								t.Fatal("single-instance arm lost")
							}
						case 2:
							if choice.SIXCOUNT == nil || choice.SIXCOUNT.MESSREC != 2 || choice.SIXCOUNT.SIXCOUNT != uint8(count) || choice.SIXCOUNT.SIXCHANGEMARK != 5 || choice.SIXCOUNT.InstanceBitmap.BitLength != count+1 {
								t.Fatal("partial-instance bitmap lost")
							}
						case 3:
							if choice.SIXCHANGEMARK == nil || choice.SIXCHANGEMARK.MESSREC != 3 || choice.SIXCHANGEMARK.SIXCHANGEMARK != 5 {
								t.Fatal("complete-instance arm lost")
							}
						}
						if len(unknown.SIMESSAGETYPEList) != 2 || unknown.SIMESSAGETYPEList[0].SIMESSAGETYPE != 0xa5 || unknown.SIMESSAGETYPEList[1].SIMESSAGETYPE != 0x5a || unknown.ADDITIONALMSGTYPE != 1 {
							t.Fatalf("unknown list %+v", unknown)
						}
						if v.PSCSISUPPORTChoice.PSCSISUPPORT == nil || v.PSCSISUPPORTChoice.PSCSISUPPORT.PSCSISUPPORT != support || v.PSCSISUPPORTChoice.PSCSISUPPORT.PSRELREQ != request {
							t.Fatal("release flags lost")
						}
						wire, err := runtime.SealBits(v, runtime.BitString{Bytes: cellBits(bits[:184]), BitLength: 184}, 184, runtime.BitString{}, r.Wire())
						if err != nil {
							t.Fatal(err)
						}
						w := runtime.NewWriter()
						w.WithWire(wire)
						if err = w.WriteUint(0x100d, 14); err != nil {
							t.Fatal(err)
						}
						if err = celldata.EncodePacketSIStatusMessageContentTo(w, v); err != nil {
							t.Fatal(err)
						}
						cellFinish(t, w, bits[:184])
						fresh := runtime.Canonical(v)
						w = runtime.NewWriter()
						w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
						if _, err = w.PushLimit(184); err != nil {
							t.Fatal(err)
						}
						if err = w.WriteUint(0x100d, 14); err != nil {
							t.Fatal(err)
						}
						if err = celldata.EncodePacketSIStatusMessageContentTo(w, fresh); err != nil {
							t.Fatal(err)
						}
						cellFinish(t, w, bits[:184])

						canonicalBytes, err := celldata.EncodePacketSIStatusMessageContentCanonicalAtLength(fresh, 22)
						if err != nil {
							t.Fatal(err)
						}
						byteValue, err := celldata.DecodePacketSIStatusMessageContent(canonicalBytes)
						if err != nil {
							t.Fatal(err)
						}
						if !runtime.SemanticallyEqual(fresh, byteValue.Value) {
							t.Fatal("canonical byte API lost fields")
						}
						replayBytes, err := celldata.EncodePacketSIStatusMessageContent(byteValue.Value)
						if err != nil || !bytes.Equal(replayBytes, canonicalBytes) {
							t.Fatal("canonical byte API replay")
						}
						if _, err = celldata.EncodePacketSIStatusMessageContentCanonical(fresh); err == nil {
							t.Fatal("canonical sender accepted missing extent")
						}
						// An edit must change precisely the change-mark field, preserving both lists.
						fresh.BCCHCHANGEMARK = 3
						w = runtime.NewWriter()
						w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
						if _, err = w.PushLimit(184); err != nil {
							t.Fatal(err)
						}
						if err = w.WriteUint(0x100d, 14); err != nil {
							t.Fatal(err)
						}
						if err = celldata.EncodePacketSIStatusMessageContentTo(w, fresh); err != nil {
							t.Fatal(err)
						}
						out, err := w.FinishBits(runtime.BitString{})
						if err != nil {
							t.Fatal(err)
						}
						expected := bits[:20] + "011" + bits[23:184]
						if !bytes.Equal(out.Bytes, cellBits(expected)) {
							t.Fatal("change-mark edit changed unrelated bits")
						}
					})
				}
			}
		}
	}
}

// Null ends exactly at the containing boundary; the zero arm ignores its rest.
func TestClause11217bReleaseAndEmptyLists(t *testing.T) {
	base := "01000000" + "001101" + strings.Repeat("0", 13)
	for _, tail := range []string{"", "0", "010101001110"} {
		bits := base + tail
		v, r := cellReadStatus(t, bits, len(bits)-14)
		if len(v.ReceivedSIMessageList.SIMESSAGETYPEGroupList) != 0 || len(v.ReceivedUnknownSIMessageList.SIMESSAGETYPEList) != 0 {
			t.Fatal("empty lists grew")
		}
		if tail == "" && v.PSCSISUPPORTChoice.AltUnlabeled == nil {
			t.Fatal("null release lost")
		}
		if tail != "" && v.PSCSISUPPORTChoice.NoString == nil {
			t.Fatal("ignored release tail lost")
		}
		wire, err := runtime.SealBits(v, runtime.BitString{Bytes: cellBits(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err = w.WriteUint(0x100d, 14); err != nil {
			t.Fatal(err)
		}
		if err = celldata.EncodePacketSIStatusMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		cellFinish(t, w, bits)
	}
	for n := 0; n < 13; n++ {
		bits := "01000000" + "001101" + strings.Repeat("0", n)
		r, _ := cellReader(t, bits, 14, n)
		if _, err := celldata.DecodePacketSIStatusMessageContentFrom(r); err == nil {
			t.Fatalf("accepted truncated mandatory extent %d", n)
		}
	}
	// A present list item without its type must fail before reading a following sentinel.
	bits := "01000000" + "001101" + strings.Repeat("0", 9) + "1" + "101"
	r, old := cellReader(t, bits, 14, 10)
	if _, err := celldata.DecodePacketSIStatusMessageContentFrom(r); err == nil {
		t.Fatal("accepted truncated SI type")
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	if r.Position() > 24 {
		t.Fatal("read outside bound")
	}
}
func FuzzClause11217bContent(f *testing.F) {
	f.Add([]byte{0, 0, 0}, uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, unused uint8) {
		if len(data) > 4096 {
			return
		}
		n := len(data) * 8
		if n > 0 {
			n -= int(unused % 8)
		}
		r, err := runtime.NewChoiceReaderBits(data, n, nil)
		if err != nil {
			return
		}
		v, err := celldata.DecodePacketSIStatusMessageContentFrom(r)
		if err != nil {
			return
		}
		if r.Position() > n {
			t.Fatal("read outside extent")
		}
		wire, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: n}, r.Position(), runtime.BitString{}, r.Wire())
		if err != nil {
			return
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err = celldata.EncodePacketSIStatusMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		if out.BitLength != n {
			t.Fatal("replay extent")
		}
		for i := 0; i < n; i++ {
			if (out.Bytes[i/8]>>uint(7-i%8))&1 != (data[i/8]>>uint(7-i%8))&1 {
				t.Fatal("replay bits")
			}
		}
	})
}

func TestClause11217bListGrowth(t *testing.T) {
	// Artificial containing extents stress the list allocator. The complete-block
	// cases above, rather than these larger probes, validate the radio carrier.
	for _, count := range []int{1, 16, 256, 1024} {
		bits := "01000000" + "001101" + strings.Repeat("0", 9) + strings.Repeat("1"+"00110101"+"00", count) + "00" + "00" + "0"
		v, _ := cellReadStatus(t, bits, len(bits)-14)
		if len(v.ReceivedSIMessageList.SIMESSAGETYPEGroupList) != count {
			t.Fatal("list length")
		}
		allocations := testing.AllocsPerRun(1, func() {
			r, _ := cellReader(t, bits, 14, len(bits)-14)
			if _, err := celldata.DecodePacketSIStatusMessageContentFrom(r); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && allocations > float64(count*160+512) {
			t.Fatalf("nonlinear list allocations at %d: %f", count, allocations)
		}
	}
}

func TestClause11217bRejectsInvalidTypedFields(t *testing.T) {
	// One partial-reception item, count zero, and its required one-bit bitmap.
	prefix := "01000000" + "001101" + strings.Repeat("0", 9) + "1" + "00110101" + "10" + "101" + "0000" + "1" + "00" + "00" + "110"
	value, _ := cellReadStatus(t, cellPad(prefix, 184), 170)
	mutations := []struct {
		name   string
		change func(*celldata.PacketSIStatusMessageContent)
	}{
		{"change-mark-width", func(v *celldata.PacketSIStatusMessageContent) { v.BCCHCHANGEMARK = 8 }},
		{"tfi-width", func(v *celldata.PacketSIStatusMessageContent) { v.GLOBALTFI.UPLINKTFIChoice.UPLINKTFI.UPLINKTFI = 32 }},
		{"reception-arm-literal", func(v *celldata.PacketSIStatusMessageContent) {
			v.ReceivedSIMessageList.SIMESSAGETYPEGroupList[0].MESSRECChoice.SIXCOUNT.MESSREC = 3
		}},
		{"instance-count-width", func(v *celldata.PacketSIStatusMessageContent) {
			v.ReceivedSIMessageList.SIMESSAGETYPEGroupList[0].MESSRECChoice.SIXCOUNT.SIXCOUNT = 16
		}},
		{"short-instance-bitmap", func(v *celldata.PacketSIStatusMessageContent) {
			v.ReceivedSIMessageList.SIMESSAGETYPEGroupList[0].MESSRECChoice.SIXCOUNT.InstanceBitmap = runtime.BitString{}
		}},
		{"long-instance-bitmap", func(v *celldata.PacketSIStatusMessageContent) {
			v.ReceivedSIMessageList.SIMESSAGETYPEGroupList[0].MESSRECChoice.SIXCOUNT.InstanceBitmap = runtime.BitString{Bytes: []byte{0xc0}, BitLength: 2}
		}},
		{"known-additional-width", func(v *celldata.PacketSIStatusMessageContent) { v.ReceivedSIMessageList.ADDITIONALMSGTYPE = 2 }},
		{"unknown-additional-width", func(v *celldata.PacketSIStatusMessageContent) { v.ReceivedUnknownSIMessageList.ADDITIONALMSGTYPE = 2 }},
		{"support-width", func(v *celldata.PacketSIStatusMessageContent) { v.PSCSISUPPORTChoice.PSCSISUPPORT.PSCSISUPPORT = 2 }},
		{"release-request-width", func(v *celldata.PacketSIStatusMessageContent) { v.PSCSISUPPORTChoice.PSCSISUPPORT.PSRELREQ = 2 }},
	}
	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			v := runtime.Canonical(value)
			m.change(&v)
			if _, err := celldata.EncodePacketSIStatusMessageContentCanonicalAtLength(v, 22); err == nil {
				t.Fatal("invalid typed field accepted")
			}
		})
	}
}

// TS 44.060 V19.0.0 §11.2.17b pp16674–16677 permits a null
// release suffix; its sender path does not fill a larger containing extent.
func TestClause11217bCanonicalUnfilledExtent(t *testing.T) {
	d, err := celldata.Lookup("Packet SI Status message content")
	if err != nil {
		t.Fatal(err)
	}
	value, err := d.Decode([]byte{0, 0})
	if err != nil {
		t.Fatal(err)
	}
	out, err := d.CanonicalAtLength(value, 3)
	var extent *runtime.ExtentError
	if len(out) != 0 || !errors.As(err, &extent) || extent.Required || extent.Actual != 2 || extent.Minimum != 3 || extent.Maximum != 3 {
		t.Fatalf("unfilled target %x: %v (%+v)", out, err, extent)
	}
}
