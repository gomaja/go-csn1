package ies

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func FuzzIEDefinitions(f *testing.F) {
	for _, seed := range [][]byte{{}, {0x00}, {0x01}, {0x7f}, {0xff}, {0x2b}, {0x2b, 0x2b}, {0x00, 0x00}, {0x55, 0x55}, {0, 0, 0, 0, 0}} {
		f.Add(seed)
	}
	// TS 44.060 §§11.1.4.4, 12.24: complete R99, cut inside Rel-4.
	f.Add(gprsCellOptionsWithIgnored(runtime.BitString{Bytes: []byte{0xd7, 0xc0}, BitLength: 10}))
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, descriptor := range Descriptors() {
			definition, err := LookupClause(descriptor.Clause, descriptor.Name)
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
				t.Fatalf("%s/%s round trip %x -> %x: %v", descriptor.Clause, descriptor.Name, data, encoded, err)
			}
		}
	})
}

func TestDependencyHelperClauses(t *testing.T) {
	for name, clause := range map[string]string{
		"Indirect encoding struct":                                     "12.8",
		"Direct encoding 1 struct":                                     "12.8",
		"Direct encoding 2 struct":                                     "12.8",
		"RFL number list struct":                                       "12.10a",
		"ARFCN index list struct":                                      "12.10a",
		"Repeated E-UTRAN Enhanced Cell Reselection Parameters struct": "12.59",
	} {
		d, err := LookupClause(clause, name)
		if err != nil || d.Clause != clause {
			t.Fatalf("%s clause: %+v, %v", name, d, err)
		}
	}
}
