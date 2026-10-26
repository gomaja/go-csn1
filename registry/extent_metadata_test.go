package registry_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

// Context is a property of the available operations. A self-delimiting reader
// can coexist with a canonical API that requires an explicit external extent.
func TestDescriptorContextMetadataMatchesOperations(t *testing.T) {
	descriptors := registry.Descriptors()
	if len(descriptors) != 236 {
		t.Fatalf("catalogue has %d descriptors, want 236", len(descriptors))
	}
	for _, d := range descriptors {
		t.Run(d.Standard+"/"+d.Clause+"/"+d.Name, func(t *testing.T) {
			_, readerErr := d.DecodeFrom(runtime.NewReader(nil))
			// Decode's error result still carries the descriptor's concrete value type.
			// Supplying that type tests canonical extent preflight, rather than the
			// unrelated wrong-dynamic-type check or a guessed semantic constructor.
			boxed, _ := d.Decode(nil)
			value := reflect.ValueOf(boxed).FieldByName("Value").Interface()
			_, canonicalErr := d.Canonical(value)
			var extent *runtime.ExtentError
			canonicalContext := errors.As(canonicalErr, &extent) && extent.Required
			want := d.DecodeWithContext != nil || errors.Is(readerErr, runtime.ErrContextRequired) || canonicalContext
			if d.Extent.NeedsContext != want {
				t.Fatalf("NeedsContext=%t want %t: reader=%v canonical=%v", d.Extent.NeedsContext, want, readerErr, canonicalErr)
			}
		})
	}
}

func TestDescriptorExtentMetadataAgainstAcceptedSamples(t *testing.T) {
	for _, d := range registry.Descriptors() {
		t.Run(d.Standard+"/"+d.Clause+"/"+d.Name, func(t *testing.T) {
			e := d.Extent
			maxOctets := 0
			if e.Finite {
				maxOctets = (e.MaximumBits + 7) / 8
			}
			if d.MaxOctets != maxOctets {
				t.Fatalf("MaxOctets=%d want %d", d.MaxOctets, maxOctets)
			}
			if e.Fixed != (e.Finite && e.MinimumBits == e.MaximumBits) {
				t.Fatalf("inconsistent Fixed: %+v", e)
			}
			lengths := []int{0, 1, (e.MinimumBits + 7) / 8, 20, 32, 50}
			if e.Finite {
				lengths = append(lengths, maxOctets, maxOctets+1)
			}
			for _, n := range lengths {
				for _, fill := range []byte{0, 0x2b, 0xff} {
					wire := bytes.Repeat([]byte{fill}, n)
					contexts := []runtime.SI4ACS{runtime.SI4ACSZero}
					if d.DecodeWithContext != nil {
						contexts = append(contexts, runtime.SI4ACSOne)
					}
					for _, acs := range contexts {
						var value any
						var err error
						if d.DecodeWithContext == nil {
							value, err = d.Decode(wire)
						} else {
							value, err = d.DecodeWithContext(wire, acs)
						}
						if err != nil {
							continue
						}
						if n*8 < e.MinimumBits {
							t.Fatalf("accepted %d bits below minimum %d", n*8, e.MinimumBits)
						}
						var canonical []byte
						if d.CanonicalWithContext == nil {
							canonical, err = d.Canonical(value)
						} else {
							canonical, err = d.CanonicalWithContext(value, acs)
						}
						if err != nil {
							continue
						}
						if len(canonical)*8 < e.MinimumBits || e.Finite && len(canonical) > maxOctets {
							t.Fatalf("canonical length %d outside %+v", len(canonical), e)
						}
						if e.Fixed && len(canonical) != (e.MinimumBits+7)/8 {
							t.Fatalf("fixed canonical has %d octets, want %d", len(canonical), (e.MinimumBits+7)/8)
						}
					}
				}
			}
			for _, fill := range []byte{0, 0x2b, 0xff} {
				for bits := 0; bits < e.MinimumBits; bits++ {
					wire := bytes.Repeat([]byte{fill}, (bits+7)/8)
					r := runtime.NewReader(wire)
					if _, err := r.PushLimit(bits); err != nil {
						t.Fatal(err)
					}
					var err error
					if d.DecodeFromWithContext == nil {
						_, err = d.DecodeFrom(r)
					} else {
						_, err = d.DecodeFromWithContext(r, runtime.SI4ACSZero)
					}
					if err == nil {
						t.Fatalf("bounded reader accepted %d bits below minimum %d", bits, e.MinimumBits)
					}
				}
			}
		})
	}
}

