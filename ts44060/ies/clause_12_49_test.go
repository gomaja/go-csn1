package ies

import (
	"reflect"
	"testing"
)

// TestClause12_49 checks TS 44.060 V19.0.0 §12.49,
// authenticated DOCX paragraphs 25228-25231.
func TestClause12_49(t *testing.T) {
	checkCellPriorityVectors(t, "12.49", "E-UTRAN Target cell IE", reflect.TypeFor[EUTRANTargetCellIE](), `[
  {
    "name": "0-bandwidth-False",
    "hex": "00005540",
    "bits": 26,
    "fields": {
      "earfcn": [
        0
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn": [
        16
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "0-bandwidth-True",
    "hex": "0000daa8",
    "bits": 29,
    "fields": {
      "earfcn": [
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
      "earfcn": [
        16
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
    "hex": "00015540",
    "bits": 26,
    "fields": {
      "earfcn": [
        1
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn": [
        16
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "1-bandwidth-True",
    "hex": "0001daa8",
    "bits": 29,
    "fields": {
      "earfcn": [
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
      "earfcn": [
        16
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
    "name": "65535-bandwidth-False",
    "hex": "ffff5540",
    "bits": 26,
    "fields": {
      "earfcn": [
        65535
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn": [
        16
      ],
      "physical_layer_cell_identity": [
        9
      ]
    }
  },
  {
    "name": "65535-bandwidth-True",
    "hex": "ffffdaa8",
    "bits": 29,
    "fields": {
      "earfcn": [
        65535
      ],
      "measurement_bandwidth": [
        5
      ],
      "physical_layer_cell_identity": [
        341
      ]
    },
    "widths": {
      "earfcn": [
        16
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
