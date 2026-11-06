package arithaudit

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestCapabilityArithmeticCitationsStayInTheirPackage(t *testing.T) {
	body, err := os.ReadFile("../../arithmetic-allowlist.tsv")
	if err != nil {
		t.Fatal(err)
	}
	const file = "ts36331/uecapability/generated.go"
	remaining := map[string]bool{"len(data) * 8": true, "r.Position() + r.Remaining()": true}
	citation := regexp.MustCompile(`([A-Za-z0-9_./-]+\.go):[0-9]+`)
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 3 || !strings.HasPrefix(fields[0], file+":arithmetic:") {
			continue
		}
		expression := strings.TrimPrefix(fields[0], file+":arithmetic:")
		if !remaining[expression] {
			continue
		}
		delete(remaining, expression)
		refs := citation.FindAllStringSubmatch(fields[2], -1)
		if len(refs) == 0 {
			t.Errorf("%s has no source citation", expression)
		}
		for _, ref := range refs {
			if ref[1] != file {
				t.Errorf("%s cites another package: %s", expression, ref[0])
			}
		}
		if expression == "len(data) * 8" && (!strings.Contains(fields[2], "runtime.CheckInput(data)") || !strings.Contains(fields[2], "r.PushLimit(len(data) * 8)")) {
			t.Error("collector citation must identify both its input guard and multiplication")
		}
		if expression == "r.Position() + r.Remaining()" && (!strings.Contains(fields[2], "TS 36.331 V19.4.0 §6.3.6") || !strings.Contains(fields[2], "TS 24.008 V20.1.0 §10.5.5.12a")) {
			t.Error("GERAN PS boundary citation must name its container and value clauses")
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("missing arithmetic rows: %v", remaining)
	}
}
