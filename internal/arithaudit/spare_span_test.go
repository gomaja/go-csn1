package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The sender proof follows diagnostic callees as well as reader operations.
const senderSpareDiagnostics = `
type operationLocation struct{}
var operationLocation0 = operationLocation{}
func (operationLocation) failure(err error, _ string, _, _ int)error{return err}
func locateIndex(err error, _ string, _ int)error{return err}
`

func spareSenderSequence() string {
	sequence := strings.TrimPrefix(senderSequence, "package sample\n")
	sequence = strings.Replace(sequence, "v.Field=x0", "v.Field=x0\n x1,err:=decodeSpare(r)\n if err!=nil{return Value{},err}\n _=x1", 1)
	return groupedSpareProofFixture + "\n" + senderSpareDiagnostics + "\n" + sequence
}

func TestDiscardedSpareSenderSpanProof(t *testing.T) {
	for _, expression := range []string{"r.Position() - senderStart", "r.VirtualBits() - senderVirtual"} {
		for _, tc := range []struct {
			name, old, new string
			valid          bool
		}{
			{"generated spare child", "", "", true},
			{"reversed physical span", "r.Position()-senderStart", "senderStart-r.Position()", false},
			{"reversed virtual span", "r.VirtualBits()-senderVirtual", "senderVirtual-r.VirtualBits()", false},
			{"discarded nonspare child", "x1,err:=decodeSpare(r)", "x1,err:=decodeField(r)", false},
			{"unrelated discard", "_=x1", "_=x0", false},
			{"unchecked spare child", "if err!=nil{return Value{},err}\n _=x1", "_=x1", false},
			{"reader rewind", "_=x1", "_=x1;r.Commit(other)", false},
			{"spare reader rewind", "var v uint64", "r.Commit(other);var v uint64", false},
			{"spare reader alias", "var v uint64", "alias:=r;alias.Commit(other);var v uint64", false},
			{"spare read arguments", "r.ReadSpare()", "r.ReadSpare(1)", false},
			{"padding instead of spare", "r.ReadSpare()", "r.ReadPadding()", false},
			{"dynamic spare bound", "width:=3", "width:=limit", false},
			{"reassigned spare bound", "var v uint64", "width=limit;var v uint64", false},
			{"snapshot reassigned", "_=x1", "_=x1;senderStart=0", false},
			{"virtual snapshot reassigned", "_=x1", "_=x1;senderVirtual=0", false},
		} {
			t.Run(expression+"/"+tc.name, func(t *testing.T) {
				source := spareSenderSequence()
				if tc.old != "" {
					if strings.Count(source, tc.old) != 1 {
						t.Fatal("mutation is not unique")
					}
					source = strings.Replace(source, tc.old, tc.new, 1)
				}
				if err := residualSourceProof(t, source, expression); (err == nil) != tc.valid {
					t.Fatalf("valid=%v error=%v", tc.valid, err)
				}
			})
		}
	}
}

func TestGroupedSpareLoopProof(t *testing.T) {
	for _, tc := range []struct {
		name, old, new string
		valid          bool
	}{
		{"fixed three", "", "", true},
		{"fixed one", "width:=3", "width:=1", true},
		{"fixed sixty four", "width:=3", "width:=64", true},
		{"dynamic width", "width:=3", "width:=limit", false},
		{"too wide", "width:=3", "width:=65", false},
		{"reassigned width", "var v uint64", "width=limit;var v uint64", false},
		{"width address escapes", "var v uint64", "escape(&width);var v uint64", false},
		{"reassigned counter", "bit,err:=r.ReadSpare()", "i=0;bit,err:=r.ReadSpare()", false},
		{"counter address escapes", "bit,err:=r.ReadSpare()", "escape(&i);bit,err:=r.ReadSpare()", false},
		{"inclusive bound", "i<width", "i<=width", false},
		{"unrelated bound", "i<width", "i<limit", false},
		{"shadowed width", "bit,err:=r.ReadSpare()", "width:=limit;_=width;bit,err:=r.ReadSpare()", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := groupedSpareProofFixture
			if tc.old != "" {
				if strings.Count(source, tc.old) != 1 {
					t.Fatal("mutation is not unique")
				}
				source = strings.Replace(source, tc.old, tc.new, 1)
			}
			increments, _, err := CheckGeneratedLoopArithmetic("generated.go", []byte(source))
			if (err == nil) != tc.valid {
				t.Fatalf("increments=%d valid=%v error=%v", increments, tc.valid, err)
			}
			if tc.valid && increments != 1 {
				t.Fatalf("increments=%d", increments)
			}
		})
	}
}

