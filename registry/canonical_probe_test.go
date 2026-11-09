package registry

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44060/access"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
	"github.com/gomaja/go-csn1/ts44060/celldata"
	"github.com/gomaja/go-csn1/ts44060/containers"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"github.com/gomaja/go-csn1/ts44060/dbpschuplinkack"
	"github.com/gomaja/go-csn1/ts44060/dbpschuplinkack2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover3"
	"github.com/gomaja/go-csn1/ts44060/ecmessages"
	"github.com/gomaja/go-csn1/ts44060/mbmsassignment"
	"github.com/gomaja/go-csn1/ts44060/mbmsmessages"
	"github.com/gomaja/go-csn1/ts44060/measurementreport"
	"github.com/gomaja/go-csn1/ts44060/neighbourcelldata"
	"github.com/gomaja/go-csn1/ts44060/packetcsreleaseindication"
	"github.com/gomaja/go-csn1/ts44060/packetmeasurementorder"
	"github.com/gomaja/go-csn1/ts44060/packetsysteminformationtype3"
	"github.com/gomaja/go-csn1/ts44060/packettimeslotreconfigure"
	"github.com/gomaja/go-csn1/ts44060/packetuplinkassignment"
	"github.com/gomaja/go-csn1/ts44060/padding"
	"github.com/gomaja/go-csn1/ts44060/pshandoverradioresources3"
	"github.com/gomaja/go-csn1/ts44060/psi"
	"github.com/gomaja/go-csn1/ts44060/psilegacy"
	"github.com/gomaja/go-csn1/ts44060/psineighbours"
	"github.com/gomaja/go-csn1/ts44060/servingcelldata"
)

// TestCanonicalAllDefinitions exercises every generated entry point with
// deterministic synthetic valid and invalid input. TS 44.018 V19.0.0 §8.9
// permits truncated rest-octet prefixes; §9.1.55 permits only redundant
// trailing no-report bitmap positions to change during canonical encoding.
func TestCanonicalAllDefinitions(t *testing.T) {
	var vectors [][]byte
	for _, length := range []int{0, 1, 2, 3, 4, 5, 8, 11, 17, 20, 23, 50} {
		for _, fill := range []byte{0, 0x2b, 0xff} {
			vectors = append(vectors, bytes.Repeat([]byte{fill}, length))
		}
	}
	vectors = append(vectors,
		[]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		[]byte{0x19, 0x30, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	)
	defs := descriptors()
	var accepted, rejected, failures int
	for _, definition := range defs {
		if definition.Canonical == nil {
			t.Errorf("%s §%s <%s>: missing canonical descriptor", definition.Standard, definition.Clause, definition.Name)
			failures++
			continue
		}
		for index, vector := range vectors {
			ok, err := probeCanonical(definition, vector)
			if err != nil {
				t.Errorf("%s §%s <%s> vector %d: %v", definition.Standard, definition.Clause, definition.Name, index, err)
				failures++
			} else if ok {
				accepted++
			} else {
				rejected++
			}
		}
		for _, dual := range []uint64{0, 1} {
			contextual, selected := canonicalDLDCDescriptor(definition, dual)
			if !selected {
				continue
			}
			for index, vector := range vectors {
				ok, err := probeCanonical(contextual, vector)
				if err != nil {
					t.Errorf("%s <%s> DLDC=%d vector %d: %v", definition.Standard, definition.Name, dual, index, err)
					failures++
				} else if ok {
					accepted++
				} else {
					rejected++
				}
			}
		}
		if definition.DecodeWithContext != nil {
			if definition.CanonicalWithContext == nil {
				t.Errorf("%s <%s>: missing canonical context descriptor", definition.Standard, definition.Name)
				failures++
				continue
			}
			for _, context := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
				contextDefinition := definition
				contextDefinition.Decode = func(b []byte) (any, error) { return definition.DecodeWithContext(b, context) }
				contextDefinition.Encode = func(v any) ([]byte, error) { return definition.EncodeWithContext(v, context) }
				contextDefinition.Canonical = func(v any) ([]byte, error) { return definition.CanonicalWithContext(v, context) }
				contextDefinition.CanonicalAtLength = nil
				contextDefinition.MaxOctets = 20
				ok, err := probeCanonical(contextDefinition, bytes.Repeat([]byte{0x2b}, 20))
				if err != nil || !ok {
					t.Errorf("%s <%s> ACS=%d: accepted=%v error=%v", definition.Standard, definition.Name, context, ok, err)
					failures++
				} else {
					accepted++
				}
			}
		}
	}
	t.Logf("definitions=%d vectors=%d accepted=%d rejected=%d failures=%d", len(defs), len(vectors), accepted, rejected, failures)
	if len(defs) != 656 || len(vectors) != 38 {
		t.Fatalf("unexpected probe breadth: %d definitions and %d vectors", len(defs), len(vectors))
	}
}

