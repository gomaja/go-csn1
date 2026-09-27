package restoctets

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestOtherRestOctetNamedVectors(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.32–35, .37a, .37e, .37j–k,
	// .37m–n and .44: L selects absent optional fields and 0x2b
	// supplies the specified L/H spare-padding pattern (TS 44.060 §11).
	for _, tc := range []struct {
		clause, name string
		length       int
		fill         byte
		empty        bool
	}{
		{"10.5.2.25", "P3 Rest Octets", 3, 0x2b, false},
		{"10.5.2.32", "SI1 Rest Octets", 1, 0x2b, false},
		{"10.5.2.33", "SI2bis Rest Octets", 1, 0x2b, false},
		{"10.5.2.33a", "SI2ter Rest Octets", 4, 0x2b, false},
		{"10.5.2.33c", "SI2n Rest Octets", 20, 0x2b, false},
		{"10.5.2.34", "SI3 Rest Octet", 4, 0x2b, false},
		{"10.5.2.35", "SI4 Rest Octets", 1, 0x2b, true},
		{"10.5.2.35a", "SI6 rest octets", 7, 0x2b, false},
		{"10.5.2.37a", "SI9 rest octets", 17, 0x2b, false},
		{"10.5.2.37e", "SI16 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37f", "SI17 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37j", "SI14 Rest Octets", 16, 0x2b, false},
		{"10.5.2.37k", "SI15 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37l", "SI 13alt Rest Octets", 20, 0x2b, false},
		{"10.5.2.37m", "SI 21 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37n", "SI 22 Rest Octets", 20, 0x2b, false},
		{"10.5.2.37o", "SI 23 Rest Octets", 20, 0x2b, false},
		{"10.5.2.44", "SI10 rest octets", 20, 0x2b, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, err := LookupClause(tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := definition.Decode(nil); (err != nil && errors.Is(err, runtime.ErrEmptyValue)) != tc.empty || err == nil {
				t.Fatalf("zero length policy: %v", err)
			}
			wire := bytes.Repeat([]byte{tc.fill}, tc.length)
			decoded, err := definition.Decode(wire)
			if err != nil {
				t.Fatalf("decode %x: %v", wire, err)
			}
			encoded, err := definition.Encode(decoded)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("descriptor round trip %x -> %x: %v", wire, encoded, err)
			}
			if tc.length > 1 && !tc.empty {
				if _, err := definition.Decode(wire[:tc.length-1]); err == nil {
					t.Fatal("short value accepted")
				}
			}
			if !tc.empty {
				if _, err := definition.Decode(append(wire, 0x2b)); err == nil {
					t.Fatal("overlength value accepted")
				}
			}
		})
	}
}

func TestSI7AndSI8RequireACSContext(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.35 selects the SI7/SI8 grammar
	// from ACS in the containing SI4 message, outside these 20 octets.
	for _, tc := range []struct{ clause, name string }{{"10.5.2.36", "SI7 Rest Octets"}, {"10.5.2.37", "SI8 Rest Octets"}} {
		d, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
		}
		if _, err := d.Decode(make([]byte, 20)); !errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("missing ACS conflict: %v", err)
		}
	}
}

func TestSI18AndSI20FailClosedOnTerminatorConflict(t *testing.T) {
	// TS 44.018 V19.0.0 §§10.5.2.37h–i: the source's
	// zero-length Non-GSM Message terminator is disputed by the
	// independent pycrate decoder; the standalone values stay closed.
	for _, tc := range []struct{ clause, name string }{{"10.5.2.37h", "SI 18 Rest Octets"}, {"10.5.2.37i", "SI 20 Rest Octets"}} {
		d, err := LookupClause(tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
		}
		wire := append([]byte{0x03, 0x00}, bytes.Repeat([]byte{0x2b}, 18)...)
		if _, err := d.Decode(wire); !errors.Is(err, runtime.ErrUnsupported) {
			t.Fatalf("missing terminator conflict: %v", err)
		}
	}
}

func TestSI19FailsClosedOnCountConflict(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.37g gives contradictory
	// NR_OF_REMAINING_CELLS count and range text.
	d, err := LookupClause("10.5.2.37g", "SI 19 Rest Octets")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Decode(nil); err == nil || errors.Is(err, runtime.ErrUnsupported) {
		t.Fatalf("zero length must fail the fixed 20 octet length: %v", err)
	}
	if _, err := d.Decode(bytes.Repeat([]byte{0x2b}, 20)); !errors.Is(err, runtime.ErrUnsupported) {
		t.Fatalf("missing count conflict: %v", err)
	}
}
