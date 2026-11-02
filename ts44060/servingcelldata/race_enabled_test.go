//go:build race

package servingcelldata

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
