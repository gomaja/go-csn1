package runtime

// ReadUintWithFill reads a numeric field and reports its inferred zero suffix.
// It does not enable zero extension: generated code scopes SetZeroExtension
// to an authenticated suffix. TS 44.060 V19.0.0 §§11.2.5b, 11.2.34,
// 11.2.45 and 12.46 permit trailing bits to be assumed zero. Other uses of
// ReadUint retain their current semantics and diagnostic cost.
func (r *Reader) ReadUintWithFill(width int, source Source, field ...string) (uint64, error) {
	before := r.virtual
	value, err := r.ReadUint(width)
	if err == nil {
		r.recordFieldFill(before, source, field)
	}
	return value, err
}

// ReadBitStringWithFill is the bit-string counterpart of ReadUintWithFill.
func (r *Reader) ReadBitStringWithFill(width int, source Source, field ...string) (BitString, error) {
	before := r.virtual
	value, err := r.ReadBitString(width)
	if err == nil {
		r.recordFieldFill(before, source, field)
	}
	return value, err
}

// ExpectWithFill reports inferred zeros used by a successfully matched source
// literal. As with the field readers, zero extension must be enabled explicitly.
func (r *Reader) ExpectWithFill(pattern string, source Source, field ...string) error {
	before := r.virtual
	err := r.Expect(pattern)
	if err == nil {
		r.recordFieldFill(before, source, field)
	}
	return err
}

func (r *Reader) recordFieldFill(before int, source Source, field []string) {
	if r.virtual == before || r.collector == nil || r.collector.receiver == nil {
		return
	}
	// A single field read is contiguous in logical order, so its newly inferred
	// bits are a suffix of the existing last implicit span, even when a previous
	// field contributed to that same run. No span scan or duplicate fill state.
	count := r.virtual - before
	spans := r.wire.ImplicitSpans
	last := spans[len(spans)-1]
	r.recordReceiver("", last.At+last.Count-count, count, source, field)
}