func TestStructuralSpareClassification(t *testing.T) {
	source := spareSenderSequence()
	for _, tc := range []struct {
		name, old, new string
		valid          bool
	}{
		{"proved without rows", "", "", true},
		{"unbounded shift", "width:=3", "width:=limit", false},
		{"reversed span", "r.Position()-senderStart", "senderStart-r.Position()", false},
		{"reversed virtual span", "r.VirtualBits()-senderVirtual", "senderVirtual-r.VirtualBits()", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := source
			if tc.old != "" {
				text = strings.Replace(text, tc.old, tc.new, 1)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "generated.go", text, 0)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "sample"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/gomaja/go-csn1\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "sample/generated.go"), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: root, files: make(map[string]citationFile)}
			_ = set
			_ = file
			var findings []Finding
			for _, expression := range []string{"v <<= 1", "r.Position() - senderStart", "r.VirtualBits() - senderVirtual"} {
				// A stale original finding must not be accepted after an operand reversal.
				line := 0
				for i, s := range strings.Split(source, "\n") {
					if strings.Contains(strings.ReplaceAll(s, " ", ""), strings.ReplaceAll(expression, " ", "")) {
						line = i + 1
						break
					}
				}
				findings = append(findings, Finding{File: "sample/generated.go", Line: line, Expression: expression})
			}
			remaining, err := checker.checkStructuralSpareFindings(findings, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if tc.valid {
				if err := CheckAllowlist(remaining, nil); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	// An unrelated post-loop increment cannot borrow the bounded spare loop.
	text := strings.Replace(groupedSpareProofFixture, "_=fieldStart;_=v", "for i:=0;i<limit;i++{};_=fieldStart;_=v", 1)
	if _, _, err := CheckGeneratedLoopArithmetic("generated.go", []byte(text)); err == nil {
		t.Fatal("unbounded sibling loop borrowed the spare proof")
	}
	// Keep unrelated, unclassified operations visible to the inventory.
	checker := &citationChecker{}
	remaining, err := checker.checkStructuralSpareFindings([]Finding{{File: "sample/generated.go", Expression: "value + 1"}}, nil)
	if err != nil || len(remaining) != 1 {
		t.Fatalf("remaining=%v error=%v", remaining, err)
	}
	if err := CheckAllowlist(remaining, nil); err == nil {
		t.Fatal("unrelated operation disappeared")
	}
}

func TestStructuralIncrementInventoryProof(t *testing.T) {
	for _, tc := range []struct {
		name, old, new string
		valid          bool
	}{
		{"new bounded increments", "", "", true},
		{"unbounded width", "width:=3", "width:=limit", false},
		{"width reassigned", "var v uint64", "width=limit;var v uint64", false},
		{"unbounded sibling", "_=fieldStart;_=v", "for i:=0;i<limit;i++{};_=fieldStart;_=v", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := groupedSpareProofFixture
			if tc.old != "" {
				source = strings.Replace(source, tc.old, tc.new, 1)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "generated.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{files: map[string]citationFile{"sample/generated.go": {set: set, file: file, source: []byte(source)}}}
			var findings []Finding
			ast.Inspect(file, func(n ast.Node) bool {
				if inc, ok := n.(*ast.IncDecStmt); ok {
					pos := set.Position(inc.Pos())
					findings = append(findings, Finding{File: "sample/generated.go", Line: pos.Line, Column: pos.Column, Expression: "i++"})
				}
				return true
			})
			if len(findings) == 0 {
				t.Fatal("missing increment")
			}
			remaining, err := checker.checkStructuralSpareFindings(findings, nil)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if !tc.valid {
				return
			}
			if len(remaining) != 0 {
				t.Fatalf("unproved findings: %v", remaining)
			}
			allowed := map[string]Classification{findings[0].Signature(): {Count: 1}}
			remaining, err = checker.checkStructuralSpareFindings(findings, allowed)
			if err != nil || len(remaining) != 1 {
				t.Fatalf("existing counted row lost: %v %v", remaining, err)
			}
			findings[0].Line = 1
			if _, err = checker.checkStructuralSpareFindings(findings, nil); err == nil {
				t.Fatal("stale finding borrowed increment")
			}
		})
	}
}

func TestFixedSpareWriterIncrementProof(t *testing.T) {
	const fixture = `package sample
 import "github.com/gomaja/go-csn1/runtime"
 func write(w *runtime.Writer,_ runtime.BitString)error{
 for i:=0;i<3;i++{if err:=w.WriteSpare();err!=nil{return locateIndex(w.ComponentError(runtime.InvalidValue,"spare",0,err),"spare",i)}}
 return nil
 }`
	for _, tc := range []struct {
		name, old, new string
		valid          bool
	}{
		{"fixed writer", "", "", true},
		{"fixed one", "i<3", "i<1", true},
		{"fixed sixty four", "i<3", "i<64", true},
		{"unbounded writer", "i<3", "i<limit", false},
		{"too wide", "i<3", "i<65", false},
		{"reset index", "if err:=", "i=0;if err:=", false},
		{"escaping index", "if err:=", "escape(&i);if err:=", false},
		{"padding writer", "w.WriteSpare()", "w.WritePadding()", false},
		{"inclusive bound", "i<3", "i<=3", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fixture
			if tc.old != "" {
				source = strings.Replace(source, tc.old, tc.new, 1)
			}
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, "generated.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{files: map[string]citationFile{"sample/generated.go": {set: set, file: file, source: []byte(source)}}}
			remaining, err := checker.checkStructuralSpareFindings([]Finding{{File: "sample/generated.go", Line: 4, Expression: "i++"}}, nil)
			if (err == nil) != tc.valid || tc.valid && len(remaining) != 0 {
				t.Fatalf("valid=%v remaining=%v error=%v", tc.valid, remaining, err)
			}
		})
	}
}

func TestDiscardedSpareEarlyReceiptProof(t *testing.T) {
	sequence := strings.TrimPrefix(senderEndingSequence, "package sample\n")
	sequence = strings.Replace(sequence, "x0,err:=decodeField(r)", "x1,err:=decodeSpare(r)\n if err!=nil{return Value{},err}\n _=x1\n x0,err:=decodeField(r)", 1)
	base := groupedSpareProofFixture + "\n" + senderSpareDiagnostics + "\n" + sequence
	for _, tc := range []struct {
		name, old, replacement string
		valid                  bool
	}{
		{name: "checked discarded spare before early receipt", valid: true},
		{"unchecked discarded value", "if err!=nil{return Value{},err}\n _=x1", "_=x1", false},
		{"discarded value without rejecting guard", "if err!=nil{return Value{},err}\n _=x1", "if err!=nil{_=err}\n _=x1", false},
		{"spare width argument", "r.ReadSpare()", "r.ReadSpare(width)", false},
		{"different loop width", "i<width", "i<4", false},
		{"receipt after early return", "return v,nil\n }", "return v,nil\n v.Wire.BitsConsumed=r.Position()-senderStart\n }", false},
		{"reversed early receipt", "r.Position()-senderStart", "senderStart-r.Position()", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := base
			if tc.old != "" {
				if !strings.Contains(source, tc.old) {
					t.Fatal("mutation missed")
				}
				source = strings.Replace(source, tc.old, tc.replacement, 1)
			}
			checkSenderReceiptSource(t, source, tc.valid)
		})
	}
}
