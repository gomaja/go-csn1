//go:build race

package notification

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
