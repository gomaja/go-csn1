package controlblock

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestAccessBurstFormats(t *testing.T) {
	// TS 44.060 V19.0.0 §10.0b.2.1 supplies the 8/11-bit forms.
	// TS 45.003 V19.0.0 §4.6.2.1 carries 11 + 19 = 30 information bits;
	// table 11.2.5a.4.1 of TS 44.060 prints the 19-bit continuation.
	for _, tc := range []struct {
		name string
		b    []byte
		n    int
		want []byte
	}{
		{"eight", []byte{0xa5}, 8, []byte{0xa5}},
		{"eleven", []byte{0xa5, 0xff}, 11, []byte{0xa5, 0xe0}},
		{"extended", []byte{0xa5, 0x7b, 0xc9, 0xff}, 30, []byte{0xa5, 0x7b, 0xc9, 0xfc}},
		{"extended_last_bit", []byte{0, 0, 0, 4}, 30, []byte{0, 0, 0, 4}},
		{"extended_initial_part", []byte{0xff, 0xe0, 0, 0}, 30, []byte{0xff, 0xe0, 0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, err := DecodeAccessBurst(tc.b, tc.n)
			if err != nil {
				t.Fatalf("decode %d bits: %v", tc.n, err)
			}
			if v.Bits.BitLength != tc.n || !bytes.Equal(v.Bits.Bytes, tc.want) {
				t.Fatal("decoded significant bits differ")
			}
			// Encode independently of decode so a decoder that drops the
			// continuation cannot conceal the same error in the encoder.
			for _, value := range []AccessBurst{v, {Bits: runtime.BitString{Bytes: tc.b, BitLength: tc.n}}} {
				b, err := EncodeAccessBurst(value)
				if err != nil || b.BitLength != tc.n || !bytes.Equal(b.Bytes, tc.want) {
					t.Fatalf("encode %d bits differs: %v", tc.n, err)
				}
			}
		})
	}
}

func TestAccessBurstRejectsUnprintedExtents(t *testing.T) {
	// TS 44.018 V19.0.0 §9.1.65 tables 9.1.65.2/4 place the 8/10-bit
	// bodies inside 11-bit messages. The 19-bit continuation in table
	// 9.1.65.2.1 is not a complete carrier either (TS 45.003 §4.6.2.1).
	for n := -1; n <= 64; n++ {
		if n == 8 || n == 11 || n == 30 {
			continue
		}
		data := make([]byte, 8)
		if _, err := DecodeAccessBurst(data, n); err == nil {
			t.Errorf("decode accepted unprinted extent %d", n)
		} else {
			var located *runtime.DecodeError
			if !errors.As(err, &located) || located.Kind != runtime.InvalidValue {
				t.Errorf("decode extent %d lacks invalid-value error: %v", n, err)
			}
		}
		if _, err := EncodeAccessBurst(AccessBurst{Bits: runtime.BitString{Bytes: data, BitLength: n}}); err == nil {
			t.Errorf("encode accepted unprinted extent %d", n)
		}
	}
}

func TestAccessBurstRejectsTruncatedStorage(t *testing.T) {
	for _, n := range []int{8, 11, 30} {
		for size := 0; size < (n+7)/8; size++ {
			data := make([]byte, size)
			if _, err := DecodeAccessBurst(data, n); err == nil {
				t.Errorf("decode accepted %d bytes for %d bits", size, n)
			}
			if _, err := EncodeAccessBurst(AccessBurst{Bits: runtime.BitString{Bytes: data, BitLength: n}}); err == nil {
				t.Errorf("encode accepted %d bytes for %d bits", size, n)
			}
		}
	}
}

func TestAccessBurstIgnoresStorageSuffix(t *testing.T) {
	for _, size := range []int{4, 32, 4096} {
		data := bytes.Repeat([]byte{0xff}, size)
		v, err := DecodeAccessBurst(data, 30)
		if err != nil || v.Bits.BitLength != 30 || !bytes.Equal(v.Bits.Bytes, []byte{0xff, 0xff, 0xff, 0xfc}) {
			t.Fatalf("storage suffix changed the 30-bit extent at %d bytes: %v", size, err)
		}
		out, err := EncodeAccessBurst(AccessBurst{Bits: runtime.BitString{Bytes: data, BitLength: 30}})
		if err != nil || out.BitLength != 30 || !bytes.Equal(out.Bytes, []byte{0xff, 0xff, 0xff, 0xfc}) {
			t.Fatalf("storage suffix changed the encoded extent at %d bytes: %v", size, err)
		}
	}
}

func FuzzAccessBurst(f *testing.F) {
	f.Add([]byte{0xa5, 0xe0}, 11)
	f.Add([]byte{0xff}, 8)
	f.Add([]byte{0xa5, 0x7b, 0xc9, 0xff}, 30)
	f.Add([]byte{0, 0, 0, 4}, 30)
	f.Add([]byte{0xff, 0xff, 0xff}, 30)
	f.Add([]byte{0xff, 0xff}, 10)
	f.Add([]byte{0xff, 0xff, 0xff}, 19)
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}, 29)
	f.Add([]byte{0xff, 0xff, 0xff, 0xff}, 31)
	f.Fuzz(func(t *testing.T, b []byte, n int) {
		v, decodeErr := DecodeAccessBurst(b, n)
		encoded, encodeErr := EncodeAccessBurst(AccessBurst{Bits: runtime.BitString{Bytes: b, BitLength: n}})
		if n != 8 && n != 11 && n != 30 || len(b) < (n+7)/8 {
			if decodeErr == nil || encodeErr == nil {
				t.Fatalf("accepted invalid extent %d with %d bytes", n, len(b))
			}
			return
		}
		var limit *runtime.DecodeError
		if errors.As(decodeErr, &limit) && limit.Kind == runtime.Limit {
			return // Backing input can exceed the runtime input ceiling.
		}
		if decodeErr != nil || encodeErr != nil {
			t.Fatalf("rejected valid extent %d: decode %v, encode %v", n, decodeErr, encodeErr)
		}
		want := append([]byte(nil), b[:(n+7)/8]...)
		if n%8 != 0 {
			want[len(want)-1] &= 0xff << (8 - uint(n%8))
		}
		if v.Bits.BitLength != n || !bytes.Equal(v.Bits.Bytes, want) || encoded.BitLength != n || !bytes.Equal(encoded.Bytes, want) {
			t.Fatal("significant bits differ")
		}
		out, err := EncodeAccessBurst(v)
		if err != nil || out.BitLength != n || !bytes.Equal(out.Bytes, want) {
			t.Fatalf("round trip changed significant bits: %v", err)
		}
	})
}
