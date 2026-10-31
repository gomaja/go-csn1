package runtime

import (
	"errors"
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
	pathFn          func(error, string, string) error
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

// Opaque scoped handles never expose pooled memory, even after reuse. Use
// ChoiceErrors.Keep to obtain an ordinary error before inspection or escape.
func (*deferredFailure) Error() string {
	panic("CSN.1 opaque choice diagnostic: call ChoiceErrors.Keep before inspection")
}
func (*deferredFailure) Unwrap() error {
	panic("CSN.1 opaque choice diagnostic: call ChoiceErrors.Keep before inspection")
}
func (*deferredFailure) Is(error) bool {
	panic("CSN.1 opaque choice diagnostic: call ChoiceErrors.Keep before inspection")
}
func (*deferredFailure) As(any) bool {
	panic("CSN.1 opaque choice diagnostic: call ChoiceErrors.Keep before inspection")
}

func releaseFailure(f *deferredFailure) {
	// Every acquired entry starts zeroed beyond pathLen, and capture only
	// writes this prefix. Clear just the owned span so rejected arms do not
	// scan the full nesting-limit array through the GC write barrier.
	// TS 24.007 V20.0.0 Annex B §B.1.1.3 preserves the captured context.
	clear(f.path[:f.pathLen])
	clear(f.frames)
	f.frames = f.frames[:0]
	f.cause = DecodeError{}
	f.width, f.pathLen = 0, -1
	f.lh, f.tracked = false, false
	f.next = nil
	deferredFailures.Put(f)
}

// Track retains a trial error until the choice is resolved. Ordinary errors
// need no scratch ownership and are left untouched. Like Keep, Track panics
// with "CSN.1 choice diagnostic already consumed or released" if a handle
// is still in the pool; reuse after another trial acquires it is undetectable.
//
//go:noinline
func (c *ChoiceErrors) Track(err error) {
	if f, ok := err.(*deferredFailure); ok {
		if f.pathLen < 0 {
			panic("CSN.1 choice diagnostic already consumed or released")
		}
		if !f.tracked {
			f.next, c.head, f.tracked = c.head, f, true
		}
	}
}

// Release returns discarded diagnostic scratch. Kept errors are detached first.
// Keep consumes its input handle. Handles are dead after Keep or Release; do
// not reuse them. Track and Keep detect a handle still in the pool, but cannot detect
// reuse after another trial acquires it.
func (c *ChoiceErrors) Release() {
	defer func() {
		for c.readers != nil {
			r := c.readers
			state := r.trial
			c.readers = state.next
			clear(state.values)
			*state = choiceTrial{values: state.values}
			// A released trial is dead; ForkScoped overwrites every field
			// before reuse. Drop references without clearing scalar state.
			// TS 24.007 V20.0.0 Annex B §B.1.1.4 isolates serial trials.
			r.data, r.vars, r.steps, r.path = nil, nil, nil, nil
			r.wire = WireInfo{}
			// Fork collectors own immutable diagnostic prefixes. Releasing this
			// reference drops uncommitted events; commits retain their own prefix.
			r.collector = nil
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
// Keep consumes the input handle; use only its returned error afterwards. A
// handle is dead after Keep or Release. Keep panics if the handle is still in
// the pool; reuse after another trial acquires it cannot be detected.
func (c *ChoiceErrors) Keep(r *Reader, err error) error {
	f, ok := err.(*deferredFailure)
	if !ok {
		return err
	}
	if f.pathLen < 0 {
		panic("CSN.1 choice diagnostic already consumed or released")
	}
	// A diagnostic owned by a different live choice must not be detached or
	// recycled here. Build an independent chain without changing its owner.
	found := false
	for link := &c.head; *link != nil; link = &(*link).next {
		if *link == f {
			*link = f.next
			found = true
			break
		}
	}
	if !found && f.tracked {
		return f.materialize()
	}
	f.next, f.tracked = nil, false
	if r.trial != nil {
		return f
	}
	out := f.materialize()
	releaseFailure(f)
	return out
}

func (f *deferredFailure) materialize() error {
	cause := f.cause
	cause.Path = strings.Join(f.path[:f.pathLen], "/")
	var out error
	if f.width != 0 {
		out = &fieldDecodeError{DecodeError: cause, width: f.width - 1}
	} else {
		out = &cause
	}
	if f.lh {
		out = &lhPositionError{cause: out}
	}
	for _, frame := range f.frames {
		if frame.operationFn != nil {
			out = frame.operationFn(out, frame.operation, frame.path, frame.offset, frame.width, frame.source, frame.kind)
		} else if frame.pathFn != nil {
			parent := strings.Join(f.path[:frame.offset], "/")
			out = frame.pathFn(out, parent, frame.path)
		} else {
			out = frame.indexFn(out, frame.path, frame.offset)
		}
	}
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

// Fork returns an independent public reader snapshot. Its ordinary errors
// support Error, errors.Is and errors.As immediately and remain valid after
// Release. Use ForkScoped only for generated, serial speculative alternatives.
func (c *ChoiceErrors) Fork(r *Reader) *Reader { return r.Fork() }

// ForkScoped borrows a reader until Release for generated serial alternatives.
// Balance Enter/Leave before Commit or another ForkScoped. The source reader
// must remain unchanged during the trial, and the trial must not be copied by
// value. Use Reader.Fork to obtain an independent snapshot.
//
// Failures are opaque scoped handles: Error, Is, As and Unwrap always panic,
// including after Release and pool reuse. Track each failure and use Keep
// before inspection or return. Only DeferOperation, DeferIndex,
// DeferReaderPath, PreferTruncation, IsReservedAlternative, IsRegionSyntaxError and
// IsLHPositionUndefined may inspect them inside the active scope. Neither a
// reader nor an opaque handle may be used by these operations after Release.
// Read-only trial state after Release is undefined. Set, ForkScoped and Commit
// refuse a released trial until its pooled reader is acquired again.
// Keep consumes its input handle; that handle is dead after Keep or Release.
// Track and Keep detect a handle still in the pool, but cannot detect reuse
// after another trial acquires it. Never format or wrap a scoped failure before Keep: fmt
// recovers its Error panic and prints a %!v(PANIC=Error method: ...) marker.
// TS 24.007 V20.0.0 Annex B §§B.1.1.3-B.1.1.5.
func (c *ChoiceErrors) ForkScoped(r *Reader) *Reader {
	r.checkTrialIdentity()
	if r.pathShared {
		r.ownPath()
	}
	r.implicitShared, r.truncShared, r.spareCountsShared = true, true, true
	fork := choiceReaders.Get().(*Reader)
	state := fork.trial
	*fork = *r
	*state = choiceTrial{base: r, next: c.readers, values: state.values, pathFloor: len(r.path), reader: fork}
	fork.trial, c.readers = state, fork
	fork.pathShared = false
	if r.collector != nil {
		c := r.collector.observationCollector()
		fork.observationStart = int32(len(c.observations))
		fork.observationDropped = c.dropped
		if r.collector.receiver != nil {
			fork.collector = r.collector.forkReceiver()
		}
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
	base      *Reader
	next      *Reader
	values    map[string]uint64
	pathFloor int
	reader    *Reader
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

func (r *Reader) applyBindings(target *bindingTarget, stop *Reader) {
	if r.trial == nil {
		return
	}
	if r.trial.base != nil && r.trial.base != stop {
		r.trial.base.applyBindings(target, stop)
	}
	// Empty overlays do not acquire a map or detach shared bindings. For a
	// written overlay, establish ownership once before applying its fields.
	// TS 24.007 V20.0.0 Annex B §B.1.1.5 preserves the binding context.
	if len(r.trial.values) == 0 {
		return
	}
	if target.shared {
		target.values = cloneVars(target.values)
		target.shared = false
	}
	if target.values == nil {
		target.values = make(map[string]uint64)
	}
	for name, value := range r.trial.values {
		target.values[name] = value
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
		base.varsShared = true
		// The destination also retains a borrowed prefix. Every source
		// sharing that storage must detach before changing that prefix.
		for owner := other; owner != nil; {
			owner.pathShared = true
			if owner.trial == nil {
				break
			}
			owner = owner.trial.base
		}
		if owned != nil {
			clear(state.values)
			state.base = nil
		}
	}
	if owned == nil {
		other.applyBindings(&target, r)
	} else {
		// Apply writes through this already-owned map without retaining a
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
			if _, exists := root.vars[label]; exists || r.bindingShadowed(owner, label) {
				continue
			}
			if strings.ReplaceAll(label, " ", "") != compact {
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
//
//go:noinline
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

// Trial readers own pooled mutable binding overlays. Copying a trial Reader by
// value would alias that overlay; use Reader.Fork for an independent snapshot.
func (r *Reader) checkTrialIdentity() {
	if r.trial != nil && r.trial.reader != r {
		panic("CSN.1 trial reader copied by value; use Reader.Fork")
	}
}

// IsLHPositionUndefined tests the source-position refusal without exposing an
// opaque scoped diagnostic. Ordinary errors use errors.Is as usual.
// TS 24.007 V20.0.0 Annex B §B.1.2.2 defines positioned L/H symbols.
func IsLHPositionUndefined(err error) bool {
	if f, ok := err.(*deferredFailure); ok {
		return f.lh
	}
	return errors.Is(err, ErrLHPositionUndefined)
}

// IsReservedAlternative tests the reserved-arm failure while a generated
// choice is active. Ordinary errors retain the same leaf-decoding rules.
func IsReservedAlternative(err error) bool {
	var cause *DecodeError
	if f, ok := err.(*deferredFailure); ok {
		cause = &f.cause
	} else {
		cause = decodeCause(err)
	}
	return cause != nil && cause.Kind == InvalidValue && (cause.Detail == "reserved CSN.1 alternative" || cause.Detail == "value reserved by source table")
}

func choiceFailurePosition(err error) (ErrorKind, int, bool) {
	if err == nil {
		return "", 0, false
	}
	if f, ok := err.(*deferredFailure); ok {
		return f.cause.Kind, f.cause.Offset, true
	}
	var cause *DecodeError
	if !errors.As(err, &cause) {
		return "", 0, false
	}
	return cause.Kind, cause.Offset, true
}

// DeferReaderPath records a pure container-child path transformation while a
// generated choice is active. The reader's current path must be a prefix of
// the failure's captured path; no pooled reader or path storage is retained.
// TS 24.007 V20.0.0 Annex B §B.1.1.3 preserves concatenation context.
func DeferReaderPath(err error, r *Reader, child string, normalize func(error, string, string) error) (error, bool) {
	f, ok := err.(*deferredFailure)
	if !ok {
		return nil, false
	}
	if len(r.path) > f.pathLen {
		panic("CSN.1 deferred child path is not a failure prefix")
	}
	for i, part := range r.path {
		if part != f.path[i] {
			panic("CSN.1 deferred child path is not a failure prefix")
		}
	}
	f.frames = append(f.frames, deferredFrame{path: child, offset: len(r.path), pathFn: normalize})
	return f, true
}
