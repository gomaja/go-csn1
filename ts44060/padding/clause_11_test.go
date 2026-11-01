package padding_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/padding"
)

// TS 44.060 V19.0.0 §11 paragraphs 12937–12940: empty padding or a
// leading zero followed by positional spare padding. The error arm is retained.
func TestClause11PaddingPhases(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		widths := []int{0, 1, 9, 31, 257}
		if phase != 7 {
			widths = append(widths, 8-phase)
		}
		for _, width := range widths {
			t.Run(fmt.Sprintf("phase%d/bits%d", phase, width), func(t *testing.T) {
				extent := phase + width
				data := make([]byte, (extent+7)/8)
				for i := phase + 1; i < extent; i++ {
					if (byte(0x2b)>>uint(7-i%8))&1 != 0 {
						data[i/8] |= 1 << uint(7-i%8)
					}
				}
				c := runtime.NewReceiverCollector()
				r, err := runtime.NewChoiceReaderBits(data, extent, c)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				v, err := padding.DecodePaddingBitsFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				if r.Position() != extent {
					t.Fatalf("end %d, want %d", r.Position(), extent)
				}
				if width == 0 {
					if v.IgnoreChoice.AltUnlabeled == nil {
						t.Fatal("null arm absent")
					}
				} else if v.IgnoreChoice.Ignore == nil || v.IgnoreChoice.Ignore.Known == nil {
					t.Fatal("valid padding lost its known arm")
				}
				if report := c.TakeReceiverReport(); len(report.Regions) != 0 || len(report.ZeroFill) != 0 {
					t.Fatalf("unexpected receiver report %+v", report)
				}
				wire, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: extent}, extent, runtime.BitString{}, r.Wire())
				if err != nil {
					t.Fatal(err)
				}
				w := runtime.NewWriter()
				w.WithWire(wire)
				if err = w.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				if err = padding.EncodePaddingBitsTo(w, v); err != nil {
					t.Fatal(err)
				}
				out, err := w.FinishBits(runtime.BitString{})
				if err != nil || out.BitLength != extent || !bytes.Equal(out.Bytes, data) {
					t.Fatalf("replay %x/%d: %v", out.Bytes, out.BitLength, err)
				}
				for _, canonical := range []bool{false, true} {
					if !canonical && width > 0 && extent%8 != 0 {
						continue
					}
					fresh := runtime.Canonical(v)
					fw := runtime.NewWriter()
					if canonical {
						fw.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
					}
					if _, err = fw.PushLimit(extent); err != nil {
						t.Fatal(err)
					}
					if err = fw.WriteUint(0, phase); err != nil {
						t.Fatal(err)
					}
					if err = padding.EncodePaddingBitsTo(fw, fresh); err != nil {
						t.Fatal(err)
					}
					// Fresh positional padding completes the current octet; canonical fills
					// the caller's exact extent (TS 24.007 V20.0.0 Annex B §B.1.2.2).
					out, err = fw.FinishBits(runtime.BitString{})
					if err != nil {
						t.Fatal(err)
					}
					if !canonical {
						want := phase
						if width > 0 {
							want = ((phase + 1 + 7) / 8) * 8
						}
						if out.BitLength != want {
							t.Fatalf("fresh end %d, want %d", out.BitLength, want)
						}
						for i := 0; i < out.BitLength; i++ {
							if (out.Bytes[i/8]>>uint(7-i%8))&1 != (data[i/8]>>uint(7-i%8))&1 {
								t.Fatalf("fresh bit %d differs from positional padding", i)
							}
						}
					}
					if canonical && (out.BitLength != extent || !bytes.Equal(out.Bytes, data)) {
						t.Fatalf("canonical %x/%d, want %x/%d", out.Bytes, out.BitLength, data, extent)
					}
				}
			})
		}
	}
}

