package runtime

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

var pointerSource = Source{Standard: "TS 44.060", Version: "19.0.0", Clause: "11.2.21", Table: "p17216"}

func pointerAggregate(t testing.TB, scope string) (Aggregate, *ExpressionContext) {
	t.Helper()
	a, err := NewAggregate(scope, "CELL_PARAMS_POINTER", pointerSource)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := NewExpressionContext(nil, []Aggregate{a})
	if err != nil {
		t.Fatal(err)
	}
	return a, ctx
}

func TestPointerMaximumScope(t *testing.T) {
	a, ctx := pointerAggregate(t, "PSI3/NeighbourCellList")
	r, w := NewReader(nil), NewWriter()
	for _, tc := range []struct {
		values []uint64
		want   int
	}{{[]uint64{3, 1}, 4}, {[]uint64{0, 2}, 3}, {[]uint64{0}, 1}, {[]uint64{4, 4, 2}, 5}} {
		if e := r.ResetAggregate(a); e != nil {
			t.Fatal(e)
		}
		if e := w.ResetAggregate(a); e != nil {
			t.Fatal(e)
		}
		for _, v := range tc.values {
			if e := r.RecordAggregate(a, v); e != nil {
				t.Fatal(e)
			}
			if e := w.RecordAggregate(a, v); e != nil {
				t.Fatal(e)
			}
			// The scalar binding intentionally contains the last value.
			r.Set("CELL_PARAMS_POINTER", v)
			w.Set("CELL_PARAMS_POINTER", v)
		}
		raw := "1+max(val(CELL_PARAMS_POINTER))"
		n, e := r.EvalWithContext(raw, ctx)
		if n != tc.want || e != nil {
			t.Fatalf("reader %v = %d %v want %d", tc.values, n, e, tc.want)
		}
		n, e = w.EvalWithContext(raw, ctx)
		if n != tc.want || e != nil {
			t.Fatalf("writer %v = %d %v want %d", tc.values, n, e, tc.want)
		}
		for _, v := range []ExpressionValue{r.ExpressionMaximum(a).Add(ExpressionConstant(1)), w.ExpressionMaximum(a).Add(ExpressionConstant(1))} {
			n, e = v.width()
			if n != tc.want || e != nil {
				t.Errorf("compiled %v = %d %v", tc.values, n, e)
			}
		}
		for _, v := range []ExpressionValue{r.ExpressionAggregateCount(a), w.ExpressionAggregateCount(a)} {
			n, e = v.width()
			if n != len(tc.values) || e != nil {
				t.Errorf("count %d %v", n, e)
			}
		}
	}
}

func TestAggregateNestedAndChoiceOwnership(t *testing.T) {
	a, ctx := pointerAggregate(t, "outer")
	b, innerCtx := pointerAggregate(t, "outer/inner")
	r := NewReader(nil)
	if e := r.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	if e := r.RecordAggregate(a, 3); e != nil {
		t.Fatal(e)
	}
	if e := r.ResetAggregate(b); e != nil {
		t.Fatal(e)
	}
	if e := r.RecordAggregate(b, 1); e != nil {
		t.Fatal(e)
	}
	check := func(r *Reader, ctx *ExpressionContext, want int) {
		t.Helper()
		n, e := r.EvalWithContext("1+max(val(CELL_PARAMS_POINTER))", ctx)
		if n != want || e != nil {
			t.Fatalf("scope got %d %v want %d", n, e, want)
		}
	}
	check(r, ctx, 4)
	check(r, innerCtx, 2)
	fork := r.Fork()
	if e := fork.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	if e := fork.RecordAggregate(a, 0); e != nil {
		t.Fatal(e)
	}
	check(r, ctx, 4)
	check(fork, ctx, 1)
	var choices ChoiceErrors
	trial := choices.ForkScoped(r)
	if e := trial.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	if e := trial.RecordAggregate(a, 7); e != nil {
		t.Fatal(e)
	}
	var nested ChoiceErrors
	nestedTrial := nested.ForkScoped(trial)
	if e := nestedTrial.RecordAggregate(a, 9); e != nil {
		t.Fatal(e)
	}
	trial.Commit(nestedTrial)
	nested.Release()
	check(trial, ctx, 10)
	snapshot := trial.Fork()
	choices.Release()
	check(r, ctx, 4)
	check(snapshot, ctx, 10)
	var next ChoiceErrors
	trial = next.ForkScoped(r)
	check(trial, ctx, 4)
	if e := trial.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	if e := trial.RecordAggregate(a, 2); e != nil {
		t.Fatal(e)
	}
	r.Commit(trial)
	next.Release()
	check(r, ctx, 3)
	check(fork, ctx, 1)
	check(snapshot, ctx, 10)
	check(r, innerCtx, 2)
}

