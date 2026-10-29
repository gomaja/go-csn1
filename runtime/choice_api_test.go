package runtime

import (
	"errors"
	"fmt"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestChoiceKeepDoesNotStealAnotherOwnersFailure(t *testing.T) {
	prior := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(prior)
	r := NewReader(nil)
	var first, other, next ChoiceErrors
	a := first.ForkScoped(r)
	_, e1 := a.ReadUint(1)
	first.Track(e1)
	_, e0 := a.ReadUint(1)
	first.Track(e0)
	kept := other.Keep(r, e0)
	var before *DecodeError
	if !errors.As(kept, &before) {
		t.Fatal("missing retained error")
	}
	saved := *before
	first.Release()
	b := next.ForkScoped(r)
	_, x := b.ReadUint(1)
	next.Track(x)
	_, y := b.ReadUint(1)
	next.Track(y)
	defer next.Release()
	if x == y {
		t.Fatal("two live failures share a pooled object")
	}
	var after *DecodeError
	if !errors.As(kept, &after) || *after != saved {
		t.Fatal("other owner's error changed")
	}
}

func TestChoiceLeaveBelowBorrowedPathIsIndependent(t *testing.T) {
	r := NewReader(nil)
	for _, name := range []string{"p0", "p1"} {
		if err := r.Enter(name); err != nil {
			t.Fatal(err)
		}
	}
	var trials ChoiceErrors
	defer trials.Release()
	a := trials.ForkScoped(r)
	a.Leave()
	if err := a.Enter("x"); err != nil {
		t.Fatal(err)
	}
	var de *DecodeError
	if !errors.As(r.Error(Truncated, "missing"), &de) || de.Path != "p0/p1" {
		t.Fatalf("source path changed: %v", de)
	}
	b := trials.ForkScoped(a)
	b.Leave()
	b.Leave()
	if err := b.Enter("other"); err != nil {
		t.Fatal(err)
	}
	if path := strings.Join(a.path, "/"); path != "p0/x" {
		t.Fatalf("ancestor trial path changed: %s", path)
	}
}

func TestChoiceCommitIntoUnrelatedReaderSharesBothSides(t *testing.T) {
	for _, write := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "written"}[write], func(t *testing.T) {
			source, destination := NewReader(nil), NewReader(nil)
			source.Set("value", 1)
			destination.Set("absent", 9)
			var trials ChoiceErrors
			arm := trials.ForkScoped(source)
			if write {
				arm.Set("extra", 3)
			}
			destination.Commit(arm)
			trials.Release()
			source.Set("value", 2)
			if got, err := destination.Eval("value"); err != nil || got != 1 {
				t.Fatalf("destination changed: %d %v", got, err)
			}
			destination.Set("value", 4)
			if got, err := source.Eval("value"); err != nil || got != 2 {
				t.Fatalf("source changed: %d %v", got, err)
			}
			if _, err := destination.Eval("absent"); err == nil {
				t.Fatal("destination-only binding survived commit")
			}
		})
	}
}

func TestChoiceReaderCopyMustUsePublicFork(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	arm := trials.ForkScoped(r)
	arm.Set("value", 1)
	snapshot := arm.Fork()
	snapshot.Set("value", 2)
	if got, err := arm.Eval("value"); err != nil || got != 1 {
		t.Fatalf("snapshot changed trial: %d %v", got, err)
	}
	copied := *arm
	defer func() {
		if recover() == nil {
			t.Fatal("copy of pooled binding overlay accepted")
		}
	}()
	copied.Set("value", 3)
}

func TestChoiceCommitIntoUnrelatedReaderKeepsPathSnapshot(t *testing.T) {
	source, destination := NewReader(nil), NewReader(nil)
	for _, name := range []string{"Root", "Before"} {
		if err := source.Enter(name); err != nil {
			t.Fatal(err)
		}
	}
	var trials ChoiceErrors
	arm := trials.ForkScoped(source)
	destination.Commit(arm)
	trials.Release()
	source.Leave()
	if err := source.Enter("After"); err != nil {
		t.Fatal(err)
	}
	var de *DecodeError
	if !errors.As(destination.Error(Truncated, "missing"), &de) || de.Path != "Root/Before" {
		t.Fatalf("committed path changed: %v", de)
	}
	destination.Leave()
	if err := destination.Enter("Destination"); err != nil {
		t.Fatal(err)
	}
	if !errors.As(source.Error(Truncated, "missing"), &de) || de.Path != "Root/After" {
		t.Fatalf("source path changed: %v", de)
	}
}

