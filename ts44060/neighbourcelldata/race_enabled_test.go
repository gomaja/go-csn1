//go:build race

package neighbourcelldata

// Allocation counts and byte totals vary with race instrumentation.
const raceEnabled = true
