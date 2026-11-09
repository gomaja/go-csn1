// Package registry resolves published CSN.1 definition names and clauses.
// Each module carries one version per standard; version is descriptor metadata,
// not part of a lookup key. Consumers update with main rather than select a
// historical version through this registry.
package registry

import (
	"fmt"
	"sort"
	"sync"

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
	"github.com/gomaja/go-csn1/ts44060/dtmhandover2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover3"
	"github.com/gomaja/go-csn1/ts44060/ecmessages"
	"github.com/gomaja/go-csn1/ts44060/egprsmode2"
	"github.com/gomaja/go-csn1/ts44060/eutrancsgtarget"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/mbmsassignment"
	"github.com/gomaja/go-csn1/ts44060/mbmsmessages"
	"github.com/gomaja/go-csn1/ts44060/measurementreport"
	ts44060messages "github.com/gomaja/go-csn1/ts44060/messages"
	"github.com/gomaja/go-csn1/ts44060/multipledownlink"
	"github.com/gomaja/go-csn1/ts44060/multipleuplinkassignment2"
	"github.com/gomaja/go-csn1/ts44060/neighbourcelldata"
	"github.com/gomaja/go-csn1/ts44060/nongprsoptions"
	"github.com/gomaja/go-csn1/ts44060/packetcsreleaseindication"
	"github.com/gomaja/go-csn1/ts44060/packetdownlinkassignment"
	"github.com/gomaja/go-csn1/ts44060/packetenhancedmeasurement"
	"github.com/gomaja/go-csn1/ts44060/packetmeasurementorder"
	"github.com/gomaja/go-csn1/ts44060/packetsysteminformationtype3"
	"github.com/gomaja/go-csn1/ts44060/packettimeslotreconfigure"
	"github.com/gomaja/go-csn1/ts44060/packetuplinkassignment"
	"github.com/gomaja/go-csn1/ts44060/padding"
	"github.com/gomaja/go-csn1/ts44060/pdchpairs"
	"github.com/gomaja/go-csn1/ts44060/pshandover"
	"github.com/gomaja/go-csn1/ts44060/pshandover2"
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
	out = append(out, uecapability.Descriptors()...)
	out = append(out, dtmhandover2.Descriptors()...)
	out = append(out, dtmhandover3.Descriptors()...)
	out = append(out, egprsmode2.Descriptors()...)
	out = append(out, multipleuplinkassignment2.Descriptors()...)
	out = append(out, packetcsreleaseindication.Descriptors()...)
	out = append(out, pshandover2.Descriptors()...)

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
	return catalogueLookup().lookup(standard, name)
}

// LookupClause resolves the exact specification, clause and printed name.
// It rejects names with more than one publication context.
func LookupClause(standard, clause, name string) (runtime.Descriptor, error) {
	return catalogueLookup().lookupClause(standard, clause, name)
}

// LookupContext resolves the exact specification, clause, printed name and opaque
// publication context. The empty context selects an unqualified definition.
func LookupContext(standard, clause, name, context string) (runtime.Descriptor, error) {
	return catalogueLookup().lookupContext(standard, clause, name, context)
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
func Descriptors() []runtime.Descriptor { return append([]runtime.Descriptor(nil), catalogue()...) }

var catalogue = sync.OnceValue(func() []runtime.Descriptor {
	entries := descriptors()
	sortDescriptors(entries)
	return entries
})

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

// SourceOrigin returns the recorded source block coordinate, or zero when no origin contract is recorded.
// TS 44.060 V19.0.0 sections 10.3.1, 10.3.2 and 11.2.0.1-11.2.0.2.
// Descriptor DecodeBits and DecodeBitsWithReport position content there.
// DecodeFrom accepts a caller-owned bounded reader at the same octet phase;
// descriptor byte functions supply the recorded coordinate.
func SourceOrigin(d runtime.Descriptor) int {
	index := catalogueLookup()
	position, ok := index.clauses[clauseKey{d.Standard, d.Clause, d.Name}]
	if !ok || index.entries[position].Version != d.Version {
		return 0
	}
	return index.entries[position].SourceOrigin
}
