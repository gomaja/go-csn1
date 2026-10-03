package registry

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts24008/msnetcap"
	"github.com/gomaja/go-csn1/ts24008/msrac"
)

func TestCapabilityValueReaderOffsets(t *testing.T) {
	// TS 24.007 V20.0.0 §11.4.2 applies a value-part extent from
	// the value start, including when the IE sits inside a larger PDU.
	tests := []struct {
		name       string
		wire       []byte
		decodeFrom func(*runtime.Reader) (any, error)
		descriptor runtime.Descriptor
	}{
		{
			name: "MS RA capability",
			wire: append([]byte{0x10, 0xb1, 0}, bytes.Repeat([]byte{0x2b}, 47)...),
			decodeFrom: func(r *runtime.Reader) (any, error) {
				return msrac.DecodeMSRACapabilityValuePartFrom(r)
			},
			descriptor: msrac.Descriptors()[0],
		},
		{
			name: "Classmark 3",
			wire: append([]byte{0x60}, make([]byte, 31)...),
			decodeFrom: func(r *runtime.Reader) (any, error) {
				return classmark.DecodeClassmark3ValuePartFrom(r)
			},
			descriptor: classmark.Descriptors()[0],
		},
		{
			name: "MS network capability",
			wire: []byte{0x75, 0x60, 0x3e, 0, 0, 0, 0, 0},
			decodeFrom: func(r *runtime.Reader) (any, error) {
				return msnetcap.DecodeMSNetworkCapabilityValuePartFrom(r)
			},
			descriptor: msnetcap.Descriptors()[0],
		},
	}
	for _, tc := range tests {
		for _, prefixBits := range []int{0, 8, 64, 512} {
			for _, entry := range []struct {
				name   string
				decode func(*runtime.Reader) (any, error)
			}{{"From", tc.decodeFrom}, {"Descriptor.DecodeFrom", tc.descriptor.DecodeFrom}} {
				t.Run(tc.name+"/"+entry.name+"/prefix="+strconv.Itoa(prefixBits), func(t *testing.T) {
					input := append(make([]byte, prefixBits/8), tc.wire...)
					r := runtime.NewReader(input)
					for i := 0; i < prefixBits/8; i++ {
						if _, err := r.ReadUint(8); err != nil {
							t.Fatal(err)
						}
					}
					if _, err := entry.decode(r); err != nil {
						t.Fatal(err)
					}
					if consumed := r.Position() - prefixBits; consumed != len(tc.wire)*8 {
						t.Fatalf("consumed %d bits from offset %d, want %d", consumed, prefixBits, len(tc.wire)*8)
					}
				})
			}
		}
	}
}
