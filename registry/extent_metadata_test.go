package registry_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

// Context is a property of the available operations. A self-delimiting reader
// can coexist with a canonical API that requires an explicit external extent.
func TestDescriptorContextMetadataMatchesOperations(t *testing.T) {
	descriptors := registry.Descriptors()
	if len(descriptors) != 632 {
		t.Fatalf("catalogue has %d descriptors, want 632", len(descriptors))
	}
	for _, d := range descriptors {
		t.Run(d.Standard+"/"+d.Clause+"/"+d.Name, func(t *testing.T) {
			_, readerErr := d.DecodeFrom(unboundedReaderAtOrigin(t, d, nil))
			// Decode's error result still carries the descriptor's concrete value type.
			// Supplying that type tests canonical extent preflight, rather than the
			// unrelated wrong-dynamic-type check or a guessed semantic constructor.
			boxed, _ := d.Decode(nil)
			value := reflect.ValueOf(boxed).FieldByName("Value").Interface()
			_, canonicalErr := d.Canonical(value)
			var extent *runtime.ExtentError
			canonicalContext := errors.As(canonicalErr, &extent) && extent.Required
			// An empty input does not reach a conditional continuation code.
			// TS 44.018 V19.0.0 table 10.5.2.37h.2 defines code 31.
			var conditionalErr error
			if d.Standard == "TS 44.018" && d.Clause == "10.5.2.37h" && d.Name == "Non-GSM Message struct" {
				_, conditionalErr = d.DecodeFrom(runtime.NewReader([]byte{0x3f, 0}))
			}
			want := d.DecodeWithContext != nil || errors.Is(readerErr, runtime.ErrContextRequired) || errors.Is(conditionalErr, runtime.ErrContextRequired) || canonicalContext
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
						var canonical []byte
						if n*8 < e.MinimumBits {
							canonical, err = canonicalAfterShortReplay(d, value, wire, n*8, acs)
							if err != nil {
								t.Fatalf("short received image: %v", err)
							}
						} else if d.CanonicalWithContext == nil {
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
				for bits := 0; bits < e.MinimumBits && !canonicalSourceCarrier(d); bits++ {
					wire := bytes.Repeat([]byte{fill}, (bits+7)/8)
					r := readerAtOrigin(t, d, wire, len(wire)*8, nil)
					if _, err := r.PushLimit(bits); err != nil {
						t.Fatal(err)
					}
					var value any
					var err error
					if d.DecodeFromWithContext == nil {
						value, err = d.DecodeFrom(r)
					} else {
						value, err = d.DecodeFromWithContext(r, runtime.SI4ACSZero)
					}
					if err == nil {
						if _, err := canonicalAfterShortReplay(d, value, wire, bits, runtime.SI4ACSZero); err != nil {
							t.Fatalf("short bounded image (%d bits): %v", bits, err)
						}
					}
				}
			}
		})
	}
}

