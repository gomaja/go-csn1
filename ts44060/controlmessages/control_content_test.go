package controlmessages_test

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

// TS 44.060 V19.0.0 §10.0b.2.1 paragraphs 7388–7389, §10.3.1.1
// figure 10.3.1.1.1 paragraphs 7665–7730, and §11.2.0.1 paragraph 13370.
// CS-1 has 184 logical bits;
// the one-octet MAC header and six-bit selector precede this content root.
func controlBits(t testing.TB, prefix string, extent int, ignoredPadding bool) []byte {
	t.Helper()
	if len(prefix) > extent {
		t.Fatal("synthetic prefix exceeds extent")
	}
	bits := prefix
	if len(bits) < extent {
		if ignoredPadding {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	for len(bits) < extent {
		if ignoredPadding {
			bits += "1"
		} else if (byte(0x2b)>>uint(7-len(bits)%8))&1 != 0 {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	out := make([]byte, (extent+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func contentReader(t testing.TB, data []byte, extent int, direction runtime.RLCMACDirection, selector uint64) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(data, extent, c)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtime.RLCMACContext{Direction: direction, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadUint(6)
	if err != nil || got != selector || r.Position() != 14 {
		t.Fatalf("selector %d at %d: %v", got, r.Position(), err)
	}
	return r, c
}

func replayControl(t testing.TB, data []byte, r *runtime.Reader, encode func(*runtime.Writer) error) {
	t.Helper()
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	w.WithWire(wire)
	// The test supplies the external framing. The generated content owns no selector.
	for _, b := range data[:1] {
		if err = w.WriteUint(uint64(b), 8); err != nil {
			t.Fatal(err)
		}
	}
	tag := uint64(data[1] >> 2)
	if err = w.WriteUint(tag, 6); err != nil {
		t.Fatal(err)
	}
	if err = encode(w); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != 184 || !bytes.Equal(out.Bytes, data) {
		t.Fatalf("replay %x/%d, want %x: %v", out.Bytes, out.BitLength, data, err)
	}
}
