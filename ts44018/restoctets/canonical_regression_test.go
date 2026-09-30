package restoctets

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestFreshFixedIERestPadding(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.17, 10.5.2.33 require
	// three and one value octets respectively, filled with L/H spare.
	iar, err := DecodeIARRestOctets([]byte{0x2b, 0x2b, 0x2b})
	if err != nil {
		t.Fatal(err)
	}
	iar.Value.Wire = runtime.WireInfo{}
	got, err := EncodeIARRestOctets(iar.Value)
	if err != nil || !bytes.Equal(got, []byte{0x2b, 0x2b, 0x2b}) {
		t.Fatalf("fresh IAR %x, %v", got, err)
	}
	si2, err := DecodeSI2bisRestOctets([]byte{0x2b})
	if err != nil {
		t.Fatal(err)
	}
	si2.Value.Wire = runtime.WireInfo{}
	got, err = EncodeSI2bisRestOctets(si2.Value)
	if err != nil || !bytes.Equal(got, []byte{0x2b}) {
		t.Fatalf("fresh SI2bis %x, %v", got, err)
	}
}

func TestCanonicalMinimumPaddingAndTruncation(t *testing.T) {
	// TS 44.018 V19.0.0 §§8.9, 10.5.2.17, 10.5.2.23–25,
	// 10.5.2.33: fixed IE lengths retain L/H spare padding, while
	// paging rest octets permit a truncated concatenation.
	for _, tc := range []struct {
		name   string
		wire   []byte
		encode func([]byte) ([]byte, error)
	}{
		{"IAR", []byte{0x2b, 0x2b, 0x2b}, func(b []byte) ([]byte, error) {
			d, e := DecodeIARRestOctets(b)
			if e != nil {
				return nil, e
			}
			return EncodeIARRestOctetsCanonical(d.Value)
		}},
		{"SI2bis", []byte{0x2b}, func(b []byte) ([]byte, error) {
			d, e := DecodeSI2bisRestOctets(b)
			if e != nil {
				return nil, e
			}
			return EncodeSI2bisRestOctetsCanonical(d.Value)
		}},
		{"P1", []byte{0x2b}, func(b []byte) ([]byte, error) {
			d, e := DecodeP1RestOctets(b)
			if e != nil {
				return nil, e
			}
			return EncodeP1RestOctetsCanonical(d.Value)
		}},
		{"P2", []byte{0x2b}, func(b []byte) ([]byte, error) {
			d, e := DecodeP2RestOctets(b)
			if e != nil {
				return nil, e
			}
			return EncodeP2RestOctetsCanonical(d.Value)
		}},
		{"P3", []byte{0x2b, 0x2b, 0x2b}, func(b []byte) ([]byte, error) {
			d, e := DecodeP3RestOctets(b)
			if e != nil {
				return nil, e
			}
			return EncodeP3RestOctetsCanonical(d.Value)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.encode(tc.wire)
			if err != nil || !bytes.Equal(got, tc.wire) {
				t.Fatalf("canonical %x, %v; want %x", got, err, tc.wire)
			}
		})
	}
}
