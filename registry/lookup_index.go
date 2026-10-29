package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gomaja/go-csn1/runtime"
)

type nameKey struct{ standard, name string }
type clauseKey struct{ standard, clause, name string }

type lookupResult struct {
	descriptor runtime.Descriptor
	ambiguity  error
}

type lookupIndex struct {
	versionErr error
	names      map[nameKey]lookupResult
	clauses    map[clauseKey]runtime.Descriptor
}

var catalogueLookup = newLookupIndex(catalogue)

// newLookupIndex snapshots the immutable catalogue, including validation failures.
// Validation errors are returned by lookup rather than panicking during initialization.
func newLookupIndex(entries []runtime.Descriptor) lookupIndex {
	index := lookupIndex{versionErr: validateVersions(entries)}
	if index.versionErr != nil {
		return index
	}
	index.names = make(map[nameKey]lookupResult)
	index.clauses = make(map[clauseKey]runtime.Descriptor)
	clauses := make(map[nameKey][]string)
	for _, d := range entries {
		key := nameKey{d.Standard, d.Name}
		index.names[key] = lookupResult{descriptor: d}
		clauses[key] = append(clauses[key], d.Clause)
		qualified := clauseKey{d.Standard, d.Clause, d.Name}
		if _, exists := index.clauses[qualified]; !exists {
			index.clauses[qualified] = d
		}
	}
	for key, candidates := range clauses {
		if len(candidates) > 1 {
			sort.Strings(candidates)
			index.names[key] = lookupResult{ambiguity: fmt.Errorf("ambiguous CSN.1 definition %q in %s; clauses: %s", key.name, key.standard, strings.Join(candidates, ", "))}
		}
	}
	return index
}

func (index lookupIndex) lookup(standard, name string) (runtime.Descriptor, error) {
	if index.versionErr != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, index.versionErr)
	}
	if result, ok := index.names[nameKey{standard, name}]; ok {
		if result.ambiguity != nil {
			return runtime.Descriptor{}, lookupFailure(standard, "", name, result.ambiguity)
		}
		return result.descriptor, nil
	}
	return runtime.Descriptor{}, lookupFailure(standard, "", name, fmt.Errorf("unknown CSN.1 definition %q in %s", name, standard))
}

func (index lookupIndex) lookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	if index.versionErr != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, index.versionErr)
	}
	if descriptor, ok := index.clauses[clauseKey{standard, clause, name}]; ok {
		return descriptor, nil
	}
	return runtime.Descriptor{}, lookupFailure(standard, clause, name, fmt.Errorf("unknown CSN.1 definition %q in %s §%s", name, standard, clause))
}
