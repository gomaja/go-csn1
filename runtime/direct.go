package runtime

import (
	"errors"
	"fmt"
	"strings"
)

// WireInfo retains non-semantic received bits and the actual truncation point.
// TS 24.008 V20.1.0 §§10.5.1.7, 10.5.5.12a permit spare bits and
// implicit zero extension; TS 24.007 V20.0.0 Annex B §B.1.2.2 and
// TS 44.060 V19.0.0 §11 define the octet-aligned 0x2b L/H pattern.
type WireInfo struct {
	BitsConsumed    int
	TransmittedBits int
	Tail            BitString
	Spare           []BitString
	Padding         []BitString
	Terminal        []BitString
	// TruncatedAt lists, per truncated concatenation (TS 44.060 V19.0.0
	// §11.1.4.4) and in decode order of its instances, the number of
	// components each instance received, or -1 when it was complete. One
	// entry per instance keeps recursive and repeated instances apart.
	TruncatedAt   map[string][]int
	SpareCounts   map[string][]int
	ImplicitZeros int
	ImplicitSpans []ImplicitSpan
	sealed        bool
	canonical     bool
	targetSet     bool
	rawErrors     bool
	targetBits    int
}

// ImplicitSpan records a receiver-inferred zero run in logical bit order.
// TS 24.008 V20.1.0 §§10.5.1.7, 10.5.5.12a allow zero extension without
// transmission, including within a bounded capability before later bits.
type ImplicitSpan struct{ At, Count int }

// IsZero reports whether a field omitted by a transmitted truncation remains
// semantically unedited, ignoring nested wire state. TS 44.018 V19.0.0 §8.9
// permits omission only as an ordered prefix.
func IsZero[T any](value T) bool { var zero T; return SemanticallyEqual(value, zero) }

// Seal records the transmitted boundary separately from inferred zero bits.
// TS 24.008 V20.1.0 §§10.5.1.7, 10.5.5.12a permit receiver-side zero
// extension; the received bytes themselves are never retained for replay.
func Seal[T any](_ T, raw []byte, consumed int, tail BitString, wire WireInfo) WireInfo {
	wire.BitsConsumed = consumed
	wire.TransmittedBits = maxBits + 8
	if len(raw) <= maxBits/8 {
		wire.TransmittedBits = len(raw) * 8
	}
	wire.Tail = tail
	wire.sealed = true
	return wire
}

// Reader is an MSB-first, bounded CSN.1 bit reader. A fork shares the work
// budget but isolates position and expression bindings for choice resolution.
// A reader borrowed from ChoiceErrors must not be copied by value; use Fork.
type Reader struct {
	data                                           []byte
	pos, end, virtual                              int
	lh                                             lhPositionState
	boundDepth                                     int
	vars                                           map[string]uint64
	steps                                          *int
	path                                           []string
	depth                                          int
	wire                                           WireInfo
	collector                                      *Collector
	trial                                          *choiceTrial
	observationDropped                             uint64
	observationStart                               int32
	fieldWidthLo                                   uint16
	fieldWidthHi                                   uint8
	allowZero, varsShared, pathShared              bool
	implicitShared, truncShared, spareCountsShared bool
}

// readerStorage keeps the shared work counter and initial path in the same
// allocation as the reader. Forks retain pointers into this storage.
type readerStorage struct {
	reader Reader
	steps  int
	path   [16]string
}

// NewReader constructs a reader whose state can remain on the caller's stack.
// Unused bindings and truncation maps are initialized on first use.
func NewReader(data []byte) *Reader {
	end := 0
	if len(data) <= maxBits/8 {
		end = len(data) * 8
	}
	steps := 0
	return &Reader{observationStart: -1, data: data, end: end, path: make([]string, 0, 16), steps: &steps}
}

// NewChoiceReader co-locates state retained by generated speculative choices.
// Such choices retain the reader beyond a leaf call; sharing one allocation
// avoids separate escaped work-counter and path allocations.
// TS 24.007 V20.0.0 Annex B §B.1.1.4 defines the alternatives.
func NewChoiceReader(data []byte, collector *Collector) *Reader {
	storage := new(readerStorage)
	r := &storage.reader
	end := 0
	if len(data) <= maxBits/8 {
		end = len(data) * 8
	}
	*r = Reader{observationStart: -1, data: data, end: end, path: storage.path[:0], steps: &storage.steps}
	r.attachCollector(collector)
	return r
}

func CheckInput(data []byte) error {
	if len(data) > maxBits/8 {
		return &DecodeError{Kind: Limit, Offset: 0, Detail: "input exceeds maximum bit length"}
	}
	return nil
}

// InputBits is the bounded input length for DecodeError offsets.
func InputBits(data []byte) int {
	if len(data) > maxBits/8 {
		return maxBits
	}
	return len(data) * 8
}

func (r *Reader) Check() error {
	if r == nil {
		return &DecodeError{Kind: InvalidValue, Offset: -1, Detail: "nil reader"}
	}
	if err := CheckInput(r.data); err != nil {
		return err
	}
	if r.pos < 0 || r.pos > maxBits {
		return r.fail(InvalidValue, "invalid reader position")
	}
	if r.steps == nil || *r.steps < 0 || r.pos < 0 || r.end < r.pos || r.end > len(r.data)*8 || r.virtual < 0 || r.virtual > maxBits-r.pos || r.boundDepth < 0 || r.boundDepth > 128 || r.depth < 0 || r.depth > 128 {
		return r.fail(InvalidValue, "invalid reader state")
	}
	return nil
}

func (r *Reader) Position() int {
	if r == nil {
		return -1
	}
	return r.pos
}
func (r *Reader) Remaining() int {
	if r.Check() != nil {
		return -1
	}
	return r.end - r.pos
}
func (r *Reader) BoundedRemaining() int {
	if r.boundDepth == 0 {
		return -1
	}
	return r.Remaining()
}
func (r *Reader) VirtualBits() int { return r.virtual }
func (r *Reader) Wire() WireInfo {
	r.ensureTruncations()
	r.wire.ImplicitZeros = r.virtual
	return r.wire
}
func (r *Reader) Tail() BitString {
	if r.Check() != nil {
		return BitString{}
	}
	return bitsAt(r.data, r.pos, r.end-r.pos)
}
func (r *Reader) SetZeroExtension(enabled bool) { r.allowZero = enabled }
func (r *Reader) ZeroExtension() bool           { return r.allowZero }

