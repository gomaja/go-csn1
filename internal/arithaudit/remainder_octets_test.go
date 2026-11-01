package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedWholeOctetRemainderSubtractionProof(t *testing.T) {
	const source = `package sample
import runtime "github.com/gomaja/go-csn1/runtime"
func encodeRemainder(w *runtime.Writer, v []uint8) error {
	remaining := w.RemainingLimit()
	if target, present := w.RemainingCanonicalTarget(); present && (remaining < 0 || target < remaining) { remaining = target }
	if remaining < 0 {
		if bits, received := w.ReceivedBits(); received {
			if bits < 0 || bits > 1048576 || w.Position() < 0 || w.Position() > bits { return w.ComponentError(runtime.InvalidValue, "Octets", 0, fmt.Errorf("invalid received whole-octet remainder extent")) }
			remaining = bits - w.Position()
		}
	}
	if remaining < 0 { return fmt.Errorf("%w: whole-octet remainder requires a containing extent", runtime.ErrContextRequired) }
	oldBound, err := w.PushLimit(remaining / 8 * 8)
	if err != nil { return err }
	if len(v) != remaining / 8 { return w.LengthBoundError("whole-octet remainder must fill every complete octet") }
	for _, element := range v { if err := encodeOctet(w, element); err != nil { return err } }
	return w.PopLimit(oldBound)
}`
	for _, tc := range []struct {
		name        string
		old         string
		replacement string
	}{
		{"guarded received remainder", "", ""},
		{"untrusted runtime path", "github.com/gomaja/go-csn1/runtime", "example.test/unsafe/runtime"},
		{"missing runtime import", `import runtime "github.com/gomaja/go-csn1/runtime"`, ""},
		{"wrong runtime alias", "import runtime ", "import other "},
		{"wrong writer type", "*runtime.Writer", "*OtherWriter"},
		{"shadowed writer", "remaining := w.RemainingLimit()", "w := other; remaining := w.RemainingLimit()"},
		{"unproven received extent", "w.ReceivedBits()", "other.ReceivedBits()"},
		{"missing received guard", "; received {", "; true {"},
		{"missing position guard", " || w.Position() > bits", ""},
		{"reversed position guard", "w.Position() > bits", "w.Position() < bits"},
		{"wrong position receiver", "w.Position() > bits", "other.Position() > bits"},
		{"unguarded fallback", "if remaining < 0 {\n\t\tif bits", "if remaining >= 0 {\n\t\tif bits"},
		{"mutated received bits", "remaining = bits -", "bits = -1; remaining = bits -"},
		{"writer advances after guard", "remaining = bits -", "w.WriteUint(0, 8); remaining = bits -"},
		{"shadowed received bits", "remaining = bits -", "{ bits := -1; remaining = bits -" + " w.Position() }; _ = bits //"},
		{"missing canonical extent", "if target, present := w.RemainingCanonicalTarget(); present && (remaining < 0 || target < remaining) { remaining = target }", ""},
		{"missing canonical presence", "present && (remaining < 0 || target < remaining)", "remaining < 0 || target < remaining"},
		{"canonical exceeds local limit", "target < remaining", "target > remaining"},
		{"writes before fallback", "if remaining < 0 {\n\t\tif bits", "remaining = -1; if remaining < 0 {\n\t\tif bits"},
		{"unrelated subtraction", "remaining = bits - w.Position()", "remaining = 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := source
			if tc.old != "" {
				if !strings.Contains(text, tc.old) {
					t.Fatalf("mutation target %q absent", tc.old)
				}
				text = strings.Replace(text, tc.old, tc.replacement, 1)
			}
			if tc.name == "unrelated subtraction" {
				text = strings.Replace(text, "return w.PopLimit(oldBound)\n}", "_ = bits - w.Position(); return w.PopLimit(oldBound)\n}", 1)
			}
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			findings, err := ScanSource("sample/generated.go", []byte(text))
			if err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: root, files: make(map[string]citationFile)}
			found := 0
			for _, finding := range findings {
				if finding.Expression != "bits - w.Position()" {
					continue
				}
				found++
				finding.File = "generated.go"
				err := checker.checkGeneratedResidual(finding)
				if (err == nil) != (tc.old == "") {
					t.Fatalf("subtraction proof: %v", err)
				}
			}
			if found != 1 {
				t.Fatalf("found %d received-bit subtractions, want 1", found)
			}
		})
	}
}

func TestGeneratedWholeOctetRemainderLocatedBody(t *testing.T) {
	const source = `package sample
import "github.com/gomaja/go-csn1/runtime"
func encodeRemainder(w *runtime.Writer,v []uint8)error{
 remaining:=w.RemainingLimit()
 if target,present:=w.RemainingCanonicalTarget();present&&(remaining<0||target<remaining){remaining=target}
 if remaining<0{if bits,received:=w.ReceivedBits();received{if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("invalid received whole-octet remainder extent"))};remaining=bits-w.Position()}}
 if remaining<0{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired))}
 oldBound,err:=w.PushLimit(remaining/8*8);if err!=nil{return err}
 if len(v)!=remaining/8{return w.LengthBoundError("whole-octet remainder must fill every complete octet")}
 for _,element:=range v{if err:=encodeOctet(w,element);err!=nil{return err}}
 return w.PopLimit(oldBound)
}`
	for _, tc := range []struct{ name, old, replacement string }{
		{name: "located missing extent"},
		{"wrong location path", `"Octets",0,fmt.Errorf("%w:`, `"Other",0,fmt.Errorf("%w:`},
		{"wrong location kind", `"Octets",0,fmt.Errorf("%w:`, `"Octets",1,fmt.Errorf("%w:`},
		{"missing canonical minimum", `if target,present:=w.RemainingCanonicalTarget();present&&(remaining<0||target<remaining){remaining=target}`, ``},
		{"missing bound", `oldBound,err:=w.PushLimit(remaining/8*8)`, `oldBound,err:=unbounded()`},
		{"nonrejecting wrapper", `return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired))`, `_ = w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := source
			if tc.old != "" {
				if !strings.Contains(text, tc.old) {
					t.Fatal("mutation missing")
				}
				text = strings.Replace(text, tc.old, tc.replacement, 1)
			}
			c, findings := coordinateFindings(t, text)
			checked := 0
			for _, f := range findings {
				if f.Expression == "bits - w.Position()" || f.Expression == "remaining / 8 * 8" {
					checked++
					err := c.checkGeneratedResidual(f)
					if (err == nil) != (tc.old == "") {
						t.Fatalf("proof: %v", err)
					}
				}
			}
			want := 2
			if tc.name == "missing bound" {
				want = 1
			}
			if checked != want {
				t.Fatalf("checked %d", checked)
			}
		})
	}
}

func TestGeneratedWholeOctetRemainderActual(t *testing.T) {
	source, err := os.ReadFile("../../ts44060/celldata/generated.go")
	if err != nil {
		t.Fatal(err)
	}
	c, findings := coordinateFindings(t, string(source))
	checked := 0
	for _, f := range findings {
		if f.Expression == "bits - w.Position()" || f.Expression == "remaining / 8 * 8" {
			checked++
			if err := c.checkGeneratedResidual(f); err != nil {
				t.Fatal(err)
			}
		}
	}
	if checked != 2 {
		t.Fatalf("actual remainder operations %d", checked)
	}
}
