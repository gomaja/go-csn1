package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const repeatCapacityFixture = `package sample
func decodeItems(r *runtime.Reader)([]uint8,error){
 var out []uint8
 count,err:=r.Eval("val(N)");if err!=nil{return nil,err}
 for i:=0;i<count;i++{
  if err:=r.EnterIndex("Items",i);err!=nil{return nil,err}
  v,err:=decodeItem(r)
  r.LeaveIndex()
  if err!=nil{return nil,err}
  if i==0{
   capacity:=128
   if !r.ZeroExtension(){capacity=max(capacity,r.Remaining()/MINIMUM+1)}
   out=make([]uint8,0,min(count,capacity))
  }
  out=append(out,v)
 }
 return out,nil
}`

func TestGeneratedTokenRepeatCapacityProof(t *testing.T) {
	source := strings.ReplaceAll(repeatCapacityFixture, `r.Eval("val(N)")`, `r.EvalTokens([]string{"val","(","N",")"})`)
	checkRepeatCapacityFixture(t, strings.ReplaceAll(source, "MINIMUM", "17"), true)
}

func TestGeneratedRepeatCapacityProof(t *testing.T) {
	for _, minimum := range []string{"1", "17"} {
		t.Run(minimum, func(t *testing.T) {
			checkRepeatCapacityFixture(t, strings.ReplaceAll(repeatCapacityFixture, "MINIMUM", minimum), true)
		})
	}
}

func TestGeneratedRepeatCapacityRejectsMutations(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"zero divisor", "/MINIMUM", "/0"},
		{"negative divisor", "/MINIMUM", "/-1"},
		{"unclamped count", "min(count,capacity)", "capacity"},
		{"missing zero extension guard", "!r.ZeroExtension()", "true"},
		{"extra capacity assignment", "capacity:=128", "capacity:=128;capacity=1<<30"},
		{"capacity after loop", "return out,nil", "capacity:=1<<30;_ = capacity;return out,nil"},
		{"count after loop", "return out,nil", "count=1<<30;return out,nil"},
		{"count inside loop", "out=append(out,v)", "count=1<<30;out=append(out,v)"},
		{"count address escape", "return out,nil", "_ = &count;return out,nil"},
		{"unbounded count source", `r.Eval("val(N)")`, "unbounded()"},
		{"wrong first index", "i==0", "i==1"},
		{"wrong slice length", "make([]uint8,0,", "make([]uint8,1,"},
		{"missing successful item guard", "if err!=nil{return nil,err}\n  if i==0", "if i==0"},
		{"removed leave", "r.LeaveIndex()", "r.Check()"},
		{"index mutation", "v,err:=decodeItem(r)", "i++;v,err:=decodeItem(r)"},
		{"builtin shadow", "var out []uint8", "max:=unsafeMaximum;_ = max;var out []uint8"},
		{"unrelated operation", "return out,nil", "_ = r.Remaining()/17+1;return out,nil"},
		{"removed loop increment", "i++", ""},
		{"conditional count shadow", `count,err:=r.Eval("val(N)");if err!=nil{return nil,err}`, `if false{count,err:=r.Eval("val(N)");_,_=count,err}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(repeatCapacityFixture, tc.old, tc.replacement, 1)
			if source == repeatCapacityFixture {
				t.Fatal("mutation did not apply")
			}
			checkRepeatCapacityFixture(t, strings.ReplaceAll(source, "MINIMUM", "17"), false)
		})
	}
}

func checkRepeatCapacityFixture(t *testing.T, source string, accepted bool) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	findings, err := ScanSource("generated.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	checked, rejected := 0, 0
	for _, finding := range findings {
		if !strings.HasPrefix(generatedWhitespace.ReplaceAllString(finding.Expression, ""), "r.Remaining()/") {
			continue
		}
		for _, expression := range []string{finding.Expression, strings.Split(finding.Expression, " + ")[0]} {
			finding.Expression = expression
			checked++
			if err := checker.checkGeneratedResidual(finding); err != nil {
				rejected++
				if accepted {
					t.Fatalf("positive proof rejected %s: %v", finding.Expression, err)
				}
			}
		}
	}
	if checked < 2 || !accepted && rejected == 0 {
		t.Fatalf("checked=%d rejected=%d accepted=%t", checked, rejected, accepted)
	}
}
