package registry

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover3"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/pshandover2"
)

// These synthetic containing IEs exercise the same cuts through their own
// length fields and canonical entry points, rather than only a supplied bound.
// TS 44.060 V19.0.0 §§11.2.6a, 12.24, 12.42a, 12.48 and 12.64.
func TestContainingCutTotality(t *testing.T) {
	t.Run("12.24", func(t *testing.T) {
		sweepContainingCuts(t, extensionContainer(strings.Repeat("0", 18), 6), ies.DecodeExtensionInformationFrom,
			ies.DecodeGPRSCellOptionsIEFrom, ies.EncodeGPRSCellOptionsIE, ies.EncodeGPRSCellOptionsIECanonicalAtLength,
			func(v *ies.GPRSCellOptionsIE) *runtime.WireInfo { return &v.Wire },
			func(v ies.GPRSCellOptionsIE) *ies.ExtensionInformation { return v.ExtensionLengthGroup.Content.Known }, nil)
	})
	t.Run("12.42a", func(t *testing.T) {
		// Zero-error arm from the clause's synthetic handover fixtures, ending
		// just before the optional extension selector.
		prefix := "0000000000101100000100001000000000000000001010000"
		sweepContainingCuts(t, extensionContainer(prefix, 6), pshandover2.DecodeExtensionInformationFrom,
			pshandover2.DecodePSHandoverRadioResources2IEFrom, pshandover2.EncodePSHandoverRadioResources2IE, pshandover2.EncodePSHandoverRadioResources2IECanonicalAtLength,
			func(v *pshandover2.PSHandoverRadioResources2IE) *runtime.WireInfo { return &v.Wire },
			func(v pshandover2.PSHandoverRadioResources2IE) *pshandover2.ExtensionInformation {
				return v.ExtensionLengthGroup.Content.Known
			}, nil)
	})
	prefix := "000000000000000000000000000000000000000000000000000000000000000000100001000000000000000000000000000000000000001000000"
	t.Run("12.48", func(t *testing.T) {
		sweepContainingCuts(t, extensionContainer(prefix, 6), dtmhandover2.DecodeExtensionInformationFrom,
			dtmhandover2.DecodeDTMHandoverPSRadioResources2IEFrom, dtmhandover2.EncodeDTMHandoverPSRadioResources2IE, dtmhandover2.EncodeDTMHandoverPSRadioResources2IECanonicalAtLength,
			func(v *dtmhandover2.DTMHandoverPSRadioResources2IE) *runtime.WireInfo { return &v.Wire },
			func(v dtmhandover2.DTMHandoverPSRadioResources2IE) *dtmhandover2.ExtensionInformation {
				return v.ExtensionLengthGroup.Content.Known
			},
			func(v dtmhandover2.ExtensionInformation) bool { return runtime.IsZero(v.Group) })
	})
	t.Run("12.64", func(t *testing.T) {
		sweepContainingCuts(t, extensionContainer(prefix, 8), dtmhandover3.DecodeExtensionInformationFrom,
			dtmhandover3.DecodeDTMHandoverPSRadioResources3IEFrom, dtmhandover3.EncodeDTMHandoverPSRadioResources3IE, dtmhandover3.EncodeDTMHandoverPSRadioResources3IECanonicalAtLength,
			func(v *dtmhandover3.DTMHandoverPSRadioResources3IE) *runtime.WireInfo { return &v.Wire },
			func(v dtmhandover3.DTMHandoverPSRadioResources3IE) *dtmhandover3.ExtensionInformation {
				return v.ExtensionLengthGroup.Content.Known
			},
			func(v dtmhandover3.ExtensionInformation) bool { return runtime.IsZero(v.Group) })
	})
	t.Run("11.2.6a", func(t *testing.T) {
		// §11.2.6a table 11.2.6a.1a: DLDC_configuration=0 makes the
		// extension index denote n-1 even for lengths 1..6. The containing
		// control block has 184 bits and positions its content at bit 14.
		input := func(n, ordinal int) runtime.BitString {
			bits := "01000000001000" + "1010110001" + fmt.Sprintf("%06b%0*b", n-1, n, ordinal)
			bits += "1000011111011010101010100"
			for len(bits) < 184 {
				bits += string("00101011"[len(bits)%8])
			}
			return cutBits(bits)
		}
		decode := func(r *runtime.Reader) (ackmessages.EGPRSPacketDownlinkAckNackMessageContent, error) {
			r.Set("DLDC_configuration", 0)
			if _, err := r.ReadUint(14); err != nil {
				return ackmessages.EGPRSPacketDownlinkAckNackMessageContent{}, err
			}
			return ackmessages.DecodeEGPRSPacketDownlinkAckNackMessageContentFrom(r)
		}
		encode := func(v ackmessages.EGPRSPacketDownlinkAckNackMessageContent, canonical bool) ([]byte, error) {
			w := runtime.NewWriter()
			if canonical {
				v = runtime.CanonicalTo(v, 184)
			}
			w.WithWire(v.Wire)
			v.Wire = runtime.WireInfo{}
			w.Set("DLDC_configuration", 0)
			if _, err := w.PushLimit(184); err != nil {
				return nil, err
			}
			if err := w.WriteLiteral("01000000001000"); err != nil {
				return nil, err
			}
			if err := ackmessages.EncodeEGPRSPacketDownlinkAckNackMessageContentTo(w, v); err != nil {
				return nil, err
			}
			out, err := w.FinishBits(runtime.BitString{})
			return out.Bytes, err
		}
		sweepContainingCuts(t, input, ackmessages.DecodeEPDANExtensionInfoFrom, decode,
			func(v ackmessages.EGPRSPacketDownlinkAckNackMessageContent) ([]byte, error) { return encode(v, false) },
			func(v ackmessages.EGPRSPacketDownlinkAckNackMessageContent, _ int) ([]byte, error) {
				return encode(v, true)
			},
			func(v *ackmessages.EGPRSPacketDownlinkAckNackMessageContent) *runtime.WireInfo { return &v.Wire },
			func(v ackmessages.EGPRSPacketDownlinkAckNackMessageContent) *ackmessages.EPDANExtensionInfo {
				return v.EPDANExtensionLengthIndexGroup.Content.Known
			}, nil)
	})
}

