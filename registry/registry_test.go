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
}
