package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const emrOriginFixture = `package sample
import "github.com/gomaja/go-csn1/runtime"
func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes(data []byte, collector *runtime.Collector) (runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent], error) {
	if err := runtime.CheckInput(data); err != nil {
		return runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]{}, err
	}
	if len(data) > 22 {
		return runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]{}, &runtime.DecodeError{Kind: runtime.Limit, Offset: 184, Detail: "PACKETENHANCEDMEASUREMENTREPORTMessageContent exceeds 170 bits"}
	}
	input := data
	r, err := emrContentReader(input, min(runtime.InputBits(input), 170), collector)
	if err != nil {
		return runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]{}, err
	}
	r.SetZeroExtension(false)
	v, err := decodePACKETENHANCEDMEASUREMENTREPORTMessageContent(r)
	if err != nil {
		return runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]{}, err
	}
	consumed := r.Position() - 14
	tail := runtime.TrailingBits(data, consumed)
	v.Wire = runtime.Seal(v, data, consumed, tail, r.Wire())
	return runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent]{Value: v, BitsConsumed: consumed, Tail: tail}, nil
}
func emrContentReader(data []byte, bits int, c *runtime.Collector) (*runtime.Reader, error) {
	if bits < 0 || bits > runtime.InputBits(data) {
		return nil, fmt.Errorf("content extent outside input")
	}
	const origin = 14
	joined := make([]byte, (origin+bits+7)/8)
	for i := 0; i < bits; i++ {
		bit := (data[i/8] >> uint(7-i%8)) & 1
		joined[(origin+i)/8] |= bit << uint(7-(origin+i)%8)
	}
	r, err := runtime.NewReaderBits(joined, origin+bits, c)
	if err != nil {
		return nil, err
	}
	if _, err := r.ReadUint(origin); err != nil {
		return nil, err
	}
	return r, nil
}
func emrContentBytes(raw runtime.BitString, completion runtime.BitString) ([]byte, error) {
	const origin = 14
	bits := raw.BitLength - origin
	if bits < 0 || bits > 170 || len(raw.Bytes) < (raw.BitLength+7)/8 || completion.BitLength < 0 || completion.BitLength > 6 || len(completion.Bytes) < (completion.BitLength+7)/8 {
		return nil, fmt.Errorf("content extent outside carrier")
	}
	out := make([]byte, (bits+completion.BitLength+7)/8)
	for i := 0; i < bits; i++ {
		bit := (raw.Bytes[(origin+i)/8] >> uint(7-(origin+i)%8)) & 1
		out[i/8] |= bit << uint(7-i%8)
	}
	for i := 0; i < completion.BitLength; i++ {
		bit := (completion.Bytes[i/8] >> uint(7-i%8)) & 1
		out[(bits+i)/8] |= bit << uint(7-(bits+i)%8)
	}
	return out, nil
}
func emrEncodeContent(v PACKETENHANCEDMEASUREMENTREPORTMessageContent) ([]byte, error) {
	probe := runtime.NewWriter()
	probe.WithWire(v.Wire)
	if target, present := probe.RemainingCanonicalTarget(); present {
		if target != 170 && target != 176 {
			return nil, fmt.Errorf("canonical target differs from the significant content carrier")
		}
		v = runtime.CanonicalTo(v, 170)
	}
	completion := runtime.BitString{}
	if v.Wire.BitsConsumed == 170 && v.Wire.TransmittedBits == 176 && v.Wire.Tail.BitLength == 6 {
		completion = v.Wire.Tail
	}
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		return nil, err
	}
	if err := EncodePACKETENHANCEDMEASUREMENTREPORTMessageContentTo(w, v); err != nil {
		return nil, err
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		return nil, err
	}
	return emrContentBytes(raw, completion)
}`

