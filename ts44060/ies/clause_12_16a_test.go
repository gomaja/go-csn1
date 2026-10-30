package ies

import "testing"

// TS 44.060 V19.0.0 §12.16a defines the fields and selectors exercised below.
func TestClause1216a(t *testing.T) {
	elementaryClause(t, "12.16a", "G-RNTI IE", `[{"name":"zero","clause":"12.16a","hex":"00000000","bits":32,"values":{"SRNTI":0,"ServingBSCIdentity":0},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]},{"name":"one","clause":"12.16a","hex":"00001001","bits":32,"values":{"SRNTI":1,"ServingBSCIdentity":1},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]},{"name":"maximum","clause":"12.16a","hex":"ffffffff","bits":32,"values":{"SRNTI":1048575,"ServingBSCIdentity":4095},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]},{"name":"alternating","clause":"12.16a","hex":"aaaaaaaa","bits":32,"values":{"SRNTI":699050,"ServingBSCIdentity":2730},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]},{"name":"boundary-SRNTI","clause":"12.16a","hex":"fffff000","bits":32,"values":{"SRNTI":1048575,"ServingBSCIdentity":0},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]},{"name":"boundary-ServingBSCIdentity","clause":"12.16a","hex":"00000fff","bits":32,"values":{"SRNTI":0,"ServingBSCIdentity":4095},"widths":{"SRNTI":20,"ServingBSCIdentity":12},"absent":[],"present":[]}]`)
}
