package uecapability

import (
	"bytes"
	"testing"
)

func TestCanonicalGERANCSUsesNestedClassmarkMinimum(t *testing.T) {
	// TS 36.331 V19.4.0 UE-CapabilityRAT-ContainerList carries
	// TS 24.008 V20.1.0 §10.5.1.7 Classmark 3. Its absent trailing
	// fields are receiver-inferred zeros, so the fresh container needs
	// only the one transmitted Classmark 3 octet.
	wire := []byte{0x33, 3, 0, 0, 0, 0}
	d, err := DecodeGERANCS(wire)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := EncodeGERANCSCanonical(d.Value)
	if err != nil || !bytes.Equal(canonical, wire) {
		t.Fatalf("canonical %x, %v; want %x", canonical, err, wire)
	}
}
