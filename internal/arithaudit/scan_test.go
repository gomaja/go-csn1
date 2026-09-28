package arithaudit

import (
	"strings"
	"testing"
)

func TestScanSourceFindsWireArithmeticAndNarrowing(t *testing.T) {
	source := []byte(`package sample
func Decode(data []byte) uint8 {
	n := len(data) * 8
	return uint8(n)
}
`)
	findings, err := ScanSource("runtime/sample.go", source)
	if err != nil {
		t.Fatal(err)
	}
	joined := make([]string, len(findings))
	for i, finding := range findings {
		joined[i] = finding.Signature()
	}
	got := strings.Join(joined, "\n")
	for _, want := range []string{
		"runtime/sample.go:arithmetic:len(data) * 8",
		"runtime/sample.go:conversion:uint8(n)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestScanSourceFindsStateArithmeticInNoArgumentMethod(t *testing.T) {
	source := []byte(`package sample
type span struct { At, Count int }
type Writer struct { span span }
func (w *Writer) current() int { return w.span.At + w.span.Count }
`)
	findings, err := ScanSource("runtime/sample.go", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Expression == "w.span.At + w.span.Count" {
			return
		}
	}
	t.Fatal("state arithmetic in receiver method was not scanned")
}

func TestClassificationRejectsMissingAndStaleEntries(t *testing.T) {
	finding := Finding{File: "runtime/sample.go", Line: 3, Kind: "arithmetic", Expression: "len(data) * 8"}
	if err := CheckAllowlist([]Finding{finding}, nil); err == nil {
		t.Fatal("unclassified hit accepted")
	}
	allowed := map[string]Classification{finding.Signature(): {Count: 1, Reason: "BOUNDED: input length checked at runtime/sample.go:2"}}
	if err := CheckAllowlist([]Finding{finding}, allowed); err != nil {
		t.Fatal(err)
	}
	if err := CheckAllowlist([]Finding{finding, finding}, allowed); err == nil {
		t.Fatal("new occurrence of a known expression accepted")
	}
	allowed["runtime/other.go:arithmetic:i++"] = Classification{Count: 1, Reason: "BOUNDED: loop at runtime/other.go:1"}
	if err := CheckAllowlist([]Finding{finding}, allowed); err == nil {
		t.Fatal("stale classification accepted")
	}
}

func TestNarrowingRejectsWidthAndOffsetOverflow(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		bad  bool
	}{
		{"fitting width", `width, _ := r.Eval("8"); v, _ := r.ReadUint(width); return uint8(v)`, false},
		{"narrowed width", `width, _ := r.Eval("9"); v, _ := r.ReadUint(width); return uint8(v)`, true},
		{"later width cannot hide narrow read", `width, _ := r.Eval("9"); v, _ := r.ReadUint(width); width, _ = r.Eval("1"); return uint8(v)`, true},
		{"offset overflow", `width, _ := r.Eval("8"); v, _ := r.ReadUint(width); v += 1; return uint8(v)`, true},
		{"bounded offset", `width, _ := r.Eval("3"); v, _ := r.ReadUint(width); v += 1; return uint8(v)`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("package sample\nfunc decode(r *Reader) uint8 {" + tc.body + "}\n")
			count, err := CheckGeneratedNarrowing("generated.go", source)
			if (err != nil) != tc.bad {
				t.Fatalf("count=%d err=%v, bad=%t", count, err, tc.bad)
			}
			if !tc.bad && count != 1 {
				t.Fatalf("checked %d casts, want 1", count)
			}
		})
	}
}

func TestGeneratedUint64ConversionsRequireUnsignedSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		bad    bool
	}{
		{"unsigned parameter", `package sample; func encode(v uint16) uint64 { return uint64(v) }`, false},
		{"decoded value", `package sample; func decode(r *Reader) uint64 { v, _ := r.ReadUint(4); return uint64(v) }`, false},
		{"signed parameter", `package sample; func encode(v int) uint64 { return uint64(v) }`, true},
		{"unproven local", `package sample; func decode(v any) uint64 { return uint64(v) }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count, err := CheckGeneratedUint64Conversions("generated.go", []byte(tc.source))
			if (err != nil) != tc.bad {
				t.Fatalf("count=%d err=%v, bad=%t", count, err, tc.bad)
			}
			if !tc.bad && count != 1 {
				t.Fatalf("checked %d conversions, want 1", count)
			}
		})
	}
}
