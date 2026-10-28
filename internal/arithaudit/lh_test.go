package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedLHUnsignedConversion(t *testing.T) {
	for _, receiver := range []string{"Reader", "Writer"} {
		for _, expression := range []string{"lhL", "lhL ^ 1"} {
			for _, mutation := range []string{"", "shadow", "method", "receiver", "import", "result order"} {
				t.Run(receiver+"/"+expression+"/"+mutation, func(t *testing.T) {
					source := "package sample\nimport \"github.com/gomaja/go-csn1/runtime\"\nfunc encode(r *runtime." + receiver + "){\nlhL,err:=r.LHBit('L');_ = err\n_ = uint64(" + expression + ")\n}\n"
					switch mutation {
					case "shadow":
						source = strings.Replace(source, "_ = uint64", "{lhL:=int(-1);_ = uint64", 1)
						source = strings.Replace(source, ")\n}\n", ")}\n}\n", 1)
					case "method":
						source = strings.Replace(source, "r.LHBit('L')", "r.Position()", 1)
					case "receiver":
						source = strings.Replace(source, "*runtime."+receiver, "*other.Reader", 1)
					case "import":
						source = strings.Replace(source, "github.com/gomaja/go-csn1/runtime", "example.com/runtime", 1)
					case "result order":
						source = strings.Replace(source, "lhL,err:=", "err,lhL:=", 1)
					}
					root := t.TempDir()
					if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
					checker := &citationChecker{root: root, files: map[string]citationFile{}}
					err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: 5, Kind: "conversion", Expression: "uint64(" + expression + ")"})
					if (err == nil) != (mutation == "") {
						t.Fatalf("conversion proof: %v", err)
					}
				})
			}
		}
	}
}

func TestLHConversionRejectsMixedScopeShadows(t *testing.T) {
	line := "_ = uint64(lhL); {lhL:=int(-1); _ = uint64(lhL)}"
	source := "package sample\nimport \"github.com/gomaja/go-csn1/runtime\"\nfunc encode(r *runtime.Reader){\nlhL,err:=r.LHBit('L');_ = err\n" + line + "\n}\n"
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: map[string]citationFile{}}
	for _, column := range []int{strings.Index(line, "uint64") + 1, strings.LastIndex(line, "uint64") + 1} {
		err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: 5, Column: column, Kind: "conversion", Expression: "uint64(lhL)"})
		if err == nil {
			t.Fatalf("column %d: %v", column, err)
		}
	}
}

func TestGeneratedLHMaskCannotBeReassigned(t *testing.T) {
	for _, write := range []string{"", "lhL=255;", "lhL++;", "if true{lhL=128;};", "func(){lhL=128}();", "_ = &lhL;"} {
		for _, width := range []string{"1", "2"} {
			source := []byte("package sample\nimport \"github.com/gomaja/go-csn1/runtime\"\nfunc decode(r *runtime.Reader)(uint8,error){lhL,err:=r.LHBit('L');if err!=nil{return 0,err};width:=" + width + ";v,err:=r.ReadUint(width);if err!=nil{return 0,err};" + write + "v^=uint64(lhL);return uint8(v),nil}")
			_, err := CheckGeneratedNarrowing("generated.go", source)
			if (err == nil) != (write == "" && width == "1") {
				t.Fatalf("width=%s write=%q: %v", width, write, err)
			}
		}
	}
}
