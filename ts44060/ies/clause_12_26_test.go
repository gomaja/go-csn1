package ies

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 Table 12.26.1, paragraphs 23453–23455:
// the six-bit code is followed by code+1 sender-zero/receiver-any spares.
// The sender and receiver rule is §10.1, paragraph 7409.
func TestExtensionBits(t *testing.T) {
	for code := 0; code < 64; code++ {
		for _, pattern := range []string{"0", "1", "10"} {
			t.Run(fmt.Sprintf("length-%d-spare-%s", code, pattern), func(t *testing.T) {
				spare := strings.Repeat(pattern, code+1)[:code+1]
				text := fmt.Sprintf("%06b", code) + spare
				wire := clausePack(text)
				decoded, err := DecodeExtensionBitsIE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if decoded.Value.ExtensionLength != uint8(code) || decoded.BitsConsumed != 7+code {
					t.Fatalf("typed length or extent: %+v", decoded)
				}
				replay, err := EncodeExtensionBitsIE(decoded.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x: %v", replay, err)
				}
				zeroWire := clausePack(fmt.Sprintf("%06b", code) + strings.Repeat("0", code+1))
				canonical, err := EncodeExtensionBitsIECanonicalAtLength(decoded.Value, len(wire))
				if err != nil || !bytes.Equal(canonical, zeroWire) {
					t.Fatalf("canonical %x want %x: %v", canonical, zeroWire, err)
				}
				fresh, err := EncodeExtensionBitsIE(ExtensionBitsIE{ExtensionLength: uint8(code)})
				if err != nil || !bytes.Equal(fresh, zeroWire) {
					t.Fatalf("fresh %x want %x: %v", fresh, zeroWire, err)
				}
				// Exact container limits protect a following sentinel at every bit phase.
				for phase := 0; phase < 8; phase++ {
					r := runtime.NewReader(clausePack(strings.Repeat("1", phase) + text + "10101"))
					if _, err := r.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(len(text))
					if err != nil {
						t.Fatal(err)
					}
					v, err := DecodeExtensionBitsIEFrom(r)
					if err != nil || v.ExtensionLength != uint8(code) || r.Remaining() != 0 {
						t.Fatalf("bounded phase %d: %+v %v", phase, v, err)
					}
					if err := r.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					sentinel, err := r.ReadUint(5)
					if err != nil || sentinel != 21 {
						t.Fatalf("sentinel %d: %v", sentinel, err)
					}
				}
				for cut := 0; cut < len(text); cut++ {
					r := runtime.NewReader(wire)
					if _, err := r.PushLimit(cut); err != nil {
						t.Fatal(err)
					}
					if _, err := DecodeExtensionBitsIEFrom(r); err == nil {
						t.Fatalf("accepted strict prefix %d/%d", cut, len(text))
					}
					if r.Position() > cut {
						t.Fatalf("crossed prefix %d", cut)
					}
				}
				edited := decoded.Value
				edited.ExtensionLength = uint8((code + 1) % 64)
				output, err := EncodeExtensionBitsIECanonicalAtLength(edited, (int(edited.ExtensionLength)+14)/8)
				if err != nil {
					t.Fatal(err)
				}
				again, err := DecodeExtensionBitsIE(output)
				if err != nil || again.Value.ExtensionLength != edited.ExtensionLength {
					t.Fatalf("length edit: %+v %v", again, err)
				}
			})
		}
	}
	if _, err := EncodeExtensionBitsIE(ExtensionBitsIE{ExtensionLength: 64}); err == nil {
		t.Fatal("seven-bit length encoded")
	}
}

func TestExtensionBitsRequiresCanonicalExtent(t *testing.T) {
	_, err := EncodeExtensionBitsIECanonical(ExtensionBitsIE{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("required canonical extent: %v", err)
	}
}
