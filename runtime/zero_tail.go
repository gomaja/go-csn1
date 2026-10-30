package runtime

// AllRemainingZero reports whether every bit from the current position to
// the enclosing input limit is zero. It neither advances nor changes reader
// state. TS 44.018 V19.0.0 §3.4.1.2.1.3 and table 9.1.55.1 distinguish
// redundant zero bitmap positions after the 96 cell positions from release
// additions. The scan takes at most one step per remaining input bit.
// State after ChoiceErrors.Release is undefined for a borrowed trial.
func (r *Reader) AllRemainingZero() bool {
	if r.Check() != nil {
		return false
	}
	for position := r.pos; position < r.end; position++ {
		if r.data[position/8]&(1<<uint(7-position%8)) != 0 {
			return false
		}
	}
	return true
}
