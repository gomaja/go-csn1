package controlmessages_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

type physicalInformationResolver struct{}

func (physicalInformationResolver) Resolve(ctx runtime.RLCMACContext, selector uint64) (controlblock.Binding, error) {
	if ctx.Direction != runtime.RLCMACDownlink || ctx.Mode != runtime.RLCMACDBPSCH || selector != 18 {
		return controlblock.Binding{}, fmt.Errorf("unexpected Physical Information context or selector")
	}
	return controlblock.Binding{Kind: controlblock.KnownMessage, Ownership: controlblock.InternalSelector, DecodeFrom: func(r *runtime.Reader, _ runtime.RLCMACContext) (any, error) {
		return controlmessages.DecodePhysicalInformationMessageContentFrom(r)
	}, EncodeTo: func(w *runtime.Writer, v any, _ runtime.RLCMACContext) error {
		x, ok := v.(controlmessages.PhysicalInformationMessageContent)
		if !ok {
			return fmt.Errorf("unexpected Physical Information body %T", v)
		}
		return controlmessages.EncodePhysicalInformationMessageContentTo(w, x)
	}}, nil
}

// TS 44.060 V19.0.0 §11.1.1.3 paragraphs 13005–13015: the complete
// logical DBPSCH message starts with its own six-bit selector, followed by
// specific information and padding (which may be null). §11.2.34 paragraphs
// 19937–19945 fixes network-to-mobile direction and selector 010010.
// Its physical carrier mapping is outside this logical-message test.
func TestClause11234PhysicalInformation(t *testing.T) {
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACDBPSCH, Channel: runtime.RLCMACPACCH}
	for _, timing := range []int{0, 1, 42, 85, 128, 165, 254, 255} {
		for tail := 0; tail <= 8; tail++ {
			extent := 14 + tail
			raw := simpleControlBits(fmt.Sprintf("010010%08b", timing), extent)
			c := runtime.NewReceiverCollector()
			r, e := runtime.NewChoiceReaderBits(raw, extent, c)
			if e != nil {
				t.Fatal(e)
			}
			m, e := controlblock.DecodeMessageFrom(r, ctx, physicalInformationResolver{})
			if e != nil {
				t.Fatal(e)
			}
			v, ok := m.Body.(controlmessages.PhysicalInformationMessageContent)
			if !ok || m.Selector != 18 || v.MESSAGETYPE != 18 || v.Group.TIMINGADVANCEVALUE != uint8(timing) || r.Position() != extent {
				t.Fatalf("body %+v selector %d position %d", v, m.Selector, r.Position())
			}
			if report := c.TakeReceiverReport(); len(report.ZeroFill) != 0 || len(report.Regions) != 0 {
				t.Fatalf("normal report %+v", report)
			}
			simpleControlReplay(t, raw, extent, 0, r, func(w *runtime.Writer) error {
				return controlblock.EncodeMessageTo(w, ctx, m, physicalInformationResolver{})
			})
			cv := runtime.Canonical(v)
			out := simpleControlCanonical(t, raw, 0, extent, func(w *runtime.Writer) error { return controlmessages.EncodePhysicalInformationMessageContentTo(w, cv) })
			if !bytes.Equal(out.Bytes, raw) {
				t.Fatal("canonical changed normal value")
			}
			cv.Group.TIMINGADVANCEVALUE ^= 1
			edited := simpleControlCanonical(t, raw, 0, extent, func(w *runtime.Writer) error { return controlmessages.EncodePhysicalInformationMessageContentTo(w, cv) })
			er, _ := runtime.NewChoiceReaderBits(edited.Bytes, extent, nil)
			ev, e := controlmessages.DecodePhysicalInformationMessageContentFrom(er)
			if e != nil || ev.Group.TIMINGADVANCEVALUE != uint8(timing^1) {
				t.Fatalf("edited %+v: %v", ev, e)
			}
			t.Logf(`SYNTHETIC {"Clause":"11.2.34","Name":"timing%d-extent%d","Hex":"%x","Bits":%d}`, timing, extent, raw, extent)
		}
	}
}
func TestClause11234ZeroFillScope(t *testing.T) {
	raw := simpleControlBits("01001011111111", 24)
	for n := 0; n <= 8; n++ {
		end := 6 + n
		c := runtime.NewReceiverCollector()
		r, _ := runtime.NewChoiceReaderBits(raw, 24, c)
		old, e := r.PushLimit(end)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePhysicalInformationMessageContentFrom(r)
		if e != nil || v.Group.TIMINGADVANCEVALUE != uint8(255<<uint(8-n)) {
			t.Fatalf("cut %d %+v: %v", n, v, e)
		}
		report := c.TakeReceiverReport()
		fills := 0
		for _, f := range report.ZeroFill {
			fills += f.Width
			if f.Source.Clause != "11.2.34" || f.BitOffset < end {
				t.Fatalf("wrong fill scope %+v", f)
			}
		}
		if fills != 8-n {
			t.Fatalf("cut %d fills %d", n, fills)
		}
		simpleControlReplay(t, raw, end, 0, r, func(w *runtime.Writer) error { return controlmessages.EncodePhysicalInformationMessageContentTo(w, v) })
		if e = r.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		got, e := r.ReadUint(1)
		if want := uint64((raw[end/8] >> uint(7-end%8)) & 1); e != nil || got != want {
			t.Fatalf("sentinel %d want %d: %v", got, want, e)
		}
		cv := runtime.Canonical(v)
		out := simpleControlCanonical(t, raw, 0, 14, func(w *runtime.Writer) error { return controlmessages.EncodePhysicalInformationMessageContentTo(w, cv) })
		fr, _ := runtime.NewChoiceReaderBits(out.Bytes, 14, nil)
		fv, e := controlmessages.DecodePhysicalInformationMessageContentFrom(fr)
		if e != nil || fv.MESSAGETYPE != 18 || fv.Group.TIMINGADVANCEVALUE != v.Group.TIMINGADVANCEVALUE {
			t.Fatalf("canonical fill %+v: %v", fv, e)
		}
	}
	for n := 0; n < 6; n++ {
		r, _ := runtime.NewChoiceReaderBits(raw, n, nil)
		if _, e := controlmessages.DecodePhysicalInformationMessageContentFrom(r); e == nil {
			t.Fatalf("inferred selector at %d", n)
		}
	}
	invalid := append([]byte(nil), raw...)
	invalid[0] ^= 0x80
	r, _ := runtime.NewChoiceReaderBits(invalid, 14, nil)
	if _, e := controlmessages.DecodePhysicalInformationMessageContentFrom(r); e == nil {
		t.Fatal("wrong internal selector accepted")
	}
}
func FuzzClause11234PhysicalInformation(f *testing.F) {
	f.Add([]byte{0x4b, 0xfc}, uint8(14))
	f.Fuzz(func(t *testing.T, raw []byte, n uint8) {
		extent := int(n) % 185
		if len(raw) != (extent+7)/8 {
			return
		}
		r, e := runtime.NewChoiceReaderBits(raw, extent, nil)
		if e != nil {
			return
		}
		v, e := controlmessages.DecodePhysicalInformationMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, extent, 0, r, func(w *runtime.Writer) error { return controlmessages.EncodePhysicalInformationMessageContentTo(w, v) })
		}
	})
}
