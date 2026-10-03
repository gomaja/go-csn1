package registry

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// canonicalAtSomeExtent returns the first explicit extent that a definition
// without a standalone maximum accepts for a decoded value, after checking
// that the canonical bytes decode to the same typed value.
func canonicalAtSomeExtent(t *testing.T, d runtime.Descriptor, value any, maxOctets int) (int, error) {
	t.Helper()
	var first error
	for octets := 0; octets <= maxOctets; octets++ {
		out, err := d.CanonicalAtLength(value, octets)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		again, err := d.Decode(out)
		if err != nil {
			t.Fatalf("canonical %x does not decode: %v", out, err)
		}
		if !runtime.SemanticallyEqual(value, reflect.ValueOf(again).FieldByName("Value").Interface()) {
			t.Fatalf("canonical %x changes typed semantics", out)
		}
		return octets, nil
	}
	return -1, first
}

// canonicalWithin encodes a definition with a standalone maximum and checks
// that the result decodes to the same typed value.
func canonicalWithin(t *testing.T, d runtime.Descriptor, value any) error {
	t.Helper()
	out, err := d.Canonical(value)
	if err != nil {
		return err
	}
	again, err := d.Decode(out)
	if err != nil {
		t.Fatalf("canonical %x does not decode: %v", out, err)
	}
	if len(out) > d.MaxOctets || !runtime.SemanticallyEqual(value, reflect.ValueOf(again).FieldByName("Value").Interface()) {
		t.Fatalf("canonical %x exceeds %d octets or changes typed semantics", out, d.MaxOctets)
	}
	return nil
}

// TestBoundedTruncatedExtension covers TS 44.060 V19.0.0 §12.24: an R99 or
// Rel-4 cell sends Extension Information truncated inside the 6-bit Extension
// Length. The canonical encoder keeps the typed length and ends the truncated
// concatenation (§11.1.4.4) where that length is exhausted. Expected bytes
// are synthetic; pycrate 0.7.11 re-encodes its own decoding of each GPRS Cell
// Options input to the same bytes, and Wireshark 4.6.8 dissects each SI 13
// pair (input, canonical) identically apart from the padding.
func TestBoundedTruncatedExtension(t *testing.T) {
	for _, tc := range []struct {
		standard, clause, name, wire, canonical string
	}{
		{"TS 44.060", "12.24", "GPRS Cell Options IE", "b0e1d5122d103fc76dfdb8ebeb652a", "b0e1d5122d00"},
		{"TS 44.060", "12.24", "GPRS Cell Options IE", "b48f27c6a62ca8ce639d2adb", "b48f27c6a600"},
		{"TS 44.018", "10.5.2.37b", "SI 13 Rest Octets", "e1b88bbdf784a1a279df24ba4896ad235775bf40", "e1b88bbdf784a1a279df24ba4896ad2b2b2b2b2b"},
		{"TS 44.018", "10.5.2.37b", "SI 13 Rest Octets", "ea15976ab36992bb4a4df259d63dd40957e08528", "ea15976ab36992bb4a4df22b2b2b2b2b2b2b2b2b"},
		{"TS 44.018", "10.5.2.37b", "SI 13 Rest Octets", "dd0011a6fea7fbd89c88ffa33d38bfbdb253df4b", "dd0011a6fea7fbd89c88fb2b2b2b2b2b2b2b2b2b"},
	} {
		d, err := LookupClause(tc.standard, tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		wire, _ := hex.DecodeString(tc.wire)
		decoded, err := d.Decode(wire)
		if err != nil {
			t.Fatalf("%s: %v", tc.wire, err)
		}
		value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
		if plain, err := d.Encode(value); err != nil || hex.EncodeToString(plain) != tc.wire {
			t.Fatalf("%s: plain re-encode %x, %v", tc.wire, plain, err)
		}
		var canonical []byte
		if d.MaxOctets == 0 {
			octets, err := canonicalAtSomeExtent(t, d, value, 64)
			if octets < 0 {
				t.Fatalf("%s: no canonical extent in 0..64: %v", tc.wire, err)
			}
			canonical, err = d.CanonicalAtLength(value, octets)
			if err != nil {
				t.Fatal(err)
			}
		} else if canonical, err = d.Canonical(value); err != nil {
			t.Fatalf("%s: %v", tc.wire, err)
		}
		if hex.EncodeToString(canonical) != tc.canonical {
			t.Errorf("%s: canonical %x, want %s", tc.wire, canonical, tc.canonical)
		}
	}
}

// TestBoundedTruncationProbe sends random input (math/rand seed 1224, 1 to
// 40 octets) to every definition whose truncated concatenation lies inside a
// length-delimited value, plus the standalone Extension Information. Every
// decodable value must re-encode byte for byte, and encode canonically at
// some valid extent to bytes that decode to an equivalent value.
func TestBoundedTruncationProbe(t *testing.T) {
	samples := 200000
	if testing.Short() {
		samples = 20000
	}
	for _, target := range boundedTruncationDefinitions {
		t.Run(target.name, func(t *testing.T) {
			d, err := LookupClause(target.standard, target.clause, target.name)
			if err != nil {
				t.Fatal(err)
			}
			rng := rand.New(rand.NewSource(1224))
			var decoded, succeeded, failed int
			for range samples {
				wire := make([]byte, 1+rng.Intn(target.octets))
				rng.Read(wire)
				result, err := d.Decode(wire)
				if err != nil {
					continue
				}
				decoded++
				value := reflect.ValueOf(result).FieldByName("Value").Interface()
				if plain, err := d.Encode(value); err != nil || !bytes.Equal(plain, wire) {
					t.Fatalf("%x: plain re-encode %x, %v", wire, plain, err)
				}
				if d.MaxOctets == 0 {
					if octets, extentErr := canonicalAtSomeExtent(t, d, value, 64); octets < 0 {
						err = extentErr
					}
				} else {
					err = canonicalWithin(t, d, value)
				}
				if err != nil {
					if failed < 5 {
						t.Errorf("%x: %v", wire, err)
					}
					failed++
					continue
				}
				succeeded++
			}
			t.Logf("samples=%d decoded=%d canonical=%d failed=%d", samples, decoded, succeeded, failed)
			if decoded == 0 || failed != 0 {
				t.Fatalf("decoded=%d failed=%d", decoded, failed)
			}
		})
	}
}

var boundedTruncationDefinitions = []struct {
	standard, clause, name string
	octets                 int
}{
	{"TS 44.060", "12.24", "GPRS Cell Options IE", 40},
	{"TS 44.018", "10.5.2.37b", "SI 13 Rest Octets", 20},
	{"TS 44.060", "12.24", "Extension Information", 40},
}
