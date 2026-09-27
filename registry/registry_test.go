package registry

import (
	"strings"
	"testing"
)

func TestClauseQualifiedLookup(t *testing.T) {
	_, err := Lookup("TS 24.008", "A5 bits")
	if err == nil || !strings.Contains(err.Error(), "10.5.1.7") || !strings.Contains(err.Error(), "10.5.5.12a") {
		t.Fatalf("ambiguous A5 bits: %v", err)
	}
	first, err := LookupClause("TS 24.008", "10.5.1.7", "A5 bits")
	if err != nil || first.Clause != "10.5.1.7" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := LookupClause("TS 24.008", "10.5.5.12a", "A5 bits")
	if err != nil || second.Clause != "10.5.5.12a" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := Lookup("TS 24.008", "MS RA capability value part"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ standard, clause, name string }{
		{"TS 44.018", "10.5.2.16", "IA Rest Octets"},
		{"TS 44.018", "10.5.2.17", "IAR Rest Octets"},
		{"TS 44.018", "10.5.2.18", "IAX Rest Octets"},
		{"TS 44.060", "12.5.2", "EGPRS Window Size IE"},
		{"TS 44.060", "12.12", "Packet Timing Advance IE"},
	} {
		if _, err := LookupClause(tc.standard, tc.clause, tc.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Lookup("TS 44.018", "PEO IMM Cell Group Details struct"); err == nil || !strings.Contains(err.Error(), "10.5.2.18") {
		t.Fatalf("clause ambiguity was lost: %v", err)
	}
}
