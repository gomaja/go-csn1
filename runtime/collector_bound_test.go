package runtime

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCollectorSeparateBoundAndDroppedRollback(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("Code"); err != nil {
		t.Fatal(err)
	}
	emit := func(reader *Reader) {
		t.Helper()
		before := *reader.steps
		if err := reader.Observe(0, 1, 1, Source{}); err != nil {
			t.Fatal(err)
		}
		if *reader.steps != before {
			t.Fatal("observation changed decoder budget")
		}
	}
	for i := 0; i < 8; i++ {
		emit(r)
	}
	if len(c.observations) != 8 || c.dropped != 0 {
		t.Fatalf("at bound: %+v", c.TakeReport())
	}
	failed := r.Fork()
	emit(failed)
	nested := failed.Fork()
	emit(nested)
	failed.Commit(nested)
	failed.Discard()
	if c.dropped != 0 {
		t.Fatal("abandoned dropped count retained")
	}
	winner := r.Fork()
	emit(winner)
	r.Commit(winner)
	r.Commit(winner)
	winner.Discard()
	failed = r.Fork()
	emit(failed)
	failed.Discard()
	report := c.TakeReport()
	if len(report.Observations) != 8 || report.Dropped != 1 {
		t.Fatalf("after bound: %+v", report)
	}
	emit(r)
	next := c.TakeReport()
	if len(next.Observations) != 1 || next.Dropped != 0 {
		t.Fatalf("report reset: %+v", next)
	}
	if report.Dropped != 1 {
		t.Fatal("returned report changed")
	}
}

func TestCollectorPathBoundNeverFailsDecode(t *testing.T) {
	for _, length := range []int{1, 1022, 1023, 1024, maxBits, maxBits + 1} {
		var c Collector
		r := NewReaderWithCollector([]byte{0xff}, &c)
		if err := r.Enter(strings.Repeat("x", length)); err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(1); err != nil {
			t.Fatal(err)
		}
		before := *r.steps
		if err := r.Observe(0, 1, 1, Source{}); err != nil {
			t.Fatal(err)
		}
		report := c.TakeReport()
		want := uint64(0)
		if length >= 1024 {
			want = 1
		}
		if report.Dropped != want || len(report.Observations) != 1-int(want) || *r.steps != before {
			t.Fatalf("length=%d events=%d dropped=%d", length, len(report.Observations), report.Dropped)
		}
		if err := r.Check(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCollectorIndexStorageIsLazy(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if c.indices != nil {
		t.Fatal("non-indexed decode allocated index storage")
	}
	for i := 0; i < 129; i++ {
		if err := r.EnterIndex("Entries", i); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Observe(0, 1, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	if report := c.TakeReport(); report.Dropped != 1 {
		t.Fatal("unrepresentable indexed path not reported")
	}
	for i := 0; i < 129; i++ {
		r.LeaveIndex()
	}
	if len(c.indices) != 0 || cap(c.indices) > 128 {
		t.Fatalf("index storage len=%d cap=%d", len(c.indices), cap(c.indices))
	}
	for _, entry := range c.indices[:cap(c.indices)] {
		if !reflect.ValueOf(entry).IsZero() {
			t.Fatal("indexed path retained after leave")
		}
	}
	if err := r.EnterIndex("Entries", -1); err != nil {
		t.Fatal(err)
	}
	if err := r.Observe(0, 1, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	if report := c.TakeReport(); report.Dropped != 1 {
		t.Fatal("invalid indexed path not reported")
	}
	r.LeaveIndex()
}

func TestCollectorFailedAttemptsConsumeOnlyCollectionBudget(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000; i++ {
		arm := r.Fork()
		if err := arm.Observe(0, 1, 1, Source{}, "Code"); err != nil {
			t.Fatal(err)
		}
		arm.Discard()
	}
	if len(c.observations) != 0 || c.dropped != 0 || cap(c.observations) > 1 {
		t.Fatal("abandoned state retained")
	}
	if err := r.Observe(0, 1, 1, Source{}, "Code"); err != nil {
		t.Fatal(err)
	}
	report := c.TakeReport()
	if len(report.Observations) != 0 || report.Dropped != 1 {
		t.Fatalf("visits refunded: %+v", report)
	}
	if err := r.Check(); err != nil {
		t.Fatal(err)
	}
}

func TestCollectorPathScopePreservesDecoderState(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	beforeSteps, beforeDepth, beforePath := *r.steps, r.depth, append([]string(nil), r.path...)
	r.EnterObservationPath("Container")
	if err := r.EnterIndex("Entries", 2); err != nil {
		t.Fatal(err)
	}
	if err := r.Observe(0, 1, 1, Source{}, "Code"); err != nil {
		t.Fatal(err)
	}
	r.LeaveIndex()
	r.LeaveObservationPath()
	if *r.steps != beforeSteps || r.depth != beforeDepth || !slices.Equal(r.path, beforePath) {
		t.Fatal("observation scope changed decoder state")
	}
	report := c.TakeReport()
	if report.Dropped != 0 || len(report.Observations) != 1 || report.Observations[0].Location.Path != "Container/Entries[2]/Code" {
		t.Fatalf("report=%+v", report)
	}
}

func TestCollectorPathBudgetAndDroppedRollback(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	failed := r.Fork()
	if err := failed.Observe(0, 1, 1, Source{}, strings.Repeat("x", 1000)); err != nil {
		t.Fatal(err)
	}
	failed.Discard()
	failed = r.Fork()
	if err := failed.Observe(0, 1, 1, Source{}, strings.Repeat("x", 23)); err != nil {
		t.Fatal(err)
	}
	if c.dropped != 1 {
		t.Fatal("materialization budget was refunded")
	}
	failed.Discard()
	if err := r.Observe(0, 1, 1, Source{}, strings.Repeat("x", 22)); err != nil {
		t.Fatal(err)
	}
	report := c.TakeReport()
	if report.Dropped != 0 || len(report.Observations) != 1 {
		t.Fatalf("at path bound: %+v", report)
	}
	if c.pathUsed != 0 {
		t.Fatal("TakeReport did not reset path accounting")
	}
}

func TestCollectorErrorRetainsCommittedDrops(t *testing.T) {
	descriptor := Descriptor{DecodeWithCollector: func(data []byte, c *Collector) (any, error) {
		r := NewReaderWithCollector(data, c)
		if _, err := r.ReadUint(1); err != nil {
			return nil, err
		}
		for i := 0; i < 9; i++ {
			if err := r.Observe(0, 1, 1, Source{}, "Code"); err != nil {
				return nil, err
			}
		}
		failed := r.Fork()
		if err := failed.Observe(0, 1, 1, Source{}, "Code"); err != nil {
			return nil, err
		}
		failed.Discard()
		return nil, r.Error(Truncated, "synthetic failure")
	}}
	_, report, err := descriptor.DecodeWithReport([]byte{0xff})
	if err == nil || len(report.Observations) != 8 || report.Dropped != 1 {
		t.Fatalf("events=%d dropped=%d error=%v", len(report.Observations), report.Dropped, err)
	}
}
