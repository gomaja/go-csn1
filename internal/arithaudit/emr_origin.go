package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// checkGeneratedEMRArithmetic pins the complete emitted origin/copy functions
// and their two callers. TS 44.060 V19.0.0 §10.3.2 figure 10.3.2.1 and
// §11.2.9d supply the 14-bit origin and 170 significant content bits.
// runtime/expr.go maxBits is 1<<20. runtime/direct.go InputBits bounds the
// reader input; Reader.bit checks the position before advancing it. Thus the
// origin reader's additions fit even a 32-bit int and Position()-14 cannot
// overflow. Its loop shifts are 0..7 and both indices address allocated bits.
//
// The raw subtraction precedes its local guard: its provenance is essential.
// runtime/bit_extent.go Writer.FinishBits returns the checked writer position
// and backing bytes, bounded by runtime/direct.go Writer.put/logicalPosition.
// The sole producer below calls FinishBits and checks its error. Consequently
// raw.BitLength-origin and raw.BitLength+7 fit int. Short-circuit guards then
// bound bits to 0..170 and completion to 0..6 before the allocation/copy loops.
// Every package reference to these private helpers must be a pinned direct
// call: an arbitrary BitString or an escaping helper cannot inherit this proof.
func (c *citationChecker) checkGeneratedEMRArithmetic(file citationFile, fn *ast.FuncDecl, finding Finding) error {
	fail := func() error { return fmt.Errorf("generated EMR origin/copy proof differs: %s", finding.Key()) }
	functions := make(map[string]*ast.FuncDecl)
	for _, decl := range file.file.Decls {
		if function, ok := decl.(*ast.FuncDecl); ok {
			if _, reviewed := emrOriginFunctions[function.Name.Name]; reviewed {
				if functions[function.Name.Name] != nil {
					return fail()
				}
				functions[function.Name.Name] = function
			}
		}
	}
	for name, expected := range emrOriginFunctions {
		function := functions[name]
		if function == nil || emrPrintedNode(file, function) != expected {
			return fail()
		}
	}
	// Local declarations are pinned above; package declarations and imports
	// must not replace builtins or the runtime package used by these functions.
	for _, decl := range file.file.Decls {
		var names []string
		switch declaration := decl.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				names = append(names, declaration.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					names = append(names, item.Name.Name)
				case *ast.ValueSpec:
					for _, name := range item.Names {
						names = append(names, name.Name)
					}
				}
			}
		}
		for _, name := range names {
			if name == "len" || name == "make" || name == "min" || name == "uint" {
				return fail()
			}
		}
	}
	runtimeImport := false
	for _, imp := range file.file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return fail()
		}
		name := filepath.Base(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." || name == "len" || name == "make" || name == "min" || name == "uint" {
			return fail()
		}
		if name == "runtime" {
			if path != "github.com/gomaja/go-csn1/runtime" {
				return fail()
			}
			runtimeImport = true
		}
	}
	if !runtimeImport {
		return fail()
	}
	proven := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		at := file.set.Position(node.Pos())
		if at.Line != finding.Line || finding.Column == 0 || at.Column != finding.Column {
			return true
		}
		kind := ""
		switch node.(type) {
		case *ast.BinaryExpr, *ast.IncDecStmt:
			kind = "arithmetic"
		case *ast.CallExpr:
			kind = "conversion"
		}
		if kind == finding.Kind && emrPrintedNode(file, node) == finding.Expression {
			proven = true
		}
		return true
	})
	if !proven {
		return fail()
	}
	if err := c.checkEMRPrivateCalls(file, finding); err != nil {
		return err
	}
	return nil
}

func emrPrintedNode(file citationFile, node ast.Node) string {
	var out bytes.Buffer
	if format.Node(&out, file.set, node) != nil {
		return ""
	}
	return out.String()
}

