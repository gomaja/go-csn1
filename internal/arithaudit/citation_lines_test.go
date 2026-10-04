package arithaudit

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// proseCitation is a file:line reference outside the structured guard= and
// post= fields. Each must be followed by a backtick anchor that quotes code
// on the cited line, so a moved line fails here instead of drifting.
var (
	proseCitation  = regexp.MustCompile(`([A-Za-z0-9_][A-Za-z0-9_./-]*\.go):([0-9]+)`)
	citationAnchor = regexp.MustCompile("^\\s+`([^`]+)`")
)

type citedText struct{ origin, text string }

// TestCitationsQuoteTheirLines checks every prose file:line citation in the
// allowlist and in this package's comments. Citations of compiler files
// (csn1/..., cmd/...) are checked against the compiler checkout named by
// GO_CSN1_COMPILER_ROOT; without it only their anchor syntax is checked.
func TestCitationsQuoteTheirLines(t *testing.T) {
	root := filepath.Join("..", "..")
	allowed, err := ReadAllowlist(filepath.Join(root, "arithmetic-allowlist.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	var texts []citedText
	for signature, classification := range allowed {
		texts = append(texts, citedText{signature, classification.Reason})
	}
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, group := range parsed.Comments {
			texts = append(texts, citedText{"internal/arithaudit/" + source, group.Text()})
		}
	}
	compilerRoot := os.Getenv("GO_CSN1_COMPILER_ROOT")
	cache := map[string][]string{}
	var checked, compilerUnchecked int
	for _, cited := range texts {
		for _, match := range proseCitation.FindAllStringSubmatchIndex(cited.text, -1) {
			before := cited.text[:match[0]]
			if strings.HasSuffix(before, "guard=") || strings.HasSuffix(before, "post=") {
				continue
			}
			path := cited.text[match[2]:match[3]]
			line, _ := strconv.Atoi(cited.text[match[4]:match[5]])
			anchor := citationAnchor.FindStringSubmatch(cited.text[match[1]:])
			if anchor == nil {
				t.Errorf("%s: citation %s:%d lacks a backtick anchor", cited.origin, path, line)
				continue
			}
			file := filepath.Join(root, filepath.FromSlash(path))
			if strings.HasPrefix(path, "csn1/") || strings.HasPrefix(path, "cmd/") {
				if compilerRoot == "" {
					compilerUnchecked++
					continue
				}
				file = filepath.Join(compilerRoot, filepath.FromSlash(path))
			}
			lines, ok := cache[file]
			if !ok {
				contents, err := os.ReadFile(file)
				if err != nil {
					t.Errorf("%s: citation %s:%d: %v", cited.origin, path, line, err)
					continue
				}
				lines = strings.Split(string(contents), "\n")
				cache[file] = lines
			}
			normalized := strings.Join(strings.Fields(anchor[1]), " ")
			if line < 1 || line > len(lines) || !strings.Contains(strings.Join(strings.Fields(lines[line-1]), " "), normalized) {
				t.Errorf("%s: %s:%d does not contain %q", cited.origin, path, line, anchor[1])
				continue
			}
			checked++
		}
	}
	t.Logf("checked %d citations; %d compiler citations need GO_CSN1_COMPILER_ROOT", checked, compilerUnchecked)
	if checked == 0 {
		t.Fatal("no citations checked")
	}
}

// TestEveryGuardCitationPartHolds requires each ";"-separated part of a
// structured BOUNDED reason to prove some occurrence of its signature. The
// classification test accepts a finding when any part proves it, which
// alone would let a stale part go unnoticed.
func TestEveryGuardCitationPartHolds(t *testing.T) {
	root := filepath.Join("..", "..")
	findings, err := ScanTree(root, []string{"runtime", "registry", "ts24008", "ts36331", "ts44018", "ts44060"})
	if err != nil {
		t.Fatal(err)
	}
	allowed, err := ReadAllowlist(filepath.Join(root, "arithmetic-allowlist.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	bySignature := map[string][]Finding{}
	for _, finding := range findings {
		bySignature[finding.Signature()] = append(bySignature[finding.Signature()], finding)
	}
	citations := &citationChecker{root: root, files: make(map[string]citationFile)}
	parts := 0
	for signature, classification := range allowed {
		if !strings.HasPrefix(classification.Reason, "BOUNDED: ") || !strings.Contains(classification.Reason, "guard=") {
			continue
		}
		for _, part := range strings.Split(strings.TrimPrefix(classification.Reason, "BOUNDED: "), ";") {
			proved := false
			for _, finding := range bySignature[signature] {
				if citations.checkOne(finding, part) == nil {
					proved = true
					break
				}
			}
			if !proved {
				t.Errorf("%s: citation part %q proves no occurrence", signature, strings.TrimSpace(part))
			}
			parts++
		}
	}
	t.Logf("checked %d guard citation parts", parts)
}
