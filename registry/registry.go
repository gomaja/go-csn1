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
	"github.com/gomaja/go-csn1/ts44018/ecaccess"
	"github.com/gomaja/go-csn1/ts44018/ecimmediateassignment2"
	rries "github.com/gomaja/go-csn1/ts44018/ies"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/messages"
	"github.com/gomaja/go-csn1/ts44018/notification"
	"github.com/gomaja/go-csn1/ts44018/packetdlassignment"
	"github.com/gomaja/go-csn1/ts44018/packetdlassignment2"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44018/vgcsreconfigure"
	"github.com/gomaja/go-csn1/ts44018/vgcsreconfigure2"
	"github.com/gomaja/go-csn1/ts44060/access"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
	"github.com/gomaja/go-csn1/ts44060/acknack"
	"github.com/gomaja/go-csn1/ts44060/assignmentmessages"
	"github.com/gomaja/go-csn1/ts44060/celldata"
	"github.com/gomaja/go-csn1/ts44060/containers"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"github.com/gomaja/go-csn1/ts44060/csgreport"
	"github.com/gomaja/go-csn1/ts44060/dbpschuplinkack"
	"github.com/gomaja/go-csn1/ts44060/dbpschuplinkack2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover"
	"github.com/gomaja/go-csn1/ts44060/ecmessages"
	"github.com/gomaja/go-csn1/ts44060/eutrancsgtarget"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/mbmsassignment"
	"github.com/gomaja/go-csn1/ts44060/mbmsmessages"
	"github.com/gomaja/go-csn1/ts44060/measurementreport"
	ts44060messages "github.com/gomaja/go-csn1/ts44060/messages"
	"github.com/gomaja/go-csn1/ts44060/multipledownlink"
	"github.com/gomaja/go-csn1/ts44060/neighbourcelldata"
	"github.com/gomaja/go-csn1/ts44060/nongprsoptions"
	"github.com/gomaja/go-csn1/ts44060/packetdownlinkassignment"
	"github.com/gomaja/go-csn1/ts44060/packetenhancedmeasurement"
	"github.com/gomaja/go-csn1/ts44060/packetmeasurementorder"
	"github.com/gomaja/go-csn1/ts44060/packetsysteminformationtype3"
	"github.com/gomaja/go-csn1/ts44060/packettimeslotreconfigure"
	"github.com/gomaja/go-csn1/ts44060/packetuplinkassignment"
	"github.com/gomaja/go-csn1/ts44060/padding"
	"github.com/gomaja/go-csn1/ts44060/pdchpairs"
	"github.com/gomaja/go-csn1/ts44060/pshandover"
	"github.com/gomaja/go-csn1/ts44060/pshandoverradioresources3"
	"github.com/gomaja/go-csn1/ts44060/psi"
	"github.com/gomaja/go-csn1/ts44060/psilegacy"
	"github.com/gomaja/go-csn1/ts44060/psineighbours"
	"github.com/gomaja/go-csn1/ts44060/pulseformat"
	"github.com/gomaja/go-csn1/ts44060/radioaccesscapability"
	"github.com/gomaja/go-csn1/ts44060/servingcelldata"
	"github.com/gomaja/go-csn1/ts44060/singledownlink"
	"github.com/gomaja/go-csn1/ts44060/singleuplink"
	"github.com/gomaja/go-csn1/ts44060/utrancsgtarget"
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
	out = append(out, rries.Descriptors()...)
	out = append(out, messages.Descriptors()...)
	out = append(out, ts44060messages.Descriptors()...)
	out = append(out, ecaccess.Descriptors()...)
	out = append(out, ecimmediateassignment2.Descriptors()...)
	out = append(out, notification.Descriptors()...)
	out = append(out, packetdlassignment.Descriptors()...)
	out = append(out, packetdlassignment2.Descriptors()...)
	out = append(out, vgcsreconfigure.Descriptors()...)
	out = append(out, vgcsreconfigure2.Descriptors()...)
	out = append(out, access.Descriptors()...)
	out = append(out, ackmessages.Descriptors()...)
	out = append(out, acknack.Descriptors()...)
	out = append(out, assignmentmessages.Descriptors()...)
	out = append(out, celldata.Descriptors()...)
	out = append(out, containers.Descriptors()...)
	out = append(out, controlmessages.Descriptors()...)
	out = append(out, csgreport.Descriptors()...)
	out = append(out, dbpschuplinkack.Descriptors()...)
	out = append(out, dbpschuplinkack2.Descriptors()...)
	out = append(out, dtmhandover.Descriptors()...)
	out = append(out, ecmessages.Descriptors()...)
	out = append(out, eutrancsgtarget.Descriptors()...)
	out = append(out, mbmsassignment.Descriptors()...)
	out = append(out, mbmsmessages.Descriptors()...)
	out = append(out, measurementreport.Descriptors()...)
	out = append(out, multipledownlink.Descriptors()...)
	out = append(out, neighbourcelldata.Descriptors()...)
	out = append(out, nongprsoptions.Descriptors()...)
	out = append(out, packetdownlinkassignment.Descriptors()...)
	out = append(out, packetenhancedmeasurement.Descriptors()...)
	out = append(out, packetmeasurementorder.Descriptors()...)
	out = append(out, packetsysteminformationtype3.Descriptors()...)
	out = append(out, packettimeslotreconfigure.Descriptors()...)
	out = append(out, packetuplinkassignment.Descriptors()...)
	out = append(out, padding.Descriptors()...)
	out = append(out, pdchpairs.Descriptors()...)
	out = append(out, pshandover.Descriptors()...)
	out = append(out, pshandoverradioresources3.Descriptors()...)
	out = append(out, psi.Descriptors()...)
	out = append(out, psilegacy.Descriptors()...)
	out = append(out, psineighbours.Descriptors()...)
	out = append(out, pulseformat.Descriptors()...)
	out = append(out, radioaccesscapability.Descriptors()...)
	out = append(out, servingcelldata.Descriptors()...)
	out = append(out, singledownlink.Descriptors()...)
	out = append(out, singleuplink.Descriptors()...)
	out = append(out, utrancsgtarget.Descriptors()...)
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
// It rejects names with more than one publication context.
func LookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	return catalogueLookup.lookupClause(standard, clause, name)
}

