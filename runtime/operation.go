package runtime

import "errors"

// operationFailure is called only after a failing operation. It normalizes
// already absolute reader/writer positions exactly once and preserves causes.
func operationFailure(cause error, operation, path string, position, width int, kind ErrorKind) error {
	if cause == nil {
		return nil
	}
	var old *OperationError
	if errors.As(cause, &old) {
		return cause
	}
	var de *DecodeError
	var be *BoundError
	var fe *FallbackError
	switch {
	case errors.As(cause, &de):
		position = de.Offset
		path = de.Path
		kind = de.Kind
	case errors.As(cause, &be):
		position = be.Position
		path = be.Path
		if be.Field != "" {
			path += "/" + be.Field
		}
		if be.Limit >= 0 && be.Limit <= maxBits && be.Position >= 0 && be.Position <= maxBits {
			width = be.Limit - be.Position
			if width < 0 {
				width = -width
			}
		}
	case errors.As(cause, &fe):
		position = fe.Position
		path = fe.Path
		kind = InvalidBranch
	case errors.Is(cause, ErrContextRequired):
		position = -1
		width = 0
	}
	return &OperationError{Operation: operation, Kind: kind, Location: Location{Path: path, BitOffset: position, Width: width}, Cause: cause}
}
func (w *Writer) operationFailure(cause error, width int, kind ErrorKind) error {
	if cause == nil || w.wire.rawErrors {
		return cause
	}
	return operationFailure(cause, "encode", w.path, w.bits, width, kind)
}

// OperationError attaches a generated field's failure context. Canonical
// candidate encodes postpone attachment until all candidates have failed.
func (w *Writer) OperationError(kind ErrorKind, path string, width int, cause error) error {
	if cause == nil {
		return nil
	}
	if w == nil {
		return invalidValueWriter()
	}
	if w.wire.rawErrors {
		return cause
	}
	return operationFailure(cause, "encode", path, w.bits, width, kind)
}
