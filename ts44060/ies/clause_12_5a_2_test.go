package ies

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.5a.2 table 12.5a.2.1 paragraphs 22546-22560:
// seven independent presence bits each precede a five-bit mean and three-bit CV.
func type2BEPVector(mask, high int) ([]byte, int) {
	var bits strings.Builder
	for i := range 7 {
		if mask&(1<<i) == 0 {
			bits.WriteByte('0')
			continue
		}
		mean, cv := i, 0
		if high != 0 {
			mean, cv = 31-i, 7
		}
		fmt.Fprintf(&bits, "1%05b%03b", mean, cv)
	}
	n := bits.Len()
	wire := make([]byte, (n+7)/8)
	for i, b := range bits.String() {
		if b == '1' {
			wire[i/8] |= 1 << uint(7-i%8)
		}
	}
	return wire, n
}

func TestType2BEPMeasurements(t *testing.T) {
	descriptor, err := LookupClause("12.5a.2", "EGPRS BEP Link Quality Measurements Type 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	groups := []string{"GMSKMEANBEPGroup", "N8PSKMEANBEPGroup", "QPSKMEANBEPGroup", "N16QAMNSRMEANBEPGroup", "N32QAMNSRMEANBEPGroup", "N16QAMHSRMEANBEPGroup", "N32QAMHSRMEANBEPGroup"}
	for mask := range 128 {
		for high := range 2 {
			t.Run(fmt.Sprintf("%03d/%d", mask, high), func(t *testing.T) {
				wire, n := type2BEPVector(mask, high)
				decoded, err := DecodeEGPRSBEPLinkQualityMeasurementsType2IE(wire)
				if err != nil {
					t.Fatal(err)
				}
				if decoded.BitsConsumed != n {
					t.Fatalf("consumed %d, want %d", decoded.BitsConsumed, n)
				}
				value := reflect.ValueOf(decoded.Value)
				for i, name := range groups {
					group := value.FieldByName(name)
					if !group.IsValid() {
						t.Fatalf("missing typed group %s", name)
					}
					present := mask&(1<<i) != 0
					if group.IsNil() == present {
						t.Fatalf("%s presence=%v", name, !group.IsNil())
					}
					if !present {
						continue
					}
					mean, cv := i, 0
					if high != 0 {
						mean, cv = 31-i, 7
					}
					fields := group.Elem()
					if got := fields.Field(0).Uint(); got != uint64(mean) {
						t.Errorf("%s mean=%d want %d", name, got, mean)
					}
					if got := fields.Field(1).Uint(); got != uint64(cv) {
						t.Errorf("%s CV=%d want %d", name, got, cv)
					}
				}
				replay, err := EncodeEGPRSBEPLinkQualityMeasurementsType2IE(decoded.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x: %v", replay, err)
				}
				// Discard received wire state to exercise the typed canonical encoder.
				fresh := runtime.Canonical(decoded.Value)
				canonical, err := descriptor.Canonical(fresh)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
				}
				next, err := descriptor.Decode(canonical)
				if err != nil {
					t.Fatal(err)
				}
				stable, err := descriptor.Canonical(next)
				if err != nil || !bytes.Equal(stable, canonical) {
					t.Fatalf("unstable canonical %x: %v", stable, err)
				}
				reader := runtime.NewReader(append(append([]byte(nil), wire...), 0xff))
				old, err := reader.PushLimit(n)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := descriptor.DecodeFrom(reader); err != nil {
					t.Fatal(err)
				}
				if reader.Position() != n {
					t.Fatalf("bounded reader at %d want %d", reader.Position(), n)
				}
				if err := reader.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				// Every shorter bit extent is malformed, including cuts inside selectors.
				for cut := range n {
					short := runtime.NewReader(wire)
					if _, err := short.PushLimit(cut); err != nil {
						t.Fatal(err)
					}
					if _, err := descriptor.DecodeFrom(short); err == nil {
						t.Fatalf("accepted %d/%d bits", cut, n)
					}
					if short.Position() > cut {
						t.Fatalf("reader escaped limit %d at %d", cut, short.Position())
					}
				}
				if _, err := DecodeEGPRSBEPLinkQualityMeasurementsType2IE(wire[:len(wire)-1]); err == nil {
					t.Fatal("accepted truncated octets")
				}
			})
		}
	}
}

func TestType2BEPFieldBounds(t *testing.T) {
	descriptor, err := LookupClause("12.5a.2", "EGPRS BEP Link Quality Measurements Type 2 IE")
	if err != nil {
		t.Fatal(err)
	}
	wire, _ := type2BEPVector(127, 1)
	// Each mean/CV field rejects exactly the first value outside its printed width.
	for group := range 7 {
		for field, invalid := range []uint64{32, 8} {
			decoded, err := DecodeEGPRSBEPLinkQualityMeasurementsType2IE(wire)
			if err != nil {
				t.Fatal(err)
			}
			fresh := runtime.Canonical(decoded.Value)
			value := reflect.ValueOf(&fresh).Elem()
			value.Field(group).Elem().Field(field).SetUint(invalid)
			if _, err := descriptor.Canonical(fresh); err == nil {
				t.Fatalf("group %d field %d accepted %d", group, field, invalid)
			}
			if _, err := EncodeEGPRSBEPLinkQualityMeasurementsType2IE(fresh); err == nil {
				t.Fatalf("direct encoder group %d field %d accepted %d", group, field, invalid)
			}
		}
	}
}

func FuzzType2BEPMeasurements(f *testing.F) {
	for mask := range 128 {
		for high := range 2 {
			wire, _ := type2BEPVector(mask, high)
			f.Add(wire)
		}
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 32 {
			return
		}
		decoded, err := DecodeEGPRSBEPLinkQualityMeasurementsType2IE(wire)
		if err != nil {
			return
		}
		replay, err := EncodeEGPRSBEPLinkQualityMeasurementsType2IE(decoded.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("accepted input %x replayed as %x: %v", wire, replay, err)
		}
	})
}
