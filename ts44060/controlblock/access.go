package controlblock

import "github.com/gomaja/go-csn1/runtime"

// AccessBurst is the complete 8-, 11- or 30-information-bit access container.
// TS 44.060 V19.0.0 §10.0b.2.1 defines the 8/11-bit forms. TS 44.018
// V19.0.0 §9.1.65 tables 9.1.65.2/4 place the EC request's 8/10-bit bodies
// inside 11-bit messages, with the table's surrounding bits;
// TS 45.003 V19.0.0 §§5.3a.1, 5.3a.6.1, 5.3a.7.1 confirm that carrier.
// The 30-bit extended access is 11 initial bits plus the 19-bit continuation
// in TS 44.060 table 11.2.5a.4.1 or TS 44.018 table 9.1.65.2.1, carried
// as specified in TS 45.003 §§4.6.2.1, 5.3a.5. No body selector is added;
// decoding the request/acknowledgement body remains the body codec's job.
type AccessBurst struct{ Bits runtime.BitString }

// DecodeAccessBurst reads exactly the significant carrier information bits.
// Storage bits below that extent are not part of the burst (§10.0b.1).
func DecodeAccessBurst(data []byte, significantBits int) (AccessBurst, error) {
	if significantBits != 8 && significantBits != 11 && significantBits != 30 {
		return AccessBurst{}, invalidHeader("access burst must have 8, 11 or 30 significant bits (TS 44.060 §10.0b.2.1; TS 45.003 §4.6.2.1)")
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
	if burst.Bits.BitLength != 8 && burst.Bits.BitLength != 11 && burst.Bits.BitLength != 30 {
		return runtime.BitString{}, invalidHeader("access burst must have 8, 11 or 30 significant bits (TS 44.060 §10.0b.2.1; TS 45.003 §4.6.2.1)")
	}
	w := runtime.NewWriter()
	if err := w.WriteBitString(burst.Bits, burst.Bits.BitLength); err != nil {
		return runtime.BitString{}, err
	}
	return w.FinishBits(runtime.BitString{})
}
