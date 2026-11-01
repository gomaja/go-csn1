package controlmessages_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.35 paragraphs 19955–19958 and
// §11.2.0.2 paragraph 13440: external selector 001110.
func TestClause11235CSRequest(t *testing.T) {
	for direction := 0; direction < 2; direction++ {
		for _, tfi := range []int{0, 17, 31} {
			for _, cause := range []uint8{0, 0x45, 0xff} {
				for _, ignored := range []bool{false, true} {
					raw := uplinkControlBits(fmt.Sprintf("01000000001110%01b%05b%08b", direction, tfi, cause), 184, ignored)
					t.Logf("input=%x", raw)
					r, c := uplinkContentReader(t, raw, 14)
					v, e := controlmessages.DecodePacketCSRequestMessageContentFrom(r)
					if e != nil || v.ESTABLISHMENTCAUSE != cause || r.Position() != 184 {
						t.Fatalf("typed %+v %v", v, e)
					}
					choice := v.GLOBALTFI.UPLINKTFIChoice
					if direction == 0 {
						if choice.UPLINKTFI == nil || int(choice.UPLINKTFI.UPLINKTFI) != tfi || choice.DOWNLINKTFI != nil {
							t.Fatalf("uplink TFI %+v", choice)
						}
					} else {
						if choice.DOWNLINKTFI == nil || int(choice.DOWNLINKTFI.DOWNLINKTFI) != tfi || choice.UPLINKTFI != nil {
							t.Fatalf("downlink TFI %+v", choice)
						}
					}
					if report := c.TakeReceiverReport(); (len(report.Regions) > 0) != ignored {
						t.Fatalf("regions %+v", report)
					}
					uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketCSRequestMessageContentTo(w, v) })
					if !ignored {
						cv := runtime.Canonical(v)
						cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
						if got := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketCSRequestMessageContentTo(w, cv) }); !bytes.Equal(got, raw) {
							t.Fatalf("canonical %x", got)
						}
						cv.ESTABLISHMENTCAUSE ^= 0x80
						edit := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketCSRequestMessageContentTo(w, cv) })
						er, _ := uplinkContentReader(t, edit, 14)
						ev, err := controlmessages.DecodePacketCSRequestMessageContentFrom(er)
						if err != nil || ev.ESTABLISHMENTCAUSE != cause^0x80 {
							t.Fatalf("edit %+v %v", ev, err)
						}
					}
					for _, end := range []int{14, 19, 20, 27} {
						tr, _ := runtime.NewChoiceReaderBits(raw, end, nil)
						if _, e = tr.ReadUint(14); e != nil {
							t.Fatal(e)
						}
						if _, e = controlmessages.DecodePacketCSRequestMessageContentFrom(tr); e == nil {
							t.Fatalf("truncation accepted at %d", end)
						}
					}
				}
			}
		}
	}
}
func FuzzClause11235CSRequest(f *testing.F) {
	f.Add(uplinkControlBits("0100000000111001010101000101", 184, false))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		if _, e := r.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketCSRequestMessageContentFrom(r)
		if e == nil {
			uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketCSRequestMessageContentTo(w, v) })
		}
	})
}
