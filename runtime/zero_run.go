package runtime

import (
	"encoding/binary"
	"math/bits"
)

// ExpectZeros consumes width successive one-bit zero literals. It preserves
// Expect("0") diagnostics and partial progress, including inferred zeros,
// while scanning aligned received input a word at a time. Nonpositive widths
// perform no operations, as with an empty loop of Expect calls.
func (r *Reader) ExpectZeros(width int) error {
	if width <= 0 {
		return nil
	}
	if err := r.Check(); err != nil {
		return err
	}
	remaining := min(width, r.end-r.pos, maxBits-r.pos-r.virtual)
	for remaining > 0 && r.pos%8 != 0 {
		if err := r.Expect("0"); err != nil {
			return err
		}
		remaining--
		width--
	}
	for remaining >= 64 {
		word := binary.BigEndian.Uint64(r.data[r.pos/8:])
		if word != 0 {
			r.pos += bits.LeadingZeros64(word)
			return r.Expect("0")
		}
		r.pos += 64
		remaining -= 64
		width -= 64
	}
	for remaining >= 8 {
		word := r.data[r.pos/8]
		if word != 0 {
			r.pos += bits.LeadingZeros8(word)
			return r.Expect("0")
		}
		r.pos += 8
		remaining -= 8
		width -= 8
	}
	for remaining > 0 {
		if err := r.Expect("0"); err != nil {
			return err
		}
		remaining--
		width--
	}
	if width == 0 {
		return nil
	}
	// The scalar transition checks the bit limit and validates or detaches the
	// inferred span. Once it succeeds, all following zeros extend that span.
	if err := r.Expect("0"); err != nil {
		return err
	}
	width--
	inferred := min(width, maxBits-r.pos-r.virtual)
	r.virtual += inferred
	r.lh.infer(r.pos, inferred)
	r.wire.ImplicitSpans[len(r.wire.ImplicitSpans)-1].Count += inferred
	if width > inferred {
		return r.Expect("0")
	}
	return nil
}

// WriteZeros emits width successive one-bit zero literals. It preserves
// WriteLiteral("0") errors and partial progress at received, canonical and
// length bounds, and emits contiguous physical zeros in bulk. Nonpositive
// widths perform no operations.
func (w *Writer) WriteZeros(width int) error {
	for width > 0 {
		// Use the existing transition at every boundary, retaining its error
		// precedence, inferred-span validation and diagnostic wrapping.
		if err := w.WriteLiteral("0"); err != nil {
			return err
		}
		width--
		if width == 0 {
			return nil
		}
		position := w.bits + w.virtual
		run := min(width, maxBits-position)
		if w.wire.canonical && w.allowZero && (w.bounded && w.bits >= w.limit || w.wire.targetSet && w.bits >= w.wire.targetBits) {
			w.virtual += run
			w.lh.infer(w.bits, run)
			width -= run
			continue
		}
		if w.wire.canonical && w.wire.targetSet {
			run = min(run, w.wire.targetBits-w.bits)
		}
		if w.wire.sealed && w.implicitSpan < len(w.wire.ImplicitSpans) {
			span, end, err := w.currentImplicitSpan()
			if err != nil {
				// The next scalar call supplies exactly the existing wrapper.
				return w.WriteLiteral("0")
			}
			if position >= span.At && position < end {
				run = min(run, end-position)
				w.virtual += run
				w.lh.infer(w.bits, run)
				if position+run == end {
					w.implicitSpan++
				}
				width -= run
				continue
			}
			if position < span.At {
				run = min(run, span.At-position)
			}
		}
		if w.bounded {
			if w.bits >= w.limit {
				return w.WriteLiteral("0")
			}
			run = min(run, w.limit-w.bits)
		}
		if run > 0 {
			newBytes := (w.bits+run+7)/8 - (w.bits+7)/8
			w.bytes = append(w.bytes, make([]byte, newBytes)...)
			w.bits += run
			width -= run
		}
	}
	return nil
}
