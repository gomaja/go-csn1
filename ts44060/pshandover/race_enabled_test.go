//go:build race

package pshandover

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
