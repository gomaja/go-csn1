package ies

import (
	"bytes"
	"reflect"
	"testing"
)

func FuzzIEDefinitions(f *testing.F) {
	for _, seed := range [][]byte{{}, {0x00}, {0x01}, {0x7f}, {0xff}, {0x2b}, {0x2b, 0x2b}, {0x00, 0x00}, {0x55, 0x55}, {0, 0, 0, 0, 0}} {
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