// Set normalizes caller-supplied names under TS 24.007 V20.0.0 Annex B
// §B.1.1.5 before applying the same binding update as generated decoders.
func (r *Reader) Set(name string, value uint64) {
	r.SetCanonical(key(name), value)
}
func (r *Reader) Eval(expression string) (int, error) {
	v, err := evalReader(expression, r)
	if err != nil {
		return 0, r.fail(InvalidSchema, err.Error())
	}
	return v, nil
}
func (r *Reader) fail(kind ErrorKind, detail string) error {
	return r.failAt(kind, r.pos, detail)
}
func (r *Reader) failAt(kind ErrorKind, offset int, detail string) error {
	if r.trial != nil {
		return r.deferredFailure(kind, offset, detail)
	}
	path := strings.Join(r.path, "/")
	width := int(r.fieldWidthHi)<<16 | int(r.fieldWidthLo)
	if width != 0 {
		return &fieldDecodeError{DecodeError: DecodeError{Kind: kind, Offset: offset, Path: path, Detail: detail}, width: width - 1}
	}
	return &DecodeError{Kind: kind, Offset: offset, Path: path, Detail: detail}
}
func (r *Reader) Error(kind ErrorKind, detail string) error { return r.fail(kind, detail) }
func (r *Reader) Enter(name string) error {
	if err := r.Check(); err != nil {
		return err
	}
	if *r.steps >= maxBits || r.depth >= 128 {
		return r.fail(Limit, "decoder work or nesting limit")
	}
	*r.steps++
	r.depth++
	if r.pathShared {
		r.ownPath()
	}
	r.path = append(r.path, name)
	return nil
}
func (r *Reader) Leave() {
	// A trial may shorten its view of the inherited prefix, but must detach
	// before a subsequent Enter can replace any of that borrowed prefix.
	// TS 24.007 V20.0.0 Annex B §B.1.1.3 preserves concatenation order.
	if r.trial != nil && len(r.path) <= r.trial.pathFloor {
		r.pathShared = true
	}
	if r.depth > 0 {
		r.depth--
	}
	if len(r.path) > 0 {
		r.path = r.path[:len(r.path)-1]
	}
}
func (r *Reader) Fork() *Reader {
	var fork Reader
	r.forkInto(&fork)
	return &fork
}

// Keep the public snapshot wrapper small enough for caller inlining. The
// copied reader may stay on the caller's stack; trial snapshots still detach
// pooled bindings and borrowed paths before escaping their choice scope.
// TS 24.007 V20.0.0 Annex B §§B.1.1.3–B.1.1.5.
func (r *Reader) forkInto(fork *Reader) {
	r.ensureTruncations()
	priorVars, priorPath := r.varsShared, r.pathShared
	r.varsShared, r.pathShared, r.implicitShared, r.truncShared, r.spareCountsShared = true, true, true, true, true
	*fork = *r
	if r.trial != nil {
		fork.freezeChoiceSnapshot()
		r.varsShared, r.pathShared = priorVars, priorPath
	}
	if r.collector != nil {
		fork.observationStart = int32(len(r.collector.observations))
		fork.observationDropped = r.collector.dropped
	}
	fork.wire.Spare, fork.wire.Padding, fork.wire.Terminal = clipWireBits(fork.wire.Spare, fork.wire.Padding, fork.wire.Terminal)
}
func (r *Reader) Commit(other *Reader) {
	if r.trial != nil || other.trial != nil {
		r.checkTrialIdentity()
		other.checkTrialIdentity()
		r.commitChoice(other)
		return
	}
	if r.collector != nil {
		r.commitCollected(other)
		return
	}
	*r = *other
}

func (r *Reader) bit() (uint8, bool, error) {
	if err := r.Check(); err != nil {
		return 0, false, err
	}
	if r.pos < 0 || r.virtual < 0 || r.pos > maxBits || r.virtual >= maxBits-r.pos {
		return 0, false, r.fail(Limit, "decoder bit limit exceeded")
	}
	if r.pos < r.end {
		b := (r.data[r.pos/8] >> (7 - uint(r.pos%8))) & 1
		r.pos++
		return b, true, nil
	}
	if r.allowZero {
		at := r.pos + r.virtual
		if r.implicitShared {
			r.wire.ImplicitSpans = append([]ImplicitSpan(nil), r.wire.ImplicitSpans...)
			r.implicitShared = false
		}
		spans := r.wire.ImplicitSpans
		if len(spans) != 0 {
			last := spans[len(spans)-1]
			if last.Count <= 0 || last.Count >= maxBits || last.At < 0 || last.At > maxBits-last.Count {
				return 0, false, r.fail(InvalidValue, "invalid receiver-inferred span")
			}
		}
		if len(spans) == 0 || spans[len(spans)-1].At+spans[len(spans)-1].Count != at {
			r.wire.ImplicitSpans = append(spans, ImplicitSpan{At: at})
		}
		r.wire.ImplicitSpans[len(r.wire.ImplicitSpans)-1].Count++
		r.virtual++
		r.lh.infer(r.pos, 1)
		return 0, false, nil
	}
	return 0, false, r.fail(Truncated, "required bit absent")
}
func (r *Reader) ReadUint(width int) (uint64, error) {
	if width < 0 || width > 64 {
		return 0, r.fail(Limit, "numeric width outside 0..64")
	}
	priorHi, priorLo := r.fieldWidthHi, r.fieldWidthLo
	r.fieldWidthHi = 0
	r.fieldWidthLo = uint16(width + 1)
	var value uint64
	for i := 0; i < width; i++ {
		b, _, err := r.bit()
		if err != nil {
			r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
			return 0, err
		}
		value = (value << 1) | uint64(b)
	}
	r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
	return value, nil
}
func (r *Reader) ReadBitString(width int) (BitString, error) {
	if width < 0 || width > maxBits || width > r.Remaining() && !r.allowZero {
		priorHi, priorLo := r.fieldWidthHi, r.fieldWidthLo
		encoded := 1
		if width >= 0 && width <= maxBits {
			encoded = width + 1
		}
		r.fieldWidthHi = uint8(encoded >> 16)
		r.fieldWidthLo = uint16(encoded & 65535)
		err := r.failAt(Truncated, r.end, "bit string exceeds input")
		r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
		return BitString{}, err
	}
	priorHi, priorLo := r.fieldWidthHi, r.fieldWidthLo
	encoded := width + 1
	r.fieldWidthHi = uint8(encoded >> 16)
	r.fieldWidthLo = uint16(encoded & 65535)
	out := BitString{Bytes: make([]byte, (width+7)/8), BitLength: width}
	for i := 0; i < width; i++ {
		b, _, err := r.bit()
		if err != nil {
			r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
			return BitString{}, err
		}
		out.Bytes[i/8] |= b << (7 - uint(i%8))
	}
	r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
	return out, nil
}
func (r *Reader) Expect(pattern string) error {
	if err := r.Check(); err != nil {
		return err
	}
	priorHi, priorLo := r.fieldWidthHi, r.fieldWidthLo
	encoded := min(len(pattern), maxBits) + 1
	r.fieldWidthHi = uint8(encoded >> 16)
	r.fieldWidthLo = uint16(encoded & 65535)
	start := r.pos
	for _, symbol := range pattern {
		expected := literalBit(symbol, r.pos)
		if (symbol == 'L' || symbol == 'H') && r.pos >= r.end && r.allowZero {
			value, err := r.LHBit(symbol)
			if err != nil {
				r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
				return err
			}
			expected = value
		}
		bit, _, err := r.bit()
		if err != nil {
			r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
			return err
		}
		if bit != expected {
			detail := ""
			if r.trial == nil {
				detail = fmt.Sprintf("literal %c differs", symbol)
			}
			err := r.failAt(InvalidValue, start, detail)
			if f, ok := err.(*deferredFailure); ok {
				f.cause.Detail = literalDifference(symbol)
			}
			r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
			return err
		}
	}
	r.fieldWidthHi, r.fieldWidthLo = priorHi, priorLo
	return nil
}