func probeCanonical(definition runtime.Descriptor, wire []byte) (accepted bool, err error) {
	decode, plain, encode := definition.Decode, definition.Encode, definition.Canonical
	atLength, maxOctets := definition.CanonicalAtLength, definition.MaxOctets
	defer func() {
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("panic: %v", panicValue)
		}
	}()
	decoded, err := decode(wire)
	if err != nil {
		return false, nil
	}
	value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
	if plain == nil {
		return true, fmt.Errorf("missing plain encoder")
	}
	replay, replayErr := plain(value)
	if replayErr != nil {
		return true, fmt.Errorf("plain encode: %w", replayErr)
	}
	if !bytes.Equal(replay, wire) {
		return true, fmt.Errorf("plain encoding changed received bytes: got %x, want %x", replay, wire)
	}
	canonical, err := encode(value)
	regionRefused := false
	senderRefused := false
	var extent *runtime.ExtentError
	if errors.As(err, &extent) && extent.Required && atLength != nil {
		// These component definitions have no standalone source maximum.
		// Supply an extent explicitly, first using the fresh value's own
		// length, then the caller's input extent as a search ceiling.
		fresh, freshErr := plain(runtime.Canonical(value))
		if freshErr == nil {
			canonical, err = atLength(value, len(fresh))
			regionRefused = canonicalRegionRefusal(value, err)
			senderRefused = canonicalContainerReservation(definition, err) || canonicalReleaseCauseReservation(definition, value, err)
		}
		if err != nil {
			for octets := 0; octets <= len(wire); octets++ {
				candidate, candidateErr := atLength(value, octets)
				regionRefused = regionRefused || canonicalRegionRefusal(value, candidateErr)
				senderRefused = senderRefused || canonicalContainerReservation(definition, candidateErr) || canonicalReleaseCauseReservation(definition, value, candidateErr)
				if candidateErr == nil {
					canonical, err = candidate, nil
					break
				}
			}
		}
	}
	if err != nil {
		if canonicalReservedRemainderRefusal(definition, wire, value, err) {
			return true, nil
		}
		if canonicalReservedDesignationRefusal(definition, wire, value, err) {
			return true, nil
		}
		if regionRefused || senderRefused {
			return true, nil
		}
		// TS 44.060 V19.0.0 §11 p12937 supplies these roots with an
		// external extent. A zero-filled received suffix may be shorter than
		// every complete sender encoding (§§11.2.5b,11.2.34,11.2.45).
		var required *runtime.ExtentError
		switch definition.Clause {
		case "11.2.3", "11.2.3a", "11.2.5b", "11.2.13", "11.2.32", "11.2.34", "11.2.45":
			if callerBoundedControlComponent(definition) && atLength != nil && errors.As(err, &required) && required.Required {
				return true, nil
			}
		}
		if canonicalContainingExtentRefusal(definition, err) {
			return true, nil
		}
		if canonicalCapabilityDefinition(definition) && canonicalCapabilityReservation(err) {
			return true, nil
		}
		return true, fmt.Errorf("encode: %w", err)
	}
	if maxOctets > 0 && len(canonical) > maxOctets {
		return true, fmt.Errorf("canonical output %d exceeds source maximum %d octets", len(canonical), maxOctets)
	}
	again, err := decode(canonical)
	if err != nil {
		return true, fmt.Errorf("decode canonical: %w", err)
	}
	other := reflect.ValueOf(again).FieldByName("Value").Interface()
	if a, ok := value.(measurement.EnhancedMeasurementReport); ok {
		b := other.(measurement.EnhancedMeasurementReport)
		return true, compareEMRProbe(a, b)
	}
	// TS 44.060 V19.0.0 §12.30 p23547 changes only the two nested
	// length fields when received spare repetitions are suppressed.
	if definition.Standard == "TS 44.060" && ((definition.Clause == "12.30" && definition.Name == "MS Radio Access Capability 2 IE") || (definition.Clause == "11.2.32" && definition.Name == "Additional MS Radio Access Capabilities message content") || (definition.Clause == "11.2.16" && definition.Name == "Packet Resource Request message content")) {
		value = capabilitySemanticValue(value)
		other = capabilitySemanticValue(other)
	}
	if !runtime.SemanticallyEqual(value, other) {
		return true, fmt.Errorf("canonical encoding changed typed semantics")
	}
	return true, nil
}

