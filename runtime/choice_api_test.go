package runtime

import (
	"errors"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestChoiceKeepDoesNotStealAnotherOwnersFailure(t *testing.T) {
	prior := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(prior)
	r := NewReader(nil)
	var first, other, next ChoiceErrors
	a := first.Fork(r)
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
	b := next.Fork(r)
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
	a := trials.Fork(r)
	a.Leave()
	if err := a.Enter("x"); err != nil {
		t.Fatal(err)
	}
	var de *DecodeError
	if !errors.As(r.Error(Truncated, "missing"), &de) || de.Path != "p0/p1" {
		t.Fatalf("source path changed: %v", de)
	}
	b := trials.Fork(a)
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
			arm := trials.Fork(source)
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
	arm := trials.Fork(r)
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
	arm := trials.Fork(source)
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
