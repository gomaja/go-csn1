//go:build race

package singleuplink

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
