package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCollectorTransactions(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(4); err != nil {
		t.Fatal(err)
	}
	source := Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.7", Table: "10.5.1.7"}
	emit := func(r *Reader, value uint64) {
		t.Helper()
		if err := r.Observe(0, 4, value, source); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Enter("SingleBandSupport"); err != nil {
		t.Fatal(err)
	}
	emit(r, 10)
	failed := r.Fork()
	emit(failed, 11)
	nested := failed.Fork()
	emit(nested, 12)
	failed.Commit(nested)
	failed.Discard()
	winner := r.Fork()
	emit(winner, 15)
	r.Commit(winner)
	r.Commit(winner)
	winner.Discard()
	report := c.TakeReport()
	if len(report.Observations) != 2 || report.Observations[0].Value != 10 || report.Observations[1].Value != 15 {
		t.Fatalf("report: %+v", report)
	}
	if report.Observations[1].Location != (Location{Path: "SingleBandSupport", BitOffset: 0, Width: 4, Source: source}) {
		t.Fatalf("location: %+v", report.Observations[1])
	}
	snapshot := append([]Observation(nil), report.Observations...)
	emit(r, 13)
	if !reflect.DeepEqual(snapshot, report.Observations) {
		t.Fatal("TakeReport shares mutable storage")
	}
	if len(c.TakeReport().Observations) != 1 || len(c.TakeReport().Observations) != 0 {
		t.Fatal("TakeReport did not detach")
	}
}

func TestCollectorDiscardClearsPaths(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	f := r.Fork()
	if err := f.Enter(strings.Repeat("x", 1000)); err != nil {
		t.Fatal(err)
	}
	if err := f.Observe(0, 1, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	f.Discard()
	f.Discard()
	r.Discard()
	if len(c.observations) != 0 {
		t.Fatal("abandoned observations retained")
	}
	for _, o := range c.observations[:cap(c.observations)] {
		if o.Location.Path != "" {
			t.Fatal("abandoned path retained")
		}
	}
}

func TestCollectorNoEventAllocations(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0}, &c)
	if got := testing.AllocsPerRun(100, func() { _ = c.TakeReport(); r.Discard() }); got != 0 {
		t.Fatalf("empty collector: %v allocations", got)
	}
	if c.observations != nil {
		t.Fatal("empty buffer allocated")
	}
	disabled := NewReader(nil)
	if got := testing.AllocsPerRun(100, func() { _ = disabled.Observe(0, 1, 1, Source{}) }); got != 0 {
		t.Fatalf("disabled collector: %v allocations", got)
	}
}

func TestCollectorFailedArmsBounded(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xff}, &c)
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	r.steps = new(int)
	*r.steps = maxBits - 12
	for i := 0; i < 20; i++ {
		arm := r.Fork()
		if err := arm.Enter("Reserved"); err != nil {
			return
		}
		err := arm.Observe(0, 1, 1, Source{})
		arm.Discard()
		if len(c.observations) != 0 || cap(c.observations) > 1 {
			t.Fatalf("retained failed log: len=%d cap=%d", len(c.observations), cap(c.observations))
		}
		if err != nil {
			var de *DecodeError
			if !errors.As(err, &de) || de.Kind != Limit {
				t.Fatalf("limit: %v", err)
			}
			return
		}
	}
	t.Fatal("failed arms evaded shared work limit")
}