// Matches reports whether pattern matches the next received bits without
// consuming them. It returns false where LookaheadLiteral returns an error,
// which is only the reader's existing failure; use LookaheadLiteral to see it.
func (r *Reader) Matches(pattern string) bool {
	matched, err := r.LookaheadLiteral(pattern)
	return err == nil && matched
}

// LookaheadLiteral looks ahead within the received input, whose L/H positions are always defined,
// without consuming bits. Past the received end it returns false even with zero extension.
func (r *Reader) LookaheadLiteral(pattern string) (bool, error) {
	if err := r.Check(); err != nil {
		return false, err
	}
	if len(pattern) > r.end-r.pos {
		return false, nil
	}
	for i, symbol := range pattern {
		pos := r.pos + i
		b := (r.data[pos/8] >> (7 - uint(pos%8))) & 1
		if b != literalBit(symbol, pos) {
			return false, nil
		}
	}
	return true, nil
}
func (r *Reader) ReadSpare() (BitString, error) {
	value, err := r.ReadLayoutBit()
	if err == nil {
		r.wire.Spare = append(r.wire.Spare, value)
	}
	return value, err
}

// ReadLayoutBit reads a bit retained by the current generated value's Wire.
// It leaves the root spare stream untouched, so repeated capability entries
// retain their own legacy indications (TS 24.008 V20.1.0 §10.5.5.12a).
func (r *Reader) ReadLayoutBit() (BitString, error) {
	start := r.pos
	_, present, err := r.bit()
	if err != nil {
		return BitString{}, err
	}
	var value BitString
	if present {
		value = bitsAt(r.data, start, 1)
	}
	return value, nil
}
func (r *Reader) ReadPadding() BitString {
	if r.Check() != nil {
		return BitString{}
	}
	value := bitsAt(r.data, r.pos, r.end-r.pos)
	r.pos = r.end
	r.wire.Padding = append(r.wire.Padding, value)
	return value
}

// ReadIgnored retains a nonsemantic "no string" extension arm. TS 44.018
// V19.0.0 §9.1.55 and TS 44.060 V19.0.0 §12.24 print this construct;
// pycrate_csn1/csnobj.py consumes the remaining bits in its bounded scope.
func (r *Reader) ReadIgnored() BitString { return r.ReadPadding() }

// ReadIgnoredFixed retains the exact nonsemantic bit in wire state while
// leaving later fields available. TS 44.018 V19.0.0 §10.5.2.33b uses a
// one-bit "no string" release field; pycrate consumes one bit here.
func (r *Reader) ReadIgnoredFixed(width int) (BitString, error) {
	value, err := r.ReadBitString(width)
	if err != nil {
		return BitString{}, err
	}
	r.wire.Padding = append(r.wire.Padding, value)
	return value, nil
}
func (r *Reader) PushLimit(width int) (int, error) {
	if err := r.Check(); err != nil {
		return 0, err
	}
	if r.boundDepth >= 128 {
		return 0, r.fail(Limit, "fixed-value nesting limit")
	}
	if width < 0 || width > r.Remaining() {
		return 0, r.failAt(Truncated, r.end, "length exceeds enclosing input")
	}
	old := r.end
	r.end = r.pos + width
	r.boundDepth++
	return old, nil
}
func (r *Reader) PopLimit(old int) error {
	if err := r.Check(); err != nil {
		return err
	}
	if r.boundDepth == 0 {
		return r.fail(InvalidValue, "no enclosing length limit")
	}
	if r.pos != r.end {
		return r.fail(InvalidValue, "length-delimited content leaves unparsed bits")
	}
	if old < r.end || old > len(r.data)*8 {
		return r.fail(InvalidValue, "restored length limit outside input")
	}
	r.end = old
	r.boundDepth--
	return nil
}

// BeginTruncation reserves the truncation entry of a truncated
// concatenation instance on entry, so instances are listed in the order an
// encoder meets them, outer before nested. The entry stays -1 unless
// RecordTruncation marks where the instance ended.
func (r *Reader) BeginTruncation(path string) int {
	r.ownTruncations()
	slot := len(r.wire.TruncatedAt[path])
	r.wire.TruncatedAt[path] = append(r.wire.TruncatedAt[path], -1)
	return slot
}

// RecordTruncation records that the instance in slot received child
// components before its input ended.
func (r *Reader) RecordTruncation(path string, slot, child int) {
	r.ownTruncations()
	if points := r.wire.TruncatedAt[path]; slot >= 0 && slot < len(points) {
		points[slot] = child
	}
}

// ensureTruncations supplies the public non-nil truncation map lazily.
func (r *Reader) ensureTruncations() {
	if r.wire.TruncatedAt == nil {
		r.wire.TruncatedAt = make(map[string][]int)
	}
}

// ownTruncations copies entries shared with a fork before they change.
func (r *Reader) ownTruncations() {
	if r.truncShared || r.wire.TruncatedAt == nil {
		copy := make(map[string][]int, len(r.wire.TruncatedAt))
		for k, v := range r.wire.TruncatedAt {
			copy[k] = append([]int(nil), v...)
		}
		r.wire.TruncatedAt = copy
		r.truncShared = false
	}
}

