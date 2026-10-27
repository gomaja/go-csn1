package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestCountSetterArithmeticProof(t *testing.T) {
	const body = `func(v *D) SetItems(items []uint8) error {
 if len(items)<1{return fmt.Errorf("offset")}
 count:=uint64(len(items)-1)
 if count>3{return fmt.Errorf("width")}
 v.Count=uint8(count)
 v.Items=items
 return nil
 }`
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"guarded", body, true},
		{"empty representative", strings.ReplaceAll(body, "v.Items=items", "if len(items)==0{items=nil};v.Items=items"), true},
		{"short list reassigned", strings.ReplaceAll(body, "v.Items=items", "if len(items)==1{items=nil};v.Items=items"), false},
		{"missing offset", strings.ReplaceAll(body, `if len(items)<1{return fmt.Errorf("offset")}`, ""), false},
		{"missing width", strings.ReplaceAll(body, `if count>3{return fmt.Errorf("width")}`, ""), false},
		{"bound too wide", strings.ReplaceAll(body, "count>3", "count>256"), false},
		{"guard after narrowing", strings.ReplaceAll(strings.ReplaceAll(body, `if count>3{return fmt.Errorf("width")}`, ""), "v.Count=uint8(count)", `v.Count=uint8(count);if count>3{return fmt.Errorf("width")}`), false},
		{"nil error guard", strings.ReplaceAll(body, `return fmt.Errorf("width")`, "return nil"), false},
		{"count reassigned", strings.ReplaceAll(body, "v.Count=uint8(count)", "count=999;v.Count=uint8(count)"), false},
		{"slice reassigned", strings.ReplaceAll(body, "count:=", "items=nil;count:="), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := parser.ParseFile(token.NewFileSet(), "setter.go", "package sample;"+tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			fn := f.Decls[0].(*ast.FuncDecl)
			for _, expression := range []string{"len(items) - 1", "uint64(len(items) - 1)", "uint8(count)"} {
				if err := checkGeneratedCountSetter(fn, expression); (err == nil) != tc.valid {
					t.Fatalf("%s: %v", expression, err)
				}
			}
		})
	}
}
