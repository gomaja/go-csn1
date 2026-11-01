package registry

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ecmessages"
)

// TS 44.060 V19.0.0 table 11.2.26.2 (DOCX paragraph 17910)
// assigns causes 0 and 2; the remaining four-bit values are receiver-only.
// Keep the actual decoded cause and the located sender refusal together.
func canonicalReleaseCauseReservation(d runtime.Descriptor, value any, err error) bool {
	if d.Standard != "TS 44.060" || d.Version != "19.0.0" || d.Clause != "11.2.26" || d.Name != "Packet TBF Release message content" || d.SourceOrigin != 14 {
		return false
	}
	v, ok := value.(ecmessages.PacketTBFReleaseMessageContent)
	if !ok || v.ReceiverRegionEnds() || v.PAGEMODEGroupFallback.Known == nil {
		return false
	}
	address := v.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known
	if address == nil || address.UPLINKRELEASEGroupFallback.Known == nil {
		return false
	}
	cause := address.UPLINKRELEASEGroupFallback.Known.TBFRELEASECAUSE
	if cause == 0 || cause == 2 || cause > 15 {
		return false
	}
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || operation.Operation != "canonical" || operation.Kind != runtime.InvalidValue || operation.Location.Source.Standard != d.Standard || operation.Location.Source.Version != d.Version || operation.Location.Source.Clause != d.Clause || operation.Location.BitOffset != 25 || operation.Location.Width != 4 || !strings.HasSuffix(operation.Location.Path, "/PacketTBFReleaseMessageContentPAGEMODEGroupFallbackKnownGLOBALTFIGroupFallbackKnownUPLINKRELEASEGroupFallbackKnownTBFRELEASECAUSE") {
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

func TestProbeCanonicalReleaseCauseReservation(t *testing.T) {
	definition, err := Lookup("TS 44.060", "Packet TBF Release message content")
	if err != nil {
		t.Fatal(err)
	}
	// These are the existing canonical sweep's final two synthetic vectors.
	vectors := [][]byte{{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, {0x19, 0x30, 0, 0, 0, 0, 0, 0, 0, 0, 0}}
	for i, wire := range vectors {
		decoded, err := definition.Decode(wire)
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		value := reflect.ValueOf(decoded).FieldByName("Value").Interface().(ecmessages.PacketTBFReleaseMessageContent)
		_, refusal := definition.CanonicalAtLength(value, len(wire))
		if !canonicalReleaseCauseReservation(definition, value, refusal) {
			t.Fatalf("vector %d: expected source-table refusal: %v", i, refusal)
		}
		accepted, err := probeCanonical(definition, wire)
		if err != nil || !accepted {
			t.Fatalf("vector %d: accepted=%v err=%v", i, accepted, err)
		}
		for _, change := range []func(*runtime.Descriptor){
			func(d *runtime.Descriptor) { d.Standard = "TS 44.018" },
			func(d *runtime.Descriptor) { d.Version = "0.0.0" },
			func(d *runtime.Descriptor) { d.Clause = "11.2.25" },
			func(d *runtime.Descriptor) { d.Name = "unselected" },
			func(d *runtime.Descriptor) { d.SourceOrigin = 0 },
		} {
			altered := definition
			change(&altered)
			if canonicalReleaseCauseReservation(altered, value, refusal) {
				t.Error("unrelated descriptor admitted")
			}
		}
		for _, cause := range []uint8{0, 2, 16} {
			altered := value
			outer := *value.PAGEMODEGroupFallback.Known
			address := *outer.GLOBALTFIGroupFallback.Known
			release := *address.UPLINKRELEASEGroupFallback.Known
			release.TBFRELEASECAUSE = cause
			address.UPLINKRELEASEGroupFallback.Known = &release
			outer.GLOBALTFIGroupFallback.Known = &address
			altered.PAGEMODEGroupFallback.Known = &outer
			if canonicalReleaseCauseReservation(definition, altered, refusal) {
				t.Errorf("non-reserved cause %d admitted", cause)
			}
		}
		var operation *runtime.OperationError
		if !errors.As(refusal, &operation) {
			t.Fatal("missing located refusal")
		}
		for _, change := range []func(*runtime.OperationError){
			func(e *runtime.OperationError) { e.Operation = "decode" },
			func(e *runtime.OperationError) { e.Kind = runtime.Truncated },
			func(e *runtime.OperationError) { e.Location.Path = "other" },
			func(e *runtime.OperationError) { e.Location.BitOffset = 26 },
			func(e *runtime.OperationError) { e.Location.Width = 3 },
			func(e *runtime.OperationError) { e.Location.Source.Standard = "TS 44.018" },
			func(e *runtime.OperationError) { e.Location.Source.Version = "0.0.0" },
			func(e *runtime.OperationError) { e.Location.Source.Clause = "11.2.25" },
			func(e *runtime.OperationError) { e.Cause = errors.New("unexpected") },
		} {
			altered := *operation
			change(&altered)
			if canonicalReleaseCauseReservation(definition, value, &altered) {
				t.Error("unrelated encoder error admitted")
			}
		}
		for _, bad := range []any{nil, uint8(1), ecmessages.PacketTBFReleaseMessageContent{}} {
			if canonicalReleaseCauseReservation(definition, bad, refusal) {
				t.Error("missing typed cause admitted")
			}
		}
		for _, bad := range []error{nil, errors.New("value outside sender code set")} {
			if canonicalReleaseCauseReservation(definition, value, bad) {
				t.Error("unlocated refusal admitted")
			}
		}
		// A different bounded-encoder failure must still fail the complete probe.
		altered := definition
		altered.CanonicalAtLength = func(any, int) ([]byte, error) { return nil, errors.New("unexpected") }
		if _, err := probeCanonical(altered, wire); err == nil {
			t.Fatal("unexpected bounded encoder failure admitted")
		}
	}
}
