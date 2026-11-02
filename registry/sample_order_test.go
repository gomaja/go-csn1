package registry

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// sampleKey names a descriptor in the sample-order fixtures: standard, clause and
// printed name, then the publication context when it has one.
func sampleKey(d runtime.Descriptor) string {
	base := d.Standard + "\t" + d.Clause + "\t" + d.Name
	if d.Context != "" {
		return base + "\t" + d.Context
	}
	return base
}

// sampleDescriptors preserves discovery order because seeded probes consume
// either a shared random stream or the descriptor's ordinal as their seed.
func sampleDescriptors(t testing.TB, current []runtime.Descriptor, fixture string) []runtime.Descriptor {
	t.Helper()
	data, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	byKey := make(map[string]runtime.Descriptor, len(current))
	for _, d := range current {
		k := sampleKey(d)
		if _, ok := byKey[k]; ok {
			t.Fatalf("duplicate descriptor %q", k)
		}
		byKey[k] = d
	}
	result := make([]runtime.Descriptor, 0, len(current))
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		d, ok := byKey[line]
		if !ok {
			t.Fatalf("missing or repeated sample descriptor %q", line)
		}
		result = append(result, d)
		delete(byKey, line)
	}
	for _, d := range current {
		if _, ok := byKey[sampleKey(d)]; ok {
			result = append(result, d)
		}
	}
	return result
}

func TestSampleKeyIncludesContext(t *testing.T) {
	d := runtime.Descriptor{Standard: "synthetic", Clause: "9", Name: "Printed"}
	if got := sampleKey(d); got != "synthetic\t9\tPrinted" {
		t.Fatalf("unqualified sample key = %q", got)
	}
	d.Context = `{"table":"a"}`
	if got := sampleKey(d); got != "synthetic\t9\tPrinted\t"+d.Context {
		t.Fatalf("contextual sample key = %q", got)
	}
}

func TestPublishedSampleOrders(t *testing.T) {
	for _, tc := range []struct{ file, hash string }{
		{"direct-probe-order.tsv", "f9440a4bc469a1a8b9933be5eed693396ecfad27c973cafdba62ac945543123d"},
		{"sorted-probe-order.tsv", "da07910d985b03efeec133a4f2ea3e1204d8e5828aa3773a3915d0c1a581da93"},
		{"canonical-probe-order.tsv", "dbeaa585ef10c5e884f9fbb2128f49fc70e0fb226f8dcab624108f20133487b0"},
	} {
		current := descriptors()
		if tc.file == "sorted-probe-order.tsv" {
			current = Descriptors()
		}
		ordered := sampleDescriptors(t, current, tc.file)
		if tc.file == "canonical-probe-order.tsv" {
			ordered = canonicalProbeDefinitions(t)
		}
		if len(ordered) != len(current) {
			t.Fatalf("%s lost descriptors", tc.file)
		}
		data, err := os.ReadFile("testdata/" + tc.file)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
			d := ordered[i]
			if got := sampleKey(d); got != line {
				t.Errorf("%s entry %d=%q, want %q", tc.file, i, got, line)
			}
		}

		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != tc.hash {
			t.Errorf("published sample order %s changed: %s", tc.file, got)
		}
		if got := len(strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")); got != 632 {
			t.Errorf("sample order %s has %d entries, want 632", tc.file, got)
		}
	}
}
