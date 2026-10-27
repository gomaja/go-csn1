package arithaudit

import (
	"os"
	"strings"
	"testing"
)

func TestManualObservationArithmeticGuards(t *testing.T) {
	const file = "ts36331/uecapability/generated.go"
	original, err := os.ReadFile("../../" + file)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, old, replacement string }{
		{"reviewed helper", "", ""},
		{"origin changed", "origin := start.Position()", "origin := r.Position()"},
		{"signed scalar", "offset, width int, value uint8", "offset, width int, value int"},
		{"offset exceeds value", `observe("RevisionLevel", 1, 2`, `observe("RevisionLevel", 25, 2`},
		{"width exceeds value", `observe("RevisionLevel", 1, 2`, `observe("RevisionLevel", 1, 64`},
		{"offset arithmetic changed", "r.Observe(origin+offset,", "r.Observe(origin+offset+1,"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := string(original)
			if tc.old != "" {
				if !strings.Contains(source, tc.old) {
					t.Fatal("mutation anchor absent")
				}
				source = strings.Replace(source, tc.old, tc.replacement, 1)
			}
			dir := t.TempDir()
			if err := os.MkdirAll(dir+"/ts36331/uecapability", 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dir+"/"+file, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			findings, err := ScanSource(file, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: dir, files: make(map[string]citationFile)}
			checked := 0
			rejected := false
			for _, finding := range findings {
				if finding.Expression != "origin + offset" && finding.Expression != "uint64(value)" {
					continue
				}
				checked++
				if err := checker.checkGeneratedResidual(finding); err != nil {
					rejected = true
				}
			}
			if checked == 0 {
				t.Fatal("no observation arithmetic checked")
			}
			if rejected != (tc.old != "") {
				t.Fatalf("rejected=%t", rejected)
			}
		})
	}
}
