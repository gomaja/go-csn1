package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"testing"
)

func expectScalarZeros(r *Reader, width int) error {
	for i := 0; i < width; i++ {
		if err := r.Expect("0"); err != nil {
			return err
		}
	}
	return nil
}

func writeScalarZeros(w *Writer, width int) error {
	for i := 0; i < width; i++ {
		if err := w.WriteLiteral("0"); err != nil {
			return err
		}
	}
	return nil
}

func compareZeroRunErrors(t *testing.T, got, want error) {
	t.Helper()
	for g, w := got, want; g != nil || w != nil; g, w = errors.Unwrap(g), errors.Unwrap(w) {
		if reflect.TypeOf(g) != reflect.TypeOf(w) || g.Error() != w.Error() {
			t.Fatalf("errors differ: got %#v (%v), want %#v (%v)", got, got, want, want)
		}
	}
	var gotOperation, wantOperation *OperationError
	if errors.As(got, &gotOperation) != errors.As(want, &wantOperation) || gotOperation != nil && (gotOperation.Operation != wantOperation.Operation || gotOperation.Kind != wantOperation.Kind || gotOperation.Location != wantOperation.Location || gotOperation.Detail != wantOperation.Detail) {
		t.Fatalf("operation errors differ: got %+v, want %+v", gotOperation, wantOperation)
	}
	var gotDecode, wantDecode *DecodeError
	if errors.As(got, &gotDecode) != errors.As(want, &wantDecode) || gotDecode != nil && *gotDecode != *wantDecode {
		t.Fatalf("decode errors differ: got %+v, want %+v", gotDecode, wantDecode)
	}
	var gotBound, wantBound *BoundError
	if errors.As(got, &gotBound) != errors.As(want, &wantBound) || gotBound != nil && *gotBound != *wantBound {
		t.Fatalf("bound errors differ: got %+v, want %+v", gotBound, wantBound)
	}
}

func compareZeroRunWriters(t *testing.T, got, want *Writer) {
	t.Helper()
	compareZeroRunErrors(t, got.stateErr, want.stateErr)
	if got.bits != want.bits || got.virtual != want.virtual || got.limit != want.limit || got.bounded != want.bounded ||
		got.path != want.path || got.depth != want.depth || got.spare != want.spare || got.padding != want.padding || got.terminal != want.terminal ||
		got.implicitSpan != want.implicitSpan || got.allowZero != want.allowZero || got.stateErrPath != want.stateErrPath ||
		!bytes.Equal(got.bytes, want.bytes) || !maps.Equal(got.vars, want.vars) || !maps.Equal(got.spareCountIndex, want.spareCountIndex) ||
		!maps.Equal(got.truncationIndex, want.truncationIndex) || !reflect.DeepEqual(got.wire, want.wire) {
		t.Fatalf("writer states differ: got %+v, want %+v", got, want)
	}
}

func TestReaderExpectZerosEveryPosition(t *testing.T) {
	for _, start := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 31, 32, 55, 56, 57, 63, 64, 65, 71} {
		for width := 0; width <= 200; width++ {
			data := make([]byte, (start+width+7)/8+1)
			// Ones outside the requested run must never be inspected.
			for bit := 0; bit < start; bit++ {
				data[bit/8] |= 1 << uint(7-bit%8)
			}
			for bit := start + width; bit < len(data)*8; bit++ {
				data[bit/8] |= 1 << uint(7-bit%8)
			}
			for bad := -1; bad < width; bad++ {
				if bad >= 0 {
					at := start + bad
					data[at/8] |= 1 << uint(7-at%8)
				}
				got, want := NewReader(data), NewReader(data)
				got.pos, want.pos = start, start
				got.path, want.path = []string{"value", "padding"}, []string{"value", "padding"}
				got.fieldWidthLo, want.fieldWidthLo = 10, 10
				gotErr, wantErr := got.ExpectZeros(width), expectScalarZeros(want, width)
				compareZeroRunErrors(t, gotErr, wantErr)
				if got.pos != want.pos || got.virtual != want.virtual || got.fieldWidthLo != 10 || got.fieldWidthHi != 0 || *got.steps != *want.steps {
					t.Fatalf("start=%d width=%d bad=%d: state differs: got %+v, want %+v", start, width, bad, got, want)
				}
				if bad >= 0 {
					var de *DecodeError
					var oe *OperationError
					if !errors.As(gotErr, &de) || de.Kind != InvalidValue || de.Offset != start+bad || de.Path != "value/padding" || de.Detail != "literal 0 differs" || got.pos != start+bad+1 || !errors.As(gotErr, &oe) || oe.Location.Width != 1 {
						t.Fatalf("start=%d width=%d bad=%d: wrong failure: %v", start, width, bad, gotErr)
					}
					at := start + bad
					data[at/8] &^= 1 << uint(7-at%8)
				} else if gotErr != nil || got.pos != start+width {
					t.Fatalf("start=%d width=%d: zero run failed: %v", start, width, gotErr)
				}
			}
		}
	}
}

func TestReaderExpectZerosBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		setup func() *Reader
		width int
	}{
		{"nil", func() *Reader { return nil }, 1},
		{"empty-nil", func() *Reader { return nil }, 0},
		{"negative", func() *Reader { return nil }, -1},
		{"invalid", func() *Reader { return &Reader{} }, 1},
		{"empty-invalid", func() *Reader { return &Reader{} }, 0},
		{"truncated", func() *Reader { return NewReader(make([]byte, 9)) }, 100},
		{"bounded", func() *Reader { r := NewReader(make([]byte, 20)); r.end, r.boundDepth = 65, 1; return r }, 100},
		{"bounded-extension", func() *Reader {
			r := NewReader(make([]byte, 20))
			r.end, r.boundDepth, r.allowZero = 65, 1, true
			return r
		}, 100},
		{"extension", func() *Reader { r := NewReader(make([]byte, 9)); r.allowZero = true; return r }, 200},
		{"limit", func() *Reader { r := NewReader(nil); r.allowZero = true; return r }, maxBits + 1},
		{"physical-limit", func() *Reader { r := NewReader(make([]byte, maxBits/8)); r.pos = maxBits - 65; return r }, 66},
		{"virtual-limit", func() *Reader { r := NewReader(make([]byte, 20)); r.virtual = maxBits - 65; return r }, 66},
		{"invalid-position", func() *Reader { r := NewReader(nil); r.pos = -1; return r }, 10},
		{"invalid-virtual", func() *Reader { r := NewReader(nil); r.virtual = -1; return r }, 10},
	}
	for _, span := range []ImplicitSpan{{At: 0, Count: 0}, {At: 0, Count: maxBits}, {At: -1, Count: 1}, {At: maxBits, Count: 1}, {At: 0, Count: 1}, {At: 0, Count: 2}} {
		cases = append(cases, struct {
			name  string
			setup func() *Reader
			width int
		}{fmt.Sprint("span-", span), func() *Reader {
			r := NewReader(nil)
			r.allowZero, r.implicitShared = true, true
			r.wire.ImplicitSpans = []ImplicitSpan{span}
			return r
		}, 100})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, want := tc.setup(), tc.setup()
			compareZeroRunErrors(t, got.ExpectZeros(tc.width), expectScalarZeros(want, tc.width))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("state differs: got %+v, want %+v", got, want)
			}
		})
	}
}

func TestReaderExpectZerosForkAndObservations(t *testing.T) {
	for _, bad := range []int{-1, 1, 63, 64, 127} {
		setup := func() (*Reader, *Reader, *Collector) {
			data := make([]byte, 20)
			data[0] = 0x80
			if bad >= 0 {
				data[bad/8] |= 1 << uint(7-bad%8)
			}
			c := new(Collector)
			r := NewReaderWithCollector(data, c)
			r.pos = 1
			if err := r.Observe(0, 1, 1, Source{}, "reserved"); err != nil {
				t.Fatal(err)
			}
			r.allowZero = true
			return r, r.Fork(), c
		}
		gotParent, got, gotCollector := setup()
		wantParent, want, wantCollector := setup()
		compareZeroRunErrors(t, got.ExpectZeros(200), expectScalarZeros(want, 200))
		if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(gotParent, wantParent) || len(gotParent.wire.ImplicitSpans) != 0 {
			t.Fatalf("bad=%d: fork or parent differs", bad)
		}
		gotParent.Commit(got)
		wantParent.Commit(want)
		if !reflect.DeepEqual(gotCollector.TakeReport(), wantCollector.TakeReport()) {
			t.Fatalf("bad=%d: observations differ", bad)
		}
	}
	// Extending an existing shared span must not modify the parent's storage.
	r := NewReader(nil)
	r.allowZero = true
	if err := r.Expect("0"); err != nil {
		t.Fatal(err)
	}
	f := r.Fork()
	if err := f.ExpectZeros(200); err != nil {
		t.Fatal(err)
	}
	if r.virtual != 1 || r.wire.ImplicitSpans[0].Count != 1 || f.virtual != 201 || f.wire.ImplicitSpans[0].Count != 201 {
		t.Fatal("fork changed shared inferred zeros")
	}
}

