package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeAndGeneratedArithmeticIsClassified(t *testing.T) {
	requireStandardLibraryTypes(t)
	root := filepath.Join("..", "..")
	findings, err := ScanTree(root, []string{"runtime", "registry", "ts24008", "ts36331", "ts44018", "ts44060"})
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := ReadAllowlist(filepath.Join(root, "arithmetic-allowlist.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	citations := &citationChecker{root: root, files: make(map[string]citationFile)}
	counted, err := citations.checkStructuralSpareFindings(findings, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckAllowlist(counted, allowed); err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if strings.HasSuffix(finding.File, "/generated.go") {
			if finding.Expression == "uint8(v)" || finding.Expression == "uint16(v)" || finding.Expression == "uint32(v)" || finding.Expression == "uint64(v)" || finding.Expression == "count++" || finding.Expression == "matches++" || finding.Expression == "i++" || finding.Expression == "i + 1" {
				continue // These patterns are checked below against every occurrence.
			}
			if err := citations.checkGeneratedResidual(finding); err != nil {
				t.Fatal(err)
			}
			continue
		}
		classification := allowed[finding.Signature()]
		if strings.HasPrefix(classification.Reason, "BOUNDED: ") {
			if err := citations.check(finding, classification.Reason); err != nil {
				t.Fatalf("%s: %v", finding.Key(), err)
			}
		}
	}
	checked := 0
	widened := 0
	choiceCounters := 0
	loopIncrements := 0
	rangeNext := 0

	generatedFiles, err := filepath.Glob(filepath.Join(root, "ts*", "*", "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, full := range generatedFiles {
		file, err := filepath.Rel(root, full)
		if err != nil {
			t.Fatal(err)
		}
		file = filepath.ToSlash(file)

		contents, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		n, err := CheckGeneratedNarrowing(file, contents)
		if err != nil {
			t.Fatal(err)
		}
		checked += n
		packageSources, err := readGeneratedPackageSources(filepath.Dir(full))
		if err != nil {
			t.Fatal(err)
		}
		n, err = checkGeneratedPackageUint64Conversions(filepath.Base(full), packageSources)
		if err != nil {
			t.Fatal(err)
		}
		widened += n
		n, err = CheckGeneratedChoiceCounters(file, contents)
		if err != nil {
			t.Fatal(err)
		}
		choiceCounters += n
		incs, next, err := CheckGeneratedLoopArithmetic(file, contents)
		if err != nil {
			t.Fatal(err)
		}
		loopIncrements += incs
		rangeNext += next
	}
	want := 0
	wantWidened := 0
	wantChoiceCounters := 0
	wantLoopIncrements := 0
	wantRangeNext := 0
	for _, finding := range findings {
		if strings.HasSuffix(finding.File, "/generated.go") &&
			(finding.Expression == "uint8(v)" || finding.Expression == "uint16(v)" || finding.Expression == "uint32(v)") {
			want++
		}
		if strings.HasSuffix(finding.File, "/generated.go") && finding.Expression == "uint64(v)" {
			wantWidened++
		}
		if strings.HasSuffix(finding.File, "/generated.go") && (finding.Expression == "count++" || finding.Expression == "matches++") {
			wantChoiceCounters++
		}
		if strings.HasSuffix(finding.File, "/generated.go") && finding.Expression == "i++" {
			wantLoopIncrements++
		}
		if strings.HasSuffix(finding.File, "/generated.go") && finding.Expression == "i + 1" {
			file, err := citations.source(finding.File)
			if err != nil {
				t.Fatal(err)
			}
			if fn := functionAt(file, finding.Line); fn != nil && fn.Name.Name != "constrainedWidth" {
				wantRangeNext++
			}
		}
	}
	if checked != want {
		t.Fatalf("checked %d generated narrowings, scanner found %d", checked, want)
	}
	if widened != wantWidened {
		t.Fatalf("checked %d generated uint64 conversions, scanner found %d", widened, wantWidened)
	}
	if choiceCounters != wantChoiceCounters {
		t.Fatalf("checked %d generated choice counters, scanner found %d", choiceCounters, wantChoiceCounters)
	}
	if loopIncrements != wantLoopIncrements || rangeNext != wantRangeNext {
		t.Fatalf("checked %d generated loop increments and %d range successors; scanner found %d and %d", loopIncrements, rangeNext, wantLoopIncrements, wantRangeNext)
	}
	t.Logf("generated arithmetic: checked=%d increments=%d nextIndexes=%d", checked, loopIncrements, rangeNext)
}

// readGeneratedPackageSources supplies the complete production namespace to the
// local-return proof. Include every production Go file regardless of build tags:
// declarations from an alternate configuration must not silently become builtins.
func readGeneratedPackageSources(directory string) (map[string][]byte, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	sources := make(map[string][]byte)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return nil, err
		}
		sources[name] = source
	}
	return sources, nil
}
