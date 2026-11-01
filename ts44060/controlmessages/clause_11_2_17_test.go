package controlmessages_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.17 paragraphs 16570–16590 and
// §11.2.0.2 paragraph 13432: external selector 000111.
func TestClause11217PSIStatus(t *testing.T) {
	for direction := 0; direction < 2; direction++ {
		for _, count := range []int{-1, 0, 1, 15} {
			for _, entries := range []int{0, 1, 2} {
				for _, req := range []int{0, 1} {
					bits := fmt.Sprintf("01000000000111%01b10101%03b", direction, 5)
					for i := 0; i < entries; i++ {
						bits += fmt.Sprintf("1%06b%02b", i+1, i%4)
						if count < 0 {
							bits += "0"
						} else {
							bits += fmt.Sprintf("1%04b", count)
							for k := 0; k <= count; k++ {
								bits += fmt.Sprint(k % 2)
							}
						}
					}
					bits += "01"
					for i := 0; i < entries; i++ {
						bits += fmt.Sprintf("1%06b", 60+i)
					}
					bits += "01"
					bits += fmt.Sprintf("1%01b", req)
					raw := uplinkControlBits(bits, 184, false)
					t.Logf("input=%x", raw)
					r, c := uplinkContentReader(t, raw, 7)
					v, e := controlmessages.DecodePacketPSIStatusMessageContentFrom(r)
					if e != nil || v.PBCCHCHANGEMARK != 5 || r.Position() != 184 {
						t.Fatalf("typed %+v at %d %v", v, r.Position(), e)
					}
					list := v.ReceivedPSIMessageList
					unknown := v.ReceivedUnknownPSIMessageList
					if len(list.MESSAGETYPEGroupList) != entries || list.ADDITIONALMSGTYPE != 1 || len(unknown.MESSAGETYPEList) != entries || unknown.ADDITIONALMSGTYPE != 1 {
						t.Fatalf("lists %+v %+v", list, unknown)
					}
					for i, item := range list.MESSAGETYPEGroupList {
						if int(item.MESSAGETYPE) != i+1 || int(item.PSIXCHANGEMARK) != i%4 {
							t.Fatalf("item %+v", item)
						}
						if count < 0 {
							if item.PSIXCOUNTGroup != nil {
								t.Fatal("unexpected bitmap")
							}
						} else {
							g := item.PSIXCOUNTGroup
							if g == nil || int(g.PSIXCOUNT) != count || g.InstanceBitmap.BitLength != count+1 {
								t.Fatalf("bitmap %+v", g)
							}
							for k := 0; k <= count; k++ {
								if (g.InstanceBitmap.Bytes[k/8]>>uint(7-k%8))&1 != byte(k%2) {
									t.Fatalf("bitmap bits %+v", g)
								}
							}
						}
					}
					for i, item := range unknown.MESSAGETYPEList {
						if int(item.MESSAGETYPE) != 60+i {
							t.Fatalf("unknown item %+v", item)
						}
					}
					if v.PSRELREQChoice.PSRELREQ == nil || int(v.PSRELREQChoice.PSRELREQ.PSRELREQ) != req {
						t.Fatalf("REL6 %+v", v.PSRELREQChoice)
					}
					if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
						t.Fatalf("regions %+v", report)
					}
					uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketPSIStatusMessageContentTo(w, v) })
					cv := runtime.Canonical(v)
					cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
					if got := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketPSIStatusMessageContentTo(w, cv) }); !bytes.Equal(got, raw) {
						t.Fatalf("canonical %x", got)
					}
					cv.PBCCHCHANGEMARK = 2
					edit := encodeUplinkContent(t, raw, 184, cw, func(w *runtime.Writer) error { return controlmessages.EncodePacketPSIStatusMessageContentTo(w, cv) })
					er, _ := uplinkContentReader(t, edit, 7)
					ev, err := controlmessages.DecodePacketPSIStatusMessageContentFrom(er)
					if err != nil || ev.PBCCHCHANGEMARK != 2 {
						t.Fatalf("edit %+v %v", ev, err)
					}
				}
			}
		}
	}
}

