package containers_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/containers"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §12.47 paragraphs 24514–24517. A shorter
// payload than its printed octet count is truncated, never zero-filled.
func TestClause1247Container(t *testing.T) {
	for origin := 0; origin < 8; origin++ {
		for _, n := range []int{0, 1, 2, 127 / 2, 127} {
			for _, extra := range []int{0, 1, 17} {
				t.Run(fmt.Sprintf("phase%d/length%d/padding%d", origin, n, extra), func(t *testing.T) {
					payload := make([]byte, n)
					bits := strings.Repeat("0", origin) + fmt.Sprintf("%07b", n)
					for i := range payload {
						payload[i] = byte(i*37 + 0xa5)
						bits += fmt.Sprintf("%08b", payload[i])
					}
					extent := len(bits) + extra
					raw := containerBits(bits, extent)
					// Three bits after the containing extent must survive the component.
					full := packedBits(fmtBits(raw, extent) + "101")
					c := runtime.NewReceiverCollector()
					rd, err := runtime.NewChoiceReaderBits(full, extent+3, c)
					if err != nil {
						t.Fatal(err)
					}
					if _, err = rd.ReadUint(origin); err != nil {
						t.Fatal(err)
					}
					old, err := rd.PushLimit(extent - origin)
					if err != nil {
						t.Fatal(err)
					}
					v, err := containers.DecodeCSHandoverRadioResourcesIEFrom(rd)
					if err != nil || int(v.CSHANDOVERRADIORESOURCESLENGTH) != n || v.CSHANDOVERRADIORESOURCESDATA.BitLength != n*8 || !bytes.Equal(v.CSHANDOVERRADIORESOURCESDATA.Bytes, payload) || rd.Position() != extent {
						t.Fatalf("typed length/payload at %d: %+v %v", rd.Position(), v, err)
					}
					if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
						t.Fatalf("regions %+v", report)
					}
					wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, rd.Wire())
					if err != nil {
						t.Fatal(err)
					}
					got := containerWriter(t, extent, origin, wire, func(w *runtime.Writer) error { return containers.EncodeCSHandoverRadioResourcesIETo(w, v) })
					if !bytes.Equal(got, raw) {
						t.Fatalf("replay %x want %x", got, raw)
					}
					canonical := runtime.Canonical(v)
					cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire
					got = containerWriter(t, extent, origin, cw, func(w *runtime.Writer) error { return containers.EncodeCSHandoverRadioResourcesIETo(w, canonical) })
					if !bytes.Equal(got, raw) {
						t.Fatalf("canonical %x want %x", got, raw)
					}
					if n > 0 {
						canonical.CSHANDOVERRADIORESOURCESDATA.Bytes = bytes.Clone(canonical.CSHANDOVERRADIORESOURCESDATA.Bytes)
						canonical.CSHANDOVERRADIORESOURCESDATA.Bytes[0] ^= 0x80
						edited := containerWriter(t, extent, origin, cw, func(w *runtime.Writer) error { return containers.EncodeCSHandoverRadioResourcesIETo(w, canonical) })
						er, _ := runtime.NewChoiceReaderBits(edited, extent, nil)
						if _, err = er.ReadUint(origin); err != nil {
							t.Fatal(err)
						}
						ev, e := containers.DecodeCSHandoverRadioResourcesIEFrom(er)
						if e != nil || ev.CSHANDOVERRADIORESOURCESDATA.Bytes[0] != payload[0]^0x80 || !bytes.Equal(ev.CSHANDOVERRADIORESOURCESDATA.Bytes[1:], payload[1:]) {
							t.Fatalf("edit %+v %v", ev, e)
						}
					}
					if err = rd.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					sentinel, err := rd.ReadUint(3)
					if err != nil || sentinel != 5 {
						t.Fatalf("sentinel %d %v", sentinel, err)
					}
					if n > 0 {
						tr, _ := runtime.NewChoiceReaderBits(raw, len(bits)-1, nil)
						if _, err = tr.ReadUint(origin); err != nil {
							t.Fatal(err)
						}
						if _, err = containers.DecodeCSHandoverRadioResourcesIEFrom(tr); err == nil {
							t.Fatal("truncated octet accepted")
						}
					}
				})
			}
		}
	}
}

