package registry

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gomaja/go-csn1/runtime"
)

type (
	nameKey    struct{ standard, name string }
	clauseKey  struct{ standard, clause, name string }
	contextKey struct{ standard, clause, name, context string }
)

type lookupIndex struct {
	entries           []runtime.Descriptor
	versionErr        error
	names             map[nameKey]int
	clauses           map[clauseKey]int
	contexts          map[contextKey]int
	nameAmbiguities   map[nameKey]error
	clauseAmbiguities map[clauseKey]error
}

var catalogueLookup = sync.OnceValue(func() lookupIndex { return newLookupIndex(catalogue()) })

// newLookupIndex references an immutable catalogue and caches validation failures.
// Callers retain the catalogue without changing its entries after indexing.
func newLookupIndex(entries []runtime.Descriptor) lookupIndex {
	index := lookupIndex{entries: entries, versionErr: validateVersions(entries)}
	if index.versionErr != nil {
		return index
	}
	index.names = make(map[nameKey]int, len(entries))
	index.clauses = make(map[clauseKey]int, len(entries))
	index.contexts = make(map[contextKey]int, len(entries))
	candidates := make(map[nameKey][]string)
	for position, d := range entries {
		key := nameKey{d.Standard, d.Name}
		if prior, exists := index.names[key]; exists {
			if candidates[key] == nil {
				candidates[key] = []string{entries[prior].Clause}
			}
			candidates[key] = append(candidates[key], d.Clause)
		} else {
			index.names[key] = position
		}
		qualified := clauseKey{d.Standard, d.Clause, d.Name}
		if _, exists := index.clauses[qualified]; exists {
			if index.clauseAmbiguities == nil {
				index.clauseAmbiguities = make(map[clauseKey]error)
			}
			index.clauseAmbiguities[qualified] = fmt.Errorf("ambiguous CSN.1 definition %q in %s §%s; specify context", d.Name, d.Standard, d.Clause)
		} else {
			index.clauses[qualified] = position
		}
		identity := contextKey{d.Standard, d.Clause, d.Name, d.Context}
		if _, exists := index.contexts[identity]; exists {
			index.versionErr = fmt.Errorf("duplicate CSN.1 descriptor %q in %s §%s with context %q", d.Name, d.Standard, d.Clause, d.Context)
			return index
		}
		index.contexts[identity] = position
	}
	if len(candidates) > 0 {
		index.nameAmbiguities = make(map[nameKey]error, len(candidates))
	}
	for key, clauses := range candidates {
		sort.Strings(clauses)
		index.nameAmbiguities[key] = fmt.Errorf("ambiguous CSN.1 definition %q in %s; clauses: %s", key.name, key.standard, strings.Join(clauses, ", "))
	}
	return index
}

func (index lookupIndex) lookup(standard, name string) (runtime.Descriptor, error) {
	if index.versionErr != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, index.versionErr)
	}
	if position, ok := index.names[nameKey{standard, name}]; ok {
		if ambiguity := index.nameAmbiguities[nameKey{standard, name}]; ambiguity != nil {
			return runtime.Descriptor{}, lookupFailure(standard, "", name, ambiguity)
		}
		return index.entries[position], nil
	}
	return runtime.Descriptor{}, lookupFailure(standard, "", name, fmt.Errorf("unknown CSN.1 definition %q in %s", name, standard))
}

func (index lookupIndex) lookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	if index.versionErr != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, index.versionErr)
	}
	if position, ok := index.clauses[clauseKey{standard, clause, name}]; ok {
		if ambiguity := index.clauseAmbiguities[clauseKey{standard, clause, name}]; ambiguity != nil {
			return runtime.Descriptor{}, lookupFailure(standard, clause, name, ambiguity)
		}
		return index.entries[position], nil
	}
	return runtime.Descriptor{}, lookupFailure(standard, clause, name, fmt.Errorf("unknown CSN.1 definition %q in %s §%s", name, standard, clause))
}

func (index lookupIndex) lookupContext(standard, clause, name, context string) (runtime.Descriptor, error) {
	if index.versionErr != nil {
		return runtime.Descriptor{}, lookupFailure(standard, clause, name, index.versionErr)
	}
	if position, ok := index.contexts[contextKey{standard, clause, name, context}]; ok {
		return index.entries[position], nil
	}
	return runtime.Descriptor{}, lookupFailure(standard, clause, name, fmt.Errorf("unknown CSN.1 definition %q in %s §%s with context %q", name, standard, clause, context))
}
