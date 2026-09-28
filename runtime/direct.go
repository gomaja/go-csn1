package runtime

import (
	"fmt"
	"reflect"
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
	steps             *int
	path              string
	depth             int
	wire              WireInfo
}

func NewReader(data []byte) *Reader {
	steps := 0
	end := 0
	if len(data) <= maxBits/8 {
		end = len(data) * 8
	}
	return &Reader{data: data, end: end, vars: make(map[string]uint64), steps: &steps,
		wire: WireInfo{TruncatedAt: make(map[string]int)}}
}

func CheckInput(data []byte) error {
	if len(data) > maxBits/8 {
		return &DecodeError{Kind: Limit, Offset: 0, Detail: "input exceeds maximum bit length"}
	}
	return nil
}

func (r *Reader) Check() error {
	if r == nil {
		return &DecodeError{Kind: InvalidValue, Detail: "nil reader"}
	}
	return CheckInput(r.data)
}

func (r *Reader) Position() int  { return r.pos }
func (r *Reader) Remaining() int { return r.end - r.pos }
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
func (r *Reader) Set(name string, value uint64) { r.vars[key(name)] = value }
func (r *Reader) Eval(expression string) (int, error) {
	v, err := eval(expression, r.vars)
	if err != nil {
		return 0, r.fail(InvalidSchema, err.Error())
	}
	return v, nil
}
func (r *Reader) fail(kind ErrorKind, detail string) error {
	return &DecodeError{Kind: kind, Offset: r.pos, Path: r.path, Detail: detail}
}
func (r *Reader) Error(kind ErrorKind, detail string) error { return r.fail(kind, detail) }
func (r *Reader) Enter(name string) error {
	*r.steps++
	if *r.steps > maxBits || r.depth >= 128 {
		return r.fail(Limit, "decoder work or nesting limit")
	}
	r.depth++
	if r.path == "" {
		r.path = name
	} else {
		r.path += "/" + name
	}
	return nil
}
func (r *Reader) Leave() {
	if r.depth > 0 {
		r.depth--
	}
	for i := len(r.path) - 1; i >= 0; i-- {
		if r.path[i] == '/' {
			r.path = r.path[:i]
			return
		}
	}
	r.path = ""
}
func (r *Reader) Fork() *Reader {
	copy := *r
	copy.vars = cloneVars(r.vars)
	copy.wire.Spare = append([]BitString(nil), r.wire.Spare...)
	copy.wire.Padding = append([]BitString(nil), r.wire.Padding...)
	copy.wire.Terminal = append([]BitString(nil), r.wire.Terminal...)
	copy.wire.ImplicitSpans = append([]ImplicitSpan(nil), r.wire.ImplicitSpans...)
	copy.wire.TruncatedAt = make(map[string]int, len(r.wire.TruncatedAt))
	for k, v := range r.wire.TruncatedAt {
		copy.wire.TruncatedAt[k] = v
	}
	copy.wire.SpareCounts = make(map[string][]int, len(r.wire.SpareCounts))
	for k, counts := range r.wire.SpareCounts {
		copy.wire.SpareCounts[k] = append([]int(nil), counts...)
	}
	return &copy
}
func (r *Reader) Commit(other *Reader) { *r = *other }
func (r *Reader) bit() (uint8, bool, error) {
	if err := r.Check(); err != nil {
		return 0, false, err
	}
	if r.pos > maxBits || r.virtual >= maxBits-r.pos {
		return 0, false, r.fail(Limit, "decoder bit limit exceeded")
	}
	if r.pos < r.end {
		b := (r.data[r.pos/8] >> (7 - uint(r.pos%8))) & 1
		r.pos++
		return b, true, nil
	}
	if r.allowZero {
		at := r.pos + r.virtual
		spans := r.wire.ImplicitSpans
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
	if len(pattern) > r.Remaining() {
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
	if width < 0 || width > r.Remaining() {
		return 0, r.fail(Truncated, "length exceeds enclosing input")
	}
	old := r.end
	r.end = r.pos + width
	r.boundDepth++
	return old, nil
}
func (r *Reader) PopLimit(old int) error {
	if r.boundDepth == 0 {
		return r.fail(InvalidValue, "no enclosing length limit")
	}
	if r.pos != r.end {
		return r.fail(InvalidValue, "length-delimited content leaves unparsed bits")
	}
	r.end = old
	r.boundDepth--
	return nil
}
func (r *Reader) RecordTruncation(path string, child int) { r.wire.TruncatedAt[path] = child }

// RecordTerminal preserves the nonsemantic zero-length stop record printed
// in TS 44.018 V19.0.0 §10.5.2.37h table 10.5.2.37h.1.
func (r *Reader) RecordTerminal(start int) {
	r.wire.Terminal = append(r.wire.Terminal, bitsAt(r.data, start, r.pos-start))
}
func (r *Reader) RecordSpareCount(path string, count int) {
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
}

func NewWriter() *Writer { return &Writer{vars: make(map[string]uint64)} }
func (w *Writer) WithWire(wire WireInfo) {
	w.wire = wire
	w.spareCountIndex = make(map[string]int)
	w.truncationUsed = make(map[string]bool)
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
	w.spareCountIndex[path] = i + 1
	return counts[i], true
}
func (w *Writer) Position() int { return w.bits }
func (w *Writer) RemainingLimit() int {
	if !w.bounded {
		return -1
	}
	return w.limit - w.bits
}
func (w *Writer) BoundEndOr(unbounded int) int {
	if w.bounded {
		return w.limit
	}
	return unbounded
}
func (w *Writer) PushLimit(width int) (int, error) {
	if width < 0 || width > maxBits-w.bits || w.bounded && width > w.limit-w.bits {
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
func (w *Writer) put(bit uint8) error {
	if w.bounded && w.bits >= w.limit {
		return fmt.Errorf("encoded bit exceeds fixed value at %s", w.path)
	}
	if w.bits+w.virtual >= maxBits {
		return fmt.Errorf("encoded bit limit exceeded at %s", w.path)
	}
	if w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans) && w.bits+w.virtual >= w.wire.ImplicitSpans[w.implicitSpan].At && w.bits+w.virtual < w.wire.ImplicitSpans[w.implicitSpan].At+w.wire.ImplicitSpans[w.implicitSpan].Count {
		if bit != 0 {
			return fmt.Errorf("edit to receiver-inferred bit at %s bit %d", w.path, w.bits+w.virtual)
		}
		w.virtual++
		if w.bits+w.virtual == w.wire.ImplicitSpans[w.implicitSpan].At+w.wire.ImplicitSpans[w.implicitSpan].Count {
			w.implicitSpan++
		}
		return nil
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
		if err := w.put(literalBit(symbol, w.bits+w.virtual)); err != nil {
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
		if w.wire.sealed && (w.implicitSpan >= len(w.wire.ImplicitSpans) || w.bits+w.virtual < w.wire.ImplicitSpans[w.implicitSpan].At || w.bits+w.virtual >= w.wire.ImplicitSpans[w.implicitSpan].At+w.wire.ImplicitSpans[w.implicitSpan].Count) {
			return fmt.Errorf("inferred spare bit is outside its zero span")
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
		if err := w.WritePadding(); err != nil {
			return nil, err
		}
	}
	return w.Bytes()
}
func (w *Writer) Bytes() ([]byte, error) {
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
	if count <= 0 {
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
