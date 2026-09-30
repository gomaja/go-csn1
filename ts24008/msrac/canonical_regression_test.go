package msrac

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestCanonicalAccessCapabilitiesReceiverZeros(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a: absent trailing capability
	// bits are inferred as zero within the length-delimited value.
	wire := []byte{0}
	decoded, err := DecodeAccessCapabilitiesStruct(wire)
	if err != nil {
		t.Fatal(err)
	}
	got, err := EncodeAccessCapabilitiesStructCanonical(decoded.Value)
	if err != nil || !bytes.Equal(got, wire) {
		t.Fatalf("canonical %x, %v; want %x", got, err, wire)
	}
	again, err := DecodeAccessCapabilitiesStruct(got)
	if err != nil || !reflect.DeepEqual(runtime.Canonical(decoded.Value), runtime.Canonical(again.Value)) {
		t.Fatalf("semantic round trip: %v", err)
	}
}