// TS 24.008 V20.1.0 §§10.5.1.6, 10.5.1.7 and 10.5.5.12a
// preserve received capability codes while their canonical sender encodings
// reject reserved source-table values. GERAN containers delegate to those IEs
// (TS 36.331 V19.4.0 UE-CapabilityRAT-ContainerList field descriptions).
func canonicalCapabilityDefinition(definition runtime.Descriptor) bool {
	switch definition.Standard + "/" + definition.Clause {
	case "TS 24.008/10.5.1.6":
		return definition.Name == "Mobile Station Classmark 2 value part"
	case "TS 24.008/10.5.1.7":
		switch definition.Name {
		case "Classmark 3 Value part", "HSCSD Multi Slot Capability", "8-PSK Struct", "Single Band Support":
			return true
		}
	case "TS 24.008/10.5.5.12a":
		switch definition.Name {
		case "MS RA capability value part", "MS RA capability value part struct", "Access capabilities struct", "Content", "Multislot capability struct", "Enhanced Flexible Timeslot Assignment struct", "DLMC Capability struct":
			return true
		}
	case "TS 44.060/12.30":
		return definition.Name == "MS Radio Access Capability 2 IE"
	case "TS 44.060/11.2.16":
		return definition.Name == "Packet Resource Request message content"
	case "TS 36.331/UE-CapabilityRAT-ContainerList field descriptions":
		return definition.Name == "geran-cs" || definition.Name == "geran-ps"
	}
	return false
}

// Sender constraints retain their exact cause through the located operation
// wrapper; only the authenticated capability reservation markers qualify.
func canonicalCapabilityReservation(err error) bool {
	var operation *runtime.OperationError
	if errors.As(err, &operation) {
		if operation.Kind != runtime.InvalidValue {
			return false
		}
		if operation.Cause == nil {
			switch operation.Detail {
			case "value reserved by source table: TS 24.008 table 10.5.6a Revision level", "value reserved by source table: TS 24.008 table 10.5.6a RF Power Capability":
				return true
			}
			return false
		}
		return canonicalCapabilityReservation(operation.Cause)
	}
	switch err.Error() {
	case "value reserved by source table", "value reserved by source table: TS 24.008 table 10.5.6a Revision level", "value reserved by source table: TS 24.008 table 10.5.6a RF Power Capability":
		return true
	}
	return false
}

