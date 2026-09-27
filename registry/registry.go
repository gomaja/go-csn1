// Package registry resolves published CSN.1 definition names and clauses.
package registry

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44060/ies"
)

func descriptors() []runtime.Descriptor {
	out := make([]runtime.Descriptor, 0, len(classmark.Definitions())+len(msrac.Definitions())+len(measurement.Definitions())+len(restoctets.Definitions())+len(ies.Definitions())+3)
	out = append(out, classmark.Descriptors()...)
	out = append(out, msrac.Descriptors()...)
	out = append(out, measurement.Descriptors()...)
	out = append(out, restoctets.Descriptors()...)
	out = append(out, ies.Descriptors()...)
	out = append(out,
		runtime.Descriptor{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.6", Name: "Mobile Station Classmark 2 value part",
			Decode: func(b []byte) (any, error) { return uecapability.DecodeClassmark2ValuePart(b) },
			Encode: func(v any) ([]byte, error) {
				typed, ok := v.(uecapability.Classmark2ValuePart)
				if !ok {
					return nil, fmt.Errorf("wrong Classmark 2 value type")
				}
				return uecapability.EncodeClassmark2ValuePart(typed)
			}},
		runtime.Descriptor{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions", Name: "geran-cs",
			Decode: func(b []byte) (any, error) { return uecapability.DecodeGERANCS(b) },
			Encode: func(v any) ([]byte, error) {
				typed, ok := v.(uecapability.GERANCS)
				if !ok {
					return nil, fmt.Errorf("wrong geran-cs value type")
				}
				return uecapability.EncodeGERANCS(typed)
			}},
		runtime.Descriptor{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions", Name: "geran-ps",
			Decode: func(b []byte) (any, error) { return uecapability.DecodeGERANPS(b) },
			Encode: func(v any) ([]byte, error) {
				typed, ok := v.(uecapability.GERANPS)
				if !ok {
					return nil, fmt.Errorf("wrong geran-ps value type")
				}
				return uecapability.EncodeGERANPS(typed)
			}},
	)
	return out
}

// Lookup accepts the definition name exactly as printed. A name with multiple
// clause-qualified definitions is rejected with the candidate clauses.
func Lookup(standard, name string) (runtime.Descriptor, error) {
	var matches []runtime.Descriptor
	for _, d := range descriptors() {
		if d.Standard == standard && d.Name == name {
			matches = append(matches, d)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return runtime.Descriptor{}, fmt.Errorf("unknown CSN.1 definition %q in %s", name, standard)
	}
	clauses := make([]string, 0, len(matches))
	for _, d := range matches {
		clauses = append(clauses, d.Clause)
	}
	sort.Strings(clauses)
	return runtime.Descriptor{}, fmt.Errorf("ambiguous CSN.1 definition %q in %s; clauses: %s", name, standard, strings.Join(clauses, ", "))
}

// LookupClause resolves the exact specification, clause and printed name.
func LookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	for _, d := range descriptors() {
		if d.Standard == standard && d.Clause == clause && d.Name == name {
			return d, nil
		}
	}
	return runtime.Descriptor{}, fmt.Errorf("unknown CSN.1 definition %q in %s §%s", name, standard, clause)
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
		return nil, fmt.Errorf("%w: %s §%s <%s> has no ACS context", runtime.ErrUnsupported, d.Standard, d.Clause, d.Name)
	}
	return d.DecodeWithContext(data, acs)
}
