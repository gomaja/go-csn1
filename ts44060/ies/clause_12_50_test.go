package ies

import (
	"reflect"
	"testing"
)

// TestClause12_50 checks TS 44.060 V19.0.0 §12.50,
// authenticated DOCX paragraphs 25241-25268.
func TestClause12_50(t *testing.T) {
	checkCellPriorityVectors(t, "12.50", "Individual priorities IE", reflect.TypeFor[IndividualPrioritiesIE](), `[
  {
    "name": "delete",
    "hex": "00",
    "bits": 1,
    "fields": {},
    "widths": {}
  },
  {
    "name": "provide-False-mode-0-default-False",
    "hex": "f0",
    "bits": 7,
    "fields": {
      "geran_priority": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ]
    }
  },
  {
    "name": "provide-False-mode-0-default-True",
    "hex": "f3c0",
    "bits": 10,
    "fields": {
      "geran_priority": [
        7
      ],
      "t3230_timeout_value": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "t3230_timeout_value": [
        3
      ]
    }
  },
  {
    "name": "provide-False-mode-1-default-False",
    "hex": "f0",
    "bits": 7,
    "fields": {
      "geran_priority": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ]
    }
  },
  {
    "name": "provide-False-mode-1-default-True",
    "hex": "f3c0",
    "bits": 10,
    "fields": {
      "geran_priority": [
        7
      ],
      "t3230_timeout_value": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "t3230_timeout_value": [
        3
      ]
    }
  },
  {
    "name": "provide-True-mode-0-default-False",
    "hex": "faffff000557ffff000130",
    "bits": 86,
    "fields": {
      "geran_priority": [
        7
      ],
      "fdd_arfcn": [
        16383,
        1
      ],
      "utran_priority": [
        5
      ],
      "earfcn": [
        65535,
        1
      ],
      "e_utran_priority": [
        3
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "fdd_arfcn": [
        14,
        14
      ],
      "utran_priority": [
        3
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ]
    }
  },
  {
    "name": "provide-True-mode-0-default-True",
    "hex": "ffdfffe000abdffffc0004de",
    "bits": 95,
    "fields": {
      "geran_priority": [
        7
      ],
      "default_utran_priority": [
        7
      ],
      "fdd_arfcn": [
        16383,
        1
      ],
      "utran_priority": [
        5
      ],
      "default_e_utran_priority": [
        6
      ],
      "earfcn": [
        65535,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "t3230_timeout_value": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "default_utran_priority": [
        3
      ],
      "fdd_arfcn": [
        14,
        14
      ],
      "utran_priority": [
        3
      ],
      "default_e_utran_priority": [
        3
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ],
      "t3230_timeout_value": [
        3
      ]
    }
  },
  {
    "name": "provide-True-mode-1-default-False",
    "hex": "fbffff000557ffff000130",
    "bits": 86,
    "fields": {
      "geran_priority": [
        7
      ],
      "tdd_arfcn": [
        16383,
        1
      ],
      "utran_priority": [
        5
      ],
      "earfcn": [
        65535,
        1
      ],
      "e_utran_priority": [
        3
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "tdd_arfcn": [
        14,
        14
      ],
      "utran_priority": [
        3
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ]
    }
  },
  {
    "name": "provide-True-mode-1-default-True",
    "hex": "ffffffe000abdffffc0004de",
    "bits": 95,
    "fields": {
      "geran_priority": [
        7
      ],
      "default_utran_priority": [
        7
      ],
      "tdd_arfcn": [
        16383,
        1
      ],
      "utran_priority": [
        5
      ],
      "default_e_utran_priority": [
        6
      ],
      "earfcn": [
        65535,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "t3230_timeout_value": [
        7
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "default_utran_priority": [
        3
      ],
      "tdd_arfcn": [
        14,
        14
      ],
      "utran_priority": [
        3
      ],
      "default_e_utran_priority": [
        3
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ],
      "t3230_timeout_value": [
        3
      ]
    }
  }
]`)
}

// TestIndividualPriorityOptionalIndependence checks the independent presence
// selectors in TS 44.060 V19.0.0 §12.50 table 12.50.1, paragraphs 25245-25249.
func TestIndividualPriorityOptionalIndependence(t *testing.T) {
	checkCellPriorityVectors(t, "12.50", "Individual priorities IE", reflect.TypeFor[IndividualPrioritiesIE](), `[
  {
    "name": "utran-only",
    "hex": "dfd0005000",
    "bits": 33,
    "fields": {
      "geran_priority": [
        5
      ],
      "default_utran_priority": [
        7
      ],
      "fdd_arfcn": [
        1
      ],
      "utran_priority": [
        4
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "default_utran_priority": [
        3
      ],
      "fdd_arfcn": [
        14
      ],
      "utran_priority": [
        3
      ]
    }
  },
  {
    "name": "eutran-only",
    "hex": "a7bffff258",
    "bits": 37,
    "fields": {
      "geran_priority": [
        2
      ],
      "default_e_utran_priority": [
        6
      ],
      "earfcn": [
        65535
      ],
      "e_utran_priority": [
        2
      ],
      "t3230_timeout_value": [
        3
      ]
    },
    "widths": {
      "geran_priority": [
        3
      ],
      "default_e_utran_priority": [
        3
      ],
      "earfcn": [
        16
      ],
      "e_utran_priority": [
        3
      ],
      "t3230_timeout_value": [
        3
      ]
    }
  }
]`)
}
