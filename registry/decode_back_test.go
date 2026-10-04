package registry

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msnetcap"
	"github.com/gomaja/go-csn1/ts44018/cellselection"
)

// runtime.CanonicalEncode wraps a decode failure as *runtime.DecodeError, but
// generated codecs never reach it: the generator rejects ambiguous choices
// and constructs that read to the end of their bound with content after
// them. A zero value at zero octets writes its first bit and stops at the
// canonical target first.
func TestCanonicalDecodeBackDoesNotFail(t *testing.T) {
	for name, call := range map[string]func() ([]byte, error){
		"GEA1Bits": func() ([]byte, error) { return msnetcap.EncodeGEA1BitsCanonicalAtLength(msnetcap.GEA1Bits{}, 0) },
		"GSMDescriptionStruct": func() ([]byte, error) {
			return cellselection.EncodeGSMDescriptionStructCanonicalAtLength(cellselection.GSMDescriptionStruct{}, 0)
		},
	} {
		_, err := call()
		var bound *runtime.BoundError
		var decode *runtime.DecodeError
		if !errors.As(err, &bound) || bound.Kind != runtime.CanonicalTarget || errors.As(err, &decode) {
			t.Errorf("%s at 0 octets: %v, want BoundError CanonicalTarget", name, err)
		}
	}
	// Every definition's zero value, at every small extent.
	rng := rand.New(rand.NewSource(19))
	checked := 0
	for _, d := range descriptors() {
		var sample any
		for i := 0; i < 2000 && sample == nil; i++ {
			b := make([]byte, 1+rng.Intn(24))
			rng.Read(b)
			if v, err := d.Decode(b); err == nil {
				sample = v
			}
		}
		if sample == nil {
			continue
		}
		zero := reflect.Zero(reflect.TypeOf(sample)).Interface()
		var decode *runtime.DecodeError
		if d.Canonical != nil {
			if _, err := d.Canonical(zero); errors.As(err, &decode) {
				t.Errorf("%s %s: Canonical of zero value: %v", d.Clause, d.Name, err)
			}
		}
		if d.CanonicalAtLength != nil {
			for octets := 0; octets <= 3; octets++ {
				if _, err := d.CanonicalAtLength(zero, octets); errors.As(err, &decode) {
					t.Errorf("%s %s: CanonicalAtLength(zero, %d): %v", d.Clause, d.Name, octets, err)
				}
			}
		}
		checked++
	}
	if checked < 200 {
		t.Fatalf("only %d definitions checked", checked)
	}
}
