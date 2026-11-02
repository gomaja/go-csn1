package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedAllocationFieldWidening(t *testing.T) {
	const root = `package codec
import "github.com/gomaja/go-csn1/ts44060/outer"
func decodeRoot()(outer.Root,error){
 v,err:=outer.DecodeRoot();if err!=nil{return v,err}
 if v.Middle!=nil {
  _=uint64((*v.Middle).Bits)
  if (*v.Middle).Extra!=nil {_=uint64((*(*v.Middle).Extra))}
 }
 return v,nil
}

func encodeRoot(v outer.Root)error{
 if v.Middle!=nil {
  _=uint64((*v.Middle).Bits)
  if (*v.Middle).Extra!=nil {_=uint64((*(*v.Middle).Extra))}
 }
 return nil
}`
	const outer = `package outer
import "github.com/gomaja/go-csn1/ts44060/inner"
type Root struct {Middle *inner.Middle}
func DecodeRoot()(Root,error){return Root{},nil}`
	const inner = `package inner
type Middle struct {Bits uint8;Extra *uint8}`
	for _, tc := range []struct {
		name, file, old, replacement string
		rejected                     int
	}{
		{name: "unsigned fields"},
		{name: "byte field", file: "inner", old: "Bits uint8", replacement: "Bits byte"},
		{name: "named unsigned terminal", file: "inner", old: "Bits uint8", replacement: "Bits Bitmap"},
		{name: "sibling unsigned terminal", file: "inner", old: "Bits uint8", replacement: "Bits Bitmap"},
		{name: "signed terminal", rejected: 2, file: "inner", old: "Bits uint8", replacement: "Bits int8"},
		{name: "signed pointer terminal", rejected: 2, file: "inner", old: "Extra *uint8", replacement: "Extra *int8"},
		{name: "missing intermediate selector", rejected: 4, file: "outer", old: "Middle *inner.Middle", replacement: "Other *inner.Middle"},
		{name: "missing terminal selector", rejected: 2, file: "inner", old: "Bits uint8", replacement: "Other uint8"},
		{name: "import shadow", rejected: 2, file: "root", old: "v,err:=outer.DecodeRoot()", replacement: "outer:=factory();v,err:=outer.DecodeRoot()"},
		{name: "wrong dereference", rejected: 2, file: "inner", old: "Extra *uint8", replacement: "Extra uint8"},
		{name: "nonpointer intermediate", rejected: 4, file: "outer", old: "Middle *inner.Middle", replacement: "Middle inner.Middle"},
		{name: "method source", rejected: 1, file: "root", old: "(*v.Middle).Bits)", replacement: "(*v.Middle).Bits())"},
		{name: "arithmetic source", rejected: 1, file: "root", old: "(*v.Middle).Bits)", replacement: "(*v.Middle).Bits-1)"},
		{name: "unsigned builtin shadow", rejected: 4, file: "inner", old: "type Middle", replacement: "type uint8 int8; type Middle"},
		{name: "conversion builtin shadow", rejected: 4, file: "root", old: "func decodeRoot", replacement: "func uint64(v int)int{return v};func decodeRoot"},
		{name: "sibling unsigned builtin shadow", rejected: 4, file: "sibling", replacement: "package inner; type uint8 int8"},
		{name: "sibling conversion builtin shadow", rejected: 4, file: "rootSibling", replacement: "package codec;func uint64(v int)int{return v}"},
		{name: "cyclic type", rejected: 2, file: "inner", old: "Bits uint8", replacement: "Bits Cycle"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sources := map[string]string{"root": root, "outer": outer, "inner": inner}
			if tc.file != "" {
				if tc.old == "" {
					sources[tc.file] = tc.replacement
				} else {
					if !strings.Contains(sources[tc.file], tc.old) {
						t.Fatal("mutation missed")
					}
					sources[tc.file] = strings.Replace(sources[tc.file], tc.old, tc.replacement, 1)
				}
			}
			if tc.name == "cyclic type" {
				sources["inner"] += ";type Cycle Cycle"
			}
			if tc.name == "named unsigned terminal" {
				sources["inner"] += ";type Bitmap uint16"
			}
			if tc.name == "sibling unsigned terminal" {
				sources["sibling"] = "package inner; type Bitmap uint16"
			}
			directory := t.TempDir()
			paths := map[string]string{"root": "ts44060/codec/generated.go", "outer": "ts44060/outer/generated.go", "inner": "ts44060/inner/generated.go", "sibling": "ts44060/inner/other.go", "rootSibling": "ts44060/codec/other.go"}
			for key, source := range sources {
				full := filepath.Join(directory, filepath.FromSlash(paths[key]))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(source), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			findings, err := ScanSource(paths["root"], []byte(sources["root"]))
			if err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: directory, files: make(map[string]citationFile)}
			checked, rejected := 0, 0
			for _, finding := range findings {
				if finding.Kind != "conversion" || !strings.HasPrefix(finding.Expression, "uint64(") {
					continue
				}
				checked++
				if err := checker.checkGeneratedResidual(finding); err != nil {
					rejected++
					if tc.rejected == 0 {
						t.Errorf("unsigned field rejected: %v", err)
					}
				}
			}
			if checked != 4 {
				t.Fatalf("checked %d conversions, want 4", checked)
			}
			if rejected != tc.rejected {
				t.Fatalf("rejected %d conversions, want %d", rejected, tc.rejected)
			}
		})
	}
}

func TestAllocationFieldWideningBindsOccurrence(t *testing.T) {
	const source = `package codec
type Root struct{Bits uint8}
func encode(v Root){{uint64:=func(v uint8)int{return -1};_=uint64(v.Bits)};_=uint64(v.Bits)}`
	const name = "codec/generated.go"
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "codec"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: directory, files: make(map[string]citationFile)}
	findings, err := ScanSource(name, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, finding := range findings {
		if finding.Expression != "uint64(v.Bits)" {
			continue
		}
		err := checker.checkGeneratedResidual(finding)
		if (err != nil) != (checked == 0) {
			t.Fatalf("occurrence %d: %v", checked, err)
		}
		checked++
	}
	if checked != 2 {
		t.Fatalf("checked %d conversion occurrences, want 2", checked)
	}
}