func zeroRunWriter(start int) *Writer {
	w := NewWriter()
	w.path = "value/padding"
	for i := 0; i < start; i++ {
		if err := w.WriteLiteral("1"); err != nil {
			panic(err)
		}
	}
	return w
}

func TestWriterWriteZerosEveryPosition(t *testing.T) {
	for start := 0; start <= 72; start++ {
		for width := 0; width <= 200; width++ {
			got, want := zeroRunWriter(start), zeroRunWriter(start)
			compareZeroRunErrors(t, got.WriteZeros(width), writeScalarZeros(want, width))
			compareZeroRunWriters(t, got, want)
			for at := 0; at < got.bits; at++ {
				bit := got.bytes[at/8] >> uint(7-at%8) & 1
				if (at < start && bit != 1) || (at >= start && bit != 0) {
					t.Fatalf("start=%d width=%d: wrong bit at %d", start, width, at)
				}
			}
		}
	}
}

func TestWriterWriteZerosBoundaries(t *testing.T) {
	for _, start := range []int{0, 1, 7, 8, 63, 64, 65} {
		for _, limit := range []int{0, 1, 7, 8, 63, 64, 65, 127} {
			for flags := 0; flags < 32; flags++ {
				setup := func() *Writer {
					w := zeroRunWriter(start)
					w.wire.canonical, w.wire.sealed, w.allowZero = flags&1 != 0, flags&2 != 0, flags&4 != 0
					w.bounded, w.wire.targetSet = flags&8 != 0, flags&16 != 0
					w.limit, w.wire.targetBits = limit, limit+1
					w.wire.ImplicitSpans = []ImplicitSpan{{At: start + 3, Count: 65}, {At: start + 90, Count: 3}}
					return w
				}
				got, want := setup(), setup()
				compareZeroRunErrors(t, got.WriteZeros(200), writeScalarZeros(want, 200))
				compareZeroRunWriters(t, got, want)
			}
		}
	}
	for _, tc := range []struct {
		name  string
		setup func() *Writer
		width int
	}{
		{"negative", NewWriter, -1},
		{"empty", NewWriter, 0},
		{"invalid-position", func() *Writer { w := NewWriter(); w.bits = -1; return w }, 100},
		{"invalid-virtual", func() *Writer { w := NewWriter(); w.virtual = -1; return w }, 100},
		{"virtual-limit", func() *Writer { w := NewWriter(); w.virtual = maxBits - 65; return w }, 66},
		{"limit", NewWriter, maxBits + 1},
		{"invalid-span", func() *Writer {
			w := NewWriter()
			w.wire.sealed = true
			w.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: 0}}
			return w
		}, 100},
		{"invalid-next-span", func() *Writer {
			w := NewWriter()
			w.wire.sealed = true
			w.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: 65}, {At: 66, Count: 0}}
			return w
		}, 100},
		{"past-span", func() *Writer {
			w := zeroRunWriter(70)
			w.wire.sealed = true
			w.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: 65}}
			return w
		}, 100},
		{"maximal-span", func() *Writer {
			w := NewWriter()
			w.wire.sealed = true
			w.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: maxBits}}
			return w
		}, maxBits + 1},
		{"raw-errors", func() *Writer { w := NewWriter(); w.wire.rawErrors = true; w.bounded = true; w.limit = 65; return w }, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, want := tc.setup(), tc.setup()
			compareZeroRunErrors(t, got.WriteZeros(tc.width), writeScalarZeros(want, tc.width))
			compareZeroRunWriters(t, got, want)
		})
	}
}

func TestWriterWriteZerosEveryBoundary(t *testing.T) {
	for boundary := 0; boundary <= 200; boundary++ {
		for _, start := range []int{0, 1, 7, 63, 65} {
			for flags := 0; flags < 32; flags++ {
				for _, width := range []int{max(0, boundary-start), max(0, boundary-start) + 1, max(0, boundary-start) + 68} {
					setup := func() *Writer {
						w := zeroRunWriter(start)
						w.wire.canonical, w.wire.sealed, w.allowZero = flags&1 != 0, flags&2 != 0, flags&4 != 0
						w.bounded, w.wire.targetSet = flags&8 != 0, flags&16 != 0
						w.limit, w.wire.targetBits = boundary, boundary+1
						if flags&2 != 0 {
							// Reverse which bound occurs first in sealed cases.
							w.limit, w.wire.targetBits = boundary+1, boundary
							w.wire.ImplicitSpans = []ImplicitSpan{{At: boundary, Count: 65}, {At: boundary + 66, Count: 1}}
						}
						return w
					}
					got, want := setup(), setup()
					compareZeroRunErrors(t, got.WriteZeros(width), writeScalarZeros(want, width))
					compareZeroRunWriters(t, got, want)
				}
			}
		}
	}
}