// RecordTerminal preserves the nonsemantic zero-length stop record printed
// in TS 44.018 V19.0.0 §10.5.2.37h table 10.5.2.37h.1.
func (r *Reader) RecordTerminal(start int) error {
	if err := r.Check(); err != nil {
		return err
	}
	if start < 0 || start > r.pos || r.pos > r.end {
		return r.fail(InvalidValue, "terminal start outside input")
	}
	r.wire.Terminal = append(r.wire.Terminal, bitsAt(r.data, start, r.pos-start))
	return nil
}
func (r *Reader) RecordSpareCount(path string, count int) {
	if r.spareCountsShared {
		copy := make(map[string][]int, len(r.wire.SpareCounts))
		for k, v := range r.wire.SpareCounts {
			copy[k] = append([]int(nil), v...)
		}
		r.wire.SpareCounts = copy
		r.spareCountsShared = false
	}
	if r.wire.SpareCounts == nil {
		r.wire.SpareCounts = make(map[string][]int)
	}
	r.wire.SpareCounts[path] = append(r.wire.SpareCounts[path], count)
}

// Writer uses the same MSB-first bit order and expression rules as Reader.
type Writer struct {
	bytes                    []byte
	bits, virtual            int
	lh                       lhPositionState
	limit                    int
	bounded                  bool
	vars                     map[string]uint64
	path                     string
	depth                    int
	spare, padding, terminal int
	spareCountIndex          map[string]int
	truncationIndex          map[string]int
	wire                     WireInfo
	implicitSpan             int
	stateErr                 error
	stateErrPath             string
	allowZero                bool
}

func NewWriter() *Writer { return &Writer{vars: make(map[string]uint64)} }
func (w *Writer) WithWire(wire WireInfo) {
	w.wire = wire
	w.spareCountIndex = make(map[string]int)
	w.truncationIndex = make(map[string]int)
	w.stateErr = nil
	w.stateErrPath = ""
}

// SetZeroExtension mirrors a printed TS 24.008 V20.1.0 §10.5.5.12a
// capability reference: missing trailing bits are receiver-inferred zeros.
func (w *Writer) SetZeroExtension(allow bool) (previous bool) {
	previous, w.allowZero = w.allowZero, allow
	return previous
}

// CanonicalTargetReached marks a complete component boundary for a
// TS 44.018 V19.0.0 §8.9 truncated concatenation.
func (w *Writer) CanonicalTargetReached() bool {
	return w.wire.canonical && w.wire.targetSet && w.bits >= w.wire.targetBits
}

// TruncationReached marks where a fresh value ends a truncated
// concatenation. TS 44.060 V19.0.0 §11.1.4.4 and TS 44.018 V19.0.0 §8.9
// make "{ <a> <b> <c> } //" any component prefix. Inside a length-delimited
// value the prefix must fill that length exactly, so the writer stops where
// the reader does, at the exhausted bound, and keeps the typed length
// (§12.24 Extension Length). The omitted components must be zero: "The
// receiver shall assume the value zero for any truncated bit" (§12.24).
// Outside a bound, the canonical target applies. A decoded value follows its
// recorded truncation point instead.
func (w *Writer) TruncationReached() bool {
	if w.wire.sealed {
		return false
	}
	if w.bounded {
		return w.bits >= w.limit
	}
	return w.CanonicalTargetReached()
}

// OmittedFieldError reports a nonzero field after the point where a truncated
// concatenation ends. Only zero components may be omitted (TS 44.060 V19.0.0
// §12.24: "The receiver shall assume the value zero for any truncated bit";
// TS 44.018 V19.0.0 §8.9). The bound is the one TruncationReached applied,
// or the decoded value's recorded truncation point.
func (w *Writer) OmittedFieldError(field string) error {
	detail := "nonzero field omitted from a truncated concatenation; only zero components may be omitted"
	switch {
	case !w.wire.sealed && w.bounded && w.bits >= w.limit:
		return w.operationFailure(w.boundError(LengthBound, w.limit, field, detail), 0, InvalidValue)
	case w.CanonicalTargetReached():
		return w.operationFailure(w.boundError(CanonicalTarget, w.wire.targetBits, field, detail), 0, InvalidValue)
	default:
		return w.operationFailure(w.boundError(ReceivedTruncation, w.bits, field, detail), 0, InvalidValue)
	}
}

// LengthBoundError reports content that cannot fill or fit its enclosing
// length-delimited or fixed-size value. Generated code calls it only inside
// such a value; without one there is no bound to report.
func (w *Writer) LengthBoundError(detail string) error {
	if !w.bounded {
		return w.operationFailure(fmt.Errorf("%s: no enclosing length-delimited value", detail), 0, InvalidValue)
	}
	return w.operationFailure(w.boundError(LengthBound, w.limit, "", detail), 0, InvalidValue)
}

// LengthFieldError reports an alternative that conflicts with the octet
// length printed before it. TS 44.018 V19.0.0 §10.5.2.16: Length of
// frequency parameters is "the number of octets occupied by the frequency
// parameters, before time field. If this length is 0, the frequency
// parameters, before time is not present." The bound starts at the
// alternative, so Limit is the current position plus that many octets.
func (w *Writer) LengthFieldError(field string, octets int, detail string) error {
	if w.bits < 0 || w.bits > maxBits || octets < 0 || octets > (maxBits-w.bits)/8 {
		return w.operationFailure(fmt.Errorf("length field of %d octets outside encoded bit limit", octets), 0, InvalidValue)
	}
	return w.operationFailure(w.boundError(LengthBound, w.bits+octets*8, field, detail), 0, InvalidValue)
}

// boundError builds a BoundError. Limit and Position are always bit offsets
// from the start of the encoding; an invalid offset is an internal error.
func (w *Writer) boundError(kind BoundKind, limit int, field, detail string) error {
	if limit < 0 || limit > maxBits || w.bits < 0 || w.bits > maxBits {
		return w.operationFailure(fmt.Errorf("invalid %s bound %d: %s", kind, limit, detail), 0, InvalidValue)
	}
	if field == "" {
		field = w.path
		if i := strings.LastIndexByte(w.path, '/'); i >= 0 {
			field = strings.TrimPrefix(w.path[i:], "/")
		}
	}
	return w.operationFailure(&BoundError{Kind: kind, Path: w.path, Field: field, Detail: detail, Limit: limit, Position: w.bits}, 0, InvalidValue)
}
func (w *Writer) Canonical() bool { return w.wire.canonical }

