package ies

import (
	"reflect"
	"testing"
)

// TestClause12_70 checks TS 44.060 V19.0.0 §12.70,
// authenticated DOCX paragraphs 25934-25943.
func TestClause12_70(t *testing.T) {
	checkCellPriorityVectors(t, "12.70", "E-UTRAN NC with extended EARFCNs IE", reflect.TypeFor[EUTRANNCWithExtendedEARFCNsIE](), `[
  {
    "name": "optional-False-groups-0",
    "hex": "00",
    "bits": 1,
    "fields": {},
    "widths": {}
  },
  {
    "name": "optional-False-groups-1",
    "hex": "fffff5000083e0",
    "bits": 54,
    "fields": {
      "earfcn_extended": [
        262143,
        65536
      ],
      "measurement_bandwidth": [
        0
      ],
      "thresh_e_utran_high": [
        31
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18
      ],
      "measurement_bandwidth": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ]
    }
  },
  {
    "name": "optional-False-groups-2",
    "hex": "fffff5000083e7ffffa800041f00",
    "bits": 107,
    "fields": {
      "earfcn_extended": [
        262143,
        65536,
        262143,
        65536
      ],
      "measurement_bandwidth": [
        0,
        0
      ],
      "thresh_e_utran_high": [
        31,
        31
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18,
        18,
        18
      ],
      "measurement_bandwidth": [
        3,
        3
      ],
      "thresh_e_utran_high": [
        5,
        5
      ]
    }
  },
  {
    "name": "optional-True-groups-0",
    "hex": "00",
    "bits": 1,
    "fields": {},
    "widths": {}
  },
  {
    "name": "optional-True-groups-1",
    "hex": "fffffda00007ffffc0",
    "bits": 67,
    "fields": {
      "earfcn_extended": [
        262143,
        65536
      ],
      "measurement_bandwidth": [
        5
      ],
      "e_utran_priority": [
        7
      ],
      "thresh_e_utran_high": [
        31
      ],
      "thresh_e_utran_low": [
        31
      ],
      "e_utran_qrxlevmin": [
        31
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18
      ],
      "measurement_bandwidth": [
        3
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ],
      "thresh_e_utran_low": [
        5
      ],
      "e_utran_qrxlevmin": [
        5
      ]
    }
  },
  {
    "name": "optional-True-groups-2",
    "hex": "fffffda00007ffffffffff680001fffff0",
    "bits": 133,
    "fields": {
      "earfcn_extended": [
        262143,
        65536,
        262143,
        65536
      ],
      "measurement_bandwidth": [
        5,
        5
      ],
      "e_utran_priority": [
        7,
        7
      ],
      "thresh_e_utran_high": [
        31,
        31
      ],
      "thresh_e_utran_low": [
        31,
        31
      ],
      "e_utran_qrxlevmin": [
        31,
        31
      ]
    },
    "widths": {
      "earfcn_extended": [
        18,
        18,
        18,
        18
      ],
      "measurement_bandwidth": [
        3,
        3
      ],
      "e_utran_priority": [
        3,
        3
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "thresh_e_utran_low": [
        5,
        5
      ],
      "e_utran_qrxlevmin": [
        5,
        5
      ]
    }
  }
]`)
}
