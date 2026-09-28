package runtime

import (
	"bytes"
	"math"
	"testing"
)

func TestWireStateRejectsArchitectureEdgeLengths(t *testing.T) {
	edge := uint64(1 << 31)
	for _, boundary := range []int{int(^uint(0) >> 1), int(edge)} {
		wire := Seal(struct{}{}, []byte{0}, 8, BitString{}, WireInfo{})
		wire.ImplicitSpans = []ImplicitSpan{{At: boundary, Count: 1}}
		wire.ImplicitZeros = 1
		writer := NewWriter()
		writer.WithWire(wire)
		if err := writer.validateState(); err == nil {
			t.Fatalf("accepted inferred span at %d", boundary)
		}
	}
	for _, boundary := range []uint64{1 << 31, 1 << 32, math.MaxUint64} {
		for _, expr := range []string{"Count", "Count+1", "Count*2"} {
			if _, err := eval(expr, map[string]uint64{"count": boundary}); err == nil {
				t.Fatalf("accepted %s=%d", expr, boundary)
			}
		}
	}
}

func TestReaderRejectsOversizedInputBeforeBitArithmetic(t *testing.T) {
	r := NewReader(make([]byte, maxBits/8+1))
	if err := r.Check(); err == nil {
		t.Fatal("accepted oversized reader input")
	}
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("read from oversized input")
	}
	if err := (*Reader)(nil).Check(); err == nil {
		t.Fatal("accepted nil reader")
	}
	r = NewReader(nil)
	r.SetZeroExtension(true)
	r.virtual = maxBits
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("accepted bit past virtual-input bound")
	}
}

func TestReaderRejectsInvalidRestoredLimitAndTerminalStart(t *testing.T) {
	r := NewReader([]byte{0})
	if _, err := r.PushLimit(0); err != nil {
		t.Fatal(err)
	}
	if err := r.PopLimit(math.MaxInt); err == nil {
		t.Fatal("accepted restored limit beyond input")
	}
	if err := r.RecordTerminal(math.MinInt); err == nil {
		t.Fatal("accepted terminal start before input")
	}
}

func TestReaderBoundsNestedFixedValueDepth(t *testing.T) {
	r := NewReader([]byte{0})
	for i := 0; i < 128; i++ {
		if _, err := r.PushLimit(0); err != nil {
			t.Fatalf("depth %d: %v", i, err)
		}
	}
	if _, err := r.PushLimit(0); err == nil {
		t.Fatal("accepted unbounded nested fixed-value depth")
	}
}

func TestWriterRejectsOverflowedImplicitSpanBeforeWriting(t *testing.T) {
	w := NewWriter()
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	w.WithWire(WireInfo{ImplicitSpans: []ImplicitSpan{{At: 1, Count: math.MaxInt}}, sealed: true})
	if err := w.WriteUint(0, 1); err == nil {
		t.Fatal("accepted overflowing receiver-inferred span")
	}
}

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

func TestWriterFixedLimitIsRelativeToCurrentPosition(t *testing.T) {
	zero := NewWriter()
	zeroOld, err := zero.PushLimit(0)
	if err != nil || zero.RemainingLimit() != 0 {
		t.Fatalf("zero bit limit: remaining=%d err=%v", zero.RemainingLimit(), err)
	}
	if err := zero.WriteUint(1, 1); err == nil {
		t.Fatal("wrote past zero bit limit")
	}
	if err := zero.PopLimit(zeroOld); err != nil {
		t.Fatal(err)
	}
	w := NewWriter()
	if err := w.WriteUint(0x55, 8); err != nil {
		t.Fatal(err)
	}
	old, err := w.PushLimit(16)
	if err != nil || w.RemainingLimit() != 16 || w.BoundEndOr(0) != 24 {
		t.Fatalf("push fixed value: remaining=%d end=%d err=%v", w.RemainingLimit(), w.BoundEndOr(0), err)
	}
	if _, err := w.PushLimit(17); err == nil {
		t.Fatal("accepted nested limit past enclosing value")
	}
	if err := w.WriteUint(0xaa, 8); err != nil {
		t.Fatal(err)
	}
	nested, err := w.PushLimit(8)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0xbb, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(1, 1); err == nil {
		t.Fatal("wrote past fixed value")
	}
	if err := w.PopLimit(nested); err != nil {
		t.Fatal(err)
	}
	if err := w.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	if w.RemainingLimit() != -1 || w.BoundEndOr(32) != 32 {
		t.Fatal("limit not restored")
	}
	if got, err := w.Bytes(); err != nil || !bytes.Equal(got, []byte{0x55, 0xaa, 0xbb}) {
		t.Fatalf("bytes=%x err=%v", got, err)
	}
}

func TestReaderBoundedRemainingRequiresExplicitLimit(t *testing.T) {
	r := NewReader([]byte{0xaa})
	if got := r.BoundedRemaining(); got != -1 {
		t.Fatalf("unbounded remaining = %d", got)
	}
	old, err := r.PushLimit(8)
	if err != nil || r.BoundedRemaining() != 8 {
		t.Fatalf("bounded remaining = %d, %v", r.BoundedRemaining(), err)
	}
	if _, err := r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	if err := r.PopLimit(old); err != nil || r.BoundedRemaining() != -1 {
		t.Fatalf("restored bound = %d, %v", r.BoundedRemaining(), err)
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