// Truncation consumes the next received truncation entry for path, in the
// order Reader.BeginTruncation listed the instances, and reports where that
// instance ended when it was truncated.
func (w *Writer) Truncation(path string) (int, bool) {
	points := w.wire.TruncatedAt[path]
	i := w.truncationIndex[path]
	if i >= len(points) {
		return 0, false
	}
	w.truncationIndex[path] = i + 1
	return points[i], points[i] >= 0
}
func (w *Writer) SpareCount(path string) (int, bool) {
	counts := w.wire.SpareCounts[path]
	i := w.spareCountIndex[path]
	if i >= len(counts) {
		return 0, false
	}
	if counts[i] < 0 || counts[i] > maxBits {
		w.stateErr = fmt.Errorf("spare repeat count outside bit limit")
		w.stateErrPath = path
		return 0, false
	}
	w.spareCountIndex[path] = i + 1
	return counts[i], true
}
func (w *Writer) Position() int {
	if w == nil {
		return -1
	}
	return w.bits
}
func (w *Writer) RemainingLimit() int {
	if !w.bounded {
		return -1
	}
	if w.bits < 0 || w.bits > maxBits || w.limit < w.bits || w.limit > maxBits {
		return -1
	}
	return w.limit - w.bits
}

// SpareFillCount completes a fresh <spare bits> run to its enclosing
// value limit or octet boundary (TS 24.007 V20.0.0 Annex B.1.2.1 Rule B7).
func (w *Writer) SpareFillCount() int {
	if remaining := w.RemainingLimit(); remaining >= 0 {
		return remaining
	}
	return (8 - w.bits%8) % 8
}
func (w *Writer) BoundEndOr(unbounded int) int {
	if w.bounded {
		return w.limit
	}
	return unbounded
}
func (w *Writer) PushLimit(width int) (int, error) {
	if w.bits < 0 || w.bits > maxBits || width < 0 || width > maxBits-w.bits || w.bounded && (w.limit < w.bits || w.limit > maxBits) {
		return 0, w.operationFailure(fmt.Errorf("fixed value outside encoded bit limit"), 0, InvalidValue)
	}
	if w.bounded && width > w.limit-w.bits {
		return 0, w.operationFailure(w.boundError(LengthBound, w.limit, "", "length exceeds the enclosing length-delimited value"), 0, InvalidValue)
	}
	old := -1
	if w.bounded {
		old = w.limit
	}
	w.limit = w.bits + width
	w.bounded = true
	return old, nil
}
func (w *Writer) PopLimit(old int) error {
	if !w.bounded || old < -1 || old > maxBits {
		return w.operationFailure(fmt.Errorf("invalid enclosing fixed value"), 0, InvalidValue)
	}
	if w.bits != w.limit {
		return w.operationFailure(w.boundError(LengthBound, w.limit, "", "content ends before the length-delimited value"), 0, InvalidValue)
	}
	w.limit = old
	w.bounded = old >= 0
	return nil
}
func (w *Writer) Set(name string, v uint64)           { w.vars[key(name)] = v }
func (w *Writer) Eval(expression string) (int, error) { return eval(expression, w.vars) }
func (w *Writer) Enter(name string) error {
	if w.depth >= 128 {
		return w.operationFailure(fmt.Errorf("encode nesting limit"), 0, InvalidValue)
	}
	w.depth++
	if w.path == "" {
		w.path = name
	} else {
		w.path += "/" + name
	}
	return nil
}
func (w *Writer) Leave() {
	if w.depth > 0 {
		w.depth--
	}
	for i := len(w.path) - 1; i >= 0; i-- {
		if w.path[i] == '/' {
			w.path = w.path[:i]
			return
		}
	}
	w.path = ""
}

// logicalPosition indexes replay spans only; L/H uses the physical or first-run
// inferred coordinate in LHBit instead. Inferred bits never shift transmission.
func (w *Writer) logicalPosition() (int, error) {
	if w.bits < 0 || w.virtual < 0 || w.bits > maxBits || w.virtual > maxBits-w.bits {
		return 0, w.operationFailure(fmt.Errorf("encoded bit limit exceeded"), 0, InvalidValue)
	}
	return w.bits + w.virtual, nil
}
func (w *Writer) put(bit uint8) error {
	position, err := w.logicalPosition()
	if err != nil {
		return w.operationFailure(err, 0, InvalidValue)
	}
	if position >= maxBits {
		return w.operationFailure(fmt.Errorf("encoded bit limit exceeded"), 0, InvalidValue)
	}
	if w.wire.canonical && w.allowZero && (w.bounded && w.bits >= w.limit || w.wire.targetSet && w.bits >= w.wire.targetBits) {
		if bit != 0 {
			detail := "nonzero bit beyond the receiver-inferred zero extension"
			if w.bounded && w.bits >= w.limit {
				return w.operationFailure(w.boundError(LengthBound, w.limit, "", detail), 0, InvalidValue)
			}
			return w.operationFailure(w.boundError(CanonicalTarget, w.wire.targetBits, "", detail), 0, InvalidValue)
		}
		w.virtual++
		w.lh.infer(w.bits, 1)
		return nil
	}
	if w.wire.canonical && w.wire.targetSet && w.bits >= w.wire.targetBits {
		return w.operationFailure(w.boundError(CanonicalTarget, w.wire.targetBits, "", "encoded bit exceeds the canonical target"), 0, InvalidValue)
	}
	if w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans) {
		span, end, err := w.currentImplicitSpan()
		if err != nil {
			return w.operationFailure(err, 0, InvalidValue)
		}
		if at := position; at >= span.At && at < end {
			if bit != 0 {
				detail := "edit to receiver-inferred bit"
				if at != w.bits {
					detail = fmt.Sprintf("edit to receiver-inferred bit %d", at)
				}
				return w.operationFailure(fmt.Errorf("%s", detail), 0, InvalidValue)
			}
			w.virtual++
			w.lh.infer(w.bits, 1)
			if w.bits+w.virtual == end {
				w.implicitSpan++
			}
			return nil
		}
	}
	if w.bounded && w.bits >= w.limit {
		return w.operationFailure(w.boundError(LengthBound, w.limit, "", "encoded bit exceeds the length-delimited value"), 0, InvalidValue)
	}
	if w.bits%8 == 0 {
		w.bytes = append(w.bytes, 0)
	}
	if bit != 0 {
		w.bytes[len(w.bytes)-1] |= 1 << (7 - uint(w.bits%8))
	}
	w.bits++
	return nil
}

