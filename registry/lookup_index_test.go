package registry

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// linearLookup retains the scan semantics independently of the lookup indexes.
func linearLookup(entries []runtime.Descriptor, standard, clause, name string, qualified bool) (runtime.Descriptor, error) {
	if err := validateVersions(entries); err != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, err)
	}
	var matches []runtime.Descriptor
	for _, d := range entries {
		if d.Standard == standard && d.Name == name && (!qualified || d.Clause == clause) {
			if qualified {
				return d, nil
			}
			matches = append(matches, d)
		}
	}
	if qualified {
		return runtime.Descriptor{}, lookupFailure(standard, clause, name, fmt.Errorf("unknown CSN.1 definition %q in %s §%s", name, standard, clause))
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, fmt.Errorf("unknown CSN.1 definition %q in %s", name, standard))
	}
	clauses := make([]string, 0, len(matches))
	for _, d := range matches {
		clauses = append(clauses, d.Clause)
	}
	sort.Strings(clauses)
	return runtime.Descriptor{}, lookupFailure(standard, "", name, fmt.Errorf("ambiguous CSN.1 definition %q in %s; clauses: %s", name, standard, strings.Join(clauses, ", ")))
}

func assertLookupEqual(t *testing.T, got, want runtime.Descriptor, gotErr, wantErr error) {
	t.Helper()
	// Compare every descriptor field, including each callable's identity.
	g, w := reflect.ValueOf(got), reflect.ValueOf(want)
	for i := 0; i < g.NumField(); i++ {
		gf, wf := g.Field(i), w.Field(i)
		if gf.Kind() == reflect.Func {
			if gf.Pointer() != wf.Pointer() {
				t.Fatalf("descriptor function %s differs", g.Type().Field(i).Name)
			}
		} else if !reflect.DeepEqual(gf.Interface(), wf.Interface()) {
			t.Fatalf("descriptor field %s differs: %v / %v", g.Type().Field(i).Name, gf.Interface(), wf.Interface())
		}
	}
	assertLookupErrorEqual(t, gotErr, wantErr)
}

func assertLookupErrorEqual(t *testing.T, got, want error) {
	t.Helper()
	for got != nil && want != nil {
		if reflect.TypeOf(got) != reflect.TypeOf(want) || got.Error() != want.Error() {
			t.Fatalf("error differs: %T %v / %T %v", got, got, want, want)
		}
		if g, ok := got.(*runtime.OperationError); ok {
			w := want.(*runtime.OperationError)
			if g.Operation != w.Operation || g.Kind != w.Kind || g.Location != w.Location || g.Detail != w.Detail {
				t.Fatalf("operation error fields differ: %+v / %+v", g, w)
			}
			if !errors.Is(got, g.Cause) {
				t.Fatal("cause identity lost")
			}
		}
		got, want = errors.Unwrap(got), errors.Unwrap(want)
	}
	if got != nil || want != nil {
		t.Fatalf("error chain lengths differ: %v / %v", got, want)
	}
}

func TestLookupCatalogueParity(t *testing.T) {
	entries := Descriptors()
	names := map[[2]string]bool{}
	for _, d := range entries {
		got, ge := LookupClause(d.Standard, d.Clause, d.Name)
		want, we := linearLookup(entries, d.Standard, d.Clause, d.Name, true)
		assertLookupEqual(t, got, want, ge, we)
		names[[2]string{d.Standard, d.Name}] = true
	}
	ambiguous := 0
	for key := range names {
		got, ge := Lookup(key[0], key[1])
		want, we := linearLookup(entries, key[0], "", key[1], false)
		assertLookupEqual(t, got, want, ge, we)
		if we != nil {
			ambiguous++
		}
	}
	unknowns := 0
	for _, d := range entries {
		for _, key := range [][3]string{
			{d.Standard, d.Clause, "missing"},
			{"missing", d.Clause, d.Name},
			{d.Standard, "missing", d.Name},
			{d.Standard, d.Clause, d.Name + "\x00"},
		} {
			got, ge := LookupClause(key[0], key[1], key[2])
			want, we := linearLookup(entries, key[0], key[1], key[2], true)
			assertLookupEqual(t, got, want, ge, we)
			unknowns++
		}
		for _, key := range [][2]string{{d.Standard, "missing"}, {"missing", d.Name}, {d.Standard, d.Name + "\x00"}} {
			got, ge := Lookup(key[0], key[1])
			want, we := linearLookup(entries, key[0], "", key[1], false)
			assertLookupEqual(t, got, want, ge, we)
			unknowns++
		}
	}
	t.Logf("%d clause entries; %d distinct names (%d ambiguous); %d unknown requests", len(entries), len(names), ambiguous, unknowns)
}

func installLookupCatalogue(t *testing.T, entries []runtime.Descriptor) {
	t.Helper()
	oldEntries, oldIndex := catalogue, catalogueLookup
	catalogue, catalogueLookup = entries, newLookupIndex(entries)
	t.Cleanup(func() { catalogue, catalogueLookup = oldEntries, oldIndex })
}