// Exhaustive byte inputs make an understated lower bound observable: checking
// only that shorter inputs fail cannot detect a minimum that is too small.
func TestDescriptorMinimumIsAttained(t *testing.T) {
	for _, d := range registry.Descriptors() {
		t.Run(d.Standard+"/"+d.Clause+"/"+d.Name, func(t *testing.T) {
			bits := d.Extent.MinimumBits
			// Larger source values are constructed separately below.
			if bits > 16 {
				return
			}
			octets := (bits + 7) / 8
			wire := make([]byte, octets)
			contexts := []runtime.SI4ACS{runtime.SI4ACSZero}
			if d.DecodeWithContext != nil {
				contexts = append(contexts, runtime.SI4ACSOne)
			}
			accepted := 0
			for input := 0; input < 1<<(octets*8); input++ {
				for i := range wire {
					wire[i] = byte(input >> (i * 8))
				}
				for _, acs := range contexts {
					var err error
					if d.DecodeWithContext == nil {
						_, err = d.Decode(wire)
					} else {
						_, err = d.DecodeWithContext(wire, acs)
					}
					if err == nil {
						accepted++
						break
					}
				}
			}
			t.Logf("enumerated %d inputs; accepted %d", 1<<(octets*8), accepted)
			if accepted == 0 {
				_, err := d.Decode(nil)
				if bits == 0 && err != nil {
					t.Log("zero structural lower bound; absent input is refused (for example when a length is supplied only by a containing value)")
					return
				}
				t.Fatalf("no accepted %d-octet input attains minimum %d", octets, bits)
			}
		})
	}
}

