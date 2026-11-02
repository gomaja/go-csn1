package registry

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
)

// TS 44.060 V19.0.0 §11.2.6a table 11.2.6a.1a pp14483–14515
// selects the extension length using application-owned DLDC_configuration.
// Exercise the same generated codecs through their Reader/Writer entry points,
// supplying both table rows without changing the context-free descriptor APIs.
func canonicalDLDCDescriptor(d runtime.Descriptor, dual uint64) (runtime.Descriptor, bool) {
	if d.Standard != "TS 44.060" || d.Clause != "11.2.6a" || d.Name != "EGPRS Packet Downlink Ack/Nack message content" {
		return d, false
	}
	minimum, maximum := (d.Extent.MinimumBits+7)/8, d.MaxOctets
	decode := func(wire []byte) (runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent], error) {
		prefix := runtime.NewWriter()
		if err := prefix.WriteUint(0, 14); err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		if err := prefix.WriteBitString(runtime.BitString{Bytes: wire, BitLength: len(wire) * 8}, len(wire)*8); err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		raw, err := prefix.FinishBits(runtime.BitString{})
		if err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, nil)
		if err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		if _, err = r.ReadUint(14); err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		if _, err := r.PushLimit(len(wire) * 8); err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		r.Set("DLDC_configuration", dual)
		start := r.Fork()
		before := r.Wire()
		v, err := ackmessages.DecodeEGPRSPacketDownlinkAckNackMessageContentFrom(r)
		if err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		layout, err := r.ValueWire(start, before, true)
		if err != nil {
			return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{}, err
		}
		consumed := layout.BitsConsumed
		tail := runtime.TrailingBits(wire, consumed)
		v.Wire = runtime.Seal(v, wire, consumed, tail, layout)
		return runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]{Value: v, BitsConsumed: consumed, Tail: tail}, nil
	}
	encode := func(v ackmessages.EGPRSPacketDownlinkAckNackMessageContent) ([]byte, error) {
		probe := runtime.NewWriter()
		probe.WithWire(v.Wire)
		w := runtime.NewWriter()
		if err := w.WriteUint(0, 14); err != nil {
			return nil, err
		}
		w.Set("DLDC_configuration", dual)
		if target, present := probe.RemainingCanonicalTarget(); present {
			v = runtime.CanonicalTo(v, target+14)
			w.WithWire(v.Wire)
			if err := ackmessages.EncodeEGPRSPacketDownlinkAckNackMessageContentTo(w, v); err != nil {
				return nil, err
			}
		} else {
			if probe.Canonical() {
				w.WithWire(v.Wire)
			}
			if err := ackmessages.EncodeEGPRSPacketDownlinkAckNackMessageContentTo(w, v); err != nil {
				return nil, err
			}
		}
		if err := w.PadMinimum(40, false); err != nil {
			return nil, err
		}
		raw, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			return nil, err
		}
		r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, nil)
		if err != nil {
			return nil, err
		}
		if _, err = r.ReadUint(14); err != nil {
			return nil, err
		}
		out, err := r.ReadBitString(r.Remaining())
		if err != nil {
			return nil, err
		}
		return out.Bytes, nil
	}
	d.Decode = func(wire []byte) (any, error) { return decode(wire) }
	d.Encode = func(value any) ([]byte, error) {
		v, ok := value.(ackmessages.EGPRSPacketDownlinkAckNackMessageContent)
		if !ok {
			return nil, fmt.Errorf("wrong value type for DLDC probe: %T", value)
		}
		return encode(v)
	}
	d.CanonicalAtLength = func(value any, octets int) ([]byte, error) {
		v, ok := value.(ackmessages.EGPRSPacketDownlinkAckNackMessageContent)
		if !ok {
			return nil, fmt.Errorf("wrong value type for DLDC probe: %T", value)
		}
		return runtime.CanonicalEncodeAtLength(v, octets, minimum, maximum, encode, decode, nil)
	}
	return d, true
}

func canonicalDLDCFallbackWitness(dual uint64) (ackmessages.EGPRSPacketDownlinkAckNackMessageContent, error) {
	decoded, err := ackmessages.DecodeEGPRSPacketDownlinkAckNackMessageContent([]byte{0, 0, 0, 0})
	if err != nil {
		return ackmessages.EGPRSPacketDownlinkAckNackMessageContent{}, err
	}
	v := runtime.Canonical(decoded.Value)
	width := 3
	if dual == 1 {
		width += 64
	}
	v.EPDANExtensionLengthIndexGroup = &ackmessages.EGPRSPacketDownlinkAckNackMessageContentEPDANExtensionLengthIndexGroup{
		EPDANExtensionLengthIndex: 2,
		Content: ackmessages.EGPRSPacketDownlinkAckNackMessageContentEPDANExtensionLengthIndexGroupEPDANExtensionInfoEPDANExtensionInfoFallback{
			Alternative: ackmessages.EGPRSPacketDownlinkAckNackMessageContentEPDANExtensionLengthIndexGroupEPDANExtensionInfoEPDANExtensionInfoFallbackAlternativeIgnored,
			Ignored:     &runtime.BitString{Bytes: make([]byte, (width+7)/8), BitLength: width},
		},
	}
	return v, nil
}

func TestCanonicalApplicationContextProbes(t *testing.T) {
	var d runtime.Descriptor
	for _, candidate := range descriptors() {
		if _, ok := canonicalDLDCDescriptor(candidate, 0); ok {
			d = candidate
			break
		}
	}
	if d.Decode == nil {
		t.Fatal("missing EGPRS acknowledgement descriptor")
	}
	for _, dual := range []uint64{0, 1} {
		t.Run(fmt.Sprint(dual), func(t *testing.T) {
			contextual, ok := canonicalDLDCDescriptor(d, dual)
			if !ok {
				t.Fatal("context not selected")
			}
			if accepted, err := probeCanonical(contextual, []byte{0, 0, 0, 0}); !accepted || err != nil {
				t.Fatalf("contextual minimum: accepted=%t err=%v", accepted, err)
			}
			decoded, err := contextual.Decode([]byte{0, 0, 0, 0})
			if err != nil {
				t.Fatal(err)
			}
			minimum := decoded.(runtime.Decoded[ackmessages.EGPRSPacketDownlinkAckNackMessageContent]).Value
			if wire, err := contextual.CanonicalAtLength(minimum, 4); err != nil || len(wire) != 4 {
				t.Fatalf("contextual canonical value: wire=%x err=%v", wire, err)
			}
			v, err := canonicalDLDCFallbackWitness(dual)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.CanonicalAtLength(v, 23); !errors.Is(err, runtime.ErrContextRequired) {
				t.Fatalf("absent application context: %v", err)
			}
			_, err = contextual.CanonicalAtLength(v, 23)
			var fallback *runtime.FallbackError
			if !errors.As(err, &fallback) {
				t.Fatalf("known-arm witness needs fallback refusal: %v", err)
			}
		})
	}
	other := d
	other.Clause = "11.2.6"
	if _, selected := canonicalDLDCDescriptor(other, 0); selected {
		t.Fatal("unrelated clause received DLDC context")
	}
	invalid, _ := canonicalDLDCDescriptor(d, 2)
	witness, err := canonicalDLDCFallbackWitness(0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = invalid.CanonicalAtLength(witness, 23)
	var fallback *runtime.FallbackError
	if err == nil || errors.As(err, &fallback) {
		t.Fatalf("invalid DLDC row accepted as known-arm witness: %v", err)
	}
}
