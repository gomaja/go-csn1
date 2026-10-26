package runtime

import "fmt"

// WithWireValue encodes one received value in the caller's writer, including
// its retained tail, and restores the containing wire state afterwards. The
// callback writes semantic fields through that same writer. Fresh values use
// the containing state directly. No intermediate byte encoding is constructed.
// TS 24.007 V20.0.0 §11.4.2 defines value-local received extent and inferred
// zeros; their offsets are translated once to the containing writer origin.
func (w *Writer) WithWireValue(wire WireInfo, encode func() error) error {
	if w == nil || encode == nil {
		return invalidValueWriter()
	}
	if !wire.sealed {
		return encode()
	}
	return w.withReceivedWireValue(wire, encode)
}
func invalidValueWriter() error {
	return &OperationError{Operation: "encode", Kind: InvalidValue, Location: Location{BitOffset: -1}, Detail: "nil writer or value encoder"}
}
func (w *Writer) withReceivedWireValue(wire WireInfo, encode func() error) error {
	if w.stateErr != nil {
		return w.stateErr
	}
	start := w.bits
	if start < 0 || start > maxBits || wire.TransmittedBits < 0 || wire.TransmittedBits > maxBits-start || wire.BitsConsumed < 0 || wire.BitsConsumed > wire.TransmittedBits || wire.ImplicitZeros < 0 || wire.ImplicitZeros > maxBits-start-wire.TransmittedBits {
		return w.operationFailure(fmt.Errorf("received value outside writer bit limit"), 0, Limit)
	}
	old := *w
	wire.TransmittedBits += start
	wire.BitsConsumed += start
	if wire.targetSet {
		if wire.targetBits < 0 || wire.targetBits > maxBits-start {
			return w.operationFailure(fmt.Errorf("canonical value outside writer bit limit"), 0, Limit)
		}
		wire.targetBits += start
	}
	if len(wire.ImplicitSpans) > 0 {
		spans := make([]ImplicitSpan, len(wire.ImplicitSpans))
		copy(spans, wire.ImplicitSpans)
		for i := range spans {
			if spans[i].At < 0 || spans[i].At > maxBits-start || spans[i].Count < 0 || spans[i].Count > maxBits-start-spans[i].At {
				return w.operationFailure(fmt.Errorf("inferred value span outside writer bit limit"), 0, Limit)
			}
			spans[i].At += start
		}
		wire.ImplicitSpans = spans
	}
	w.WithWire(wire)
	w.virtual = 0
	w.implicitSpan = 0
	w.spare = 0
	w.padding = 0
	w.terminal = 0
	defer func() {
		localVirtual := w.virtual
		w.wire = old.wire
		w.spare = old.spare
		w.padding = old.padding
		w.terminal = old.terminal
		w.spareCountIndex = old.spareCountIndex
		w.truncationIndex = old.truncationIndex
		w.implicitSpan = old.implicitSpan
		w.allowZero = old.allowZero
		w.limit = old.limit
		w.bounded = old.bounded
		w.stateErr = old.stateErr
		w.virtual = old.virtual + localVirtual
		// An enclosing received image may contain the same inferred spans. Their
		// logical prefix has now been consumed by the value's own state once.
		for w.implicitSpan < len(w.wire.ImplicitSpans) {
			span := w.wire.ImplicitSpans[w.implicitSpan]
			if span.Count < 0 || span.At > maxBits-span.Count || span.At+span.Count > w.bits+w.virtual {
				break
			}
			w.implicitSpan++
		}
	}()
	oldLimit, err := w.PushLimit(wire.TransmittedBits - start)
	if err != nil {
		return err
	}
	if err := encode(); err != nil {
		return err
	}
	w.bits -= start
	w.wire.BitsConsumed -= start
	w.wire.TransmittedBits -= start
	validationErr := w.validateValueSemantic(false)
	w.bits += start
	w.wire.BitsConsumed += start
	w.wire.TransmittedBits += start
	if validationErr != nil {
		return w.operationFailure(validationErr, 0, InvalidValue)
	}
	if err := w.WriteBitString(wire.Tail, wire.Tail.BitLength); err != nil {
		return err
	}
	return w.PopLimit(oldLimit)
}
