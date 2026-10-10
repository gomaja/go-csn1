package msrac

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestReplaceA5BitsInDecodedValue(t *testing.T) {
	// Synthetic two-entry vector; pycrate 0.7.11 independently decodes it.
	// TS 24.008 V20.1.0 §10.5.5.12a sends A5/2 as zero, while accepting
	// either received value independently for each access-technology entry.
	wire := []byte{0x12, 0x07, 0x80, 0x13, 0x20, 0x70, 0}
	for _, tc := range []struct {
		name string
		edit func(*A5Bits) *A5Bits
		want byte
	}{
		{"unchanged", func(a *A5Bits) *A5Bits { return a }, 0x80},
		{"fresh", func(*A5Bits) *A5Bits { return &A5Bits{A51: 1} }, 0},
		{"in place", func(a *A5Bits) *A5Bits { a.A53 = 1; return a }, 0xc0},
		{"received zero", func(*A5Bits) *A5Bits {
			d, err := DecodeA5Bits([]byte{0x80})
			if err != nil {
				t.Fatal(err)
			}
			return &d.Value
		}, 0},
		{"received one", func(*A5Bits) *A5Bits {
			d, err := DecodeA5Bits([]byte{0xc0})
			if err != nil {
				t.Fatal(err)
			}
			return &d.Value
		}, 0x80},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := DecodeMSRACapabilityValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			content := &d.Value.MSRACapabilityValuePartStruct.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities
			content.A5Bits = tc.edit(content.A5Bits)
			want := bytes.Clone(wire)
			want[2] = tc.want
			out, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(out, want) {
				t.Fatalf("plain %x, want %x: %v", out, want, err)
			}
			canonicalWant := bytes.Clone(want)
			canonicalWant[2] &= 0x7f
			canonical, err := EncodeMSRACapabilityValuePartCanonical(d.Value)
			if err != nil || !bytes.Equal(canonical, canonicalWant) {
				t.Fatalf("canonical %x, want %x: %v", canonical, canonicalWant, err)
			}
			out, err = EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(out, want) {
				t.Fatalf("canonical changed source: %x: %v", out, err)
			}
			other := d.Value.MSRACapabilityValuePartStruct.Entries[1].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits
			if other == nil || other.ReceivedA52() {
				t.Fatal("replacement changed another entry's layout")
			}
		})
	}
	fresh, err := EncodeA5Bits(A5Bits{A51: 1})
	if err != nil || !bytes.Equal(fresh, []byte{0x80}) {
		t.Fatalf("fresh local encoding %x: %v", fresh, err)
	}
}

func TestInferredA52RecordSurvivesFreshReplacement(t *testing.T) {
	// Standards-derived synthetic vector: a four-bit Content retains RF power
	// and the A5-presence flag; missing A5 bits are receiver-inferred zeros
	// (TS 24.008 V20.1.0 §10.5.5.12a, Access capabilities Length and Content).
	wire := []byte{0x10, 0x82}
	for _, fresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "received", true: "fresh"}[fresh], func(t *testing.T) {
			d, err := DecodeMSRACapabilityValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			content := &d.Value.MSRACapabilityValuePartStruct.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities
			if content.A5Bits == nil || len(content.A5Bits.Wire.Spare) != 1 || content.A5Bits.Wire.Spare[0].BitLength != 0 {
				t.Fatal("missing inferred local record")
			}
			if fresh {
				content.A5Bits = &A5Bits{}
			}
			out, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(out, wire) {
				t.Fatalf("plain %x, want %x: %v", out, wire, err)
			}
			content.A5Bits.A51 = 1
			if _, err := EncodeMSRACapabilityValuePart(d.Value); err == nil {
				t.Fatal("nonzero edit to an inferred bit accepted")
			}
		})
	}
}

