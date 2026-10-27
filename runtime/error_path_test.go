package runtime

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestWriterErrorPaths(t *testing.T) {
	tests := []struct {
		name          string
		call          func(*Writer) error
		detail        string
		offset, width int
	}{
		{"uint", func(w *Writer) error { return w.WriteUint(32, 5) }, "value outside 5-bit field", 0, 5},
		{"bit string", func(w *Writer) error { return w.WriteBitString(BitString{}, 1) }, "invalid bit string width", 0, 1},
		{"unbounded length", func(w *Writer) error { return w.LengthBoundError("content missing") }, "content missing: no enclosing length-delimited value", 0, 0},
		{"length field", func(w *Writer) error { return w.LengthFieldError("Later", -1, "missing") }, "length field of -1 octets outside encoded bit limit", 0, 0},
		{"invalid fixed value", func(w *Writer) error { _, err := w.PushLimit(-1); return err }, "fixed value outside encoded bit limit", 0, 0},
		{"invalid enclosing value", func(w *Writer) error { return w.PopLimit(-2) }, "invalid enclosing fixed value", 0, 0},
		{"nesting", func(w *Writer) error { w.depth = 128; return w.Enter("Child") }, "encode nesting limit", 0, 0},
		{"logical bit limit", func(w *Writer) error { w.virtual = -1; _, err := w.logicalPosition(); return err }, "encoded bit limit exceeded", 0, 0},
		{"put bit limit", func(w *Writer) error { w.bits = maxBits; return w.put(0) }, "encoded bit limit exceeded", maxBits, 0},
		{"implicit edit", func(w *Writer) error {
			w.wire = WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}}
			return w.put(1)
		}, "edit to receiver-inferred bit", 0, 0},
		{"fallback mark", func(w *Writer) error {
			return w.CheckIgnoredFallback(FallbackMark{bits: -1}, false, func(*Reader) error { return nil })
		}, "invalid fallback mark", 0, 0},
		{"ignored width", func(w *Writer) error { return w.WriteIgnoredFixed(-1) }, "ignored field width outside 0..64", 0, 0},
		{"spare count", func(w *Writer) error {
			w.WithWire(WireInfo{SpareCounts: map[string][]int{"Root/Field": {-1}}})
			w.SpareCount("Root/Field")
			_, err := w.Bytes()
			return err
		}, "spare repeat count outside bit limit", 0, 0},
		{"invalid bound", func(w *Writer) error { return w.boundError(LengthBound, -1, "", "missing") }, "invalid length bound -1: missing", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWriter()
			if err := w.Enter("Root"); err != nil {
				t.Fatal(err)
			}
			if err := w.Enter("Field"); err != nil {
				t.Fatal(err)
			}
			err := tt.call(w)
			var op *OperationError
			if !errors.As(err, &op) {
				t.Fatalf("missing operation: %v", err)
			}
			if op.Location.Path != "Root/Field" || op.Location.BitOffset != tt.offset || op.Location.Width != tt.width {
				t.Fatalf("location: %+v", op.Location)
			}
			want := fmt.Sprintf("CSN.1 encode invalid-value at bit %d in Root/Field: %s", tt.offset, tt.detail)
			t.Logf("MESSAGE\t%s\t%s", tt.name, err)
			if err.Error() != want {
				t.Fatalf("got %q want %q", err, want)
			}
			if op.Cause == nil || errors.Unwrap(op) != op.Cause || !errors.Is(op, op.Cause) {
				t.Fatal("cause identity lost")
			}
		})
	}
}

func TestBoundPathComposition(t *testing.T) {
	for _, tt := range []struct{ path, field, want string }{{"Root/Field", "Field", "Root/Field"}, {"Root/Field", "Later", "Root/Field/Later"}, {"Root/Field", "Root/Field", "Root/Field"}, {"", "Field", "Field"}, {"Root/Field", "", "Root/Field"}} {
		t.Run(tt.path+"+"+tt.field, func(t *testing.T) {
			cause := &BoundError{Kind: LengthBound, Path: tt.path, Field: tt.field, Limit: 8, Position: 3, Detail: "missing"}
			standalone := "CSN.1 length bound (limit 8) at bit 3 in " + tt.want + ": missing"
			if cause.Error() != standalone {
				t.Fatalf("standalone: %v", cause)
			}
			err := operationFailure(cause, "encode", "unused", 0, 0, InvalidValue)
			var op *OperationError
			if !errors.As(err, &op) || op.Location.Path != tt.want {
				t.Fatalf("path: %v", err)
			}
			want := "CSN.1 encode invalid-value at bit 3 in " + tt.want + ": length bound (limit 8): missing"
			if err.Error() != want {
				t.Fatalf("got %q want %q", err, want)
			}
			if !errors.Is(err, ErrBoundConflict) || !errors.Is(err, cause) || errors.Unwrap(err) != cause {
				t.Fatal("bound cause lost")
			}
			if cause.Path != tt.path || cause.Field != tt.field {
				t.Fatal("bound fields changed")
			}
		})
	}
}

func TestIndexedCausePath(t *testing.T) {
	for _, tt := range []struct {
		outer, inner string
		included     bool
	}{
		{"Root/List[1]/Entry/Field", "Root/Entry/Field", true},
		{"Root/List[1]/Entry/Field", "Root/List/Entry/Field", true},
		{"Root/List[1]/Entry/Inner[2]/Field", "Root/Entry/Field", true},
		{"Root/List[1]/Entry/Field", "Other/Entry/Field", false},
		{"Root/List[1]/Entry/Field", "Root/Entry/Other", false},
		{"Root/Extra/Entry/Field", "Root/Entry/Field", false},
		{"Root/List[1]/Field", "List[2]/Field", false},
	} {
		if got := causePathIncluded(tt.outer, tt.inner); got != tt.included {
			t.Errorf("%q includes %q = %v", tt.outer, tt.inner, got)
		}
	}
}

func TestIndexedCausePathAllocationBound(t *testing.T) {
	for _, size := range []int{16, 256, 4096} {
		parent := strings.Repeat("R", size)
		outer := parent + "/" + strings.Repeat("Items[1]/", size) + "Field"
		inner := parent + "/Field"
		if !causePathIncluded(outer, inner) {
			t.Fatal("indexed path lost its parent")
		}
		if got := testing.AllocsPerRun(5, func() {
			if !causePathIncluded(outer, inner) {
				panic("path changed")
			}
		}); got != 0 {
			t.Fatalf("size %d allocated %.0f times", size, got)
		}
	}
}

func TestDeferredWriterPathPreservesPosition(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{SpareCounts: map[string][]int{"Root/Count": {-1}}})
	w.SpareCount("Root/Count")
	if err := w.WriteUint(0, 3); err != nil {
		t.Fatal(err)
	}
	_, err := w.Bytes()
	var op *OperationError
	if !errors.As(err, &op) || op.Location.BitOffset != 3 || op.Location.Path != "Root/Count" {
		t.Fatalf("deferred error: %v", err)
	}
	if err.Error() != "CSN.1 encode invalid-value at bit 3 in Root/Count: spare repeat count outside bit limit" {
		t.Fatal(err)
	}
}