func (w *Writer) currentImplicitSpan() (ImplicitSpan, int, error) {
	if w.implicitSpan >= len(w.wire.ImplicitSpans) {
		return ImplicitSpan{}, 0, w.operationFailure(fmt.Errorf("receiver-inferred span is missing"), 0, InvalidValue)
	}
	span := w.wire.ImplicitSpans[w.implicitSpan]
	if span.Count <= 0 || span.Count > maxBits || span.At < 0 || span.At > maxBits-span.Count {
		return ImplicitSpan{}, 0, w.operationFailure(fmt.Errorf("receiver-inferred span outside bit limit"), 0, InvalidValue)
	}
	return span, span.At + span.Count, nil
}
func (w *Writer) WriteUint(value uint64, width int) error {
	if width < 0 || width > 64 || width < 64 && value >= uint64(1)<<uint(width) {
		return w.operationFailure(fmt.Errorf("value outside %d-bit field", width), width, InvalidValue)
	}
	for i := width - 1; i >= 0; i-- {
		if err := w.put(uint8((value >> uint(i)) & 1)); err != nil {
			return w.operationFailure(err, width, InvalidValue)
		}
	}
	return nil
}
func (w *Writer) WriteBitString(value BitString, width int) error {
	if width < 0 || width != value.BitLength || width > maxBits || len(value.Bytes) < (width+7)/8 {
		return w.operationFailure(fmt.Errorf("invalid bit string width"), width, InvalidValue)
	}
	for i := 0; i < width; i++ {
		if err := w.put((value.Bytes[i/8] >> (7 - uint(i%8))) & 1); err != nil {
			return w.operationFailure(err, width, InvalidValue)
		}
	}
	return nil
}
func (w *Writer) WriteLiteral(pattern string) error {
	for _, symbol := range pattern {
		bit := literalBit(symbol, w.bits)
		if (symbol == 'L' || symbol == 'H') && w.inferredLHPositionPossible() {
			value, err := w.LHBit(symbol)
			if err != nil {
				return err
			}
			bit = value
		}
		if err := w.put(bit); err != nil {
			return w.operationFailure(err, 0, InvalidValue)
		}
	}
	return nil
}
func (w *Writer) WriteSpare() error {
	if w.spare < len(w.wire.Spare) {
		v := w.wire.Spare[w.spare]
		w.spare++
		return w.operationFailure(w.WriteLayoutBit(v), 0, InvalidValue)
	}
	if w.wire.sealed {
		return w.operationFailure(fmt.Errorf("received spare-bit state exhausted"), 0, InvalidValue)
	}
	return w.operationFailure(w.WriteUint(0, 1), 0, InvalidValue)
}

// WriteLayoutBit replays a value-local received bit. An empty record writes
// zero. Inferred records carry no transmitted bit, so a replacement at a
// transmitted position is fresh. WriteUint preserves the containing root's
// inferred spans and rejects nonzero edits there.
// Canonical copies clear these records (TS 24.008 §§10.5.5.12, 10.5.5.12a).
func (w *Writer) WriteLayoutBit(v BitString) error {
	if v.BitLength == 1 {
		return w.operationFailure(w.WriteBitString(v, 1), 0, InvalidValue)
	}
	if v.BitLength != 0 || len(v.Bytes) != 0 {
		return w.operationFailure(fmt.Errorf("invalid received spare-bit width"), 0, InvalidValue)
	}

	return w.operationFailure(w.WriteUint(0, 1), 0, InvalidValue)
}
func (w *Writer) WritePadding() error {
	if w.padding < len(w.wire.Padding) {
		v := w.wire.Padding[w.padding]
		w.padding++
		return w.operationFailure(w.WriteBitString(v, v.BitLength), 0, InvalidValue)
	}
	if w.wire.sealed {
		return w.operationFailure(fmt.Errorf("received padding state exhausted"), 0, InvalidValue)
	}
	if w.wire.canonical && w.wire.targetSet && !w.bounded {
		return w.operationFailure(w.WritePaddingTo(w.wire.targetBits), 0, InvalidValue)
	}
	for w.bits%8 != 0 {
		if w.inferredLHPositionPossible() {
			if err := w.WriteLiteral("L"); err != nil {
				return w.operationFailure(err, 0, InvalidValue)
			}
		} else if err := w.put(literalBit('L', w.bits)); err != nil {
			return w.operationFailure(err, 0, InvalidValue)
		}
	}
	return nil
}

// WritePaddingTo fills a fixed-size rest-octet value with the GSM 0x2b
// L/H pattern. TS 44.018 V19.0.0 §§10.5.2.37h–i specify 20 octets;
// TS 44.060 V19.0.0 §11 defines the padding pattern. A decoded value's
// received padding is retained when its width still fits after editing.
func (w *Writer) WritePaddingTo(bits int) error {
	if bits > maxBits {
		return w.operationFailure(fmt.Errorf("padding target outside encoded bit limit"), 0, InvalidValue)
	}
	if bits < w.bits {
		return w.operationFailure(w.boundError(LengthBound, bits, "", "content exceeds the fixed-size value"), 0, InvalidValue)
	}
	if w.wire.Tail.BitLength == bits-w.bits && w.wire.Tail.BitLength > 0 {
		return w.operationFailure(w.WriteBitString(w.wire.Tail, w.wire.Tail.BitLength), 0, InvalidValue)
	}
	if w.padding < len(w.wire.Padding) {
		v := w.wire.Padding[w.padding]
		w.padding++
		if v.BitLength == bits-w.bits {
			return w.operationFailure(w.WriteBitString(v, v.BitLength), 0, InvalidValue)
		}
		if w.wire.sealed {
			return w.operationFailure(fmt.Errorf("received padding width differs from fixed IE"), 0, InvalidValue)
		}
	}
	if w.wire.sealed && bits > w.bits {
		return w.operationFailure(fmt.Errorf("missing received fixed-IE padding"), 0, InvalidValue)
	}
	for w.bits < bits {
		if w.inferredLHPositionPossible() {
			if err := w.WriteLiteral("L"); err != nil {
				return w.operationFailure(err, 0, InvalidValue)
			}
		} else if err := w.put(literalBit('L', w.bits)); err != nil {
			return w.operationFailure(err, 0, InvalidValue)
		}
	}
	return nil
}

