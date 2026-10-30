package registry

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// sampleDescriptors preserves discovery order because seeded probes consume
// either a shared random stream or the descriptor's ordinal as their seed.
func sampleDescriptors(t testing.TB, current []runtime.Descriptor, fixture string) []runtime.Descriptor {
	t.Helper()
	data, err := os.ReadFile("testdata/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	key := func(d runtime.Descriptor) string { return d.Standard + "\t" + d.Clause + "\t" + d.Name }
	byKey := make(map[string]runtime.Descriptor, len(current))
	for _, d := range current {
		k := key(d)
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
		if _, ok := byKey[key(d)]; ok {
			result = append(result, d)
		}
	}
	return result
}

func TestPublishedSampleOrders(t *testing.T) {
	for _, tc := range []struct{ file, hash string }{
		{"direct-probe-order.tsv", "f367ac89e9a4024ada8981185bd5a06e53bc1a38ce54cc95534187af0405f937"},
		{"sorted-probe-order.tsv", "ad02ecc395617a8ef1b6a5e0990035bace964cace3b2af04d774b6cdb166d760"},
		{"canonical-probe-order.tsv", "4eef9aede8885e4828fb3be1c5f1d2ba2b80b49f97b6af5d5a24fe89e58b7c17"},
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
			if got := d.Standard + "\t" + d.Clause + "\t" + d.Name; got != line {
				t.Errorf("%s entry %d=%q, want %q", tc.file, i, got, line)
			}
		}

		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != tc.hash {
			t.Errorf("published sample order %s changed: %s", tc.file, got)
		}
		if got := len(strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")); got != 288 {
			t.Errorf("sample order %s has %d entries, want 288", tc.file, got)
		}
	}
}
