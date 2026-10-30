package ies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"strings"
	"testing"
)

// TestClause12_31 checks TS 44.060 V19.0.0 §12.31,
// authenticated DOCX paragraphs 23552-23556.
func TestClause12_31(t *testing.T) {
	checkCellPriorityVectors(t, "12.31", "UTRAN FDD Target cell IE", reflect.TypeFor[UTRANFDDTargetCellIE](), `[
  {
    "name": "0-bandwidth-False",
    "hex": "0000aa80",
    "bits": 25,
    "fields": {
      "fdd_arfcn": [
        0
      ],
      "diversity": [
        0
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "scrambling_code": [
        9
      ]
    }
  },
  {
    "name": "0-bandwidth-True",
    "hex": "0001b550",
    "bits": 28,
    "fields": {
      "fdd_arfcn": [
        0
      ],
      "diversity": [
        0
      ],
      "bandwidth_fdd": [
        5
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "bandwidth_fdd": [
        3
      ],
      "scrambling_code": [
        9
      ]
    }
  },
  {
    "name": "1-bandwidth-False",
    "hex": "0006aa80",
    "bits": 25,
    "fields": {
      "fdd_arfcn": [
        1
      ],
      "diversity": [
        1
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "scrambling_code": [
        9
      ]
    }
  },
  {
    "name": "1-bandwidth-True",
    "hex": "0007b550",
    "bits": 28,
    "fields": {
      "fdd_arfcn": [
        1
      ],
      "diversity": [
        1
      ],
      "bandwidth_fdd": [
        5
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "bandwidth_fdd": [
        3
      ],
      "scrambling_code": [
        9
      ]
    }
  },
  {
    "name": "16383-bandwidth-False",
    "hex": "fffeaa80",
    "bits": 25,
    "fields": {
      "fdd_arfcn": [
        16383
      ],
      "diversity": [
        1
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "scrambling_code": [
        9
      ]
    }
  },
  {
    "name": "16383-bandwidth-True",
    "hex": "ffffb550",
    "bits": 28,
    "fields": {
      "fdd_arfcn": [
        16383
      ],
      "diversity": [
        1
      ],
      "bandwidth_fdd": [
        5
      ],
      "scrambling_code": [
        341
      ]
    },
    "widths": {
      "fdd_arfcn": [
        14
      ],
      "diversity": [
        1
      ],
      "bandwidth_fdd": [
        3
      ],
      "scrambling_code": [
        9
      ]
    }
  }
]`)
}

// The synthetic vectors follow TS 44.060 V19.0.0 clauses cited by each test.
// Numeric expectations and exact canonical bytes also agree with an independent
// implementation of the printed grammar.
type cellPriorityVector struct {
	Name   string              `json:"name"`
	Hex    string              `json:"hex"`
	Bits   int                 `json:"bits"`
	Fields map[string][]uint64 `json:"fields"`
	Widths map[string][]int    `json:"widths"`
}

func cellPriorityName(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, name)
}

func cellPriorityWalk(value reflect.Value, name string, visit func(string, reflect.Value)) {
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if !field.IsExported() || field.Name == "Wire" {
				continue
			}
			cellPriorityWalk(value.Field(i), field.Name, visit)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			cellPriorityWalk(value.Index(i), name, visit)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		visit(name, value)
	}
}

