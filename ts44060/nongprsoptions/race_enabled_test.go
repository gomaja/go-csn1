//go:build race

package nongprsoptions

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