func TestChoiceErrorCanBeInspectedBeforeKeep(t *testing.T) {
	r := NewReader(nil)
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	var trials ChoiceErrors
	arm := trials.Fork(r)
	if err := arm.Enter("Field"); err != nil {
		t.Fatal(err)
	}
	_, err := arm.ReadUint(3)
	trials.Track(err)
	want := &fieldDecodeError{DecodeError: DecodeError{Kind: Truncated, Offset: 0, Path: "Root/Field", Detail: "required bit absent"}, width: 3}
	if err.Error() != want.Error() {
		t.Fatalf("text %q; want %q", err.Error(), want.Error())
	}
	var de *DecodeError
	if !errors.As(err, &de) || *de != want.DecodeError {
		t.Fatalf("decode facts: %#v", de)
	}
	var op *OperationError
	if !errors.As(err, &op) || op.Location.Path != "Root/Field" || op.Location.Width != 3 {
		t.Fatalf("operation facts: %#v", op)
	}
	saved := *de
	trials.Release()
	for range 100 {
		var discarded ChoiceErrors
		b := discarded.Fork(r)
		discarded.Track(b.Error(InvalidValue, "replacement"))
		discarded.Release()
	}
	if *de != saved || err.Error() != want.Error() {
		t.Fatal("observed error changed after release")
	}
}

func TestChoiceOpaqueDiagnosticsAlwaysRefuseInspection(t *testing.T) {
	prior := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(prior)
	r := NewReader(nil)
	var trials ChoiceErrors
	a := trials.ForkScoped(r)
	_, err := a.ReadUint(1)
	trials.Track(err)
	inspect := func(stage string) {
		t.Helper()
		for name, call := range map[string]func(){"text": func() { _ = err.Error() }, "is": func() { _ = errors.Is(err, ErrLHPositionUndefined) }, "as": func() { var de *DecodeError; _ = errors.As(err, &de) }, "unwrap": func() { _ = errors.Unwrap(err) }} {
			t.Run(stage+"/"+name, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Fatal("opaque handle inspected")
					}
				}()
				call()
			})
		}
	}
	inspect("active")
	trials.Release()
	// The original alias must still refuse inspection when its slot is live
	// again. A released bit on the recycled object cannot provide that property.
	var replacement ChoiceErrors
	b := replacement.ForkScoped(r)
	_, fresh := b.ReadUint(2)
	replacement.Track(fresh)
	defer replacement.Release()
	inspect("reused")
}

func TestChoicePublicErrorFirstInspectionAfterRelease(t *testing.T) {
	r := NewReader(nil)
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	var trials ChoiceErrors
	arm := trials.Fork(r)
	_, err := arm.ReadUint(5)
	trials.Track(err)
	trials.Release()
	for range 32 {
		var discarded ChoiceErrors
		other := discarded.ForkScoped(r)
		discarded.Track(other.Error(InvalidValue, "reuse"))
		discarded.Release()
	}
	var de *DecodeError
	var op *OperationError
	if !errors.As(err, &de) || de.Path != "Root" || !errors.As(err, &op) || op.Location.Width != 5 {
		t.Fatalf("public error changed: %v", err)
	}
	if err.Error() != "CSN.1 truncated at bit 0 in Root: required bit absent" {
		t.Fatal(err)
	}
}

func TestChoiceDeferredPathMatchesMaterializedPath(t *testing.T) {
	r := NewReader(nil)
	for _, part := range []string{"Root", "Container"} {
		if err := r.Enter(part); err != nil {
			t.Fatal(err)
		}
	}
	var trials ChoiceErrors
	defer trials.Release()
	a := trials.ForkScoped(r)
	if err := a.Enter("Leaf"); err != nil {
		t.Fatal(err)
	}
	_, err := a.ReadUint(3)
	normalize := func(err error, parent, child string) error {
		var de *DecodeError
		if !errors.As(err, &de) {
			t.Fatal("missing decode error")
		}
		copy := *de
		copy.Path = parent + "/" + child + "/" + copy.Path
		return &copy
	}
	err, ok := DeferReaderPath(err, r, "Child", normalize)
	if !ok {
		t.Fatal("path materialized early")
	}
	trials.Track(err)
	a.Leave()
	a.Leave()
	if enterErr := a.Enter("Replacement"); enterErr != nil {
		t.Fatal(enterErr)
	}
	kept := trials.Keep(r, err)
	var de *DecodeError
	if !errors.As(kept, &de) || de.Path != "Root/Container/Child/Root/Container/Leaf" {
		t.Fatalf("deferred container path changed: %v", kept)
	}
}

