package runtime

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestOperationMessageCauseFacts(t *testing.T) {
	de := &DecodeError{Kind: Truncated, Offset: 11, Path: "Value/Field", Detail: "absent"}
	be := &BoundError{Kind: LengthBound, Path: "Value", Field: "Field", Limit: 16, Position: 11, Detail: "nonzero component"}
	fe := &FallbackError{Path: "Value/Field", Position: 11}
	ce := &ContextRequiredError{Standard: "synthetic", Clause: "1", Name: "Value/Field"}
	ee := &ExtentError{Actual: 3, Minimum: 1, Maximum: 2}
	nested := &OperationError{Operation: "encode", Kind: Truncated, Location: Location{Path: de.Path, BitOffset: 11}, Cause: de}
	tests := []struct {
		name      string
		cause     error
		operation string
		kind      ErrorKind
		location  Location
		want      string
	}{
		{"field cause", &fieldDecodeError{DecodeError: *de, width: 3}, "decode", Truncated, Location{Path: de.Path, BitOffset: 11}, "CSN.1 decode truncated at bit 11 in Value/Field: absent"},
		{"decode", de, "decode", Truncated, Location{Path: de.Path, BitOffset: 11}, "CSN.1 decode truncated at bit 11 in Value/Field: absent"},
		{"empty cause path", &DecodeError{Kind: Truncated, Offset: 11, Detail: "absent"}, "decode", Truncated, Location{Path: de.Path, BitOffset: 11}, "CSN.1 decode truncated at bit 11 in Value/Field: absent"},
		{"different cause facts", de, "canonical", InvalidSchema, Location{Path: "Other", BitOffset: 4}, "CSN.1 canonical invalid-schema at bit 4 in Other: truncated at bit 11 in Value/Field: absent"},
		{"different kind", de, "decode", InvalidValue, Location{Path: de.Path, BitOffset: 11}, "CSN.1 decode invalid-value at bit 11 in Value/Field: truncated: absent"},
		{"indexed path", de, "decode", Truncated, Location{Path: "Items[2]/Value/Field", BitOffset: 11}, "CSN.1 decode truncated at bit 11 in Items[2]/Value/Field: absent"},
		{"bound", be, "encode", InvalidValue, Location{Path: "Value/Field", BitOffset: 11}, "CSN.1 encode invalid-value at bit 11 in Value/Field: length bound (limit 16): nonzero component"},
		{"different bound location", be, "encode", InvalidValue, Location{Path: "Other", BitOffset: 4}, "CSN.1 encode invalid-value at bit 4 in Other: at bit 11 in Value/Field: length bound (limit 16): nonzero component"},
		{"fallback", fe, "encode", InvalidBranch, Location{Path: fe.Path, BitOffset: 11}, "CSN.1 encode invalid-branch at bit 11 in Value/Field: ignored fallback bits decode through the known arm"},
		{"context", ce, "decode", InvalidValue, Location{Path: ce.Name, BitOffset: -1}, "CSN.1 decode invalid-value in Value/Field: synthetic §1: CSN.1 context required"},
		{"context sentinel", ErrContextRequired, "encode", InvalidValue, Location{Path: "Value", BitOffset: -1}, "CSN.1 encode invalid-value in Value: CSN.1 context required"},
		{"extent", ee, "canonical", InvalidValue, Location{Path: "Value", BitOffset: -1}, "CSN.1 canonical invalid-value in Value: CSN.1 extent 3 outside 1..2 octets"},
		{"nested operation", nested, "canonical", Truncated, nested.Location, "CSN.1 canonical truncated at bit 11 in Value/Field: encode: absent"},
		{"same operation", nested, "encode", Truncated, nested.Location, "CSN.1 encode truncated at bit 11 in Value/Field: absent"},
		{"unsupported", ErrUnsupported, "context", InvalidValue, Location{Path: "Value", BitOffset: -1}, "CSN.1 context invalid-value in Value: unsupported CSN.1 value"},
		{"empty value", ErrEmptyValue, "decode", InvalidValue, Location{Path: "Value", BitOffset: -1}, "CSN.1 decode invalid-value in Value: empty CSN.1 value"},
		{"bound sentinel", ErrBoundConflict, "encode", InvalidValue, Location{Path: "Value", BitOffset: -1}, "CSN.1 encode invalid-value in Value: CSN.1 value conflicts with its enclosing bound"},
		{"fallback sentinel", ErrFallbackKnown, "encode", InvalidBranch, Location{Path: "Value", BitOffset: -1}, "CSN.1 encode invalid-branch in Value: CSN.1 ignored fallback bits decode through the known arm"},
		{"wrapped decode", fmt.Errorf("decode-back: %w", de), "canonical", Truncated, Location{Path: de.Path, BitOffset: 11}, "CSN.1 canonical truncated at bit 11 in Value/Field: decode-back: absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &OperationError{Operation: tt.operation, Kind: tt.kind, Location: tt.location, Cause: tt.cause}
			t.Logf("MESSAGE\t%s\t%s", tt.name, err.Error())
			if got := err.Error(); got != tt.want {
				t.Fatalf("got %q\nwant %q", got, tt.want)
			}
			if errors.Unwrap(err) != tt.cause || !errors.Is(err, tt.cause) {
				t.Fatal("cause identity changed")
			}
			for _, sentinel := range []error{ErrContextRequired, ErrUnsupported, ErrEmptyValue, ErrBoundConflict, ErrFallbackKnown} {
				if errors.Is(err, sentinel) != errors.Is(tt.cause, sentinel) {
					t.Fatalf("sentinel %v changed", sentinel)
				}
			}
			var gotField, wantField *fieldDecodeError
			if errors.As(err, &gotField) != errors.As(tt.cause, &wantField) || gotField != wantField {
				t.Fatal("field cause changed")
			}
			var gotDecode, wantDecode *DecodeError
			if errors.As(err, &gotDecode) != errors.As(tt.cause, &wantDecode) || gotDecode != wantDecode {
				t.Fatal("decode cause changed")
			}
			var gotBound, wantBound *BoundError
			if errors.As(err, &gotBound) != errors.As(tt.cause, &wantBound) || gotBound != wantBound {
				t.Fatal("bound cause changed")
			}
			var gotFallback, wantFallback *FallbackError
			if errors.As(err, &gotFallback) != errors.As(tt.cause, &wantFallback) || gotFallback != wantFallback {
				t.Fatal("fallback cause changed")
			}
			var gotContext, wantContext *ContextRequiredError
			if errors.As(err, &gotContext) != errors.As(tt.cause, &wantContext) || gotContext != wantContext {
				t.Fatal("context cause changed")
			}
			var gotExtent, wantExtent *ExtentError
			if errors.As(err, &gotExtent) != errors.As(tt.cause, &wantExtent) || gotExtent != wantExtent {
				t.Fatal("extent cause changed")
			}
			for _, fact := range []string{string(tt.kind), tt.location.Path} {
				if fact != "" && strings.Count(err.Error(), fact) != 1 {
					t.Fatalf("fact %q repeated: %v", fact, err)
				}
			}
			if tt.location.BitOffset >= 0 {
				fact := fmt.Sprintf("at bit %d", tt.location.BitOffset)
				if strings.Count(err.Error(), fact) != 1 {
					t.Fatalf("fact %q repeated: %v", fact, err)
				}
			}
		})
	}
}

