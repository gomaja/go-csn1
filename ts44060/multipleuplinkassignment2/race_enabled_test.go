//go:build race

package multipleuplinkassignment2

// raceEnabled reports a -race build. Its instrumentation adds to every
// allocation, so byte totals measured through runtime.MemStats grow.
const raceEnabled = true
