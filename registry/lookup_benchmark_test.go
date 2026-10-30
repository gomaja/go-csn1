package registry

import "testing"

func BenchmarkLookup(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		lookupDescriptorSink, lookupErrorSink = Lookup("TS 24.008", "Mobile Station Classmark 2 value part")
	}
}

func BenchmarkLookupClauseLast(b *testing.B) {
	all := Descriptors()
	d := all[len(all)-1]
	b.ReportAllocs()
	for b.Loop() {
		lookupDescriptorSink, lookupErrorSink = LookupClause(d.Standard, d.Clause, d.Name)
	}
}

func BenchmarkDecodeDirect(b *testing.B) {
	d, err := LookupClause("TS 24.008", "10.5.1.6", "Mobile Station Classmark 2 value part")
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, 3)
	b.ReportAllocs()
	for b.Loop() {
		lookupValueSink, lookupErrorSink = d.Decode(data)
	}
}

func BenchmarkDecodeRegistry(b *testing.B) {
	data := make([]byte, 3)
	b.ReportAllocs()
	for b.Loop() {
		lookupValueSink, lookupErrorSink = Decode("TS 24.008", "Mobile Station Classmark 2 value part", data)
	}
}
