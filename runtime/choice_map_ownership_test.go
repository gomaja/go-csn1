package runtime

import (
	"fmt"
	"testing"
)

func TestChoiceMapOwnershipAfterNestedRelease(t *testing.T) {
	r := NewReader(nil)
	var outer, inner ChoiceErrors
	a := outer.ForkScoped(r)
	for i := 0; i < 24; i++ {
		a.Set(fmt.Sprintf("binding%d", i), uint64(i))
	}
	b := inner.ForkScoped(a)
	b.Set("binding23", 900)
	b.Set("nested", 1000)
	snapshot := b.Fork()
	a.Commit(b)
	inner.Release()
	for range 32 {
		var discarded ChoiceErrors
		c := discarded.ForkScoped(r)
		for i := 0; i < 24; i++ {
			c.Set(fmt.Sprintf("binding%d", i), 9999)
		}
		discarded.Release()
	}
	for _, reader := range []*Reader{a, snapshot} {
		for name, want := range map[string]int{"binding23": 900, "nested": 1000} {
			if got, err := reader.Eval(name); err != nil || got != want {
				t.Fatalf("after inner release %s: %d %v", name, got, err)
			}
		}
	}
	r.Commit(a)
	outer.Release()
	for _, reader := range []*Reader{r, snapshot} {
		if got, err := reader.Eval("binding23"); err != nil || got != 900 {
			t.Fatalf("after outer release: %d %v", got, err)
		}
	}
}

func TestChoiceMapSnapshotCommitKeepsVisibleBindings(t *testing.T) {
	r := NewReader(nil)
	r.Set("base", 1)
	var trials ChoiceErrors
	a := trials.ForkScoped(r)
	a.Set("local", 2)
	snapshot := a.Fork()
	snapshot.Set("local", 3)
	snapshot.Set("snapshot only", 6)
	a.Commit(snapshot)
	if got, err := a.Eval("local"); err != nil || got != 3 {
		t.Fatalf("snapshot commit into trial: %d %v", got, err)
	}
	a.Set("local", 4)
	a.Set("after", 5)
	r.Commit(a)
	trials.Release()
	if got, err := r.Eval("local"); err != nil || got != 4 {
		t.Fatalf("snapshot commit through outer reader: %d %v", got, err)
	}
	if got, err := r.Eval("snapshot only"); err != nil || got != 6 {
		t.Fatalf("unmodified snapshot binding lost: %d %v", got, err)
	}
	if got, err := r.Eval("after"); err != nil || got != 5 {
		t.Fatalf("write following snapshot commit: %d %v", got, err)
	}
	if got, err := snapshot.Eval("local"); err != nil || got != 3 {
		t.Fatalf("snapshot changed through trial ownership: %d %v", got, err)
	}
}

func TestChoiceMapDetachedSnapshotReplacesBindings(t *testing.T) {
	r := NewReader(nil)
	r.Set("dropped", 1)
	var outer, inner ChoiceErrors
	a := outer.ForkScoped(r)
	a.Set("old local", 2)
	unrelated := NewReader(nil)
	unrelated.Set("kept", 3)
	snapshot := unrelated.Fork()
	a.Commit(snapshot)
	a.Set("after", 4)
	b := inner.ForkScoped(a)
	b.Set("nested", 5)
	a.Commit(b)
	inner.Release()
	a.Commit(a)
	r.Commit(a)
	outer.Release()
	for _, name := range []string{"dropped", "old local"} {
		if _, err := r.Eval(name); err == nil {
			t.Fatalf("snapshot retained absent binding %q", name)
		}
	}
	for name, want := range map[string]int{"kept": 3, "after": 4, "nested": 5} {
		if got, err := r.Eval(name); err != nil || got != want {
			t.Fatalf("detached snapshot binding %s: %d %v", name, got, err)
		}
	}
}
