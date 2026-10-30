//go:build race

package messages

// raceEnabled reports a -race build. Its instrumentation drops pooled
// objects at random, so allocation counts vary between runs.
const raceEnabled = true
