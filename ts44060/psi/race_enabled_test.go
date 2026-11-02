//go:build race

package psi_test

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
