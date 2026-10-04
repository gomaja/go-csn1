package ies

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// gprsCellOptionsWithIgnored builds, bit by bit, a GPRS Cell Options IE
// with zero fields, no PAN group and an Extension Length covering the
// given ignored bits (TS 44.060 V19.0.0 §12.24).
func gprsCellOptionsWithIgnored(ignored runtime.BitString) []byte {
	bits := make([]byte, 0, 25+ignored.BitLength)
	bits = append(bits, make([]byte, 17)...) // NMO through BS_CV_MAX
	bits = append(bits, 0, 1)                // no PAN group, extension present
	for i := 5; i >= 0; i-- {
		bits = append(bits, byte(ignored.BitLength-1)>>uint(i)&1)
	}
	for i := range ignored.BitLength {
		bits = append(bits, ignored.Bytes[i/8]>>uint(7-i%8)&1)
	}
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		out[i/8] |= b << uint(7-i%8)
	}
	return out
}

// The decoder keeps extension bits as ignored only when Extension
// Information fails on them. The encoders accept an ignored value exactly
// when its bytes decode back to it, and otherwise return FallbackError.
func TestIgnoredExtensionBitsThatDecodeAsKnownAreRejected(t *testing.T) {
	rng := rand.New(rand.NewSource(1224))
	outcomes := map[bool]int{}
	for range 4000 {
		n := 1 + rng.Intn(64)
		raw := make([]byte, (n+7)/8)
		rng.Read(raw)
		if n%8 != 0 {
			raw[len(raw)-1] &= byte(0xff << uint(8-n%8))
		}
		ignored := runtime.BitString{Bytes: raw, BitLength: n}
		wire := gprsCellOptionsWithIgnored(ignored)
		decoded, err := DecodeGPRSCellOptionsIE(wire)
		keepsIgnored := err == nil && decoded.Value.ExtensionLengthGroup != nil && decoded.Value.ExtensionLengthGroup.Content.Ignored != nil
		outcomes[keepsIgnored]++

		value := GPRSCellOptionsIE{ExtensionLengthGroup: &GPRSCellOptionsIEExtensionLengthGroup{
			ExtensionLength: uint8(n - 1),
			Content: GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{
				Alternative: GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallbackAlternativeIgnored,
				Ignored:     &ignored,
			},
		}}
		plain, plainErr := EncodeGPRSCellOptionsIE(value)
		canonical, canonicalErr := EncodeGPRSCellOptionsIECanonicalAtLength(value, len(wire))
		for name, result := range map[string]struct {
			encoded []byte
			err     error
		}{"plain": {plain, plainErr}, "canonical": {canonical, canonicalErr}} {
			if keepsIgnored {
				if result.err != nil || !bytes.Equal(result.encoded, wire) {
					t.Fatalf("%s %x: %x, %v; want the input", name, wire, result.encoded, result.err)
				}
				continue
			}
			var fallback *runtime.FallbackError
			if !errors.As(result.err, &fallback) || !errors.Is(result.err, runtime.ErrFallbackKnown) || fallback.Position != 25 {
				t.Fatalf("%s %x: %v; want FallbackError at bit 25", name, wire, result.err)
			}
		}
	}
	if outcomes[true] == 0 || outcomes[false] == 0 {
		t.Fatalf("outcomes %v do not cover both arms", outcomes)
	}
	t.Logf("kept as ignored %d, decoded through the known arm %d", outcomes[true], outcomes[false])
}
