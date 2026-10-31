package runtime

import "fmt"

// NewReaderBits constructs a reader with an explicit containing extent. Bits
// outside bitLength, including unused backing-octet bits, are not input. The
// absolute origin and L/H phase remain those of data. TS 44.060 V19.0.0
// §11.2.2 defines the non-octet access-burst forms; TS 24.007 V20.0.0
// Annex B §B.1.2.2 defines positioned L/H symbols.
func NewReaderBits(data []byte, bitLength int, c *Collector) (*Reader, error) {
	if err := checkBitExtent(data, bitLength); err != nil {
		return nil, err
	}
	r := NewReader(data)
	r.end, r.boundDepth = bitLength, 1
	r.attachCollector(c)
	return r, nil
}

// NewChoiceReaderBits co-locates state retained by generated choices while
// preserving the explicit containing extent, as in NewReaderBits.
func NewChoiceReaderBits(data []byte, bitLength int, c *Collector) (*Reader, error) {
	if err := checkBitExtent(data, bitLength); err != nil {
		return nil, err
	}
	r := NewChoiceReader(data, nil)
	r.end, r.boundDepth = bitLength, 1
	r.attachCollector(c)
	return r, nil
}

func checkBitExtent(data []byte, bitLength int) error {
	if err := CheckInput(data); err != nil {
		return err
	}
	if bitLength < 0 || bitLength > len(data)*8 {
		return &DecodeError{Kind: InvalidValue, Offset: 0, Detail: "bit extent outside backing input"}
	}
	return nil
}

// SealBits captures an exact received extent. The tail must describe the bits
// after the semantic boundary. Received inferred zeros continue to use
// WireInfo.ImplicitSpans; a fresh sender emits those semantic bits normally.
func SealBits[T any](value T, raw BitString, consumed int, tail BitString, wire WireInfo) (WireInfo, error) {
	if err := checkBitExtent(raw.Bytes, raw.BitLength); err != nil {
		return WireInfo{}, err
	}
	if err := checkBitExtent(tail.Bytes, tail.BitLength); err != nil {
		return WireInfo{}, err
	}
	if consumed < 0 || consumed > raw.BitLength || tail.BitLength != raw.BitLength-consumed {
		return WireInfo{}, &DecodeError{Kind: InvalidValue, Offset: 0, Detail: "inconsistent received bit boundary or tail"}
	}
	wire = Seal(value, raw.Bytes, consumed, tail, wire)
	wire.TransmittedBits = raw.BitLength
	return wire, nil
}

// FinishBits appends a value's tail and returns exactly the significant bits,
// without octet completion. Unused low bits in the final backing byte are zero.
// Byte-entry encoders retain Finish's existing octet-completion policy.
func (w *Writer) FinishBits(tail BitString) (BitString, error) {
	if w.wire.sealed {
		if err := w.validateValueSemantic(false); err != nil {
			return BitString{}, w.operationFailure(err, 0, InvalidValue)
		}
		if tail.BitLength != w.wire.Tail.BitLength {
			return BitString{}, w.operationFailure(fmt.Errorf("wire tail length differs from received boundary"), 0, InvalidValue)
		}
	}
	if err := w.WriteBitString(tail, tail.BitLength); err != nil {
		return BitString{}, err
	}
	if w.stateErr != nil {
		return BitString{}, w.OperationError(InvalidValue, w.stateErrPath, 0, w.stateErr)
	}
	if w.wire.sealed {
		if err := w.validateValueState(false); err != nil {
			return BitString{}, w.operationFailure(err, 0, InvalidValue)
		}
		if w.bits != w.wire.TransmittedBits {
			return BitString{}, w.operationFailure(fmt.Errorf("encoded length %d differs from transmitted %d", w.bits, w.wire.TransmittedBits), 0, InvalidValue)
		}
	}
	return BitString{Bytes: append([]byte(nil), w.bytes...), BitLength: w.bits}, nil
}

// DecodeBits uses the descriptor's typed reader entry point with an explicit
// containing extent. It does not change the policy of the byte entry point.
func (d Descriptor) DecodeBits(data []byte, bitLength int) (any, error) {
	if d.DecodeFrom == nil {
		return nil, ErrUnsupported
	}
	r, err := NewChoiceReaderBits(data, bitLength, nil)
	if err != nil {
		return nil, err
	}
	return d.DecodeFrom(r)
}

// DecodeBitsWithReport returns reserved observations and receiver diagnostics
// separately, including the committed prefix when decoding fails.
func (d Descriptor) DecodeBitsWithReport(data []byte, bitLength int) (any, Report, ReceiverReport, error) {
	if d.DecodeFrom == nil {
		return nil, Report{}, ReceiverReport{}, ErrUnsupported
	}
	c := NewReceiverCollector()
	r, err := NewChoiceReaderBits(data, bitLength, c)
	if err != nil {
		return nil, Report{}, ReceiverReport{}, err
	}
	value, err := d.DecodeFrom(r)
	return value, c.TakeReport(), c.TakeReceiverReport(), err
}