// TestProbeCanonicalCapabilityReservation keeps the receiver replay oracle
// active before allowing a sender-side capability reservation.
func TestProbeCanonicalCapabilityReservation(t *testing.T) {
	var definition runtime.Descriptor
	for _, candidate := range descriptors() {
		if candidate.Standard == "TS 24.008" && candidate.Clause == "10.5.1.6" {
			definition = candidate
			break
		}
	}
	if definition.Decode == nil {
		t.Fatal("missing Classmark 2 descriptor")
	}
	wire := []byte{0, 0, 0} // TS 24.008 table 10.5.6a: revision level 0.
	accepted, err := probeCanonical(definition, wire)
	if err != nil || !accepted {
		t.Fatalf("reserved receiver capability: accepted=%v error=%v", accepted, err)
	}
	for _, tc := range []struct {
		name   string
		change func(*runtime.Descriptor)
		want   string
	}{
		{"unrelated definition", func(d *runtime.Descriptor) { d.Standard = "TS 44.018" }, "encode:"},
		{"unaffected capability definition", func(d *runtime.Descriptor) { d.Name = "A5 bits" }, "encode:"},
		{"unexpected canonical error", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("unexpected") }
		}, "encode: unexpected"},
		{"reservation prefix lookalike", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) { return nil, errors.New("value reserved by source table but unexpected") }
		}, "but unexpected"},
		{"unexpected value error", func(d *runtime.Descriptor) {
			d.Canonical = func(any) ([]byte, error) {
				return nil, &runtime.DecodeError{Kind: runtime.InvalidValue, Detail: "unexpected value"}
			}
		}, "unexpected value"},
		{"plain replay failure", func(d *runtime.Descriptor) {
			d.Encode = func(any) ([]byte, error) { return nil, errors.New("plain failure") }
		}, "plain encode: plain failure"},
		{"plain replay mismatch", func(d *runtime.Descriptor) { d.Encode = func(any) ([]byte, error) { return []byte{0}, nil } }, "plain encoding changed received bytes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := definition
			tc.change(&candidate)
			_, err := probeCanonical(candidate, wire)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want failure containing %q", err, tc.want)
			}
		})
	}
}

func compareEMRProbe(a, b measurement.EnhancedMeasurementReport) error {
	// TS 44.018 V19.0.0 §9.1.55: absent pre-Rel-8 no-report
	// positions are redundant when the bitmap is present (table 9.1.55.1).
	normalize := func(v measurement.EnhancedMeasurementReport) measurement.EnhancedMeasurementReport {
		v = runtime.Canonical(v)
		if v.REPORTINGQUANTITYList != nil {
			list := *v.REPORTINGQUANTITYList
			for len(list) > 0 && list[len(list)-1] == nil {
				list = list[:len(list)-1]
			}
			if len(list) == 0 {
				list = nil
			}
			v.REPORTINGQUANTITYList = &list
		}
		// Figure 9.1.55.1: only null and L have no release fields.
		if v.BITMAPLENGTHChoice.NoString != nil {
			v.BITMAPLENGTHChoice = measurement.EnhancedMeasurementReportBITMAPLENGTHChoice{Alternative: measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}
		}
		if rel8 := v.BITMAPLENGTHChoice.BITMAPLENGTH; rel8 != nil {
			if rel8.UTRANCSGMeasurementReportChoice.NoString != nil {
				rel8.UTRANCSGMeasurementReportChoice = measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoice{Alternative: measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}
			}
			if rel9 := rel8.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport; rel9 != nil && rel9.SI23BAUSEDChoice.NoString != nil {
				rel9.SI23BAUSEDChoice = measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoice{Alternative: measurement.EnhancedMeasurementReportBITMAPLENGTHChoiceBITMAPLENGTHUTRANCSGMeasurementReportChoiceUTRANCSGMeasurementReportSI23BAUSEDChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}
			}
		}
		return v
	}
	if !runtime.SemanticallyEqual(normalize(a), normalize(b)) {
		return fmt.Errorf("canonical EMR changed typed semantics")
	}
	return nil
}

func FuzzCanonicalAllDefinitions(f *testing.F) {
	for _, seed := range []struct {
		index uint16
		wire  []byte
	}{
		{0, []byte{0}}, {12, []byte{0}}, {27, []byte{0x10, 0x02}},
		{100, bytes.Repeat([]byte{0x2b}, 20)}, {200, []byte{0xff, 0xff}},
	} {
		f.Add(seed.index, seed.wire)
	}
	definitions := sampleDescriptors(f, descriptors(), "direct-probe-order.tsv")
	f.Fuzz(func(t *testing.T, index uint16, wire []byte) {
		if len(wire) > 50 {
			return
		}
		definition := definitions[int(index)%len(definitions)]
		if definition.Canonical == nil {
			t.Fatalf("missing canonical entry point for %s", definition.Name)
		}
		_, err := probeCanonical(definition, wire)
		if err != nil {
			t.Fatalf("%s §%s <%s>: %v", definition.Standard, definition.Clause, definition.Name, err)
		}
	})
}