func emrOriginFixtureFindings(t *testing.T, source, sibling string) (*citationChecker, []Finding) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if sibling != "" {
		if err := os.WriteFile(filepath.Join(root, "extra.go"), []byte(sibling), 0600); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := ScanSource("sample/generated.go", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for index := range findings {
		findings[index].File = "generated.go"
	}
	if len(findings) == 0 {
		t.Fatal("no origin/copy arithmetic scanned")
	}
	return &citationChecker{root: root, files: make(map[string]citationFile)}, findings
}

func TestGeneratedEMROriginArithmetic(t *testing.T) {
	checker, findings := emrOriginFixtureFindings(t, emrOriginFixture, "")
	for _, finding := range findings {
		if err := checker.checkGeneratedResidual(finding); err != nil {
			t.Fatalf("%s: %v", finding.Key(), err)
		}
	}
	if len(findings) != 33 {
		t.Fatalf("verified %d occurrences, want 33", len(findings))
	}
	t.Logf("verified %d origin/copy arithmetic occurrences", len(findings))
}

func TestGeneratedEMROriginArithmeticRejectsChanges(t *testing.T) {
	cases := []struct{ name, old, replacement, sibling string }{
		{name: "reader lower guard", old: "bits < 0 || bits > runtime.InputBits(data)", replacement: "bits > runtime.InputBits(data)"},
		{name: "reader upper guard", old: "bits < 0 || bits > runtime.InputBits(data)", replacement: "bits < 0"},
		{name: "unbounded input source", old: "runtime.InputBits(data)", replacement: "len(data)*8"},
		{name: "origin changed", old: "const origin = 14", replacement: "const origin = 15"},
		{name: "allocation shrunk", old: "(origin+bits+7)/8", replacement: "(bits+7)/8"},
		{name: "reader loop off by one", old: "i < bits", replacement: "i <= bits"},
		{name: "reader advances loop bound", old: "bit := (data[i/8]", replacement: "bits++; bit := (data[i/8]"},
		{name: "reader index changed", old: "joined[(origin+i)/8]", replacement: "joined[(origin+i+8)/8]"},
		{name: "shift changed", old: "uint(7-i%8)", replacement: "uint(8-i%8)"},
		{name: "negative shift", old: "uint(7-i%8)", replacement: "uint(i%8-7)"},
		{name: "reader bound enlarged", old: "runtime.NewReaderBits(joined, origin+bits, c)", replacement: "runtime.NewReaderBits(joined, origin+bits+1, c)"},
		{name: "unchecked origin read", old: "if _, err := r.ReadUint(origin); err != nil", replacement: "if _, err := r.ReadUint(origin); false"},
		{name: "raw subtraction source", old: "bits := raw.BitLength - origin", replacement: "bits := completion.BitLength - origin"},
		{name: "raw ceiling enlarged", old: "bits > 170", replacement: "bits > 176"},
		{name: "raw short circuit order", old: "bits < 0 || bits > 170 || len(raw.Bytes) < (raw.BitLength+7)/8", replacement: "len(raw.Bytes) < (raw.BitLength+7)/8 || bits < 0 || bits > 170"},
		{name: "raw backing guard removed", old: "len(raw.Bytes) < (raw.BitLength+7)/8 || ", replacement: ""},
		{name: "completion negative guard", old: "completion.BitLength < 0 || ", replacement: ""},
		{name: "completion ceiling enlarged", old: "completion.BitLength > 6", replacement: "completion.BitLength > 7"},
		{name: "completion backing guard removed", old: " || len(completion.Bytes) < (completion.BitLength+7)/8", replacement: ""},
		{name: "copy under allocation", old: "(bits+completion.BitLength+7)/8", replacement: "(bits+completion.BitLength)/8"},
		{name: "copy variable shadow", old: "out := make([]byte", replacement: "bits = -1; out := make([]byte"},
		{name: "completion index changed", old: "out[(bits+i)/8]", replacement: "out[(bits+i+8)/8]"},
		{name: "producer bypass finish", old: "raw, err := w.FinishBits(runtime.BitString{})", replacement: "raw, err := unsafeRaw()"},
		{name: "producer ignored finish error", old: "raw, err := w.FinishBits(runtime.BitString{})\n\tif err != nil", replacement: "raw, err := w.FinishBits(runtime.BitString{})\n\tif false"},
		{name: "producer changes raw", old: "return emrContentBytes(raw, completion)", replacement: "raw.BitLength = -1; return emrContentBytes(raw, completion)"},
		{name: "producer substitutes completion", old: "return emrContentBytes(raw, completion)", replacement: "return emrContentBytes(completion, completion)"},
		{name: "producer fake writer", old: "w := runtime.NewWriter()", replacement: "w := otherWriter()"},
		{name: "decode subtraction changed", old: "consumed := r.Position() - 14", replacement: "consumed := r.Position() - 15"},
		{name: "decode ignored reader failure", old: "r, err := emrContentReader(input, min(runtime.InputBits(input), 170), collector)\n\tif err != nil", replacement: "r, err := emrContentReader(input, min(runtime.InputBits(input), 170), collector)\n\tif false"},
		{name: "runtime import replaced", old: "github.com/gomaja/go-csn1/runtime", replacement: "example.test/unsafe/runtime"},
		{name: "same file helper escape", old: "func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes", replacement: "var escaped=emrContentBytes; func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes"},
		{name: "same file helper arbitrary caller", old: "func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes", replacement: "func extra(){emrContentBytes(runtime.BitString{}, runtime.BitString{})}; func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes"},
		{name: "same file builtin shadow", old: "func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes", replacement: "func len(v any)int{return -1}; func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes"},
		{name: "sibling helper caller", sibling: "package sample; func extra(){emrContentBytes(runtime.BitString{},runtime.BitString{})}"},
		{name: "sibling helper escape", sibling: "package sample; var escaped=emrContentBytes"},
		{name: "sibling reader escape", sibling: "package sample; var escaped=emrContentReader"},
		{name: "sibling builtin shadow", sibling: "package sample; func min(a,b int)int{return a+b}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := emrOriginFixture
			if tc.old != "" {
				if !strings.Contains(source, tc.old) {
					t.Fatalf("mutation target absent: %q", tc.old)
				}
				source = strings.Replace(source, tc.old, tc.replacement, 1)
			}
			checker, findings := emrOriginFixtureFindings(t, source, tc.sibling)
			for _, finding := range findings {
				if err := checker.checkGeneratedResidual(finding); err == nil {
					t.Fatalf("mutation accepted: %s", finding.Key())
				}
			}
		})
	}
}

