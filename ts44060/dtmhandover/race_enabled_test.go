//go:build race

package dtmhandover

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