func TestMeasurementParametersIgnoredBitMinimum(t *testing.T) {
	d, err := registry.Lookup("TS 44.018", "GPRS 3G MEASUREMENT PARAMETERS Description struct")
	if err != nil {
		t.Fatal(err)
	}
	if d.Extent.MinimumBits != 9 {
		t.Fatalf("minimum=%d want 9", d.Extent.MinimumBits)
	}
	for input := 0; input < 256; input++ {
		if _, err := d.Decode([]byte{byte(input)}); err == nil {
			t.Fatalf("accepted one octet %02x", input)
		}
	}
	// The eight field bits followed by the mandatory ignored bit are accepted.
	decoded, err := d.Decode([]byte{0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if got := reflect.ValueOf(decoded).FieldByName("BitsConsumed").Int(); got != 9 {
		t.Fatalf("consumed %d bits, want 9", got)
	}
	canonical, err := d.Canonical(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(canonical) != 2 {
		t.Fatalf("canonical length=%d want 2", len(canonical))
	}
}

// Synthetic minimal values use zero-valued fields and absent optionals, with
// source-required nonzero selectors, length fields and L bits. Counts are
// independent expectations, rather than lengths derived from the metadata.
func TestDescriptorLargeMinimumIsAttained(t *testing.T) {
	fixtures := map[string]struct {
		bits int
		hex  string
	}{
		"TS 24.008/10.5.1.6/Mobile Station Classmark 2 value part":             {24, "200000"},
		"TS 36.331/UE-CapabilityRAT-ContainerList field descriptions/geran-cs": {48, "330320000000"},
		"TS 44.018/10.5.2.16/EGPRS Packet Uplink Assignment":                   {32, "00000000"},
		"TS 44.018/10.5.2.16/Multiple Blocks Packet Downlink Assignment":       {48, "000008000000"},
		"TS 44.018/10.5.2.16/Packet Downlink Assignment":                       {36, "000000000a"},
		"TS 44.018/10.5.2.16/Packet Uplink Assignment":                         {24, "800000"},
		"TS 44.018/10.5.2.17/IAR Rest Octets":                                  {24, "000000"},
		"TS 44.018/10.5.2.1e/E-UTRAN Description struct":                       {19, "000000"},
		"TS 44.018/10.5.2.1e/GSM Description struct":                           {17, "000000"},
		"TS 44.018/10.5.2.22c/NT/N Rest Octets":                                {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.25/P3 Rest Octets":                                   {24, "002100"},
		"TS 44.018/10.5.2.33a/SI2ter Rest Octets":                              {32, "00000000"},
		"TS 44.018/10.5.2.33a/UTRAN FDD Description struct":                    {17, "400000"},
		"TS 44.018/10.5.2.33a/UTRAN TDD Description struct":                    {17, "400000"},
		"TS 44.018/10.5.2.33b/Repeated UTRAN FDD Neighbour Cells struct":       {21, "000000"},
		"TS 44.018/10.5.2.33b/Repeated UTRAN TDD Neighbour Cells struct":       {21, "000000"},
		"TS 44.018/10.5.2.33b/SI2quater Rest Octets":                           {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.33c/SI2n Rest Octets":                                {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.34/SI3 Rest Octet":                                   {32, "00000000"},
		"TS 44.018/10.5.2.35a/SI6 rest octets":                                 {56, "00000000000000"},
		"TS 44.018/10.5.2.36/SI7 Rest Octets":                                  {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37/SI8 Rest Octets":                                  {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37a/SI9 rest octets":                                 {136, "0000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37b/PEO IMM Cell Group Definition struct":            {40, "0000000000"},
		"TS 44.018/10.5.2.37b/SI 13 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37e/LSA Parameters":                                  {19, "010000"},
		"TS 44.018/10.5.2.37e/SI16 Rest Octets":                                {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37f/SI17 Rest Octets":                                {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37g/SI 19 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37h/SI 18 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37i/SI 20 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37j/DYNAMIC ARFCN MAPPING":                           {31, "00000000"},
		"TS 44.018/10.5.2.37j/SI14 Rest Octets":                                {128, "00000000000000000000000000000000"},
		"TS 44.018/10.5.2.37k/DYNAMIC ARFCN MAPPING":                           {31, "00000000"},
		"TS 44.018/10.5.2.37k/SI15 Rest Octets":                                {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37l/PBCCH Description 2 struct":                      {26, "00000000"},
		"TS 44.018/10.5.2.37l/SI 13alt Rest Octets":                            {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37m/SI 21 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37n/Network Sharing Information struct":              {28, "00000000"},
		"TS 44.018/10.5.2.37n/SI 22 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.37o/SI 23 Rest Octets":                               {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.44/SI10 rest octets":                                 {160, "0000000000000000000000000000000000000000"},
		"TS 44.018/10.5.2.44/further cell info":                                {26, "00000000"},
		"TS 44.018/10.5.2.70/SI10bis Neighbour Cell Info":                      {30, "00000000"},
		"TS 44.018/10.5.2.78/IPA Rest Octets":                                  {152, "00000000000000000000000000000000000000"},
		"TS 44.018/9.1.21a/Group Call information":                             {37, "0000000000"},
		"TS 44.018/9.1.21a/Group Channel Description":                          {25, "00000000"},
		"TS 44.018/9.1.55/E-UTRAN Measurement Report struct":                   {20, "000000"},
		"TS 44.018/9.1.55/Repeated Invalid_BSIC_Information struct":            {17, "000000"},
		"TS 44.018/9.1.55/Serving cell data struct":                            {23, "000000"},
		"TS 44.060/12.24/GPRS Cell Options IE":                                 {19, "000000"},
		"TS 44.060/12.33/TMGI IE":                                              {25, "00000000"},
		"TS 44.060/12.61/PLMN-ID struct":                                       {24, "000000"},
		"TS 44.060/12.61/UTRAN CSG Measurement Report IE":                      {63, "0000000000000000"},
		"TS 44.060/12.8/Direct encoding 2 struct":                              {40, "0000000000"},
		"TS 44.060/12.9a/GPRS Power Control Parameters IE":                     {19, "000000"},
	}
	checked := 0
	for _, d := range registry.Descriptors() {
		if d.Extent.MinimumBits <= 16 {
			continue
		}
		key := d.Standard + "/" + d.Clause + "/" + d.Name
		t.Run(key, func(t *testing.T) {
			fixture, ok := fixtures[key]
			if !ok {
				t.Fatal("missing minimal-value construction")
			}
			if d.Extent.MinimumBits != fixture.bits {
				t.Fatalf("minimum=%d want constructed %d", d.Extent.MinimumBits, fixture.bits)
			}
			wire, err := hex.DecodeString(fixture.hex)
			if err != nil {
				t.Fatal(err)
			}
			contexts := []runtime.SI4ACS{runtime.SI4ACSZero}
			if d.DecodeWithContext != nil {
				contexts = append(contexts, runtime.SI4ACSOne)
			}
			for _, acs := range contexts {
				var decoded any
				if d.DecodeWithContext == nil {
					decoded, err = d.Decode(wire)
				} else {
					decoded, err = d.DecodeWithContext(wire, acs)
				}
				if err != nil {
					t.Fatalf("minimal byte value (ACS %v): %v", acs, err)
				}
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(fixture.bits); err != nil {
					t.Fatal(err)
				}
				if d.DecodeFromWithContext == nil {
					_, err = d.DecodeFrom(r)
				} else {
					_, err = d.DecodeFromWithContext(r, acs)
				}
				if err != nil {
					t.Fatalf("minimal bounded value (ACS %v): %v", acs, err)
				}
				if d.DecodeWithContext == nil {
					if r.Position() != fixture.bits {
						t.Fatalf("minimal value consumes %d bits, want %d", r.Position(), fixture.bits)
					}
				} else {
					// ACS descriptors describe the fixed carrier. Its selected source
					// prefix can be shorter; the canonical writer fills the carrier.
					canonical, err := d.CanonicalWithContext(decoded, acs)
					if err != nil {
						t.Fatal(err)
					}
					if len(canonical)*8 != fixture.bits {
						t.Fatalf("ACS canonical length=%d bits want %d", len(canonical)*8, fixture.bits)
					}
				}
			}
			checked++
		})
	}
	if checked != len(fixtures) {
		t.Fatalf("checked %d constructions, have %d", checked, len(fixtures))
	}
}
