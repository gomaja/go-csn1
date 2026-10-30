package runtime

import "testing"

func TestTerminalSpareFillsCanonicalTarget(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.11a p15123 prints <spare bit>**.
	// TS 24.007 V20.0.0 §B.1.2.1 p4044 sends each spare bit as zero.
	for _, target := range []int{32, 40, 48, 64, 4096} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: target})
		if err := w.WriteUint(0xfffffff, 28); err != nil {
			t.Fatal(err)
		}
		count := w.SpareFillCountToTarget()
		if count != target-28 {
			t.Fatalf("target %d: spare count %d, want %d", target, count, target-28)
		}
		for i := 0; i < count; i++ {
			if err := w.WriteSpare(); err != nil {
				t.Fatal(err)
			}
		}
		got, err := w.Bytes()
		if err != nil || len(got) != target/8 {
			t.Fatalf("target %d: bytes %x, %v", target, got, err)
		}
		if got[3] != 0xf0 {
			t.Fatalf("first spare octet=%x, want f0", got[3])
		}
		for _, b := range got[4:] {
			if b != 0 {
				t.Fatalf("nonzero transmitted spare: %x", got)
			}
		}
	}
}

func TestTerminalSparePrefersInnerBound(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 64})
	if _, err := w.PushLimit(32); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 28); err != nil {
		t.Fatal(err)
	}
	if got := w.SpareFillCountToTarget(); got != 4 {
		t.Fatalf("inner spare count=%d, want 4", got)
	}
}

func TestTerminalSpareDefaultAndInvalidTargets(t *testing.T) {
	for _, wire := range []WireInfo{{}, {canonical: true}, {canonical: true, targetSet: true, targetBits: 16}, {canonical: true, targetSet: true, targetBits: maxBits + 1}} {
		w := NewWriter()
		if err := w.WriteUint(0, 28); err != nil {
			t.Fatal(err)
		}
		w.WithWire(wire)
		if got := w.SpareFillCountToTarget(); got != 4 {
			t.Fatalf("wire=%+v: spare count=%d, want safe octet fallback", wire, got)
		}
	}
}

func TestSpareFillCountKeepsOctetCompletion(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 64})
	if err := w.WriteUint(0, 28); err != nil {
		t.Fatal(err)
	}
	if got := w.SpareFillCount(); got != 4 {
		t.Fatalf("ordinary spare count=%d, want 4", got)
	}
}
