package messages

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func accessDescriptor(t *testing.T, clause, name string, bits int) runtime.Descriptor {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	if d.Standard != "TS 44.060" || d.Version != "19.0.0" || d.Clause != clause || d.Name != name ||
		d.Extent.MinimumBits != bits || d.Extent.MaximumBits != bits || !d.Extent.Finite || !d.Extent.Fixed || d.Extent.NeedsContext || d.MaxOctets != (bits+7)/8 {
		t.Fatalf("source or extent differs: %+v", d)
	}
	return d
}

func checkAccessBoundaries(t *testing.T, d runtime.Descriptor, bits int) {
	t.Helper()
	wire := []byte{0xa5, 0x1b, 0x69}
	for prefix := range 8 {
		for width := range bits + 1 {
			r := runtime.NewReader(wire)
			if _, err := r.ReadUint(prefix); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(width)
			if err != nil {
				t.Fatal(err)
			}
			_, err = d.DecodeFrom(r)
			if width < bits {
				var located *runtime.OperationError
				if err == nil || !errors.As(err, &located) {
					t.Fatalf("%s prefix %d width %d: wanted located truncation, got %v", d.Name, prefix, width, err)
				}
				if r.Position() > prefix+width {
					t.Fatalf("reader crossed bound: %d > %d", r.Position(), prefix+width)
				}
			} else if err != nil || r.Position() != prefix+bits {
				t.Fatalf("exact bound prefix %d: position %d error %v", prefix, r.Position(), err)
			}
			position := r.Position()
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			if r.Position() != position {
				t.Fatal("restoring bound changed reader position")
			}
			// The first containing bit remains available after a successful decode.
			if width == bits {
				got, err := r.ReadUint(1)
				want := uint64((wire[position/8] >> uint(7-position%8)) & 1)
				if err != nil || got != want {
					t.Fatalf("containing bit %d: %d want %d: %v", position, got, want, err)
				}
			}
		}
	}
	for octets := range (bits + 7) / 8 {
		if _, err := d.Decode(wire[:octets]); err == nil {
			t.Fatalf("accepted %d-octet truncation", octets)
		}
	}
	value, err := d.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	for _, length := range []int{-1, 0, (bits+7)/8 + 1} {
		if _, err := d.CanonicalAtLength(value, length); err == nil {
			t.Fatalf("accepted canonical length %d", length)
		}
	}
	if _, err := d.Encode(struct{}{}); err == nil {
		t.Fatal("accepted unrelated value type")
	}
}

// TS 44.060 V19.0.0 §11.2.33 table 11.2.33.1 prints only an eight-bit
// reference, without a message-type selector. All 256 codewords are payloads.
func TestHandoverAccessEightBit(t *testing.T) {
	d := accessDescriptor(t, "11.2.33", "Handover Access 8 bit message", 8)
	for reference := range 256 {
		wire := []byte{byte(reference)}
		decoded, err := DecodeHandoverAccess8BitMessage(wire)
		if err != nil || decoded.Value.HANDOVERREFERENCEVALUE != uint8(reference) || decoded.BitsConsumed != 8 || decoded.Tail.BitLength != 0 {
			t.Fatalf("reference %d: %+v %v", reference, decoded, err)
		}
		replay, err := EncodeHandoverAccess8BitMessage(decoded.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("reference %d replay %x: %v", reference, replay, err)
		}
		fresh := HandoverAccess8BitMessage{HANDOVERREFERENCEVALUE: uint8(reference)}
		canonical, err := EncodeHandoverAccess8BitMessageCanonical(fresh)
		if err != nil || !bytes.Equal(canonical, wire) {
			t.Fatalf("reference %d canonical %x: %v", reference, canonical, err)
		}
		again, err := d.Canonical(decoded)
		if err != nil || !bytes.Equal(again, canonical) {
			t.Fatalf("reference %d unstable canonical %x: %v", reference, again, err)
		}
		bounded, err := d.CanonicalAtLength(fresh, 1)
		if err != nil || !bytes.Equal(bounded, canonical) {
			t.Fatalf("reference %d canonical extent: %x %v", reference, bounded, err)
		}
		withTail := []byte{byte(reference), 0x6d}
		value, err := d.Decode(withTail)
		if err != nil {
			t.Fatal(err)
		}
		replay, err = d.Encode(value)
		if err != nil || !bytes.Equal(replay, withTail) {
			t.Fatalf("containing tail lost: %x %v", replay, err)
		}
	}
	checkAccessBoundaries(t, d, 8)
}

func FuzzHandoverAccessReplay(f *testing.F) {
	f.Add([]byte{0xa5})
	f.Add([]byte{})
	f.Add([]byte{0xff, 0x5a})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64 {
			return
		}
		value, err := DecodeHandoverAccess8BitMessage(wire)
		if err != nil {
			return
		}
		replay, err := EncodeHandoverAccess8BitMessage(value.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x from %x: %v", replay, wire, err)
		}
	})
}
