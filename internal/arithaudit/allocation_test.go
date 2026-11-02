package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

const allocationBitmapFixture = `package codec
import "github.com/gomaja/go-csn1/runtime"
func decodeAllocationExpressionBody(r *runtime.Reader)(runtime.BitString,error){
 width,err:=r.ExpressionResultWithContext(r.ExpressionVariable("N").Operand());if err!=nil{return runtime.BitString{},err}
 v,err:=r.ReadBitString(width);if err!=nil{return runtime.BitString{},err};return runtime.BitString(v),nil
}
func decodeAllocation(r *runtime.Reader)(runtime.BitString,error){
 v,err:=decodeAllocationExpressionBody(r);if err!=nil{return runtime.BitString{},err}
 if active,err:=r.ExpressionResultWithContext(r.ExpressionVariable("scope"));err==nil&&active==1{
  if v.BitLength>64{return runtime.BitString{},err}
  var bitmap uint64
  for bit:=0;bit<v.BitLength;bit++{
   bitmap=bitmap<<1|uint64((v.Bytes[bit/8]>>uint(7-bit%8))&1)
  }
  if err:=r.RecordAggregate(scope,bitmap);err!=nil{return runtime.BitString{},err}
 }
 return v,nil
}
`

func TestGeneratedAllocationBitmapArithmetic(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"valid", "", ""},
		{"width guard", "v.BitLength>64", "v.BitLength>65"},
		{"shadowed bitmap value", "if v.BitLength>64", "v:=runtime.BitString{BitLength:8}; if v.BitLength>64"},
		{"wrong runtime", "github.com/gomaja/go-csn1/runtime", "sample/runtime"},
		{"guard does not reject", "return runtime.BitString{},err}\n  var bitmap", "_ = err}\n  var bitmap"},
		{"nonzero accumulator", "var bitmap uint64", "var bitmap uint64 = 1"},
		{"initial index", "bit:=0", "bit:=1"},
		{"inclusive bound", "bit<v.BitLength", "bit<=v.BitLength"},
		{"increment", "bit++", "bit+=2"},
		{"accumulator shift", "bitmap<<1", "bitmap<<2"},
		{"source index", "bit/8", "bit/7"},
		{"bit position", "7-bit%8", "8-bit%8"},
		{"mask", "&1)", "&3)"},
		{"index write", "bitmap=bitmap<<1", "bit=0;bitmap=bitmap<<1"},
		{"unvalidated result", "r.ReadBitString(width)", "r.UncheckedBitString(width)"},
		{"changed bytes after validation", "if active,err:=", "v.Bytes=nil; if active,err:="},
		{"changed width after validation", "if active,err:=", "v.BitLength=65; if active,err:="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := allocationBitmapFixture
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
			file := citationFile{set: set, file: node, source: []byte(source)}
			fn := node.Decls[len(node.Decls)-1].(*ast.FuncDecl)
			var loop *ast.ForStmt
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if v, ok := n.(*ast.ForStmt); ok {
					loop = v
				}
				return true
			})
			f := Finding{File: "fixture/generated.go", Line: set.Position(loop.Body.List[len(loop.Body.List)-1].Pos()).Line, Expression: "bitmap << 1"}
			handled, err := checkGeneratedAllocationArithmetic(file, fn, f)
			if tc.name == "valid" {
				if !handled || err != nil {
					t.Fatalf("valid bitmap: handled=%t err=%v", handled, err)
				}
			} else if handled && err == nil {
				t.Fatal("mutated bitmap proof accepted")
			}
		})
	}
}

func TestGeneratedAllocationPopulationConversion(t *testing.T) {
	const fixture = `package codec
import "github.com/gomaja/go-csn1/runtime"
func decodeValue(r *runtime.Reader)(int,error){
 {value,err:=r.ExpressionResultWithContext(r.ExpressionVariable("N"));if err!=nil{return 0,err}
  if value<1{return 0,err}
  r.Set("N",uint64(value))
 }
 return 0,nil
}`
	for _, tc := range []struct{ name, old, replacement string }{
		{"valid", "", ""},
		{"negative guard", "value<1", "value < -1"},
		{"key side effect", `r.Set("N",uint64(value))`, `r.Set(func()string{value=-1;return "N"}(),uint64(value))`},
		{"noninteger value", `value,err:=r.ExpressionResultWithContext(r.ExpressionVariable("N"))`, `value,err:=float64(1e100),error(nil)`},
		{"nonrejecting guard", "return 0,err}\n  r.Set", "_ = err}\n  r.Set"},
		{"changed value", "r.Set(\"N\"", "value=-1; r.Set(\"N\""},
		{"inner value", "r.Set(\"N\",uint64(value))", "func(value int){r.Set(\"N\",uint64(value))}(-1)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fixture
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
			file := citationFile{set: set, file: node, source: []byte(source)}
			fn := node.Decls[len(node.Decls)-1].(*ast.FuncDecl)
			line := 1 + strings.Count(source[:strings.Index(source, "uint64(value)")], "\n")
			handled, err := checkGeneratedAllocationArithmetic(file, fn, Finding{File: "fixture/generated.go", Line: line, Expression: "uint64(value)"})
			if tc.name == "valid" {
				if !handled || err != nil {
					t.Fatalf("valid population: handled=%t err=%v", handled, err)
				}
			} else if handled && err == nil {
				t.Fatal("mutated population proof accepted")
			}
		})
	}
}
