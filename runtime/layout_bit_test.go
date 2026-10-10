package runtime

import (
	"bytes"
	"fmt"
	"testing"
)

func TestLocalLayoutBitDoesNotEnterRootSpare(t *testing.T) {
	r := NewReader([]byte{0x80})
	bit, err := r.ReadLayoutBit()
	if err != nil || bit.BitLength != 1 || !bytes.Equal(bit.Bytes, []byte{0x80}) {
		t.Fatalf("bit %+v: %v", bit, err)
	}
	if len(r.Wire().Spare) != 0 {
		t.Fatal("local bit leaked to root")
	}
	w := NewWriter()
	if err := w.WriteLayoutBit(bit); err != nil {
		t.Fatal(err)
	}
	out, err := w.Finish(BitString{})
	if err != nil || !bytes.Equal(out, []byte{0x80}) {
		t.Fatalf("plain %x: %v", out, err)
	}
	if err := NewWriter().WriteLayoutBit(BitString{BitLength: 2, Bytes: []byte{0}}); err == nil {
		t.Fatal("accepted invalid layout width")
	}
	r = NewReader(nil)
	r.SetZeroExtension(true)
	bit, err = r.ReadLayoutBit()
	if err != nil || bit.BitLength != 0 || r.Wire().ImplicitZeros != 1 {
		t.Fatalf("inferred %+v: %v", bit, err)
	}
}

func TestInferredLayoutBitUsesContainingRootSpans(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		spans                 []ImplicitSpan
		prefix                int
		wantBits, wantVirtual int
	}{
		{name: "transmitted", wantBits: 1},
		{name: "before span", spans: []ImplicitSpan{{At: 2, Count: 1}}, wantBits: 1},
		{name: "in span", spans: []ImplicitSpan{{At: 0, Count: 1}}, wantVirtual: 1},
		{name: "after span", spans: []ImplicitSpan{{At: 0, Count: 1}}, prefix: 1, wantBits: 1, wantVirtual: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWriter()
			w.WithWire(WireInfo{ImplicitSpans: tc.spans, sealed: true})
			if err := w.WriteUint(0, tc.prefix); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteLayoutBit(BitString{}); err != nil {
				t.Fatal(err)
			}
			if w.bits != tc.wantBits || w.virtual != tc.wantVirtual {
				t.Fatalf("bits %d+%d, want %d+%d", w.bits, w.virtual, tc.wantBits, tc.wantVirtual)
			}
		})
	}
	for i, record := range []BitString{{BitLength: -1}, {BitLength: 2, Bytes: []byte{0}}, {BitLength: 0, Bytes: []byte{0}}, {BitLength: 1}} {
		t.Run(fmt.Sprintf("invalid %d", i), func(t *testing.T) {
			w := NewWriter()
			w.WithWire(WireInfo{sealed: true})
			if err := w.WriteLayoutBit(record); err == nil {
				t.Fatal("accepted invalid layout record")
			}
		})
	}
	w := NewWriter()
	w.WithWire(WireInfo{ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}, sealed: true})
	if err := w.WriteLayoutBit(BitString{BitLength: 1, Bytes: []byte{0x80}}); err == nil {
		t.Fatal("accepted received one at inferred position")
	}
}
