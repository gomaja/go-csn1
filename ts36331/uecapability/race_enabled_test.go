//go:build race

package uecapability

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
