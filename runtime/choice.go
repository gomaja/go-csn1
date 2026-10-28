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
			c.readers = r.choiceNext
			extra := r.bindings.extra
			clear(extra)
			*r = Reader{}
			r.bindings.extra = extra[:0]
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
	if r.choice {
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
var choiceReaders = sync.Pool{New: func() any { return new(Reader) }}

// Fork borrows a trial reader until Release. Generated alternatives must
// balance their path scopes before Commit or another call to Fork.
func (c *ChoiceErrors) Fork(r *Reader) *Reader {
	if r.pathShared {
		r.ownPath()
	}
	r.implicitShared, r.truncShared, r.spareCountsShared = true, true, true
	fork := choiceReaders.Get().(*Reader)
	extra := fork.bindings.extra
	*fork = *r
	fork.choiceNext, c.readers = c.readers, fork
	fork.choice = true
	fork.pathShared = false
	fork.bindings = choiceBindings{base: r, extra: extra[:0]}
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

type choiceBinding struct {
	name  string
	value uint64
}
type choiceBindings struct {
	base   *Reader
	inline [8]choiceBinding
	count  int
	extra  []choiceBinding
	shared bool
}

func (b *choiceBindings) set(name string, value uint64) {
	for i := 0; i < b.count; i++ {
		if b.inline[i].name == name {
			b.inline[i].value = value
			return
		}
	}
	for i := range b.extra {
		if b.extra[i].name == name {
			if b.shared {
				b.extra = append([]choiceBinding(nil), b.extra...)
				b.shared = false
			}
			b.extra[i].value = value
			return
		}
	}
	if b.count < len(b.inline) {
		b.inline[b.count] = choiceBinding{name, value}
		b.count++
		return
	}
	if b.shared {
		b.extra = append([]choiceBinding(nil), b.extra...)
		b.shared = false
	}
	b.extra = append(b.extra, choiceBinding{name, value})
}

func (r *Reader) binding(name string) (uint64, bool) {
	if r.choice {
		for i := 0; i < r.bindings.count; i++ {
			if r.bindings.inline[i].name == name {
				return r.bindings.inline[i].value, true
			}
		}
		for _, entry := range r.bindings.extra {
			if entry.name == name {
				return entry.value, true
			}
		}
		if r.bindings.base != nil {
			return r.bindings.base.binding(name)
		}
	}
	n, ok := r.vars[name]
	return n, ok
}

func (r *Reader) applyBindings(target *Reader, stop *Reader) {
	if !r.choice {
		return
	}
	if r.bindings.base != nil && r.bindings.base != stop {
		r.bindings.base.applyBindings(target, stop)
	}
	for i := 0; i < r.bindings.count; i++ {
		e := r.bindings.inline[i]
		target.Set(e.name, e.value)
	}
	for _, e := range r.bindings.extra {
		target.Set(e.name, e.value)
	}
}

func (r *Reader) commitChoice(other *Reader) {
	if !other.choice {
		next := r.choiceNext
		if r.collector != nil {
			r.commitCollected(other)
		} else {
			*r = *other
		}
		r.choice, r.bindings, r.choiceNext = true, choiceBindings{}, next
		return
	}
	mode, vars, shared, bindings, next := r.choice, r.vars, r.varsShared, r.bindings, r.choiceNext
	// Apply into a separate receiver so inherited overlay links to r remain
	// immutable until all writes have been read.
	target := Reader{choice: mode, vars: vars, varsShared: shared, bindings: bindings}
	other.applyBindings(&target, r)
	if r.collector != nil {
		r.commitCollected(other)
	} else {
		*r = *other
	}
	r.choice, r.vars, r.varsShared, r.bindings, r.choiceNext = mode, target.vars, target.varsShared, target.bindings, next
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
	accept := func(label string, n uint64) bool {
		if strings.ReplaceAll(label, " ", "") != compact {
			return true
		}
		if found {
			return false
		}
		value, found = n, true
		return true
	}
	root := r
	for root.choice && root.bindings.base != nil {
		root = root.bindings.base
	}
	for label := range root.vars {
		n, _ := r.binding(label)
		if !accept(label, n) {
			return 0, false, fmt.Errorf("ambiguous width variable %q", name)
		}
	}
	for owner := r; owner.choice; owner = owner.bindings.base {
		check := func(e choiceBinding) bool {
			if _, exists := root.vars[e.name]; exists {
				return true
			}
			for newer := r; newer != owner; newer = newer.bindings.base {
				if newer.bindings.has(e.name) {
					return true
				}
			}
			return accept(e.name, e.value)
		}
		for i := 0; i < owner.bindings.count; i++ {
			if !check(owner.bindings.inline[i]) {
				return 0, false, fmt.Errorf("ambiguous width variable %q", name)
			}
		}
		for _, e := range owner.bindings.extra {
			if !check(e) {
				return 0, false, fmt.Errorf("ambiguous width variable %q", name)
			}
		}
		if owner.bindings.base == nil {
			break
		}
	}
	return value, found, nil
}
func (b *choiceBindings) has(name string) bool {
	for i := 0; i < b.count; i++ {
		if b.inline[i].name == name {
			return true
		}
	}
	for _, e := range b.extra {
		if e.name == name {
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
	for root.choice && root.bindings.base != nil {
		root = root.bindings.base
	}
	snapshot := Reader{vars: cloneVars(root.vars)}
	r.applyBindings(&snapshot, nil)
	r.vars, r.varsShared, r.choice, r.bindings = snapshot.vars, true, false, choiceBindings{}
	r.choiceNext = nil
	path := make([]string, len(r.path), max(16, len(r.path)))
	copy(path, r.path)
	r.path, r.pathShared = path, false
}