func TestGeneratedEMROriginArithmeticRequiresExactOccurrence(t *testing.T) {
	checker, findings := emrOriginFixtureFindings(t, emrOriginFixture, "")
	for _, change := range []string{"column", "line", "expression", "kind", "missing column"} {
		t.Run(change, func(t *testing.T) {
			finding := findings[0]
			switch change {
			case "column":
				finding.Column++
			case "line":
				finding.Line++
			case "expression":
				finding.Expression = "raw.BitLength - origin"
			case "kind":
				finding.Kind = "conversion"
			case "missing column":
				finding.Column = 0
			}
			if err := checker.checkGeneratedResidual(finding); err == nil {
				t.Fatalf("unproven occurrence accepted: %s", finding.Key())
			}
		})
	}
}

func TestGeneratedEMROriginPublishedHelpers(t *testing.T) {
	root := filepath.Join("..", "..")
	path := "ts44060/packetenhancedmeasurement/generated.go"
	contents, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	findings, err := ScanSource(path, contents)
	if err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	file, err := checker.source(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := CheckGeneratedLoopArithmetic(path, contents); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, finding := range findings {
		fn := functionAt(file, finding.Line)
		if fn == nil {
			continue
		}
		switch fn.Name.Name {
		case "emrContentReader", "emrContentBytes", "decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes", "emrBitmapFields":
			if err := checker.checkGeneratedResidual(finding); err != nil {
				t.Fatalf("%s: %v", finding.Key(), err)
			}
			checked++
			t.Logf("%s", finding.Key())
		}
	}
	if checked != 34 {
		t.Fatalf("verified %d published origin/copy/bitmap occurrences, want 34", checked)
	}
}

func TestGeneratedEMRCopyLoopIntegration(t *testing.T) {
	increments, next, err := CheckGeneratedLoopArithmetic("sample/generated.go", []byte(emrOriginFixture))
	if err != nil {
		t.Fatal(err)
	}
	if increments != 3 || next != 0 {
		t.Fatalf("increments=%d next=%d, want3/0", increments, next)
	}
	for _, tc := range []struct{ old, replacement string }{
		{"i < bits", "i <= bits"},
		{"i < completion.BitLength", "i <= completion.BitLength"},
		{"i++", "i += 2"},
		{"bits < 0 || bits > runtime.InputBits(data)", "bits > runtime.InputBits(data)"},
	} {
		modified := strings.Replace(emrOriginFixture, tc.old, tc.replacement, 1)
		if _, _, err := CheckGeneratedLoopArithmetic("sample/generated.go", []byte(modified)); err == nil {
			t.Fatalf("copy-loop mutation accepted: %s", tc.old)
		}
	}
}

const packetEMRBitmapFixture = `package sample
func emrBitmapFields(v *PACKETENHANCEDMEASUREMENTREPORTMessageContent) {
	if v.NCMeasurementReport.REPORTINGQUANTITYList != nil {
		last := 0
		for index, report := range *v.NCMeasurementReport.REPORTINGQUANTITYList {
			if report != nil {
				last = index + 1
			}
		}
		entries := (*v.NCMeasurementReport.REPORTINGQUANTITYList)[:last]
		if last == 0 {
			entries = nil
		}
		v.NCMeasurementReport.REPORTINGQUANTITYList = &entries
	}
}`

func TestGeneratedPacketEMRBitmapProof(t *testing.T) {
	checker, findings := emrOriginFixtureFindings(t, packetEMRBitmapFixture, "")
	if len(findings) != 1 {
		t.Fatalf("got%d findings, want1", len(findings))
	}
	if err := checker.checkGeneratedResidual(findings[0]); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedPacketEMRBitmapProofRejectsChanges(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"last initial value", "last := 0", "last := -1"},
		{"range changed", "range *v.NCMeasurementReport.REPORTINGQUANTITYList", "range other"},
		{"index shadowed", "last = index + 1", "index := -2; last = index + 1"},
		{"last changed after range", "entries := (*v.NCMeasurementReport.REPORTINGQUANTITYList)[:last]", "last++; entries := (*v.NCMeasurementReport.REPORTINGQUANTITYList)[:last]"},
		{"nil guard removed", "if v.NCMeasurementReport.REPORTINGQUANTITYList != nil", "if true"},
		{"report guard reversed", "if report != nil", "if report == nil"},
		{"wrong prefix high", "[:last]", "[:last+1]"},
		{"missing empty normalization", "entries = nil", "entries = other"},
		{"RR root substitution", "v.NCMeasurementReport.REPORTINGQUANTITYList", "v.REPORTINGQUANTITYList"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Replace(packetEMRBitmapFixture, tc.old, tc.replacement, 1)
			if text == packetEMRBitmapFixture {
				t.Fatal("mutation target absent")
			}
			checker, findings := emrOriginFixtureFindings(t, text, "")
			found := 0
			for _, finding := range findings {
				if finding.Expression != "index + 1" {
					continue
				}
				found++
				if err := checker.checkGeneratedResidual(finding); err == nil {
					t.Fatal("bitmap mutation accepted")
				}
			}
			if found != 1 {
				t.Fatalf("got%d index+1 occurrences, want1", found)
			}
		})
	}
}
