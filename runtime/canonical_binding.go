package runtime

// SetCanonical binds a name already normalized by the generated decoder.
// Except for internal names beginning with NUL, name must equal its lowercase,
// whitespace-folded form. Set accepts unnormalized names from other callers.
// Normalization at generation preserves TS 24.007 V20.0.0 Annex B §B.1.1.5
// without rescanning the same literal after every received field.
func (r *Reader) SetCanonical(name string, value uint64) {
	r.setBinding(name, value, true)
}

func (r *Reader) setBinding(name string, value uint64, canonical bool) {
	r.checkTrialIdentity()
	if !canonical {
		name = key(name)
	}
	if r.trial != nil {
		r.trial.values[name] = value
		return
	}
	if r.varsShared {
		r.vars = cloneVars(r.vars)
		r.varsShared = false
	}
	if r.vars == nil {
		r.vars = make(map[string]uint64)
	}
	r.vars[name] = value
}
