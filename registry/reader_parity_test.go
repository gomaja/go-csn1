package registry_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

// readerOutcome is what a caller can observe from one decode: acceptance, the
// bits consumed, the received tail, and the replayed and canonical bytes.
type readerOutcome struct {
	ok            bool
	bits          int
	tail          string
	replay, canon string
	replayFailed  bool
	canonFailed   bool
}

func observeDecode(d runtime.Descriptor, decoded any, err error) readerOutcome {
	if err != nil {
		return readerOutcome{}
	}
	out := readerOutcome{ok: true, bits: -1, tail: "absent"}
	if v := reflect.ValueOf(decoded); v.Kind() == reflect.Struct {
		if f := v.FieldByName("BitsConsumed"); f.IsValid() {
			out.bits = int(f.Int())
		}
		if f := v.FieldByName("Tail"); f.IsValid() {
			tail := f.Interface().(runtime.BitString)
			out.tail = fmt.Sprintf("%d:%x", tail.BitLength, tail.Bytes)
		}
	}
	replay, err := d.Encode(decoded)
	out.replay, out.replayFailed = hex.EncodeToString(replay), err != nil
	canon, err := d.Canonical(decoded)
	out.canon, out.canonFailed = hex.EncodeToString(canon), err != nil
	return out
}

func readerBoundTo(t *testing.T, data []byte) *runtime.Reader {
	t.Helper()
	r := runtime.NewReader(data)
	if _, err := r.PushLimit(len(data) * 8); err != nil {
		t.Fatal(err)
	}
	return r
}

// knownReaderDifference reports the one entry-point difference the catalogue
// keeps: in TS 44.018 V19.0.0 §10.5.2.37h, NR OF CONTAINER OCTETS 31 makes the
// container run to the end of the enclosing SI value. The byte entry point has
// no enclosing value and refuses it; a bounded reader supplies one, and can
// then either decode the container or reject its content under that extent.
func knownReaderDifference(d runtime.Descriptor, byteErr error) bool {
	return d.Standard == "TS 44.018" && d.Clause == "10.5.2.37h" && d.Name == "Non-GSM Message struct" &&
		errors.Is(byteErr, runtime.ErrContextRequired)
}

// sameReaderError compares the public diagnostics, including errors wrapped by
// OperationError. An identical message alone can hide a different error kind.
func sameReaderError(a, b error) bool {
	var ao, bo *runtime.OperationError
	if errors.As(a, &ao) != errors.As(b, &bo) {
		return false
	}
	if ao != nil && (ao.Operation != bo.Operation || ao.Kind != bo.Kind || ao.Location != bo.Location) {
		return false
	}
	var ad, bd *runtime.DecodeError
	if errors.As(a, &ad) != errors.As(b, &bd) {
		return false
	}
	return ad == nil || *ad == *bd
}

