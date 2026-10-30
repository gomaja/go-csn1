package registry

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44060/ies"
)

func TestDecodeOperationMessages(t *testing.T) {
	_, si13 := restoctets.DecodeSI13RestOctets([]byte{0x9b})
	_, cm3 := classmark.DecodeClassmark3ValuePart([]byte{255, 255})
	_, emr := measurement.DecodeEnhancedMeasurementReport([]byte{6, 5})
	_, p1 := restoctets.DecodeP1RestOctets([]byte{171, 255})
	tests := []struct {
		err  error
		want string
	}{
		{si13, "CSN.1 decode truncated at bit 8 in SI13RestOctets: SI13RestOctets requires at least 160 bits"},
		{cm3, "CSN.1 decode invalid-branch at bit 1 in Classmark3ValuePart: choice has 0 matching alternatives"},
		{emr, "CSN.1 decode invalid-value at bit 1 in EnhancedMeasurementReport/EnhancedMeasurementReportMessageType: equality constraint failed"},
		{p1, "CSN.1 decode truncated at bit 16 in P1RestOctets/P1RestOctetsCELLGLOBALCOUNTChoiceCELLGLOBALCOUNT/P1RestOctetsCELLGLOBALCOUNTChoiceCELLGLOBALCOUNTCELLGLOBALCOUNTChoiceValueAlt11/P1RestOctetsCELLGLOBALCOUNTChoiceCELLGLOBALCOUNTCELLGLOBALCOUNTChoiceValueAlt11ReducedGCR: required bit absent"},
	}
	for _, tt := range tests {
		t.Logf("MESSAGE\t%s\t%s", tt.want, tt.err.Error())
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("got %q\nwant %q", got, tt.want)
		}
		var oe *runtime.OperationError
		var de *runtime.DecodeError
		if !errors.As(tt.err, &oe) || !errors.As(tt.err, &de) || !errors.Is(tt.err, de) {
			t.Fatal("cause lost")
		}
		for _, fact := range []string{string(oe.Kind), fmt.Sprintf("at bit %d", oe.Location.BitOffset), "in " + oe.Location.Path + ":"} {
			if strings.Count(tt.err.Error(), fact) != 1 {
				t.Errorf("fact %q repeated: %v", fact, tt.err)
			}
		}
	}
}

func TestRegistryOperationMessages(t *testing.T) {
	_, unknown := Lookup("synthetic", "Missing")
	_, clause := LookupClause("synthetic", "1", "Missing")
	_, ambiguous := Lookup("TS 24.008", "A5 bits")
	_, unsupported := DecodeWithContext("TS 24.008", "Mobile Station Classmark 2 value part", nil, 0)
	source := runtime.Source{Standard: "synthetic", Clause: "1"}
	tests := []struct {
		err  error
		want string
	}{
		{unknown, "CSN.1 lookup invalid-schema in Missing: unknown CSN.1 definition in synthetic"},
		{clause, "CSN.1 lookup invalid-schema in Missing: unknown CSN.1 definition in synthetic §1"},
		{ambiguous, "CSN.1 lookup invalid-schema in A5 bits: ambiguous CSN.1 definition in TS 24.008; clauses: 10.5.1.7, 10.5.5.12a"},
		{lookupFailure("synthetic", "", "Value", fmt.Errorf("multiple versions of synthetic: 1 and 2")), "CSN.1 lookup invalid-schema in Value: multiple versions of synthetic: 1 and 2"},
		{lookupFailure("synthetic", "", "Value", fmt.Errorf("CSN.1 descriptor lacks standard or version")), "CSN.1 lookup invalid-schema in Value: CSN.1 descriptor lacks standard or version"},
		{manualReaderCheck(nil, "Value", source), "CSN.1 decode invalid-value in Value: nil reader"},
		{manualReaderCheck(runtime.NewReader(make([]byte, (1<<20)/8+1)), "Value", source), "CSN.1 decode invalid-value in Value: limit at bit 0: input exceeds maximum bit length"},
		{manualTypeError("encode", "Value", source), "CSN.1 encode invalid-value in Value: wrong value type"},
		{unsupported, "CSN.1 context invalid-value in Mobile Station Classmark 2 value part: unsupported CSN.1 value"},
	}
	for _, tt := range tests {
		t.Logf("MESSAGE\t%s\t%s", tt.want, tt.err.Error())
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("got %q\nwant %q", got, tt.want)
		}
		var oe *runtime.OperationError
		if !errors.As(tt.err, &oe) {
			t.Fatal("operation lost")
		}
		if errors.Unwrap(tt.err) != oe.Cause {
			t.Fatal("unwrap changed")
		}
		if oe.Cause != nil && !errors.Is(tt.err, oe.Cause) {
			t.Fatal("cause identity lost")
		}
		if strings.Count(tt.err.Error(), oe.Location.Path) != 1 {
			t.Errorf("path repeated: %v", tt.err)
		}
	}
	if !errors.Is(unsupported, runtime.ErrUnsupported) {
		t.Fatal("unsupported sentinel lost")
	}
	var de *runtime.DecodeError
	if !errors.As(tests[5].err, &de) || de.Offset != -1 || de.Kind != runtime.InvalidValue {
		t.Fatal("reader typed cause changed")
	}
}

