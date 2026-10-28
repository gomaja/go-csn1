package registry_test

import (
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
)

func TestClassmark2DescriptorExtendsOnlyTail(t *testing.T) {
	data := []byte{0x20, 0, 0, 0xa5}
	typed, err := uecapability.DecodeClassmark2ValuePartFrom(runtime.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	d, err := registry.Lookup("TS 24.008", "Mobile Station Classmark 2 value part")
	if err != nil {
		t.Fatal(err)
	}
	got, err := d.DecodeFrom(readerBoundTo(t, data))
	if err != nil {
		t.Fatal(err)
	}
	typed.Wire.Tail = runtime.BitString{Bytes: []byte{0xa5}, BitLength: 8}
	typed.Wire.TransmittedBits = 32
	want := runtime.Decoded[uecapability.Classmark2ValuePart]{Value: typed, BitsConsumed: 24, Tail: typed.Wire.Tail}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("descriptor replaced typed wire: got %#v, want %#v", got, want)
	}
}
