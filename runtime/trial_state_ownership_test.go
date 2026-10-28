package runtime

import (
	"testing"
	"unsafe"
)

func TestChoiceTrialStatePreservesReceiverSlot(t *testing.T) {
	r := NewReader(nil)
	if r.trial != nil {
		t.Fatal("ordinary reader owns speculative state")
	}
	var outer, inner ChoiceErrors
	a := outer.ForkScoped(r)
	owned := a.trial
	a.Set("local", 2)
	snapshot := a.Fork()
	if snapshot.trial != nil {
		t.Fatal("public snapshot retained pooled state")
	}
	a.Commit(snapshot)
	if a.trial != owned {
		t.Fatal("snapshot commit replaced receiver state")
	}
	b := inner.ForkScoped(a)
	if b.trial == owned {
		t.Fatal("nested trial shares receiver state")
	}
	b.Set("local", 3)
	a.Commit(b)
	if a.trial != owned {
		t.Fatal("nested commit replaced receiver state")
	}
	inner.Release()
	r.Commit(a)
	outer.Release()
	if r.trial != nil || a.trial != owned || len(owned.values) != 0 || owned.base != nil || owned.next != nil {
		t.Fatal("release retained or transferred pooled ownership")
	}
	if value, err := r.Eval("local"); err != nil || value != 3 {
		t.Fatalf("released binding: %d %v", value, err)
	}
	t.Logf("Reader=%d bytes; choiceTrial=%d bytes", unsafe.Sizeof(Reader{}), unsafe.Sizeof(choiceTrial{}))
}

func TestChoiceTrialStateCommitAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("pool entries can be dropped under race instrumentation")
	}
	r := NewReader(nil)
	r.Set("seed", 1)
	allocs := testing.AllocsPerRun(100, func() {
		var outer, inner ChoiceErrors
		a := outer.ForkScoped(r)
		a.Set("one", 2)
		b := inner.ForkScoped(a)
		b.Set("two", 3)
		a.Commit(b)
		inner.Release()
		r.Commit(a)
		outer.Release()
	})
	if allocs != 0 {
		t.Fatalf("trial metadata escaped during commit: %g allocations", allocs)
	}
}