func TestGeneratedOperationCauseMessages(t *testing.T) {
	wire, _ := hex.DecodeString("b0e1d5122a103fc76dfdb8ebeb652a")
	gprs, err := ies.DecodeGPRSCellOptionsIE(wire)
	if err != nil {
		t.Fatal(err)
	}
	bound := runtime.Canonical(gprs.Value)
	bound.ExtensionLengthGroup.Content.Known.Group2.CCNACTIVE = 1
	ignored := runtime.BitString{Bytes: []byte{0}, BitLength: 4}
	fallback := ies.GPRSCellOptionsIE{ExtensionLengthGroup: &ies.GPRSCellOptionsIEExtensionLengthGroup{ExtensionLength: 3, Content: ies.GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallback{Alternative: ies.GPRSCellOptionsIEExtensionLengthGroupExtensionInformationExtensionInformationFallbackAlternativeIgnored, Ignored: &ignored}}}
	calls := []struct {
		Name string
		Run  func() error
		Want string
	}{
		{"generated/nil writer", func() error { return ies.EncodeGPRSCellOptionsIETo(nil, ies.GPRSCellOptionsIE{}) }, "CSN.1 encode invalid-value in GPRSCellOptionsIE: nil writer"},
		{"generated/classmark2 detail", func() error {
			_, e := uecapability.EncodeClassmark2ValuePart(uecapability.Classmark2ValuePart{RFPowerCapability: 8})
			return e
		}, "CSN.1 encode invalid-value at bit 5 in Classmark2ValuePart/RFPowerCapability: invalid Classmark 2 field"},
		{"generated/context typed", func() error { _, e := restoctets.EncodeSI7RestOctetsCanonical(restoctets.SI7RestOctets{}); return e }, "CSN.1 canonical invalid-value in SI7RestOctets: encode: TS 44.018 §10.5.2.36 <SI7 Rest Octets>: CSN.1 context required"},
		{"generated/context sentinel", func() error {
			_, e := restoctets.EncodeNonGSMMessageStructCanonicalAtLength(restoctets.NonGSMMessageStruct{NonGSMProtocolDiscriminator: 1, NROFCONTAINEROCTETS: 31}, 2)
			return e
		}, "CSN.1 canonical invalid-value in NonGSMMessageStruct/NonGSMMessageStructCONTAINERList: encode: CSN.1 context required: continuation count requires enclosing SI value"},
		{"generated/extent required", func() error { _, e := ies.EncodeGPRSCellOptionsIECanonical(gprs.Value); return e }, "CSN.1 canonical invalid-value in GPRSCellOptionsIE: CSN.1 canonical encoding requires an explicit containing-message length"},
		{"generated/extent range", func() error { _, e := ies.EncodeGPRSCellOptionsIECanonicalAtLength(gprs.Value, -1); return e }, "CSN.1 canonical invalid-value in GPRSCellOptionsIE: CSN.1 extent -1 outside 0..13 octets"},
		{"generated/bound", func() error { _, e := ies.EncodeGPRSCellOptionsIE(bound); return e }, "CSN.1 encode invalid-value at bit 43 in GPRSCellOptionsIE/GPRSCellOptionsIEExtensionLengthGroup/ExtensionInformation/Group2: length bound (limit 43): nonzero field omitted from a truncated concatenation; only zero components may be omitted"},
		{"generated/canonical bound", func() error { _, e := ies.EncodeGPRSCellOptionsIECanonicalAtLength(bound, 6); return e }, "CSN.1 canonical invalid-value at bit 43 in GPRSCellOptionsIE/GPRSCellOptionsIEExtensionLengthGroup/ExtensionInformation/Group2: encode: length bound (limit 43): nonzero field omitted from a truncated concatenation; only zero components may be omitted"},
		{"generated/fallback", func() error { _, e := ies.EncodeGPRSCellOptionsIE(fallback); return e }, "CSN.1 encode invalid-branch at bit 25 in GPRSCellOptionsIE/GPRSCellOptionsIEExtensionLengthGroup: ignored fallback bits decode through the known arm"},
		{"generated/canonical fallback", func() error { _, e := ies.EncodeGPRSCellOptionsIECanonicalAtLength(fallback, 4); return e }, "CSN.1 canonical invalid-branch at bit 25 in GPRSCellOptionsIE/GPRSCellOptionsIEExtensionLengthGroup: encode: ignored fallback bits decode through the known arm"},
		{"generated/canonical encode failure", func() error {
			_, e := classmark.EncodeClassmark3ValuePartCanonical(classmark.Classmark3ValuePart{})
			return e
		}, "CSN.1 canonical invalid-branch at bit 1 in Classmark3ValuePart/Classmark3ValuePartMultibandSupportedChoice: encode: choice requires exactly one alternative"},
		{"generated/wrong Go type", func() error {
			d, e := Lookup("TS 24.008", "Classmark 3 Value part")
			if e != nil {
				return e
			}
			_, e = d.Encode(false)
			return e
		}, "CSN.1 encode invalid-value in Classmark 3 Value part: wrong value type"},
		{"registry/unknown", func() error { _, e := Lookup("synthetic", "Missing"); return e }, "CSN.1 lookup invalid-schema in Missing: unknown CSN.1 definition in synthetic"},
		{"registry/unknown clause", func() error { _, e := LookupClause("synthetic", "1", "Missing"); return e }, "CSN.1 lookup invalid-schema in Missing: unknown CSN.1 definition in synthetic §1"},
		{"registry/ambiguous", func() error { _, e := Lookup("TS 24.008", "A5 bits"); return e }, "CSN.1 lookup invalid-schema in A5 bits: ambiguous CSN.1 definition in TS 24.008; clauses: 10.5.1.7, 10.5.5.12a"},
		{"runtime/canonical decode-back", func() error {
			_, e := runtime.CanonicalEncodeAtLength(0, 1, 1, 1, func(int) ([]byte, error) { return []byte{0}, nil }, func([]byte) (runtime.Decoded[int], error) {
				return runtime.Decoded[int]{}, &runtime.DecodeError{Kind: runtime.Truncated, Offset: 3, Path: "Value/Field", Detail: "absent"}
			}, nil)
			return e
		}, "CSN.1 canonical invalid-schema: canonical bytes do not decode: truncated at bit 3 in Value/Field: absent"},
		{"runtime/canonical invalid extent", func() error {
			_, e := runtime.CanonicalEncode(0, -1, 1, false, func(int) ([]byte, error) { return []byte{0}, nil }, func([]byte) (runtime.Decoded[int], error) { return runtime.Decoded[int]{}, nil }, nil)
			return e
		}, "CSN.1 canonical invalid-schema: invalid canonical extent range"},
		{"runtime/canonical semantic mismatch", func() error {
			_, e := runtime.CanonicalEncodeAtLength(1, 1, 1, 1, func(int) ([]byte, error) { return []byte{0}, nil }, func([]byte) (runtime.Decoded[int], error) { return runtime.Decoded[int]{Value: 0}, nil }, nil)
			return e
		}, "CSN.1 canonical invalid-schema: canonical bytes change typed semantics"},
		{"runtime/canonical located decode-back", func() error {
			_, e := runtime.CanonicalEncodeAtLength(0, 1, 1, 1, func(int) ([]byte, error) { return []byte{0}, nil }, func([]byte) (runtime.Decoded[int], error) {
				de := &runtime.DecodeError{Kind: runtime.Truncated, Offset: 3, Path: "Value/Field", Detail: "absent"}
				return runtime.Decoded[int]{}, &runtime.OperationError{Operation: "decode", Kind: de.Kind, Location: runtime.Location{Path: de.Path, BitOffset: de.Offset}, Cause: de}
			}, nil)
			return e
		}, "CSN.1 canonical invalid-schema: canonical bytes do not decode: decode truncated at bit 3 in Value/Field: absent"},
	}
	for _, tc := range calls {
		t.Run(tc.Name, func(t *testing.T) {
			err := tc.Run()
			if err == nil {
				t.Fatal("expected failure")
			}
			t.Logf("MESSAGE\t%s\t%s", tc.Name, err.Error())
			if got := err.Error(); got != tc.Want {
				t.Fatalf("got %q\nwant %q", got, tc.Want)
			}
			var oe *runtime.OperationError
			if !errors.As(err, &oe) {
				t.Fatal("missing operation")
			}
			before := *oe
			for _, fact := range []string{string(oe.Kind), "in " + oe.Location.Path + ":"} {
				if oe.Location.Path != "" && strings.Count(err.Error(), fact) != 1 {
					t.Fatalf("fact %q repeated: %v", fact, err)
				}
			}
			if oe.Location.BitOffset >= 0 {
				fact := fmt.Sprintf("at bit %d", oe.Location.BitOffset)
				if strings.Count(err.Error(), fact) != 1 {
					t.Fatalf("fact %q repeated: %v", fact, err)
				}
			}
			if *oe != before {
				t.Fatal("formatting changed operation fields")
			}
			for e := err; e != nil; e = errors.Unwrap(e) {
				if !errors.Is(err, e) {
					t.Fatalf("lost cause %T", e)
				}
			}
			var be *runtime.BoundError
			if errors.As(err, &be) && !errors.Is(err, runtime.ErrBoundConflict) {
				t.Fatal("lost bound sentinel")
			}
			var fe *runtime.FallbackError
			if errors.As(err, &fe) && !errors.Is(err, runtime.ErrFallbackKnown) {
				t.Fatal("lost fallback sentinel")
			}
			var ce *runtime.ContextRequiredError
			if errors.As(err, &ce) && !errors.Is(err, runtime.ErrContextRequired) {
				t.Fatal("lost context sentinel")
			}
		})
	}
}

func TestLookupMessageQuotedNames(t *testing.T) {
	for _, name := range []string{"", "has \"quotes\"", "line\nbreak", "back\\slash", "\xff", strings.Repeat("\\\"", 4096)} {
		_, err := Lookup("synthetic", name)
		// The path prints the name as given, so a name that is empty or needs
		// escaping keeps its quoted form in the detail.
		want := "CSN.1 lookup invalid-schema"
		if name != "" {
			want += " in " + name
		}
		want += fmt.Sprintf(": unknown CSN.1 definition %q in synthetic", name)
		if got := err.Error(); got != want {
			t.Fatalf("name %q: got %q want %q", name, got, want)
		}
		cause := errors.Unwrap(err)
		if got, want := cause.Error(), fmt.Sprintf("unknown CSN.1 definition %q in synthetic", name); got != want {
			t.Fatalf("lookup cause changed: got %q want %q", got, want)
		}
	}
}
