//go:build race

package controlmessages_test

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
