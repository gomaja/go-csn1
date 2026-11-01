//go:build race

package restoctets_test

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
