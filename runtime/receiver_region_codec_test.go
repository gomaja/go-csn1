package runtime_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

// Exercise the published codec, so the runtime receipt is checked through
// plain Encode with the received root layout attached.
func TestPublishedReceiverRegionEdits(t *testing.T) {
	input := []byte{0xff}
	for _, edit := range []struct {
		name  string
		apply func(*runtime.ErrorRegion)
	}{
		{"fabricate", func(v *runtime.ErrorRegion) {
			*v = runtime.ErrorRegion{Label: v.Label, Location: v.Location, Bits: v.Bits}
		}},
		{"bits", func(v *runtime.ErrorRegion) {
			if len(v.Bits.Bytes) == 0 {
				t.Fatal("empty fixture region")
			}
			v.Bits.Bytes[0] ^= 1
		}},
		{"path", func(v *runtime.ErrorRegion) { v.Location.Path += "/edited" }},
	} {
		t.Run(edit.name, func(t *testing.T) {
			d, err := controlmessages.DecodePacketAccessRejectMessageContent(input)
			if err != nil {
				t.Fatal(err)
			}
			out, err := controlmessages.EncodePacketAccessRejectMessageContent(d.Value)
			if err != nil || !bytes.Equal(out, input) {
				t.Fatalf("replay %v", err)
			}
			var find func(reflect.Value) *runtime.ErrorRegion
			find = func(v reflect.Value) *runtime.ErrorRegion {
				if !v.IsValid() {
					return nil
				}
				if v.Type() == reflect.TypeFor[runtime.ErrorRegion]() && v.CanAddr() {
					return v.Addr().Interface().(*runtime.ErrorRegion)
				}
				switch v.Kind() {
				case reflect.Pointer:
					if !v.IsNil() {
						return find(v.Elem())
					}
				case reflect.Struct:
					for i := 0; i < v.NumField(); i++ {
						if v.Type().Field(i).IsExported() {
							if r := find(v.Field(i)); r != nil {
								return r
							}
						}
					}
				}
				return nil
			}
			region := find(reflect.ValueOf(&d.Value).Elem())
			if region == nil {
				t.Fatal("fixture selected no receiver region")
			}
			edit.apply(region)
			if _, err = controlmessages.EncodePacketAccessRejectMessageContent(d.Value); err == nil {
				t.Fatal("edited published region accepted")
			}
		})
	}
}
