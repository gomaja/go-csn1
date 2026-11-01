//go:build race

package measurement

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
