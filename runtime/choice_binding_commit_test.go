package runtime

import "testing"

func TestChoiceEmptyChildCommitPreservesAncestorBindings(t *testing.T) {
	r := NewReader(nil)
	r.Set("value", 1)
	snapshot := r.Fork()
	var outer, inner ChoiceErrors
	parent := outer.ForkScoped(r)
	parent.Set("value", 2)
	parent.Set("new", 3)
	child := inner.ForkScoped(parent)
	r.Commit(child)
	inner.Release()
	outer.Release()
	for name, want := range map[string]int{"value": 2, "new": 3} {
		if got, err := r.Eval(name); err != nil || got != want {
			t.Fatalf("%s: %d, %v; want %d", name, got, err, want)
		}
	}
	if got, err := snapshot.Eval("value"); err != nil || got != 1 {
		t.Fatalf("changed snapshot: %d, %v", got, err)
	}
}

func TestChoiceEmptyCommitDoesNotCloneBindings(t *testing.T) {
	if raceEnabled {
		t.Skip("pool entries may be dropped by race instrumentation")
	}
	r := NewReader(nil)
	r.Set("value", 1)
	_ = r.Wire()
	var owner ChoiceErrors
	defer owner.Release()
	arm := owner.ForkScoped(r)
	var baselineSnapshot, committedSnapshot *Reader
	baseline := testing.AllocsPerRun(100, func() { baselineSnapshot = r.Fork() })
	withCommit := testing.AllocsPerRun(100, func() {
		committedSnapshot = r.Fork()
		r.Commit(arm)
	})
	if baselineSnapshot == nil || committedSnapshot == nil {
		t.Fatal("missing snapshot")
	}
	if withCommit != baseline {
		t.Fatalf("empty commit adds allocations: fork %g, fork and commit %g", baseline, withCommit)
	}
	r.Set("value", 2)
	if got, err := committedSnapshot.Eval("value"); err != nil || got != 1 {
		t.Fatalf("changed snapshot: %d, %v", got, err)
	}
}
