package runtime

// fieldDecodeError retains the read width in the same allocation class as a
// DecodeError. Speculative choice failures therefore acquire no operation
// record; a final public failure requests one through errors.As.
type fieldDecodeError struct {
	DecodeError
	width int
}

func (e *fieldDecodeError) As(target any) bool {
	switch target := target.(type) {
	case **DecodeError:
		*target = &e.DecodeError
		return true
	case **OperationError:
		*target = &OperationError{Operation: "decode", Kind: e.Kind, Location: Location{Path: e.Path, BitOffset: e.Offset, Width: e.width}, Cause: &e.DecodeError}
		return true
	}
	return false
}
func (e *fieldDecodeError) Unwrap() error { return &e.DecodeError }

func decodeCause(err error) *DecodeError {
	for err != nil {
		switch e := err.(type) {
		case *DecodeError:
			return e
		case interface{ Unwrap() error }:
			err = e.Unwrap()
		default:
			return nil
		}
	}
	return nil
}
