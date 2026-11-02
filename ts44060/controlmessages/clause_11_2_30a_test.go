package controlmessages_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.30a paragraphs 18882–18885;
// the external uplink selector is 001001 (§11.2.0.2 paragraph 13436).
func TestClause11230aPause(t *testing.T) {
	for _, identity := range []uint32{0, 0x10203040, 0xffffffff} {
		for _, rai := range []uint64{0, 0x123456789abc, 0xffffffffffff} {
			for _, ignored := range []bool{false, true} {
				raw := uplinkControlBits(fmt.Sprintf("01000000001001%032b%048b", identity, rai), 184, ignored)
				t.Logf("input=%x", raw)
				r, c := uplinkContentReader(t, raw, 9)
				v, e := controlmessages.DecodePacketPauseMessageContentFrom(r)
				if e != nil || v.TLLI != identity || v.RAI != rai || r.Position() != 184 {
					t.Fatalf("typed %+v at %d: %v", v, r.Position(), e)
				}
				report := c.TakeReceiverReport()
				if (len(report.Regions) > 0) != ignored {
					t.Fatalf("padding regions %+v", report)
				}
				uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketPauseMessageContentTo(w, v) })
				if !ignored {
					cv := runtime.Canonical(v)
					cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
					if got := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketPauseMessageContentTo(w, cv) }); !bytes.Equal(got, raw) {
						t.Fatalf("canonical %x", got)
					}
					cv.RAI ^= 1
					edited := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketPauseMessageContentTo(w, cv) })
					er, _ := uplinkContentReader(t, edited, 9)
					ev, err := controlmessages.DecodePacketPauseMessageContentFrom(er)
					if err != nil || ev.RAI != rai^1 || ev.TLLI != identity {
						t.Fatalf("edit %+v %v", ev, err)
					}
				}
				for _, end := range []int{14, 45, 46, 93} {
					tr, _ := runtime.NewChoiceReaderBits(raw, end, nil)
					if _, e = tr.ReadUint(14); e != nil {
						t.Fatal(e)
					}
					if _, e = controlmessages.DecodePacketPauseMessageContentFrom(tr); e == nil {
						t.Fatalf("truncation accepted at %d", end)
					}
				}
			}
		}
	}
	// The printed null padding arm ends exactly after the 80 payload bits.
	raw := uplinkControlBits("01000000001001"+fmt.Sprintf("%080b", 0), 184, false)
	rd, _ := uplinkContentReader(t, raw, 9)
	old, e := rd.PushLimit(80)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = controlmessages.DecodePacketPauseMessageContentFrom(rd); e != nil || rd.Position() != 94 {
		t.Fatalf("null padding at %d: %v", rd.Position(), e)
	}
	if e = rd.PopLimit(old); e != nil {
		t.Fatal(e)
	}
	sentinel, e := rd.ReadUint(3)
	if e != nil || sentinel != 2 {
		t.Fatalf("following bits %d: %v", sentinel, e)
	}
}

func FuzzClause11230aPause(f *testing.F) {
	f.Add(uplinkControlBits(fmt.Sprintf("01000000001001%032b%048b", uint32(0x10203040), uint64(0x123456789abc)), 184, false))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, e := runtime.NewChoiceReaderBits(raw, 184, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = r.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketPauseMessageContentFrom(r)
		if e == nil {
			uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketPauseMessageContentTo(w, v) })
		}
	})
}
