package runtime

import (
	"fmt"
	"strings"
	"sync"
)

// ChoiceErrors owns the deferred diagnostics of serial generated choice arms.
// Its zero value is ready; every tracked error must be kept or released before
// the enclosing generated choice returns. TS 24.007 V20.0.0 Annex B §B.1.1.4.
type ChoiceErrors struct {
	head    *deferredFailure
	readers *Reader
}

type deferredFrame struct {
	operation, path string
	offset, width   int
	source          Source
	kind            ErrorKind
	operationFn     func(error, string, string, int, int, Source, ErrorKind) error
	indexFn         func(error, string, int) error
}

type deferredFailure struct {
	cause        DecodeError
	width        int
	lh           bool
	path         [128]string
	pathLen      int
	frames       []deferredFrame
	inlineFrames [8]deferredFrame
	next         *deferredFailure
	tracked      bool
}

var deferredFailures = sync.Pool{New: func() any {
	f := new(deferredFailure)
	f.frames = f.inlineFrames[:0]
	return f
}}

func (f *deferredFailure) Error() string        { return f.cause.Error() }
func (f *deferredFailure) Unwrap() error        { return &f.cause }
func (f *deferredFailure) Is(target error) bool { return f.lh && target == ErrLHPositionUndefined }

func releaseFailure(f *deferredFailure) {
	clear(f.path[:])
	clear(f.frames)
	f.frames = f.frames[:0]
	f.cause = DecodeError{}
	f.width, f.pathLen = 0, 0
	f.lh, f.tracked = false, false
	f.next = nil
	deferredFailures.Put(f)
}

// Track retains a trial error until the choice is resolved. Ordinary errors
// need no scratch ownership and are left untouched.
//
//go:noinline
func (c *ChoiceErrors) Track(err error) {
	if f, ok := err.(*deferredFailure); ok && !f.tracked {
		f.next, c.head, f.tracked = c.head, f, true
	}
}

// Release returns discarded diagnostic scratch. Kept errors are detached first.
func (c *ChoiceErrors) Release() {
	defer func() {
		for c.readers != nil {
			r := c.readers
			state := r.trial
			c.readers = state.next
			clear(state.values)
			*state = choiceTrial{values: state.values}
			*r = Reader{trial: state}
			choiceReaders.Put(r)
		}
	}()

	for c.head != nil {
		f := c.head
		c.head = f.next
		releaseFailure(f)
	}
}

// Keep transfers the chosen diagnostic to its containing trial, or builds its
// original error chain when returning to a nontrial caller. Only diagnostic
// normalization is replayed: decoder work and observations are never replayed.
func (c *ChoiceErrors) Keep(r *Reader, err error) error {
	f, ok := err.(*deferredFailure)
	if !ok {
		return err
	}
	for link := &c.head; *link != nil; link = &(*link).next {
		if *link == f {
			*link = f.next
			break
		}
	}
	f.next, f.tracked = nil, false
	if r.trial != nil {
		return f
	}
	f.cause.Path = strings.Join(f.path[:f.pathLen], "/")
	var out error
	if f.width != 0 {
		out = &fieldDecodeError{DecodeError: f.cause, width: f.width - 1}
	} else {
		cause := f.cause
		out = &cause
	}
	if f.lh {
		out = &lhPositionError{cause: out}
	}
	for _, frame := range f.frames {
		if frame.operationFn != nil {
			out = frame.operationFn(out, frame.operation, frame.path, frame.offset, frame.width, frame.source, frame.kind)
		} else {
			out = frame.indexFn(out, frame.path, frame.offset)
		}
	}
	releaseFailure(f)
	return out
}

// DeferOperation records a generated error normalizer while an alternative is
// speculative. The callback must be a pure diagnostic transformation.
func DeferOperation(err error, operation, path string, offset, width int, source Source, kind ErrorKind, normalize func(error, string, string, int, int, Source, ErrorKind) error) (error, bool) {
	f, ok := err.(*deferredFailure)
	if !ok {
		return nil, false
	}
	f.frames = append(f.frames, deferredFrame{operation: operation, path: path, offset: offset, width: width, source: source, kind: kind, operationFn: normalize})
	return f, true
}

