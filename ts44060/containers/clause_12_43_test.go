package containers_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/containers"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"strings"
	"testing"
)

func packedBits(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

// TS 44.060 V19.0.0 §11 paragraphs 12937–12940: zero then spare
// padding, whose phase is measured from the original octet boundary.
func containerBits(prefix string, extent int) []byte {
	bits := prefix
	if len(bits) < extent {
		bits += "0"
	}
	for len(bits) < extent {
		bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
	}
	return packedBits(bits)
}

func containerWriter(t testing.TB, extent, origin int, wire runtime.WireInfo, encode func(*runtime.Writer) error) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, err := w.PushLimit(extent); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, origin); err != nil {
		t.Fatal(err)
	}
	if err := encode(w); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != extent {
		t.Fatalf("encoded extent %d: %v", out.BitLength, err)
	}
	return out.Bytes
}

func fmtBits(data []byte, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte('0' + ((data[i/8] >> uint(7-i%8)) & 1))
	}
	return b.String()
}

// TS 44.060 V19.0.0 §12.43 paragraphs 24238–24241. A shorter
// payload than its printed octet count is truncated, never zero-filled.
func TestClause1243Container(t *testing.T) {
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
					v, err := containers.DecodeNASContainerForPSHandoverIEFrom(rd)
					if err != nil || int(v.NASCONTAINERLENGTH) != n || v.NASCONTAINERDATA.BitLength != n*8 || !bytes.Equal(v.NASCONTAINERDATA.Bytes, payload) || rd.Position() != extent {
						t.Fatalf("typed length/payload at %d: %+v %v", rd.Position(), v, err)
					}
					if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
						t.Fatalf("regions %+v", report)
					}
					wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, rd.Wire())
					if err != nil {
						t.Fatal(err)
					}
					got := containerWriter(t, extent, origin, wire, func(w *runtime.Writer) error { return containers.EncodeNASContainerForPSHandoverIETo(w, v) })
					if !bytes.Equal(got, raw) {
						t.Fatalf("replay %x want %x", got, raw)
					}
					canonical := runtime.Canonical(v)
					cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire
					got = containerWriter(t, extent, origin, cw, func(w *runtime.Writer) error { return containers.EncodeNASContainerForPSHandoverIETo(w, canonical) })
					if !bytes.Equal(got, raw) {
						t.Fatalf("canonical %x want %x", got, raw)
					}
					if n > 0 {
						canonical.NASCONTAINERDATA.Bytes = bytes.Clone(canonical.NASCONTAINERDATA.Bytes)
						canonical.NASCONTAINERDATA.Bytes[0] ^= 0x80
						edited := containerWriter(t, extent, origin, cw, func(w *runtime.Writer) error { return containers.EncodeNASContainerForPSHandoverIETo(w, canonical) })
						er, _ := runtime.NewChoiceReaderBits(edited, extent, nil)
						if _, err = er.ReadUint(origin); err != nil {
							t.Fatal(err)
						}
						ev, e := containers.DecodeNASContainerForPSHandoverIEFrom(er)
						if e != nil || ev.NASCONTAINERDATA.Bytes[0] != payload[0]^0x80 || !bytes.Equal(ev.NASCONTAINERDATA.Bytes[1:], payload[1:]) {
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
						if _, err = containers.DecodeNASContainerForPSHandoverIEFrom(tr); err == nil {
							t.Fatal("truncated octet accepted")
						}
					}
				})
			}
		}
	}
}