func TestClause11IgnoreBoundAndReplay(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		for _, width := range []int{1, 7, 65, 1025, 16385, 65537} {
			extent := phase + width
			data := make([]byte, (extent+3+7)/8)
			for i := phase; i < extent; i++ {
				data[i/8] |= 1 << uint(7-i%8)
			}
			// A three-bit sentinel is outside the explicit padding bound.
			for i := extent; i < extent+3; i++ {
				data[i/8] |= 1 << uint(7-i%8)
			}
			c := runtime.NewReceiverCollector()
			r, err := runtime.NewChoiceReaderBits(data, extent+3, c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(width)
			if err != nil {
				t.Fatal(err)
			}
			v, err := padding.DecodePaddingBitsFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			arm := v.IgnoreChoice.Ignore
			if arm == nil || arm.Ignored == nil || arm.Ignored.NoString.Label != runtime.Ignore || arm.Ignored.NoString.Bits.BitLength != width-1 {
				t.Fatalf("lost Ignore: %+v", v)
			}
			if got := len(arm.Ignored.NoString.Bits.Bytes); got != (width-1+7)/8 {
				t.Fatalf("retained region bytes %d, want linear packed extent %d", got, (width-1+7)/8)
			}
			report := c.TakeReceiverReport()
			if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore || report.Regions[0].Location.BitOffset != phase+1 || report.Regions[0].Location.Width != width-1 {
				t.Fatalf("Ignore report %+v", report)
			}
			wire, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: extent}, extent, runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			if err = r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(3)
			if err != nil || sentinel != 7 {
				t.Fatalf("sentinel %d %v", sentinel, err)
			}
			w := runtime.NewWriter()
			w.WithWire(wire)
			if err = w.WriteUint(0, phase); err != nil {
				t.Fatal(err)
			}
			if err = padding.EncodePaddingBitsTo(w, v); err != nil {
				t.Fatal(err)
			}
			out, err := w.FinishBits(runtime.BitString{})
			if err != nil || out.BitLength != extent {
				t.Fatalf("replay length %d %v", out.BitLength, err)
			}
			for i := 0; i < extent; i++ {
				if (out.Bytes[i/8]>>uint(7-i%8))&1 != (data[i/8]>>uint(7-i%8))&1 {
					t.Fatalf("replay bit %d", i)
				}
			}
			cw := runtime.NewWriter()
			cw.WithWire(runtime.CanonicalTo(v, extent).Wire)
			if err = cw.WriteUint(0, phase); err != nil {
				t.Fatal(err)
			}
			if err = padding.EncodePaddingBitsTo(cw, runtime.Canonical(v)); err == nil || !strings.Contains(err.Error(), "receiver error region has no canonical sender encoding") {
				t.Fatalf("canonical Ignore: %v", err)
			}
			if err = padding.EncodePaddingBitsTo(runtime.NewWriter(), v); err == nil || !strings.Contains(err.Error(), "receiver error region requires received wire") {
				t.Fatalf("fresh Ignore: %v", err)
			}
			wrongLabel := v
			saved := arm.Ignored.NoString
			wrongLabel.IgnoreChoice.Ignore.Ignored.NoString.Label = runtime.MessageEscape
			lw := runtime.NewWriter()
			lw.WithWire(wire)
			if err = lw.WriteUint(0, phase); err != nil {
				t.Fatal(err)
			}
			if err = padding.EncodePaddingBitsTo(lw, wrongLabel); err == nil || !strings.Contains(err.Error(), "receiver error label mismatch") {
				t.Fatalf("wrong label: %v", err)
			}
			arm.Ignored.NoString = saved
			if width > 1 {
				arm.Ignored.NoString.Bits.BitLength--
				arm.Ignored.NoString.Location.Width--
				short := runtime.NewWriter()
				short.WithWire(wire)
				if _, err = short.PushLimit(extent); err != nil {
					t.Fatal(err)
				}
				if err = short.WriteUint(0, phase); err != nil {
					t.Fatal(err)
				}
				if err = padding.EncodePaddingBitsTo(short, v); err == nil || !strings.Contains(err.Error(), "receiver error region must fill its containing extent") {
					t.Fatalf("short region: %v", err)
				}
				arm.Ignored.NoString = saved
			}
			arm.Known = &padding.PaddingBitsIgnoreChoiceIgnoreKnown{}
			if err = padding.EncodePaddingBitsTo(runtime.NewWriter(), v); err == nil || !strings.Contains(err.Error(), "receiver fallback requires exactly one arm") {
				t.Fatalf("multiple receiver arms: %v", err)
			}
			arm.Known = nil
		}
	}
}

func FuzzClause11Padding(f *testing.F) {
	f.Add([]byte{0x2b, 0x2b}, uint8(3), uint8(9))
	f.Add([]byte{0xff}, uint8(0), uint8(8))
	f.Fuzz(func(t *testing.T, data []byte, phaseByte, widthByte uint8) {
		if len(data) > 4096 {
			return
		}
		phase := int(phaseByte % 8)
		extent := min(len(data)*8, int(widthByte)+phase)
		if phase > extent {
			return
		}
		r, err := runtime.NewChoiceReaderBits(data, extent, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		_, _ = padding.DecodePaddingBitsFrom(r)
		if r.Position() > extent {
			t.Fatalf("read beyond bound %d > %d", r.Position(), extent)
		}
	})
}
