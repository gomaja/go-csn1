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
	fail := func(detail string) (WireInfo, error) {
		return WireInfo{}, operationFailure(fmt.Errorf("%s", detail), "decode", strings.Join(r.path, "/"), r.pos, 0, InvalidValue)
	}
	if r.steps != start.steps || start.pos > r.pos || before.ImplicitZeros < 0 || before.ImplicitZeros > r.virtual {
		return fail("value origin does not belong to this reader")
	}
	consumed := r.pos - start.pos
	wire := r.Wire()
	if len(before.Spare) > len(wire.Spare) || len(before.Padding) > len(wire.Padding) || len(before.Terminal) > len(wire.Terminal) {
		return fail("value wire prefix exceeds received state")
	}
	wire.Spare = append([]BitString(nil), wire.Spare[len(before.Spare):]...)
	wire.Padding = append([]BitString(nil), wire.Padding[len(before.Padding):]...)
	wire.Terminal = append([]BitString(nil), wire.Terminal[len(before.Terminal):]...)
	trunc := map[string][]int{}
	for key, values := range wire.TruncatedAt {
		n := len(before.TruncatedAt[key])
		if n > len(values) {
			return fail("value truncation prefix exceeds received state")
		}
		if n < len(values) {
			trunc[key] = append([]int(nil), values[n:]...)
		}
	}
	wire.TruncatedAt = trunc
	counts := map[string][]int{}
	for key, values := range wire.SpareCounts {
		n := len(before.SpareCounts[key])
		if n > len(values) {
			return fail("value spare prefix exceeds received state")
		}
		if n < len(values) {
			counts[key] = append([]int(nil), values[n:]...)
		}
	}
	wire.SpareCounts = counts
	wire.ImplicitZeros -= before.ImplicitZeros
	spans := []ImplicitSpan{}
	begin := start.pos + before.ImplicitZeros
	for _, span := range wire.ImplicitSpans {
		if span.At < 0 || span.At > maxBits || span.Count < 0 || span.Count > maxBits-span.At {
			return fail("invalid inferred span in received value")
		}
		end := span.At + span.Count
		if end > begin {
			at := max(span.At, begin)
			spans = append(spans, ImplicitSpan{At: at - begin, Count: end - at})
		}
	}
	wire.ImplicitSpans = spans
	tail := BitString{}
	if includeTail && r.BoundedRemaining() >= 0 {
		tail = r.Tail()
	}
	if tail.BitLength > maxBits-consumed {
		return fail("value tail exceeds runtime bit limit")
	}
	image, err := start.ReadBitString(consumed + tail.BitLength)
	if err != nil {
		return WireInfo{}, err
	}
	wire = Seal(struct{}{}, image.Bytes, consumed, tail, wire)
	wire.TransmittedBits = consumed + tail.BitLength
	return wire, nil
}