// PadMinimum completes a fresh value to the minimum length of its containing
// IE. Explicit <spare padding> uses L/H (TS 24.007 V20.0.0 Annex B
// §B.1.2.2); otherwise the value uses zero completion bits.
func (w *Writer) PadMinimum(bits int, sparePadding bool) error {
	if w.wire.sealed || w.bits >= bits {
		return nil
	}
	if sparePadding {
		return w.operationFailure(w.WritePaddingTo(bits), 0, InvalidValue)
	}
	for w.bits < bits {
		if err := w.put(0); err != nil {
			return w.operationFailure(err, 0, InvalidValue)
		}
	}
	return nil
}

// WriteZeroLengthTerminal emits the SI18/SI20 list stop record. Its
// discriminator bits are retained from the received wire when available;
// the canonical newly constructed record is zero (TS 44.018 V19.0.0
// §10.5.2.37h table 10.5.2.37h.1, zero container-octet count).
func (w *Writer) WriteZeroLengthTerminal() error {
	if w.terminal < len(w.wire.Terminal) {
		v := w.wire.Terminal[w.terminal]
		w.terminal++
		if v.BitLength != 8 || len(v.Bytes) < 1 || v.Bytes[0]&0x1f != 0 {
			return w.operationFailure(fmt.Errorf("invalid zero-length SI terminal record"), 0, InvalidValue)
		}
		return w.operationFailure(w.WriteBitString(v, 8), 0, InvalidValue)
	}
	return w.operationFailure(w.WriteUint(0, 8), 0, InvalidValue)
}

// WriteIgnored emits exactly the received or supplied fallback bits, without
// adding L/H spare padding inside a length-delimited extension.
func (w *Writer) WriteIgnored(value BitString) error {
	if w.padding < len(w.wire.Padding) {
		stored := w.wire.Padding[w.padding]
		w.padding++
		if value.BitLength == 0 {
			value = stored
		}
	}
	return w.operationFailure(w.WriteBitString(value, value.BitLength), 0, InvalidValue)
}

// FallbackMark is the writer state where an ignored fallback arm starts.
type FallbackMark struct {
	bits, virtual int
	lh            lhPositionState
	allowZero     bool
	vars          map[string]uint64
}

// MarkFallback records the state the decoder's known-arm trial starts from:
// the bit position, the receiver-inferred zeros so far, zero extension and
// the field bindings, which the writer binds as the reader does.
func (w *Writer) MarkFallback() FallbackMark {
	vars := make(map[string]uint64, len(w.vars))
	for k, v := range w.vars {
		vars[k] = v
	}
	return FallbackMark{bits: w.bits, virtual: w.virtual, lh: w.lh, allowZero: w.allowZero, vars: vars}
}

// CheckIgnoredFallback returns a *FallbackError when the bits written since
// mark decode through known, which the decoder would then choose. The trial
// reads exactly the bits the decoder's trial can read: with toBound, the
// completed length-delimited value that the ignored arm fills; otherwise the
// ignored bits alone, which the generator permits only when the known arm
// cannot read past them. An ignored arm that leaves its bound unfilled is
// not tried here, because PopLimit rejects it. Recognized reserved values
// remain value-constraint errors (TS 44.060 V19.0.0 §11.1, §12.24).
func (w *Writer) CheckIgnoredFallback(mark FallbackMark, toBound bool, known func(*Reader) error) error {
	if toBound && (!w.bounded || w.bits != w.limit) {
		return nil
	}
	if mark.bits < 0 || mark.bits > w.bits {
		return w.operationFailure(fmt.Errorf("invalid fallback mark"), 0, InvalidValue)
	}
	r := NewReader(w.bytes)
	// Reader.Check bounds the window by the written bytes; the shared
	// bindings are copied before the trial binds anything.
	r.pos, r.end, r.virtual, r.allowZero, r.vars, r.varsShared = mark.bits, w.bits, mark.virtual, mark.allowZero, mark.vars, true
	r.lh = mark.lh
	if toBound {
		r.boundDepth = 1
	}
	if err := r.Check(); err != nil {
		return w.operationFailure(err, 0, InvalidValue)
	}
	err := known(r)
	if errors.Is(err, ErrLHPositionUndefined) {
		return w.operationFailure(err, 0, InvalidSchema)
	}
	if err == nil {
		return w.operationFailure(&FallbackError{Path: w.path, Position: mark.bits}, 0, InvalidValue)
	}
	if de := decodeCause(err); de != nil && de.Kind == InvalidValue && (de.Detail == "reserved CSN.1 alternative" || de.Detail == "value reserved by source table") {
		return w.operationFailure(fmt.Errorf("%s", de.Detail), 0, InvalidValue)
	}
	return nil
}

// WriteIgnoredFixed re-emits a received ignored field or a zero-valued
// field for a newly constructed value (TS 44.018 V19.0.0 §10.5.2.33b).
func (w *Writer) WriteIgnoredFixed(width int) error {
	if width < 0 || width > 64 {
		return w.operationFailure(fmt.Errorf("ignored field width outside 0..64"), 0, InvalidValue)
	}
	if w.padding < len(w.wire.Padding) {
		value := w.wire.Padding[w.padding]
		w.padding++
		return w.operationFailure(w.WriteBitString(value, width), 0, InvalidValue)
	}
	return w.operationFailure(w.WriteUint(0, width), 0, InvalidValue)
}
func (w *Writer) Finish(tail BitString) ([]byte, error) {
	if w.wire.sealed {
		if err := w.validateSemantic(); err != nil {
			return nil, w.operationFailure(err, 0, InvalidValue)
		}
		if tail.BitLength != w.wire.Tail.BitLength {
			return nil, w.operationFailure(fmt.Errorf("wire tail length differs from received boundary"), 0, InvalidValue)
		}
	}
	if tail.BitLength > 0 {
		if err := w.WriteBitString(tail, tail.BitLength); err != nil {
			return nil, w.operationFailure(err, 0, InvalidValue)
		}
	}
	if w.bits%8 != 0 {
		// TS 24.007 V20.0.0 Annex B.1.2.1 Rule B7:
		// octet completion outside an explicit <spare padding>
		// construct uses zero bits. WritePadding alone emits L.
		for w.bits%8 != 0 {
			if err := w.put(0); err != nil {
				return nil, w.operationFailure(err, 0, InvalidValue)
			}
		}
	}
	return w.Bytes()
}
func (w *Writer) Bytes() ([]byte, error) {
	if w.stateErr != nil {
		return nil, w.OperationError(InvalidValue, w.stateErrPath, 0, w.stateErr)
	}
	if w.wire.sealed {
		if err := w.validateState(); err != nil {
			return nil, w.operationFailure(err, 0, InvalidValue)
		}
		if w.bits != w.wire.TransmittedBits {
			return nil, w.operationFailure(fmt.Errorf("encoded length %d differs from transmitted %d", w.bits, w.wire.TransmittedBits), 0, InvalidValue)
		}
	}
	if w.bits%8 != 0 {
		return nil, w.operationFailure(fmt.Errorf("encoded %d bits is not octet-aligned", w.bits), 0, InvalidValue)
	}
	return append([]byte(nil), w.bytes...), nil
}