func (c *citationChecker) checkEMRPrivateCalls(file citationFile, finding Finding) error {
	fail := func() error { return fmt.Errorf("generated EMR helper has an unreviewed reference: %s", finding.Key()) }
	allowed := make(map[token.Pos]bool)
	for _, decl := range file.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "emrContentReader" || fn.Name.Name == "emrContentBytes" {
			allowed[fn.Name.Pos()] = true
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			if fn.Name.Name == "decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes" && id.Name == "emrContentReader" || fn.Name.Name == "emrEncodeContent" && id.Name == "emrContentBytes" {
				allowed[id.Pos()] = true
			}
			return true
		})
	}
	invalid := false
	ast.Inspect(file.file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && (id.Name == "emrContentReader" || id.Name == "emrContentBytes") && !allowed[id.Pos()] {
			invalid = true
		}
		return true
	})
	if invalid {
		return fail()
	}
	// A new sibling source file can call an unexported helper too. Reject those
	// references rather than relying on the current one-file generated layout.
	directory := filepath.Dir(finding.File)
	entries, err := os.ReadDir(filepath.Join(c.root, directory))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") || entry.Name() == filepath.Base(finding.File) {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(c.root, directory, entry.Name()))
		if err != nil {
			return err
		}
		sibling, err := parser.ParseFile(token.NewFileSet(), entry.Name(), contents, 0)
		if err != nil {
			return err
		}
		if sibling.Name.Name != file.file.Name.Name {
			return fail()
		}
		ast.Inspect(sibling, func(node ast.Node) bool {
			if id, ok := node.(*ast.Ident); ok && (id.Name == "emrContentReader" || id.Name == "emrContentBytes" || id.Name == "len" || id.Name == "make" || id.Name == "min" || id.Name == "uint") {
				// Builtin uses are harmless; their declarations are not.
				if id.Name == "emrContentReader" || id.Name == "emrContentBytes" || id.Obj != nil && id.Obj.Pos() == id.Pos() {
					invalid = true
				}
			}
			return true
		})
	}
	if invalid {
		return fail()
	}
	return nil
}

var emrOriginFunctions = map[string]string{
	"decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes": `func decodePACKETENHANCEDMEASUREMENTREPORTMessageContentBytes(data []byte, collector *runtime.Collector) (runtime.Decoded[PACKETENHANCEDMEASUREMENTREPORTMessageContent], error) {
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
}`,
	"emrContentReader": `func emrContentReader(data []byte, bits int, c *runtime.Collector) (*runtime.Reader, error) {
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
}`,
	"emrContentBytes": `func emrContentBytes(raw runtime.BitString, completion runtime.BitString) ([]byte, error) {
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
}`,
	"emrEncodeContent": `func emrEncodeContent(v PACKETENHANCEDMEASUREMENTREPORTMessageContent) ([]byte, error) {
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
}`,
}

// checkGeneratedEMRLoopIncrements admits only the two fully pinned copy-loop
// functions. Each i starts at zero and advances strictly below its int bound;
// the raw helper first limits content to 170 bits and completion to six.
func checkGeneratedEMRLoopIncrements(file citationFile, fn *ast.FuncDecl) (int, error) {
	expected, present := emrOriginFunctions[fn.Name.Name]
	if !present || emrPrintedNode(file, fn) != expected {
		return 0, fmt.Errorf("generated EMR copy-loop proof differs")
	}
	count := 0
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.IncDecStmt); ok {
			count++
		}
		return true
	})
	return count, nil
}

func isPacketEMRBitmap(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
		return false
	}
	pointer, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	name, ok := pointer.X.(*ast.Ident)
	return ok && name.Name == "PACKETENHANCEDMEASUREMENTREPORTMessageContent"
}

// checkGeneratedPacketEMRBitmap pins the Packet EMR helper separately from the
// existing RR helper. A Go slice range index is 0..len(slice)-1, so index+1
// is at most len(slice) and fits int even at the host's largest slice length.
// The full body pin rules out changed ranges, shadowed indices and last writes;
// the same bound also proves the retained prefix slice high index.
func checkGeneratedPacketEMRBitmap(file citationFile, fn *ast.FuncDecl, finding Finding) error {
	fail := func() error { return fmt.Errorf("generated Packet EMR bitmap proof differs: %s", finding.Key()) }
	if emrPrintedNode(file, fn) != packetEMRBitmapFunction {
		return fail()
	}
	proven := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		binary, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		at := file.set.Position(binary.Pos())
		if finding.Kind == "arithmetic" && finding.Expression == "index + 1" && at.Line == finding.Line && finding.Column != 0 && at.Column == finding.Column && emrPrintedNode(file, binary) == finding.Expression {
			proven = true
		}
		return true
	})
	if !proven {
		return fail()
	}
	return nil
}

const packetEMRBitmapFunction = `func emrBitmapFields(v *PACKETENHANCEDMEASUREMENTREPORTMessageContent) {
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
