// Package registry resolves published CSN.1 definition names and clauses.
// Each module carries one version per standard; version is descriptor metadata,
// not part of a lookup key. Consumers update with main rather than select a
// historical version through this registry.
package registry

import (
	"fmt"
	"sort"

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
		manualDescriptor(manualSourceClassmark2ValuePart, "Mobile Station Classmark 2 value part", 24, 24, 3, manualDecodeClassmark2ValuePart, manualDecodeClassmark2ValuePartFrom, manualEncodeClassmark2ValuePart, manualCanonicalClassmark2ValuePart, manualCanonicalClassmark2ValuePartAtLength),
		manualDescriptor(manualSourceGERANCS, "geran-cs", 48, 296, 37, manualDecodeGERANCS, manualDecodeGERANCSFrom, manualEncodeGERANCS, manualCanonicalGERANCS, manualCanonicalGERANCSAtLength),
		manualDescriptor(manualSourceGERANPS, "geran-ps", 16, 400, 50, manualDecodeGERANPS, manualDecodeGERANPSFrom, manualEncodeGERANPS, manualCanonicalGERANPS, manualCanonicalGERANPSAtLength),
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
	return catalogueLookup.lookup(standard, name)
}

// LookupClause resolves the exact specification, clause and printed name.
func LookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	return catalogueLookup.lookupClause(standard, clause, name)
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
func manualDescriptor(source runtime.Source, name string, minBits, maxBits, maxOctets int, decode func([]byte) (any, error), from func(*runtime.Reader) (any, error), encode, canonical func(any) ([]byte, error), atLength func(any, int) ([]byte, error)) runtime.Descriptor {
	return runtime.Descriptor{Standard: source.Standard, Version: source.Version, Clause: source.Clause, Name: name, MaxOctets: maxOctets,
		Extent: runtime.Extent{MinimumBits: minBits, MaximumBits: maxBits, Finite: true, Fixed: minBits == maxBits, NeedsContext: minBits != maxBits, Source: source},
		Decode: decode, DecodeFrom: from, Encode: encode, Canonical: canonical, CanonicalAtLength: atLength,
		DecodeWithCollector: func(data []byte, c *runtime.Collector) (any, error) {
			if err := runtime.CheckInput(data); err != nil {
				return nil, err
			}
			r := runtime.NewReaderWithCollector(data, c)
			if _, err := r.PushLimit(len(data) * 8); err != nil {
				return nil, err
			}
			// Keep the report entry point's existing nil value on failure.
			value, err := from(r)
			if err != nil {
				return nil, err
			}
			return value, nil
		}}
}
func manualTypeError(operation, name string, source runtime.Source) error {
	return &runtime.OperationError{Operation: operation, Kind: runtime.InvalidValue, Location: runtime.Location{Path: name, BitOffset: -1, Source: source}, Detail: "wrong value type"}
}
func manualReaderCheck(r *runtime.Reader, name string, source runtime.Source) error {
	if err := r.Check(); err != nil {
		return &runtime.OperationError{Operation: "decode", Kind: runtime.InvalidValue, Location: runtime.Location{Path: name, BitOffset: -1, Source: source}, Cause: err}
	}
	return nil
}

var manualSourceClassmark2ValuePart = runtime.Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.6"}

func manualDecodeClassmark2ValuePart(data []byte) (any, error) {
	return uecapability.DecodeClassmark2ValuePart(data)
}
func manualDecodeClassmark2ValuePartFrom(r *runtime.Reader) (any, error) {
	if err := manualReaderCheck(r, "Mobile Station Classmark 2 value part", manualSourceClassmark2ValuePart); err != nil {
		return runtime.Decoded[uecapability.Classmark2ValuePart]{}, err
	}
	start := r.Position()
	before := r.Wire()
	origin := r.Fork()
	v, err := uecapability.DecodeClassmark2ValuePartFrom(r)
	if err != nil {
		return runtime.Decoded[uecapability.Classmark2ValuePart]{}, err
	}
	// The typed reader stops after the three octets, so a containing value can
	// continue; the descriptor keeps the rest of a bounded reader in Tail and in
	// the value's received form, as its byte entry point does.
	wire, err := r.ValueWire(origin, before, true)
	if err != nil {
		return runtime.Decoded[uecapability.Classmark2ValuePart]{}, err
	}
	// TS 24.008 V20.1.0 §10.5.1.6 fixes the value at three octets.
	// Extend only its received tail; keep the typed reader's wire conventions.
	v.Wire.Tail = wire.Tail
	v.Wire.TransmittedBits = wire.TransmittedBits
	return runtime.Decoded[uecapability.Classmark2ValuePart]{Value: v, BitsConsumed: r.Position() - start, Tail: wire.Tail}, nil
}
func manualUnwrapClassmark2ValuePart(value any) (uecapability.Classmark2ValuePart, bool) {
	switch v := value.(type) {
	case uecapability.Classmark2ValuePart:
		return v, true
	case runtime.Decoded[uecapability.Classmark2ValuePart]:
		return v.Value, true
	default:
		return uecapability.Classmark2ValuePart{}, false
	}
}
func manualEncodeClassmark2ValuePart(value any) ([]byte, error) {
	v, ok := manualUnwrapClassmark2ValuePart(value)
	if !ok {
		return nil, manualTypeError("encode", "Mobile Station Classmark 2 value part", manualSourceClassmark2ValuePart)
	}
	return uecapability.EncodeClassmark2ValuePart(v)
}
func manualCanonicalClassmark2ValuePart(value any) ([]byte, error) {
	v, ok := manualUnwrapClassmark2ValuePart(value)
	if !ok {
		return nil, manualTypeError("canonical", "Mobile Station Classmark 2 value part", manualSourceClassmark2ValuePart)
	}
	return uecapability.EncodeClassmark2ValuePartCanonical(v)
}
func manualCanonicalClassmark2ValuePartAtLength(value any, octets int) ([]byte, error) {
	v, ok := manualUnwrapClassmark2ValuePart(value)
	if !ok {
		return nil, manualTypeError("canonical", "Mobile Station Classmark 2 value part", manualSourceClassmark2ValuePart)
	}
	return uecapability.EncodeClassmark2ValuePartCanonicalAtLength(v, octets)
}

