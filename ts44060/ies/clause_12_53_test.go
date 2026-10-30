package ies

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/gomaja/go-csn1/runtime"
	"reflect"
	"testing"
)

// TestClause12_53 checks TS 44.060 V19.0.0 §12.53, authenticated
// DOCX paragraphs 25343-25377, with independently checked synthetic values.
func TestClause12_53(t *testing.T) {
	checkEUTRANParametersVectors(t, "12.53", "E-UTRAN Parameters IE", reflect.TypeFor[EUTRANParametersIE](), `[
  {
    "name": "measure-False-threshold-0",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "measure-False-threshold-1",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "measure-False-threshold-2",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "measure-False-threshold-3",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "measure-True-threshold-0",
    "hex": "fff9e7ffffafffffa3cc3c",
    "bits": 88,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        15
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        3
      ],
      "e_utran_fdd_reporting_threshold": [
        7
      ],
      "e_utran_tdd_reporting_threshold": [
        7
      ],
      "earfcn": [
        65535
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
      ],
      "e_utran_frequency_index": [
        7,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "earfcn": [
        16
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
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "measure-True-threshold-1",
    "hex": "fffff7ff7ffffafffffa3cc3c0",
    "bits": 100,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        15
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        3
      ],
      "e_utran_fdd_reporting_threshold": [
        7
      ],
      "e_utran_fdd_reporting_threshold_2": [
        63
      ],
      "e_utran_tdd_reporting_threshold": [
        7
      ],
      "e_utran_tdd_reporting_threshold_2": [
        63
      ],
      "earfcn": [
        65535
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
      ],
      "e_utran_frequency_index": [
        7,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "earfcn": [
        16
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
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "measure-True-threshold-2",
    "hex": "fffbfdfffffebffffe8f30f0",
    "bits": 94,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        15
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        3
      ],
      "e_utran_fdd_reporting_threshold": [
        7
      ],
      "e_utran_fdd_reporting_offset": [
        7
      ],
      "e_utran_tdd_reporting_threshold": [
        7
      ],
      "e_utran_tdd_reporting_offset": [
        7
      ],
      "earfcn": [
        65535
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
      ],
      "e_utran_frequency_index": [
        7,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ],
      "earfcn": [
        16
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
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "measure-True-threshold-3",
    "hex": "ffffffffffffffebffffe8f30f00",
    "bits": 106,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        15
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        3
      ],
      "e_utran_fdd_reporting_threshold": [
        7
      ],
      "e_utran_fdd_reporting_threshold_2": [
        63
      ],
      "e_utran_fdd_reporting_offset": [
        7
      ],
      "e_utran_tdd_reporting_threshold": [
        7
      ],
      "e_utran_tdd_reporting_threshold_2": [
        63
      ],
      "e_utran_tdd_reporting_offset": [
        7
      ],
      "earfcn": [
        65535
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
      ],
      "e_utran_frequency_index": [
        7,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ],
      "earfcn": [
        16
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
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "ccn-0",
    "hex": "00",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        0
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "ccn-1",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "fdd-0-tdd-0",
    "hex": "6500",
    "bits": 14,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ]
    }
  },
  {
    "name": "fdd-0-tdd-1",
    "hex": "653400",
    "bits": 19,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-0-tdd-2",
    "hex": "65372800",
    "bits": 25,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-0-tdd-3",
    "hex": "653560",
    "bits": 22,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-0-tdd-4",
    "hex": "65372d80",
    "bits": 28,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-1-tdd-0",
    "hex": "656800",
    "bits": 19,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-1-tdd-1",
    "hex": "6569a0",
    "bits": 24,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-1-tdd-2",
    "hex": "6569b940",
    "bits": 30,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-1-tdd-3",
    "hex": "6569ab00",
    "bits": 27,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-1-tdd-4",
    "hex": "6569b96c00",
    "bits": 33,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-2-tdd-0",
    "hex": "656e5000",
    "bits": 25,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-2-tdd-1",
    "hex": "656e5680",
    "bits": 30,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-2-tdd-2",
    "hex": "656e56e500",
    "bits": 36,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-2-tdd-3",
    "hex": "656e56ac00",
    "bits": 33,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-2-tdd-4",
    "hex": "656e56e5b0",
    "bits": 39,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-3-tdd-0",
    "hex": "656ac0",
    "bits": 22,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-3-tdd-1",
    "hex": "656af400",
    "bits": 27,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-3-tdd-2",
    "hex": "656af72800",
    "bits": 33,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-3-tdd-3",
    "hex": "656af560",
    "bits": 30,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-3-tdd-4",
    "hex": "656af72d80",
    "bits": 36,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-4-tdd-0",
    "hex": "656e5b00",
    "bits": 28,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-4-tdd-1",
    "hex": "656e5bd000",
    "bits": 33,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ]
    }
  },
  {
    "name": "fdd-4-tdd-2",
    "hex": "656e5bdca0",
    "bits": 39,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ]
    }
  },
  {
    "name": "fdd-4-tdd-3",
    "hex": "656e5bd580",
    "bits": 36,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "fdd-4-tdd-4",
    "hex": "656e5bdcb600",
    "bits": 42,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "qsearch_p_e_utran": [
        9
      ],
      "e_utran_rep_quant": [
        0
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        5
      ],
      "e_utran_fdd_reporting_threshold_2": [
        37
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        5
      ],
      "e_utran_tdd_reporting_threshold_2": [
        37
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "qsearch_p_e_utran": [
        4
      ],
      "e_utran_rep_quant": [
        1
      ],
      "e_utran_multirat_reporting": [
        2
      ],
      "e_utran_fdd_reporting_threshold": [
        3
      ],
      "e_utran_fdd_reporting_threshold_2": [
        6
      ],
      "e_utran_fdd_reporting_offset": [
        3
      ],
      "e_utran_tdd_reporting_threshold": [
        3
      ],
      "e_utran_tdd_reporting_threshold_2": [
        6
      ],
      "e_utran_tdd_reporting_offset": [
        3
      ]
    }
  },
  {
    "name": "neighbor-options-0",
    "hex": "33039400044400",
    "bits": 51,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "thresh_e_utran_high": [
        17
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-1",
    "hex": "33039d80008880",
    "bits": 54,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "thresh_e_utran_high": [
        17
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
    "name": "neighbor-options-2",
    "hex": "3303940004b880",
    "bits": 54,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-3",
    "hex": "33039d8000971000",
    "bits": 57,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "measurement_bandwidth": [
        3
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-4",
    "hex": "33039400044670",
    "bits": 56,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "thresh_e_utran_low": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-5",
    "hex": "33039d800088ce00",
    "bits": 59,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "measurement_bandwidth": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ],
      "thresh_e_utran_low": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-6",
    "hex": "3303940004b8ce00",
    "bits": 59,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ],
      "thresh_e_utran_low": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-7",
    "hex": "33039d80009719c0",
    "bits": 62,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
      ]
    }
  },
  {
    "name": "neighbor-options-8",
    "hex": "33039400044598",
    "bits": 56,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "thresh_e_utran_high": [
        17
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "e_utran_qrxlevmin": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-9",
    "hex": "33039d800088b300",
    "bits": 59,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "thresh_e_utran_high": [
        17
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "measurement_bandwidth": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ],
      "e_utran_qrxlevmin": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-10",
    "hex": "3303940004b8b300",
    "bits": 59,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        5
      ],
      "e_utran_qrxlevmin": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-11",
    "hex": "33039d8000971660",
    "bits": 62,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
      "e_utran_qrxlevmin": [
        5
      ]
    }
  },
  {
    "name": "neighbor-options-12",
    "hex": "3303940004467cc0",
    "bits": 61,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
    "name": "neighbor-options-13",
    "hex": "33039d800088cf98",
    "bits": 64,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "measurement_bandwidth": [
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
    "name": "neighbor-options-14",
    "hex": "3303940004b8cf98",
    "bits": 64,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
    "name": "neighbor-options-15",
    "hex": "33039d80009719f300",
    "bits": 67,
    "fields": {
      "e_utran_ccn_active": [
        0
      ],
      "earfcn": [
        12345,
        1
      ],
      "measurement_bandwidth": [
        5
      ],
      "e_utran_priority": [
        3
      ],
      "thresh_e_utran_high": [
        17
      ],
      "thresh_e_utran_low": [
        7
      ],
      "e_utran_qrxlevmin": [
        19
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
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
    "name": "list-counts-0-0-0",
    "hex": "80",
    "bits": 5,
    "fields": {
      "e_utran_ccn_active": [
        1
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ]
    }
  },
  {
    "name": "list-counts-0-0-1",
    "hex": "8e11fdd96ad847f765ab2780",
    "bits": 92,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-0-2",
    "hex": "8e11fdd96ad847f765ab279c23fbb2d5b08feecb564f00",
    "bits": 179,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-1-0",
    "hex": "9847f765ab4780",
    "bits": 52,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503
      ],
      "pcid_bitmap_group": [
        37
      ],
      "pcid_pattern_length": [
        2
      ],
      "pcid_pattern": [
        5
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9
      ],
      "pcid_bitmap_group": [
        6
      ],
      "pcid_pattern_length": [
        3
      ],
      "pcid_pattern": [
        3
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-1-1",
    "hex": "9847f765ab479c23fbb2d5b08feecb564f00",
    "bits": 139,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-1-2",
    "hex": "9847f765ab479c23fbb2d5b08feecb564f3847f765ab611fdd96ac9e00",
    "bits": 226,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-2-0",
    "hex": "9847f765ab47b08feecb568f00",
    "bits": 99,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-2-1",
    "hex": "9847f765ab47b08feecb568f3847f765ab611fdd96ac9e00",
    "bits": 186,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-0-2-2",
    "hex": "9847f765ab47b08feecb568f3847f765ab611fdd96ac9e708feecb56c23fbb2d593c00",
    "bits": 273,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-0-0",
    "hex": "b001406000",
    "bits": 33,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ]
    }
  },
  {
    "name": "list-counts-1-0-1",
    "hex": "b0014060e11fdd96ad847f765ab278",
    "bits": 120,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-0-2",
    "hex": "b0014060e11fdd96ad847f765ab279c23fbb2d5b08feecb564f0",
    "bits": 207,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-1-0",
    "hex": "b0014061847f765ab478",
    "bits": 80,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503
      ],
      "pcid_bitmap_group": [
        37
      ],
      "pcid_pattern_length": [
        2
      ],
      "pcid_pattern": [
        5
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9
      ],
      "pcid_bitmap_group": [
        6
      ],
      "pcid_pattern_length": [
        3
      ],
      "pcid_pattern": [
        3
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-1-1",
    "hex": "b0014061847f765ab479c23fbb2d5b08feecb564f0",
    "bits": 167,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-1-2",
    "hex": "b0014061847f765ab479c23fbb2d5b08feecb564f3847f765ab611fdd96ac9e0",
    "bits": 254,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-2-0",
    "hex": "b0014061847f765ab47b08feecb568f0",
    "bits": 127,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-2-1",
    "hex": "b0014061847f765ab47b08feecb568f3847f765ab611fdd96ac9e0",
    "bits": 214,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-1-2-2",
    "hex": "b0014061847f765ab47b08feecb568f3847f765ab611fdd96ac9e708feecb56c23fbb2d593c0",
    "bits": 301,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20
      ],
      "thresh_e_utran_high": [
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16
      ],
      "thresh_e_utran_high": [
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-0-0",
    "hex": "b001406300140600",
    "bits": 61,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ]
    }
  },
  {
    "name": "list-counts-2-0-1",
    "hex": "b00140630014060e11fdd96ad847f765ab2780",
    "bits": 148,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-0-2",
    "hex": "b00140630014060e11fdd96ad847f765ab279c23fbb2d5b08feecb564f00",
    "bits": 235,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-1-0",
    "hex": "b00140630014061847f765ab4780",
    "bits": 108,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503
      ],
      "pcid_bitmap_group": [
        37
      ],
      "pcid_pattern_length": [
        2
      ],
      "pcid_pattern": [
        5
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9
      ],
      "pcid_bitmap_group": [
        6
      ],
      "pcid_pattern_length": [
        3
      ],
      "pcid_pattern": [
        3
      ],
      "pcid_pattern_sense": [
        1
      ],
      "e_utran_frequency_index": [
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-1-1",
    "hex": "b00140630014061847f765ab479c23fbb2d5b08feecb564f00",
    "bits": 195,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-1-2",
    "hex": "b00140630014061847f765ab479c23fbb2d5b08feecb564f3847f765ab611fdd96ac9e00",
    "bits": 282,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-2-0",
    "hex": "b00140630014061847f765ab47b08feecb568f00",
    "bits": 155,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-2-1",
    "hex": "b00140630014061847f765ab47b08feecb568f3847f765ab611fdd96ac9e00",
    "bits": 242,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  },
  {
    "name": "list-counts-2-2-2",
    "hex": "b00140630014061847f765ab47b08feecb568f3847f765ab611fdd96ac9e708feecb56c23fbb2d593c00",
    "bits": 329,
    "fields": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        20,
        20
      ],
      "thresh_e_utran_high": [
        6,
        6
      ],
      "pcid": [
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503,
        17,
        503
      ],
      "pcid_bitmap_group": [
        37,
        37,
        37,
        37,
        37,
        37
      ],
      "pcid_pattern_length": [
        2,
        2,
        2,
        2,
        2,
        2
      ],
      "pcid_pattern": [
        5,
        5,
        5,
        5,
        5,
        5
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        0,
        7,
        0,
        7,
        1,
        6,
        1,
        6
      ]
    },
    "widths": {
      "e_utran_ccn_active": [
        1
      ],
      "earfcn": [
        16,
        16
      ],
      "thresh_e_utran_high": [
        5,
        5
      ],
      "pcid": [
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9,
        9
      ],
      "pcid_bitmap_group": [
        6,
        6,
        6,
        6,
        6,
        6
      ],
      "pcid_pattern_length": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern": [
        3,
        3,
        3,
        3,
        3,
        3
      ],
      "pcid_pattern_sense": [
        1,
        1,
        1,
        1,
        1,
        1
      ],
      "e_utran_frequency_index": [
        3,
        3,
        3,
        3,
        3,
        3,
        3,
        3
      ]
    }
  }
]`)
}

