package controlmessages_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

// TS 44.060 V19.0.0 §11.2.8b paragraphs 15422–15426, §10.3.2
// figure 10.3.2.1 paragraphs 7887–7957 and §11.2.0.2 paragraph 13427:
// content begins after the MAC octet
// and the external six-bit selector, at absolute bit 14.
func TestClause1128bUplinkContent(t *testing.T) {
	for _, identity := range []uint32{0x10203040, 0xfedcba98} {
		for _, extension := range []int{-1, 0, 15} {
			t.Run(fmt.Sprintf("identity%08x/extension%d", identity, extension), func(t *testing.T) {
				prefix := fmt.Sprintf("01000000000011%032b1", identity)
				if extension < 0 {
					prefix += "0"
				} else {
					prefix += fmt.Sprintf("1%04b", extension)
				}
				data := controlBits(t, prefix, 184, false)
				t.Logf("input=%x", data)
				r, c := contentReader(t, data, 184, runtime.RLCMACUplink, 3)
				v, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				arm := v.GRNTIExtensionChoice.GRNTIExtension
				if v.TLLIGRNTI != identity || arm == nil || r.Position() != 184 {
					t.Fatalf("typed content %+v at %d", v, r.Position())
				}
				if extension < 0 {
					if arm.GRNTIExtension != nil {
						t.Fatal("unexpected extension")
					}
				} else if arm.GRNTIExtension == nil || int(*arm.GRNTIExtension) != extension {
					t.Fatalf("extension %+v", arm.GRNTIExtension)
				}
				if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
					t.Fatalf("unexpected report %+v", report)
				}
				replayControl(t, data, r, func(w *runtime.Writer) error {
					return controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(w, v)
				})
				fresh := runtime.Canonical(v)
				w := runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
				if _, err = w.PushLimit(184); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(0x40, 8); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(3, 6); err != nil {
					t.Fatal(err)
				}
				if err = controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(w, fresh); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != 184 || !bytes.Equal(out.Bytes, data) {
					t.Fatalf("canonical %x/%d %v", out.Bytes, out.BitLength, err)
				}
				fresh.TLLIGRNTI ^= 1
				ew := runtime.NewWriter()
				ew.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
				if _, err = ew.PushLimit(184); err != nil {
					t.Fatal(err)
				}
				if err = ew.WriteUint(0x40, 8); err != nil {
					t.Fatal(err)
				}
				if err = ew.WriteUint(3, 6); err != nil {
					t.Fatal(err)
				}
				if err = controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(ew, fresh); err != nil {
					t.Fatal(err)
				}
				out, err = ew.FinishBits(runtime.BitString{})
				if err != nil {
					t.Fatal(err)
				}
				er, _ := contentReader(t, out.Bytes, 184, runtime.RLCMACUplink, 3)
				ev, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(er)
				if err != nil || ev.TLLIGRNTI != identity^1 {
					t.Fatalf("edit %+v %v", ev, err)
				}
			})
		}
	}
}

