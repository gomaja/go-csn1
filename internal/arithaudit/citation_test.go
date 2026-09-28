package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoundCitationRequiresLocalGuardOrNamedCallee(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := `package runtime
func bounded(n int) int {
	if n > 8 { return 0 }
	return n + 1
}
func unrelated(n int) int {
	if n > 8 { return 0 }
	return n + 1
}
func caller(n int) int { return bounded(n) + 1 }
func late(n int) int { x := n + 1; bounded(n); return x }
func modulo(n int) int { return 7 - (n % 8) }
type state struct { pos, virtual int }
func mismatched(s state) int { if s.virtual > 8 { return 0 }; return s.pos + 1 }
`
	if err := os.WriteFile(filepath.Join(root, "runtime", "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := ScanSource("runtime/sample.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var target Finding
	for _, finding := range findings {
		if finding.Line == 4 && finding.Expression == "n + 1" {
			target = finding
			break
		}
	}
	if target.Line == 0 {
		t.Fatalf("missing test finding: %+v", findings)
	}
	for _, tc := range []struct {
		name, reason string
		valid        bool
	}{
		{"local guard", "BOUNDED: operand=n guard=runtime/sample.go:3", true},
		{"wrong function", "BOUNDED: operand=n guard=runtime/sample.go:7", false},
		{"operation itself", "BOUNDED: operand=n guard=runtime/sample.go:4", false},
		{"wrong operand", "BOUNDED: operand=x guard=runtime/sample.go:3", false},
		{"missing citation", "BOUNDED: operand=n checked", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckBoundCitation(root, target, tc.reason)
			if (err == nil) != tc.valid {
				t.Fatalf("reason %q: err=%v, valid=%t", tc.reason, err, tc.valid)
			}
		})
	}
	caller := Finding{File: "runtime/sample.go", Line: 10, Kind: "arithmetic", Expression: "bounded(n) + 1"}
	if err := CheckBoundCitation(root, caller, "BOUNDED: operand=n callee=bounded guard=runtime/sample.go:3"); err != nil {
		t.Fatal(err)
	}
	if err := CheckBoundCitation(root, caller, "BOUNDED: operand=n callee=unrelated guard=runtime/sample.go:7"); err == nil || !strings.Contains(err.Error(), "callee") {
		t.Fatalf("accepted unrelated callee: %v", err)
	}
	if err := CheckBoundCitation(root, caller, "BOUNDED: operand=n callee=bounded guard=runtime/sample.go:3 post=runtime/sample.go:7"); err == nil {
		t.Fatal("accepted postcondition in a different function")
	}
	if err := CheckBoundCitation(root, target, "BOUNDED: operand=x guard=runtime/sample.go:7; operand=n guard=runtime/sample.go:3"); err != nil {
		t.Fatalf("rejected matching guard among grouped citations: %v", err)
	}
	late := Finding{File: "runtime/sample.go", Line: 11, Column: 29, Kind: "arithmetic", Expression: "n + 1"}
	if err := CheckBoundCitation(root, late, "BOUNDED: operand=n callee=bounded guard=runtime/sample.go:3"); err == nil {
		t.Fatal("accepted callee invoked after the operation")
	}
	modulo := Finding{File: "runtime/sample.go", Line: 12, Kind: "arithmetic", Expression: "7 - (n % 8)"}
	if err := CheckBoundCitation(root, modulo, "BOUNDED: operand=n intrinsic=mod8 guard=runtime/sample.go:12"); err != nil {
		t.Fatal(err)
	}
	if err := CheckBoundCitation(root, target, "BOUNDED: operand=n intrinsic=mod8 guard=runtime/sample.go:4"); err == nil {
		t.Fatal("accepted modulo proof without a modulo")
	}
	wrongField := Finding{File: "runtime/sample.go", Line: 14, Kind: "arithmetic", Expression: "s.pos + 1"}
	if err := CheckBoundCitation(root, wrongField, "BOUNDED: operand=s.pos guard=runtime/sample.go:14"); err == nil {
		t.Fatal("accepted comparison of a different selector")
	}
}

func TestIntrinsicBoundCitationsMatchTheirOperation(t *testing.T) {
	for _, tc := range []struct {
		name, source, expression, proof string
	}{
		{"len minus one", "func f(data []byte) int { return len(data)-1 }", "len(data) - 1", "operand=data intrinsic=lenMinusOne"},
		{"constant", "const maxBits=1<<20; func f() int { return maxBits+8 }", "maxBits + 8", "operand=maxBits intrinsic=constant"},
		{"mask", "func f(n uint64) uint8 { return uint8(n&1) }", "uint8(n & 1)", "operand=n intrinsic=maskOne"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "runtime"), 0o755); err != nil {
				t.Fatal(err)
			}
			source := []byte("package runtime\n" + tc.source + "\n")
			if err := os.WriteFile(filepath.Join(root, "runtime", "sample.go"), source, 0o644); err != nil {
				t.Fatal(err)
			}
			finding := Finding{File: "runtime/sample.go", Line: 2, Kind: "arithmetic", Expression: tc.expression}
			reason := "BOUNDED: " + tc.proof + " guard=runtime/sample.go:2"
			if err := CheckBoundCitation(root, finding, reason); err != nil {
				t.Fatal(err)
			}
			if err := CheckBoundCitation(root, finding, strings.Replace(reason, "intrinsic=", "intrinsic=wrong", 1)); err == nil {
				t.Fatal("accepted unknown intrinsic proof")
			}
		})
	}
}

func TestGeneratedResidualRequiresItsLocalGuard(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ts44018"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "ts44018", "generated.go")
	good := `package ts44018
func decode(r Reader) error {
	width, err := r.Eval("1")
	if err != nil { return err }
	v, err := r.ReadUint(width)
	if err != nil { return err }
	_ = v
	_ = r.Position() - 1
	return nil
}`
	finding := Finding{File: "ts44018/generated.go", Line: 8, Kind: "arithmetic", Expression: "r.Position() - 1"}
	for _, tc := range []struct {
		source string
		valid  bool
	}{
		{good, true},
		{strings.Replace(good, "if err != nil { return err }\n\t_ = v", "_ = err\n\t_ = v", 1), false},
	} {
		if err := os.WriteFile(path, []byte(tc.source), 0o644); err != nil {
			t.Fatal(err)
		}
		checker := &citationChecker{root: root, files: make(map[string]citationFile)}
		if err := checker.checkGeneratedResidual(finding); (err == nil) != tc.valid {
			t.Fatalf("guard check err=%v, valid=%t", err, tc.valid)
		}
	}
}