// DeferIndex records a repeated-field error normalizer without building its
// indexed path for a rejected alternative.
func DeferIndex(err error, path string, index int, normalize func(error, string, int) error) (error, bool) {
	f, ok := err.(*deferredFailure)
	if !ok {
		return nil, false
	}
	f.frames = append(f.frames, deferredFrame{path: path, offset: index, indexFn: normalize})
	return f, true
}

// ErrorAt reports the same decode failure as Error at an explicit absolute bit
// offset. Generated source constraints use the field's starting position.
func (r *Reader) ErrorAt(kind ErrorKind, offset int, detail string) error {
	return r.failAt(kind, offset, detail)
}

func (r *Reader) deferredFailure(kind ErrorKind, offset int, detail string) error {
	f := deferredFailures.Get().(*deferredFailure)
	f.cause = DecodeError{Kind: kind, Offset: offset, Detail: detail}
	f.width = int(r.fieldWidthHi)<<16 | int(r.fieldWidthLo)
	f.pathLen = copy(f.path[:], r.path)
	return f
}

// A choice trial borrows the path stack for a serial generated alternative. Every
// successful Enter must be paired with Leave before resolving the choice;
// neither the trial nor its borrowed stack may escape that scope. Failures
// snapshot their path before a later arm can write that stack. Bindings remain
// isolated overlays until Commit. Public Fork retains independent COW paths.
// TS 24.007 V20.0.0 Annex B §§B.1.1.3–B.1.1.5.
var choiceReaders = sync.Pool{New: func() any {
	slot := new(struct {
		reader Reader
		trial  choiceTrial
	})
	slot.reader.trial = &slot.trial
	slot.trial.values = make(map[string]uint64)
	return &slot.reader
}}

// Fork borrows a trial reader until Release. Generated alternatives must
// balance their path scopes before Commit or another call to Fork.
func (c *ChoiceErrors) Fork(r *Reader) *Reader {
	if r.pathShared {
		r.ownPath()
	}
	r.implicitShared, r.truncShared, r.spareCountsShared = true, true, true
	fork := choiceReaders.Get().(*Reader)
	state := fork.trial
	*fork = *r
	*state = choiceTrial{base: r, next: c.readers, values: state.values}
	fork.trial, c.readers = state, fork
	fork.pathShared = false
	if r.collector != nil {
		fork.observationStart = int32(len(r.collector.observations))
		fork.observationDropped = r.collector.dropped
	}
	fork.wire.Spare, fork.wire.Padding, fork.wire.Terminal = clipWireBits(fork.wire.Spare, fork.wire.Padding, fork.wire.Terminal)
	return fork
}

func (r *Reader) ownPath() {
	fresh := make([]string, len(r.path), max(16, len(r.path)))
	copy(fresh, r.path)
	r.path, r.pathShared = fresh, false
}

type choiceTrial struct {
	base   *Reader
	next   *Reader
	values map[string]uint64
}

func (r *Reader) binding(name string) (uint64, bool) {
	if r.trial != nil {
		if value, ok := r.trial.values[name]; ok {
			return value, true
		}
		if r.trial.base != nil {
			return r.trial.base.binding(name)
		}
	}
	n, ok := r.vars[name]
	return n, ok
}

// bindingTarget carries only the variable state of a commit. A complete Reader
// would also copy wire and path state that binding application never reads.
// Names in the source maps are already canonical under TS 24.007 V20.0.0
// Annex B §B.1.1.5; applying them must not normalize or retain a trial reader.
type bindingTarget struct {
	values map[string]uint64
	shared bool
}

//go:noinline
func (b *bindingTarget) set(name string, value uint64) {
	if b.shared {
		b.values = cloneVars(b.values)
		b.shared = false
	}
	if b.values == nil {
		b.values = make(map[string]uint64)
	}
	b.values[name] = value
}

func (r *Reader) applyBindings(target *bindingTarget, stop *Reader) {
	if r.trial == nil {
		return
	}
	if r.trial.base != nil && r.trial.base != stop {
		r.trial.base.applyBindings(target, stop)
	}
	for name, value := range r.trial.values {
		target.set(name, value)
	}
}

