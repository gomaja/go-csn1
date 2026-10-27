package uecapability

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestImportedErrorPathComposition(t *testing.T) {
	for _, path := range []string{"", "Classmark3ValuePart"} {
		wantPath := "GERANCS/Classmark3"
		if path != "" {
			wantPath += "/" + path
		}
		de := &runtime.DecodeError{Kind: runtime.Truncated, Offset: 7, Path: path, Detail: "absent"}
		be := &runtime.BoundError{Kind: runtime.LengthBound, Path: path, Position: 7, Limit: 8, Detail: "missing"}
		fe := &runtime.FallbackError{Path: path, Position: 7}
		oe := &runtime.OperationError{Operation: "decode", Kind: runtime.Truncated, Location: runtime.Location{Path: path, BitOffset: 7, Width: 3}, Cause: de}
		for _, cause := range []error{de, be, fe, oe} {
			t.Run(fmt.Sprintf("%T/%s", cause, path), func(t *testing.T) {
				err := shiftOperation(cause, 40, "GERANCS/Classmark3")
				t.Logf("MESSAGE\t%T/%s\t%s", cause, path, err)
				var gotDecode *runtime.DecodeError
				var gotBound *runtime.BoundError
				var gotFallback *runtime.FallbackError
				var gotOperation *runtime.OperationError
				switch cause.(type) {
				case *runtime.OperationError:
					if !errors.As(err, &gotOperation) || gotOperation.Location.Path != wantPath || gotOperation.Location.BitOffset != 47 || gotOperation.Location.Width != 3 {
						t.Fatalf("operation: %v", err)
					}
					if gotOperation == oe || oe.Location.Path != path || oe.Location.BitOffset != 7 {
						t.Fatal("original operation changed")
					}
				case *runtime.DecodeError:
					if !errors.As(err, &gotDecode) || gotDecode.Path != wantPath || gotDecode.Offset != 47 || de.Path != path || de.Offset != 7 {
						t.Fatalf("decode: %v", err)
					}
					if err.Error() != "CSN.1 truncated at bit 47 in "+wantPath+": absent" {
						t.Fatal(err)
					}
				case *runtime.BoundError:
					if !errors.As(err, &gotBound) || gotBound.Path != wantPath || gotBound.Position != 47 || gotBound.Limit != 48 || !errors.Is(err, runtime.ErrBoundConflict) {
						t.Fatalf("bound: %v", err)
					}
					if be.Path != path || be.Position != 7 || be.Limit != 8 {
						t.Fatal("original bound changed")
					}
				case *runtime.FallbackError:
					if !errors.As(err, &gotFallback) || gotFallback.Path != wantPath || gotFallback.Position != 47 || !errors.Is(err, runtime.ErrFallbackKnown) {
						t.Fatalf("fallback: %v", err)
					}
					if fe.Path != path || fe.Position != 7 {
						t.Fatal("original fallback changed")
					}
				}
			})
		}
	}
	if shiftOperation(nil, 40, "GERANCS") != nil {
		t.Fatal("nil changed")
	}
	plain := errors.New("invalid component")
	if shiftOperation(plain, 40, "GERANCS") != plain {
		t.Fatal("generic identity changed")
	}
}
