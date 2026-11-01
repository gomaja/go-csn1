package arithaudit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestCapabilitySenderArithmeticTemplate(t *testing.T) {
	for _, pkg := range []string{"radioaccesscapability", "assignmentmessages"} {
		t.Run(pkg, func(t *testing.T) {
			original, err := os.ReadFile("../../ts44060/" + pkg + "/generated.go")
			if err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []struct{ old, new string }{{"", ""}, {"bits > 127", "bits > 128"}, {"count > 12", "count > 13"}, {"count*10 + 1", "count*11 + 1"}, {"i < v.Len()", "i <= v.Len()"}} {
				data := original
				if mutation.old != "" {
					if !strings.Contains(string(data), mutation.old) {
						t.Fatalf("mutation anchor absent: %s", mutation.old)
					}
					data = []byte(strings.Replace(string(data), mutation.old, mutation.new, 1))
				}
				set := token.NewFileSet()
				file, err := parser.ParseFile(set, "generated.go", data, 0)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Name.Name != "capabilitySenderValue" {
						continue
					}
					found = true
					err = checkGeneratedCapabilitySender(citationFile{set: set, file: file, source: data}, fn)
					if (err != nil) != (mutation.old != "") {
						t.Fatalf("mutation %q: %v", mutation.old, err)
					}
				}
				if !found {
					t.Fatal("missing sender template")
				}
			}
		})
	}
}
