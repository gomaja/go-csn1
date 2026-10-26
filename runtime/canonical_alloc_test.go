package runtime

import "testing"

// TestCanonicalCopyWireInfoAllocatesNothing pins the canonical copy of a value's
// wire record to a shared record: a WireInfo field adds no allocation to a
// canonical copy, whether failures are located or raw.
func TestCanonicalCopyWireInfoAllocatesNothing(t *testing.T) {
	type withWire struct {
		A, B uint8
		Wire WireInfo
	}
	type withoutWire struct {
		A, B uint8
	}
	for _, raw := range []bool{false, true} {
		with := withWire{A: 1, B: 2}
		without := withoutWire{A: 1, B: 2}
		got := testing.AllocsPerRun(100, func() { _ = canonicalCopy(with, raw) })
		want := testing.AllocsPerRun(100, func() { _ = canonicalCopy(without, raw) })
		if got != want {
			t.Fatalf("raw errors %v: canonical copy with a wire record allocates %v times, without %v", raw, got, want)
		}
		if copied := canonicalCopy(with, raw); !copied.Wire.canonical || copied.Wire.rawErrors != raw {
			t.Fatalf("raw errors %v: copied wire record %+v", raw, copied.Wire)
		}
	}
}
