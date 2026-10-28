package runtime

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestChoiceDeferredErrorRetainsDeepestFailure(t *testing.T) {
	r := NewReader(nil)
	var failures ChoiceErrors
	defer failures.Release()
	first := failures.Fork(r)
	if err := first.Enter("First"); err != nil {
		t.Fatal(err)
	}
	a := first.ErrorAt(Truncated, 2, "first")
	first.Leave()
	failures.Track(a)
	second := failures.Fork(r)
	if err := second.Enter("Second"); err != nil {
		t.Fatal(err)
	}
	b := second.ErrorAt(Truncated, 7, "deepest")
	second.Leave()
	failures.Track(b)
	err := failures.Keep(r, PreferTruncation(a, b))
	var de *DecodeError
	if !errors.As(err, &de) || de.Kind != Truncated || de.Offset != 7 || de.Path != "Second" || de.Detail != "deepest" {
		t.Fatalf("wrong selected failure: %#v", err)
	}
}

func TestChoiceBindingsAreIsolatedAndCommitted(t *testing.T) {
	r := NewReader(nil)
	var failures ChoiceErrors
	defer failures.Release()
	r.Set("A B", 3)
	first := failures.Fork(r)
	first.Set("A B", 9)
	first.Set("Other", 7)
	second := failures.Fork(r)
	if got, err := second.Eval("val(A B)"); err != nil || got != 3 {
		t.Fatalf("leaked binding: %d %v", got, err)
	}
	if _, err := second.Eval("Other"); err == nil {
		t.Fatal("leaked new binding")
	}
	nested := failures.Fork(first)
	nested.Set("A B", 11)
	first.Commit(nested)
	if got, err := first.Eval("AB"); err != nil || got != 11 {
		t.Fatalf("nested binding: %d %v", got, err)
	}
	r.Commit(first)
	if got, err := r.Eval("AB"); err != nil || got != 11 {
		t.Fatalf("lost chosen binding: %d %v", got, err)
	}
	if got, err := r.Eval("Other"); err != nil || got != 7 {
		t.Fatalf("lost new binding: %d %v", got, err)
	}
}

func TestChoiceBorrowedPathCannotOverwriteRetainedFailure(t *testing.T) {
	r := NewReader(nil)
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	var failures ChoiceErrors
	defer failures.Release()
	chosen := failures.Fork(r)
	if err := chosen.Enter("Chosen"); err != nil {
		t.Fatal(err)
	}
	err := chosen.Error(Truncated, "retained")
	failures.Track(err)
	chosen.Leave()
	later := failures.Fork(r)
	if enterErr := later.Enter("Later"); enterErr != nil {
		t.Fatal(enterErr)
	}
	later.Leave()
	actual := failures.Keep(r, err)
	var de *DecodeError
	if !errors.As(actual, &de) || de.Path != "Root/Chosen" {
		t.Fatalf("overwritten path: %v", actual)
	}
	if *r.steps != 3 {
		t.Fatalf("rejected work lost: %d", *r.steps)
	}
	if !reflect.DeepEqual(chosen.path, []string{"Root"}) {
		t.Fatalf("chosen prefix changed: %v", chosen.path)
	}
}

func TestChoiceRejectedFailureAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("pool entries can be dropped under race instrumentation")
	}
	r := NewReader(nil)
	var owner ChoiceErrors
	defer owner.Release()
	r.Set("count", 2)
	// Keep the reader construction outside the measurement: this measures the
	// rejected diagnostic and binding writes, not a retained trial reader.
	candidate := owner.Fork(r)
	allocs := testing.AllocsPerRun(100, func() {
		var failures ChoiceErrors
		candidate.Set("count", 3)
		failures.Track(candidate.Error(InvalidValue, "rejected"))
		failures.Release()
	})
	if allocs != 0 {
		t.Fatalf("rejected failure allocated: %g", allocs)
	}
}