// TS 44.060 V19.0.0 §11 paragraph 12938 and §11.2.8 paragraph 15411
// define receiver error arms. Their received bits replay, while a fresh sender
// must select a source-valid semantic arm rather than transmit an error region.
func canonicalRegionRefusal(value any, err error) bool {
	received := false
	switch v := value.(type) {
	case ecmessages.PacketTBFReleaseMessageContent:
		received = v.ReceiverRegionEnds()
	case packetuplinkassignment.PacketUplinkAssignmentMessageContent, dtmhandover2.DTMHandoverPSRadioResources2IE, dtmhandover3.DTMHandoverPSRadioResources3IE, packetcsreleaseindication.PacketCSReleaseMessageContent:
		received = v.(interface{ ReceiverRegionEnds() bool }).ReceiverRegionEnds()
	case dbpschuplinkack2.PacketDBPSCHUplinkAckNackMessageContent:
		received = v.MESSAGETYPEGroupFallback.Ignored != nil
		if known := v.MESSAGETYPEGroupFallback.Known; known != nil {
			p := known.PaddingBits
			received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
		}
	case packetmeasurementorder.PacketMeasurementOrderMessageContent:
		received = v.ReceiverRegionEnds()
	case packettimeslotreconfigure.PacketTimeslotReconfigureMessageContent, pshandoverradioresources3.PSHandoverRadioResources3IE:
		received = v.(interface{ ReceiverRegionEnds() bool }).ReceiverRegionEnds()
	// TS 44.060 V19.0.0 §11.2.39: the selected non-distribution assignment receiver arms.
	case mbmsassignment.MBMSAssignmentNonDistributionMessageContent:
		received = v.ReceiverRegionEnds()
	case dbpschuplinkack.PacketDBPSCHUplinkAckNackMessageContent:
		received = v.MESSAGETYPEGroupFallback.Ignored != nil
		if known := v.MESSAGETYPEGroupFallback.Known; known != nil {
			p := known.PaddingBits
			received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
		}
	// TS 44.060 V19.0.0: selected typed content receiver regions.
	case ackmessages.PacketDownlinkAckNackMessageContent:
		received = v.ReceiverRegionEnds()
	// TS 44.060 V19.0.0: selected typed content receiver regions.
	case mbmsmessages.MBMSServiceRequestMessageContent:
		received = v.ReceiverRegionEnds()
	// TS 44.060 V19.0.0: selected typed content receiver regions.
	case psineighbours.PSI3QuaterMessageContent:
		received = v.ReceiverRegionEnds()
	// TS 44.060 V19.0.0: selected typed content receiver regions.
	case psi.PSI1MessageContent, psi.PSI2MessageContent, psi.PSI3TerMessageContent, psi.PSI13MessageContent, psi.PSI14MessageContent, psi.PSI15MessageContent:
		received = psiSelectedReceiverRegion(reflect.ValueOf(value))
	case psilegacy.PSI8MessageContent:
		received = v.PAGEMODEGroupFallback.Ignored != nil
		if known := v.PAGEMODEGroupFallback.Known; known != nil {
			if additions := known.DynamicARFCNMappingDescriptionChoice.DynamicARFCNMappingDescription; additions != nil {
				p := additions.PaddingBits
				received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
			}
		}
	case packetsysteminformationtype3.PSI3MessageContent:
		received = v.PAGEMODEGroupFallback.Ignored != nil
		if known := v.PAGEMODEGroupFallback.Known; known != nil {
			if r98 := known.ServingCellLSAIDInformationChoice.ServingCellLSAIDInformation; r98 != nil {
				if r99 := r98.COMPACTInformationChoice.COMPACTInformation; r99 != nil {
					if r4 := r99.CCNSupportDescriptionChoice.CCNSupportDescription; r4 != nil {
						if r5 := r4.CELLBARQUALIFY3Choice.CELLBARQUALIFY3; r5 != nil {
							p := r5.PaddingBits
							received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
						}
					}
				}
			}
		}
	case access.PacketControlAcknowledgementMessageContent:
		if r5 := v.TNRRBPChoice.TNRRBP; r5 != nil {
			if r6 := r5.CTRLACKEXTENSIONChoice.CTRLACKEXTENSION; r6 != nil {
				p := r6.PaddingBits
				received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
			}
		}
	case controlmessages.PacketMobileTBFStatusMessageContent, controlmessages.PacketPollingRequestMessageContent, controlmessages.PacketPSIStatusMessageContent, celldata.PacketSIStatusMessageContent, measurementreport.PacketMeasurementReportMessageContent, controlmessages.PacketAccessRejectMessageContent, controlmessages.RejectStruct, controlmessages.PacketUplinkDummyControlBlockMessageContent, controlmessages.PacketCellChangeContinueMessageContent, controlmessages.PacketPDCHReleaseMessageContent, controlmessages.PacketPRACHParametersMessageContent, controlmessages.PacketQueueingNotificationMessageContent:
		received = v.(interface{ ReceiverRegionEnds() bool }).ReceiverRegionEnds()
	case controlmessages.PacketPowerControlTimingAdvanceMessageContent:
		received = v.ReceiverRegionEnds()
	case controlmessages.PacketCellChangeFailureMessageContent:
		received = v.ReceiverRegionEnds()
	case controlmessages.PhysicalInformationMessageContent:
		received = v.ReceiverRegionEnds()
	case controlmessages.PacketCellChangeNotificationMessageContent:
		received = v.ReceiverRegionEnds()
	case controlmessages.PacketPhysicalInformationMessageContent:
		received = v.ReceiverRegionEnds()
	case controlmessages.PacketDBPSCHAssignmentMessageContent:
		received = v.ReceiverRegionEnds()
	case neighbourcelldata.PacketNeighbourCellDataMessageContent, servingcelldata.PacketServingCellDataMessageContent:
		received = v.(interface{ ReceiverRegionEnds() bool }).ReceiverRegionEnds()
	case padding.PaddingBits:
		received = v.IgnoreChoice.Ignore != nil && v.IgnoreChoice.Ignore.Ignored != nil
	case controlmessages.PacketPauseMessageContent:
		p := v.PaddingBits
		received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
	case controlmessages.PacketCSRequestMessageContent:
		p := v.PaddingBits
		received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
	case ackmessages.EGPRSPacketDownlinkAckNackMessageContent:
		p := v.PaddingBits
		received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
	case celldata.PacketServingCellSIMessageContent:
		received = v.PAGEMODEGroupFallback.Ignored != nil
		if known := v.PAGEMODEGroupFallback.Known; known != nil {
			p := known.PaddingBits
			received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
		}
	case containers.NASContainerForPSHandoverIE:
		p := v.PaddingBits
		received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
	case containers.CSHandoverRadioResourcesIE:
		p := v.PaddingBits
		received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
	case controlmessages.PacketDownlinkDummyControlBlockMessageContent:
		received = v.PAGEMODEGroupFallback.Ignored != nil
		if known := v.PAGEMODEGroupFallback.Known; known != nil {
			p := known.PaddingBits
			received = p.IgnoreChoice.Ignore != nil && p.IgnoreChoice.Ignore.Ignored != nil
		}

	}
	if !received {
		return false
	}
	for err != nil {
		if operation, ok := err.(*runtime.OperationError); ok && operation.Kind == runtime.InvalidValue {
			message := operation.Detail
			if operation.Cause != nil {
				message = operation.Cause.Error()
			}
			if message == "receiver error region has no canonical sender encoding" {
				return true
			}
		}
		err = errors.Unwrap(err)
	}
	return false
}

