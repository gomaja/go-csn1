package controlmessages_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func TestClause1128DownlinkContent(t *testing.T) {
	for page := 0; page < 4; page++ {
		for _, levels := range [][]uint8{nil, {0, 5, 10, 15}} {
			t.Run(fmt.Sprintf("page%d/persistence%t", page, levels != nil), func(t *testing.T) {
				prefix := fmt.Sprintf("01000000100101%02b", page)
				if levels == nil {
					prefix += "0"
				} else {
					prefix += "1"
					for _, level := range levels {
						prefix += fmt.Sprintf("%04b", level)
					}
				}
				data := controlBits(t, prefix, 184, false)
				t.Logf("input=%x", data)
				r, c := contentReader(t, data, 184, runtime.RLCMACDownlink, 0x25)
				v, err := controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				known := v.PAGEMODEGroupFallback.Known
				if known == nil || int(known.PAGEMODE) != page || r.Position() != 184 {
					t.Fatalf("typed content %+v at %d", v, r.Position())
				}
				if levels == nil {
					if known.PERSISTENCELEVELList != nil {
						t.Fatal("unexpected persistence")
					}
				} else if known.PERSISTENCELEVELList == nil || !bytes.Equal(*known.PERSISTENCELEVELList, levels) {
					t.Fatalf("persistence %+v", known.PERSISTENCELEVELList)
				}
				if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
					t.Fatalf("unexpected regions %+v", report)
				}
				replayControl(t, data, r, func(w *runtime.Writer) error {
					return controlmessages.EncodePacketDownlinkDummyControlBlockMessageContentTo(w, v)
				})
				// A canonical send uses the same writer and absolute bit phase as framing.
				fresh := runtime.Canonical(v)
				w := runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
				if _, err = w.PushLimit(184); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(0x40, 8); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(0x25, 6); err != nil {
					t.Fatal(err)
				}
				if err = controlmessages.EncodePacketDownlinkDummyControlBlockMessageContentTo(w, fresh); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != 184 || !bytes.Equal(out.Bytes, data) {
					t.Fatalf("canonical %x/%d: %v", out.Bytes, out.BitLength, err)
				}
				edited := v
				edited.PAGEMODEGroupFallback.Known.PAGEMODE = uint8((page + 1) % 4)
				ew := runtime.NewWriter()
				ew.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
				if _, err = ew.PushLimit(184); err != nil {
					t.Fatal(err)
				}
				if err = ew.WriteUint(0x40, 8); err != nil {
					t.Fatal(err)
				}
				if err = ew.WriteUint(0x25, 6); err != nil {
					t.Fatal(err)
				}
				if err = controlmessages.EncodePacketDownlinkDummyControlBlockMessageContentTo(ew, runtime.Canonical(edited)); err != nil {
					t.Fatal(err)
				}
				editBytes, err := ew.FinishBits(runtime.BitString{})
				if err != nil {
					t.Fatal(err)
				}
				er, _ := contentReader(t, editBytes.Bytes, 184, runtime.RLCMACDownlink, 0x25)
				ev, err := controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(er)
				if err != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != uint8((page+1)%4) {
					t.Fatalf("edit %+v %v", ev, err)
				}
			})
		}
	}
}

func TestClause1128DownlinkIgnoreAndTruncation(t *testing.T) {
	data := controlBits(t, "01000000100101100", 184, true)
	r, c := contentReader(t, data, 184, runtime.RLCMACDownlink, 0x25)
	v, err := controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Known.PAGEMODE != 2 {
		t.Fatal("Ignore lost known PAGE_MODE")
	}
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore || report.Regions[0].Location.BitOffset != 18 {
		t.Fatalf("Ignore report %+v", report)
	}
	replayControl(t, data, r, func(w *runtime.Writer) error {
		return controlmessages.EncodePacketDownlinkDummyControlBlockMessageContentTo(w, v)
	})
	for width := 0; width < 19; width++ {
		prefix := "001"
		if width < 3 {
			prefix = prefix[:width]
		}
		raw := controlBits(t, prefix, width, false)
		cc := runtime.NewReceiverCollector()
		rr, e := sourceContentFixtureReader(t, raw, width, cc)
		if e != nil {
			t.Fatal(e)
		}
		partial, e := controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(rr)
		if e != nil || partial.PAGEMODEGroupFallback.Ignored == nil || rr.Position() != width+14 {
			t.Fatalf("truncated persistence at %d: %+v %v", width, partial, e)
		}
		facts := cc.TakeReceiverReport()
		if len(facts.Regions) != 1 || facts.Regions[0].Label != runtime.DistributionPartError || facts.Regions[0].Location.Width != width {
			t.Fatalf("truncation report at %d: %+v", width, facts)
		}
	}
}

func FuzzClause1128DownlinkContent(f *testing.F) {
	f.Add([]byte{0, 0x2b, 0x2b}, uint8(17))
	f.Fuzz(func(t *testing.T, data []byte, n uint8) {
		if len(data) > 4096 {
			return
		}
		extent := min(len(data)*8, int(n))
		r, err := sourceContentFixtureReader(t, data, extent, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(r)
		if r.Position() > extent+14 {
			t.Fatal("crossed extent")
		}
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12937: the externally supplied message
// extent ends at the control block, even when more input follows it.
func TestClause1128BoundSentinel(t *testing.T) {
	data := append(controlBits(t, "01000000100101000", 184, false), 0xa5)
	r, err := runtime.NewChoiceReaderBits(data, 192, nil)
	if err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(6); err != nil {
		t.Fatal(err)
	}
	_, err = controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(r)
	if err != nil || r.Position() != 184 {
		t.Fatalf("content bound at %d: %v", r.Position(), err)
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(8)
	if err != nil || sentinel != 0xa5 {
		t.Fatalf("sentinel %02x: %v", sentinel, err)
	}
}

func TestClause1128PersistenceSendBounds(t *testing.T) {
	data := controlBits(t, "01000000100101001"+"0000010110101111", 184, false)
	r, _ := contentReader(t, data, 184, runtime.RLCMACDownlink, 0x25)
	v, err := controlmessages.DecodePacketDownlinkDummyControlBlockMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, levels := range [][]uint8{{0, 5, 10}, {0, 5, 10, 15, 0}, {0, 5, 10, 16}} {
		invalid := runtime.Canonical(v)
		invalid.PAGEMODEGroupFallback.Known.PERSISTENCELEVELList = &levels
		w := runtime.NewWriter()
		if err = w.WriteUint(0, 14); err != nil {
			t.Fatal(err)
		}
		if err = controlmessages.EncodePacketDownlinkDummyControlBlockMessageContentTo(w, invalid); err == nil {
			t.Fatalf("invalid persistence accepted: %v", levels)
		}
	}
}
