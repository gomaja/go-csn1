//go:build race

package vgcsreconfigure

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
