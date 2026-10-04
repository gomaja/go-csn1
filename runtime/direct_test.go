package runtime

import (
	"bytes"
	"errors"
	"math"
	"strings"
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

func TestReaderWorkCounterRejectsHostBoundaryBeforeIncrement(t *testing.T) {
	for _, steps := range []int{maxBits, math.MaxInt} {
		r := NewReader([]byte{0})
		*r.steps = steps
		if err := r.Enter("boundary"); err == nil {
			t.Fatalf("accepted work counter %d", steps)
		}
		if *r.steps != steps {
			t.Fatalf("work counter changed from %d to %d", steps, *r.steps)
		}
	}
	if err := (&Reader{}).Enter("uninitialized"); err == nil {
		t.Fatal("accepted uninitialized reader")
	}
}

func TestWriterRejectsHostBoundaryBeforeCounterArithmetic(t *testing.T) {
	for _, tc := range []struct{ bits, virtual int }{
		{math.MaxInt, 1},
		{1, math.MaxInt},
		{maxBits, 0},
		{-1, 0},
		{0, -1},
	} {
		w := NewWriter()
		w.bits, w.virtual = tc.bits, tc.virtual
		if err := w.put(0); err == nil {
			t.Fatalf("accepted counters bits=%d virtual=%d", tc.bits, tc.virtual)
		}
	}
	for _, bits := range []int{-1, math.MinInt, math.MaxInt} {
		w := NewWriter()
		w.bits = bits
		if _, err := w.PushLimit(0); err == nil {
			t.Fatalf("accepted limit at bit position %d", bits)
		}
	}
}

func TestBitsAtRejectsHostBoundaryRanges(t *testing.T) {
	for _, tc := range []struct{ start, count int }{
		{-1, 1}, {0, -1}, {0, math.MaxInt}, {math.MaxInt, 1},
		{math.MaxInt - 1, math.MaxInt}, {8, 1},
	} {
		if got := bitsAt([]byte{0xaa}, tc.start, tc.count); got.BitLength != 0 || len(got.Bytes) != 0 {
			t.Fatalf("accepted range start=%d count=%d: %+v", tc.start, tc.count, got)
		}
	}
}

func TestWriterRejectsOutOfRangeSpareRepeatCounts(t *testing.T) {
	for _, count := range []int{-1, maxBits + 1, math.MaxInt} {
		w := NewWriter()
		w.WithWire(WireInfo{SpareCounts: map[string][]int{"bits": {count}}})
		if got, ok := w.SpareCount("bits"); ok || got != 0 {
			t.Fatalf("accepted spare count %d as %d, %t", count, got, ok)
		}
		if _, err := w.Bytes(); err == nil {
			t.Fatalf("silently ignored spare count %d", count)
		}
	}
}

func TestReaderRejectsOverflowingImplicitSpanBeforeAppend(t *testing.T) {
	r := NewReader(nil)
	r.SetZeroExtension(true)
	r.wire.ImplicitSpans = []ImplicitSpan{{At: math.MaxInt, Count: 1}}
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("accepted an overflowing prior implicit span")
	}
}

func TestReaderRejectsImplicitSpanAtBitBudget(t *testing.T) {
	r := NewReader(nil)
	r.SetZeroExtension(true)
	r.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: maxBits}}
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("accepted an inferred span that cannot grow")
	}
}

func TestPublicReaderStateQueriesRejectUninitializedReader(t *testing.T) {
	r := &Reader{}
	if got := r.Remaining(); got != -1 {
		t.Fatalf("uninitialized remaining=%d", got)
	}
	if r.Matches("") {
		t.Fatal("uninitialized reader matched empty pattern")
	}
	if err := r.Expect(""); err == nil {
		t.Fatal("uninitialized reader accepted empty literal")
	}
	if _, err := r.PushLimit(0); err == nil {
		t.Fatal("uninitialized reader accepted fixed limit")
	}
}

func TestReaderCheckRejectsCombinedBitBudget(t *testing.T) {
	r := NewReader(make([]byte, maxBits/8))
	r.pos = maxBits
	r.virtual = 1
	if err := r.Check(); err == nil {
		t.Fatal("accepted combined transmitted and virtual bits beyond limit")
	}
}

