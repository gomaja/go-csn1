package msrac

import (
	"bytes"
	"reflect"
	"testing"
)

func TestA52IsReceivedLayout(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a: send A5/2 as zero; accept any value.
	for _, wire := range [][]byte{{0}, {0x40}, {0xfe}} {
		d, err := DecodeA5Bits(wire)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.ValueOf(d.Value).FieldByName("A52").IsValid() {
			t.Error("A52 remains a typed field")
		}
		plain, err := EncodeA5Bits(d.Value)
		if err != nil || !bytes.Equal(plain, wire) {
			t.Fatalf("plain %x: %v", plain, err)
		}
		canonical, err := EncodeA5BitsCanonicalAtLength(d.Value, 1)
		want := []byte{wire[0] & 0xbe}
		if err != nil || !bytes.Equal(canonical, want) {
			t.Fatalf("canonical %x, want %x: %v", canonical, want, err)
		}
	}
}
