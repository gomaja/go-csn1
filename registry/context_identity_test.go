package registry

import (
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestContextIdentity(t *testing.T) {
	a := runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: "9", Name: "Printed", Context: `{"table":"a"}`}
	b := a
	b.Context = `{"table":"b"}`
	installLookupCatalogue(t, []runtime.Descriptor{b, a})
	for _, want := range []runtime.Descriptor{a, b} {
		got, err := LookupContext(want.Standard, want.Clause, want.Name, want.Context)
		assertLookupEqual(t, got, want, err, nil)
	}
	if _, err := LookupClause(a.Standard, a.Clause, a.Name); err == nil {
		t.Fatal("clause lookup accepted ambiguous contexts")
	}
	if _, err := Lookup(a.Standard, a.Name); err == nil {
		t.Fatal("name lookup accepted ambiguous contexts")
	}
	if _, err := LookupContext(a.Standard, a.Clause, a.Name, strings.ToUpper(a.Context)); err == nil {
		t.Fatal("context lookup was not exact")
	}
}

func TestDuplicateContextIdentityRejected(t *testing.T) {
	a := runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: "9", Name: "Printed", Context: `{"table":"a"}`}
	installLookupCatalogue(t, []runtime.Descriptor{a, a})
	if _, err := LookupContext(a.Standard, a.Clause, a.Name, a.Context); err == nil {
		t.Fatal("duplicate identity accepted")
	}
	// The name and clause lookups refuse the same catalogue, as the linear oracle does.
	got, err := Lookup(a.Standard, a.Name)
	want, wantErr := linearLookup(catalogue, a.Standard, "", a.Name, false)
	if err == nil {
		t.Fatal("name lookup accepted a duplicate identity")
	}
	assertLookupEqual(t, got, want, err, wantErr)
	got, err = LookupClause(a.Standard, a.Clause, a.Name)
	want, wantErr = linearLookup(catalogue, a.Standard, a.Clause, a.Name, true)
	if err == nil {
		t.Fatal("clause lookup accepted a duplicate identity")
	}
	assertLookupEqual(t, got, want, err, wantErr)
}

func TestDescriptorContextSort(t *testing.T) {
	a := runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: "9", Name: "Printed", Context: "a"}
	b := a
	b.Context = "b"
	entries := []runtime.Descriptor{b, a}
	sortDescriptors(entries)
	if entries[0].Context != "a" || entries[1].Context != "b" {
		t.Fatalf("context order: %q, %q", entries[0].Context, entries[1].Context)
	}
}

func TestLookupContextEmptyAndVersionFailure(t *testing.T) {
	a := runtime.Descriptor{Standard: "synthetic", Version: "1", Clause: "9", Name: "Printed"}
	b := a
	b.Context = `{"table":"a"}`
	installLookupCatalogue(t, []runtime.Descriptor{a, b})
	got, err := LookupContext(a.Standard, a.Clause, a.Name, "")
	assertLookupEqual(t, got, a, err, nil)
	if _, err := LookupContext(a.Standard, a.Clause, a.Name, "missing"); err == nil {
		t.Fatal("unknown context accepted")
	}
	b.Version = "2"
	catalogueLookup = newLookupIndex([]runtime.Descriptor{a, b})
	if _, err := LookupContext(a.Standard, a.Clause, a.Name, ""); err == nil {
		t.Fatal("inconsistent versions accepted")
	}
}