// TestReaderEntryMatchesByteEntry decodes the same octets through every
// descriptor's byte entry point and through its reader entry point bounded to
// those octets, and requires the same acceptance, bits consumed, tail, replay
// and canonical bytes. The bounded reader must also preserve every typed field
// and diagnostic. Each definition receives 2,000 fixed-fill and seeded random
// inputs through two octets past its canonical ceiling (40 when unbounded).
func TestReaderEntryMatchesByteEntry(t *testing.T) {
	started := time.Now()
	rng := rand.New(rand.NewSource(20261026))
	compare := func(d runtime.Descriptor, form string, data []byte, byteValue any, byteErr error, readerValue any, readerErr error) {
		t.Helper()
		if form != "report" && form != "report context" && knownReaderDifference(d, byteErr) {
			return
		}
		if form != "report" && form != "report context" {
			if (byteErr == nil) != (readerErr == nil) || (byteErr != nil && (byteErr.Error() != readerErr.Error() || !sameReaderError(byteErr, readerErr))) {
				t.Fatalf("%s §%s %s (%s) on %x: byte error %#v (%v), reader error %#v (%v)", d.Standard, d.Clause, d.Name, form, data, byteErr, byteErr, readerErr, readerErr)
				return
			}
			if !reflect.DeepEqual(byteValue, readerValue) {
				t.Fatalf("%s §%s %s (%s) on %x: typed value differs", d.Standard, d.Clause, d.Name, form, data)
				return
			}
			return
		}
		b, r := observeDecode(d, byteValue, byteErr), observeDecode(d, readerValue, readerErr)
		if b != r {
			t.Errorf("%s §%s %s (%s) on %x: byte entry %+v (%v), reader entry %+v (%v)", d.Standard, d.Clause, d.Name, form, data, b, byteErr, r, readerErr)
		}
	}
	// With no containing extent, caller octets must preserve complete values
	// and context-required refusals. Other refusals can be incomplete input:
	// for example Classmark 2 needs three octets before it can decode.
	checkCallerTail := func(d runtime.Descriptor, form string, data []byte, decode func(*runtime.Reader) (any, error)) {
		t.Helper()
		before, beforeErr := decode(runtime.NewReader(data))
		if beforeErr != nil && !errors.Is(beforeErr, runtime.ErrContextRequired) {
			return
		}
		afterData := append(append([]byte(nil), data...), 0x5a, 0xa5)
		after, afterErr := decode(runtime.NewReader(afterData))
		if errors.Is(afterErr, runtime.ErrContextRequired) {
			return
		}
		if beforeErr != nil && afterErr == nil {
			t.Fatalf("%s %s (%s) on %x: caller tail rescued refusal %v", d.Standard, d.Name, form, data, beforeErr)
		}
		if beforeErr == nil && (afterErr != nil || !reflect.DeepEqual(before, after)) {
			t.Fatalf("%s %s (%s) on %x: caller tail changed value: %v", d.Standard, d.Name, form, data, afterErr)
		}
	}
	inputs := 0
	for _, d := range registry.Descriptors() {
		limit := d.MaxOctets + 2
		if d.MaxOctets == 0 {
			limit = 40
		}
		for i := 0; i < 2000; i++ {
			n := i % (limit + 1)
			data := make([]byte, n)
			fills := []byte{0, 0xff, 0x55, 0xaa}
			if group := i / (limit + 1); group < len(fills) {
				for j := range data {
					data[j] = fills[group]
				}
			} else {
				rng.Read(data)
			}
			inputs++
			if d.Decode == nil || d.DecodeFrom == nil {
				t.Fatalf("%s %s: missing entry point", d.Standard, d.Name)
			}
			bv, be := d.Decode(data)
			rv, re := d.DecodeFrom(readerBoundTo(t, data))
			compare(d, "plain", data, bv, be, rv, re)
			checkCallerTail(d, "plain", data, d.DecodeFrom)
			if i < 16*(limit+1) {
				// Keep four fills and twelve random report cases at every length.
				pv, _, pe := d.DecodeWithReport(data)
				compare(d, "report", data, bv, be, pv, pe)
				if be == nil && pe == nil && !runtime.SemanticallyEqual(bv, pv) {
					t.Errorf("%s %s: report changes typed value", d.Standard, d.Name)
				}
				var before, after *runtime.DecodeError
				if errors.As(be, &before) && errors.As(pe, &after) && (before.Kind != after.Kind || before.Offset != after.Offset) {
					t.Errorf("%s %s: report decode error %v differs from %v", d.Standard, d.Name, pe, be)
				}
			}
			if d.DecodeWithContext != nil {
				if d.DecodeFromWithContext == nil {
					t.Fatalf("%s %s: missing context reader", d.Standard, d.Name)
				}
				for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne, runtime.SI4ACS(2)} {
					bv, be := d.DecodeWithContext(data, acs)
					rv, re := d.DecodeFromWithContext(readerBoundTo(t, data), acs)
					compare(d, fmt.Sprintf("SI4 ACS %d", acs), data, bv, be, rv, re)
					checkCallerTail(d, fmt.Sprintf("SI4 ACS %d", acs), data, func(r *runtime.Reader) (any, error) { return d.DecodeFromWithContext(r, acs) })
					if i < 16*(limit+1) {
						pv, _, pe := d.DecodeWithReportWithContext(data, acs)
						compare(d, "report context", data, bv, be, pv, pe)
					}
				}
			}
		}
	}
	t.Logf("%d definitions, %d inputs, 2000 per definition; elapsed %s", len(registry.Descriptors()), inputs, time.Since(started))
	if inputs < 60000 {
		t.Fatalf("only %d inputs compared", inputs)
	}
}

