package registry

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ies"
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

// TestBoundConflictIsTyped covers an Extension Length too short for the typed
// Extension Information (TS 44.060 V19.0.0 §12.24). Encoding refuses it and
// keeps the typed length; the refusal is a *runtime.BoundError naming the
// bound, the encoder path and the conflicting field.
func TestBoundConflictIsTyped(t *testing.T) {
	wire, _ := hex.DecodeString("b0e1d5122d103fc76dfdb8ebeb652a")
	decoded, err := ies.DecodeGPRSCellOptionsIE(wire)
	if err != nil {
		t.Fatal(err)
	}
	const extension = "GPRSCellOptionsIE/GPRSCellOptionsIEExtensionLengthGroup/ExtensionInformation"
	for _, tc := range []struct {
		name  string
		edit  func(*ies.GPRSCellOptionsIE)
		fresh bool
		want  runtime.BoundError
	}{
		// The 9-bit extension ends at bit 43 after the R99 group; CCN_ACTIVE
		// = 1 cannot be omitted, and the bound cannot carry it.
		{"canonical nonzero omitted field", func(v *ies.GPRSCellOptionsIE) {
			v.ExtensionLengthGroup.Content.Known.Group2.CCNACTIVE = 1
		}, true, runtime.BoundError{Kind: runtime.LengthBound, Path: extension, Field: "Group2", Limit: 43, Position: 43}},
		// The decoded value recorded its truncation after the R99 group.
		{"decoded nonzero omitted field", func(v *ies.GPRSCellOptionsIE) {
			v.ExtensionLengthGroup.Content.Known.Group2.CCNACTIVE = 1
		}, false, runtime.BoundError{Kind: runtime.ReceivedTruncation, Path: extension, Field: "Group2", Limit: 43, Position: 43}},
		// An 8-bit extension ends at bit 42, inside the 9-bit R99 group.
		{"length inside a component", func(v *ies.GPRSCellOptionsIE) {
			v.ExtensionLengthGroup.ExtensionLength = 7
		}, true, runtime.BoundError{Kind: runtime.LengthBound, Field: "ExtensionInformationEGPRSPACKETCHANNELREQUESTGroupBSSPAGINGCOORDINATION", Limit: 42, Position: 42}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := runtime.Canonical(decoded.Value)
			if !tc.fresh {
				value = decoded.Value
				known := *value.ExtensionLengthGroup.Content.Known
				group := *value.ExtensionLengthGroup
				group.Content.Known = &known
				value.ExtensionLengthGroup = &group
			}
			tc.edit(&value)
			var err error
			if tc.fresh {
				_, err = ies.EncodeGPRSCellOptionsIECanonicalAtLength(value, 6)
			} else {
				_, err = ies.EncodeGPRSCellOptionsIE(value)
			}
			var bound *runtime.BoundError
			if !errors.As(err, &bound) || !errors.Is(err, runtime.ErrBoundConflict) {
				t.Fatalf("error %v (%T) is not a typed bound conflict", err, err)
			}
			path := bound.Path == tc.want.Path
			if tc.want.Path == "" {
				// A conflict inside a component names that component's path.
				path = strings.HasPrefix(bound.Path, extension+"/") && strings.HasSuffix(bound.Path, "/"+tc.want.Field)
			}
			if !path || bound.Kind != tc.want.Kind || bound.Field != tc.want.Field || bound.Limit != tc.want.Limit || bound.Position != tc.want.Position {
				t.Fatalf("bound conflict %+v, want %+v", *bound, tc.want)
			}
			if decoded.Value.ExtensionLengthGroup.ExtensionLength != 8 || decoded.Value.ExtensionLengthGroup.Content.Known.Group2.CCNACTIVE != 0 {
				t.Fatal("test edited the decoded value")
			}
		})
	}
}

// TestGeneratedBoundConflictsAreTyped checks every generated codec: an
// encoder that rejects content for its enclosing length, fixed size or
// truncation point must return a typed runtime error, never fmt.Errorf.
func TestGeneratedBoundConflictsAreTyped(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "ts*", "*", "generated.go"))
	if err != nil || len(files) != 8 {
		t.Fatalf("generated files %v, %v", files, err)
	}
	var omitted, length int
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, untyped := range []string{"beyond transmitted truncation", `fmt.Errorf("SI stop record`, `fmt.Errorf("continuation container`, `fmt.Errorf("empty alternative`, `fmt.Errorf("nonempty alternative`, `fmt.Errorf("fixed value`} {
			if strings.Contains(string(source), untyped) {
				t.Errorf("%s emits untyped %q", file, untyped)
			}
		}
		omitted += strings.Count(string(source), "w.OmittedFieldError(")
		length += strings.Count(string(source), "w.LengthBoundError(")
	}
	if omitted == 0 || length != 5 {
		t.Fatalf("typed bound checks: %d omitted-field, %d length", omitted, length)
	}
}
