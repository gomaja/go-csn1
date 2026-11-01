package access_test

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
)

func FuzzHandoverAccessReplay(f *testing.F) {
	for _, v := range [][]byte{{0, 0}, {255, 255}, {0x81, 0xe0}} {
		f.Add(v, uint8(11))
	}
	f.Fuzz(func(t *testing.T, data []byte, limit uint8) {
		if len(data) > 64 {
			return
		}
		bits := int(limit) % 16
		if bits > len(data)*8 {
			bits = len(data) * 8
		}
		for _, width := range []int{8, 11} {
			r, err := runtime.NewReaderBits(data, bits, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(bits); err != nil {
				t.Fatal(err)
			}
			if width == 8 {
				v, err := access.DecodePSHandoverAccessMessageContent8BitMessageFrom(r)
				if bits < width {
					if err == nil {
						t.Fatal("short 8-bit access accepted")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				image := accessEncode(t, v, access.EncodePSHandoverAccessMessageContent8BitMessageTo)
				if image.BitLength != width || !bytes.Equal(image.Bytes, data[:1]) {
					t.Fatal("8-bit access replay changed")
				}
			} else {
				v, err := access.DecodePSHandoverAccessMessageContent11BitMessageFrom(r)
				if bits < width {
					if err == nil {
						t.Fatal("short 11-bit access accepted")
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				expected := []byte{data[0], data[1] & 0xe0}
				v.Wire, err = runtime.SealBits(v, runtime.BitString{Bytes: expected, BitLength: width}, width, runtime.BitString{}, r.Wire())
				if err != nil {
					t.Fatal(err)
				}
				image := accessEncode(t, v, access.EncodePSHandoverAccessMessageContent11BitMessageTo)
				if image.BitLength != width || !bytes.Equal(image.Bytes, expected) {
					t.Fatal("11-bit access replay changed")
				}
				canonical := accessEncode(t, runtime.Canonical(v), access.EncodePSHandoverAccessMessageContent11BitMessageTo)
				if canonical.BitLength != width || !bytes.Equal(canonical.Bytes, []byte{data[0], 0}) {
					t.Fatal("sender spare bits changed")
				}
			}
			if r.Position() != width {
				t.Fatal("access crossed its printed extent")
			}
		}
	})
}