func TestAggregateFailures(t *testing.T) {
	a, ctx := pointerAggregate(t, "scope")
	r, w := NewReader([]byte{0}), NewWriter()
	if e := r.Enter("list"); e != nil {
		t.Fatal(e)
	}
	if e := w.Enter("list"); e != nil {
		t.Fatal(e)
	}
	_, e := r.ReadUint(2)
	if e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(0, 2); e != nil {
		t.Fatal(e)
	}
	check := func(want string) {
		t.Helper()
		_, re := r.EvalWithContext("max(val(CELL_PARAMS_POINTER))", ctx)
		_, rc := r.ExpressionResultWithContext(r.ExpressionMaximum(a))
		_, we := w.EvalWithContext("max(val(CELL_PARAMS_POINTER))", ctx)
		_, wc := w.ExpressionResultWithContext(w.ExpressionMaximum(a))
		for _, err := range []error{re, rc, we, wc} {
			var oe *OperationError
			if err == nil || !strings.Contains(err.Error(), want) || !errors.As(err, &oe) || oe.Location.Path != "list" || oe.Location.BitOffset != 2 || oe.Location.Source != pointerSource {
				t.Fatalf("%s: %+v", want, err)
			}
		}
		if re.Error() != rc.Error() || we.Error() != wc.Error() {
			t.Fatal("aggregate error parity")
		}
	}
	check("not been reset")
	if err := r.RecordAggregate(a, 0); !errors.Is(err, ErrContextRequired) {
		t.Errorf("uninitialized %v", err)
	}
	if err := w.RecordAggregate(a, 0); !errors.Is(err, ErrContextRequired) {
		t.Errorf("uninitialized writer %v", err)
	}
	if e := r.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	if e := w.ResetAggregate(a); e != nil {
		t.Fatal(e)
	}
	check("p17216")
	for _, v := range []uint64{maxBits + 1, math.MaxUint64} {
		if e := r.RecordAggregate(a, v); e != nil {
			t.Fatal(e)
		}
		if e := w.RecordAggregate(a, v); e != nil {
			t.Fatal(e)
		}
		check("outside limits")
	}
	r.Set(a.count, maxBits)
	w.Set(a.count, maxBits)
	if e := r.RecordAggregate(a, 0); e == nil {
		t.Fatal("count overflow")
	}
	if e := w.RecordAggregate(a, 0); e == nil {
		t.Fatal("writer count overflow")
	}
	// Failed records do not partially update the counters.
	if got, _ := r.binding(a.count); got != maxBits {
		t.Fatal("partial count write")
	}
	r.Set(a.count, 0)
	w.Set(a.count, 0)
	r.Set(a.population, maxBits)
	w.Set(a.population, maxBits)
	if e := r.RecordAggregate(a, 1); e == nil {
		t.Fatal("population overflow")
	}
	if e := w.RecordAggregate(a, 1); e == nil {
		t.Fatal("writer population overflow")
	}
	if e := r.ResetAggregate(Aggregate{}); e == nil {
		t.Fatal("zero aggregate")
	}
	if e := w.ResetAggregate(Aggregate{}); e == nil {
		t.Fatal("zero writer aggregate")
	}
	if _, e := r.EvalWithContext("max(val(CELL_PARAMS_POINTER))", nil); !errors.Is(e, ErrContextRequired) {
		t.Errorf("missing scope %v", e)
	}
	for _, v := range []ExpressionValue{r.ExpressionMaximum(Aggregate{}), w.ExpressionMaximum(Aggregate{}), r.ExpressionAggregateCount(Aggregate{}), w.ExpressionAggregatePopulation(Aggregate{})} {
		if _, e := v.width(); !errors.Is(e, ErrContextRequired) {
			t.Errorf("zero identity %v", e)
		}
	}
}

func TestTimeslotAggregatePopulations(t *testing.T) {
	// TS 44.060 §11.2.29a pp18719–18729 and §12.42 pp23778–23792:
	// N counts global allocations, M counts the current TBF subset. Carriers
	// and RTTI PDCH-pair bitmaps contribute one count per set allocation bit.
	global, _ := pointerAggregate(t, "global")
	tbf, _ := pointerAggregate(t, "TBF")
	for _, masks := range [][]uint64{{0x81}, {0xff}, {0x81, 0x24}, {0x05, 0x80}, {0}, {1 << 63}, {math.MaxUint64}} {
		r, w := NewReader(nil), NewWriter()
		if e := r.ResetAggregate(global); e != nil {
			t.Fatal(e)
		}
		if e := w.ResetAggregate(global); e != nil {
			t.Fatal(e)
		}
		want := 0
		for _, mask := range masks {
			if e := r.RecordAggregate(global, mask); e != nil {
				t.Fatal(e)
			}
			if e := w.RecordAggregate(global, mask); e != nil {
				t.Fatal(e)
			}
			for b := range 64 {
				if mask&(uint64(1)<<uint(b)) != 0 {
					want++
				}
			}
		}
		for _, v := range []ExpressionValue{r.ExpressionAggregatePopulation(global), w.ExpressionAggregatePopulation(global)} {
			if n, e := v.width(); n != want || e != nil {
				t.Fatalf("mask %v got %d %v want %d", masks, n, e, want)
			}
		}
		for _, selected := range []uint64{3, 1, 0, 0x80} {
			if e := r.ResetAggregate(tbf); e != nil {
				t.Fatal(e)
			}
			if e := w.ResetAggregate(tbf); e != nil {
				t.Fatal(e)
			}
			if e := r.RecordAggregate(tbf, selected); e != nil {
				t.Fatal(e)
			}
			if e := w.RecordAggregate(tbf, selected); e != nil {
				t.Fatal(e)
			}
			m := 1
			switch selected {
			case 3:
				m = 2
			case 0:
				m = 0
			}
			if n, e := r.ExpressionAggregatePopulation(tbf).width(); n != m || e != nil {
				t.Fatalf("selected %d got %d %v", selected, n, e)
			}
			if n, e := w.ExpressionAggregatePopulation(tbf).width(); n != m || e != nil {
				t.Fatal(n, e)
			}
		}
	}
}