func checkCellPriorityVectors(t *testing.T, clause, definition string, typ reflect.Type, text string) {
	t.Helper()
	descriptor, err := LookupClause(clause, definition)
	if err != nil {
		t.Fatal(err)
	}
	var vectors []cellPriorityVector
	if err := json.Unmarshal([]byte(text), &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) == 0 {
		t.Fatal("empty vector set")
	}
	for _, vector := range vectors {
		t.Run(vector.Name, func(t *testing.T) {
			wire, err := hex.DecodeString(vector.Hex)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := descriptor.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			result := reflect.ValueOf(decoded)
			value := result.FieldByName("Value")
			if !value.IsValid() || value.Type() != typ {
				t.Fatalf("lost typed value: %T, want %s", decoded, typ)
			}
			if result.FieldByName("BitsConsumed").Int() != int64(vector.Bits) {
				t.Fatalf("consumed %d, want %d", result.FieldByName("BitsConsumed").Int(), vector.Bits)
			}
			expected := map[string][]uint64{}
			widths := map[string][]int{}
			for name, values := range vector.Fields {
				expected[cellPriorityName(name)] = values
			}
			for name, values := range vector.Widths {
				widths[cellPriorityName(name)] = values
			}
			actual := map[string][]uint64{}
			cellPriorityWalk(value, "", func(name string, field reflect.Value) {
				if name == "Alternative" {
					return
				}
				key := cellPriorityName(name)
				actual[key] = append(actual[key], field.Uint())
			})
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("typed fields %v, want %v", actual, expected)
			}
			replay, err := descriptor.Encode(decoded)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			canonical, err := descriptor.CanonicalAtLength(value.Interface(), len(wire))
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("canonical %x, want %x: %v", canonical, wire, err)
			}
			fresh, err := descriptor.Decode(canonical)
			if err != nil {
				t.Fatal(err)
			}
			stable, err := descriptor.CanonicalAtLength(fresh, len(wire))
			if err != nil || !bytes.Equal(stable, canonical) {
				t.Fatalf("unstable canonical %x: %v", stable, err)
			}
			// TS 24.007 V20.0.0 Annex B: each field and selector must fit
			// in the enclosing string. A following valid suffix cannot rescue it.
			for limit := 0; limit < vector.Bits; limit++ {
				reader := runtime.NewReader(append(bytes.Clone(wire), 0xff))
				if _, err := reader.PushLimit(limit); err != nil {
					t.Fatal(err)
				}
				if _, err := descriptor.DecodeFrom(reader); err == nil {
					t.Fatalf("accepted %d-bit prefix of %d-bit value", limit, vector.Bits)
				}
				if reader.Position() > limit {
					t.Fatalf("read beyond %d-bit bound: %d", limit, reader.Position())
				}
			}
			reader := runtime.NewReader(append(bytes.Clone(wire), 0xa5))
			old, err := reader.PushLimit(vector.Bits)
			if err != nil {
				t.Fatal(err)
			}
			bounded, err := descriptor.DecodeFrom(reader)
			if err != nil {
				t.Fatal(err)
			}
			boundedValue := reflect.ValueOf(bounded).FieldByName("Value")
			boundedFields := map[string][]uint64{}
			cellPriorityWalk(boundedValue, "", func(name string, field reflect.Value) {
				if name != "Alternative" {
					key := cellPriorityName(name)
					boundedFields[key] = append(boundedFields[key], field.Uint())
				}
			})
			if !reflect.DeepEqual(boundedFields, expected) {
				t.Fatalf("bounded fields %v, want %v", boundedFields, expected)
			}
			if err := reader.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			tail, err := reader.ReadUint(len(wire)*8 - vector.Bits)
			if err != nil || tail != 0 {
				t.Fatalf("tail %d: %v", tail, err)
			}
			sentinel, err := reader.ReadUint(8)
			if err != nil || sentinel != 0xa5 {
				t.Fatalf("sentinel %#x: %v", sentinel, err)
			}
			// Check the first unrepresentable value of each field whose Go storage
			// is wider than its printed width, and every typed union selector.
			for target := 0; ; target++ {
				fresh, err := descriptor.Decode(wire)
				if err != nil {
					t.Fatal(err)
				}
				original := reflect.ValueOf(fresh).FieldByName("Value")
				changed := reflect.New(original.Type()).Elem()
				changed.Set(original)
				index := 0
				found := false
				seen := map[string]int{}
				cellPriorityWalk(changed, "", func(name string, field reflect.Value) {
					key := cellPriorityName(name)
					occurrence := seen[key]
					seen[key]++
					width := 0
					if name == "Alternative" {
						width = 7
					} else {
						if occurrence >= len(widths[key]) {
							t.Fatalf("missing width for %s[%d]", name, occurrence)
						}
						width = widths[key][occurrence]
					}
					if width >= field.Type().Bits() {
						return
					}
					if index == target {
						field.SetUint(uint64(1) << uint(width))
						found = true
					}
					index++
				})
				if !found {
					break
				}
				if _, err := descriptor.Encode(changed.Interface()); err == nil {
					t.Fatalf("accepted out-of-range field or selector %d", target)
				}
			}
		})
	}
}

