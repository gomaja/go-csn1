package ies

import (
	"reflect"
	"testing"
)

// TestClause12_68 checks TS 44.060 V19.0.0 §12.68,
// authenticated DOCX paragraphs 25896-25899.
func TestClause12_68(t *testing.T) {
	checkCellPriorityVectors(t, "12.68", "E-UTRAN Target cell with extended EARFCN IE", reflect.TypeFor[EUTRANTargetCellWithExtendedEARFCNIE](), `[
  {
    "name": "0-bandwidth-False",
    "hex": "00001550",
    "bits": 28,
    "fields": {
      "earfcn_extended": [
        0
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "0-bandwidth-True",
    "hex": "000036aa",
    "bits": 31,
    "fields": {
      "earfcn_extended": [
        0
      ],
      "measurement_bandwidth": [
        5
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "measurement_bandwidth": [
        3
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "1-bandwidth-False",
    "hex": "00005550",
    "bits": 28,
    "fields": {
      "earfcn_extended": [
        1
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "1-bandwidth-True",
    "hex": "000076aa",
    "bits": 31,
    "fields": {
      "earfcn_extended": [
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "measurement_bandwidth": [
        3
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "262143-bandwidth-False",
    "hex": "ffffd550",
    "bits": 28,
    "fields": {
      "earfcn_extended": [
        262143
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "262143-bandwidth-True",
    "hex": "fffff6aa",
    "bits": 31,
    "fields": {
      "earfcn_extended": [
        262143
      ],
      "measurement_bandwidth": [
        5
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn_extended": [
        18
      ],
      "measurement_bandwidth": [
        3
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  }
]`)
}
