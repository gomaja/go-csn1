package runtime

import "fmt"

// WithCompleteWireValue encodes an independently complete value. A received
// value replays its own extent through WithWireValue. A fresh value must fill bits
// with transmitted bits using its own layout state, while retaining the caller's
// physical bound and canonical target. Its explicit CanonicalTo target, if
// present, must equal bits. Parent layout state is restored on success or error.
// Ordinary nested components continue to use WithWireValue and inherit layout.
// TS 44.006 V19.0.0 §§5.2, 6.4a, 8.8.3 require the complete SACCH Bter
// carrier to contain all N201 octets, independently of a received parent image.
func (w *Writer) WithCompleteWireValue(wire WireInfo, bits int, encode func() error) error {
	if w == nil || encode == nil {
		return invalidValueWriter()
	}
	if wire.sealed {
		return w.WithWireValue(wire, encode)
	}
	if w.stateErr != nil {
		return w.stateErr
	}
	if wire.targetSet && wire.targetBits != bits {
		return w.operationFailure(fmt.Errorf("complete value requires canonical target %d bits", bits), 0, InvalidValue)
	}
	old := *w
	oldBound, err := w.PushLimit(bits)
	if err != nil {
		return err
	}
	defer func() {
		w.wire = old.wire
		w.spare, w.padding, w.terminal = old.spare, old.padding, old.terminal
		w.spareCountIndex, w.truncationIndex = old.spareCountIndex, old.truncationIndex
		w.implicitSpan, w.virtual = old.implicitSpan, old.virtual
		// The child transmitted its own bits; parent inferred spans that now
		// lie behind the logical position must not hide later parent spans.
		for w.implicitSpan < len(w.wire.ImplicitSpans) {
			_, end, spanErr := w.currentImplicitSpan()
			position, positionErr := w.logicalPosition()
			if spanErr != nil || positionErr != nil || end > position {
				break
			}
			w.implicitSpan++
		}
		w.allowZero = old.allowZero
		w.limit, w.bounded = old.limit, old.bounded
		w.stateErr, w.stateErrPath = old.stateErr, old.stateErrPath
	}()
	if wire.targetSet {
		wire.targetBits = w.BoundEndOr(0)
	}
	wire.canonical = wire.canonical || old.wire.canonical
	wire.rawErrors = wire.rawErrors || old.wire.rawErrors
	if old.wire.canonical && old.wire.targetSet && (!wire.targetSet || old.wire.targetBits < wire.targetBits) {
		wire.targetSet, wire.targetBits = true, old.wire.targetBits
	}
	w.WithWire(wire)
	w.spare, w.padding, w.terminal = 0, 0, 0
	w.implicitSpan = 0
	w.allowZero = false
	if err := encode(); err != nil {
		return err
	}
	if w.virtual != old.virtual {
		return w.operationFailure(fmt.Errorf("complete value contains inferred rather than transmitted bits"), 0, InvalidValue)
	}
	return w.PopLimit(oldBound)
}