// FuzzEUTRANParametersReplay checks the §12.53 decoder's replay contract.
func FuzzEUTRANParametersReplay(f *testing.F) {
	f.Add([]byte{0x80})
	f.Add([]byte{0x80})
	f.Add([]byte{0x80})
	f.Add([]byte{0x80})
	f.Add([]byte{0xff, 0xf9, 0xe7, 0xff, 0xff, 0xaf, 0xff, 0xff, 0xa3, 0xcc, 0x3c})
	f.Add([]byte{0xff, 0xff, 0xf7, 0xff, 0x7f, 0xff, 0xfa, 0xff, 0xff, 0xfa, 0x3c, 0xc3, 0xc0})
	f.Add([]byte{0xff, 0xfb, 0xfd, 0xff, 0xff, 0xfe, 0xbf, 0xff, 0xfe, 0x8f, 0x30, 0xf0})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xeb, 0xff, 0xff, 0xe8, 0xf3, 0x0f, 0x00})
	f.Add([]byte{0x00})
	f.Add([]byte{0x80})
	f.Add([]byte{0x65, 0x00})
	f.Add([]byte{0x65, 0x34, 0x00})
	f.Add([]byte{0x65, 0x37, 0x28, 0x00})
	f.Add([]byte{0x65, 0x35, 0x60})
	f.Add([]byte{0x65, 0x37, 0x2d, 0x80})
	f.Add([]byte{0x65, 0x68, 0x00})
	f.Add([]byte{0x65, 0x69, 0xa0})
	f.Add([]byte{0x65, 0x69, 0xb9, 0x40})
	f.Add([]byte{0x65, 0x69, 0xab, 0x00})
	f.Add([]byte{0x65, 0x69, 0xb9, 0x6c, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x50, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x56, 0x80})
	f.Add([]byte{0x65, 0x6e, 0x56, 0xe5, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x56, 0xac, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x56, 0xe5, 0xb0})
	f.Add([]byte{0x65, 0x6a, 0xc0})
	f.Add([]byte{0x65, 0x6a, 0xf4, 0x00})
	f.Add([]byte{0x65, 0x6a, 0xf7, 0x28, 0x00})
	f.Add([]byte{0x65, 0x6a, 0xf5, 0x60})
	f.Add([]byte{0x65, 0x6a, 0xf7, 0x2d, 0x80})
	f.Add([]byte{0x65, 0x6e, 0x5b, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x5b, 0xd0, 0x00})
	f.Add([]byte{0x65, 0x6e, 0x5b, 0xdc, 0xa0})
	f.Add([]byte{0x65, 0x6e, 0x5b, 0xd5, 0x80})
	f.Add([]byte{0x65, 0x6e, 0x5b, 0xdc, 0xb6, 0x00})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0x44, 0x00})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x88, 0x80})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0xb8, 0x80})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x97, 0x10, 0x00})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0x46, 0x70})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x88, 0xce, 0x00})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0xb8, 0xce, 0x00})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x97, 0x19, 0xc0})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0x45, 0x98})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x88, 0xb3, 0x00})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0xb8, 0xb3, 0x00})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x97, 0x16, 0x60})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0x46, 0x7c, 0xc0})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x88, 0xcf, 0x98})
	f.Add([]byte{0x33, 0x03, 0x94, 0x00, 0x04, 0xb8, 0xcf, 0x98})
	f.Add([]byte{0x33, 0x03, 0x9d, 0x80, 0x00, 0x97, 0x19, 0xf3, 0x00})
	f.Add([]byte{0x80})
	f.Add([]byte{0x8e, 0x11, 0xfd, 0xd9, 0x6a, 0xd8, 0x47, 0xf7, 0x65, 0xab, 0x27, 0x80})
	f.Add([]byte{0x8e, 0x11, 0xfd, 0xd9, 0x6a, 0xd8, 0x47, 0xf7, 0x65, 0xab, 0x27, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x00})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x80})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x00})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x00})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x00})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x00})
	f.Add([]byte{0x98, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x70, 0x8f, 0xee, 0xcb, 0x56, 0xc2, 0x3f, 0xbb, 0x2d, 0x59, 0x3c, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x60, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x60, 0xe1, 0x1f, 0xdd, 0x96, 0xad, 0x84, 0x7f, 0x76, 0x5a, 0xb2, 0x78})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x60, 0xe1, 0x1f, 0xdd, 0x96, 0xad, 0x84, 0x7f, 0x76, 0x5a, 0xb2, 0x79, 0xc2, 0x3f, 0xbb, 0x2d, 0x5b, 0x08, 0xfe, 0xec, 0xb5, 0x64, 0xf0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x78})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x79, 0xc2, 0x3f, 0xbb, 0x2d, 0x5b, 0x08, 0xfe, 0xec, 0xb5, 0x64, 0xf0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x79, 0xc2, 0x3f, 0xbb, 0x2d, 0x5b, 0x08, 0xfe, 0xec, 0xb5, 0x64, 0xf3, 0x84, 0x7f, 0x76, 0x5a, 0xb6, 0x11, 0xfd, 0xd9, 0x6a, 0xc9, 0xe0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x7b, 0x08, 0xfe, 0xec, 0xb5, 0x68, 0xf0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x7b, 0x08, 0xfe, 0xec, 0xb5, 0x68, 0xf3, 0x84, 0x7f, 0x76, 0x5a, 0xb6, 0x11, 0xfd, 0xd9, 0x6a, 0xc9, 0xe0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x61, 0x84, 0x7f, 0x76, 0x5a, 0xb4, 0x7b, 0x08, 0xfe, 0xec, 0xb5, 0x68, 0xf3, 0x84, 0x7f, 0x76, 0x5a, 0xb6, 0x11, 0xfd, 0xd9, 0x6a, 0xc9, 0xe7, 0x08, 0xfe, 0xec, 0xb5, 0x6c, 0x23, 0xfb, 0xb2, 0xd5, 0x93, 0xc0})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x0e, 0x11, 0xfd, 0xd9, 0x6a, 0xd8, 0x47, 0xf7, 0x65, 0xab, 0x27, 0x80})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x0e, 0x11, 0xfd, 0xd9, 0x6a, 0xd8, 0x47, 0xf7, 0x65, 0xab, 0x27, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x80})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0x9c, 0x23, 0xfb, 0xb2, 0xd5, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x4f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x00})
	f.Add([]byte{0xb0, 0x01, 0x40, 0x63, 0x00, 0x14, 0x06, 0x18, 0x47, 0xf7, 0x65, 0xab, 0x47, 0xb0, 0x8f, 0xee, 0xcb, 0x56, 0x8f, 0x38, 0x47, 0xf7, 0x65, 0xab, 0x61, 0x1f, 0xdd, 0x96, 0xac, 0x9e, 0x70, 0x8f, 0xee, 0xcb, 0x56, 0xc2, 0x3f, 0xbb, 0x2d, 0x59, 0x3c, 0x00})

	descriptor, err := LookupClause("12.53", "E-UTRAN Parameters IE")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 512 {
			t.Skip()
		}
		decoded, err := descriptor.Decode(input)
		if err != nil {
			return
		}
		replay, err := descriptor.Encode(decoded)
		if err != nil || !bytes.Equal(replay, input) {
			t.Fatalf("replay %x, want %x: %v", replay, input, err)
		}
	})
}

