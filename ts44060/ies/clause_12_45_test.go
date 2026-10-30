package ies

import "testing"

// TS 44.060 V19.0.0 §12.45 defines the fields and selectors exercised below.
func TestClause1245(t *testing.T) {
	elementaryClause(t, "12.45", "MBMS In-band Signalling Indicator IE", `[{"name":"zero","clause":"12.45","hex":"00","bits":1,"values":{"MBMSInBandSignallingIndicator":0},"widths":{"MBMSInBandSignallingIndicator":1},"absent":[],"present":[]},{"name":"one","clause":"12.45","hex":"80","bits":1,"values":{"MBMSInBandSignallingIndicator":1},"widths":{"MBMSInBandSignallingIndicator":1},"absent":[],"present":[]},{"name":"maximum","clause":"12.45","hex":"80","bits":1,"values":{"MBMSInBandSignallingIndicator":1},"widths":{"MBMSInBandSignallingIndicator":1},"absent":[],"present":[]},{"name":"alternating","clause":"12.45","hex":"00","bits":1,"values":{"MBMSInBandSignallingIndicator":0},"widths":{"MBMSInBandSignallingIndicator":1},"absent":[],"present":[]},{"name":"boundary-MBMSInBandSignallingIndicator","clause":"12.45","hex":"80","bits":1,"values":{"MBMSInBandSignallingIndicator":1},"widths":{"MBMSInBandSignallingIndicator":1},"absent":[],"present":[]}]`)
}
