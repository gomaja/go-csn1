package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

const groupedSpareProofFixture = `package sample
import "github.com/gomaja/go-csn1/runtime"
func decodeSpare(r *runtime.Reader)(runtime.BitString,error){
 if err:=r.Enter("spare");err!=nil{return runtime.BitString{},err}
 defer r.Leave()
 width:=3
 fieldStart:=0
 if r.Collecting(){fieldStart=r.Position()}
 var v uint64
 for i:=0;i<width;i++{
  bit,err:=r.ReadSpare()
  if err!=nil{return runtime.BitString{},locateIndex(operationLocation0.failure(err,"",r.Position(),width),"spare",i)}
  v <<= 1
  if len(bit.Bytes)>0&&bit.Bytes[0]&0x80!=0{v|=1}
 }
 _=fieldStart;_=v
 return runtime.BitString{},nil
}
`

func TestGeneratedGroupedSpareProof(t *testing.T) {
	errorReturn := `return runtime.BitString{},locateIndex(operationLocation0.failure(err,"",r.Position(),width),"spare",i)`
	tests := []struct {
		name, old, replacement string
		valid                  bool
	}{
		{"width one", "width:=3", "width:=1", true},
		{"width three", "", "", true},
		{"width sixty four", "width:=3", "width:=64", true},
		{"width zero", "width:=3", "width:=0", false},
		{"width sixty five", "width:=3", "width:=65", false},
		{"negative width", "width:=3", "width:=-1", false},
		{"dynamic width", "width:=3", "width:=limit", false},
		{"nonzero accumulator", "var v uint64", "var v uint64=1", false},
		{"wrong accumulator type", "var v uint64", "var v uint32", false},
		{"shadowed accumulator", "v <<= 1", "v:=uint64(0);v <<= 1", false},
		{"shift two", "v <<= 1", "v <<= 2", false},
		{"different accumulator", "v <<= 1", "other <<= 1", false},
		{"counter one", "i:=0", "i:=1", false},
		{"inclusive bound", "i<width", "i<=width", false},
		{"wrong bound", "i<width", "i<65", false},
		{"counter step", "i++", "i+=2", false},
		{"counter reset", "bit,err:=r.ReadSpare()", "i=0;bit,err:=r.ReadSpare()", false},
		{"wrong read", "r.ReadSpare()", "r.ReadPadding()", false},
		{"read substitution", "bit,err:=r.ReadSpare()", "bit,err:=other.ReadSpare()", false},
		{"read arguments", "r.ReadSpare()", "r.ReadSpare(width)", false},
		{"missing error guard", "if err!=nil{" + errorReturn + "}", "", false},
		{"error success", errorReturn, "return runtime.BitString{},nil", false},
		{"error no return", errorReturn, "_ = err", false},
		{"missing len guard", "len(bit.Bytes)>0&&", "", false},
		{"wrong len guard", "len(bit.Bytes)>0", "len(bit.Bytes)>=0", false},
		{"wrong byte", "bit.Bytes[0]&", "bit.Bytes[1]&", false},
		{"wrong mask", "&0x80", "&0x40", false},
		{"wrong bit", "v|=1", "v|=2", false},
		{"unconditional or", "if len(bit.Bytes)>0&&bit.Bytes[0]&0x80!=0{v|=1}", "v|=1", false},
		{"accumulator escape", "bit,err:=r.ReadSpare()", "escape(&v);bit,err:=r.ReadSpare()", false},
		{"width reassignment", "var v uint64", "width=65;var v uint64", false},
		{"reader reassignment", "var v uint64", "r=other;var v uint64", false},
		{"early success", "var v uint64", "return runtime.BitString{},nil;var v uint64", false},
		{"builtin len", "var v uint64", "len:=func([]byte)int{return 1};var v uint64", false},
		{"global len", "func decodeSpare", "var len=func([]byte)int{return 1}\nfunc decodeSpare", false},
		{"builtin uint64", "func decodeSpare", "type uint64=uint32\nfunc decodeSpare", false},
		{"runtime import", "github.com/gomaja/go-csn1/runtime", "sample/runtime", false},
		{"runtime shadow", "var v uint64", "runtime:=other;_=runtime;var v uint64", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := groupedSpareProofFixture
			if tc.old != "" {
				if !strings.Contains(source, tc.old) {
					t.Fatal("mutation missed")
				}
				source = strings.Replace(source, tc.old, tc.replacement, 1)
			}
			set := token.NewFileSet()
			node, err := parser.ParseFile(set, "generated.go", source, 0)
			if err != nil {
				t.Fatal(err)
			}
			line, column := 0, 0
			ast.Inspect(node, func(n ast.Node) bool {
				if a, ok := n.(*ast.AssignStmt); ok && a.Tok == token.SHL_ASSIGN {
					p := set.Position(a.Pos())
					line, column = p.Line, p.Column
				}
				return true
			})
			if line == 0 {
				t.Fatal("missing shift finding")
			}
			checker := &citationChecker{files: map[string]citationFile{"generated.go": {set: set, file: node, source: []byte(source)}}}
			err = checker.checkGeneratedResidual(Finding{File: "generated.go", Line: line, Column: column, Expression: "v <<= 1"})
			if (err == nil) != tc.valid {
				t.Fatalf("error=%v valid=%v", err, tc.valid)
			}
		})
	}
}
