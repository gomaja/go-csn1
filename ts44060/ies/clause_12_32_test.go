package ies

import (
	"reflect"
	"testing"
)

// TestClause12_32 checks TS 44.060 V19.0.0 §12.32,
// authenticated DOCX paragraphs 23568-23573.
func TestClause12_32(t *testing.T) {
	checkCellPriorityVectors(t, "12.32", "UTRAN TDD Target cell IE", reflect.TypeFor[UTRANTDDTargetCellIE](), `[
  {
    "name": "0-bandwidth-False",
    "hex": "0000a6",
    "bits": 24,
    "fields": {
      "tdd_arfcn": [
        0
      ],
      "diversity_tdd": [
        0
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        0
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  },
  {
    "name": "0-bandwidth-True",
    "hex": "0001b4c0",
    "bits": 27,
    "fields": {
      "tdd_arfcn": [
        0
      ],
      "diversity_tdd": [
        0
      ],
      "bandwidth_tdd": [
        5
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        0
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "bandwidth_tdd": [
        3
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  },
  {
    "name": "1-bandwidth-False",
    "hex": "0006a7",
    "bits": 24,
    "fields": {
      "tdd_arfcn": [
        1
      ],
      "diversity_tdd": [
        1
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        1
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  },
  {
    "name": "1-bandwidth-True",
    "hex": "0007b4e0",
    "bits": 27,
    "fields": {
      "tdd_arfcn": [
        1
      ],
      "diversity_tdd": [
        1
      ],
      "bandwidth_tdd": [
        5
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        1
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "bandwidth_tdd": [
        3
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  },
  {
    "name": "16383-bandwidth-False",
    "hex": "fffea7",
    "bits": 24,
    "fields": {
      "tdd_arfcn": [
        16383
      ],
      "diversity_tdd": [
        1
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        1
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  },
  {
    "name": "16383-bandwidth-True",
    "hex": "ffffb4e0",
    "bits": 27,
    "fields": {
      "tdd_arfcn": [
        16383
      ],
      "diversity_tdd": [
        1
      ],
      "bandwidth_tdd": [
        5
      ],
      "cell_parameter": [
        83
      ],
      "sync_case_tstd": [
        1
      ]
    },
    "widths": {
      "tdd_arfcn": [
        14
      ],
      "diversity_tdd": [
        1
      ],
      "bandwidth_tdd": [
        3
      ],
      "cell_parameter": [
        7
      ],
      "sync_case_tstd": [
        1
      ]
    }
  }
]`)
}
