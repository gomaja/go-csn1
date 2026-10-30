package registry_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/registry"
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

func TestPublicProbePublishedOrder(t *testing.T) {
	data, err := os.ReadFile("testdata/sorted-probe-order.tsv")
	if err != nil {
		t.Fatal(err)
	}
	current := registry.Descriptors()
	ordered := sampleDescriptors(t, current, "sorted-probe-order.tsv")
	if len(ordered) != len(current) {
		t.Fatal("sample order lost descriptors")
	}
	for i, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		d := ordered[i]
		if got := d.Standard + "\t" + d.Clause + "\t" + d.Name; got != line {
			t.Errorf("sample entry %d=%q, want %q", i, got, line)
		}
	}
}
