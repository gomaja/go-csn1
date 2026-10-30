package ies

import (
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"testing"
)

// TestClause12_69 checks TS 44.060 V19.0.0 §12.69,
// authenticated DOCX paragraphs 25910-25917.
func TestClause12_69(t *testing.T) {
	checkCellPriorityVectors(t, "12.69", "E-UTRAN IPP with extended EARFCNs IE", reflect.TypeFor[EUTRANIPPWithExtendedEARFCNsIE](), `[
  {
    "name": "default-False-groups-0",
    "hex": "00",
    "bits": 2,
    "fields": {},
    "widths": {}
  },
  {
    "name": "default-False-groups-1",
    "hex": "7ffffd000070",
    "bits": 45,
    "fields": {
      "earfcn_extended": [
        262143,
        65536
      ],
      "e_utran_priority": [
        7
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18
      ],
      "e_utran_priority": [
        3
      ]
    }
  },
  {
    "name": "default-False-groups-2",
    "hex": "7ffffd00007fffffa0000e",
    "bits": 88,
    "fields": {
      "earfcn_extended": [
        262143,
        65536,
        262143,
        65536
      ],
      "e_utran_priority": [
        7,
        7
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18,
        18,
        18
      ],
      "e_utran_priority": [
        3,
        3
      ]
    }
  },
  {
    "name": "default-True-groups-0",
    "hex": "f0",
    "bits": 5,
    "fields": {
      "default_e_utran_priority": [
        7
      ]
    },
    "widths": {
      "default_e_utran_priority": [
        3
      ]
    }
  },
  {
    "name": "default-True-groups-1",
    "hex": "ffffffa0000e",
    "bits": 48,
    "fields": {
      "default_e_utran_priority": [
        7
      ],
      "earfcn_extended": [
        262143,
        65536
      ],
      "e_utran_priority": [
        7
      ]
    },
    "widths": {
      "default_e_utran_priority": [
        3
      ],
      "earfcn_extended": [
        18,
        18
      ],
      "e_utran_priority": [
        3
      ]
    }
  },
  {
    "name": "default-True-groups-2",
    "hex": "ffffffa0000ffffff40001c0",
    "bits": 91,
    "fields": {
      "default_e_utran_priority": [
        7
      ],
      "earfcn_extended": [
        262143,
        65536,
        262143,
        65536
      ],
      "e_utran_priority": [
        7,
        7
      ]
    },
    "widths": {
      "default_e_utran_priority": [
        3
      ],
      "earfcn_extended": [
        18,
        18,
        18,
        18
      ],
      "e_utran_priority": [
        3,
        3
      ]
    }
  }
]`)
}

// TestExtendedPriorityListGrowth checks the repeated 18-bit EARFCN structure
// printed in TS 44.060 V19.0.0 §12.69, paragraphs 25910-25917.
func TestExtendedPriorityListGrowth(t *testing.T) {
	for _, count := range []int{0, 1, 16, 64, 256, 1024} {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, 1); err != nil {
			t.Fatal(err)
		} // default priority absent
		for i := 0; i < count; i++ {
			if err := w.WriteLiteral("11"); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(uint64(i), 18); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteLiteral("0"); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(7, 3); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.WriteLiteral("0"); err != nil {
			t.Fatal(err)
		}
		wire, err := w.Finish(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		var decodeErr error
		allocations := testing.AllocsPerRun(3, func() {
			value, err := DecodeEUTRANIPPWithExtendedEARFCNsIE(wire)
			decodeErr = err
			if err == nil && len(value.Value.RepeatedIndividualEUTRANPPWithExtendedEARFCNsList) != count {
				t.Fatalf("decoded %d entries, want %d", len(value.Value.RepeatedIndividualEUTRANPPWithExtendedEARFCNsList), count)
			}
		})
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if allocations > 64+32*float64(count) {
			t.Fatalf("%d entries: %g allocations exceed linear bound", count, allocations)
		}
		t.Logf("%d entries, %d bytes, %g allocations", count, len(wire), allocations)
	}
}