// Extent describes canonical source values (docs/m3-api-note.md, Descriptor
// convention), rather than a minimum received-image length. An accepted shorter
// image must still replay every significant bit and canonicalize within that
// source interval; accepting it never exempts either encoding check.
func canonicalAfterShortReplay(d runtime.Descriptor, value any, wire []byte, bits int, acs runtime.SI4ACS) ([]byte, error) {
	if bits < 0 || bits > len(wire)*8 {
		return nil, fmt.Errorf("invalid significant extent %d", bits)
	}
	var replay []byte
	var err error
	if d.EncodeWithContext == nil {
		replay, err = d.Encode(value)
	} else {
		replay, err = d.EncodeWithContext(value, acs)
	}
	if err != nil {
		return nil, fmt.Errorf("plain replay: %w", err)
	}
	expected := append([]byte(nil), wire[:(bits+7)/8]...)
	if bits%8 != 0 {
		expected[len(expected)-1] &= byte(0xff << (8 - bits%8))
	}
	if !bytes.Equal(replay, expected) {
		return nil, fmt.Errorf("plain replay changed significant input: %x != %x/%d", replay, expected, bits)
	}
	var canonical []byte
	if d.CanonicalWithContext == nil {
		canonical, err = d.Canonical(value)
	} else {
		canonical, err = d.CanonicalWithContext(value, acs)
	}
	if err != nil {
		return nil, fmt.Errorf("canonical: %w", err)
	}
	maximum := (d.Extent.MaximumBits + 7) / 8
	if len(canonical)*8 < d.Extent.MinimumBits || d.Extent.Finite && len(canonical) > maximum || d.Extent.Fixed && len(canonical) != maximum {
		return nil, fmt.Errorf("canonical length %d outside %+v", len(canonical), d.Extent)
	}
	return canonical, nil
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

		// TS 44.060 V19.0.0 §11.2.16: source-calculated minimum witness.
		"TS 44.060/11.2.16/Packet Resource Request message content": {49, "00000000000000"},
		// TS 44.060 V19.0.0 §12.42: source-calculated minimum witness.
		"TS 44.060/12.42/PS Handover Radio Resources IE": {49, "00000000000000"},
		// TS 44.060 V19.0.0 §12.42: source-calculated minimum witness.
		"TS 44.060/12.42/Downlink Assignment struct": {17, "000000"},

		// TS 44.018 V19.0.0 §10.5.2.25e: source-calculated minimum witness.
		"TS 44.018/10.5.2.25e/Downlink TBF assignment 2 struct": {17, "000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/Add Frequency list struct": {25, "00000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/Add Iu Mode Only Cell List struct": {25, "00000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/CDMA 2000 Description struct": {21, "000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/REMOVED_3GCELL_Description struct": {17, "000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/Repeated UTRAN FDD Neighbour Cells struct": {21, "000000"},
		// TS 44.060 V19.0.0 §11.2.9b: source-calculated minimum witness.
		"TS 44.060/11.2.9b/Repeated UTRAN TDD Neighbour Cells struct": {21, "000000"},

		// TS 44.060 V19.0.0 §12.46 pp24357–24451: 64+3+5+19+19
		// fixed fields followed by eight absent/mode/list selectors; null release tail.
		"TS 44.060/12.46/DTM Handover PS Radio Resources IE": {118, "000000000000000000000000000000"},
		// §12.46 pp24357–24451: allocation8, absent PFI1, RLC1, TFI5,
		// CONTROL_ACK1 and absent window1: seventeen received bits.
		"TS 44.060/12.46/Downlink Assignment struct": {17, "000000"},
		// TS 44.060 V19.0.0 §11.2.39: two absent options, allocation 8, length 3, identity 1, three absent options.
		"TS 44.060/11.2.39/MBMS bearer description struct": {17, "000800"},
		// TS 44.060 V19.0.0 §11.2.19: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.19/Non-Hopping PCCCH Carriers struct": {18, "000000"},
		// TS 44.060 V19.0.0 §11.2.19: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.19/Reference Frequency struct": {32, "00000000"},
		// TS 44.060 V19.0.0 §11.2.25a: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.25a/CCCH Access Information struct": {49, "00000000000000"},
		// TS 44.060 V19.0.0 §11.2.25a: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.25a/PBCCH Description struct 2": {26, "00000000"},
		// TS 44.060 V19.0.0 §11.2.21b: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.21b/Repeated UTRAN FDD Neighbour Cells struct": {21, "000000"},
		// TS 44.060 V19.0.0 §11.2.21b: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.21b/Repeated UTRAN TDD Neighbour Cells struct": {21, "000000"},
		// TS 44.060 V19.0.0 §11.2.38: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.38/MBMS service request message content": {59, "0000000000000000"},
		// TS 44.060 V19.0.0 §12.48a.2: exact source-path minimum witness retained from the clause validation.
		// TS 44.060 V19.0.0 §12.48a.3: exact source-path minimum witness retained from the clause validation.
		// TS 44.060 V19.0.0 §11.2.6: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.6/Packet Downlink Ack/Nack message content": {101, "00000000000000000000000000"},
		// TS 44.060 V19.0.0 §11.2.6: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.6/Channel Quality Report struct": {23, "000000"},
		// TS 44.060 V19.0.0 §11.2.6c: exact source-path minimum witness retained from the clause validation.
		"TS 44.060/11.2.6c/Packet DBPSCH Downlink Ack/Nack Type 2 message": {23, "080000"},

		// TS 44.018 V19.0.0 tables 9.1.65.2 and 9.1.65.2.1 and
		// TS 44.060 V19.0.0 tables 11.2.5a.4 and 11.2.5a.4.1:
		// printed tags with zero-valued fields and the 19-bit continuation.
		"TS 44.018/9.1.65/EC MULTILATERATION REQUEST message content (Extended Access Burst Method – part 2)":     {30, "80200000"},
		"TS 44.060/11.2.5a/EGPRS MULTILATERATION REQUEST message content (Extended Access Burst Method – part 2)": {30, "80000000"},
		// TS 44.060 V19.0.0 §11.2.8b paragraphs 15423–15424:
		// the 32-bit identity followed by the null extension alternative.
		"TS 44.060/11.2.8b/Packet Uplink Dummy Control Block message content": {32, "00000000"},
		// TS 44.060 V19.0.0 §12.27 pp23460–23479: 31 prefix bits
		// including the absent T3212 selector, then the absent extension bit.
		"TS 44.060/12.27/Non GPRS Cell Options IE": {32, "00000000"},
		// TS 44.018 V19.0.0 §§9.1.21h–9.1.21i pp7638–7683:
		// reconfigure is 8+25+16+1 bits; reconfigure2 is 8+128+16 bits.
		"TS 44.018/9.1.21h/VBS/VGCS RECONFIGURE":          {50, "18000000000000"},
		"TS 44.018/9.1.21h/new Group Channel Description": {25, "00000000"},
		"TS 44.018/9.1.21i/VBS/VGCS RECONFIGURE2":         {152, "1c000000000000000000000000000000000000"},
		// TS 44.018 V19.0.0 §9.1.49b pp10596–10613: 8 header bits,
		// segmented selector, 4-bit reference, 4-bit segment, final selector.
		"TS 44.018/9.1.49b/VGCS SMS INFORMATION": {18, "248000"},

		// TS 44.018 V19.0.0 §§9.1.60, 9.1.61, 9.1.63 and 9.1.69:
		// zero fields, absent optionals and printed tags; one IMSI digit.
		"TS 44.018/9.1.60/EC Fixed Uplink Allocation struct":              {27, "00000000"},
		"TS 44.018/9.1.60/EC Immediate Assignment Type 2 message content": {62, "1000000000000000"},
		"TS 44.018/9.1.61/EC Immediate Assignment Reject message content": {30, "20000000"},
		"TS 44.018/9.1.63/EC Paging Request message content":              {17, "910000"},
		"TS 44.018/9.1.69/EC Paging Indication":                           {88, "8000000000000000000000"},
		// TS 44.060 V19.0.0 §11.2.2 pp13520–13523: TLLI, CTRL_ACK, null tail.
		"TS 44.060/11.2.2/Packet Control Acknowledgement message content": {34, "0000000000"},
		// TS 44.060 V19.0.0 §11.2.20 pp16991–17074: zero counts and absent optionals.
		"TS 44.060/11.2.20/COMPACT Information struct":                {65, "000000000000000000"},
		"TS 44.060/11.2.20/COMPACT Neighbour Cell params struct":      {32, "00000000"},
		"TS 44.060/11.2.20/Iu mode Only Neighbour Cell params struct": {31, "00000000"},
		"TS 44.060/11.2.20/Neighbour Cell params struct":              {31, "00000000"},
		// TS 44.060 V19.0.0 §11.2.24 pp17641–17654: direct ARFCN, zero mapping.
		"TS 44.060/11.2.24/CBCH Channel Description struct": {23, "000000"},
		"TS 44.060/11.2.24/DYNAMIC ARFCN MAPPING":           {31, "00000000"},

		// TS 44.018 V19.0.0 §§9.1.50–9.1.50b: exact 21-octet carriers.
		"TS 44.018/9.1.50/SYSTEM INFORMATION TYPE 10":     {168, "002b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
		"TS 44.018/9.1.50a/SYSTEM INFORMATION TYPE 10bis": {168, "282b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
		"TS 44.018/9.1.50b/SYSTEM INFORMATION TYPE 10ter": {168, "2c0000000000000000000000000000000000032b2b"},
		// TS 44.018 V19.0.0 §10.5.2.11a paragraphs 15114–15117: 3+5+16 bits, then null.
		"TS 44.018/10.5.2.11a/DTM Information Details value part":              {24, "000000"},
		"TS 24.008/10.5.1.6/Mobile Station Classmark 2 value part":             {24, "200000"},
		"TS 36.331/UE-CapabilityRAT-ContainerList field descriptions/geran-cs": {48, "330320000000"},
		"TS 44.018/10.5.2.16/EGPRS Packet Uplink Assignment":                   {32, "00000000"},
		"TS 44.018/10.5.2.16/Multiple Blocks Packet Downlink Assignment":       {48, "000008000000"},
		"TS 44.018/10.5.2.16/Packet Downlink Assignment":                       {36, "000000000a"},
		"TS 44.018/10.5.2.16/Packet Uplink Assignment":                         {24, "800000"},
		"TS 44.018/10.5.2.17/IAR Rest Octets":                                  {24, "000000"},
		"TS 44.018/10.5.2.1e/E-UTRAN Description struct":                       {19, "000000"},
		"TS 44.018/10.5.2.1e/GSM Description struct":                           {17, "000000"},
		"TS 44.018/10.5.2.20/Measurement Results Contents":                     {128, "00000000000000000000000000000000"},
		"TS 44.018/10.5.2.20/NCELL Report struct":                              {17, "000000"},
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
		"TS 44.018/9.1.55/Enhanced Measurement report":                         {168, "10012b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
		"TS 44.018/9.1.55/E-UTRAN Measurement Report struct":                   {20, "000000"},
		"TS 44.018/9.1.55/Repeated Invalid_BSIC_Information struct":            {17, "000000"},
		"TS 44.018/9.1.55/Serving cell data struct":                            {23, "000000"},
		"TS 44.060/12.24/GPRS Cell Options IE":                                 {19, "000000"},
		"TS 44.060/12.33/TMGI IE":                                              {25, "00000000"},
		"TS 44.060/12.61/PLMN-ID struct":                                       {24, "000000"},
		"TS 44.060/12.61/UTRAN CSG Measurement Report IE":                      {63, "0000000000000000"},
		"TS 44.060/12.8/Direct encoding 2 struct":                              {40, "0000000000"},
		"TS 44.060/12.9a/GPRS Power Control Parameters IE":                     {19, "000000"},

		// TS 44.018 V19.0.0 §9.1.58 paragraphs 11515–11523: absent MSISDN, 89 bits.
		"TS 44.018/9.1.58/NOTIFY APPLICATION DATA": {89, "340000000000000000000000"},
		// TS 44.060 V19.0.0 tables 12.60.1, 12.62.1 and 12.63.1:
		"TS 44.060/12.60/E-UTRAN CSG Measurement Report IE": {79, "00000000000000000000"},
		"TS 44.060/12.60/PLMN-ID struct":                    {24, "000000"},
		"TS 44.060/12.62/E-UTRAN CSG Target cell IE":        {45, "000000000000"},
		"TS 44.060/12.62/PLMN-ID struct":                    {24, "000000"},
		"TS 44.060/12.63/UTRAN CSG Target cell IE":          {29, "00000000"},
		"TS 44.060/12.63/PLMN-ID struct":                    {24, "000000"},
		// TS 44.018 V19.0.0 §10.5.2.11b p15136,p15140–15149.
		"TS 44.018/10.5.2.11b/Dynamic ARFCN Mapping":                 {40, "0400000000"},
		"TS 44.018/10.5.2.11b/Dynamic ARFCN Mapping entry":           {31, "00000000"},
		"TS 44.018/10.5.2.14d/GPRS broadcast information value part": {38, "0000000000"},
		// TS 44.018 V19.0.0 §9.1.21a pp7281–7304,7337–7340: header, paging discriminator, LV, channel and absent priority.
		"TS 44.018/9.1.21a/NOTIFICATION FACCH": {28, "0480800b"},
		"TS 44.018/9.1.21a/Paging Information": {19, "010000"},
		// TS 44.060 V19.0.0 §11.2.30a pp18882–18885: 32-bit TLLI, 48-bit RAI and null padding.
		"TS 44.060/11.2.30a/Packet pause message content": {80, "00000000000000000000"},
		// TS 44.060 V19.0.0 §11.2.9 pp15438–15479: the root
		// has 32+1+1+11 bits with no optional PSI5 mark or NC entries;
		// the E-UTRAN report has two count bits and one 18-bit entry.
		"TS 44.060/11.2.9/Packet Measurement Report message content": {45, "000000000000"},
		"TS 44.060/11.2.9/E-UTRAN Measurement Report struct":         {20, "000000"},
		// TS 44.060 V19.0.0 §§12.3 and 12.72: zero sequence number and bitmap.
		"TS 44.060/12.3/Ack/Nack Description IE":     {72, "000000000000000000"},
		"TS 44.060/12.72/EC Ack/Nack Description IE": {21, "000000"},
		// TS 44.060 V19.0.0 §§12.31, 12.32, 12.49 and 12.68:
		// zero frequency and identity, with optional bandwidth absent.
		"TS 44.060/12.31/UTRAN FDD Target cell IE":                    {25, "00000000"},
		"TS 44.060/12.32/UTRAN TDD Target cell IE":                    {24, "000000"},
		"TS 44.060/12.49/E-UTRAN Target cell IE":                      {26, "00000000"},
		"TS 44.060/12.68/E-UTRAN Target cell with extended EARFCN IE": {28, "00000000"},
		// TS 44.060 V19.0.0 tables 12.8.2.1 and 12.8.4.1:
		// absent MAIOs, HSN 0, length 0 and three zero octets.
		"TS 44.060/12.8.2/Dual Carrier Direct encoding 2 struct": {36, "0000000000"},
		"TS 44.060/12.8.4/DLMC Direct encoding 2 struct":         {34, "0000000000"},
		// TS 44.018 V19.0.0 §9.1.43s: 1+2+1+12+1+1 = 18 bits.
		"TS 44.018/9.1.43s/Network Sharing Information struct": {18, "000000"},
		// TS 44.060 V19.0.0 section 11.2.9d: one zero E-UTRAN report,
		// 170 canonical carrier bits, and the 5+6+6-bit invalid-BSIC record.
		"TS 44.060/11.2.9d/E-UTRAN Measurement Report struct":                  {20, "000000"},
		"TS 44.060/11.2.9d/PACKET ENHANCED MEASUREMENT REPORT message content": {170, "00000000000000000000000000000000000000000000"},
		"TS 44.060/11.2.9d/Repeated Invalid_BSIC_Information struct":           {17, "000000"},
		// TS 44.060 V19.0.0 §§12.7, 12.9, 12.11, 12.14, 12.16a and 12.23.
		"TS 44.060/12.11/Packet Request Reference IE":       {27, "00000000"},
		"TS 44.060/12.14/PRACH Control Parameters IE":       {33, "0000000000"},
		"TS 44.060/12.16a/G-RNTI IE":                        {32, "00000000"},
		"TS 44.060/12.23/Cell Identification IE":            {64, "0000000000000000"},
		"TS 44.060/12.7/Channel Request Description IE":     {24, "000000"},
		"TS 44.060/12.9/Global Power Control Parameters IE": {24, "000000"},
		// TS 44.018 V19.0.0 §10.5.2.8b: 2+24+1+1 bits,
		// with PFI absent; byte completion selects the L release tail.
		"TS 44.018/10.5.2.8b/Channel Request Description 2 value part": {28, "0000000b"},
		// TS 44.018 V19.0.0 §10.5.2.25d: 2+1+8 mandatory
		// fields, five absent selectors, then the required zero bit.
		"TS 44.018/10.5.2.25d/RR Packet Downlink Assignment value part": {17, "b4a000"},
		// TS 44.060 V19.0.0 §11.2.6a: 5+1+4 selectors,
		// followed by §12.3.1's direct selector and 15 mandatory bits.
		"TS 44.060/11.2.6a/EGPRS Packet Downlink Ack/Nack message content": {26, "00000000"},
	}

	// TS 44.060 V19.0.0 §§11.2.3 p13605–13608, 11.2.3a
	// p13715–13731, 11.2.5b p14305–14340 and 11.2.32 p19912–19914.
	// The notification escape's 6+4 minimum is exercised by accepted samples.
	for key, fixture := range map[string]struct {
		bits int
		hex  string
	}{
		"TS 44.060/11.2.3/Packet Cell Change Failure message content":       {52, "00000000000000"},
		"TS 44.060/11.2.3a/E-UTRAN CCN Measurement Report struct":           {21, "000000"},
		"TS 44.060/11.2.3a/E-UTRAN Target Cell struct":                      {32, "00000000"},
		"TS 44.060/11.2.3a/E-UTRAN Target Cell with extended EARFCN struct": {34, "0000000000"},
		// 6+8+1+1+(1+8)+1+3+1+10; the last selector chooses ARFCN.
		"TS 44.060/11.2.5b/Channel Description struct": {40, "0000000400"},
		"TS 44.060/11.2.5b/Response Time struct":       {20, "000000"},
		// Address selector + Global TFI + MS RA struct (4+7+0+1).
		"TS 44.060/11.2.32/Additional MS Radio Access Capabilities message content": {19, "000000"},
	} {
		fixtures[key] = fixture
	}

	// TS 44.018 V19.0.0 §§10.5.2.1c and 10.5.2.25a: minimum range and direct channel fields.
	fixtures["TS 44.018/10.5.2.1c/RANGE LIMITS"] = struct {
		bits int
		hex  string
	}{20, "000000"}
	fixtures["TS 44.018/10.5.2.25a/Packet Channel Description"] = struct {
		bits int
		hex  string
	}{24, "080000"}
	// TS 44.060 §12.42b: 13 initial bits, DOWNLINK_RLC_MODE/CONTROL_ACK,
	// repeat selector 1 and reserved UFPS 11 terminate at bit 18.
	fixtures["TS 44.060/12.42b/PS Handover Radio Resources 3 IE"] = struct {
		bits int
		hex  string
	}{18, "0001c0"}

	// TS 44.060 V19.0.0 §11.2.29: 3 TIMESLOT_NUMBER bits,
	// two absent option selectors, and §12.21's 14-bit relative frame
	// description give 19 bits. Multi Block Allocation adds its 2-bit count.
	fixtures["TS 44.060/11.2.29/Single Block Allocation struct"] = struct {
		bits int
		hex  string
	}{19, "040000"}
	fixtures["TS 44.060/11.2.29/Multi Block Allocation struct"] = struct {
		bits int
		hex  string
	}{21, "040000"}

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
				r := readerAtOrigin(t, d, wire, len(wire)*8, nil)
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
					if r.Position() != fixture.bits+registry.SourceOrigin(d) {
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

// TS 44.060 V19.0.0 §§12.48a.2–12.48a.3 normal sender paths remain
// 23/24 bits; the shorter receiver-error paths now supply descriptor minima.
func TestAssignmentNormalPathConstructions(t *testing.T) {
	for _, tc := range []struct {
		clause, name, hex string
		bits              int
	}{
		{"12.48a.2", "Single Downlink Assignment 2 IE", "020000", 23},
		{"12.48a.3", "Single Uplink Assignment 2 IE", "000000", 24},
	} {
		t.Run(tc.clause, func(t *testing.T) {
			d, err := registry.LookupClause("TS 44.060", tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			data, err := hex.DecodeString(tc.hex)
			if err != nil {
				t.Fatal(err)
			}
			r, err := runtime.NewChoiceReaderBits(data, tc.bits, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = d.DecodeFrom(r); err != nil || r.Position() != tc.bits {
				t.Fatalf("normal source path consumed%d/%d: %v", r.Position(), tc.bits, err)
			}
		})
	}
}
