package runtime

import (
	"errors"
	"testing"
)

type componentErrorWriter interface {
	ComponentError(ErrorKind, string, int, error) error
}
type errorPathReader interface{ ErrorPath(string) string }

func TestComponentErrorPaths(t *testing.T) {
	for _, tc := range []struct{ parent, child, want string }{
		{"", "", ""}, {"", "Field", "Field"}, {"Root", "", "Root"},
		{"Root", "Field", "Root/Field"}, {"Root/Field", "Field", "Root/Field"},
		{"Root/Field", "Root/Field", "Root/Field"}, {"Root/Field", "Other", "Root/Field/Other"},
		{"Root/Field2", "Field", "Root/Field2/Field"},
		{"First", "Shared", "First/Shared"}, {"Second/Nested", "Shared", "Second/Nested/Shared"},
	} {
		t.Run(tc.parent+"+"+tc.child, func(t *testing.T) {
			w := NewWriter()
			r := NewReader(nil)
			if tc.parent != "" {
				if err := w.Enter(tc.parent); err != nil {
					t.Fatal(err)
				}
				if err := r.Enter(tc.parent); err != nil {
					t.Fatal(err)
				}
			}
			writer, ok := any(w).(componentErrorWriter)
			if !ok {
				t.Fatal("writer has no component error method")
			}
			reader, ok := any(r).(errorPathReader)
			if !ok {
				t.Fatal("reader has no error path method")
			}
			cause := errors.New("invalid component")
			err := writer.ComponentError(InvalidBranch, tc.child, 3, cause)
			var op *OperationError
			if !errors.As(err, &op) || op.Operation != "encode" || op.Kind != InvalidBranch || op.Location.Path != tc.want || op.Location.BitOffset != 0 || op.Location.Width != 3 || errors.Unwrap(err) != cause {
				t.Fatalf("component error: %v; want %q", err, tc.want)
			}
			if got := reader.ErrorPath(tc.child); got != tc.want {
				t.Fatalf("reader path %q want %q", got, tc.want)
			}
			explicit := w.OperationError(InvalidBranch, "Already/Composed", 3, cause)
			if !errors.As(explicit, &op) || op.Location.Path != "Already/Composed" {
				t.Fatal(explicit)
			}
			if writer.ComponentError(InvalidBranch, tc.child, 3, nil) != nil {
				t.Fatal("nil cause changed")
			}
			w.Leave()
			r.Leave()
			if got := reader.ErrorPath("Field"); got != "Field" {
				t.Fatal(got)
			}
		})
	}
}

func TestComponentErrorPreservesTypedCauses(t *testing.T) {
	w := NewWriter()
	if err := w.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	writer, ok := any(w).(componentErrorWriter)
	if !ok {
		t.Fatal("writer has no component error method")
	}
	existing := &OperationError{Operation: "encode", Kind: InvalidValue, Location: Location{Path: "Root/Existing", BitOffset: 7}}
	if got := writer.ComponentError(InvalidBranch, "Other", 3, existing); got != existing {
		t.Fatal("existing error changed")
	}
	cause := &DecodeError{Kind: Truncated, Offset: 7, Path: "Root/Existing", Detail: "absent"}
	err := writer.ComponentError(InvalidBranch, "Other", 3, cause)
	var op *OperationError
	if !errors.As(err, &op) || op.Location.Path != cause.Path || op.Location.BitOffset != 7 || op.Kind != Truncated || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	w.wire.rawErrors = true
	plain := errors.New("candidate failed")
	if writer.ComponentError(InvalidValue, "Field", 0, plain) != plain {
		t.Fatal("raw candidate changed")
	}
	if n := testing.AllocsPerRun(5, func() { _ = writer.ComponentError(InvalidValue, "Field", 0, plain) }); n != 0 {
		t.Fatalf("raw candidate allocated %g times", n)
	}
	var nilWriter *Writer
	nilMethod, ok := any(nilWriter).(componentErrorWriter)
	if !ok {
		t.Fatal("writer has no component error method")
	}
	if nilMethod.ComponentError(InvalidValue, "Field", 0, plain) == nil {
		t.Fatal("nil writer accepted")
	}
	if nilMethod.ComponentError(InvalidValue, "Field", 0, nil) != nil {
		t.Fatal("nil cause changed")
	}
	var nilReader *Reader
	nilPath, ok := any(nilReader).(errorPathReader)
	if !ok {
		t.Fatal("reader has no error path method")
	}
	if nilPath.ErrorPath("Field") != "Field" {
		t.Fatal("nil reader path")
	}
}
