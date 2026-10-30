package ies

import (
	"reflect"
	"testing"
)

// TS 44.060 V19.0.0 §12.8.4, DOCX paragraphs 22865-22883.
func TestClause1284Vectors(t *testing.T) {
	checkPublicationVectors(t, "12.8.4", "DLMC Frequency Parameters IE", reflect.TypeFor[DLMCFrequencyParametersIE](), []publicationVector{
		{"e7fe", 15, `{"TSC":7,"ARFCNChoice":{"Alternative":0,"ARFCN":{"ARFCN":1023},"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":null}}`, nil},
		{"ef80", 10, `{"TSC":7,"ARFCNChoice":{"Alternative":1,"ARFCN":null,"DLMCIndirectEncoding":{"DLMCIndirectEncoding":{"MANUMBER":15,"CHANGEMARK1Group":null}},"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":null}}`, nil},
		{"eff0", 13, `{"TSC":7,"ARFCNChoice":{"Alternative":1,"ARFCN":null,"DLMCIndirectEncoding":{"DLMCIndirectEncoding":{"MANUMBER":15,"CHANGEMARK1Group":{"CHANGEMARK1":3,"CHANGEMARK2":null}}},"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":null}}`, nil},
		{"effc", 15, `{"TSC":7,"ARFCNChoice":{"Alternative":1,"ARFCN":null,"DLMCIndirectEncoding":{"DLMCIndirectEncoding":{"MANUMBER":15,"CHANGEMARK1Group":{"CHANGEMARK1":3,"CHANGEMARK2":2}}},"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":null}}`, nil},
		{"f00010", 20, `{"TSC":7,"ARFCNChoice":{"Alternative":2,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":{"DLMCDirectEncoding1":{"GPRSMobileAllocation":{"HSN":0,"RFLNumberList":null,"MALENGTHChoice":{"Alternative":0,"MALENGTH":{"MALENGTH":0,"MABITMAP":{"Bytes":"gA==","BitLength":1}},"ARFCNIndexList":null}}}},"DLMCDirectEncoding2":null}}`, nil},
		{"f7ff93fe00", 38, `{"TSC":7,"ARFCNChoice":{"Alternative":2,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":{"DLMCDirectEncoding1":{"GPRSMobileAllocation":{"HSN":63,"RFLNumberList":{"Entries":[{"RFLNUMBER":15},{"RFLNUMBER":2}]},"MALENGTHChoice":{"Alternative":1,"MALENGTH":null,"ARFCNIndexList":{"ARFCNIndexList":{"Entries":[{"ARFCNINDEX":63},{"ARFCNINDEX":0}]}}}}}},"DLMCDirectEncoding2":null}}`, nil},
		{"f028", 14, `{"TSC":7,"ARFCNChoice":{"Alternative":2,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":{"DLMCDirectEncoding1":{"GPRSMobileAllocation":{"HSN":1,"RFLNumberList":null,"MALENGTHChoice":{"Alternative":1,"MALENGTH":null,"ARFCNIndexList":{"ARFCNIndexList":null}}}}},"DLMCDirectEncoding2":null}}`, nil},
		{"f3e7f55555555555555540", 83, `{"TSC":7,"ARFCNChoice":{"Alternative":2,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":{"DLMCDirectEncoding1":{"GPRSMobileAllocation":{"HSN":31,"RFLNumberList":null,"MALENGTHChoice":{"Alternative":0,"MALENGTH":{"MALENGTH":63,"MABITMAP":{"Bytes":"qqqqqqqqqqo=","BitLength":64}},"ARFCNIndexList":null}}}},"DLMCDirectEncoding2":null}}`, nil},
		{"ffe0002244", 39, `{"TSC":7,"ARFCNChoice":{"Alternative":3,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":{"DLMCDirectEncoding2":{"HSN":63,"LengthOfMAFrequencyListContents":0,"MAFrequencyListContents":{"Bytes":"ABEi","BitLength":24}}}}}`, nil},
		{"fffe0022446688aaccef1133557799bbddfe2042", 159, `{"TSC":7,"ARFCNChoice":{"Alternative":3,"ARFCN":null,"DLMCIndirectEncoding":null,"DLMCDirectEncoding1":null,"DLMCDirectEncoding2":{"DLMCDirectEncoding2":{"HSN":63,"LengthOfMAFrequencyListContents":15,"MAFrequencyListContents":{"Bytes":"ABEiM0RVZneImaq7zN3u/xAh","BitLength":144}}}}}`, nil},
	})
}
