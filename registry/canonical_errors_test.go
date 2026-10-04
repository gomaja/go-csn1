package registry

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
	"github.com/gomaja/go-csn1/ts44060/ies"
)

// TestCanonicalErrorTypes pins the error types documented on the generated
// canonical entry points, using synthetic values: a missing or violated
// extent is *runtime.ExtentError, content that does not fit a bound is
// *runtime.BoundError, an extent owned by an absent containing message wraps
// runtime.ErrContextRequired, and an SI4-ACS-dependent layout without its
// context is *runtime.ContextRequiredError.
func TestCanonicalErrorTypes(t *testing.T) {
	decodeHex := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	gprs, err := ies.DecodeGPRSCellOptionsIE(decodeHex("b0e1d5122d103fc76dfdb8ebeb652a"))
	if err != nil {
		t.Fatal(err)
	}
	ccn := runtime.Canonical(gprs.Value)
	ccn.ExtensionLengthGroup.Content.Known.Group2.CCNACTIVE = 1
	si13, err := restoctets.DecodeSI13RestOctets(decodeHex("e1b88bbdf784a1a279df24ba4896ad235775bf40"))
	if err != nil {
		t.Fatal(err)
	}
	// The extension was truncated after its Rel-6 group (TS 44.060 V19.0.0
	// §12.24); a nonzero REDUCED_LATENCY_ACCESS cannot be omitted.
	reduced := runtime.Canonical(si13.Value)
	reduced.BCCHCHANGEMARKGroup.RACChoice.RAC.GPRSCellOptions.ExtensionLengthGroup.Content.Known.REDUCEDLATENCYACCESS = 1
	emr, err := measurement.DecodeEnhancedMeasurementReport([]byte{0x10, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	if err != nil {
		t.Fatal(err)
	}
	// 96 reported positions need far more than the TS 44.018 V19.0.0
	// §9.1.55 21-octet maximum.
	long := runtime.Canonical(emr.Value)
	list := make([]*uint8, 96)
	for i := range list {
		value := uint8(63)
		list[i] = &value
	}
	long.REPORTINGQUANTITYList = &list
	container := restoctets.NonGSMMessageStruct{NonGSMProtocolDiscriminator: 1, NROFCONTAINEROCTETS: 31}

	extent := func(want runtime.ExtentError) func(error) bool {
		return func(err error) bool {
			var got *runtime.ExtentError
			return errors.As(err, &got) && *got == want
		}
	}
	bound := func(kind runtime.BoundKind) func(error) bool {
		return func(err error) bool {
			var got *runtime.BoundError
			return errors.As(err, &got) && got.Kind == kind && got.Limit >= 0 && got.Position >= 0
		}
	}
	for _, tc := range []struct {
		name string
		run  func() error
		want func(error) bool
	}{
		{"no standalone maximum", func() error { _, err := ies.EncodeGPRSCellOptionsIECanonical(gprs.Value); return err }, extent(runtime.ExtentError{Required: true})},
		{"extent below range", func() error { _, err := ies.EncodeGPRSCellOptionsIECanonicalAtLength(gprs.Value, -1); return err }, extent(runtime.ExtentError{Actual: -1, Maximum: 1 << 17})},
		{"extent above runtime limit", func() error {
			_, err := ies.EncodeGPRSCellOptionsIECanonicalAtLength(gprs.Value, 1<<17+1)
			return err
		}, extent(runtime.ExtentError{Actual: 1<<17 + 1, Maximum: 1 << 17})},
		{"extent not filled", func() error { _, err := ies.EncodeGPRSCellOptionsIECanonicalAtLength(gprs.Value, 10); return err }, extent(runtime.ExtentError{Actual: 6, Minimum: 10, Maximum: 10})},
		{"extent too small", func() error { _, err := ies.EncodeGPRSCellOptionsIECanonicalAtLength(gprs.Value, 1); return err }, bound(runtime.CanonicalTarget)},
		{"length too short at extent", func() error { _, err := ies.EncodeGPRSCellOptionsIECanonicalAtLength(ccn, 6); return err }, bound(runtime.LengthBound)},
		{"length too short", func() error { _, err := restoctets.EncodeSI13RestOctetsCanonical(reduced); return err }, bound(runtime.LengthBound)},
		{"maximum exceeded", func() error { _, err := measurement.EncodeEnhancedMeasurementReportCanonical(long); return err }, func(err error) bool {
			var got *runtime.ExtentError
			return errors.As(err, &got) && got.Maximum == 21 && got.Actual > 21 && !got.Required
		}},
		{"containing message absent", func() error {
			_, err := restoctets.EncodeNonGSMMessageStructCanonicalAtLength(container, 2)
			return err
		}, func(err error) bool { return errors.Is(err, runtime.ErrContextRequired) }},
		{"ACS context absent", func() error {
			_, err := restoctets.EncodeSI7RestOctetsCanonical(restoctets.SI7RestOctets{})
			return err
		}, func(err error) bool {
			var got *runtime.ContextRequiredError
			return errors.As(err, &got) && errors.Is(err, runtime.ErrContextRequired)
		}},
	} {
		if err := tc.run(); !tc.want(err) {
			t.Errorf("%s: %v (%T)", tc.name, err, err)
		}
	}
}