func TestChoicePredicatesAndMixedTruncationKeepSemantics(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	a := trials.ForkScoped(r)
	reserved := a.Error(InvalidValue, "reserved CSN.1 alternative")
	trials.Track(reserved)
	if !IsReservedAlternative(reserved) || !IsReservedAlternative(trials.Keep(r, reserved)) {
		t.Fatal("lost reserved predicate")
	}
	for _, reverse := range []bool{false, true} {
		scoped := a.ErrorAt(Truncated, 7, "scoped")
		trials.Track(scoped)
		ordinary := r.ErrorAt(InvalidValue, 9, "ordinary")
		current, candidate := scoped, ordinary
		if reverse {
			current, candidate = candidate, current
		}
		if got := PreferTruncation(current, candidate); got != current {
			t.Fatal("mixed offset precedence changed")
		}
	}
	refusal := a.Error(InvalidSchema, "position")
	refusal.(*deferredFailure).lh = true
	trials.Track(refusal)
	if !IsLHPositionUndefined(refusal) || !IsLHPositionUndefined(trials.Keep(r, refusal)) {
		t.Fatal("lost refusal predicate")
	}
}

func TestChoiceKeepRejectsRecycledHandle(t *testing.T) {
	prior := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(prior)
	for _, mode := range []string{"keep", "release", "trial"} {
		t.Run(mode, func(t *testing.T) {
			r := NewReader(nil)
			if err := r.Enter("Root"); err != nil {
				t.Fatal(err)
			}
			var trials, parent ChoiceErrors
			a := trials.ForkScoped(r)
			_, handle := a.ReadUint(2)
			trials.Track(handle)
			switch mode {
			case "keep":
				_ = trials.Keep(r, handle)
			case "release":
				trials.Release()
			case "trial":
				outer := parent.ForkScoped(r)
				forwarded := trials.Keep(outer, handle)
				parent.Track(forwarded)
				_ = parent.Keep(r, forwarded)
			}
			func() {
				defer func() {
					recovered := recover()
					if recovered == nil {
						t.Error("Keep accepted a recycled diagnostic")
					} else if message, ok := recovered.(string); !ok || !strings.Contains(message, "consumed or released") {
						t.Errorf("unclear panic: %v", recovered)
					}
				}()
				_ = trials.Keep(r, handle)
			}()
			trials.Release()
			parent.Release()
			var next ChoiceErrors
			b := next.ForkScoped(r)
			_, x := b.ReadUint(1)
			_, y := b.ReadUint(2)
			if x == y {
				t.Fatal("two live failures share a pooled object")
			}
			next.Track(x)
			next.Track(y)
			next.Release()
		})
	}
}

func TestChoiceCommitRejectsCopiedTrial(t *testing.T) {
	for _, mode := range []string{"receiver", "receiver-to-trial", "argument"} {
		t.Run(mode, func(t *testing.T) {
			r := NewReader(nil)
			var trials ChoiceErrors
			defer trials.Release()
			arm := trials.ForkScoped(r)
			arm.Set("v", 1)
			copied := *arm
			destination := NewReader(nil)
			if mode == "receiver-to-trial" {
				destination = trials.ForkScoped(r)
				destination.Set("other", 2)
			}
			func() {
				defer func() {
					recovered := recover()
					if recovered == nil {
						t.Error("Commit accepted a copied trial")
					} else if message, ok := recovered.(string); !ok || !strings.Contains(message, "copied by value") {
						t.Errorf("unclear panic: %v", recovered)
					}
				}()
				if mode == "argument" {
					destination.Commit(&copied)
				} else {
					copied.Commit(destination)
				}
			}()
			if got, err := arm.Eval("v"); err != nil || got != 1 {
				t.Error("copied commit changed original binding")
			}
			if mode == "receiver-to-trial" {
				if _, exists := arm.trial.values["other"]; exists {
					t.Error("copied commit wrote another trial's bindings")
				}
			}
		})
	}
}

func TestChoiceOpaqueFormattingPrintsPanicMarker(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	arm := trials.ForkScoped(r)
	_, handle := arm.ReadUint(1)
	trials.Track(handle)
	for _, text := range []string{fmt.Sprintf("%v", handle), fmt.Errorf("ctx: %w", handle).Error()} {
		if !strings.Contains(text, "%!v(PANIC=Error method: CSN.1 opaque choice diagnostic") {
			t.Fatalf("missing panic marker: %s", text)
		}
	}
}

func TestChoiceKeepAcceptsLiveEmptyKind(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	arm := trials.ForkScoped(r)
	handle := arm.Error("", "custom diagnostic")
	trials.Track(handle)
	kept := trials.Keep(r, handle)
	var cause *DecodeError
	if !errors.As(kept, &cause) || cause.Kind != "" || cause.Detail != "custom diagnostic" {
		t.Fatal("live custom diagnostic rejected")
	}
}
