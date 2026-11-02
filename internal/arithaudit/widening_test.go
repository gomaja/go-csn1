package arithaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedUint64ConversionsFromLocalUnsignedReturn(t *testing.T) {
	const wrapper = `func decode(r *Reader) (uint64, error) {
	v, err := decodeValue(r)
	if err != nil { return 0, err }
	return uint64(v), nil
}`
	for _, typ := range []string{"uint8", "uint16", "uint32", "uint64", "uint", "byte"} {
		t.Run(typ, func(t *testing.T) {
			source := fmt.Sprintf("package sample\n%s\nfunc decodeValue(r *Reader) (%s, error) { return 0, nil }", wrapper, typ)
			count, err := checkGeneratedPackageUint64Conversions("generated.go", map[string][]byte{"generated.go": []byte(source)})
			if err != nil || count != 1 {
				t.Fatalf("unsigned local return: count=%d err=%v", count, err)
			}
		})
	}
}

func TestGeneratedUint64ConversionsRejectUnprovenLocalReturns(t *testing.T) {
	const wrapper = `func decode(r *Reader) (uint64, error) {
	v, err := decodeValue(r)
	if err != nil { return 0, err }
	return uint64(v), nil
}`
	const callee = `func decodeValue(r *Reader) (uint8, error) { return 0, nil }`
	for _, tc := range []struct {
		name    string
		wrapper string
		callee  string
	}{
		{"signed first result", wrapper, strings.Replace(callee, "uint8", "int8", 1)},
		{"unsigned second result", wrapper, `func decodeValue(r *Reader) (int, uint8) { return -1, 0 }`},
		{"unknown result type", wrapper, strings.Replace(callee, "uint8", "Value", 1)},
		{"shadowed builtin type", wrapper, "type uint8 = int8\n" + callee},
		{"absent local callee", wrapper, ""},
		{"external callee", strings.Replace(wrapper, "decodeValue(r)", "other.decodeValue(r)", 1), callee},
		{"shadowed function", strings.Replace(wrapper, "v, err :=", "decodeValue := func(*Reader) (int, error) { return -1, nil }; v, err :=", 1), callee},
		{"function parameter", strings.Replace(wrapper, "r *Reader)", "r *Reader, decodeValue func(*Reader) (int, error))", 1), callee},
		{"reassigned value", strings.Replace(wrapper, "return uint64(v)", "v = -1; return uint64(v)", 1), callee},
		{"incremented value", strings.Replace(wrapper, "return uint64(v)", "v++; return uint64(v)", 1), callee},
		{"redeclared signed value", strings.Replace(wrapper, "return uint64(v), nil", "{ var v int = -1; return uint64(v), nil }", 1), callee},
		{"branch does not prove outer value", `func decode(r *Reader, v int) uint64 { if r != nil { v, _ := decodeValue(r); _ = v }; return uint64(v) }`, callee},
		{"second assigned result", strings.Replace(wrapper, "v, err :=", "err, v :=", 1), callee},
		{"closure mutates value", strings.Replace(wrapper, "return uint64(v)", "change := func() { v = -1 }; change(); return uint64(v)", 1), callee},
		{"address escapes", strings.Replace(wrapper, "return uint64(v)", "_ = &v; return uint64(v)", 1), callee},
		{"nested declaration", strings.Replace(wrapper, "v, err := decodeValue(r)", "if r != nil { v, err := decodeValue(r)", 1) + "}", callee},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "package sample\n" + tc.wrapper + "\n" + tc.callee
			if count, err := checkGeneratedPackageUint64Conversions("generated.go", map[string][]byte{"generated.go": []byte(source)}); err == nil {
				t.Fatalf("accepted unproven local return: count=%d", count)
			}
		})
	}
}

func TestGeneratedUint64ConversionsRequirePackageContext(t *testing.T) {
	const source = `package sample
 type Reader struct{}
 func decode(r *Reader) (uint64, error) { v, err := decodeValue(r); if err != nil { return 0, err }; return uint64(v), nil }
 func decodeValue(r *Reader) (uint8, error) { return 0, nil }`
	if count, err := CheckGeneratedUint64Conversions("generated.go", []byte(source)); err == nil {
		t.Fatalf("source-only proof admitted a local return: count=%d", count)
	}
}

func TestGeneratedUint64ConversionsRejectSiblingSignedAlias(t *testing.T) {
	const source = `package sample
 type Reader struct{}
 func decode(r *Reader) (uint64, error) { v, err := decodeValue(r); if err != nil { return 0, err }; return uint64(v), nil }
 func decodeValue(r *Reader) (uint8, error) { return -1, nil }`
	const sibling = "package sample; type uint8 = int8"
	directory := t.TempDir()
	for name, contents := range map[string]string{"generated.go": source, "alias.go": sibling} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Prove that this is a valid Go package with an actually signed first result.
	set := token.NewFileSet()
	var parsed []*ast.File
	sources, err := readGeneratedPackageSources(directory)
	if err != nil {
		t.Fatal(err)
	}
	for name, contents := range sources {
		file, err := parser.ParseFile(set, name, contents, 0)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, file)
	}
	if _, err := new(types.Config).Check("sample", set, parsed, nil); err != nil {
		t.Fatal(err)
	}
	if count, err := checkGeneratedPackageUint64Conversions("generated.go", sources); err == nil {
		t.Fatalf("package proof admitted a signed sibling alias: count=%d", count)
	}
}

func TestGeneratedUint64PackageNamespaceFailsClosed(t *testing.T) {
	const source = "package sample; func encode(v uint8) uint64 { return uint64(v) }"
	for _, builtin := range []string{"int", "uint", "uint8", "uint16", "uint32", "uint64", "byte"} {
		t.Run(builtin, func(t *testing.T) {
			sources := map[string][]byte{"generated.go": []byte(source), "alias.go": []byte("package sample; type " + builtin + " = int8")}
			if count, err := checkGeneratedPackageUint64Conversions("generated.go", sources); err == nil {
				t.Fatalf("shadowed builtin admitted: count=%d", count)
			}
		})
	}
	for name, sources := range map[string]map[string][]byte{
		"missing checked source":    {"other.go": []byte(source)},
		"unparseable sibling":       {"generated.go": []byte(source), "other.go": []byte("package sample; type")},
		"different sibling package": {"generated.go": []byte(source), "other.go": []byte("package other")},
	} {
		t.Run(name, func(t *testing.T) {
			if count, err := checkGeneratedPackageUint64Conversions("generated.go", sources); err == nil {
				t.Fatalf("incomplete namespace admitted: count=%d", count)
			}
		})
	}
}

func TestGeneratedPackageUint64Conversions(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "ts*", "*", "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no generated package sources found")
	}
	for _, file := range files {
		sources, err := readGeneratedPackageSources(filepath.Dir(file))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := checkGeneratedPackageUint64Conversions(filepath.Base(file), sources); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
}
