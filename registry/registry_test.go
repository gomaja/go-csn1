package registry

import (
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
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

func TestSI7ContextRegistry(t *testing.T) {
	data := make([]byte, 20)
	if _, err := Decode("TS 44.018", "SI7 Rest Octets", data); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatalf("missing context error: %v", err)
	} else {
		var required *runtime.ContextRequiredError
		if !errors.As(err,&required)||required.Clause!="10.5.2.36"||required.Name!="SI7 Rest Octets"{t.Fatalf("missing typed context details: %v",err)}
	}
	if _, err := DecodeWithContext("TS 44.018", "SI7 Rest Octets", data, runtime.SI4ACSOne); err != nil {
		t.Fatal(err)
	}
}
