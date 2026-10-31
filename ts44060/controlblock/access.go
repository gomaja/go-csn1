package controlblock

import "github.com/gomaja/go-csn1/runtime"

// AccessBurst is the complete physical 8/11-bit container. It does not claim
// to decode the request/acknowledgement body. TS 44.060 V19.0.0 §§10.0b.2.1,
// 11.2.0.2 maps these directly; §§11.2.2/5/5a/5c/33/44/50 define the bodies.
// An access burst has no external control-message selector.
type AccessBurst struct{ Bits runtime.BitString }

// DecodeAccessBurst reads exactly the significant physical extent. Storage
// bits below that extent are not part of the burst (§10.0b.1).
func DecodeAccessBurst(data []byte, significantBits int) (AccessBurst, error) {
	if significantBits != 8 && significantBits != 11 {
		return AccessBurst{}, invalidHeader("access burst must have 8 or 11 significant bits (§10.0b.2.1)")
	}
	r, err := runtime.NewReaderBits(data, significantBits, nil)
	if err != nil {
		return AccessBurst{}, err
	}
	bits, err := r.ReadBitString(significantBits)
	if err != nil {
		return AccessBurst{}, err
	}
	return AccessBurst{Bits: bits}, nil
}

// EncodeAccessBurst returns the same significant bits, with unused low storage
// bits cleared. A body codec must separately establish its access-format rule.
func EncodeAccessBurst(burst AccessBurst) (runtime.BitString, error) {
	if burst.Bits.BitLength != 8 && burst.Bits.BitLength != 11 {
		return runtime.BitString{}, invalidHeader("access burst must have 8 or 11 significant bits (§10.0b.2.1)")
	}
	w := runtime.NewWriter()
	if err := w.WriteBitString(burst.Bits, burst.Bits.BitLength); err != nil {
		return runtime.BitString{}, err
	}
	return w.FinishBits(runtime.BitString{})
}
