package runtime

import (
	"errors"
	"strings"
	"testing"
)

func TestOperationErrorPreservesCause(t *testing.T) {
	cause := &DecodeError{Kind: Truncated, Offset: 11, Path: "Value/Field", Detail: "absent"}
	err := &OperationError{Operation: "decode", Kind: Truncated, Location: Location{Path: cause.Path, BitOffset: 11, Width: 3, Source: Source{Standard: "synthetic", Clause: "1"}}, Cause: cause}
	var de *DecodeError
	if !errors.As(err, &de) || de != cause {
		t.Fatal("typed cause lost")
	}
	if !strings.Contains(err.Error(), "bit 11") {
		t.Fatal(err)
	}
	e := &OperationError{Operation: "context", Location: Location{BitOffset: -1}, Cause: ErrContextRequired}
	if !errors.Is(e, ErrContextRequired) {
		t.Fatal("sentinel lost")
	}
	if strings.Contains(e.Error(), "bit") {
		t.Fatal("invented position", e)
	}
}
