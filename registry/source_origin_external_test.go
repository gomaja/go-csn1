package registry_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

func readerAtOrigin(t testing.TB, d runtime.Descriptor, data []byte, bits int, c *runtime.Collector) *runtime.Reader {
	t.Helper()
	origin := registry.SourceOrigin(d)
	w := runtime.NewWriter()
	if err := w.WriteUint(0, origin); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteBitString(runtime.BitString{Bytes: data, BitLength: bits}, bits); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	return r
}

func descriptorReader(t testing.TB, d runtime.Descriptor, data []byte) *runtime.Reader {
	t.Helper()
	bits := len(data) * 8
	if registry.SourceOrigin(d) != 0 && d.Extent.Fixed && d.Extent.MaximumBits%8 != 0 && bits == (d.Extent.MaximumBits+7)/8*8 {
		bits = d.Extent.MaximumBits
	}
	return readerAtOrigin(t, d, data, bits, nil)
}

func TestEveryOrdinaryContentOrigin(t *testing.T) {
	roots := 0
	for _, d := range registry.Descriptors() {
		origin := registry.SourceOrigin(d)
		if origin == 0 {
			continue
		}
		roots++
		if origin != 14 {
			t.Fatalf("%s: origin %d", d.Name, origin)
		}
		t.Run(d.Clause+"/"+d.Name, func(t *testing.T) {
			for phase := 0; phase < 8; phase++ {
				if phase == 6 {
					continue
				}
				r := readerAtOrigin(t, runtime.Descriptor{}, make([]byte, 32), 256, nil)
				if _, err := r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				_, err := d.DecodeFrom(r)
				if !isContentPhaseError(err) || r.Position() != phase {
					t.Fatalf("phase %d: cursor %d, error %v", phase, r.Position(), err)
				}
			}
			if d.SourceOrigin != origin {
				t.Fatalf("descriptor origin %d, recorded %d", d.SourceOrigin, origin)
			}
			accepted := 0
			for _, n := range []int{0, 1, 2, 8, 12, 22, 23} {
				for _, fill := range []byte{0, 0x2b, 0xff} {
					data := bytes.Repeat([]byte{fill}, n)
					bv, be := d.Decode(data)
					checkDescriptorBitParity(t, d, data)
					rv, re := d.DecodeFrom(descriptorReader(t, d, data))
					if (be == nil) != (re == nil) {
						t.Fatalf("%d bytes: byte %v, reader %v", n, be, re)
					}
					if be != nil {
						if !sameReaderError(be, re) {
							t.Fatalf("%d bytes: diagnostics differ: %v / %v", n, be, re)
						}
						continue
					}
					accepted++
					bvalue := reflect.ValueOf(bv).FieldByName("Value").Interface()
					rvalue := reflect.ValueOf(rv).FieldByName("Value").Interface()
					if !runtime.SemanticallyEqual(bvalue, rvalue) {
						t.Fatalf("%d bytes: typed semantic values differ", n)
					}
					replay, e := d.Encode(bv)
					if e != nil || !bytes.Equal(replay, data) {
						t.Fatalf("%d bytes: byte replay: %v", n, e)
					}
				}
			}
			if accepted == 0 {
				t.Fatal("no accepted synthetic control content")
			}
		})
	}
	if roots != 43 {
		t.Fatalf("source-coordinate roots %d, want 43", roots)
	}
}

func isContentPhaseError(err error) bool {
	var phase interface {
		error
		SourceOrigin() int
		BitPosition() int
	}
	return errors.As(err, &phase) && phase.SourceOrigin() == 14 && phase.BitPosition() >= 0
}

// TS 44.060 section 11 and TS 24.007 Annex B B.1.2.2: after the
// 80-bit Pause fields and padding selector, each L uses the block coordinate.
func TestContentCanonicalPaddingUsesBlockPhase(t *testing.T) {
	d, err := registry.Lookup("TS 44.060", "Packet pause message content")
	if err != nil {
		t.Fatal(err)
	}
	v, err := d.Decode(make([]byte, 22))
	if err != nil {
		t.Fatal(err)
	}
	out, err := d.CanonicalAtLength(v, 22)
	if err != nil {
		t.Fatal(err)
	}
	different := false
	for bit := 81; bit < 176; bit++ {
		got := (out[bit/8] >> uint(7-bit%8)) & 1
		want := (byte(0x2b) >> uint(7-(14+bit)%8)) & 1
		if got != want {
			t.Fatalf("padding bit %d: got %d want %d", bit, got, want)
		}
		if want != (byte(0x2b)>>uint(7-bit%8))&1 {
			different = true
		}
	}
	if !different {
		t.Fatal("padding vector does not distinguish octet phases")
	}
}

func unboundedReaderAtOrigin(t testing.TB, d runtime.Descriptor, data []byte) *runtime.Reader {
	t.Helper()
	origin := registry.SourceOrigin(d)
	if origin == 0 {
		return runtime.NewReader(data)
	}
	w := runtime.NewWriter()
	if err := w.WriteUint(0, origin); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteBitString(runtime.BitString{Bytes: data, BitLength: len(data) * 8}, len(data)*8); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r := runtime.NewReader(raw.Bytes)
	if _, err := r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	return r
}