func TestZeroRunsExtremeIntegersAndWriterStorage(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	for _, width := range []int{minInt, -1, 0, 1, 65, maxInt} {
		for _, position := range []int{minInt, -1, 0, maxBits - 1, maxBits, maxBits + 1, maxInt} {
			gotReader, wantReader := NewReader(nil), NewReader(nil)
			gotReader.pos, wantReader.pos = position, position
			gotReader.allowZero, wantReader.allowZero = true, true
			compareZeroRunErrors(t, gotReader.ExpectZeros(width), expectScalarZeros(wantReader, width))
			if !reflect.DeepEqual(gotReader, wantReader) {
				t.Fatalf("width=%d position=%d: reader state differs", width, position)
			}
			gotWriter, wantWriter := NewWriter(), NewWriter()
			gotWriter.bits, wantWriter.bits = position, position
			compareZeroRunErrors(t, gotWriter.WriteZeros(width), writeScalarZeros(wantWriter, width))
			compareZeroRunWriters(t, gotWriter, wantWriter)
		}
	}
	for _, limit := range []int{minInt, -1, maxBits + 1, maxInt} {
		for flags := 0; flags < 32; flags++ {
			setup := func() *Writer {
				w := zeroRunWriter(1)
				w.wire.canonical, w.wire.sealed, w.allowZero = flags&1 != 0, flags&2 != 0, flags&4 != 0
				w.bounded, w.wire.targetSet = flags&8 != 0, flags&16 != 0
				w.limit, w.wire.targetBits = limit, limit
				w.wire.ImplicitSpans = []ImplicitSpan{{At: limit, Count: 1}}
				return w
			}
			got, want := setup(), setup()
			compareZeroRunErrors(t, got.WriteZeros(100), writeScalarZeros(want, 100))
			compareZeroRunWriters(t, got, want)
		}
	}
	// A received inferred bit takes precedence over the physical bound. Once
	// that span ends, the next bit must reject even the lowest signed limit.
	for _, limit := range []int{minInt, -1, 0, 1} {
		setup := func() *Writer {
			w := zeroRunWriter(1)
			w.bounded, w.limit, w.wire.sealed = true, limit, true
			w.wire.ImplicitSpans = []ImplicitSpan{{At: 1, Count: 1}}
			return w
		}
		got, want := setup(), setup()
		compareZeroRunErrors(t, got.WriteZeros(100), writeScalarZeros(want, 100))
		compareZeroRunWriters(t, got, want)
	}
	for _, start := range []int{0, 1, 7, 8, 63, 64, 65} {
		for _, storage := range [][]byte{nil, {0xff}, {0xff, 0xa5, 0x81}, make([]byte, 100)} {
			for _, width := range []int{0, 1, 7, 8, 63, 64, 65, 200} {
				setup := func() *Writer {
					w := NewWriter()
					w.bits = start
					w.bytes = append([]byte(nil), storage...)
					return w
				}
				got, want := setup(), setup()
				compareZeroRunErrors(t, got.WriteZeros(width), writeScalarZeros(want, width))
				compareZeroRunWriters(t, got, want)
			}
		}
	}
}

func TestZeroRunGrowth(t *testing.T) {
	for width := 64; width <= maxBits; width *= 2 {
		data := make([]byte, width/8)
		r := NewReader(data)
		if allocations := testing.AllocsPerRun(3, func() {
			r.pos = 0
			if err := r.ExpectZeros(width); err != nil {
				t.Fatal(err)
			}
		}); allocations != 0 {
			t.Fatalf("width=%d: reader allocates %g", width, allocations)
		}
		w := NewWriter()
		// A -race build allocates the temporary of append(bytes, make(...)...),
		// which ordinary builds fold into the growth; only the bound differs.
		if allocations := testing.AllocsPerRun(3, func() {
			w.bits, w.bytes = 0, nil
			if err := w.WriteZeros(width); err != nil {
				t.Fatal(err)
			}
		}); allocations > 2 && !raceEnabled {
			t.Fatalf("width=%d: writer allocations %g exceed constant bound", width, allocations)
		}
		if len(w.bytes) != width/8 || cap(w.bytes) > width/8+64 {
			t.Fatalf("width=%d: output storage len=%d cap=%d exceeds linear bound", width, len(w.bytes), cap(w.bytes))
		}
		data[len(data)-1] = 1
		r.pos = 0
		var de *DecodeError
		if err := r.ExpectZeros(width); !errors.As(err, &de) || de.Offset != width-1 {
			t.Fatalf("width=%d: last nonzero bit accepted or misplaced: %v", width, err)
		}
	}
}