// The two shapes found in review: the GERAN-PS reader must not infer zeros
// across the whole MS RA capability, and the Classmark 2 descriptor reader
// must keep bounded octets after its three-octet value.
func TestReaderEntryReviewCases(t *testing.T) {
	ps, err := registry.Lookup("TS 36.331", "geran-ps")
	if err != nil {
		t.Fatal(err)
	}
	data := []byte{0xf2, 0x57, 0x86, 0x38, 0x9f}
	if _, err := ps.Decode(data); err == nil {
		t.Fatal("byte entry accepted a truncated GERAN-PS value")
	}
	if _, err := ps.DecodeFrom(readerBoundTo(t, data)); err == nil {
		t.Fatal("reader entry accepted a GERAN-PS value the byte entry refuses")
	}
	cm2, err := registry.Lookup("TS 24.008", "Mobile Station Classmark 2 value part")
	if err != nil {
		t.Fatal(err)
	}
	data = []byte{0x20, 0x00, 0x00, 0xa5}
	decoded, err := cm2.DecodeFrom(readerBoundTo(t, data))
	if err != nil {
		t.Fatal(err)
	}
	out, err := cm2.Encode(decoded)
	if err != nil || !bytes.Equal(out, data) {
		t.Fatalf("Classmark 2 reader entry replays %x (%v), want %x", out, err, data)
	}
}

// TS 44.018 V19.0.0 §§10.5.2.1e, 10.5.2.20, 10.5.2.35–37,
// 10.5.2.37h–i define source value sizes. A caller's trailing bytes do not
// supply the missing containing extent for those source size checks.
func TestReaderSourceBoundsRequireContainingLimit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		octets int
	}{
		{"SI7 Rest Octets", 20}, {"SI8 Rest Octets", 20},
		{"SI 18 Rest Octets", 20}, {"SI 20 Rest Octets", 20},
		{"Measurement Results Contents", 16},
		{"Cell Selection Indicator after release of all TCH and SDCCH value part", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := registry.Lookup("TS 44.018", tc.name)
			if err != nil {
				t.Fatal(err)
			}
			for _, n := range []int{0, tc.octets - 1, tc.octets, tc.octets + 1} {
				data := make([]byte, n)
				for _, tail := range [][]byte{nil, {0x5a}} {
					input := append(append([]byte(nil), data...), tail...)
					if _, err := d.DecodeFrom(runtime.NewReader(input)); !errors.Is(err, runtime.ErrContextRequired) {
						t.Errorf("plain %x: %v", input, err)
					}
					if d.DecodeFromWithContext != nil {
						for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
							if _, err := d.DecodeFromWithContext(runtime.NewReader(input), acs); !errors.Is(err, runtime.ErrContextRequired) {
								t.Errorf("ACS %d %x: %v", acs, input, err)
							}
						}
					}
				}
			}
		})
	}
}

func TestReaderContextRequiredKeepsPrintedName(t *testing.T) {
	for _, d := range registry.Descriptors() {
		_, err := d.DecodeFrom(runtime.NewReader([]byte{0x2b}))
		var context *runtime.ContextRequiredError
		if errors.As(err, &context) {
			if context.Name != d.Name {
				t.Errorf("%s: context name %q, want %q", d.Standard, context.Name, d.Name)
			}
		}
	}
}

func TestReaderUsesOwnZeroExtensionPolicy(t *testing.T) {
	for _, tc := range []struct {
		standard, name string
		data           []byte
	}{
		{"TS 24.008", "A5 bits", nil},
		{"TS 24.008", "A5 bits", []byte{0}},
		{"TS 44.018", "GSM Description struct", []byte{0, 0}},
		{"TS 44.018", "GSM Description struct", []byte{0, 0, 0, 0}},
	} {
		t.Run(tc.name+fmt.Sprintf("/%x", tc.data), func(t *testing.T) {
			var d runtime.Descriptor
			for _, candidate := range registry.Descriptors() {
				if candidate.Standard == tc.standard && candidate.Name == tc.name {
					d = candidate
					break
				}
			}
			if d.Decode == nil {
				t.Fatal("missing descriptor")
			}
			want, we := d.Decode(tc.data)
			for _, setting := range []bool{false, true} {
				r := readerBoundTo(t, tc.data)
				r.SetZeroExtension(setting)
				got, ge := d.DecodeFrom(r)
				if fmt.Sprint(we) != fmt.Sprint(ge) || !sameReaderError(we, ge) || !reflect.DeepEqual(want, got) {
					t.Errorf("caller zero extension %t: byte error %v, reader error %v, equal %t", setting, we, ge, reflect.DeepEqual(want, got))
				}
				if r.ZeroExtension() != setting {
					t.Errorf("caller zero extension changed from %t", setting)
				}
			}
		})
	}
}
