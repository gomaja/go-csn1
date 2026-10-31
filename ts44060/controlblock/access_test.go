package controlblock

import (
	"bytes"
	"testing"
)

func TestAccessBurstFormats(t *testing.T) {
	// §10.0b.2.1 maps the entire 8/11-bit control message directly.
	// No six-bit MESSAGE_TYPE is added, including EGPRS/MPRACH/EC forms.
	for _, tc := range []struct {
		b    []byte
		n    int
		want []byte
	}{{[]byte{0xa5}, 8, []byte{0xa5}}, {[]byte{0xa5, 0xff}, 11, []byte{0xa5, 0xe0}}} {
		v, err := DecodeAccessBurst(tc.b, tc.n)
		if err != nil || v.Bits.BitLength != tc.n {
			t.Fatal(v, err)
		}
		b, err := EncodeAccessBurst(v)
		if err != nil || b.BitLength != tc.n || !bytes.Equal(b.Bytes, tc.want) {
			t.Fatal(b, err)
		}
	}
	for _, n := range []int{-1, 0, 7, 9, 10, 12, 16} {
		if _, err := DecodeAccessBurst([]byte{0xff, 0xff}, n); err == nil {
			t.Fatalf("accepted %d", n)
		}
	}
	if _, err := DecodeAccessBurst([]byte{0xff}, 11); err == nil {
		t.Fatal("truncated 11-bit burst")
	}
}

func FuzzAccessBurst(f *testing.F) {
	f.Add([]byte{0xa5, 0xe0}, 11)
	f.Add([]byte{0xff}, 8)
	f.Fuzz(func(t *testing.T, b []byte, n int) {
		v, err := DecodeAccessBurst(b, n)
		if err != nil {
			return
		}
		out, err := EncodeAccessBurst(v)
		if err != nil || out.BitLength != n || out.BitLength != 8 && out.BitLength != 11 {
			t.Fatal(out, err)
		}
	})
}
