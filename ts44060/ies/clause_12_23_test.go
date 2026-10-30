package ies

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.23 defines the fields and selectors exercised below.
func TestClause1223(t *testing.T) {
	elementaryClause(t, "12.23", "Cell Identification IE", `[{"name":"zero","clause":"12.23","hex":"0000000000000000","bits":64,"values":{"LocationAreaIdentificationIE":0,"RAC":0,"CellIdentityIE":0},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"one","clause":"12.23","hex":"0000000001010001","bits":64,"values":{"LocationAreaIdentificationIE":1,"RAC":1,"CellIdentityIE":1},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"maximum","clause":"12.23","hex":"ffffffffffffffff","bits":64,"values":{"LocationAreaIdentificationIE":1099511627775,"RAC":255,"CellIdentityIE":65535},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"alternating","clause":"12.23","hex":"aaaaaaaaaaaaaaaa","bits":64,"values":{"LocationAreaIdentificationIE":733007751850,"RAC":170,"CellIdentityIE":43690},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"boundary-LocationAreaIdentificationIE","clause":"12.23","hex":"ffffffffff000000","bits":64,"values":{"LocationAreaIdentificationIE":1099511627775,"RAC":0,"CellIdentityIE":0},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"boundary-RAC","clause":"12.23","hex":"0000000000ff0000","bits":64,"values":{"LocationAreaIdentificationIE":0,"RAC":255,"CellIdentityIE":0},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]},{"name":"boundary-CellIdentityIE","clause":"12.23","hex":"000000000000ffff","bits":64,"values":{"LocationAreaIdentificationIE":0,"RAC":0,"CellIdentityIE":65535},"widths":{"LocationAreaIdentificationIE":40,"RAC":8,"CellIdentityIE":16},"absent":[],"present":[]}]`)
}

func TestCellIdentificationFixedOctetBoundaries(t *testing.T) {
	d, err := LookupClause("12.23", "Cell Identification IE")
	if err != nil {
		t.Fatal(err)
	}
	// TS 44.060 V19.0.0 §12.23 paragraphs 23328 and 23330 require five and two octets.
	for _, width := range []int{0, 1, 15, 17, 39, 41, 48} {
		lai := runtime.BitString{Bytes: make([]byte, (width+7)/8), BitLength: width}
		cell := runtime.BitString{Bytes: make([]byte, 2), BitLength: 16}
		if _, err := d.CanonicalAtLength(CellIdentificationIE{LocationAreaIdentificationIE: lai, CellIdentityIE: cell}, 8); err == nil {
			t.Errorf("accepted %d-bit location area", width)
		}
		lai = runtime.BitString{Bytes: make([]byte, 5), BitLength: 40}
		cell = runtime.BitString{Bytes: make([]byte, (width+7)/8), BitLength: width}
		if _, err := d.CanonicalAtLength(CellIdentificationIE{LocationAreaIdentificationIE: lai, CellIdentityIE: cell}, 8); err == nil {
			t.Errorf("accepted %d-bit cell identity", width)
		}
	}
}
