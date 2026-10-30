package registry

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

var lookupDescriptorSink runtime.Descriptor
var lookupErrorSink error
var lookupValueSink any

func TestLookupAllocations(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under race instrumentation")
	}
	for _, d := range Descriptors() {
		if _, err := Lookup(d.Standard, d.Name); err == nil {
			if n := testing.AllocsPerRun(100, func() { lookupDescriptorSink, lookupErrorSink = Lookup(d.Standard, d.Name) }); n != 0 {
				t.Errorf("Lookup %s/%s: %g allocations, want zero", d.Standard, d.Name, n)
			}
		}
		if n := testing.AllocsPerRun(100, func() { lookupDescriptorSink, lookupErrorSink = LookupClause(d.Standard, d.Clause, d.Name) }); n != 0 {
			t.Errorf("LookupClause %s/%s/%s: %g allocations, want zero", d.Standard, d.Clause, d.Name, n)
		}
	}
}

func TestLookupAllocationProfile(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation counts differ under race instrumentation")
	}
	d, err := LookupClause("TS 24.008", "10.5.1.6", "Mobile Station Classmark 2 value part")
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 3)
	for _, tc := range []struct {
		name string
		call func()
	}{
		{"Lookup", func() { lookupDescriptorSink, lookupErrorSink = Lookup(d.Standard, d.Name) }},
		{"LookupClause", func() { lookupDescriptorSink, lookupErrorSink = LookupClause(d.Standard, d.Clause, d.Name) }},
		{"unknown name", func() { lookupDescriptorSink, lookupErrorSink = Lookup(d.Standard, "missing") }},
		{"unknown clause", func() { lookupDescriptorSink, lookupErrorSink = LookupClause(d.Standard, "missing", d.Name) }},
		{"ambiguous", func() { lookupDescriptorSink, lookupErrorSink = Lookup("TS 24.008", "A5 bits") }},
		{"descriptor Decode", func() { lookupValueSink, lookupErrorSink = d.Decode(data) }},
		{"registry Decode", func() { lookupValueSink, lookupErrorSink = Decode(d.Standard, d.Name, data) }},
	} {
		t.Logf("%s: %g allocations", tc.name, testing.AllocsPerRun(100, tc.call))
	}
}
