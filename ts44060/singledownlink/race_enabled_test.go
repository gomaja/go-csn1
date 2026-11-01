//go:build race

package singledownlink

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
