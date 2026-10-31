package arithaudit

import (
	"go/importer"
	"testing"
)

// requireStandardLibraryTypes skips a repository scan on a host whose Go
// installation cannot supply standard-library export data, such as a Windows
// test host without a Go toolchain. The scanner then cannot resolve the types
// of imported values and conservatively reports every unresolved operand as
// possible integer arithmetic, so its findings would describe the host rather
// than the code.
func requireStandardLibraryTypes(t *testing.T) {
	t.Helper()
	if _, err := importer.Default().Import("strconv"); err != nil {
		t.Skipf("standard-library export data is unavailable, so imported types cannot be resolved: %v", err)
	}
}