func TestCollectorIndexedPathsAndGrowth(t *testing.T) {
	for _, count := range []int{1, 8, 64, 512} {
		var c Collector
		r := NewReaderWithCollector(make([]byte, count), &c)
		if err := r.Enter("Root"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < count; i++ {
			if err := r.EnterIndex("Entries", i); err != nil {
				t.Fatal(err)
			}
			_, err := func(r *Reader) (uint64, error) {
				start := r.Position()
				v, err := r.ReadUint(8)
				if err != nil {
					return 0, err
				}
				if err := r.Enter("Code"); err != nil {
					return 0, err
				}
				defer r.Leave()
				return v, r.Observe(start, 8, v, Source{})
			}(r)
			r.LeaveIndex()
			if err != nil {
				t.Fatal(err)
			}
		}
		if len(c.observations) != count || cap(c.observations) > 2*count+8 {
			t.Fatalf("count=%d len=%d cap=%d", count, len(c.observations), cap(c.observations))
		}
		if c.observations[count-1].Location.Path != fmt.Sprintf("Root/Entries[%d]/Code", count-1) {
			t.Fatalf("path: %s", c.observations[count-1].Location.Path)
		}
		if *r.steps > count*64+1 {
			t.Fatalf("nonlinear work: count=%d steps=%d", count, *r.steps)
		}
		allocations := testing.AllocsPerRun(3, func() {
			var collector Collector
			reader := NewReaderWithCollector(make([]byte, count), &collector)
			for i := 0; i < count; i++ {
				_, _ = reader.ReadUint(8)
				if err := reader.Observe(i*8, 8, 1, Source{}); err != nil {
					panic(err)
				}
			}
			_ = collector.TakeReport()
		})
		if allocations > float64(count+32) {
			t.Fatalf("count=%d allocations=%g", count, allocations)
		}
	}
}

func TestDescriptorReportsCommittedPrefix(t *testing.T) {
	source := Source{Standard: "TS 24.008", Version: "20.1.0", Table: "10.5.1.7"}
	d := Descriptor{DecodeWithCollector: func(data []byte, c *Collector) (any, error) {
		r := NewReaderWithCollector(data, c)
		if _, err := r.ReadUint(4); err != nil {
			return nil, err
		}
		if err := r.Enter("Committed"); err != nil {
			return nil, err
		}
		if err := r.Observe(0, 4, 15, source); err != nil {
			return nil, err
		}
		arm := r.Fork()
		if err := arm.Enter("Abandoned"); err != nil {
			return nil, err
		}
		if err := arm.Observe(0, 4, 15, source); err != nil {
			return nil, err
		}
		arm.Discard()
		return nil, r.Error(Truncated, "synthetic failure")
	}}
	_, report, err := d.DecodeWithReport([]byte{0xf0})
	var de *DecodeError
	if !errors.As(err, &de) || de.Kind != Truncated || len(report.Observations) != 1 || report.Observations[0].Location.Path != "Committed" {
		t.Fatalf("report=%+v error=%v", report, err)
	}
}

func TestCollectorMaterializesOnePathPerEvent(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xf0}, &c)
	if _, err := r.ReadUint(4); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("ReservedCode"); err != nil {
		t.Fatal(err)
	}
	c.observations = make([]Observation, 0, 1)
	allocations := testing.AllocsPerRun(100, func() {
		c.observations = c.observations[:0]
		c.visits, c.pathUsed = 0, 0
		if err := r.Observe(0, 4, 15, Source{}); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 1 {
		t.Fatalf("materialized path allocations=%g, want 1", allocations)
	}
	before := &c.observations[0]
	report := c.TakeReport()
	if &report.Observations[0] != before {
		t.Fatal("TakeReport copied observation storage")
	}
}

func TestCollectorBorrowedLeaf(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector([]byte{0xf0}, &c)
	if _, err := r.ReadUint(4); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	c.observations = make([]Observation, 0, 1)
	allocations := testing.AllocsPerRun(100, func() {
		c.observations = c.observations[:0]
		c.visits, c.pathUsed = 0, 0
		if err := r.Observe(0, 4, 15, Source{}, "Leaf"); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 1 {
		t.Fatalf("borrowed leaf allocated %g times, want one owned path", allocations)
	}
	if c.observations[0].Location.Path != "Root/Leaf" || len(r.path) != 1 || r.depth != 1 {
		t.Fatalf("borrowed path changed reader: %+v", c.observations)
	}
}

func TestCollectorDoesNotChargeDecodeWork(t *testing.T) {
	var c Collector
	r := NewReaderWithCollector(make([]byte, 16384), &c)
	for i := 0; i < 32768; i++ {
		if err := r.Enter("SingleBandSupport"); err != nil {
			t.Fatal(err)
		}
		if err := r.Enter("SingleBandSupportGSMBand"); err != nil {
			t.Fatal(err)
		}
		start := r.Position()
		v, err := r.ReadUint(4)
		if err != nil {
			t.Fatal(err)
		}
		before := *r.steps
		if err := r.Observe(start, 4, v, Source{}); err != nil {
			t.Fatal(err)
		}
		if *r.steps != before {
			t.Fatal("collection charged decoder work")
		}
		r.Leave()
		r.Leave()
	}
	if len(c.TakeReport().Observations) != 32768 {
		t.Fatal("large decode lost events")
	}
}
