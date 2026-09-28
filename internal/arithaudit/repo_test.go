package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeAndGeneratedArithmeticIsClassified(t *testing.T) {
	root := filepath.Join("..", "..")
	findings, err := ScanTree(root, []string{"runtime", "registry", "ts24008", "ts36331", "ts44018", "ts44060"})
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := ReadAllowlist(filepath.Join(root, "arithmetic-allowlist.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckAllowlist(findings, allowed); err != nil {
		t.Fatal(err)
	}
	checked := 0
	widened := 0
	for _, file := range []string{
		"ts24008/classmark/generated.go", "ts24008/msrac/generated.go",
		"ts36331/uecapability/generated.go", "ts44018/measurement/generated.go",
		"ts44018/restoctets/generated.go", "ts44060/ies/generated.go",
	} {
		contents, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		n, err := CheckGeneratedNarrowing(file, contents)
		if err != nil {
			t.Fatal(err)
		}
		checked += n
		n, err = CheckGeneratedUint64Conversions(file, contents)
		if err != nil {
			t.Fatal(err)
		}
		widened += n
	}
	want := 0
	wantWidened := 0
	for _, finding := range findings {
		if strings.HasSuffix(finding.File, "/generated.go") &&
			(finding.Expression == "uint8(v)" || finding.Expression == "uint16(v)" || finding.Expression == "uint32(v)") {
			want++
		}
		if strings.HasSuffix(finding.File, "/generated.go") && finding.Expression == "uint64(v)" {
			wantWidened++
		}
	}
	if checked != want {
		t.Fatalf("checked %d generated narrowings, scanner found %d", checked, want)
	}
	if widened != wantWidened {
		t.Fatalf("checked %d generated uint64 conversions, scanner found %d", widened, wantWidened)
	}
}