func FuzzCellPriorityReplay(f *testing.F) {
	names := []struct{ clause, name string }{
		{"12.31", "UTRAN FDD Target cell IE"},
		{"12.32", "UTRAN TDD Target cell IE"},
		{"12.49", "E-UTRAN Target cell IE"},
		{"12.50", "Individual priorities IE"},
		{"12.68", "E-UTRAN Target cell with extended EARFCN IE"},
		{"12.69", "E-UTRAN IPP with extended EARFCNs IE"},
		{"12.70", "E-UTRAN NC with extended EARFCNs IE"},
	}
	descriptors := make([]runtime.Descriptor, len(names))
	for i, name := range names {
		d, err := LookupClause(name.clause, name.name)
		if err != nil {
			f.Fatal(err)
		}
		descriptors[i] = d
	}
	f.Add(uint8(0), []byte{0x0, 0x0, 0xaa, 0x80})
	f.Add(uint8(0), []byte{0x0, 0x1, 0xb5, 0x50})
	f.Add(uint8(0), []byte{0x0, 0x6, 0xaa, 0x80})
	f.Add(uint8(0), []byte{0x0, 0x7, 0xb5, 0x50})
	f.Add(uint8(0), []byte{0xff, 0xfe, 0xaa, 0x80})
	f.Add(uint8(0), []byte{0xff, 0xff, 0xb5, 0x50})
	f.Add(uint8(1), []byte{0x0, 0x0, 0xa6})
	f.Add(uint8(1), []byte{0x0, 0x1, 0xb4, 0xc0})
	f.Add(uint8(1), []byte{0x0, 0x6, 0xa7})
	f.Add(uint8(1), []byte{0x0, 0x7, 0xb4, 0xe0})
	f.Add(uint8(1), []byte{0xff, 0xfe, 0xa7})
	f.Add(uint8(1), []byte{0xff, 0xff, 0xb4, 0xe0})
	f.Add(uint8(2), []byte{0x0, 0x0, 0x55, 0x40})
	f.Add(uint8(2), []byte{0x0, 0x0, 0xda, 0xa8})
	f.Add(uint8(2), []byte{0x0, 0x1, 0x55, 0x40})
	f.Add(uint8(2), []byte{0x0, 0x1, 0xda, 0xa8})
	f.Add(uint8(2), []byte{0xff, 0xff, 0x55, 0x40})
	f.Add(uint8(2), []byte{0xff, 0xff, 0xda, 0xa8})
	f.Add(uint8(4), []byte{0x0, 0x0, 0x15, 0x50})
	f.Add(uint8(4), []byte{0x0, 0x0, 0x36, 0xaa})
	f.Add(uint8(4), []byte{0x0, 0x0, 0x55, 0x50})
	f.Add(uint8(4), []byte{0x0, 0x0, 0x76, 0xaa})
	f.Add(uint8(4), []byte{0xff, 0xff, 0xd5, 0x50})
	f.Add(uint8(4), []byte{0xff, 0xff, 0xf6, 0xaa})
	f.Add(uint8(3), []byte{0x0})
	f.Add(uint8(3), []byte{0xf0})
	f.Add(uint8(3), []byte{0xf3, 0xc0})
	f.Add(uint8(3), []byte{0xf0})
	f.Add(uint8(3), []byte{0xf3, 0xc0})
	f.Add(uint8(3), []byte{0xfa, 0xff, 0xff, 0x0, 0x5, 0x57, 0xff, 0xff, 0x0, 0x1, 0x30})
	f.Add(uint8(3), []byte{0xff, 0xdf, 0xff, 0xe0, 0x0, 0xab, 0xdf, 0xff, 0xfc, 0x0, 0x4, 0xde})
	f.Add(uint8(3), []byte{0xfb, 0xff, 0xff, 0x0, 0x5, 0x57, 0xff, 0xff, 0x0, 0x1, 0x30})
	f.Add(uint8(3), []byte{0xff, 0xff, 0xff, 0xe0, 0x0, 0xab, 0xdf, 0xff, 0xfc, 0x0, 0x4, 0xde})
	f.Add(uint8(5), []byte{0x0})
	f.Add(uint8(5), []byte{0x7f, 0xff, 0xfd, 0x0, 0x0, 0x70})
	f.Add(uint8(5), []byte{0x7f, 0xff, 0xfd, 0x0, 0x0, 0x7f, 0xff, 0xff, 0xa0, 0x0, 0xe})
	f.Add(uint8(5), []byte{0xf0})
	f.Add(uint8(5), []byte{0xff, 0xff, 0xff, 0xa0, 0x0, 0xe})
	f.Add(uint8(5), []byte{0xff, 0xff, 0xff, 0xa0, 0x0, 0xf, 0xff, 0xff, 0xf4, 0x0, 0x1, 0xc0})
	f.Add(uint8(6), []byte{0x0})
	f.Add(uint8(6), []byte{0xff, 0xff, 0xf5, 0x0, 0x0, 0x83, 0xe0})
	f.Add(uint8(6), []byte{0xff, 0xff, 0xf5, 0x0, 0x0, 0x83, 0xe7, 0xff, 0xff, 0xa8, 0x0, 0x4, 0x1f, 0x0})
	f.Add(uint8(6), []byte{0x0})
	f.Add(uint8(6), []byte{0xff, 0xff, 0xfd, 0xa0, 0x0, 0x7, 0xff, 0xff, 0xc0})
	f.Add(uint8(6), []byte{0xff, 0xff, 0xfd, 0xa0, 0x0, 0x7, 0xff, 0xff, 0xff, 0xff, 0xff, 0x68, 0x0, 0x1, 0xff, 0xff, 0xf0})
	f.Add(uint8(3), []byte{0xdf, 0xd0, 0x0, 0x50, 0x0})
	f.Add(uint8(3), []byte{0xa7, 0xbf, 0xff, 0xf2, 0x58})
	f.Fuzz(func(t *testing.T, which uint8, data []byte) {
		if len(data) > 256 {
			t.Skip()
		}
		d := descriptors[int(which)%len(descriptors)]
		v, err := d.Decode(data)
		if err != nil {
			return
		}
		replay, err := d.Encode(v)
		if err != nil || !bytes.Equal(replay, data) {
			t.Fatalf("%s replay %x, want %x: %v", d.Name, replay, data, err)
		}
	})
}