func TestChoiceBindingsMatchMapResolution(t *testing.T) {
	r := NewReader(nil)
	for i := range 24 {
		r.Set(fmt.Sprintf("base %d", i), uint64(i))
	}
	var failures ChoiceErrors
	defer failures.Release()
	a := failures.Fork(r)
	for i := range 24 {
		a.Set(fmt.Sprintf("local %d", i), uint64(i+1))
	}
	a.Set("a b", 7)
	a.Set("ab", 8)
	b := failures.Fork(a)
	b.Set("local 15", 91)
	b.Set("a b", 9)
	for _, expression := range []string{"base 15", "local 15", "ab", "a b", "BASE15", "local23"} {
		vars := cloneVars(r.vars)
		for i := range 24 {
			vars[fmt.Sprintf("local %d", i)] = uint64(i + 1)
		}
		vars["local 15"], vars["a b"], vars["ab"] = 91, 9, 8
		want, we := eval(expression, vars)
		got, ge := b.Eval(expression)
		failures.Track(ge)
		if got != want || (ge == nil) != (we == nil) || ge != nil && !strings.Contains(ge.Error(), we.Error()) {
			t.Fatalf("%s: %d %v; want %d %v", expression, got, ge, want, we)
		}
	}
	a.Commit(b)
	r.Commit(a)
	if got, err := r.Eval("local15"); err != nil || got != 91 {
		t.Fatalf("overflow overlay commit: %d %v", got, err)
	}
	// Compact ambiguity must count distinct visible names once, including
	// overwritten names inherited through several trial levels.
	r.Set("a  bc", 1)
	r.Set("ab c", 2)
	c := failures.Fork(r)
	c.Set("a bc", 3)
	_, err := c.Eval("abc")
	failures.Track(err)
	if err == nil {
		t.Fatal("compact ambiguity accepted")
	}
}

func normalizeChoiceTest(err error, operation, path string, offset, width int, source Source, kind ErrorKind) error {
	return &OperationError{Operation: operation, Kind: kind, Location: Location{Path: path, BitOffset: offset, Width: width, Source: source}, Cause: err}
}

func TestChoiceDiagnosticNormalizationPreservesChains(t *testing.T) {
	r := NewReader(nil)
	var failures ChoiceErrors
	defer failures.Release()
	candidate := failures.Fork(r)
	if err := candidate.Enter("Kept"); err != nil {
		t.Fatal(err)
	}
	candidate.fieldWidthLo = 4
	err := candidate.ErrorAt(Truncated, 5, "missing")
	source := Source{Standard: "TS 24.007", Version: "20.0.0", Clause: "B.1.1.4"}
	err, ok := DeferOperation(err, "decode", "leaf", 5, 3, source, Truncated, normalizeChoiceTest)
	if !ok {
		t.Fatal("trial diagnostic was materialized")
	}
	failures.Track(err)
	candidate.Leave()
	got := failures.Keep(r, err)
	want := normalizeChoiceTest(&fieldDecodeError{DecodeError: DecodeError{Kind: Truncated, Offset: 5, Path: "Kept", Detail: "missing"}, width: 3}, "decode", "leaf", 5, 3, source, Truncated)
	if !equalChoiceDiagnostic(got, want) {
		t.Fatalf("diagnostic changed: %#v / %#v", got, want)
	}
	for range 100 {
		var discarded ChoiceErrors
		other := discarded.Fork(r)
		discarded.Track(other.Error(InvalidValue, "replacement"))
		discarded.Release()
	}
	if !equalChoiceDiagnostic(got, want) {
		t.Fatal("returned error changed after scratch reuse")
	}
}

func TestChoiceRejectedObservationAndWorkStayTransactional(t *testing.T) {
	var collector Collector
	r := NewReaderWithCollector([]byte{0}, &collector)
	var trials ChoiceErrors
	defer trials.Release()
	for _, chosen := range []bool{false, true, false} {
		candidate := trials.Fork(r)
		if err := candidate.Enter("Value"); err != nil {
			t.Fatal(err)
		}
		if _, err := candidate.ReadUint(1); err != nil {
			t.Fatal(err)
		}
		if err := candidate.Observe(0, 1, 0, Source{}); err != nil {
			t.Fatal(err)
		}
		candidate.Leave()
		if chosen {
			r.Commit(candidate)
		} else {
			candidate.Discard()
		}
	}
	report := collector.TakeReport()
	if len(report.Observations) != 1 || report.Dropped != 0 || report.Observations[0].Location.Path != "Value" {
		t.Fatalf("changed observations: %#v", report)
	}
	if *r.steps != 3 {
		t.Fatalf("rejected work was refunded: %d", *r.steps)
	}
}

