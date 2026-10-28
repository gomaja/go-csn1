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
