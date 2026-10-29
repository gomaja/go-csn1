//go:build race

package measurement

import "testing"

func TestMeasurementResultsReportAllocations(t *testing.T) {
	t.Skip("race instrumentation randomly drops sync.Pool values")
}