func FuzzClause1243Container(f *testing.F) {
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
		v, err := containers.DecodeNASContainerForPSHandoverIEFrom(rd)
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
		if err = containers.EncodeNASContainerForPSHandoverIETo(w, v); err != nil {
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
func TestClause1243IgnoredGrowingExtent(t *testing.T) {
	for _, extent := range []int{64, 256, 4096, 65537} {
		prefix := fmt.Sprintf("%07b", 1) + "10100101"
		raw := packedBits(prefix + strings.Repeat("1", extent-len(prefix)))
		c := runtime.NewReceiverCollector()
		rd, e := runtime.NewChoiceReaderBits(raw, extent, c)
		if e != nil {
			t.Fatal(e)
		}
		v, e := containers.DecodeNASContainerForPSHandoverIEFrom(rd)
		if e != nil || v.NASCONTAINERLENGTH != 1 || v.NASCONTAINERDATA.BitLength != 8 || !bytes.Equal(v.NASCONTAINERDATA.Bytes, []byte{0xa5}) || rd.Position() != extent {
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
		if got := containerWriter(t, extent, 0, wire, func(w *runtime.Writer) error { return containers.EncodeNASContainerForPSHandoverIETo(w, v) }); !bytes.Equal(got, raw) {
			t.Fatalf("replay %x", got)
		}
		if _, e = containers.EncodeNASContainerForPSHandoverIECanonicalAtLength(v, (extent+7)/8); e == nil {
			t.Fatal("receiver Ignore accepted as canonical sender")
		}
	}
}

// TS 44.060 V19.0.0 §11.2.43 paragraphs 20659–20674:
// complete PS Handover control blocks carry this exact component. The
// selector/header remain external to the component.
func TestClause1243WholeControlBlocks(t *testing.T) {
	for _, tc := range []struct {
		wire, payload string
		origin        int
	}{
		{"40542a0000000000000206952b2b2b2b2b2b2b2b2b2b2b", "a5", 79},
		{"40542a000000000000020a95692b2b2b2b2b2b2b2b2b2b", "a55a", 79},
		{"40542a00000000000002280004080c1014181c20252b2b", "00010203040506070809", 79},
	} {
		raw, e := hex.DecodeString(tc.wire)
		if e != nil {
			t.Fatal(e)
		}
		payload, e := hex.DecodeString(tc.payload)
		if e != nil {
			t.Fatal(e)
		}
		rd := containerBlockReader(t, raw, 0x15)
		// Preserve the original absolute origin, including every containing field.
		if _, e = rd.ReadBitString(tc.origin - 14); e != nil {
			t.Fatal(e)
		}
		v, e := containers.DecodeNASContainerForPSHandoverIEFrom(rd)
		if e != nil || int(v.NASCONTAINERLENGTH) != len(payload) || !bytes.Equal(v.NASCONTAINERDATA.Bytes, payload) || rd.Position() != 184 {
			t.Fatalf("whole-block component %+v %v", v, e)
		}
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, rd.Wire())
		if e != nil {
			t.Fatal(e)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		prefix := runtime.NewReader(raw)
		bits, e := prefix.ReadBitString(tc.origin)
		if e != nil {
			t.Fatal(e)
		}
		if e = w.WriteBitString(bits, tc.origin); e != nil {
			t.Fatal(e)
		}
		if e = containers.EncodeNASContainerForPSHandoverIETo(w, v); e != nil {
			t.Fatal(e)
		}
		got, e := w.FinishBits(runtime.BitString{})
		if e != nil || !bytes.Equal(got.Bytes, raw) || got.BitLength != 184 {
			t.Fatalf("whole-block replay %x %v", got.Bytes, e)
		}
	}
}

// TS 44.060 V19.0.0 §10.3.1.1 figure 10.3.1.1.1; the six-bit
// selectors are §11.2.0.1 paragraphs 13367 and 13369.
func containerBlockReader(t testing.TB, raw []byte, selector uint64) *runtime.Reader {
	t.Helper()
	rd, e := runtime.NewChoiceReaderBits(raw, 184, nil)
	if e != nil {
		t.Fatal(e)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, e = controlblock.DecodeBlockHeaderFrom(rd, ctx, controlblock.CS1); e != nil {
		t.Fatal(e)
	}
	tag, e := rd.ReadUint(6)
	if e != nil || tag != selector || rd.Position() != 14 {
		t.Fatalf("selector %d at %d %v", tag, rd.Position(), e)
	}
	return rd
}
