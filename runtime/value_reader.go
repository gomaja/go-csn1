package runtime

import (
	"fmt"
	"strings"
)

// ValueWire captures only the received facts added since start, a Fork of this
// reader taken after before was obtained from Wire. It never includes bytes
// beyond the active enclosing limit. start is consumed while capturing the
// image; the original reader position remains at its semantic boundary.
func (r *Reader) ValueWire(start *Reader, before WireInfo, includeTail bool) (WireInfo, error) {
	if err := r.Check(); err != nil {
		return WireInfo{}, err
	}
	if err := start.Check(); err != nil {
		return WireInfo{}, err
	}
	if r.steps != start.steps || start.pos > r.pos || before.ImplicitZeros < 0 || before.ImplicitZeros > r.virtual {
		return r.valueWireFailure("value origin does not belong to this reader")
	}
	consumed := r.pos - start.pos
	wire := r.Wire()
	if len(before.Spare) > len(wire.Spare) || len(before.Padding) > len(wire.Padding) || len(before.Terminal) > len(wire.Terminal) {
		return r.valueWireFailure("value wire prefix exceeds received state")
	}
	wire.Spare = append([]BitString(nil), wire.Spare[len(before.Spare):]...)
	wire.Padding = append([]BitString(nil), wire.Padding[len(before.Padding):]...)
	wire.Terminal = append([]BitString(nil), wire.Terminal[len(before.Terminal):]...)
	trunc, ok := valueCountDelta(before.TruncatedAt, wire.TruncatedAt)
	if !ok {
		return r.valueWireFailure("value truncation prefix exceeds received state")
	}
	wire.TruncatedAt = trunc
	counts, ok := valueCountDelta(before.SpareCounts, wire.SpareCounts)
	if !ok {
		return r.valueWireFailure("value spare prefix exceeds received state")
	}
	wire.SpareCounts = counts
	wire.ImplicitZeros -= before.ImplicitZeros
	begin := start.pos + before.ImplicitZeros
	spans, ok := valueImplicitSpans(wire.ImplicitSpans, begin)
	if !ok {
		return r.valueWireFailure("invalid inferred span in received value")
	}
	wire.ImplicitSpans = spans
	tail := BitString{}
	if includeTail && r.BoundedRemaining() >= 0 {
		tail = r.Tail()
	}
	if tail.BitLength > maxBits-consumed {
		return r.valueWireFailure("value tail exceeds runtime bit limit")
	}
	image, err := start.ReadBitString(consumed + tail.BitLength)
	if err != nil {
		return WireInfo{}, err
	}
	wire = Seal(struct{}{}, image.Bytes, consumed, tail, wire)
	wire.TransmittedBits = consumed + tail.BitLength
	return wire, nil
}

// valueCountDelta shares the identical map-prefix copy used for both received
// count records, keeping their loops in one compiler body.
func valueCountDelta(before, after map[string][]int) (map[string][]int, bool) {
	out := map[string][]int{}
	for key, values := range after {
		n := len(before[key])
		if n > len(values) {
			return nil, false
		}
		if n < len(values) {
			out[key] = append([]int(nil), values[n:]...)
		}
	}
	return out, true
}

// Failed value captures share one formatting body; successful captures never
// enter it.
//
//go:noinline
func (r *Reader) valueWireFailure(detail string) (WireInfo, error) {
	return WireInfo{}, operationFailure(fmt.Errorf("%s", detail), "decode", strings.Join(r.path, "/"), r.pos, 0, InvalidValue)
}

// valueImplicitSpans clips received spans with the same checked arithmetic as
// the enclosing value capture, without expanding that compiler body.
func valueImplicitSpans(received []ImplicitSpan, begin int) ([]ImplicitSpan, bool) {
	spans := []ImplicitSpan{}
	for _, span := range received {
		if span.At < 0 || span.At > maxBits || span.Count < 0 || span.Count > maxBits-span.At {
			return nil, false
		}
		end := span.At + span.Count
		if end > begin {
			at := max(span.At, begin)
			spans = append(spans, ImplicitSpan{At: at - begin, Count: end - at})
		}
	}
	return spans, true
}