// LookupContext resolves the exact specification, clause, printed name and opaque
// publication context. The empty context selects an unqualified definition.
func LookupContext(standard, clause, name, context string) (runtime.Descriptor, error) {
	return catalogueLookup.lookupContext(standard, clause, name, context)
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

// Descriptors returns a detached catalogue in stable standard, clause, name and context
// order. Changing entries or function slots never changes subsequent lookups.
func Descriptors() []runtime.Descriptor { return append([]runtime.Descriptor(nil), catalogue...) }

var catalogue = func() []runtime.Descriptor {
	entries := descriptors()
	sortDescriptors(entries)
	return entries
}()

func sortDescriptors(entries []runtime.Descriptor) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Standard != b.Standard {
			return a.Standard < b.Standard
		}
		if a.Clause != b.Clause {
			return a.Clause < b.Clause
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Context < b.Context
	})
}

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

// SourceOrigin returns the recorded source block coordinate, or zero when no origin contract is recorded.
// TS 44.060 V19.0.0 sections 10.3.1, 10.3.2 and 11.2.0.1-11.2.0.2.
// Descriptor DecodeBits and DecodeBitsWithReport position content there.
// DecodeFrom accepts a caller-owned bounded reader at the same octet phase;
// descriptor byte functions supply the recorded coordinate.
func SourceOrigin(d runtime.Descriptor) int {
	switch {
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.1" && d.Name == "Packet Access Reject message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.11" && d.Name == "Packet PDCH Release message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.12" && d.Name == "Packet Polling Request message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.13" && d.Name == "Packet Power Control/Timing Advance message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.14" && d.Name == "Packet PRACH Parameters message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.15" && d.Name == "Packet Queueing Notification message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.16" && d.Name == "Packet Resource Request message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.17" && d.Name == "Packet PSI Status message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.17a" && d.Name == "Packet Serving Cell Data message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.17b" && d.Name == "Packet SI Status message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.17c" && d.Name == "Packet Serving Cell SI message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.18" && d.Name == "PSI1 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.19" && d.Name == "PSI2 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.2" && d.Name == "Packet Control Acknowledgement message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.20" && d.Name == "PSI3 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.21a" && d.Name == "PSI3 ter message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.21b" && d.Name == "PSI3 quater message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.24" && d.Name == "PSI8 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.25" && d.Name == "PSI13 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.25a" && d.Name == "PSI14 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.25b" && d.Name == "PSI15 message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.26" && d.Name == "Packet TBF Release message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.29" && d.Name == "Packet Uplink Assignment message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.2a" && d.Name == "Packet Cell Change Continue message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.3" && d.Name == "Packet Cell Change Failure message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.30a" && d.Name == "Packet pause message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.31" && d.Name == "Packet Timeslot Reconfigure message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.32" && d.Name == "Additional MS Radio Access Capabilities message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.35" && d.Name == "Packet CS Request message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.38" && d.Name == "MBMS service request message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.39" && d.Name == "MBMS Assignment Non-distribution message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.3a" && d.Name == "Packet Cell Change Notification message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.45" && d.Name == "Packet Physical information message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.5b" && d.Name == "Packet DBPSCH Assignment message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.6" && d.Name == "Packet Downlink Ack/Nack message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.6a" && d.Name == "EGPRS Packet Downlink Ack/Nack message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.8" && d.Name == "Packet Downlink Dummy Control Block message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.8b" && d.Name == "Packet Uplink Dummy Control Block message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.9" && d.Name == "Packet Measurement Report message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.9b" && d.Name == "Packet Measurement Order message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.9c" && d.Name == "Packet Mobile TBF Status message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.9d" && d.Name == "PACKET ENHANCED MEASUREMENT REPORT message content":
		return 14
	case d.Standard == "TS 44.060" && d.Version == "19.0.0" && d.Clause == "11.2.9e" && d.Name == "Packet Neighbour Cell Data message content":
		return 14
	}
	return 0
}
