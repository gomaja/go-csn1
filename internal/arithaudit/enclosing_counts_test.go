package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestEnclosingCountArithmeticGuards(t *testing.T) {
	const body = `package p
func prepareRootCounts(v *Root)error{
 {if len(v.Items)<1{return fmt.Errorf("offset")};count:=uint64(len(v.Items)-1);if count>7{return fmt.Errorf("width")};if countSeen1&&uint64(v.Count)!=count{return fmt.Errorf("conflict")};v.Count=uint8(count);countSeen1=true}
 return nil
}`
	for _, tc := range []struct {
		name, source string
		ok           bool
	}{
		{"guarded", body, true},
		{"zero offset", strings.ReplaceAll(strings.ReplaceAll(body, `if len(v.Items)<1{return fmt.Errorf("offset")};`, ""), "len(v.Items)-1", "len(v.Items)-0"), true},
		{"missing length", strings.Replace(body, `if len(v.Items)<1{return fmt.Errorf("offset")};`, "", 1), false},
		{"missing width", strings.Replace(body, `if count>7{return fmt.Errorf("width")};`, "", 1), false},
		{"weakened width", strings.Replace(body, "count>7", "count>511", 1), false},
		{"wrong length", strings.Replace(body, "len(v.Items)<1", "len(v.Other)<1", 1), false},
		{"extra count write", strings.Replace(body, "countSeen1=true", "countSeen1=true;count++", 1), false},
		{"extra list write", strings.Replace(body, "countSeen1=true", "countSeen1=true;v.Items=nil", 1), false},
		{"wrong target", strings.Replace(body, "v.Count=uint8(count)", "v.OtherCount=uint8(count)", 1), false},
		{"missing sibling", strings.Replace(body, "countSeen1&&uint64(v.Count)!=count", "false&&countSeen1&&uint64(v.Count)!=count", 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := token.NewFileSet()
			tree, err := parser.ParseFile(set, "count.go", tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			fn := tree.Decls[0].(*ast.FuncDecl)
			file := citationFile{set: set, source: []byte(tc.source)}
			for _, expression := range []string{"len(v.Items) - 1", "uint64(len(v.Items) - 1)", "uint8(count)", "uint64(v.Count)"} {
				if strings.Contains(tc.source, "len(v.Items)-0") {
					expression = strings.ReplaceAll(expression, " - 1", " - 0")
				}
				err := checkEnclosingCountArithmetic(file, fn, Finding{Line: 3, Expression: expression})
				if (err == nil) != tc.ok {
					t.Fatalf("%s: %v", expression, err)
				}
			}
		})
	}
}
