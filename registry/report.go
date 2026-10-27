package registry

import "github.com/gomaja/go-csn1/runtime"

// DecodeWithReport resolves a printed definition and returns observations
// separately from the decoded value. Errors retain the committed prefix.
func DecodeWithReport(standard, name string, data []byte) (any, runtime.Report, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, runtime.Report{}, err
	}
	return d.DecodeWithReport(data)
}

// DecodeWithReportWithContext supplies the containing SI4 ACS bit for SI7/SI8.
func DecodeWithReportWithContext(standard, name string, data []byte, acs runtime.SI4ACS) (any, runtime.Report, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, runtime.Report{}, err
	}
	return d.DecodeWithReportWithContext(data, acs)
}
