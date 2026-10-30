package ies

import "testing"

// TS 44.060 V19.0.0 §12.14 defines the fields and selectors exercised below.
func TestClause1214(t *testing.T) {
	elementaryClause(t, "12.14", "PRACH Control Parameters IE", `[{"name":"persistence-False-0","clause":"12.14","hex":"00001b0000","bits":33,"values":{"ACCCONTRCLASS":0,"S":0,"TXINT":0,"MAXRETRANSList.0":0,"MAXRETRANSList.1":1,"MAXRETRANSList.2":2,"MAXRETRANSList.3":3},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2},"absent":["PERSISTENCELEVELList"],"present":[]},{"name":"persistence-False-1","clause":"12.14","hex":"00016c1100","bits":33,"values":{"ACCCONTRCLASS":1,"S":1,"TXINT":1,"MAXRETRANSList.0":1,"MAXRETRANSList.1":2,"MAXRETRANSList.2":3,"MAXRETRANSList.3":0},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2},"absent":["PERSISTENCELEVELList"],"present":[]},{"name":"persistence-False-2","clause":"12.14","hex":"ffffb1ff00","bits":33,"values":{"ACCCONTRCLASS":65535,"S":15,"TXINT":15,"MAXRETRANSList.0":2,"MAXRETRANSList.1":3,"MAXRETRANSList.2":0,"MAXRETRANSList.3":1},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2},"absent":["PERSISTENCELEVELList"],"present":[]},{"name":"persistence-False-3","clause":"12.14","hex":"7fffc67700","bits":33,"values":{"ACCCONTRCLASS":32767,"S":7,"TXINT":7,"MAXRETRANSList.0":3,"MAXRETRANSList.1":0,"MAXRETRANSList.2":1,"MAXRETRANSList.3":2},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2},"absent":["PERSISTENCELEVELList"],"present":[]},{"name":"persistence-True-0","clause":"12.14","hex":"00001b00809180","bits":49,"values":{"ACCCONTRCLASS":0,"S":0,"TXINT":0,"MAXRETRANSList.0":0,"MAXRETRANSList.1":1,"MAXRETRANSList.2":2,"MAXRETRANSList.3":3,"PERSISTENCELEVELList.0":0,"PERSISTENCELEVELList.1":1,"PERSISTENCELEVELList.2":2,"PERSISTENCELEVELList.3":3},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2,"PERSISTENCELEVELList.0":4,"PERSISTENCELEVELList.1":4,"PERSISTENCELEVELList.2":4,"PERSISTENCELEVELList.3":4},"absent":[],"present":["PERSISTENCELEVELList"]},{"name":"persistence-True-1","clause":"12.14","hex":"00016c11ab3c00","bits":49,"values":{"ACCCONTRCLASS":1,"S":1,"TXINT":1,"MAXRETRANSList.0":1,"MAXRETRANSList.1":2,"MAXRETRANSList.2":3,"MAXRETRANSList.3":0,"PERSISTENCELEVELList.0":5,"PERSISTENCELEVELList.1":6,"PERSISTENCELEVELList.2":7,"PERSISTENCELEVELList.3":8},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2,"PERSISTENCELEVELList.0":4,"PERSISTENCELEVELList.1":4,"PERSISTENCELEVELList.2":4,"PERSISTENCELEVELList.3":4},"absent":[],"present":["PERSISTENCELEVELList"]},{"name":"persistence-True-2","clause":"12.14","hex":"ffffb1ffd5e680","bits":49,"values":{"ACCCONTRCLASS":65535,"S":15,"TXINT":15,"MAXRETRANSList.0":2,"MAXRETRANSList.1":3,"MAXRETRANSList.2":0,"MAXRETRANSList.3":1,"PERSISTENCELEVELList.0":10,"PERSISTENCELEVELList.1":11,"PERSISTENCELEVELList.2":12,"PERSISTENCELEVELList.3":13},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2,"PERSISTENCELEVELList.0":4,"PERSISTENCELEVELList.1":4,"PERSISTENCELEVELList.2":4,"PERSISTENCELEVELList.3":4},"absent":[],"present":["PERSISTENCELEVELList"]},{"name":"persistence-True-3","clause":"12.14","hex":"7fffc677f80900","bits":49,"values":{"ACCCONTRCLASS":32767,"S":7,"TXINT":7,"MAXRETRANSList.0":3,"MAXRETRANSList.1":0,"MAXRETRANSList.2":1,"MAXRETRANSList.3":2,"PERSISTENCELEVELList.0":15,"PERSISTENCELEVELList.1":0,"PERSISTENCELEVELList.2":1,"PERSISTENCELEVELList.3":2},"widths":{"ACCCONTRCLASS":16,"S":4,"TXINT":4,"MAXRETRANSList.0":2,"MAXRETRANSList.1":2,"MAXRETRANSList.2":2,"MAXRETRANSList.3":2,"PERSISTENCELEVELList.0":4,"PERSISTENCELEVELList.1":4,"PERSISTENCELEVELList.2":4,"PERSISTENCELEVELList.3":4},"absent":[],"present":["PERSISTENCELEVELList"]}]`)
}

func TestPRACHFixedArrayCounts(t *testing.T) {
	d, err := LookupClause("12.14", "PRACH Control Parameters IE")
	if err != nil {
		t.Fatal(err)
	}
	// TS 44.060 V19.0.0 §12.14 paragraphs 23202 and 23205 require four entries.
	for _, n := range []int{0, 1, 3, 5, 8} {
		if _, err := d.CanonicalAtLength(PRACHControlParametersIE{MAXRETRANSList: make([]uint8, n)}, 5); err == nil {
			t.Errorf("accepted %d retransmission values", n)
		}
		persistence := make([]uint8, n)
		if _, err := d.CanonicalAtLength(PRACHControlParametersIE{MAXRETRANSList: make([]uint8, 4), PERSISTENCELEVELList: &persistence}, 7); err == nil {
			t.Errorf("accepted %d persistence values", n)
		}
	}
}
