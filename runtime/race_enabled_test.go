//go:build race

package runtime

// raceEnabled reports a -race build. Its instrumentation turns off the
// compiler's append-of-make optimization, so allocation counts differ.
const raceEnabled = true
