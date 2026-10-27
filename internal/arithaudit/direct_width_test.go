package arithaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectWidthNarrowingProof(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`width:=8;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{`width:=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{`width:=8;width=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{`width:=8;width=n;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{`width:=8;v,_:=r.ReadUint(width);width=9;return uint8(v)`, true},
		{`width:=3;v,_:=r.ReadUint(width);v+=1;return uint8(v)`, true},
	} {
		_, err := CheckGeneratedNarrowing("sample.go", []byte("package sample;func decode(r reader,n int) uint8{"+tc.body+"}"))
		if (err == nil) != tc.ok {
			t.Fatalf("%s: %v", tc.body, err)
		}
	}
}

func TestDirectCountLoopProof(t *testing.T) {
	for _, tc := range []struct {
		count string
		ok    bool
	}{
		{`count:=2`, true},
		{`count:=1048576`, true},
		{`count:=1048577`, false},
		{`count:=-1`, false},
		{`count:=n`, false},
		{`count,err:=r.ExpressionResult(r.ExpressionVariable("n").Operand());if err!=nil{return}`, true},
		{`count,err:=r.ExpressionResult(r.ExpressionVariable("n").Operand());if err!=nil{return};count=n`, false},
	} {
		_, _, err := CheckGeneratedLoopArithmetic("sample.go", []byte("package sample;func decode(r reader,n int){"+tc.count+";for i:=0;i<count;i++{}}"))
		if (err == nil) != tc.ok {
			t.Fatalf("%s: %v", tc.count, err)
		}
	}
}

func TestDirectLHWidthProofRejectsLongerLiterals(t *testing.T) {
	for _, width := range []string{"1", "10", "11", "16", "1+1", "1*16", "1; width=2", "1; width++"} {
		source := fmt.Appendf(nil, "package sample;func encodeField(w *runtime.Writer){width:=%s;_ = width;_ = uint64(runtime.LHBit('L',w.Position()))}", width)
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "sample"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "sample", "generated.go"), source, 0o600); err != nil {
			t.Fatal(err)
		}
		findings, err := ScanSource("sample/generated.go", source)
		if err != nil {
			t.Fatal(err)
		}
		checker := &citationChecker{root: dir, files: make(map[string]citationFile)}
		checked := 0
		for _, finding := range findings {
			if finding.Expression != "uint64(runtime.LHBit('L', w.Position()))" {
				continue
			}
			checked++
			if err := checker.checkGeneratedResidual(finding); (err == nil) != (width == "1") {
				t.Fatalf("literal %s: %v", width, err)
			}
		}
		if checked != 1 {
			t.Fatalf("literal %s: checked %d expressions: %+v", width, checked, findings)
		}
	}
}