func TestPublicForkOfTrialDetachesPooledBindings(t *testing.T) {
	r := NewReader(nil)
	r.Set("before", 1)
	var trials ChoiceErrors
	a := trials.Fork(r)
	a.Set("local", 3)
	b := trials.Fork(a)
	b.Set("nested", 4)
	snapshot := b.Fork()
	r.Set("before", 2)
	a.Set("local", 5)
	b.Set("nested", 6)
	trials.Release()
	for name, want := range map[string]int{"before": 1, "local": 3, "nested": 4} {
		got, err := snapshot.Eval(name)
		if err != nil || got != want {
			t.Fatalf("%s: got %d %v; want %d", name, got, err, want)
		}
	}
	r.Commit(snapshot)
	if got, err := r.Eval("before"); err != nil || got != 1 {
		t.Fatalf("public snapshot commit: %d %v", got, err)
	}
}

func TestDeferredPreferTruncationUsesExistingDecodeOffset(t *testing.T) {
	r := NewReader(nil)
	var trials ChoiceErrors
	defer trials.Release()
	a, b := trials.Fork(r), trials.Fork(r)
	current := a.ErrorAt(InvalidValue, 7, "prior")
	candidate := b.ErrorAt(Truncated, 5, "shallower")
	trials.Track(current)
	trials.Track(candidate)
	if got := PreferTruncation(current, candidate); got != current {
		t.Fatal("changed existing DecodeError offset precedence")
	}
}

func TestPublicForkOfTrialOwnsItsActivePath(t *testing.T) {
	r := NewReader(nil)
	if err := r.Enter("Root"); err != nil {
		t.Fatal(err)
	}
	var trials ChoiceErrors
	a := trials.Fork(r)
	if err := a.Enter("Trial"); err != nil {
		t.Fatal(err)
	}
	snapshot := a.Fork()
	a.Leave()
	b := trials.Fork(r)
	if err := b.Enter("Later"); err != nil {
		t.Fatal(err)
	}
	b.Leave()
	trials.Release()
	err := snapshot.Error(Truncated, "missing")
	var de *DecodeError
	if !errors.As(err, &de) || de.Path != "Root/Trial" {
		t.Fatalf("public fork path overwritten: %v", err)
	}
}

// The independently constructed chains have distinct identities. Compare
// their exact concrete leaf representation and every operation field.
func equalChoiceDiagnostic(got, want error) bool {
	left, lok := got.(*OperationError)
	right, rok := want.(*OperationError)
	if !lok || !rok {
		return false
	}
	leftLeaf, lok := left.Cause.(*fieldDecodeError)
	rightLeaf, rok := right.Cause.(*fieldDecodeError)
	if !lok || !rok {
		return false
	}
	a, b := *left, *right
	a.Cause, b.Cause = nil, nil
	return a == b && *leftLeaf == *rightLeaf
}

func TestChoiceCompiledBindingsMatchInterpreter(t *testing.T) {
	r := NewReader(nil)
	r.Set("a b", 3)
	var trials ChoiceErrors
	defer trials.Release()
	candidate := trials.Fork(r)
	candidate.Set("a b", 5)
	candidate.Set("ab", 7)
	candidate.Set("a bc", 9)
	candidate.Set("ab c", 11)
	for _, name := range []string{"A B", "ab", "ABC", "absent"} {
		interpreted, ie := candidate.Eval("val(" + name + ")")
		compiled, ce := candidate.ExpressionResult(candidate.ExpressionVariable(name).Operand())
		trials.Track(ie)
		trials.Track(ce)
		if interpreted != compiled || (ie == nil) != (ce == nil) {
			t.Fatalf("%s: interpreter %d %v; compiled %d %v", name, interpreted, ie, compiled, ce)
		}
		if ie == nil {
			continue
		}
		ie, ce = trials.Keep(r, ie), trials.Keep(r, ce)
		var a, b *DecodeError
		if !errors.As(ie, &a) || !errors.As(ce, &b) || *a != *b {
			t.Fatalf("%s: interpreter %v; compiled %v", name, ie, ce)
		}
	}
}
