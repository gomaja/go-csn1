//go:build race

package ies

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
