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

// DecodeBits resolves a definition with an explicit significant bit extent.
func DecodeBits(standard, name string, data []byte, bitLength int) (any, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, err
	}
	return d.DecodeBits(data, bitLength)
}

// DecodeBitsWithReport keeps receiver diagnostics separate from observations
// of received reserved values. Errors retain the committed prefix reports.
func DecodeBitsWithReport(standard, name string, data []byte, bitLength int) (any, runtime.Report, runtime.ReceiverReport, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, runtime.Report{}, runtime.ReceiverReport{}, err
	}
	return d.DecodeBitsWithReport(data, bitLength)
}