// The test set includes each selector independently and repeated structures
// with zero, one and two entries. Every expectation is synthetic.
func TestEUTRANParametersTypedConstruction(t *testing.T) {
	data, err := hex.DecodeString("80")
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeEUTRANParametersIE(data)
	if err != nil {
		t.Fatal(err)
	}
	fresh := EUTRANParametersIE{EUTRANCCNACTIVE: got.Value.EUTRANCCNACTIVE}
	canonical, err := EncodeEUTRANParametersIECanonicalAtLength(fresh, 1)
	if err != nil || !bytes.Equal(canonical, data) {
		t.Fatalf("typed canonical %x: %v", canonical, err)
	}
}

func eutranParametersWalk(value reflect.Value, name string, visit func(string, reflect.Value)) {
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if value.Type() == reflect.TypeFor[runtime.BitString]() {
		visit(name, value)
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if !field.IsExported() || field.Name == "Wire" {
				continue
			}
			eutranParametersWalk(value.Field(i), field.Name, visit)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			eutranParametersWalk(value.Index(i), name, visit)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		visit(name, value)
	}
}

func checkEUTRANParametersVectors(t *testing.T, clause, definition string, typ reflect.Type, text string) {
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
			eutranParametersWalk(value, "", func(name string, field reflect.Value) {
				if name == "Alternative" {
					return
				}
				key := cellPriorityName(name)
				if field.Type() == reflect.TypeFor[runtime.BitString]() {
					bits := field.Interface().(runtime.BitString)
					occurrence := len(actual[key])
					if occurrence >= len(widths[key]) || bits.BitLength != widths[key][occurrence] {
						t.Fatalf("%s bit length %d", name, bits.BitLength)
					}
				}
				actual[key] = append(actual[key], eutranParametersUint(field))
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
			eutranParametersWalk(boundedValue, "", func(name string, field reflect.Value) {
				if name != "Alternative" {
					key := cellPriorityName(name)
					boundedFields[key] = append(boundedFields[key], eutranParametersUint(field))
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
				eutranParametersWalk(changed, "", func(name string, field reflect.Value) {
					if field.Type() == reflect.TypeFor[runtime.BitString]() {
						return
					}
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

func eutranParametersUint(field reflect.Value) uint64 {
	if field.Type() != reflect.TypeFor[runtime.BitString]() {
		return field.Uint()
	}
	bits := field.Interface().(runtime.BitString)
	var value uint64
	for i := 0; i < bits.BitLength; i++ {
		value = value<<1 | uint64((bits.Bytes[i/8]>>uint(7-i%8))&1)
	}
	return value
}
