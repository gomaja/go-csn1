package runtime

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// WireInfo retains non-semantic received bits and the actual truncation point.
// TS 24.008 V20.1.0 §§10.5.1.7, 10.5.5.12a permit spare bits and
// implicit zero extension; TS 24.007 V20.0.0 Annex B §B.1.2.2 and
// TS 44.060 V19.0.0 §11 define the octet-aligned 0x2b L/H pattern.
type WireInfo struct {
	Original      []byte
	BitsConsumed  int
	Tail          BitString
	Spare         []BitString
	Padding       []BitString
	TruncatedAt   map[string]int
	SpareCounts   map[string][]int
	ImplicitZeros int
	valueChecksum [sha256.Size]byte
	rawChecksum   [sha256.Size]byte
}

// Seal records a received value after all of its typed fields are populated.
// Generated codecs can replay an unchanged value exactly; edited and newly
// constructed values take the generated encoder path.
func Seal[T any](value T, raw []byte, consumed int, tail BitString, wire WireInfo) WireInfo {
	wire.Original = make([]byte, len(raw))
	copy(wire.Original, raw)
	wire.BitsConsumed = consumed
	wire.Tail = tail
	wire.valueChecksum = checksum(value)
	wire.rawChecksum = sha256.Sum256(wire.Original)
	return wire
}

func OriginalIfUnchanged[T any](value T, wire WireInfo) ([]byte, bool) {
	if wire.Original == nil || checksum(value) != wire.valueChecksum || sha256.Sum256(wire.Original) != wire.rawChecksum {
		return nil, false
	}
	return append([]byte(nil), wire.Original...), true
}

func checksum[T any](value T) [sha256.Size]byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		return [sha256.Size]byte{}
	}
	return sha256.Sum256(encoded)
}

// Reader is an MSB-first, bounded CSN.1 bit reader. A fork shares the work
// budget but isolates position and expression bindings for choice resolution.
type Reader struct {
	data              []byte
	pos, end, virtual int
	allowZero         bool
	vars              map[string]uint64
	steps             *int
	path              string
	depth             int
	wire              WireInfo
}

func NewReader(data []byte) *Reader {
	steps := 0
	return &Reader{data: data, end: len(data) * 8, vars: make(map[string]uint64), steps: &steps,
		wire: WireInfo{TruncatedAt: make(map[string]int)}}
}

func CheckInput(data []byte) error {
	if len(data) > maxBits/8 {
		return &DecodeError{Kind: Limit, Offset: 0, Detail: "input exceeds maximum bit length"}
	}
	return nil
}

func (r *Reader) Position() int                 { return r.pos }
func (r *Reader) Remaining() int                { return r.end - r.pos }
func (r *Reader) VirtualBits() int              { return r.virtual }
func (r *Reader) Wire() WireInfo                { r.wire.ImplicitZeros = r.virtual; return r.wire }
func (r *Reader) Tail() BitString               { return bitsAt(r.data, r.pos, len(r.data)*8-r.pos) }
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
	if r.pos < r.end {
		b := (r.data[r.pos/8] >> (7 - uint(r.pos%8))) & 1
		r.pos++
		return b, true, nil
	}
	if r.allowZero {
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
	if r.pos+len(pattern) > r.end {
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
	return old, nil
}
func (r *Reader) PopLimit(old int) error {
	if r.pos != r.end {
		return r.fail(InvalidValue, "length-delimited content leaves unparsed bits")
	}
	r.end = old
	return nil
}
func (r *Reader) RecordTruncation(path string, child int) { r.wire.TruncatedAt[path] = child }
func (r *Reader) RecordSpareCount(path string, count int) {
	if r.wire.SpareCounts == nil {
		r.wire.SpareCounts = make(map[string][]int)
	}
	r.wire.SpareCounts[path] = append(r.wire.SpareCounts[path], count)
}

// Writer uses the same MSB-first bit order and expression rules as Reader.
type Writer struct {
	bytes           []byte
	bits            int
	vars            map[string]uint64
	path            string
	depth           int
	spare, padding  int
	spareCountIndex map[string]int
	wire            WireInfo
}

func NewWriter() *Writer                             { return &Writer{vars: make(map[string]uint64)} }
func (w *Writer) WithWire(wire WireInfo)             { w.wire = wire; w.spareCountIndex = make(map[string]int) }
func (w *Writer) Truncation(path string) (int, bool) { n, ok := w.wire.TruncatedAt[path]; return n, ok }
func (w *Writer) SpareCount(path string) (int, bool) {
	counts := w.wire.SpareCounts[path]
	i := w.spareCountIndex[path]
	if i >= len(counts) {
		return 0, false
	}
	w.spareCountIndex[path] = i + 1
	return counts[i], true
}
func (w *Writer) Position() int                       { return w.bits }
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
	if w.bits >= maxBits {
		return fmt.Errorf("encoded bit limit exceeded at %s", w.path)
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
	if width < 0 || width != value.BitLength || width > len(value.Bytes)*8 || width > maxBits {
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
		if err := w.put(literalBit(symbol, w.bits)); err != nil {
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
	}
	return w.WriteUint(0, 1)
}
func (w *Writer) WritePadding() error {
	if w.padding < len(w.wire.Padding) {
		v := w.wire.Padding[w.padding]
		w.padding++
		return w.WriteBitString(v, v.BitLength)
	}
	for w.bits%8 != 0 {
		if err := w.put(literalBit('L', w.bits)); err != nil {
			return err
		}
	}
	return nil
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
	if w.bits%8 != 0 {
		return nil, fmt.Errorf("encoded %d bits is not octet-aligned", w.bits)
	}
	return append([]byte(nil), w.bytes...), nil
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
