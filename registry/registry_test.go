package registry

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts44018/measurement"
)

func TestEveryGeneratedDescriptorRejectsMalformedReader(t *testing.T) {
	oversized := runtime.NewReader(make([]byte, (1<<20)/8+1))
	for _, descriptor := range descriptors() {
		if descriptor.DecodeFrom == nil {
			continue // Framing wrappers expose only byte-slice entry points.
		}
		if _, err := descriptor.DecodeFrom(oversized); err == nil {
			t.Fatalf("%s §%s %s accepted oversized reader", descriptor.Standard, descriptor.Clause, descriptor.Name)
		}
		if _, err := descriptor.DecodeFrom(nil); err == nil {
			t.Fatalf("%s §%s %s accepted nil reader", descriptor.Standard, descriptor.Clause, descriptor.Name)
		}
	}
}

// Every generated descriptor must round-trip through its direct encoder.
// The fixed corpus covers short truncations, padding, and random branches.
func TestDirectEncoderCorpus(t *testing.T) {
	accepted := 0
	for i, descriptor := range descriptors() {
		rng := rand.New(rand.NewSource(int64(0x5c51 + i)))
		inputs := [][]byte{{0}, {0x2b}, {0xff}, {0x10, 0x01}, {0, 0}, {0x2b, 0x2b}}
		for j := 0; j < 80; j++ {
			b := make([]byte, 1+rng.Intn(12))
			_, _ = rng.Read(b)
			inputs = append(inputs, b)
		}
		for _, input := range inputs {
			decoded, err := descriptor.Decode(input)
			if err != nil {
				continue
			}
			encoded, err := descriptor.Encode(decoded)
			if err != nil {
				// The three TS 36.331 wrappers use their concrete value in
				// the registry; direct generated descriptors also accept Decoded.
				value := reflect.ValueOf(decoded).FieldByName("Value")
				if value.IsValid() {
					encoded, err = descriptor.Encode(value.Interface())
				}
			}
			accepted++
			if err != nil || !bytes.Equal(encoded, input) {
				t.Fatalf("%s V%s §%s <%s> input=%x encoded=%x err=%v", descriptor.Standard, descriptor.Version, descriptor.Clause, descriptor.Name, input, encoded, err)
			}
		}
	}
	if accepted == 0 {
		t.Fatal("corpus exercised no accepted values")
	}
	t.Logf("%d accepted direct-encoder corpus values across %d descriptors", accepted, len(descriptors()))
	cm3, err := classmark.DecodeClassmark3ValuePart([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	cm3.Value.MultibandSupportedChoice.A5Bits.A5Bits.A57 = 1
	if out, err := classmark.EncodeClassmark3ValuePart(cm3.Value); err != nil || !bytes.Equal(out, []byte{0x08}) {
		t.Fatalf("one-octet Classmark 3 edit: %x, %v", out, err)
	}
	emr, err := measurement.DecodeEnhancedMeasurementReport([]byte{0x10, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	emr.Value.Wire.Tail = runtime.BitString{Bytes: []byte{0xaa}, BitLength: 8}
	if _, err := measurement.EncodeEnhancedMeasurementReport(emr.Value); err == nil {
		t.Fatal("invented EMR tail accepted")
	}
}

func FuzzGeneratedDecodeFrom(f *testing.F) {
	entries := descriptors()
	for i, entry := range entries {
		if entry.DecodeFrom != nil {
			f.Add([]byte{byte(i), 0x2b, 0x2b})
		}
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 || len(data) > 4096 {
			return
		}
		entry := entries[int(data[0])%len(entries)]
		if entry.DecodeFrom == nil {
			return
		}
		reader := runtime.NewReader(data[1:])
		_, _ = entry.DecodeFrom(reader)
	})
}

func TestClauseQualifiedLookup(t *testing.T) {
	_, err := Lookup("TS 24.008", "A5 bits")
	if err == nil || !strings.Contains(err.Error(), "10.5.1.7") || !strings.Contains(err.Error(), "10.5.5.12a") {
		t.Fatalf("ambiguous A5 bits: %v", err)
	}
	first, err := LookupClause("TS 24.008", "10.5.1.7", "A5 bits")
	if err != nil || first.Clause != "10.5.1.7" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := LookupClause("TS 24.008", "10.5.5.12a", "A5 bits")
	if err != nil || second.Clause != "10.5.5.12a" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := Lookup("TS 24.008", "MS RA capability value part"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ standard, clause, name string }{
		{"TS 44.018", "10.5.2.16", "IA Rest Octets"},
		{"TS 44.018", "10.5.2.17", "IAR Rest Octets"},
		{"TS 44.018", "10.5.2.18", "IAX Rest Octets"},
		{"TS 44.060", "12.5.2", "EGPRS Window Size IE"},
		{"TS 44.060", "12.12", "Packet Timing Advance IE"},
	} {
		if _, err := LookupClause(tc.standard, tc.clause, tc.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Lookup("TS 44.018", "PEO IMM Cell Group Details struct"); err == nil || !strings.Contains(err.Error(), "10.5.2.18") {
		t.Fatalf("clause ambiguity was lost: %v", err)
	}
}

func TestOneVersionPerStandard(t *testing.T) {
	entries := descriptors()
	if err := validateVersions(entries); err != nil {
		t.Fatal(err)
	}
	first := entries[0]
	entries = append(entries, runtime.Descriptor{Standard: first.Standard, Version: first.Version + ".other"})
	if err := validateVersions(entries); err == nil {
		t.Fatal("accepted a second version for one standard")
	}
}

func TestSI7ContextRegistry(t *testing.T) {
	data := make([]byte, 20)
	if _, err := Decode("TS 44.018", "SI7 Rest Octets", data); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatalf("missing context error: %v", err)
	} else {
		var required *runtime.ContextRequiredError
		if !errors.As(err, &required) || required.Clause != "10.5.2.36" || required.Name != "SI7 Rest Octets" {
			t.Fatalf("missing typed context details: %v", err)
		}
	}
	if _, err := DecodeWithContext("TS 44.018", "SI7 Rest Octets", data, runtime.SI4ACSOne); err != nil {
		t.Fatal(err)
	}
}