// The fixture contains synthetic CS-1 blocks decoded by unmodified upstream
// pycrate 0.7.11. TS 44.060 V19.0.0 sections 10.3.1–10.3.2 and the cited
// 11.2 clauses define the headers, concrete fields and padding interval.
func TestContentByteFunctionsAgainstBlockOracle(t *testing.T) {
	var cases []struct {
		Clause, Name, Block, Content string
		Origin                       int
		SignificantBits              int `json:"significant_bits"`
		Scalars                      map[string][]uint64
	}
	fixture, err := os.ReadFile("testdata/ordinary-content-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(fixture, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 9 {
		t.Fatalf("ordinary source blocks %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.Clause, func(t *testing.T) {
			d, err := registry.LookupClause("TS 44.060", tc.Clause, tc.Name)
			if err != nil {
				t.Fatal(err)
			}
			block, err := hex.DecodeString(tc.Block)
			if err != nil {
				t.Fatal(err)
			}
			content, err := hex.DecodeString(tc.Content)
			if err != nil {
				t.Fatal(err)
			}
			r, err := runtime.NewReaderBits(block, 184, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.ReadUint(tc.Origin); err != nil {
				t.Fatal(err)
			}
			from, err := d.DecodeFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := d.Decode(content)
			if err != nil {
				t.Fatal(err)
			}
			if tc.SignificantBits != 184-tc.Origin || registry.SourceOrigin(d) != tc.Origin || r.Position() != 184 {
				t.Fatalf("block coordinates: origin %d, end %d", registry.SourceOrigin(d), r.Position())
			}
			fv := reflect.ValueOf(from).FieldByName("Value").Interface()
			bv := reflect.ValueOf(decoded).FieldByName("Value").Interface()
			if !runtime.SemanticallyEqual(fv, bv) {
				t.Fatal("byte function differs from the complete block's typed content")
			}
			fields := make(map[string][]uint64)
			collectContentScalars(reflect.ValueOf(bv), fields)
			for name, want := range tc.Scalars {
				got := fields[contentFieldName(name)]
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("oracle field %s: got %v want %v", name, got, want)
				}
			}
			replay, err := d.Encode(decoded)
			if err != nil || !bytes.Equal(replay, content) {
				t.Fatalf("byte replay: %v", err)
			}
		})
	}
}

func contentFieldName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, name)
}
func collectContentScalars(v reflect.Value, out map[string][]uint64) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		if name == "Wire" {
			continue
		}
		field := v.Field(i)
		if field.Kind() >= reflect.Uint && field.Kind() <= reflect.Uint64 {
			out[contentFieldName(name)] = append(out[contentFieldName(name)], field.Uint())
			continue
		}
		if name == "MAXRETRANSList" {
			var packed uint64
			for j := 0; j < field.Len(); j++ {
				packed = packed<<2 | field.Index(j).Uint()
			}
			out["maxretrans"] = []uint64{packed}
			continue
		}
		collectContentScalars(field, out)
	}
}

// A fixed, non-octet source carrier has backing completion only at its byte
// entry. Compare every other wire field exactly after excluding that completion.
func sameSourceDecoded(d runtime.Descriptor, data []byte, byteValue, readerValue any) bool {
	if reflect.DeepEqual(byteValue, readerValue) {
		return true
	}
	if registry.SourceOrigin(d) == 0 || !d.Extent.Fixed || d.Extent.MaximumBits%8 == 0 || len(data) != (d.Extent.MaximumBits+7)/8 {
		return false
	}
	b, r := reflect.ValueOf(byteValue), reflect.ValueOf(readerValue)
	if !b.IsValid() || b.Type() != r.Type() || b.Kind() != reflect.Struct {
		return false
	}
	byteWire := b.FieldByName("Value").FieldByName("Wire").Interface().(runtime.WireInfo)
	readerWire := r.FieldByName("Value").FieldByName("Wire").Interface().(runtime.WireInfo)
	completion := len(data)*8 - d.Extent.MaximumBits
	if byteWire.TransmittedBits != len(data)*8 || readerWire.TransmittedBits != d.Extent.MaximumBits || byteWire.BitsConsumed != readerWire.BitsConsumed || byteWire.Tail.BitLength != readerWire.Tail.BitLength+completion {
		return false
	}
	for i := 0; i < readerWire.Tail.BitLength; i++ {
		if (byteWire.Tail.Bytes[i/8]>>uint(7-i%8))&1 != (readerWire.Tail.Bytes[i/8]>>uint(7-i%8))&1 {
			return false
		}
	}
	normalized := reflect.New(b.Type()).Elem()
	normalized.Set(b)
	byteWire.TransmittedBits = readerWire.TransmittedBits
	byteWire.Tail = readerWire.Tail
	normalized.FieldByName("Value").FieldByName("Wire").Set(reflect.ValueOf(byteWire))
	normalized.FieldByName("Tail").Set(r.FieldByName("Tail"))
	return reflect.DeepEqual(normalized.Interface(), readerValue)
}

// R2 Extent describes canonical source bits. A complete content carrier may
// retain a shorter received image; its canonical minimum is checked after
// canonical encoding, while the source grammar checks receiver field bounds.
func canonicalSourceCarrier(d runtime.Descriptor) bool {
	return registry.SourceOrigin(d) != 0 && d.Extent.Fixed
}