func TestAggregateIdentityValidation(t *testing.T) {
	for _, pair := range [][2]string{{"", "x"}, {"scope", ""}, {"scope", " "}, {"a\x00b", "c"}, {"a", "b\x00c"}} {
		if _, e := NewAggregate(pair[0], pair[1], pointerSource); e == nil {
			t.Errorf("accepted %q", pair)
		}
	}
	if _, e := NewAggregate("a", "b", Source{}); e == nil {
		t.Fatal("missing source")
	}
	a, _ := pointerAggregate(t, "outer")
	b, _ := pointerAggregate(t, "inner")
	if _, e := NewExpressionContext(nil, []Aggregate{a, b}); e == nil {
		t.Fatal("ambiguous scopes")
	}
	if _, e := NewExpressionContext(nil, []Aggregate{{}}); e == nil {
		t.Fatal("zero scope")
	}
}

func TestAggregateStorageBound(t *testing.T) {
	a, _ := pointerAggregate(t, "list")
	for _, records := range []int{16, 256, 4096, 65536} {
		r := NewReader(nil)
		if e := r.ResetAggregate(a); e != nil {
			t.Fatal(e)
		}
		allocs := testing.AllocsPerRun(3, func() {
			for range records {
				if e := r.RecordAggregate(a, 0); e != nil {
					panic(e)
				}
			}
		})
		if allocs != 0 || len(r.vars) != 3 {
			t.Errorf("%d records allocations=%g bindings=%d", records, allocs, len(r.vars))
		}
	}
}

func FuzzAggregateOwnership(f *testing.F) {
	f.Add([]byte{3, 1, 0, 2})
	f.Add([]byte{})
	f.Add([]byte{255, 0})
	f.Fuzz(func(t *testing.T, values []byte) {
		if len(values) > 4096 {
			t.Skip()
		}
		a, ctx := pointerAggregate(t, "list")
		r := NewReader(nil)
		if e := r.ResetAggregate(a); e != nil {
			t.Fatal(e)
		}
		maximum := byte(0)
		for _, v := range values {
			if e := r.RecordAggregate(a, uint64(v)); e != nil {
				t.Fatal(e)
			}
			maximum = max(maximum, v)
		}
		n, e := r.EvalWithContext("max(val(CELL_PARAMS_POINTER))", ctx)
		if len(values) == 0 {
			if e == nil {
				t.Fatal("empty maximum")
			}
		} else if n != int(maximum) || e != nil {
			t.Fatal(n, e, maximum)
		}
		var choices ChoiceErrors
		trial := choices.ForkScoped(r)
		if e := trial.ResetAggregate(a); e != nil {
			t.Fatal(e)
		}
		if e := trial.RecordAggregate(a, 256); e != nil {
			t.Fatal(e)
		}
		choices.Release()
		got, err := r.EvalWithContext("max(val(CELL_PARAMS_POINTER))", ctx)
		if got != n || fmt.Sprint(err) != fmt.Sprint(e) {
			t.Fatal("trial leaked")
		}
	})
}

func TestAggregateSourceIdentity(t *testing.T) {
	a, _ := pointerAggregate(t, "list")
	source := pointerSource
	source.Clause = "12.42"
	b, e := NewAggregate("list", "CELL_PARAMS_POINTER", source)
	if e != nil {
		t.Fatal(e)
	}
	r := NewReader(nil)
	for _, v := range []struct {
		a Aggregate
		n uint64
	}{{a, 3}, {b, 1}} {
		if e := r.ResetAggregate(v.a); e != nil {
			t.Fatal(e)
		}
		if e := r.RecordAggregate(v.a, v.n); e != nil {
			t.Fatal(e)
		}
	}
	if n, e := r.ExpressionMaximum(a).width(); n != 3 || e != nil {
		t.Fatal("source scopes alias", n, e)
	}
	if n, e := r.ExpressionMaximum(b).width(); n != 1 || e != nil {
		t.Fatal("source scopes alias", n, e)
	}
	source.Table = "p17216\x00list"
	if _, e := NewAggregate("list", "CELL_PARAMS_POINTER", source); e == nil {
		t.Fatal("NUL in source")
	}
}
