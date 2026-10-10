package uecapability

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalClassmark2CapabilityReservations(t *testing.T) {
	// TS 24.008 V20.1.0 table 10.5.6a: revision 0 and 3 are reserved.
	// Across the table's band/RAT contexts, RF codes 0..4 and 7 are defined;
	// 5 and 6 have no conforming context. Reception stays accepted (§8.1).
	for revision := uint8(0); revision < 4; revision++ {
		for power := uint8(0); power < 8; power++ {
			wire := []byte{revision<<5 | power, 0, 1}
			d, err := DecodeClassmark2ValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			if !d.Value.ReceivedA52() {
				t.Fatal("received A5/2 unavailable")
			}
			plain, err := EncodeClassmark2ValuePart(d.Value)
			if err != nil || !bytes.Equal(plain, wire) {
				t.Fatalf("plain %x: %v", plain, err)
			}
			out, err := EncodeClassmark2ValuePartCanonical(d.Value)
			valid := (revision == 1 || revision == 2) && power != 5 && power != 6
			if valid {
				want := []byte{revision<<5 | power, 0, 0}
				if err != nil || !bytes.Equal(out, want) {
					t.Fatalf("canonical %x, want %x: %v", out, want, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
				t.Fatalf("canonical revision %d power %d: %v", revision, power, err)
			}
		}
	}
	if (Classmark2ValuePart{}).ReceivedA52() {
		t.Fatal("fresh A5/2 set")
	}
	out, err := EncodeClassmark2ValuePartCanonical(Classmark2ValuePart{RevisionLevel: 1})
	if err != nil || !bytes.Equal(out, []byte{0x20, 0, 0}) {
		t.Fatalf("fresh canonical %x: %v", out, err)
	}
}
