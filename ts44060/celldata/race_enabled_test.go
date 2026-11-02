//go:build race

package celldata_test

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