var manualSourceGERANCS = runtime.Source{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions"}

func manualDecodeGERANCS(data []byte) (any, error) { return uecapability.DecodeGERANCS(data) }
func manualDecodeGERANCSFrom(r *runtime.Reader) (any, error) {
	if err := manualReaderCheck(r, "geran-cs", manualSourceGERANCS); err != nil {
		return runtime.Decoded[uecapability.GERANCS]{}, err
	}
	start := r.Position()
	v, err := uecapability.DecodeGERANCSFrom(r)
	if err != nil {
		return runtime.Decoded[uecapability.GERANCS]{}, err
	}
	return runtime.Decoded[uecapability.GERANCS]{Value: v, BitsConsumed: r.Position() - start, Tail: v.Wire.Tail}, nil
}
func manualUnwrapGERANCS(value any) (uecapability.GERANCS, bool) {
	switch v := value.(type) {
	case uecapability.GERANCS:
		return v, true
	case runtime.Decoded[uecapability.GERANCS]:
		return v.Value, true
	default:
		return uecapability.GERANCS{}, false
	}
}
func manualEncodeGERANCS(value any) ([]byte, error) {
	v, ok := manualUnwrapGERANCS(value)
	if !ok {
		return nil, manualTypeError("encode", "geran-cs", manualSourceGERANCS)
	}
	return uecapability.EncodeGERANCS(v)
}
func manualCanonicalGERANCS(value any) ([]byte, error) {
	v, ok := manualUnwrapGERANCS(value)
	if !ok {
		return nil, manualTypeError("canonical", "geran-cs", manualSourceGERANCS)
	}
	return uecapability.EncodeGERANCSCanonical(v)
}
func manualCanonicalGERANCSAtLength(value any, octets int) ([]byte, error) {
	v, ok := manualUnwrapGERANCS(value)
	if !ok {
		return nil, manualTypeError("canonical", "geran-cs", manualSourceGERANCS)
	}
	return uecapability.EncodeGERANCSCanonicalAtLength(v, octets)
}

var manualSourceGERANPS = runtime.Source{Standard: "TS 36.331", Version: "19.4.0", Clause: "UE-CapabilityRAT-ContainerList field descriptions"}

func manualDecodeGERANPS(data []byte) (any, error) { return uecapability.DecodeGERANPS(data) }
func manualDecodeGERANPSFrom(r *runtime.Reader) (any, error) {
	if err := manualReaderCheck(r, "geran-ps", manualSourceGERANPS); err != nil {
		return runtime.Decoded[uecapability.GERANPS]{}, err
	}
	start := r.Position()
	v, err := uecapability.DecodeGERANPSFrom(r)
	if err != nil {
		return runtime.Decoded[uecapability.GERANPS]{}, err
	}
	return runtime.Decoded[uecapability.GERANPS]{Value: v, BitsConsumed: r.Position() - start, Tail: v.Wire.Tail}, nil
}
func manualUnwrapGERANPS(value any) (uecapability.GERANPS, bool) {
	switch v := value.(type) {
	case uecapability.GERANPS:
		return v, true
	case runtime.Decoded[uecapability.GERANPS]:
		return v.Value, true
	default:
		return uecapability.GERANPS{}, false
	}
}
func manualEncodeGERANPS(value any) ([]byte, error) {
	v, ok := manualUnwrapGERANPS(value)
	if !ok {
		return nil, manualTypeError("encode", "geran-ps", manualSourceGERANPS)
	}
	return uecapability.EncodeGERANPS(v)
}
func manualCanonicalGERANPS(value any) ([]byte, error) {
	v, ok := manualUnwrapGERANPS(value)
	if !ok {
		return nil, manualTypeError("canonical", "geran-ps", manualSourceGERANPS)
	}
	return uecapability.EncodeGERANPSCanonical(v)
}
func manualCanonicalGERANPSAtLength(value any, octets int) ([]byte, error) {
	v, ok := manualUnwrapGERANPS(value)
	if !ok {
		return nil, manualTypeError("canonical", "geran-ps", manualSourceGERANPS)
	}
	return uecapability.EncodeGERANPSCanonicalAtLength(v, octets)
}
