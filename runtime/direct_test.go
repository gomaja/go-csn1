package runtime

import (
	"bytes"
	"testing"
)

func TestDirectBitIOBoundsAndPaddingOrigin(t *testing.T) {
	if LHBit('L', 0) != 0 || LHBit('H', 0) != 1 || LHBit('L', 2) != 1 || LHBit('H', 2) != 0 {
		t.Fatal("L/H constraint must follow the absolute padding-pattern position")
	}
	r := NewReader([]byte{0x2b})
	for _, symbol := range []string{"L", "L", "L", "L", "L", "L", "L", "L"} {
		if err := r.Expect(symbol); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("read beyond input")
	}
	w := NewWriter()
	if err := w.WriteLiteral("LLLLLLLL"); err != nil {
		t.Fatal(err)
	}
	if got, err := w.Bytes(); err != nil || !bytes.Equal(got, []byte{0x2b}) {
		t.Fatalf("bytes=%x err=%v", got, err)
	}
}

func TestDirectReaderChoiceSnapshot(t *testing.T) {
	r := NewReader([]byte{0x80})
	a := r.Fork()
	if err := a.Expect("0"); err == nil {
		t.Fatal("wrong branch accepted")
	}
	b := r.Fork()
	if err := b.Expect("1"); err != nil {
		t.Fatal(err)
	}
	r.Commit(b)
	if r.Position() != 1 {
		t.Fatalf("position=%d", r.Position())
	}
}

func TestSealRecordsTransmittedBoundary(t *testing.T) {
	wire := Seal(struct{}{}, []byte{0x30}, 4, BitString{Bytes: []byte{0}, BitLength: 4}, WireInfo{})
	if wire.BitsConsumed != 4 || wire.TransmittedBits != 8 || wire.Tail.BitLength != 4 {
		t.Fatalf("wire boundary: %+v", wire)
	}
}

func TestWireStateCannotBeSilentlyDropped(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*WireInfo)
	}{
		{"spare", func(w *WireInfo) { w.Spare = append(w.Spare, BitString{Bytes: []byte{0}, BitLength: 1}) }},
		{"padding", func(w *WireInfo) { w.Padding = append(w.Padding, BitString{Bytes: []byte{0}, BitLength: 1}) }},
		{"terminal", func(w *WireInfo) { w.Terminal = append(w.Terminal, BitString{Bytes: []byte{0}, BitLength: 8}) }},
		{"truncation", func(w *WireInfo) { w.TruncatedAt = map[string]int{"missing": 1} }},
		{"spare count", func(w *WireInfo) { w.SpareCounts = map[string][]int{"missing": {1}} }},
		{"consumed", func(w *WireInfo) { w.BitsConsumed = 7 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire := Seal(struct{}{}, []byte{0x00}, 8, BitString{}, WireInfo{})
			tc.mutate(&wire)
			writer := NewWriter()
			writer.WithWire(wire)
			if err := writer.WriteUint(0, 8); err != nil {
				t.Fatal(err)
			}
			if out, err := writer.Finish(wire.Tail); err == nil {
				t.Fatalf("dropped inconsistent %s state: %x", tc.name, out)
			}
		})
	}
}
