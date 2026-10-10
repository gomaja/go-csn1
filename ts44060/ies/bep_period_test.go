package ies

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestReservedBEPPeriodInR99Extension(t *testing.T) {
	// Synthetic bounded-truncation reproducer from the extension tests.
	// Its BEP_PERIOD 13 is reserved even though its layout is well formed.
	wire, err := hex.DecodeString("b0e1d5122d103fc76dfdb8ebeb652a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = DecodeGPRSCellOptionsIE(wire)
	requireDelegatedValueError(t, err, true)
}

// TS 44.060 V19.0.0 §12.24 table 12.24.2 delegates BEP_PERIOD to
// TS 45.008 V19.0.0 §10.2.3.2.1: codes 11–15 are reserved, without
// a receiver interpretation. These synthetic vectors also decode in
// pycrate 0.7.11 with the same BEP_PERIOD and replay byte for byte.
func TestDelegatedBEPPeriodReservation(t *testing.T) {
	for _, layout := range []struct {
		name string
		wire []byte
	}{
		{"R99", []byte{0, 0, 0x24, 0x40, 0}},
		{"Rel4", []byte{0, 0, 0x25, 0x40, 0}},
		{"complete", []byte{0, 0, 0x28, 0xc0, 0, 0}},
	} {
		for code := uint8(0); code < 16; code++ {
			t.Run(fmt.Sprintf("%s/%d", layout.name, code), func(t *testing.T) {
				wire := bytes.Clone(layout.wire)
				wire[3] |= code << 1
				t.Run("decode", func(t *testing.T) {
					d, err := DecodeGPRSCellOptionsIE(wire)
					if code > 10 {
						requireDelegatedValueError(t, err, true)
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					known := d.Value.ExtensionLengthGroup.Content.Known
					if known == nil || known.Group.EGPRSPACKETCHANNELREQUESTGroup == nil || known.Group.EGPRSPACKETCHANNELREQUESTGroup.BEPPERIOD != code {
						t.Fatalf("BEP_PERIOD %d lost or decoded as ignored", code)
					}
				})
				for _, canonical := range []bool{false, true} {
					t.Run(fmt.Sprintf("canonical=%t", canonical), func(t *testing.T) {
						d, err := DecodeGPRSCellOptionsIE(layout.wire)
						if err != nil {
							t.Fatal(err)
						}
						d.Value.ExtensionLengthGroup.Content.Known.Group.EGPRSPACKETCHANNELREQUESTGroup.BEPPERIOD = code
						var out []byte
						if canonical {
							out, err = EncodeGPRSCellOptionsIECanonicalAtLength(d.Value, len(wire))
						} else {
							out, err = EncodeGPRSCellOptionsIE(d.Value)
						}
						if code > 10 {
							requireDelegatedValueError(t, err, false)
						} else if err != nil || !bytes.Equal(out, wire) {
							t.Fatalf("BEP_PERIOD %d: %x, want %x: %v", code, out, wire, err)
						}
					})
				}
			})
		}
	}
}

func TestIgnoredBEPPeriodReservation(t *testing.T) {
	// The ! fallback in TS 44.060 V19.0.0 §12.24 does not turn a
	// syntactically incorrect reserved value (§11.1) into an unknown IE.
	for code := uint8(11); code < 16; code++ {
		for _, canonical := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/canonical=%t", code, canonical), func(t *testing.T) {
				v := GPRSCellOptionsIE{ExtensionLengthGroup: &GPRSCellOptionsIEExtensionLengthGroup{
					ExtensionLength: 17,
					Content: GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{
						Alternative: GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallbackAlternativeIgnored,
						Ignored:     &runtime.BitString{Bytes: []byte{0x80 | code<<2, 0, 0}, BitLength: 18},
					},
				}}
				var err error
				if canonical {
					_, err = EncodeGPRSCellOptionsIECanonicalAtLength(v, 6)
				} else {
					_, err = EncodeGPRSCellOptionsIE(v)
				}
				requireDelegatedValueError(t, err, false)
			})
		}
	}
}

func TestStandaloneBEPPeriodReservation(t *testing.T) {
	// Same delegated table applies to the standalone Extension Information.
	for code := uint8(0); code < 16; code++ {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			wire := []byte{0x80 | code<<2, 0, 0}
			t.Run("decode", func(t *testing.T) {
				d, err := DecodeExtensionInformation(wire)
				if code > 10 {
					requireDelegatedValueError(t, err, true)
				} else if err != nil || d.Value.Group.EGPRSPACKETCHANNELREQUESTGroup == nil || d.Value.Group.EGPRSPACKETCHANNELREQUESTGroup.BEPPERIOD != code {
					t.Fatalf("BEP_PERIOD %d: %+v: %v", code, d.Value, err)
				}
			})
			for _, canonical := range []bool{false, true} {
				t.Run(fmt.Sprintf("canonical=%t", canonical), func(t *testing.T) {
					d, err := DecodeExtensionInformation([]byte{0x80, 0, 0})
					if err != nil {
						t.Fatal(err)
					}
					d.Value.Group.EGPRSPACKETCHANNELREQUESTGroup.BEPPERIOD = code
					var out []byte
					if canonical {
						out, err = EncodeExtensionInformationCanonicalAtLength(d.Value, len(wire))
					} else {
						out, err = EncodeExtensionInformation(d.Value)
					}
					if code > 10 {
						requireDelegatedValueError(t, err, false)
					} else if err != nil || !bytes.Equal(out, wire) {
						t.Fatalf("BEP_PERIOD %d: %x, want %x: %v", code, out, wire, err)
					}
				})
			}
		})
	}
}