func (w *Writer) validateSemantic() error { return w.validateValueSemantic(true) }
func (w *Writer) validateValueSemantic(octets bool) error {
	if err := w.validateValueState(octets); err != nil {
		return err
	}
	if w.bits != w.wire.BitsConsumed || w.virtual != w.wire.ImplicitZeros || w.implicitSpan != len(w.wire.ImplicitSpans) {
		return fmt.Errorf("encoded semantic boundary %d+%d differs from received %d+%d", w.bits, w.virtual, w.wire.BitsConsumed, w.wire.ImplicitZeros)
	}
	return nil
}

func (w *Writer) validateState() error { return w.validateValueState(true) }
func (w *Writer) validateValueState(octets bool) error {
	if w.wire.TransmittedBits < 0 || w.wire.TransmittedBits > maxBits || (octets && w.wire.TransmittedBits%8 != 0) || w.wire.BitsConsumed < 0 || w.wire.BitsConsumed > w.wire.TransmittedBits || w.wire.ImplicitZeros < 0 || w.wire.ImplicitZeros > maxBits || w.wire.Tail.BitLength < 0 || w.wire.Tail.BitLength > maxBits {
		return fmt.Errorf("inconsistent received wire boundary or tail")
	}
	if w.wire.Tail.BitLength != w.wire.TransmittedBits-w.wire.BitsConsumed || len(w.wire.Tail.Bytes) < (w.wire.Tail.BitLength+7)/8 {
		return fmt.Errorf("inconsistent received wire boundary or tail")
	}
	count, end := 0, 0
	for _, span := range w.wire.ImplicitSpans {
		if span.Count <= 0 || span.Count > maxBits || span.At < end || span.At > maxBits-span.Count || count > maxBits-span.Count {
			return fmt.Errorf("inconsistent inferred-zero span")
		}
		count += span.Count
		end = span.At + span.Count
	}
	if count != w.wire.ImplicitZeros {
		return fmt.Errorf("inconsistent inferred-zero count")
	}
	if w.spare != len(w.wire.Spare) || w.padding != len(w.wire.Padding) || w.terminal != len(w.wire.Terminal) {
		return fmt.Errorf("unconsumed received spare, padding, or terminal state")
	}
	for path, points := range w.wire.TruncatedAt {
		if w.truncationIndex[path] != len(points) {
			return fmt.Errorf("unused truncation point %s", path)
		}
	}
	for path, counts := range w.wire.SpareCounts {
		if w.spareCountIndex[path] != len(counts) {
			return fmt.Errorf("unused spare count %s", path)
		}
	}
	return nil
}

// ValidateOutput checks a non-writer wrapper against its received boundary.
// Wrapper encoders still construct every byte from typed fields and wire state.
func (wire WireInfo) ValidateOutput(encoded []byte, consumed int) error {
	if !wire.sealed {
		return nil
	}
	w := NewWriter()
	w.WithWire(wire)
	if len(encoded) > maxBits/8 || wire.TransmittedBits != len(encoded)*8 || consumed != wire.BitsConsumed {
		return fmt.Errorf("inconsistent wrapper wire boundary or tail")
	}
	if len(wire.Spare) != 0 || len(wire.Padding) != 0 || len(wire.Terminal) != 0 || len(wire.TruncatedAt) != 0 || len(wire.SpareCounts) != 0 || wire.ImplicitZeros != 0 || len(wire.ImplicitSpans) != 0 {
		return fmt.Errorf("unexpected wrapper wire state")
	}
	if err := w.validateState(); err != nil {
		return err
	}
	for i := 0; i < wire.Tail.BitLength; i++ {
		got := (encoded[(consumed+i)/8] >> (7 - uint((consumed+i)%8))) & 1
		want := (wire.Tail.Bytes[i/8] >> (7 - uint(i%8))) & 1
		if got != want {
			return fmt.Errorf("wrapper tail differs from wire state")
		}
	}
	return nil
}

// LHBit returns the L/H value at the absolute bit position. TS 24.007
// V20.0.0 Annex B §B.1.2.2 and TS 44.018 V19.0.0 §10.5.2.32 use the
// octet-aligned 0x2b pattern for position-relative L/H constraints.
func LHBit(symbol rune, position int) uint8 { return literalBit(symbol, position) }

func literalBit(symbol rune, position int) uint8 {
	// TS 24.007 V20.0.0 Annex B §B.1.2.2 and TS 44.060 V19.0.0
	// §11: L is the bit in octet-aligned 0x2b; H is its complement.
	// Wireshark packet-csn1.c and pycrate_csn1/csnobj.py agree.
	padding := uint8((0x2b >> (7 - uint(position%8))) & 1)
	switch symbol {
	case '0':
		return 0
	case '1':
		return 1
	case 'L':
		return padding
	case 'H':
		return padding ^ 1
	default:
		return 2
	}
}

func bitsAt(data []byte, start, count int) BitString {
	if len(data) > maxBits/8 || start < 0 || count <= 0 || start > len(data)*8 || count > len(data)*8-start {
		return BitString{}
	}
	bytes := make([]byte, (count+7)/8)
	for i := 0; i < count; i++ {
		if data[(start+i)/8]&(1<<(7-uint((start+i)%8))) != 0 {
			bytes[i/8] |= 1 << (7 - uint(i%8))
		}
	}
	return BitString{Bytes: bytes, BitLength: count}
}

// TrailingBits retains received bits beyond a decoded value boundary.
// TS 24.008 V20.1.0 §10.5.5.12a bounds MS RA capability at 50
// octets; any accepted excess carrier padding is outside that value.
func TrailingBits(data []byte, consumed int) BitString {
	if len(data) > maxBits/8 || consumed < 0 || consumed > len(data)*8 {
		return BitString{}
	}
	return bitsAt(data, consumed, len(data)*8-consumed)
}

// ReceivedBits reports the original whole-message extent when encoding a
// decoded value. TS 44.018 V19.0.0 §9.1.55 permits a shorter pre-Rel-8
// bitmap only when the trailing no-report positions did not fit.
func (w *Writer) ReceivedBits() (int, bool) {
	return w.wire.TransmittedBits, w.wire.sealed
}