func TestCanonicalRegionRefusalIsSpecific(t *testing.T) {
	region := padding.PaddingBits{IgnoreChoice: padding.PaddingBitsIgnoreChoice{Alternative: padding.PaddingBitsIgnoreChoiceAlternativeIgnore, Ignore: &padding.PaddingBitsIgnoreChoiceIgnore{Alternative: padding.PaddingBitsIgnoreChoiceIgnoreAlternativeIgnored, Ignored: &padding.PaddingBitsIgnoreChoiceIgnoreIgnoredContent{}}}}
	expected := &runtime.OperationError{Kind: runtime.InvalidValue, Detail: "receiver error region has no canonical sender encoding"}
	for _, value := range []any{
		region,
		ackmessages.EGPRSPacketDownlinkAckNackMessageContent{PaddingBits: region},
		celldata.PacketServingCellSIMessageContent{PAGEMODEGroupFallback: celldata.PacketServingCellSIMessageContentPAGEMODEGroupFallback{Ignored: &runtime.ErrorRegion{}}},
		celldata.PacketServingCellSIMessageContent{PAGEMODEGroupFallback: celldata.PacketServingCellSIMessageContentPAGEMODEGroupFallback{Known: &celldata.PacketServingCellSIMessageContentPAGEMODEGroupFallbackKnown{PaddingBits: region}}},
	} {
		if !canonicalRegionRefusal(value, fmt.Errorf("wrapped: %w", expected)) {
			t.Fatalf("%T located receiver refusal not recognized", value)
		}
	}
	withCause := &runtime.OperationError{Kind: runtime.InvalidValue, Cause: errors.New(expected.Detail)}
	if !canonicalRegionRefusal(region, withCause) {
		t.Fatal("located receiver cause not recognized")
	}

	for _, tc := range []struct {
		value any
		err   error
	}{
		{padding.PaddingBits{}, expected},
		{ackmessages.EGPRSPacketDownlinkAckNackMessageContent{}, expected},
		{celldata.PacketServingCellSIMessageContent{}, expected},
		{struct{}{}, expected},
		{region, nil},
		{region, errors.New("receiver error region has no canonical sender encoding")},
		{region, &runtime.OperationError{Kind: runtime.InvalidValue, Detail: "different failure"}},
		{region, &runtime.OperationError{Kind: runtime.InvalidSchema, Detail: expected.Detail}},
	} {
		if canonicalRegionRefusal(tc.value, tc.err) {
			t.Fatalf("unexpected refusal accepted: %T %v", tc.value, tc.err)
		}
	}
}