func TestClause11217LegacyAndBounds(t *testing.T) {
	// Empty lists; additional flags 0 and 1.
	prefix := "01000000000111" + "010101" + "101" + "00" + "01"
	raw := uplinkControlBits(prefix+"0", 184, true)
	r, _ := uplinkContentReader(t, raw, 7)
	v, e := controlmessages.DecodePacketPSIStatusMessageContentFrom(r)
	if e != nil || v.PSRELREQChoice.NoString == nil || r.Position() != 184 {
		t.Fatalf("legacy %+v %v", v, e)
	}
	uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketPSIStatusMessageContentTo(w, v) })
	nr, _ := uplinkContentReader(t, raw, 7)
	old, e := nr.PushLimit(len(prefix) - 14)
	if e != nil {
		t.Fatal(e)
	}
	nv, e := controlmessages.DecodePacketPSIStatusMessageContentFrom(nr)
	if e != nil || nv.PSRELREQChoice.AltUnlabeled == nil || nr.Position() != len(prefix) {
		t.Fatalf("null %+v %v", nv, e)
	}
	if e = nr.PopLimit(old); e != nil {
		t.Fatal(e)
	}
	next, e := nr.ReadUint(2)
	if e != nil || next != 1 {
		t.Fatalf("sentinel %d %v", next, e)
	}
	for end := 14; end < len(prefix); end++ {
		tr, _ := runtime.NewChoiceReaderBits(raw, end, nil)
		if _, e = tr.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		if _, e = controlmessages.DecodePacketPSIStatusMessageContentFrom(tr); e == nil {
			t.Fatalf("truncated required field at %d", end)
		}
	}
}
func FuzzClause11217PSIStatus(f *testing.F) {
	f.Add(uplinkControlBits("01000000000111010101101000110", 184, false))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		if _, e := r.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketPSIStatusMessageContentFrom(r)
		if e == nil {
			uplinkReplay(t, raw, r, func(w *runtime.Writer) error { return controlmessages.EncodePacketPSIStatusMessageContentTo(w, v) })
		}
	})
}

// TS 44.060 V19.0.0 §11.2.17 pp16570–16590: each known
// entry below has its prefix, six-bit type, two-bit change mark and absent
// bitmap flag. These growing component extents are outside a radio carrier.
func TestClause11217ListGrowth(t *testing.T) {
	for _, count := range []int{1, 16, 256, 1024} {
		bits := "01000000" + "000111" + "010101" + "101" + strings.Repeat("1000001010", count) + "00" + "00"
		raw := make([]byte, (len(bits)+3+7)/8)
		for i, b := range bits + "101" {
			if b == '1' {
				raw[i/8] |= 1 << uint(7-i%8)
			}
		}
		r, err := runtime.NewChoiceReaderBits(raw, len(bits)+3, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(len(bits) - 14)
		if err != nil {
			t.Fatal(err)
		}
		v, err := controlmessages.DecodePacketPSIStatusMessageContentFrom(r)
		if err != nil || len(v.ReceivedPSIMessageList.MESSAGETYPEGroupList) != count || r.Position() != len(bits) {
			t.Fatalf("count %d at %d: %v", count, r.Position(), err)
		}
		for _, item := range v.ReceivedPSIMessageList.MESSAGETYPEGroupList {
			if item.MESSAGETYPE != 1 || item.PSIXCHANGEMARK != 1 || item.PSIXCOUNTGroup != nil {
				t.Fatalf("typed item %+v", item)
			}
		}
		if len(v.ReceivedUnknownPSIMessageList.MESSAGETYPEList) != 0 || v.ReceivedPSIMessageList.ADDITIONALMSGTYPE != 0 || v.ReceivedUnknownPSIMessageList.ADDITIONALMSGTYPE != 0 {
			t.Fatal("list termination changed")
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if sentinel, err := r.ReadUint(3); err != nil || sentinel != 5 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
		allocations := testing.AllocsPerRun(1, func() {
			r, err := runtime.NewChoiceReaderBits(raw, len(bits), nil)
			if err != nil {
				panic(err)
			}
			if _, err = r.ReadUint(14); err != nil {
				panic(err)
			}
			value, err := controlmessages.DecodePacketPSIStatusMessageContentFrom(r)
			if err != nil {
				panic(err)
			}
			if len(value.ReceivedPSIMessageList.MESSAGETYPEGroupList) != count || r.Position() != len(bits) {
				panic("wrong growth decode")
			}
		})
		if !raceEnabled && allocations > float64(count*160+512) {
			t.Fatalf("nonlinear list allocations at %d: %f", count, allocations)
		}
	}
}
