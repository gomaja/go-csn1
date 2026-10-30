package ies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type sourceVector struct {
	InvalidValues                 map[string]uint64 `json:"invalid_values"`
	Name, Clause, Definition, Hex string
	Values                        map[string]uint64
	Present, Absent               []string
	CanonicalHex                  string `json:"canonical_hex"`
}

func sourceField(t *testing.T, value any, path string) reflect.Value {
	t.Helper()
	field := reflect.ValueOf(value)
	if field.Kind() == reflect.Struct {
		if decoded := field.FieldByName("Value"); decoded.IsValid() {
			field = decoded
		}
	}
	for _, name := range strings.Split(path, ".") {
		for field.Kind() == reflect.Pointer || field.Kind() == reflect.Interface {
			if field.IsNil() {
				t.Fatalf("nil field in %s", path)
			}
			field = field.Elem()
		}
		field = field.FieldByName(name)
		if !field.IsValid() {
			t.Fatalf("missing field %s", path)
		}
	}
	return field
}

func sourceNumber(t *testing.T, value any, path string) uint64 {
	t.Helper()
	field := sourceField(t, value, path)
	for field.Kind() == reflect.Pointer {
		if field.IsNil() {
			t.Fatalf("nil numeric field %s", path)
		}
		field = field.Elem()
	}
	return field.Uint()
}

func TestPublishedSourceVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/source-roots.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []sourceVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("empty vector set")
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			descriptor, err := LookupClause(vector.Clause, vector.Definition)
			if err != nil {
				t.Fatal(err)
			}
			wire, err := hex.DecodeString(vector.Hex)
			if err != nil {
				t.Fatal(err)
			}
			value, err := descriptor.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := descriptor.Encode(value)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			for path, want := range vector.Values {
				if got := sourceNumber(t, value, path); got != want {
					t.Errorf("%s=%d want %d", path, got, want)
				}
			}
			for _, path := range vector.Present {
				if sourceField(t, value, path).IsNil() {
					t.Errorf("%s absent", path)
				}
			}
			for _, path := range vector.Absent {
				if !sourceField(t, value, path).IsNil() {
					t.Errorf("%s present", path)
				}
			}
			canonical, err := descriptor.CanonicalAtLength(value, len(wire))
			if err != nil {
				t.Fatalf("canonical: %v", err)
			}
			if vector.CanonicalHex != "" && hex.EncodeToString(canonical) != vector.CanonicalHex {
				t.Errorf("canonical %x want %s", canonical, vector.CanonicalHex)
			}
			fresh, err := descriptor.Decode(canonical)
			if err != nil {
				t.Fatalf("canonical output rejected: %v", err)
			}
			stable, err := descriptor.CanonicalAtLength(fresh, len(wire))
			if err != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, err)
			}
			for path, want := range vector.Values {
				if got := sourceNumber(t, fresh, path); got != want {
					t.Errorf("canonical %s=%d want %d", path, got, want)
				}
			}

			for path, invalid := range vector.InvalidValues {
				copyValue := reflect.ValueOf(runtime.Canonical(value))
				mutable := reflect.New(copyValue.Type()).Elem()
				mutable.Set(copyValue)
				// The decoded envelope is copied into settable storage before the edit.
				field := mutable.FieldByName("Value")
				for _, name := range strings.Split(path, ".") {
					for field.Kind() == reflect.Pointer {
						field = field.Elem()
					}
					field = field.FieldByName(name)
				}
				for field.Kind() == reflect.Pointer {
					field = field.Elem()
				}
				field.SetUint(invalid)
				if _, err := descriptor.CanonicalAtLength(mutable.Interface(), len(wire)); err == nil {
					t.Errorf("out-of-range %s=%d encoded", path, invalid)
				}
			}
			if _, err := descriptor.Decode(wire[:len(wire)-1]); err == nil {
				t.Fatal("truncated value accepted")
			}
		})
	}
}

func FuzzPublishedSourceRoundTrip(f *testing.F) {
	data, err := os.ReadFile("testdata/source-roots.json")
	if err != nil {
		f.Fatal(err)
	}
	var vectors []sourceVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		f.Fatal(err)
	}
	for i, vector := range vectors {
		wire, err := hex.DecodeString(vector.Hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(uint8(i), wire)
	}
	f.Fuzz(func(t *testing.T, index uint8, wire []byte) {
		if len(wire) > 64 {
			return
		}
		vector := vectors[int(index)%len(vectors)]
		descriptor, err := LookupClause(vector.Clause, vector.Definition)
		if err != nil {
			t.Fatal(err)
		}
		value, err := descriptor.Decode(wire)
		if err != nil {
			return
		}
		replay, err := descriptor.Encode(value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("accepted input did not replay: %x %v", replay, err)
		}
	})
}