// TS 44.060 V19.0.0 §§11.2.9e pp16187–16197 and 11.2.17a
// pp16647–16657: received reserved PD/CD_LENGTH codes replay, while fresh
// and canonical sends use the assigned code set.
func canonicalContainerReservation(d runtime.Descriptor, err error) bool {
	if d.Standard != "TS 44.060" || d.Clause != "11.2.9e" && d.Clause != "11.2.17a" {
		return false
	}
	switch d.Name {
	case "Container repetition struct", "Packet Neighbour Cell Data message content", "Packet Serving Cell Data message content":
	default:
		return false
	}
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || operation.Operation != "canonical" || operation.Kind != runtime.InvalidValue || operation.Location.Source.Standard != d.Standard || operation.Location.Source.Version != d.Version || operation.Location.Source.Clause != d.Clause {
		return false
	}
	for operation.Cause != nil {
		inner, ok := operation.Cause.(*runtime.OperationError)
		if !ok {
			return operation.Cause.Error() == "value outside sender code set"
		}
		if inner.Kind != runtime.InvalidValue {
			return false
		}
		operation = inner
	}
	return operation.Detail == "value outside sender code set"
}

// TS 44.060 V19.0.0 §§11.1–11.1.3.2 and §§12.5.5, 12.48a.2, 12.48a.3, 12.48a.5: the reserved
// designation is a receiver-only error region, never a canonical sender arm.
func canonicalReservedDesignationRefusal(d runtime.Descriptor, wire []byte, value any, err error) bool {
	var extent *runtime.ExtentError
	selected := map[string]string{"12.5.5": "PDCH Pairs Description IE", "12.48a.2": "Single Downlink Assignment 2 IE", "12.48a.3": "Single Uplink Assignment 2 IE", "12.48a.5": "Multiple Downlink Assignment 2 IE"}
	selected["12.48a.6"] = "Multiple Uplink Assignment 2 IE"
	selected["12.48a.1"] = "EGPRS mode 2 IE"
	// TS 44.060 V19.0.0 §12.42a p23904 defines this receiver designation.
	selected["12.42a"] = "PS Handover Radio Resources 2 IE"
	// TS 44.060 V19.0.0 §§12.48 p24544 and 12.64 p25616.
	selected["12.48"] = "DTM Handover PS Radio Resources 2 IE"
	selected["12.64"] = "DTM Handover PS Radio Resources 3 IE"
	if (d.Standard != "TS 44.060" || selected[d.Clause] != d.Name) && (d.Standard != "TS 44.018" || d.Clause != "10.5.2.25e" || d.Name != "RR Packet Downlink Assignment Type 2 value part") || !errors.As(err, &extent) || !extent.Required {
		return false
	}
	_, _, report, decodeErr := d.DecodeBitsWithReport(wire, len(wire)*8)
	if decodeErr != nil || len(report.Regions) != 1 {
		return false
	}
	label := report.Regions[0].Label
	allowed := label == runtime.PDCHPairsConfigurationError
	if d.Standard == "TS 44.060" {
		allowed = allowed || (d.Clause == "12.48" || d.Clause == "12.64") && label == runtime.TBFModeError
		allowed = allowed || d.Clause == "12.42a" && label == runtime.FrequencyParametersError
	}
	if !allowed {
		return false
	}
	_, refusal := d.CanonicalAtLength(value, len(wire))
	var operation *runtime.OperationError
	if !errors.As(refusal, &operation) || operation.Kind != runtime.InvalidValue {
		return false
	}
	for refusal != nil {
		if refusal.Error() == "receiver error region has no canonical sender encoding" {
			return true
		}
		refusal = errors.Unwrap(refusal)
	}
	return false
}

