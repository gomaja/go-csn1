//go:build race

package pdchpairs

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