func (r *Reader) commitChoice(other *Reader) {
	if r == other {
		return
	}
	owned := r.trial
	if other.trial == nil {
		if r.collector != nil {
			r.commitCollected(other)
		} else {
			*r = *other
		}
		clear(owned.values)
		owned.base = nil
		r.trial = owned
		return
	}
	// Apply into a separate receiver so inherited overlay links to r remain
	// immutable until all writes have been read.
	target := bindingTarget{values: r.vars, shared: r.varsShared}
	var state choiceTrial
	if owned != nil {
		state = *owned
	}
	base := other
	for base.trial != nil && base.trial.base != nil && base.trial.base != r {
		base = base.trial.base
	}
	if base.trial == nil || base.trial.base == nil {
		// A detached snapshot supplies the complete variable state, including
		// the absence of names that only the receiving reader had retained.
		target.values, target.shared = base.vars, true
		if owned != nil {
			clear(state.values)
			state.base = nil
		}
	}
	if owned == nil {
		other.applyBindings(&target, r)
	} else {
		// Set writes through this already-owned map without retaining a
		// pointer to the stack copy of the trial metadata.
		overlay := bindingTarget{values: state.values}
		other.applyBindings(&overlay, r)
	}
	if r.collector != nil {
		r.commitCollected(other)
	} else {
		*r = *other
	}
	r.vars, r.varsShared, r.trial = target.values, target.shared, owned
	if owned != nil {
		*owned = state
	}
}

func literalDifference(symbol rune) string {
	switch symbol {
	case '0':
		return "literal 0 differs"
	case '1':
		return "literal 1 differs"
	case 'L':
		return "literal L differs"
	case 'H':
		return "literal H differs"
	}
	return fmt.Sprintf("literal %c differs", symbol)
}

// resolveBinding applies exactly the map resolver's canonical and compact
// name rules (TS 24.007 V20.0.0 Annex B §B.1.1.5) to the visible overlay.
func (r *Reader) resolveBinding(name, canonical string) (uint64, bool, error) {
	if n, ok := r.binding(canonical); ok {
		return n, true, nil
	}
	compact := strings.ReplaceAll(canonical, " ", "")
	var value uint64
	found := false
	root := r
	for root.trial != nil && root.trial.base != nil {
		root = root.trial.base
	}
	for label := range root.vars {
		if strings.ReplaceAll(label, " ", "") != compact {
			continue
		}
		if found {
			return 0, false, fmt.Errorf("ambiguous width variable %q", name)
		}
		value, _ = r.binding(label)
		found = true
	}
	for owner := r; owner.trial != nil; owner = owner.trial.base {
		for label, n := range owner.trial.values {
			if strings.ReplaceAll(label, " ", "") != compact {
				continue
			}
			if _, exists := root.vars[label]; exists || r.bindingShadowed(owner, label) {
				continue
			}
			if found {
				return 0, false, fmt.Errorf("ambiguous width variable %q", name)
			}
			value, found = n, true
		}
		if owner.trial.base == nil {
			break
		}
	}
	return value, found, nil
}

// bindingShadowed skips a name already supplied by a newer overlay. Keep this
// walk separate from compact matching so the compiler does not duplicate the
// nested traversal in the resolver's map loop.
func (r *Reader) bindingShadowed(owner *Reader, name string) bool {
	for newer := r; newer != owner; newer = newer.trial.base {
		if _, exists := newer.trial.values[name]; exists {
			return true
		}
	}
	return false
}

// A public Fork must remain independent even when its source is a borrowed
// trial. Copying its active path and visible bindings detaches every pooled
// ancestor before another alternative can reuse the borrowed stack.
func (r *Reader) freezeChoiceSnapshot() {
	root := r
	for root.trial != nil && root.trial.base != nil {
		root = root.trial.base
	}
	snapshot := bindingTarget{values: cloneVars(root.vars)}
	r.applyBindings(&snapshot, nil)
	r.vars, r.varsShared, r.trial = snapshot.values, true, nil
	path := make([]string, len(r.path), max(16, len(r.path)))
	copy(path, r.path)
	r.path, r.pathShared = path, false
}
