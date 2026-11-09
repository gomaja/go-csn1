package registry

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/egprsmode2"
	"github.com/gomaja/go-csn1/ts44060/packetcsreleaseindication"
)

// TS 44.060 V19.0.0 §11.2.37 p20103 delegates EGPRS mode 2 (§12.48a.1).
// Its Single Uplink (§12.48a.3) carries the Pulse Format bitmap (§12.8.3
// pp22817–22819). The synthetic source arm selects that exact typed path.
func canonicalReleaseImportedErrors(t *testing.T, d runtime.Descriptor, record func(string, error, int)) {
	t.Helper()
	rootWire, err := hex.DecodeString("0001022ac000")
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewChoiceReaderBits(rootWire, 41, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(27); err != nil {
		t.Fatal(err)
	}
	root, err := packetcsreleaseindication.DecodePacketCSReleaseMessageContentFrom(r)
	if err != nil || r.Position() != 41 {
		t.Fatalf("typed EGPRS delegation seed: position %d, %v", r.Position(), err)
	}
	bitmapWire, err := hex.DecodeString("139081ab02a0")
	if err != nil {
		t.Fatal(err)
	}
	bitmapReader, err := runtime.NewReaderBits(bitmapWire, 43, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bitmapReader.PushLimit(43); err != nil {
		t.Fatal(err)
	}
	original, err := egprsmode2.DecodeEGPRSMode2IEFrom(bitmapReader)
	if err != nil {
		t.Fatal(err)
	}
	for _, ignored := range []bool{false, true} {
		v := runtime.Canonical(root)
		assignment := v.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known.PaddingBitsChoice.FrequencyParameters.FrequencyParametersChoice.BEPPERIOD2.FrequencyParametersChoice.AssignmentInfo.Known.AssignmentInfoGroupFallback.Known
		assignment.EGPRSMode = runtime.Canonical(original)
		group := assignment.EGPRSMode.DownlinkAssignmentChoice.DownlinkAssignment.UplinkAssignment.PulseFormat.PulseFormatCoding1Choice.PulseFormatCoding2.PulseFormatCoding2.PulseFormatBitmapLengthChoice.PulseFormatBitmapLength
		if group == nil || group.PulseFormatBitmapLength != 2 || group.PulseFormatBitmap.Known == nil {
			t.Fatal("witness did not select the typed three-bit bitmap")
		}
		group.PulseFormatBitmapLength = 0
		if ignored {
			group.PulseFormatBitmap.Alternative = 1
			group.PulseFormatBitmap.Known = nil
			group.PulseFormatBitmap.Ignored = &runtime.BitString{Bytes: []byte{0x80}, BitLength: 1}
		}
		_, err := d.CanonicalAtLength(v, 64)
		if ignored {
			var fallback *runtime.FallbackError
			if !errors.As(err, &fallback) {
				t.Errorf("release imported known fallback: %v", err)
				continue
			}
		} else {
			var bound *runtime.BoundError
			if !errors.As(err, &bound) || bound.Kind != runtime.LengthBound {
				t.Errorf("release imported bitmap length bound: %v", err)
				continue
			}
		}
		record(entryAtLength, err, 64)
	}
}

func TestCanonicalReleaseImportedErrors(t *testing.T) {
	d, err := packetcsreleaseindication.Lookup("Packet CS Release message content")
	if err != nil {
		t.Fatal(err)
	}
	observed := map[string]bool{}
	canonicalReleaseImportedErrors(t, d, func(entry string, err error, octets int) {
		observed[classify(err, octets, entry)] = true
	})
	for _, class := range []string{classLength, classFallback} {
		if !observed[class] {
			t.Errorf("missing imported release error class %s", class)
		}
	}
	if len(observed) != 2 {
		t.Fatalf("unexpected imported release error classes %v", observed)
	}
}

// TS 44.060 V19.0.0 §11.2.37 pp20193–20207 supplies every allocation
// scope in the complete message. A standalone assignment still needs that scope.
func TestCanonicalReleaseContextProfile(t *testing.T) {
	documented := documentedClasses(t)
	for _, entry := range []struct {
		function string
		context  bool
	}{
		{"EncodePacketCSReleaseMessageContentCanonicalAtLength", false},
		{"EncodeUplinkTBFAssignmentStructCanonicalAtLength", true},
	} {
		key := "ts44060/packetcsreleaseindication." + entry.function
		classes, exists := documented[key]
		if !exists {
			t.Errorf("missing documented canonical function %s", key)
			continue
		}
		if classes[classContext] != entry.context {
			t.Errorf("%s context class=%t, want %t", key, classes[classContext], entry.context)
		}
	}
	standalone := packetcsreleaseindication.UplinkTBFAssignmentStruct{
		TBFTIMESLOTALLOCATION: &runtime.BitString{Bytes: []byte{0x80}, BitLength: 1},
	}
	_, err := packetcsreleaseindication.EncodeUplinkTBFAssignmentStructCanonicalAtLength(standalone, 64)
	var operation *runtime.OperationError
	if !errors.Is(err, runtime.ErrContextRequired) || !errors.As(err, &operation) || operation.Kind != runtime.InvalidValue || operation.Location.Source.Clause != "11.2.37" {
		t.Fatalf("standalone assignment lost its enclosing allocation refusal: %v", err)
	}
}

// TS 44.060 V19.0.0 §11.2.37's legacy GPRS extension choice permits
// 0 bit** = < no string > before the Primary TSC additions. This synthetic
// source arm ends after 20 content bits: three octets fit, four remain unfilled.
func canonicalReleaseUnfilledError(t *testing.T, d runtime.Descriptor, record func(string, error, int)) {
	t.Helper()
	wire, err := hex.DecodeString("0001022000")
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewChoiceReaderBits(wire, 34, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(20); err != nil {
		t.Fatal(err)
	}
	v, err := packetcsreleaseindication.DecodePacketCSReleaseMessageContentFrom(r)
	if err != nil || r.Position() != 34 {
		t.Fatalf("typed unfilled release seed: position %d, %v", r.Position(), err)
	}
	choice := v.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known.PaddingBitsChoice.FrequencyParameters.FrequencyParametersChoice.CHANNELCODINGCOMMAND.FrequencyParametersGroupFallback.Known.PrimaryTSCSetChoice
	if choice.Alternative != 1 || choice.NoString == nil || choice.AltUnlabeled != nil || choice.PrimaryTSCSet != nil {
		t.Fatal("witness did not select the source no-string extension arm")
	}
	out, err := d.CanonicalAtLength(v, 3)
	if err != nil || len(out) != 3 {
		t.Fatalf("three-octet release control: length %d, %v", len(out), err)
	}
	_, err = d.CanonicalAtLength(v, 4)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Required || extent.Actual != 3 || extent.Minimum != 4 || extent.Maximum != 4 || classify(err, 4, entryAtLength) != classUnfilled {
		t.Fatalf("release unfilled extent: %v", err)
	}
	record(entryAtLength, err, 4)
}

func TestCanonicalReleaseUnfilledError(t *testing.T) {
	d, err := packetcsreleaseindication.Lookup("Packet CS Release message content")
	if err != nil {
		t.Fatal(err)
	}
	observed := map[string]bool{}
	canonicalReleaseUnfilledError(t, d, func(entry string, err error, octets int) {
		observed[classify(err, octets, entry)] = true
	})
	if !observed[classUnfilled] || len(observed) != 1 {
		t.Fatalf("missing release extent-unfilled witness: %v", observed)
	}
}
