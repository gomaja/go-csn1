// Package registry resolves published CSN.1 definition names and clauses.
// Each module carries one version per standard; version is descriptor metadata,
// not part of a lookup key. Consumers update with main rather than select a
// historical version through this registry.
package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts24008/msnetcap"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
	"github.com/gomaja/go-csn1/ts44018/cellselection"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44060/ies"
)

func descriptors() []runtime.Descriptor {
	var out []runtime.Descriptor
	out = append(out, classmark.Descriptors()...)
	out = append(out, msrac.Descriptors()...)
	out = append(out, msnetcap.Descriptors()...)
	out = append(out, cellselection.Descriptors()...)
	out = append(out, measurement.Descriptors()...)
	out = append(out, restoctets.Descriptors()...)
	out = append(out, ies.Descriptors()...)
	out = append(out,
		manualDescriptor(runtime.Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.6"}, "Mobile Station Classmark 2 value part", 24, 24, 3, uecapability.DecodeClassmark2ValuePart, uecapability.DecodeClassmark2ValuePartFrom, uecapability.EncodeClassmark2ValuePart, uecapability.EncodeClassmark2ValuePartCanonical, uecapability.EncodeClassmark2ValuePartCanonicalAtLength),
		manualDescriptor(runtime.Source{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions"}, "geran-cs", 48, 296, 37, uecapability.DecodeGERANCS, uecapability.DecodeGERANCSFrom, uecapability.EncodeGERANCS, uecapability.EncodeGERANCSCanonical, uecapability.EncodeGERANCSCanonicalAtLength),
		manualDescriptor(runtime.Source{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions"}, "geran-ps", 16, 400, 50, uecapability.DecodeGERANPS, uecapability.DecodeGERANPSFrom, uecapability.EncodeGERANPS, uecapability.EncodeGERANPSCanonical, uecapability.EncodeGERANPSCanonicalAtLength),
	)
	return out
}

func validateVersions(entries []runtime.Descriptor) error {
	versions := make(map[string]string)
	for _, entry := range entries {
		if entry.Standard == "" || entry.Version == "" {
			return fmt.Errorf("CSN.1 descriptor lacks standard or version")
		}
		if prior, ok := versions[entry.Standard]; ok && prior != entry.Version {
			return fmt.Errorf("multiple versions of %s: %s and %s", entry.Standard, prior, entry.Version)
		}
		versions[entry.Standard] = entry.Version
	}
	return nil
}

// Lookup accepts the definition name exactly as printed. A name with multiple
// clause-qualified definitions is rejected with the candidate clauses.
func Lookup(standard, name string) (runtime.Descriptor, error) {
	var matches []runtime.Descriptor
	entries := catalogue
	if err := validateVersions(entries); err != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, err)
	}
	for _, d := range entries {
		if d.Standard == standard && d.Name == name {
			matches = append(matches, d)
		}
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

// LookupClause resolves the exact specification, clause and printed name.
func LookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	entries := catalogue
	if err := validateVersions(entries); err != nil {
		return runtime.Descriptor{}, lookupFailure(standard, "", name, err)
	}
	for _, d := range entries {
		if d.Standard == standard && d.Clause == clause && d.Name == name {
			return d, nil
		}
	}
	return runtime.Descriptor{}, lookupFailure(standard, clause, name, fmt.Errorf("unknown CSN.1 definition %q in %s §%s", name, standard, clause))
}

// Decode resolves a printed definition name. SI7/SI8 return a typed
// context-required error because TS 44.018 V19.0.0 §10.5.2.35 selects
// their layout from the ACS bit in the containing SI4 message.
func Decode(standard, name string, data []byte) (any, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, err
	}
	return d.Decode(data)
}

// DecodeWithContext supplies the containing SI4 ACS bit for SI7/SI8.
func DecodeWithContext(standard, name string, data []byte, acs runtime.SI4ACS) (any, error) {
	d, err := Lookup(standard, name)
	if err != nil {
		return nil, err
	}
	if d.DecodeWithContext == nil {
		return nil, &runtime.OperationError{Operation: "context", Kind: runtime.InvalidValue, Location: runtime.Location{Path: d.Name, BitOffset: -1, Source: d.Extent.Source}, Cause: runtime.ErrUnsupported}
	}
	return d.DecodeWithContext(data, acs)
}

// Descriptors returns a detached catalogue in stable standard, clause and name
// order. Changing entries or function slots never changes subsequent lookups.
func Descriptors() []runtime.Descriptor { return append([]runtime.Descriptor(nil), catalogue...) }

var catalogue = func() []runtime.Descriptor {
	entries := descriptors()
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Standard != b.Standard {
			return a.Standard < b.Standard
		}
		if a.Clause != b.Clause {
			return a.Clause < b.Clause
		}
		return a.Name < b.Name
	})
	return entries
}()

func lookupFailure(standard, clause, name string, cause error) error {
	return &runtime.OperationError{Operation: "lookup", Kind: runtime.InvalidSchema, Location: runtime.Location{Path: name, BitOffset: -1, Source: runtime.Source{Standard: standard, Clause: clause}}, Cause: cause}
}

// The manual finite canonical bounds are TS 24.008 V20.1.0 §§10.5.1.6,
// 10.5.1.7, 10.5.5.12a and TS 36.331 V19.4.0 §6.3.6
// UE-CapabilityRAT-ContainerList field descriptions; receiver tails are separate.
func manualDescriptor[T any](source runtime.Source, name string, minBits, maxBits, maxOctets int, decode func([]byte) (runtime.Decoded[T], error), from func(*runtime.Reader) (T, error), encode, canonical func(T) ([]byte, error), atLength func(T, int) ([]byte, error)) runtime.Descriptor {
	wrong := func(operation string) error {
		return &runtime.OperationError{Operation: operation, Kind: runtime.InvalidValue, Location: runtime.Location{Path: name, BitOffset: -1, Source: source}, Detail: "wrong value type"}
	}
	value := func(v any) (T, bool) {
		switch v := v.(type) {
		case T:
			return v, true
		case runtime.Decoded[T]:
			return v.Value, true
		default:
			var zero T
			return zero, false
		}
	}
	return runtime.Descriptor{Standard: source.Standard, Version: source.Version, Clause: source.Clause, Name: name, MaxOctets: maxOctets,
		Extent: runtime.Extent{MinimumBits: minBits, MaximumBits: maxBits, Finite: true, Fixed: minBits == maxBits, NeedsContext: minBits != maxBits, Source: source},
		Decode: func(b []byte) (any, error) { return decode(b) },
		DecodeFrom: func(r *runtime.Reader) (any, error) {
			if err := r.Check(); err != nil {
				return nil, &runtime.OperationError{Operation: "decode", Kind: runtime.InvalidValue, Location: runtime.Location{Path: name, BitOffset: -1, Source: source}, Cause: err}
			}
			start := r.Position()
			v, err := from(r)
			if err != nil {
				return nil, err
			}
			tail := runtime.BitString{}
			switch value := any(v).(type) {
			case uecapability.Classmark2ValuePart:
				tail = value.Wire.Tail
			case uecapability.GERANCS:
				tail = value.Wire.Tail
			case uecapability.GERANPS:
				tail = value.Wire.Tail
			}
			return runtime.Decoded[T]{Value: v, BitsConsumed: r.Position() - start, Tail: tail}, nil
		},
		Encode: func(v any) ([]byte, error) {
			x, ok := value(v)
			if !ok {
				return nil, wrong("encode")
			}
			return encode(x)
		},
		Canonical: func(v any) ([]byte, error) {
			x, ok := value(v)
			if !ok {
				return nil, wrong("canonical")
			}
			return canonical(x)
		},
		CanonicalAtLength: func(v any, n int) ([]byte, error) {
			x, ok := value(v)
			if !ok {
				return nil, wrong("canonical")
			}
			return atLength(x, n)
		},
	}
}
