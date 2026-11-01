//go:build race

package vgcsreconfigure2

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
