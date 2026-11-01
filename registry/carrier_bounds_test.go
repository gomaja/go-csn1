package registry

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/messages"
)

// TS 44.018 V19.0.0 §§9.1.7, 9.1.43p/r/s, 9.1.58, 9.1.62;
// TS 44.006 V19.0.0 §§5.2, 6.4a, 8.8.3, 8.8.5; TS 45.002
// V19.0.0 §§6.3.4.2, 6.3.4.4; TS 45.003 V19.0.0 §§5.2a.1, 5.2b.1.
// Each synthetic prefix is followed by the printed spare-padding pattern.
func TestAuthenticatedCarrierBoundaries(t *testing.T) {
	for _, tc := range []struct {
		clause, name string
		maximum      int
		prefix       []byte
	}{
		{"10.5.2.1c", "BA List Pref", 247, []byte{246}},
		{"9.1.43p", "EC System Information Type 1", 23, []byte{0x20, 0x01}},
		{"9.1.43r", "EC System Information Type 3", 23, []byte{0x60, 0x00}},
		{"9.1.43s", "EC System Information Type 4", 23, []byte{0x80, 0x03}},
		{"9.1.58", "NOTIFY APPLICATION DATA", 23, []byte{0x34, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{"9.1.62", "EC-Dummy message content", 11, []byte{0x33}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := LookupClause("TS 44.018", tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			wire := bytes.Repeat([]byte{0x2b}, tc.maximum)
			copy(wire, tc.prefix)
			value, err := d.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := d.CanonicalAtLength(value, tc.maximum)
			if err != nil || len(encoded) != tc.maximum {
				t.Fatalf("boundary: %x %v", encoded, err)
			}
			for _, n := range []int{tc.maximum + 1, 300} {
				encoded, err = d.CanonicalAtLength(value, n)
				var extent *runtime.ExtentError
				if encoded != nil || !errors.As(err, &extent) || extent.Maximum != tc.maximum {
					t.Fatalf("length %d: %x %v", n, encoded, err)
				}
			}
			// The new sender maximum does not tighten tolerant byte reception.
			longer := append(append([]byte(nil), wire...), 0x2b)
			received, err := d.Decode(longer)
			if err != nil {
				t.Fatal("receiver policy changed:", err)
			}
			replay, err := d.Encode(received)
			if err != nil || !bytes.Equal(replay, longer) {
				t.Fatalf("received replay %x %v", replay, err)
			}
		})
	}
}

func TestEarlierCarrierLimitsRemainEffective(t *testing.T) {
	for _, tc := range []struct {
		clause, name, input string
		maximum             int
	}{
		{"10.5.2.71", "SI10ter Rest Octets", "ff69b8a1", 4},
		{"10.5.2.44", "neighbour information", "cf7e5bc53b3f", 20},
		{"10.5.2.62", "MBMS Session Identity value part", "00", 1},
	} {
		d, err := LookupClause("TS 44.018", tc.clause, tc.name)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := hex.DecodeString(tc.input)
		if err != nil {
			t.Fatal(err)
		}
		value, err := d.Decode(wire)
		if err != nil {
			t.Fatal(err)
		}
		if b, e := d.CanonicalAtLength(value, tc.maximum); e != nil || len(b) != tc.maximum {
			t.Fatalf("%s boundary: %x %v", tc.name, b, e)
		}
		for _, n := range []int{21, 300} {
			b, e := d.CanonicalAtLength(value, n)
			var extent *runtime.ExtentError
			if b != nil || !errors.As(e, &extent) || extent.Maximum != tc.maximum {
				t.Fatalf("%s length %d: %x %v", tc.name, n, b, e)
			}
		}
	}
}

func TestNotifyApplicationRejectsOversizeMSISDNCarrier(t *testing.T) {
	d, err := LookupClause("TS 44.018", "9.1.58", "NOTIFY APPLICATION DATA")
	if err != nil {
		t.Fatal(err)
	}
	value := messages.NOTIFYAPPLICATIONDATA{MessageType: 13, ApplicationData: runtime.BitString{Bytes: make([]byte, 9), BitLength: 72}, MSISDNLengthGroup: &messages.NOTIFYAPPLICATIONDATAMSISDNLengthGroup{MSISDNLength: 15, MSISDN: runtime.BitString{Bytes: make([]byte, 15), BitLength: 120}}}
	if wire, err := d.CanonicalAtLength(value, 23); wire != nil || err == nil {
		t.Fatalf("oversize value hidden by padding: %x %v", wire, err)
	}
	wire, err := d.CanonicalAtLength(value, 27)
	var extent *runtime.ExtentError
	if wire != nil || !errors.As(err, &extent) || extent.Maximum != 23 {
		t.Fatalf("oversize carrier: %x %v", wire, err)
	}
}