func TestLookupVersionFailure(t *testing.T) {
	for i, entries := range [][]runtime.Descriptor{
		{{Standard: "synthetic", Version: "1", Name: "Value"}, {Standard: "synthetic", Version: "2", Name: "Other"}},
		{{Standard: "synthetic", Name: "Value"}},
		{{Version: "1", Name: "Value"}},
	} {
		t.Run([]string{"conflicting versions", "missing version", "missing standard"}[i], func(t *testing.T) {
			installLookupCatalogue(t, entries)
			cause := catalogueLookup.versionErr
			if cause == nil {
				t.Fatal("missing cached validation failure")
			}
			for _, key := range [][3]string{{"synthetic", "1", "Value"}, {"missing", "", "Other"}, {"", "", ""}} {
				for _, qualified := range []bool{false, true} {
					var got runtime.Descriptor
					var ge error
					if qualified {
						got, ge = LookupClause(key[0], key[1], key[2])
					} else {
						got, ge = Lookup(key[0], key[2])
					}
					want, we := linearLookup(entries, key[0], key[1], key[2], qualified)
					assertLookupEqual(t, got, want, ge, we)
					if errors.Unwrap(ge) != cause {
						t.Fatal("lookup did not retain the cached error value")
					}
				}
				_, de := Decode(key[0], key[2], nil)
				_, ce := DecodeWithContext(key[0], key[2], nil, 0)
				want := lookupFailure(key[0], "", key[2], cause)
				assertLookupErrorEqual(t, de, want)
				assertLookupErrorEqual(t, ce, want)
			}
		})
	}
}

func TestLookupUsesCatalogueSnapshot(t *testing.T) {
	entry := Descriptors()[0]
	entries := []runtime.Descriptor{entry}
	installLookupCatalogue(t, entries)
	// The test seam can alter the input after indexing; production never does.
	// Both scanning and revalidating that input would change these results.
	entries[0] = runtime.Descriptor{}
	got, ge := Lookup(entry.Standard, entry.Name)
	assertLookupEqual(t, got, entry, ge, nil)
	got, ge = LookupClause(entry.Standard, entry.Clause, entry.Name)
	assertLookupEqual(t, got, entry, ge, nil)
}

func TestLookupDuplicateClauseFirstMatch(t *testing.T) {
	first := Descriptors()[0]
	second := first
	second.MaxOctets++
	installLookupCatalogue(t, []runtime.Descriptor{first, second})
	got, ge := LookupClause(first.Standard, first.Clause, first.Name)
	assertLookupEqual(t, got, first, ge, nil)
	got, ge = Lookup(first.Standard, first.Name)
	want, we := linearLookup(catalogue, first.Standard, "", first.Name, false)
	assertLookupEqual(t, got, want, ge, we)
}

func TestLookupAmbiguityErrorDetached(t *testing.T) {
	_, err := Lookup("TS 24.008", "A5 bits")
	var operation *runtime.OperationError
	if !errors.As(err, &operation) {
		t.Fatal(err)
	}
	operation.Operation, operation.Kind, operation.Cause = "changed", runtime.InvalidValue, nil
	operation.Location.Path = "changed"
	got, ge := Lookup("TS 24.008", "A5 bits")
	want, we := linearLookup(Descriptors(), "TS 24.008", "", "A5 bits", false)
	assertLookupEqual(t, got, want, ge, we)
}

func TestLookupUnsortedAndSeparatedKeys(t *testing.T) {
	entries := []runtime.Descriptor{
		{Standard: "synthetic", Version: "1", Clause: "9", Name: "Value"},
		{Standard: "synthetic", Version: "1", Clause: "1", Name: "Value"},
		{Standard: "synthetic", Version: "1", Clause: "5", Name: "Value"},
		{Standard: "a", Version: "1", Clause: "b\x00c", Name: "d"},
		{Standard: "a\x00b", Version: "1", Clause: "c", Name: "d"},
		{Standard: "a", Version: "1", Clause: "b", Name: "c\x00d"},
	}
	installLookupCatalogue(t, entries)
	for _, d := range entries {
		got, ge := Lookup(d.Standard, d.Name)
		want, we := linearLookup(entries, d.Standard, "", d.Name, false)
		assertLookupEqual(t, got, want, ge, we)
		got, ge = LookupClause(d.Standard, d.Clause, d.Name)
		want, we = linearLookup(entries, d.Standard, d.Clause, d.Name, true)
		assertLookupEqual(t, got, want, ge, we)
	}
}

func TestLookupEmptyCatalogue(t *testing.T) {
	installLookupCatalogue(t, nil)
	got, ge := Lookup("", "")
	want, we := linearLookup(nil, "", "", "", false)
	assertLookupEqual(t, got, want, ge, we)
	got, ge = LookupClause("", "", "")
	want, we = linearLookup(nil, "", "", "", true)
	assertLookupEqual(t, got, want, ge, we)
}
