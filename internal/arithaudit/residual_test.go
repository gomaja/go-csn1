package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassmarkWriterByteWideningProof(t *testing.T) {
	for _, parameter := range []string{"uint8", "int", "int64"} {
		t.Run(parameter, func(t *testing.T) {
			root := t.TempDir()
			source := "package sample\nfunc encode(){\nwrite := func(value " + parameter + ", width int) { if writeErr == nil { writeErr = w.WriteUint(uint64(value), width) } }\n_ = write\n}\n"
			if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: root, files: make(map[string]citationFile)}
			err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: 3, Kind: "conversion", Expression: "uint64(value)"})
			if (err == nil) != (parameter == "uint8") {
				t.Fatalf("%s widening proof: %v", parameter, err)
			}
			if parameter == "uint8" {
				changed := strings.Replace(source, "if writeErr == nil {", "value := int(-1); if writeErr == nil {", 1)
				if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(changed), 0o600); err != nil {
					t.Fatal(err)
				}
				checker.files = make(map[string]citationFile)
				if err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: 3, Expression: "uint64(value)"}); err == nil {
					t.Fatal("accepted changed helper body")
				}
			}
		})
	}
}

func TestClassmarkReaderByteNarrowingProof(t *testing.T) {
	for _, limit := range []string{"8", "9", "64"} {
		root := t.TempDir()
		source := "package sample\nfunc Decode(){\nread:=func(width int)uint8{if width<0||width>" + limit + "{readErr=fmt.Errorf(\"invalid Classmark 2 read width\");return 0};value,err:=r.ReadUint(width);if err!=nil{readErr=err};return uint8(value)}\n_ = read\n}\n"
		if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		checker := &citationChecker{root: root, files: make(map[string]citationFile)}
		err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: 3, Expression: "uint8(value)"})
		if (err == nil) != (limit == "8") {
			t.Fatalf("read width %s proof: %v", limit, err)
		}
	}
}