func FuzzClause1247Container(f *testing.F) {
	f.Add([]byte{0x03, 0x4b, 0x2b}, uint8(0))
	f.Add([]byte{0xff, 0xff}, uint8(7))
	f.Fuzz(func(t *testing.T, raw []byte, phase uint8) {
		if len(raw) > 4096 {
			return
		}
		origin := int(phase % 8)
		if len(raw)*8 < origin {
			return
		}
		rd, _ := runtime.NewChoiceReaderBits(raw, len(raw)*8, nil)
		if _, err := rd.ReadUint(origin); err != nil {
			t.Fatal(err)
		}
		v, err := containers.DecodeCSHandoverRadioResourcesIEFrom(rd)
		if err != nil {
			return
		}
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: len(raw) * 8}, len(raw)*8, runtime.BitString{}, rd.Wire())
		if err != nil {
			t.Fatal(err)
		}
		// Fuzzed origins need the exact received prefix, unlike the zero prefix
		// used by the clause-derived phase vectors.
		w := runtime.NewWriter()
		w.WithWire(wire)
		prefix, _ := runtime.NewReader(raw).ReadUint(origin)
		if err = w.WriteUint(prefix, origin); err != nil {
			t.Fatal(err)
		}
		if err = containers.EncodeCSHandoverRadioResourcesIETo(w, v); err != nil {
			t.Fatal(err)
		}
		got, err := w.FinishBits(runtime.BitString{})
		if err != nil || !bytes.Equal(got.Bytes, raw) {
			t.Fatalf("replay %x %v", got.Bytes, err)
		}
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938: the receiver Ignore arm
// retains the containing remainder; the foreign payload stays at its own size.
func TestClause1247IgnoredGrowingExtent(t *testing.T) {
	for _, extent := range []int{64, 256, 4096, 65537} {
		prefix := fmt.Sprintf("%07b", 1) + "10100101"
		raw := packedBits(prefix + strings.Repeat("1", extent-len(prefix)))
		c := runtime.NewReceiverCollector()
		rd, e := runtime.NewChoiceReaderBits(raw, extent, c)
		if e != nil {
			t.Fatal(e)
		}
		v, e := containers.DecodeCSHandoverRadioResourcesIEFrom(rd)
		if e != nil || v.CSHANDOVERRADIORESOURCESLENGTH != 1 || v.CSHANDOVERRADIORESOURCESDATA.BitLength != 8 || !bytes.Equal(v.CSHANDOVERRADIORESOURCESDATA.Bytes, []byte{0xa5}) || rd.Position() != extent {
			t.Fatalf("typed payload %+v %v", v, e)
		}
		report := c.TakeReceiverReport()
		if len(report.Regions) != 1 {
			t.Fatalf("regions %+v", report)
		}
		region := v.PaddingBits.IgnoreChoice.Ignore.Ignored.NoString
		if region.Label != runtime.Ignore || region.Location.BitOffset != len(prefix)+1 || region.Location.Width != extent-len(prefix)-1 || region.Bits.BitLength != extent-len(prefix)-1 || len(region.Bits.Bytes) != (extent-len(prefix)-1+7)/8 {
			t.Fatalf("retained Ignore extent %+v", region)
		}
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, rd.Wire())
		if e != nil {
			t.Fatal(e)
		}
		if got := containerWriter(t, extent, 0, wire, func(w *runtime.Writer) error { return containers.EncodeCSHandoverRadioResourcesIETo(w, v) }); !bytes.Equal(got, raw) {
			t.Fatalf("replay %x", got)
		}
		if _, e = containers.EncodeCSHandoverRadioResourcesIECanonicalAtLength(v, (extent+7)/8); e == nil {
			t.Fatal("receiver Ignore accepted as canonical sender")
		}
	}
}

// TS 44.060 V19.0.0 §11.2.46 paragraphs 20817–20836 and
// §12.47 paragraphs 24514–24517: CS data starts at bit 32 after
// the enclosing header/type/page/address/TFI/payload tag and seven-bit count.
func TestClause1247WholeControlBlockBounds(t *testing.T) {
	for _, tc := range []struct {
		wire, payload string
		end           int
	}{
		{"405c2a000000000000000000000000000000002b2b2b2b", "", 32},
		{"405c2a01a50000000000000000000000000000002b2b2b", "a5", 40},
		{"405c2a02a55a0000000000000000000000000000002b2b", "a55a", 48},
	} {
		raw, e := hex.DecodeString(tc.wire)
		if e != nil {
			t.Fatal(e)
		}
		payload, e := hex.DecodeString(tc.payload)
		if e != nil {
			t.Fatal(e)
		}
		rd := containerBlockReader(t, raw, 0x17)
		if _, e = rd.ReadUint(11); e != nil {
			t.Fatal(e)
		}
		old, e := rd.PushLimit(tc.end - 25)
		if e != nil {
			t.Fatal(e)
		}
		v, e := containers.DecodeCSHandoverRadioResourcesIEFrom(rd)
		if e != nil || int(v.CSHANDOVERRADIORESOURCESLENGTH) != len(payload) || !bytes.Equal(v.CSHANDOVERRADIORESOURCESDATA.Bytes, payload) || rd.Position() != tc.end {
			t.Fatalf("bounded CS payload %+v at %d: %v", v, rd.Position(), e)
		}
		if e = rd.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		tag, e := rd.ReadUint(2)
		if e != nil || tag != 0 {
			t.Fatalf("following PS selector %d %v", tag, e)
		}
	}
}
