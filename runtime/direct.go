package runtime

import (
	"fmt"
	"reflect"
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
	TruncatedAt     map[string]int
	SpareCounts     map[string][]int
	ImplicitZeros   int
	ImplicitSpans   []ImplicitSpan
	sealed          bool
}

// ImplicitSpan records a receiver-inferred zero run in logical bit order.
// TS 24.008 V20.1.0 §§10.5.1.7, 10.5.5.12a allow zero extension without
// transmission, including within a bounded capability before later bits.
type ImplicitSpan struct{ At, Count int }

// IsZero reports whether a field omitted by a transmitted truncation remains
// unedited. TS 44.018 V19.0.0 §8.9 permits omission only as an ordered prefix.
func IsZero[T any](value T) bool { var zero T; return reflect.DeepEqual(value, zero) }

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
type Reader struct {
	data              []byte
	pos, end, virtual int
	boundDepth        int
	allowZero         bool
	vars              map[string]uint64
	varsShared        bool
	steps             *int
	path              []string
	pathShared        bool
	depth             int
	wire              WireInfo
	implicitShared    bool
	truncShared       bool
	spareCountsShared bool
}

func NewReader(data []byte) *Reader {
	steps := 0
	end := 0
	if len(data) <= maxBits/8 {
		end = len(data) * 8
	}
	return &Reader{data: data, end: end, vars: make(map[string]uint64), path: make([]string, 0, 16), steps: &steps,
		wire: WireInfo{TruncatedAt: make(map[string]int)}}
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
		return &DecodeError{Kind: InvalidValue, Detail: "nil reader"}
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

func (r *Reader) Position() int { return r.pos }
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
func (r *Reader) Wire() WireInfo   { r.wire.ImplicitZeros = r.virtual; return r.wire }
func (r *Reader) Tail() BitString {
	if r.Check() != nil {
		return BitString{}
	}
	return bitsAt(r.data, r.pos, len(r.data)*8-r.pos)
}
func (r *Reader) SetZeroExtension(enabled bool) { r.allowZero = enabled }
func (r *Reader) ZeroExtension() bool           { return r.allowZero }
func (r *Reader) Set(name string, value uint64) {
	if r.varsShared {
		r.vars = cloneVars(r.vars)
		r.varsShared = false
	}
	r.vars[key(name)] = value
}
func (r *Reader) Eval(expression string) (int, error) {
	v, err := eval(expression, r.vars)
	if err != nil {
		return 0, r.fail(InvalidSchema, err.Error())
	}
	return v, nil
}
func (r *Reader) fail(kind ErrorKind, detail string) error {
	return &DecodeError{Kind: kind, Offset: r.pos, Path: strings.Join(r.path, "/"), Detail: detail}
}
func (r *Reader) failAt(kind ErrorKind, offset int, detail string) error {
	return &DecodeError{Kind: kind, Offset: offset, Path: strings.Join(r.path, "/"), Detail: detail}
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
		fresh := make([]string, len(r.path), max(16, len(r.path)))
		copy(fresh, r.path)
		r.path = fresh
		r.pathShared = false
	}
	r.path = append(r.path, name)
	return nil
}
func (r *Reader) Leave() {
	if r.depth > 0 {
		r.depth--
	}
	if len(r.path) > 0 {
		r.path = r.path[:len(r.path)-1]
	}
}
func (r *Reader) Fork() *Reader {
	r.varsShared, r.pathShared, r.implicitShared, r.truncShared, r.spareCountsShared = true, true, true, true, true
	fork := *r
	fork.wire.Spare = fork.wire.Spare[:len(fork.wire.Spare):len(fork.wire.Spare)]
	fork.wire.Padding = fork.wire.Padding[:len(fork.wire.Padding):len(fork.wire.Padding)]
	fork.wire.Terminal = fork.wire.Terminal[:len(fork.wire.Terminal):len(fork.wire.Terminal)]
	return &fork
}
func (r *Reader) Commit(other *Reader) { *r = *other }
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
		return 0, false, nil
	}
	return 0, false, r.fail(Truncated, "required bit absent")
}
func (r *Reader) ReadUint(width int) (uint64, error) {
	if width < 0 || width > 64 {
		return 0, r.fail(Limit, "numeric width outside 0..64")
	}
	var value uint64
	for i := 0; i < width; i++ {
		b, _, err := r.bit()
		if err != nil {
			return 0, err
		}
		value = (value << 1) | uint64(b)
	}
	return value, nil
}
func (r *Reader) ReadBitString(width int) (BitString, error) {
	if width < 0 || width > maxBits || width > r.Remaining() && !r.allowZero {
		return BitString{}, r.fail(Truncated, "bit string exceeds input")
	}
	out := BitString{Bytes: make([]byte, (width+7)/8), BitLength: width}
	for i := 0; i < width; i++ {
		b, _, err := r.bit()
		if err != nil {
			return BitString{}, err
		}
		out.Bytes[i/8] |= b << (7 - uint(i%8))
	}
	return out, nil
}
func (r *Reader) Expect(pattern string) error {
	if err := r.Check(); err != nil {
		return err
	}
	for _, symbol := range pattern {
		at := r.pos + r.virtual
		bit, _, err := r.bit()
		if err != nil {
			return err
		}
		if bit != literalBit(symbol, at) {
			return r.fail(InvalidValue, fmt.Sprintf("literal %c differs", symbol))
		}
	}
	return nil
}
func (r *Reader) Matches(pattern string) bool {
	if r.Check() != nil || len(pattern) > r.Remaining() {
		return false
	}
	for i, symbol := range pattern {
		b := (r.data[(r.pos+i)/8] >> (7 - uint((r.pos+i)%8))) & 1
		if b != literalBit(symbol, r.pos+i) {
			return false
		}
	}
	return true
}
func (r *Reader) ReadSpare() (BitString, error) {
	start := r.pos
	_, present, err := r.bit()
	if err != nil {
		return BitString{}, err
	}
	var value BitString
	if present {
		value = bitsAt(r.data, start, 1)
	}
	r.wire.Spare = append(r.wire.Spare, value)
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
func (r *Reader) RecordTruncation(path string, child int) {
	if r.truncShared {
		copy := make(map[string]int, len(r.wire.TruncatedAt))
		for k, v := range r.wire.TruncatedAt {
			copy[k] = v
		}
		r.wire.TruncatedAt = copy
		r.truncShared = false
	}
	r.wire.TruncatedAt[path] = child
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
	limit                    int
	bounded                  bool
	vars                     map[string]uint64
	path                     string
	depth                    int
	spare, padding, terminal int
	spareCountIndex          map[string]int
	truncationUsed           map[string]bool
	wire                     WireInfo
	implicitSpan             int
	stateErr                 error
}

func NewWriter() *Writer { return &Writer{vars: make(map[string]uint64)} }
func (w *Writer) WithWire(wire WireInfo) {
	w.wire = wire
	w.spareCountIndex = make(map[string]int)
	w.truncationUsed = make(map[string]bool)
	w.stateErr = nil
}
func (w *Writer) Truncation(path string) (int, bool) {
	n, ok := w.wire.TruncatedAt[path]
	if ok {
		w.truncationUsed[path] = true
	}
	return n, ok
}
func (w *Writer) SpareCount(path string) (int, bool) {
	counts := w.wire.SpareCounts[path]
	i := w.spareCountIndex[path]
	if i >= len(counts) {
		return 0, false
	}
	if counts[i] < 0 || counts[i] > maxBits {
		w.stateErr = fmt.Errorf("spare repeat count outside bit limit at %s", path)
		return 0, false
	}
	w.spareCountIndex[path] = i + 1
	return counts[i], true
}
func (w *Writer) Position() int { return w.bits }
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
	if w.bits < 0 || w.bits > maxBits || width < 0 || width > maxBits-w.bits || w.bounded && (w.limit < w.bits || w.limit > maxBits || width > w.limit-w.bits) {
		return 0, fmt.Errorf("fixed value exceeds enclosing limit at %s", w.path)
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
		return fmt.Errorf("invalid enclosing fixed value at %s", w.path)
	}
	if w.bits != w.limit {
		return fmt.Errorf("fixed value has %d bits, want %d at %s", w.bits, w.limit, w.path)
	}
	w.limit = old
	w.bounded = old >= 0
	return nil
}
func (w *Writer) Set(name string, v uint64)           { w.vars[key(name)] = v }
func (w *Writer) Eval(expression string) (int, error) { return eval(expression, w.vars) }
func (w *Writer) Enter(name string) error {
	if w.depth >= 128 {
		return fmt.Errorf("encode nesting limit at %s", w.path)
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
func (w *Writer) logicalPosition() (int, error) {
	if w.bits < 0 || w.virtual < 0 || w.bits > maxBits || w.virtual > maxBits-w.bits {
		return 0, fmt.Errorf("encoded bit limit exceeded at %s", w.path)
	}
	return w.bits + w.virtual, nil
}
func (w *Writer) put(bit uint8) error {
	position, err := w.logicalPosition()
	if err != nil {
		return err
	}
	if position >= maxBits {
		return fmt.Errorf("encoded bit limit exceeded at %s", w.path)
	}
	if w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans) {
		span, end, err := w.currentImplicitSpan()
		if err != nil {
			return err
		}
		if at := position; at >= span.At && at < end {
			if bit != 0 {
				return fmt.Errorf("edit to receiver-inferred bit at %s bit %d", w.path, at)
			}
			w.virtual++
			if w.bits+w.virtual == end {
				w.implicitSpan++
			}
			return nil
		}
	}
	if w.bounded && w.bits >= w.limit {
		return fmt.Errorf("encoded bit exceeds fixed value at %s", w.path)
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
		return ImplicitSpan{}, 0, fmt.Errorf("receiver-inferred span is missing")
	}
	span := w.wire.ImplicitSpans[w.implicitSpan]
	if span.Count <= 0 || span.Count > maxBits || span.At < 0 || span.At > maxBits-span.Count {
		return ImplicitSpan{}, 0, fmt.Errorf("receiver-inferred span outside bit limit")
	}
	return span, span.At + span.Count, nil
}
func (w *Writer) WriteUint(value uint64, width int) error {
	if width < 0 || width > 64 || width < 64 && value >= uint64(1)<<uint(width) {
		return fmt.Errorf("value outside %d-bit field at %s", width, w.path)
	}
	for i := width - 1; i >= 0; i-- {
		if err := w.put(uint8((value >> uint(i)) & 1)); err != nil {
			return err
		}
	}
	return nil
}
func (w *Writer) WriteBitString(value BitString, width int) error {
	if width < 0 || width != value.BitLength || width > maxBits || len(value.Bytes) < (width+7)/8 {
		return fmt.Errorf("invalid bit string width at %s", w.path)
	}
	for i := 0; i < width; i++ {
		if err := w.put((value.Bytes[i/8] >> (7 - uint(i%8))) & 1); err != nil {
			return err
		}
	}
	return nil
}
func (w *Writer) WriteLiteral(pattern string) error {
	for _, symbol := range pattern {
		position, err := w.logicalPosition()
		if err != nil {
			return err
		}
		if err := w.put(literalBit(symbol, position)); err != nil {
			return err
		}
	}
	return nil
}
func (w *Writer) WriteSpare() error {
	if w.spare < len(w.wire.Spare) {
		v := w.wire.Spare[w.spare]
		w.spare++
		if v.BitLength == 1 {
			return w.WriteBitString(v, 1)
		}
		if v.BitLength != 0 {
			return fmt.Errorf("invalid received spare-bit width")
		}
		if w.wire.sealed {
			span, end, err := w.currentImplicitSpan()
			if err != nil {
				return err
			}
			at, err := w.logicalPosition()
			if err != nil {
				return err
			}
			if at < span.At || at >= end {
				return fmt.Errorf("inferred spare bit is outside its zero span")
			}
		}
	} else if w.wire.sealed {
		return fmt.Errorf("received spare-bit state exhausted")
	}
	return w.WriteUint(0, 1)
}
func (w *Writer) WritePadding() error {
	if w.padding < len(w.wire.Padding) {
		v := w.wire.Padding[w.padding]
		w.padding++
		return w.WriteBitString(v, v.BitLength)
	}
	if w.wire.sealed {
		return fmt.Errorf("received padding state exhausted")
	}
	for w.bits%8 != 0 {
		if err := w.put(literalBit('L', w.bits)); err != nil {
			return err
		}
	}
	return nil
}

// WritePaddingTo fills a fixed-size rest-octet value with the GSM 0x2b
// L/H pattern. TS 44.018 V19.0.0 §§10.5.2.37h–i specify 20 octets;
// TS 44.060 V19.0.0 §11 defines the padding pattern. A decoded value's
// received padding is retained when its width still fits after editing.
func (w *Writer) WritePaddingTo(bits int) error {
	if bits < w.bits || bits > maxBits {
		return fmt.Errorf("padding target outside bounded value")
	}
	if w.wire.Tail.BitLength == bits-w.bits && w.wire.Tail.BitLength > 0 {
		return w.WriteBitString(w.wire.Tail, w.wire.Tail.BitLength)
	}
	if w.padding < len(w.wire.Padding) {
		v := w.wire.Padding[w.padding]
		w.padding++
		if v.BitLength == bits-w.bits {
			return w.WriteBitString(v, v.BitLength)
		}
		if w.wire.sealed {
			return fmt.Errorf("received padding width differs from fixed IE")
		}
	}
	if w.wire.sealed && bits > w.bits {
		return fmt.Errorf("missing received fixed-IE padding")
	}
	for w.bits < bits {
		if err := w.put(literalBit('L', w.bits)); err != nil {
			return err
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
			return fmt.Errorf("invalid zero-length SI terminal record")
		}
		return w.WriteBitString(v, 8)
	}
	return w.WriteUint(0, 8)
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
	return w.WriteBitString(value, value.BitLength)
}

// WriteIgnoredFixed re-emits a received ignored field or a zero-valued
// field for a newly constructed value (TS 44.018 V19.0.0 §10.5.2.33b).
func (w *Writer) WriteIgnoredFixed(width int) error {
	if width < 0 || width > 64 {
		return fmt.Errorf("ignored field width outside 0..64 at %s", w.path)
	}
	if w.padding < len(w.wire.Padding) {
		value := w.wire.Padding[w.padding]
		w.padding++
		return w.WriteBitString(value, width)
	}
	return w.WriteUint(0, width)
}
func (w *Writer) Finish(tail BitString) ([]byte, error) {
	if w.wire.sealed {
		if err := w.validateSemantic(); err != nil {
			return nil, err
		}
		if tail.BitLength != w.wire.Tail.BitLength {
			return nil, fmt.Errorf("wire tail length differs from received boundary")
		}
	}
	if tail.BitLength > 0 {
		if err := w.WriteBitString(tail, tail.BitLength); err != nil {
			return nil, err
		}
	}
	if w.bits%8 != 0 {
		// TS 24.007 V20.0.0 Annex B.1.2.1 Rule B7:
		// octet completion outside an explicit <spare padding>
		// construct uses zero bits. WritePadding alone emits L.
		for w.bits%8 != 0 {
			if err := w.put(0); err != nil {
				return nil, err
			}
		}
	}
	return w.Bytes()
}
func (w *Writer) Bytes() ([]byte, error) {
	if w.stateErr != nil {
		return nil, w.stateErr
	}
	if w.wire.sealed {
		if err := w.validateState(); err != nil {
			return nil, err
		}
		if w.bits != w.wire.TransmittedBits {
			return nil, fmt.Errorf("encoded length %d differs from transmitted %d", w.bits, w.wire.TransmittedBits)
		}
	}
	if w.bits%8 != 0 {
		return nil, fmt.Errorf("encoded %d bits is not octet-aligned", w.bits)
	}
	return append([]byte(nil), w.bytes...), nil
}

func (w *Writer) validateSemantic() error {
	if err := w.validateState(); err != nil {
		return err
	}
	if w.bits != w.wire.BitsConsumed || w.virtual != w.wire.ImplicitZeros || w.implicitSpan != len(w.wire.ImplicitSpans) {
		return fmt.Errorf("encoded semantic boundary %d+%d differs from received %d+%d", w.bits, w.virtual, w.wire.BitsConsumed, w.wire.ImplicitZeros)
	}
	return nil
}

func (w *Writer) validateState() error {
	if w.wire.TransmittedBits < 0 || w.wire.TransmittedBits > maxBits || w.wire.TransmittedBits%8 != 0 || w.wire.BitsConsumed < 0 || w.wire.BitsConsumed > w.wire.TransmittedBits || w.wire.ImplicitZeros < 0 || w.wire.ImplicitZeros > maxBits || w.wire.Tail.BitLength < 0 || w.wire.Tail.BitLength != w.wire.TransmittedBits-w.wire.BitsConsumed || len(w.wire.Tail.Bytes) < (w.wire.Tail.BitLength+7)/8 {
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
	for path := range w.wire.TruncatedAt {
		if !w.truncationUsed[path] {
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

// SetNeighbourCellCount binds EMR's pre-Rel-8 bitmap to the serving cell's
// Neighbour Cell list (TS 44.018 V19.0.0 §§3.4.1.2.1.3, 9.1.55).
func (r *Reader) SetNeighbourCellCount(count int) error {
	if count < 0 || count > 96 {
		return fmt.Errorf("neighbour cell count outside 0..96: %d", count)
	}
	r.Set("NEIGHBOUR_CELL_COUNT", uint64(count))
	return nil
}

func (r *Reader) NeighbourCellCount() (int, bool) {
	v, ok := r.vars[key("NEIGHBOUR_CELL_COUNT")]
	if !ok || v > 96 {
		return 0, false
	}
	return int(v), true
}

func (w *Writer) SetNeighbourCellCount(count int) error {
	if count < 0 || count > 96 {
		return fmt.Errorf("neighbour cell count outside 0..96: %d", count)
	}
	w.Set("NEIGHBOUR_CELL_COUNT", uint64(count))
	return nil
}

func (w *Writer) NeighbourCellCount() (int, bool) {
	v, ok := w.vars[key("NEIGHBOUR_CELL_COUNT")]
	if !ok || v > 96 {
		return 0, false
	}
	return int(v), true
}