func TestProbeCanonicalReservedDesignation(t *testing.T) {
	d, err := Lookup("TS 44.060", "PDCH Pairs Description IE")
	if err != nil {
		t.Fatal(err)
	}
	if accepted, err := probeCanonical(d, []byte{0x75}); !accepted || err != nil {
		t.Fatalf("receiver-only designation: accepted=%t error=%v", accepted, err)
	}
	for _, change := range []func(*runtime.Descriptor){
		func(d *runtime.Descriptor) { d.Clause = "12.5" },
		func(d *runtime.Descriptor) {
			d.CanonicalAtLength = func(any, int) ([]byte, error) {
				return nil, &runtime.OperationError{Kind: runtime.InvalidValue, Cause: errors.New("unexpected")}
			}
		},
	} {
		altered := d
		change(&altered)
		if _, err := probeCanonical(altered, []byte{0x75}); err == nil {
			t.Fatal("unrelated canonical refusal was accepted")
		}
	}
}

func TestProbeCanonicalAssignmentDesignations(t *testing.T) {
	for _, tc := range []struct {
		clause, name string
		wire         []byte
	}{
		{"12.48a.2", "Single Downlink Assignment 2 IE", []byte{74, 192}},
		{"12.48a.5", "Multiple Downlink Assignment 2 IE", []byte{74, 192}},
		{"12.48a.3", "Single Uplink Assignment 2 IE", []byte{57, 86}},
	} {
		d, err := LookupClause("TS 44.060", tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if accepted, err := probeCanonical(d, tc.wire); !accepted || err != nil {
			t.Fatalf("%s: accepted=%t error=%v", tc.name, accepted, err)
		}
		d.Name += " unrelated"
		if _, err := probeCanonical(d, tc.wire); err == nil {
			t.Fatal("unrelated descriptor accepted reserved canonical refusal")
		}
	}
}

// capabilitySemanticValue independently excludes the two transmitted length
// fields; every other typed field remains in the semantic comparison.
func capabilitySemanticValue(value any) any {
	v := reflect.New(reflect.TypeOf(value))
	v.Elem().Set(reflect.ValueOf(runtime.Canonical(value)))
	var walk func(reflect.Value)
	walk = func(x reflect.Value) {
		if x.Type() == reflect.TypeFor[msrac.MSRACapabilityValuePartStruct]() {
			cap := x.Interface().(msrac.MSRACapabilityValuePartStruct)
			for i := range cap.Entries {
				a := &cap.Entries[i].AccessTechnologyTypeChoice
				if a.AccessCapabilities != nil {
					a.AccessCapabilities.AccessCapabilities.Length = 0
				}
				if a.Length != nil {
					a.Length.Length = 0
				}
			}
			x.Set(reflect.ValueOf(cap))
			return
		}
		switch x.Kind() {
		case reflect.Pointer:
			if !x.IsNil() {
				walk(x.Elem())
			}
		case reflect.Struct:
			for i := 0; i < x.NumField(); i++ {
				if x.Type().Field(i).Name != "Wire" {
					walk(x.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < x.Len(); i++ {
				walk(x.Index(i))
			}
		}
	}
	walk(v.Elem())
	return v.Elem().Interface()
}
