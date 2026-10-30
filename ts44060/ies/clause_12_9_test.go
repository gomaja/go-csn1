package ies

import "testing"

// TS 44.060 V19.0.0 §12.9 defines the fields and selectors exercised below.
func TestClause129(t *testing.T) {
	elementaryClause(t, "12.9", "Global Power Control Parameters IE", `[{"name":"zero","clause":"12.9","hex":"000000","bits":24,"values":{"ALPHA":0,"TAVGW":0,"TAVGT":0,"Pb":0,"PCMEASCHAN":0,"NAVGI":0},"widths":{"ALPHA":4,"TAVGW":5,"TAVGT":5,"Pb":4,"PCMEASCHAN":1,"NAVGI":4},"absent":[],"present":[]},{"name":"one","clause":"12.9","hex":"108461","bits":24,"values":{"ALPHA":1,"TAVGW":1,"TAVGT":1,"Pb":1,"PCMEASCHAN":1,"NAVGI":1},"widths":{"ALPHA":4,"TAVGW":5,"TAVGT":5,"Pb":4,"PCMEASCHAN":1,"NAVGI":4},"absent":[],"present":[]},{"name":"maximum","clause":"12.9","hex":"ffffef","bits":24,"values":{"ALPHA":15,"TAVGW":31,"TAVGT":31,"Pb":15,"PCMEASCHAN":1,"NAVGI":15},"widths":{"ALPHA":4,"TAVGW":5,"TAVGT":5,"Pb":4,"PCMEASCHAN":1,"NAVGI":4},"absent":[],"present":[]},{"name":"middle","clause":"12.9","hex":"77bdc7","bits":24,"values":{"ALPHA":7,"TAVGW":15,"TAVGT":15,"Pb":7,"PCMEASCHAN":0,"NAVGI":7},"widths":{"ALPHA":4,"TAVGW":5,"TAVGT":5,"Pb":4,"PCMEASCHAN":1,"NAVGI":4},"absent":[],"present":[]}]`)
}

func TestGlobalPowerControlRequiresPrintedZero(t *testing.T) {
	d, err := LookupClause("12.9", "Global Power Control Parameters IE")
	if err != nil {
		t.Fatal(err)
	}
	// TS 44.060 V19.0.0 §12.9 paragraph 22904 prohibits the old value one.
	if _, err := d.Decode([]byte{0, 0, 0x10}); err == nil {
		t.Fatal("accepted prohibited literal one at bit 19")
	}
}