func TestClause1128bUplinkLegacyNullAndIgnore(t *testing.T) {
	for _, suffix := range []string{"", "0", "10", "111111"} {
		prefix := fmt.Sprintf("%032b", uint32(0x10203040))
		extent := len(prefix) + len(suffix)
		data := controlBits(t, prefix+suffix, extent, false)
		c := runtime.NewReceiverCollector()
		r, err := sourceContentFixtureReader(t, data, extent, c)
		if err != nil {
			t.Fatal(err)
		}
		v, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		if v.TLLIGRNTI != 0x10203040 || r.Position() != extent+14 {
			t.Fatalf("legacy/null %+v at %d", v, r.Position())
		}
		switch suffix {
		case "":
			if v.GRNTIExtensionChoice.AltUnlabeled == nil {
				t.Fatal("null arm absent")
			}
		case "0":
			if v.GRNTIExtensionChoice.NoString == nil {
				t.Fatal("legacy Ignore tail absent")
			}
		default:
			if v.GRNTIExtensionChoice.GRNTIExtension == nil {
				t.Fatal("release 5 arm absent")
			}
		}
	}
	data := controlBits(t, fmt.Sprintf("01000000000011%032b10", uint32(0x10203040)), 184, true)
	r, c := contentReader(t, data, 184, runtime.RLCMACUplink, 3)
	v, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if v.TLLIGRNTI != 0x10203040 || v.GRNTIExtensionChoice.GRNTIExtension == nil {
		t.Fatalf("Ignore lost identity %+v", v)
	}
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore {
		t.Fatalf("Ignore report %+v", report)
	}
	replayControl(t, data, r, func(w *runtime.Writer) error {
		return controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(w, v)
	})
	for width := 0; width < 32; width++ {
		raw := make([]byte, 4)
		rr, e := sourceContentFixtureReader(t, raw, width, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(rr); e == nil {
			t.Fatalf("truncated identity accepted at %d", width)
		}
	}
}

func FuzzClause1128bUplinkContent(f *testing.F) {
	f.Add([]byte{0x10, 0x20, 0x30, 0x40, 0x80}, uint8(34))
	f.Fuzz(func(t *testing.T, data []byte, n uint8) {
		if len(data) > 4096 {
			return
		}
		extent := min(len(data)*8, int(n))
		r, err := sourceContentFixtureReader(t, data, extent, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
		if r.Position() > extent+14 {
			t.Fatal("crossed extent")
		}
	})
}

func TestClause1128bLegacyTailAndSendBounds(t *testing.T) {
	prefix := fmt.Sprintf("01000000000011%032b0", uint32(0x10203040))
	data := controlBits(t, prefix, 184, true)
	r, c := contentReader(t, data, 184, runtime.RLCMACUplink, 3)
	v, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if v.TLLIGRNTI != 0x10203040 || v.GRNTIExtensionChoice.NoString == nil {
		t.Fatalf("legacy identity/tail lost: %+v", v)
	}
	if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
		t.Fatalf("ordinary ignored tail reported as error: %+v", report)
	}
	replayControl(t, data, r, func(w *runtime.Writer) error {
		return controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(w, v)
	})
	for _, suffix := range []string{"1", "11", "11111"} {
		raw := controlBits(t, fmt.Sprintf("%032b", uint32(0x10203040))+suffix, 32+len(suffix), false)
		rr, e := sourceContentFixtureReader(t, raw, 32+len(suffix), nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(rr); e == nil {
			t.Fatalf("truncated extension accepted: %s", suffix)
		}
	}
	data = controlBits(t, fmt.Sprintf("01000000000011%032b111111", uint32(0x10203040)), 184, false)
	r, _ = contentReader(t, data, 184, runtime.RLCMACUplink, 3)
	v, err = controlmessages.DecodePacketUplinkDummyControlBlockMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	invalid := runtime.Canonical(v)
	bad := uint8(16)
	invalid.GRNTIExtensionChoice.GRNTIExtension.GRNTIExtension = &bad
	if err = controlmessages.EncodePacketUplinkDummyControlBlockMessageContentTo(sourceContentFixtureWriter(t), invalid); err == nil {
		t.Fatal("five-bit extension admitted into a four-bit field")
	}
}

// TS 44.060 V19.0.0 §11.2.8b paragraph 15424: the earlier-release
// arm discards its received ignored tail during a fresh canonical send.
func TestClause1128bCanonicalLegacyTailExtent(t *testing.T) {
	decoded, err := controlmessages.DecodePacketUplinkDummyControlBlockMessageContent([]byte{0, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := controlmessages.EncodePacketUplinkDummyControlBlockMessageContent(decoded.Value)
	if err != nil || !bytes.Equal(replay, []byte{0, 0, 0, 0, 0}) {
		t.Fatalf("legacy replay %x %v", replay, err)
	}
	_, err = controlmessages.EncodePacketUplinkDummyControlBlockMessageContentCanonicalAtLength(decoded.Value, 6)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Actual != 5 || extent.Minimum != 6 || extent.Maximum != 6 {
		t.Fatalf("expected unfilled explicit extent, got %v", err)
	}
}

func sourceContentFixtureWriter(t testing.TB) *runtime.Writer {
	t.Helper()
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	return w
}
