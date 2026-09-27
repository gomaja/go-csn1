package ies

import (
	"bytes"
	"reflect"
	"testing"
)

func FuzzIEDefinitions(f *testing.F) {
	for _, seed := range [][]byte{{}, {0x00}, {0x01}, {0x7f}, {0xff}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, name := range Definitions() {
			definition, err := Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := definition.Decode(data)
			if err != nil {
				continue
			}
			value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			encoded, err := definition.Encode(value)
			if err != nil || !bytes.Equal(encoded, data) {
				t.Fatalf("%s round trip %x -> %x: %v", name, data, encoded, err)
			}
		}
	})
}
