package runtime

import "errors"

// lhPositionState separates replay's logical spans from L/H coordinates.
// TS 24.007 V20.0.0 Annex B §B.1.2.2 Rule B8 fixes transmitted bits
// relative to the message start. TS 24.008 V20.1.0 §10.5.1.7 permits
// zeros appended at the end of the received string. No source assigns an
// inferred L/H bit a position after transmission resumes; refuse that case.
type lhPositionState struct{ start, count int }

// ErrLHPositionUndefined identifies an L/H evaluation on a later inferred run.
// TS 24.007 V20.0.0 Annex B §B.1.2.2 rule B8 assigns transmitted bits
// their physical position. TS 24.008 V20.1.0 §10.5.1.7 appends inferred
// zeros at the received end; neither assigns a position after transmission
// resumes. This refusal also applies to non-selected trial choice arms.
var ErrLHPositionUndefined = errors.New("no source assigns this receiver-inferred L/H bit a position after transmitted bits resumed")

var errLHPositionLimit = errors.New("L/H position exceeds bit limit")

// lhPositionError preserves the located decode facts without changing the
// representation or allocation size of ordinary speculative decode errors.
type lhPositionError struct{ cause error }

func (e *lhPositionError) Error() string        { return e.cause.Error() }
func (e *lhPositionError) Unwrap() error        { return e.cause }
func (e *lhPositionError) Is(target error) bool { return target == ErrLHPositionUndefined }

func (e *lhPositionError) As(target any) bool {
	// fieldDecodeError supplies an operation record on demand. Keep this
	// refusal in that record's cause instead of losing it during conversion.
	if operation, ok := target.(**OperationError); ok {
		var located *OperationError
		if errors.As(e.cause, &located) {
			copy := *located
			copy.Cause = e
			*operation = &copy
			return true
		}
	}
	return false
}

func (s *lhPositionState) infer(physical, count int) {
	if count == 0 {
		return
	}
	if s.count == 0 {
		s.start = physical
	}
	// Saturate at an invalid coordinate rather than overflow if independent
	// received scopes collectively exceed the logical bit budget.
	if s.count <= maxBits {
		s.count += min(count, maxBits+1-s.count)
	}
}
func (s lhPositionState) position(physical int, inferred bool) (int, error) {
	if !inferred {
		return physical, nil
	}
	return s.inferredPosition(physical)
}

func (s lhPositionState) inferredPosition(physical int) (int, error) {
	if s.count > 0 && physical != s.start {
		return 0, ErrLHPositionUndefined
	}
	if physical < 0 || physical > maxBits || s.count > maxBits-physical {
		return 0, errLHPositionLimit
	}
	return physical + s.count, nil
}

// LHBit returns the L/H value of the next bit without consuming it. A first
// inferred run continues from the received end (TS 24.008 V20.1.0 §10.5.1.7).
// A later inferred run returns InvalidSchema wrapping ErrLHPositionUndefined.
func (r *Reader) LHBit(symbol rune) (uint8, error) {
	position, err := r.lh.position(r.pos, r.pos >= r.end && r.allowZero)
	if err != nil {
		return 0, r.lhPositionFailure(err)
	}
	return literalBit(symbol, position), nil
}

func (r *Reader) lhPositionFailure(err error) error {
	if err == ErrLHPositionUndefined {
		return &lhPositionError{cause: r.fail(InvalidSchema, err.Error())}
	}
	return r.fail(Limit, err.Error())
}

// LHBit returns the L/H value of the next transmitted or receiver-inferred
// bit, with the same coordinate and later-run refusal as Reader.LHBit.
func (w *Writer) LHBit(symbol rune) (uint8, error) {
	logical, err := w.logicalPosition()
	if err != nil {
		return 0, err
	}
	inferred := w.wire.canonical && w.allowZero && (w.bounded && w.bits >= w.limit || w.wire.targetSet && w.bits >= w.wire.targetBits)
	if !inferred && w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans) {
		span, end, err := w.currentImplicitSpan()
		if err != nil {
			return 0, err
		}
		inferred = logical >= span.At && logical < end
	}
	position, err := w.lh.position(w.bits, inferred)
	if err != nil {
		// Canonical candidates defer operation wrapping. Preserve the field facts
		// in their cause so the final wrapper keeps this source refusal.
		kind := Limit
		if err == ErrLHPositionUndefined {
			kind = InvalidSchema
		}
		var cause error = &DecodeError{Kind: kind, Offset: w.bits, Path: w.path, Detail: err.Error()}
		if err == ErrLHPositionUndefined {
			cause = &lhPositionError{cause: cause}
		}
		return 0, w.operationFailure(cause, 0, kind)
	}
	return literalBit(symbol, position), nil
}

// inferredLHPositionPossible keeps transmitted literals on their physical
// coordinate (TS 24.007 V20.0.0 Annex B §B.1.2.2 rule B8). A canonical
// boundary or remaining replay span requires the full inferred-position check.
func (w *Writer) inferredLHPositionPossible() bool {
	return w.wire.canonical && w.allowZero && (w.bounded && w.bits >= w.limit || w.wire.targetSet && w.bits >= w.wire.targetBits) ||
		w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans)
}