func TestWriterRemainingLimitRejectsInvalidState(t *testing.T) {
	w := NewWriter()
	w.bounded, w.limit, w.bits = true, 0, 2
	if got := w.RemainingLimit(); got != -1 {
		t.Fatalf("invalid remaining limit=%d", got)
	}
	for _, limit := range []int{-1, math.MaxInt} {
		w := NewWriter()
		w.bounded, w.limit = true, limit
		if _, err := w.PushLimit(0); err == nil {
			t.Fatalf("accepted invalid enclosing limit %d", limit)
		}
	}
}

func TestWriterLogicalPositionChecksHostBoundary(t *testing.T) {
	for _, tc := range []struct{ bits, virtual int }{{math.MaxInt, 1}, {1, math.MaxInt}, {-1, 0}, {maxBits, 1}} {
		w := NewWriter()
		w.bits, w.virtual = tc.bits, tc.virtual
		if _, err := w.logicalPosition(); err == nil {
			t.Fatalf("accepted logical position %d+%d", tc.bits, tc.virtual)
		}
	}
	w := NewWriter()
	w.bits, w.virtual = maxBits-2, 1
	if got, err := w.logicalPosition(); err != nil || got != maxBits-1 {
		t.Fatalf("logical position=%d err=%v", got, err)
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

func TestForkIsolatesBindingsPathAndWireState(t *testing.T) {
	r := NewReader([]byte{0x00, 0x00})
	r.Set("N_E-UTRAN", 1)
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	a := r.Fork()
	b := r.Fork()
	a.Set("N_E-UTRAN", 2)
	if err := a.Enter("A"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.ReadSpare(); err != nil {
		t.Fatal(err)
	}
	a.RecordTruncation("A", a.BeginTruncation("A"), 1)
	a.RecordSpareCount("A", 1)
	if err := b.Enter("B"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReadSpare(); err != nil {
		t.Fatal(err)
	}
	if got, err := r.Eval("N_E-UTRAN"); err != nil || got != 1 {
		t.Fatalf("parent binding = %d, %v", got, err)
	}
	if got, err := b.Eval("N_E-UTRAN"); err != nil || got != 1 {
		t.Fatalf("sibling binding = %d, %v", got, err)
	}
	if got := a.Error(InvalidValue, "test").(*DecodeError).Path; got != "Root/A" {
		t.Fatalf("fork A path = %q", got)
	}
	if got := b.Error(InvalidValue, "test").(*DecodeError).Path; got != "Root/B" {
		t.Fatalf("fork B path = %q", got)
	}
	if len(r.Wire().Spare) != 0 || len(r.Wire().TruncatedAt) != 0 || len(r.Wire().SpareCounts) != 0 {
		t.Fatalf("fork changed parent wire: %+v", r.Wire())
	}
	if len(b.Wire().TruncatedAt) != 0 || len(b.Wire().SpareCounts) != 0 {
		t.Fatalf("fork changed sibling wire: %+v", b.Wire())
	}
	r.Commit(a)
	if got, err := r.Eval("N_E-UTRAN"); err != nil || got != 2 {
		t.Fatalf("committed binding = %d, %v", got, err)
	}
	zero := NewReader(nil)
	zero.SetZeroExtension(true)
	left, right := zero.Fork(), zero.Fork()
	if _, err := left.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if len(zero.Wire().ImplicitSpans) != 0 || len(right.Wire().ImplicitSpans) != 0 || len(left.Wire().ImplicitSpans) != 1 {
		t.Fatal("receiver-inferred bits leaked between forked candidates")
	}
}

func TestFreshSpareRunCompletesOctetOrValue(t *testing.T) {
	// TS 24.007 V20.0.0 Annex B.1.2.1 Rule B7: a fresh
	// <spare bits> run fills the remaining field or octet with zero.
	for _, tc := range []struct {
		limit int
		want  []byte
	}{{0, []byte{0xa0}}, {16, []byte{0xa0, 0}}} {
		w := NewWriter()
		old := -1
		if tc.limit != 0 {
			var err error
			old, err = w.PushLimit(tc.limit)
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := w.WriteUint(5, 3); err != nil {
			t.Fatal(err)
		}
		count := w.SpareFillCount()
		if count != len(tc.want)*8-3 {
			t.Fatalf("limit %d spare count %d", tc.limit, count)
		}
		for range count {
			if err := w.WriteSpare(); err != nil {
				t.Fatal(err)
			}
		}
		if tc.limit != 0 {
			if err := w.PopLimit(old); err != nil {
				t.Fatal(err)
			}
		}
		got, err := w.Finish(BitString{})
		if err != nil || !bytes.Equal(got, tc.want) {
			t.Fatalf("limit %d spare bytes %x, %v", tc.limit, got, err)
		}
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
		{"truncation", func(w *WireInfo) { w.TruncatedAt = map[string][]int{"missing": {1}} }},
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

// TS 44.060 V19.0.0 §11.1.4.4 and §12.24: a fresh truncated concatenation
// stops at an exhausted length-delimited value, as the reader does; a
// decoded value keeps its recorded truncation point instead.
func TestTruncationReachedFollowsEnclosingLength(t *testing.T) {
	for _, tc := range []struct {
		name string
		wire WireInfo
	}{
		{"canonical", WireInfo{canonical: true}},
		{"canonical with larger target", WireInfo{canonical: true, targetSet: true, targetBits: 64}},
		{"newly constructed", WireInfo{}},
	} {
		w := NewWriter()
		w.WithWire(tc.wire)
		old, err := w.PushLimit(3)
		if err != nil {
			t.Fatal(err)
		}
		if w.TruncationReached() {
			t.Fatalf("%s: truncated before the length was filled", tc.name)
		}
		if err := w.WriteUint(5, 3); err != nil {
			t.Fatal(err)
		}
		if !w.TruncationReached() {
			t.Fatalf("%s: continued past an exhausted length", tc.name)
		}
		if err := w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if w.TruncationReached() != (tc.wire.targetSet && w.Position() >= tc.wire.targetBits) {
			t.Fatalf("%s: unbounded truncation ignores the canonical target", tc.name)
		}
	}
	sealed := NewWriter()
	sealed.WithWire(Seal(struct{}{}, []byte{0}, 3, BitString{Bytes: []byte{0}, BitLength: 5}, WireInfo{}))
	if _, err := sealed.PushLimit(0); err != nil {
		t.Fatal(err)
	}
	if sealed.TruncationReached() {
		t.Fatal("decoded value ignored its recorded truncation point")
	}
	target := NewWriter()
	target.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 0})
	if !target.TruncationReached() {
		t.Fatal("unbounded canonical value ignored its target")
	}
}

// Every writer rejection for content that does not fit an enclosing bound is
// a *BoundError naming the bound kind, limit, position, path and field.
func TestWriterBoundConflictsAreTyped(t *testing.T) {
	fresh := func(wire WireInfo) *Writer {
		w := NewWriter()
		w.WithWire(wire)
		if err := w.Enter("Value"); err != nil {
			t.Fatal(err)
		}
		if err := w.Enter("Field"); err != nil {
			t.Fatal(err)
		}
		return w
	}
	bounded := func(wire WireInfo, width int) *Writer {
		w := fresh(wire)
		if _, err := w.PushLimit(width); err != nil {
			t.Fatal(err)
		}
		return w
	}
	sealed := Seal(struct{}{}, []byte{0}, 0, BitString{Bytes: []byte{0}, BitLength: 8}, WireInfo{})
	for _, tc := range []struct {
		name         string
		run          func() error
		kind         BoundKind
		field        string
		limit, at    int
		detailPrefix string
	}{
		{"bit beyond length", func() error { return bounded(WireInfo{}, 1).WriteUint(1, 2) }, LengthBound, "Field", 1, 1, "encoded bit exceeds"},
		{"bit beyond canonical target", func() error {
			return fresh(WireInfo{canonical: true, targetSet: true, targetBits: 2}).WriteUint(0, 3)
		}, CanonicalTarget, "Field", 2, 2, "encoded bit exceeds"},
		{"nonzero inferred bit beyond length", func() error {
			w := bounded(WireInfo{canonical: true}, 1)
			w.SetZeroExtension(true)
			return w.WriteUint(1, 2)
		}, LengthBound, "Field", 1, 1, "nonzero bit"},
		{"nonzero inferred bit beyond target", func() error {
			w := fresh(WireInfo{canonical: true, targetSet: true, targetBits: 1})
			w.SetZeroExtension(true)
			return w.WriteUint(1, 2)
		}, CanonicalTarget, "Field", 1, 1, "nonzero bit"},
		{"inner length beyond outer", func() error { _, err := bounded(WireInfo{}, 2).PushLimit(3); return err }, LengthBound, "Field", 2, 0, "length exceeds"},
		{"content before length", func() error {
			w := fresh(WireInfo{})
			old, err := w.PushLimit(2)
			if err != nil {
				return err
			}
			return w.PopLimit(old)
		}, LengthBound, "Field", 2, 0, "content ends"},
		{"content beyond fixed size", func() error {
			w := fresh(WireInfo{})
			if err := w.WriteUint(0, 3); err != nil {
				return err
			}
			return w.WritePaddingTo(2)
		}, LengthBound, "Field", 2, 3, "content exceeds"},
		{"omitted field at length", func() error { w := bounded(WireInfo{}, 0); return w.OmittedFieldError("Later") }, LengthBound, "Later", 0, 0, "nonzero field"},
		{"omitted field at target", func() error {
			return fresh(WireInfo{canonical: true, targetSet: true, targetBits: 0}).OmittedFieldError("Later")
		}, CanonicalTarget, "Later", 0, 0, "nonzero field"},
		{"omitted field at received truncation", func() error { return bounded(sealed, 0).OmittedFieldError("Later") }, ReceivedTruncation, "Later", 0, 0, "nonzero field"},
		{"length field conflict", func() error { return bounded(WireInfo{}, 4).LengthBoundError("stop record does not fit") }, LengthBound, "Field", 4, 0, "stop record"},
	} {
		err := tc.run()
		var bound *BoundError
		if !errors.As(err, &bound) || !errors.Is(err, ErrBoundConflict) {
			t.Fatalf("%s: error %v (%T) is not a typed bound conflict", tc.name, err, err)
		}
		if bound.Kind != tc.kind || bound.Path != "Value/Field" || bound.Field != tc.field || bound.Limit != tc.limit || bound.Position != tc.at || !strings.HasPrefix(bound.Detail, tc.detailPrefix) {
			t.Fatalf("%s: %+v", tc.name, *bound)
		}
	}
}

// A BoundError always names a real bound: without an enclosing value, or
// with an offset outside the bit limit, the writer reports a plain error.
// TS 44.018 V19.0.0 §10.5.2.16 sets the length-field bound in octets.
func TestBoundErrorRequiresRealBound(t *testing.T) {
	var bound *BoundError
	w := NewWriter()
	w.WithWire(WireInfo{})
	if err := w.LengthBoundError("stop record does not fit"); err == nil || errors.As(err, &bound) {
		t.Fatalf("unbounded length conflict: %v", err)
	}
	for _, octets := range []int{-1, maxBits/8 + 1} {
		if err := w.LengthFieldError("Arm", octets, "conflict"); err == nil || errors.As(err, &bound) {
			t.Fatalf("length field of %d octets: %v", octets, err)
		}
	}
	if err := w.boundError(LengthBound, -1, "Arm", "conflict"); err == nil || errors.As(err, &bound) {
		t.Fatalf("negative limit: %v", err)
	}
	if err := w.WriteUint(0, 8); err != nil {
		t.Fatal(err)
	}
	err := w.LengthFieldError("Arm", 2, "empty alternative requires zero length")
	if !errors.As(err, &bound) || bound.Kind != LengthBound || bound.Field != "Arm" || bound.Limit != 24 || bound.Position != 8 {
		t.Fatalf("length field bound: %v", err)
	}
}
