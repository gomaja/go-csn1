package controlmessages_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"testing"
)

// TS 44.060 V19.0.0 §10.0b.2.1 paragraphs 7388–7389 and
// §10.3.2 figures 10.3.2.1/2 paragraphs 7887–7957: 184 bits,
// eight MAC header bits, then six selector bits (§11.2.0.2).
func uplinkControlBits(prefix string, extent int, ignore bool) []byte {
	bits := prefix
	if len(bits) < extent {
		if ignore {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	for len(bits) < extent {
		if ignore {
			bits += "1"
		} else {
			bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
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

func uplinkContentReader(t testing.TB, data []byte, selector uint64) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, e := runtime.NewChoiceReaderBits(data, 184, c)
	if e != nil {
		t.Fatal(e)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, e = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); e != nil {
		t.Fatal(e)
	}
	tag, e := r.ReadUint(6)
	if e != nil || tag != selector || r.Position() != 14 {
		t.Fatalf("selector %d origin %d: %v", tag, r.Position(), e)
	}
	return r, c
}

func encodeUplinkContent(t testing.TB, data []byte, extent int, wire runtime.WireInfo, encode func(*runtime.Writer) error) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(uint64(data[0]), 8); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(uint64(data[1]>>2), 6); e != nil {
		t.Fatal(e)
	}
	if e := encode(w); e != nil {
		t.Fatal(e)
	}
	out, e := w.FinishBits(runtime.BitString{})
	if e != nil || out.BitLength != extent {
		t.Fatalf("extent %d: %v", out.BitLength, e)
	}
	return out.Bytes
}

func uplinkReplay(t testing.TB, data []byte, r *runtime.Reader, encode func(*runtime.Writer) error) {
	t.Helper()
	wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if e != nil {
		t.Fatal(e)
	}
	if got := encodeUplinkContent(t, data, 184, wire, encode); !bytes.Equal(got, data) {
		t.Fatalf("replay %x want %x", got, data)
	}
}