func cutBits(bits string) runtime.BitString {
	data := runtime.BitString{Bytes: make([]byte, (len(bits)+7)/8), BitLength: len(bits)}
	for i, bit := range bits {
		if bit == '1' {
			data.Bytes[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return data
}

func extensionContainer(prefix string, width int) func(int, int) runtime.BitString {
	return func(n, ordinal int) runtime.BitString {
		return cutBits(prefix + "1" + fmt.Sprintf("%0*b%0*b", width, n-1, n, ordinal))
	}
}

func sweepContainingCuts[P, E any](t *testing.T, input func(int, int) runtime.BitString, extensionDecode func(*runtime.Reader) (E, error), decode func(*runtime.Reader) (P, error), encode func(P) ([]byte, error), canonical func(P, int) ([]byte, error), wire func(*P) *runtime.WireInfo, extension func(P) *E, unselected func(E) bool) {
	t.Helper()
	var accepted, refused int
	read := func(data runtime.BitString) (P, *runtime.Reader, error) {
		r, err := runtime.NewReaderBits(data.Bytes, data.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.PushLimit(data.BitLength); err != nil {
			t.Fatal(err)
		}
		v, err := decode(r)
		return v, r, err
	}
	for n := 1; n <= 14; n++ {
		for ordinal := 0; ordinal < 1<<n; ordinal++ {
			data := cutBits(fmt.Sprintf("%0*b", n, ordinal))
			r, err := runtime.NewReaderBits(data.Bytes, n, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(n); err != nil {
				t.Fatal(err)
			}
			ext, err := extensionDecode(r)
			if err != nil {
				continue
			}
			data = input(n, ordinal)
			// The containing byte API includes its final backing-octet tail;
			// the extension's own length still bounds it to exactly n bits.
			data.BitLength = len(data.Bytes) * 8
			value, r, err := read(data)
			if err != nil {
				t.Fatalf("containing decode at length %d ordinal %d: %v", n, ordinal, err)
			}
			known := extension(value)
			if known == nil || !runtime.SemanticallyEqual(ext, *known) {
				t.Fatalf("containing IE lost the known extension at length %d ordinal %d", n, ordinal)
			}
			*wire(&value), err = runtime.SealBits(value, data, r.Position(), r.Tail(), r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			out, err := encode(value)
			if err != nil || !bytes.Equal(data.Bytes, out) {
				t.Fatalf("containing replay at length %d ordinal %d: %v", n, ordinal, err)
			}
			out, err = canonical(value, len(data.Bytes))
			if err != nil {
				var bound *runtime.BoundError
				if unselected != nil && unselected(ext) && errors.As(err, &bound) && bound.Kind == runtime.LengthBound {
					refused++
					continue
				}
				t.Fatalf("containing canonical at length %d ordinal %d: %v", n, ordinal, err)
			}
			again, _, err := read(runtime.BitString{Bytes: out, BitLength: len(out) * 8})
			if err != nil || !runtime.SemanticallyEqual(value, again) {
				t.Fatalf("containing canonical decode/semantics at length %d ordinal %d: %v", n, ordinal, err)
			}
			accepted++
		}
	}
	t.Logf("canonical_successes=%d typed_refusals=%d", accepted, refused)
	if accepted == 0 {
		t.Fatal("no containing canonical successes")
	}
}
