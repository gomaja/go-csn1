package runtime

// RemainingCanonicalTarget reports the bits left in an explicit canonical
// target. It does not report an ordinary octet completion, containing writer
// limit, or received extent. A nil writer or invalid target returns (0, false).
// An exhausted target returns (0, true). A local limit does not narrow the
// target; callers must also respect RemainingLimit. The query changes no state.
// This permits whole-octet remainders such as TS 44.060 V19.0.0 §11.2.17c
// to use the explicit extent supplied by CanonicalEncodeAtLength.
func (w *Writer) RemainingCanonicalTarget() (remaining int, present bool) {
	if w == nil || !w.wire.canonical || !w.wire.targetSet || w.bits < 0 || w.bits > w.wire.targetBits || w.wire.targetBits > maxBits {
		return 0, false
	}
	return w.wire.targetBits - w.bits, true
}