func TestInferredA52TransplantedIntoFullEntry(t *testing.T) {
	// Synthetic short Content has no transmitted A5/2 bit. When its A5Bits
	// move to a full Content, sender layout defaults apply: A5/2 is zero
	// (TS 24.008 V20.1.0 §10.5.5.12a, table 10.5.146).
	short, err := DecodeMSRACapabilityValuePart([]byte{0x10, 0x82})
	if err != nil {
		t.Fatal(err)
	}
	inferred := *short.Value.MSRACapabilityValuePartStruct.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits
	if len(inferred.Wire.Spare) != 1 || inferred.Wire.Spare[0].BitLength != 0 || inferred.ReceivedA52() {
		t.Fatal("missing inferred A5/2 record")
	}
	alone, err := EncodeA5Bits(inferred)
	if err != nil || !bytes.Equal(alone, []byte{0}) {
		t.Fatalf("standalone %x: %v", alone, err)
	}
	for _, entry := range []int{0, 1} {
		t.Run(fmt.Sprintf("entry %d", entry), func(t *testing.T) {
			wire := []byte{0x12, 0x07, 0x80, 0x13, 0x20, 0x78, 0}
			d, err := DecodeMSRACapabilityValuePart(wire)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(unchanged, wire) {
				t.Fatalf("unchanged %x: %v", unchanged, err)
			}
			content := &d.Value.MSRACapabilityValuePartStruct.Entries[entry].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities
			content.A5Bits = &inferred
			want := bytes.Clone(wire)
			if entry == 0 {
				want[1], want[2] = 0x06, 0
			} else {
				want[5] &^= 0x18
			}
			plain, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(plain, want) {
				t.Fatalf("plain %x, want %x: %v", plain, want, err)
			}
			content.A5Bits = &A5Bits{}
			fresh, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(plain, fresh) {
				t.Fatalf("fresh %x differs from inferred %x: %v", fresh, plain, err)
			}
			canonical, err := EncodeMSRACapabilityValuePartCanonical(d.Value)
			content.A5Bits = &inferred
			fromInferred, inferredErr := EncodeMSRACapabilityValuePartCanonical(d.Value)
			if err != nil || inferredErr != nil || !bytes.Equal(canonical, fromInferred) {
				t.Fatalf("canonical %x / %x: %v / %v", canonical, fromInferred, err, inferredErr)
			}
		})
	}
	// Moving a received one into an inferred position must still fail closed.
	one, err := DecodeA5Bits([]byte{0x40})
	if err != nil {
		t.Fatal(err)
	}
	short.Value.MSRACapabilityValuePartStruct.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits = &one.Value
	if _, err := EncodeMSRACapabilityValuePart(short.Value); err == nil || !strings.Contains(err.Error(), "edit to receiver-inferred bit") {
		t.Fatalf("transmitted one in inferred position: %v", err)
	}
}

func FuzzInferredA52Transplant(f *testing.F) {
	for _, wire := range [][]byte{{0x12, 0x07, 0x80, 0x13, 0x20, 0x78, 0}, {0x10, 0x82}, {0x12, 0x07, 0, 0x13, 0x20, 0x70, 0}} {
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 256 {
			t.Skip()
		}
		d, err := DecodeMSRACapabilityValuePart(wire)
		if err != nil {
			return
		}
		short, err := DecodeMSRACapabilityValuePart([]byte{0x10, 0x82})
		if err != nil {
			t.Fatal(err)
		}
		inferred := *short.Value.MSRACapabilityValuePartStruct.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities.A5Bits
		for i, entry := range d.Value.MSRACapabilityValuePartStruct.Entries {
			if entry.AccessTechnologyTypeChoice.AccessCapabilities == nil {
				continue
			}
			content := &d.Value.MSRACapabilityValuePartStruct.Entries[i].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities
			if content.A5Bits == nil {
				continue
			}
			original := content.A5Bits
			content.A5Bits = &inferred
			plain, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil {
				t.Fatalf("inferred replacement entry %d: %v", i, err)
			}
			content.A5Bits = &A5Bits{}
			fresh, err := EncodeMSRACapabilityValuePart(d.Value)
			if err != nil || !bytes.Equal(plain, fresh) {
				t.Fatalf("inferred %x / fresh %x: %v", plain, fresh, err)
			}
			replay, err := DecodeMSRACapabilityValuePart(plain)
			if err != nil {
				t.Fatalf("replacement decode: %v", err)
			}
			out, err := EncodeMSRACapabilityValuePart(replay.Value)
			if err != nil || !bytes.Equal(plain, out) {
				t.Fatalf("replacement replay %x / %x: %v", plain, out, err)
			}
			content.A5Bits = original
		}
	})
}
