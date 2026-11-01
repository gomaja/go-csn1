//go:build race

package access_test

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
