package nongprsoptions

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 Table 12.27.1 paragraphs 23477-23479: a six-bit Extension Length
// followed by bit(val(Extension Length)+1) bounds the component to 64 bits.
func TestExtensionInformationContainingMaximum(t *testing.T) {
	value := ExtensionInformation{}
	if got, err := EncodeExtensionInformationCanonicalAtLength(value, 8); err != nil || len(got) != 8 {
		t.Fatalf("64-bit maximum: %x %v", got, err)
	}
	_, err := EncodeExtensionInformationCanonicalAtLength(value, 9)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Maximum != 8 || extent.Actual != 9 {
		t.Fatalf("accepted beyond 64-bit containing field: %v", err)
	}
}

// TS 44.060 V19.0.0 Table 12.27.1 paragraphs 23460–23479:
// 31 + 8 + 1 + 6 + 64 = 110 meaningful bits, rounded to 14 octets.
func TestNonGPRSCellOptionsIEContainingMaximum(t *testing.T) {
	value, _ := cellOptionsValue(true, true, 1, 1, 63)
	if got, err := EncodeNonGPRSCellOptionsIECanonicalAtLength(value, 14); err != nil || len(got) != 14 {
		t.Fatalf("110-bit maximum: %x %v", got, err)
	}
	_, err := EncodeNonGPRSCellOptionsIECanonicalAtLength(value, 15)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Maximum != 14 || extent.Actual != 15 {
		t.Fatalf("accepted beyond standalone ceiling: %v", err)
	}
}