func TestDecodeMessageEmptyPath(t *testing.T) {
	err := &DecodeError{Kind: Truncated, Offset: 8, Detail: "absent"}
	if got, want := err.Error(), "CSN.1 truncated at bit 8: absent"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOperationMessageConstructors(t *testing.T) {
	de := &DecodeError{Kind: Truncated, Offset: 11, Path: "Value/Field", Detail: "absent"}
	field := &fieldDecodeError{DecodeError: *de, width: 3}
	var located *OperationError
	if !errors.As(field, &located) {
		t.Fatal("field location absent")
	}
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"field As", located, "CSN.1 decode truncated at bit 11 in Value/Field: absent"},
		{"writer decode", operationFailure(de, "encode", "unused", 0, 0, InvalidValue), "CSN.1 encode truncated at bit 11 in Value/Field: absent"},
		{"writer field", operationFailure(field, "encode", "unused", 0, 0, InvalidValue), "CSN.1 truncated at bit 11 in Value/Field: absent"},
		{"writer bound", operationFailure(&BoundError{Kind: CanonicalTarget, Limit: 16, Position: 11, Path: "Value", Field: "Field", Detail: "nonzero"}, "encode", "unused", 0, 0, InvalidValue), "CSN.1 encode invalid-value at bit 11 in Value/Field: canonical-target bound (limit 16): nonzero"},
		{"writer fallback", operationFailure(&FallbackError{Path: "Value/Field", Position: 11}, "encode", "unused", 0, 0, InvalidValue), "CSN.1 encode invalid-branch at bit 11 in Value/Field: ignored fallback bits decode through the known arm"},
		{"writer context", operationFailure(ErrContextRequired, "encode", "Value", 0, 0, InvalidValue), "CSN.1 encode invalid-value in Value: CSN.1 context required"},
		{"canonical decode-back", canonicalFailure(fmt.Errorf("canonical bytes do not decode: %w", de)), "CSN.1 canonical invalid-schema: canonical bytes do not decode: truncated at bit 11 in Value/Field: absent"},
		{"invalid writer", invalidValueWriter(), "CSN.1 encode invalid-value: nil writer or value encoder"},
		{"writer typed context", operationFailure(&ContextRequiredError{Standard: "synthetic", Clause: "1", Name: "Value"}, "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value in Value: synthetic §1: CSN.1 context required"},
		{"writer extent", operationFailure(&ExtentError{Required: true}, "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value at bit 5 in Value: CSN.1 canonical encoding requires an explicit containing-message length"},
		{"writer unsupported", operationFailure(ErrUnsupported, "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value at bit 5 in Value: unsupported CSN.1 value"},
		{"writer empty value", operationFailure(ErrEmptyValue, "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value at bit 5 in Value: empty CSN.1 value"},
		{"writer bound sentinel", operationFailure(ErrBoundConflict, "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value at bit 5 in Value: CSN.1 value conflicts with its enclosing bound"},
		{"writer fallback sentinel", operationFailure(ErrFallbackKnown, "encode", "Value", 5, 2, InvalidBranch), "CSN.1 encode invalid-branch at bit 5 in Value: CSN.1 ignored fallback bits decode through the known arm"},
		{"writer generic", operationFailure(fmt.Errorf("invalid component"), "encode", "Value", 5, 2, InvalidValue), "CSN.1 encode invalid-value at bit 5 in Value: invalid component"},
		{"writer wrapped decode", operationFailure(fmt.Errorf("field: %w", de), "encode", "unused", 0, 0, InvalidValue), "CSN.1 encode truncated at bit 11 in Value/Field: field: absent"},
		{"writer existing operation", operationFailure(located, "encode", "unused", 0, 0, InvalidValue), "CSN.1 decode truncated at bit 11 in Value/Field: absent"},
		{"canonical generic", canonicalFailure(fmt.Errorf("invalid canonical extent range")), "CSN.1 canonical invalid-schema: invalid canonical extent range"},
		{"canonical mismatch", canonicalFailure(errCanonicalMismatch), "CSN.1 canonical invalid-schema: canonical bytes change typed semantics"},
		{"canonical nested operation", canonicalFailure(located), "CSN.1 canonical invalid-schema: decode truncated at bit 11 in Value/Field: absent"},
		{"canonical wrapped field", canonicalFailure(fmt.Errorf("canonical bytes do not decode: %w", field)), "CSN.1 canonical invalid-schema: canonical bytes do not decode: truncated at bit 11 in Value/Field: absent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("MESSAGE\t%s\t%s", tt.name, tt.err.Error())
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestOperationMessageExplicitDetail(t *testing.T) {
	inner := &OperationError{Operation: "encode", Kind: InvalidValue, Location: Location{Path: "Value", BitOffset: 7}, Detail: "wrong component"}
	outer := *inner
	outer.Operation = "canonical"
	outer.Cause = inner
	want := "CSN.1 canonical invalid-value at bit 7 in Value: encode: wrong component"
	if got := outer.Error(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	outer.Detail = "explicit explanation"
	want = "CSN.1 canonical invalid-value at bit 7 in Value: explicit explanation"
	if got := outer.Error(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOperationMessageBoundAtPosition(t *testing.T) {
	cause := &BoundError{Kind: ReceivedTruncation, Path: "Value", Field: "Field", Position: 11, Limit: 11, Detail: "nonzero"}
	err := &OperationError{Operation: "encode", Kind: InvalidValue, Location: Location{Path: "Other", BitOffset: 4}, Cause: cause}
	want := "CSN.1 encode invalid-value at bit 4 in Other: at bit 11 in Value/Field: received-truncation bound (limit 11): nonzero"
	if got := err.Error(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOperationMessageDifferentLookupName(t *testing.T) {
	err := &OperationError{Operation: "lookup", Kind: InvalidSchema, Location: Location{Path: "Outer", BitOffset: -1}, Cause: fmt.Errorf("unknown CSN.1 definition %q in synthetic", "Inner")}
	want := "CSN.1 lookup invalid-schema in Outer: unknown CSN.1 definition \"Inner\" in synthetic"
	if got := err.Error(); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestOperationMessageLongPaths(t *testing.T) {
	for _, n := range []int{16, 1024, 65536} {
		path := "Value/" + strings.Repeat("Field", n)
		cause := &DecodeError{Kind: Truncated, Offset: 11, Path: path, Detail: "absent"}
		err := &OperationError{Operation: "decode", Kind: Truncated, Location: Location{Path: path, BitOffset: 11}, Cause: cause}
		want := "CSN.1 decode truncated at bit 11 in " + path + ": absent"
		if got := err.Error(); got != want {
			t.Fatalf("path length %d: message does not grow by the path length", len(path))
		}
	}
}
