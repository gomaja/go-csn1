package registry_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/celldata"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover"
	"github.com/gomaja/go-csn1/ts44060/padding"
)

// TS 44.060 V19.0.0 §11 pp12938–12944: a received syntax-error arm
// replays exactly; the syntax-trial sentinel is internal to that validation.
func receiverSyntaxReplayWitness(t *testing.T) {
	t.Helper()
	v, err := padding.DecodePaddingBits([]byte{0xff})
	if err != nil {
		t.Fatal(err)
	}
	if v.Value.IgnoreChoice.Ignore == nil || v.Value.IgnoreChoice.Ignore.Ignored == nil {
		t.Fatal("fixture did not select receiver Ignore")
	}
	out, err := padding.EncodePaddingBits(v.Value)
	if err != nil || !bytes.Equal(out, []byte{0xff}) {
		t.Fatalf("receiver syntax replay: %x %v", out, err)
	}
	if _, err = padding.EncodePaddingBitsCanonical(v.Value); err == nil {
		t.Fatal("receiver arm became a canonical sender")
	}
}

// §11.2.17c consumes complete octets only and refuses corrupt received extents.
func wholeOctetReceivedExtentWitness(t *testing.T) {
	t.Helper()
	decoded, err := celldata.DecodePacketServingCellSIMessageContent([]byte{0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Value.PAGEMODEGroupFallback.Known == nil {
		t.Fatal("missing typed container")
	}
	wire := decoded.Value.Wire
	wire.TransmittedBits = -1
	value := celldata.PacketServingCellSIMessageContent{PAGEMODEGroupFallback: celldata.PacketServingCellSIMessageContentPAGEMODEGroupFallback{Known: &celldata.PacketServingCellSIMessageContentPAGEMODEGroupFallbackKnown{}}}
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err = w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	err = celldata.EncodePacketServingCellSIMessageContentTo(w, value)
	requireSenderWitnessError(t, err, "11.2.17c", "invalid received whole-octet remainder extent", false)
}

func requireSenderWitnessError(t *testing.T, err error, clause, message string, context bool) {
	t.Helper()
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || operation.Kind != runtime.InvalidValue || operation.Location.Source.Standard != "TS 44.060" || operation.Location.Source.Clause != clause || !strings.Contains(err.Error(), message) {
		t.Fatalf("source-specific typed sender refusal %q: %v", message, err)
	}
	if errors.Is(err, runtime.ErrContextRequired) != context {
		t.Fatalf("context classification: %v", err)
	}
}

func dtmSenderSeed(t *testing.T) dtmhandover.DTMHandoverPSRadioResourcesIE {
	t.Helper()
	data, err := os.ReadFile("../ts44060/dtmhandover/testdata/handover_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name string `json:"name"`
		Bits string `json:"bits"`
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Name != "12.46-mode1-bitmap128-power0-subset1-usfs0-rich0" {
			continue
		}
		raw := make([]byte, (len(row.Bits)+7)/8)
		for i, b := range row.Bits {
			if b == '1' {
				raw[i/8] |= 1 << uint(7-i%8)
			}
		}
		r, err := runtime.NewChoiceReaderBits(raw, len(row.Bits), nil)
		if err != nil {
			t.Fatal(err)
		}
		v, err := dtmhandover.DecodeDTMHandoverPSRadioResourcesIEFrom(r)
		if err != nil || r.Position() != len(row.Bits) {
			t.Fatalf("DTM typed seed: %v", err)
		}
		return runtime.CanonicalTo(v, len(row.Bits))
	}
	t.Fatal("missing exact selected DTM seed")
	return dtmhandover.DTMHandoverPSRadioResourcesIE{}
}

// The source fixture has a non-octet null release boundary. Its exact target
// must be retained; byte completion is a separate canonical-padding question.
func encodeDTMSourceWitness(v dtmhandover.DTMHandoverPSRadioResourcesIE) ([]byte, error) {
	w := runtime.NewWriter()
	w.WithWire(v.Wire)
	if err := dtmhandover.EncodeDTMHandoverPSRadioResourcesIETo(w, v); err != nil {
		return nil, err
	}
	bits, err := w.FinishBits(runtime.BitString{})
	return bits.Bytes, err
}

// TS 44.060 V19.0.0 §12.46 pp24412–24426 derives N and M from
// the enclosing typed allocation. Standalone N/M values cannot supply scope.
func dtmAllocationSenderWitnesses(t *testing.T) {
	t.Helper()
	v := dtmSenderSeed(t)
	if _, err := encodeDTMSourceWitness(v); err != nil {
		t.Fatalf("valid source allocation control: %v", err)
	}
	group := v.GPRSModeChoice.EGPRSMode.EGPRSMode.Group.GlobalTimeslotDescriptionGroup
	if group == nil || len(group.UplinkAssignmentList) == 0 || group.UplinkAssignmentList[0].UplinkAssignment.TBFTIMESLOTALLOCATION == nil {
		t.Fatal("seed lacks its typed TBF subset")
	}
	u := group.UplinkAssignmentList[0].UplinkAssignment
	w := runtime.NewWriter()
	w.Set("N", 8)
	w.Set("M", 8)
	err := dtmhandover.EncodeUplinkTBFAssignmentStructTo(w, u)
	requireSenderWitnessError(t, err, "12.46", "source allocation or repeated scope is absent", true)
	group.GlobalTimeslotDescription.MSTIMESLOTALLOCATIONChoice.MSTIMESLOTALLOCATION.MSTIMESLOTALLOCATION = 0
	_, err = encodeDTMSourceWitness(v)
	requireSenderWitnessError(t, err, "12.46", "allocation population is below its source minimum", false)
	v = dtmSenderSeed(t)
	group = v.GPRSModeChoice.EGPRSMode.EGPRSMode.Group.GlobalTimeslotDescriptionGroup
	subset := group.UplinkAssignmentList[0].UplinkAssignment.TBFTIMESLOTALLOCATION
	subset.Bytes = make([]byte, len(subset.Bytes))
	_, err = encodeDTMSourceWitness(v)
	requireSenderWitnessError(t, err, "12.46", "TBF allocation must assign at least one timeslot", false)
	dtmDefensiveBitmapGuardWitness(t)
}

// TS 44.060 §12.46 pp24412–24426 limits the enclosing source allocation
// to eight timeslots. Verify the actual scoped producer/consumer chain and
// checked primitive before accepting the defensive uint64 error family.
func dtmDefensiveBitmapGuardWitness(t *testing.T) {
	t.Helper()
	generated, err := os.ReadFile("../ts44060/dtmhandover/generated.go")
	if err != nil {
		t.Fatal(err)
	}
	primitive, err := os.ReadFile("../runtime/direct.go")
	if err != nil {
		t.Fatal(err)
	}
	if !dtmBitmapGuardProof(generated, primitive) {
		t.Fatal("DTM source/primitive/defensive guard proof differs")
	}
	// These controls mutate actual producer/consumer prerequisites, not a copy
	// of the inner expected guard. Every changed prerequisite must be rejected.
	const wrapperName = "encodeUplinkTBFAssignmentStructTBFTIMESLOTALLOCATIONValue"
	const bodyName = wrapperName + "ExpressionBody"
	const bitmapName = "encodeTimeslotDescriptionStructMSTIMESLOTALLOCATIONChoiceMSTIMESLOTALLOCATIONMSTIMESLOTALLOCATION"
	mutations := []struct {
		name, function, before, after string
	}{
		{"source population", wrapperName, "value, err := w.ExpressionResultWithContext(w.ExpressionAggregatePopulation(sourceAggregate0))", "value, err := w.ExpressionResultWithContext(w.ExpressionAggregatePopulation(sourceAggregate1))"},
		{"primitive width", bodyName, `w.ExpressionVariable("N").Operand()`, `w.ExpressionVariable("M").Operand()`},
		{"primitive call", bodyName, "if err := w.WriteBitString(v, width); err != nil {", "if err := w.WriteBitString(v, 65); err != nil {"},
		{"defensive threshold", wrapperName, "if v.BitLength > 64 {", "if v.BitLength > 63 {"},
		{"defensive field", wrapperName, `w.ComponentError(runtime.InvalidValue, "UplinkTBFAssignmentStructTBFTIMESLOTALLOCATIONValue", 0, fmt.Errorf("allocation bitmap exceeds 64 bits"))`, `w.ComponentError(runtime.InvalidValue, "OtherField", 0, fmt.Errorf("allocation bitmap exceeds 64 bits"))`},
		{"bitmap source width", bitmapName, "v uint8", "v uint16"},
		{"EGPRS source reset", "encodeEGPRSModeStructEGPRSWindowSizeGroupGlobalTimeslotDescriptionGroupGlobalTimeslotDescription", "w.ResetAggregate(sourceAggregate0)", "w.ResetAggregate(sourceAggregate1)"},
		{"GPRS source reset", "encodeGPRSModeStructGlobalTimeslotDescriptionGroupGlobalTimeslotDescription", "w.ResetAggregate(sourceAggregate0)", "w.ResetAggregate(sourceAggregate1)"},
	}
	for _, mutation := range mutations {
		mutant := mutateDTMProofFunction(t, generated, mutation.function, "", mutation.before, mutation.after)
		if dtmBitmapGuardProof(mutant, primitive) {
			t.Fatalf("proof accepted changed %s in %s", mutation.name, mutation.function)
		}
	}
	mutantPrimitive := mutateDTMProofFunction(t, primitive, "WriteBitString", "*Writer", "width != value.BitLength", "width == value.BitLength")
	if dtmBitmapGuardProof(generated, mutantPrimitive) {
		t.Fatal("proof accepted changed runtime width equality")
	}
	// Swap the actual wrapper's checked primitive and defensive block.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	matches := 0
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == wrapperName && fn.Recv == nil {
			matches++
			if len(fn.Body.List) != 5 {
				t.Fatal("wrapper mutation prerequisite differs")
			}
			fn.Body.List[2], fn.Body.List[3] = fn.Body.List[3], fn.Body.List[2]
		}
	}
	if matches != 1 {
		t.Fatalf("wrapper reorder mutation matched %d declarations", matches)
	}
	var reordered bytes.Buffer
	if err := format.Node(&reordered, fset, file); err != nil {
		t.Fatal(err)
	}
	if dtmBitmapGuardProof(reordered.Bytes(), primitive) {
		t.Fatal("proof accepted defensive guard before checked primitive")
	}
}

// Mutate only the exact declaration used by the proof. Pin both declaration
// and prerequisite cardinalities so a duplicate elsewhere cannot absorb a control.
func mutateDTMProofFunction(t *testing.T, source []byte, name, receiver, before, after string) []byte {
	t.Helper()
	span := func(data []byte) (int, int) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "proof-source.go", data, 0)
		if err != nil {
			t.Fatal(err)
		}
		start, end, matches := 0, 0, 0
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != name {
				continue
			}
			var recv bytes.Buffer
			if fn.Recv != nil {
				if len(fn.Recv.List) != 1 {
					continue
				}
				if err := format.Node(&recv, fset, fn.Recv.List[0].Type); err != nil {
					t.Fatal(err)
				}
			}
			if recv.String() != receiver {
				continue
			}
			matches++
			start, end = fset.Position(fn.Pos()).Offset, fset.Position(fn.End()).Offset
		}
		if matches != 1 {
			t.Fatalf("mutation target %s.%s matched %d declarations", receiver, name, matches)
		}
		return start, end
	}
	start, end := span(source)
	declaration := source[start:end]
	old, replacement := []byte(before), []byte(after)
	if bytes.Count(declaration, old) != 1 || bytes.Contains(declaration, replacement) {
		t.Fatalf("mutation prerequisite in %s is not unique or already changed", name)
	}
	changed := bytes.Replace(declaration, old, replacement, 1)
	mutant := append([]byte(nil), source[:start]...)
	mutant = append(mutant, changed...)
	mutant = append(mutant, source[end:]...)
	newStart, newEnd := span(mutant)
	if newStart != start || !bytes.Equal(mutant[newStart:newEnd], changed) || bytes.Count(mutant[newStart:newEnd], replacement) != 1 || bytes.Contains(mutant[newStart:newEnd], old) || !bytes.Equal(mutant[:newStart], source[:start]) || !bytes.Equal(mutant[newEnd:], source[end:]) {
		t.Fatalf("mutation escaped exact prerequisite/declaration %s", name)
	}
	return mutant
}

func dtmBitmapGuardProof(generated, primitive []byte) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "generated.go", generated, 0)
	if err != nil {
		return false
	}
	functions := map[string]*ast.FuncDecl{}
	render := func(node any) string {
		var out bytes.Buffer
		if format.Node(&out, fset, node) != nil {
			return ""
		}
		return out.String()
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			functions[fn.Name.Name] = fn
		}
	}
	const field = "UplinkTBFAssignmentStructTBFTIMESLOTALLOCATIONValue"
	wrapper, body := functions["encode"+field], functions["encode"+field+"ExpressionBody"]
	if wrapper == nil || body == nil || len(wrapper.Body.List) != 5 || len(body.Body.List) != 6 {
		return false
	}
	const scope = "\"egprs mode struct\"/\"body/sequence[0]/choice[3]/sequence[1]/reference[1]\"/\"global timeslots\""
	const seen = "\x00source-expression-active/TS 44.060/19.0.0/12.46/" + scope + "/seen"
	seenMatched := false
	ast.Inspect(wrapper.Body.List[0], func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ExpressionVariable" {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil && value == seen {
			seenMatched = true
		}
		return true
	})
	prefix := render(wrapper.Body.List[0])
	if !seenMatched || !strings.Contains(prefix, "err != nil || present != 1") || !strings.Contains(prefix, "runtime.ErrContextRequired") {
		return false
	}
	population, ok := wrapper.Body.List[1].(*ast.BlockStmt)
	if !ok || len(population.List) != 7 {
		return false
	}
	if render(population.List[3]) != "value, err := w.ExpressionResultWithContext(w.ExpressionAggregatePopulation(sourceAggregate0))" || render(population.List[4]) != "if err != nil {\n\treturn err\n}" || !strings.HasPrefix(render(population.List[5]), "if value < 1 {") || render(population.List[6]) != "w.Set(\"n\", uint64(value))" {
		return false
	}
	checked := "if err := encode" + field + "ExpressionBody(w, v); err != nil {\n\treturn err\n}"
	if render(wrapper.Body.List[2]) != checked {
		return false
	}
	defensive, ok := wrapper.Body.List[3].(*ast.IfStmt)
	if !ok || len(defensive.Body.List) < 1 {
		return false
	}
	exactGuard := "if v.BitLength > 64 {\n\treturn w.ComponentError(runtime.InvalidValue, \"" + field + "\", 0, fmt.Errorf(\"allocation bitmap exceeds 64 bits\"))\n}"
	if render(defensive.Body.List[0]) != exactGuard {
		return false
	}
	if render(body.Body.List[2]) != "width, err := w.ExpressionResultWithContext(w.ExpressionVariable(\"N\").Operand())" || render(body.Body.List[3]) != "if err != nil {\n\treturn err\n}" || render(body.Body.List[4]) != "if err := w.WriteBitString(v, width); err != nil {\n\treturn err\n}" {
		return false
	}
	// Each global allocation begins a new aggregate before its typed description.
	for _, name := range []string{"encodeEGPRSModeStructEGPRSWindowSizeGroupGlobalTimeslotDescriptionGroupGlobalTimeslotDescription", "encodeGPRSModeStructGlobalTimeslotDescriptionGroupGlobalTimeslotDescription"} {
		fn := functions[name]
		if fn == nil || len(fn.Body.List) < 2 || render(fn.Body.List[0]) != "if err := w.ResetAggregate(sourceAggregate0); err != nil {\n\treturn err\n}" {
			return false
		}
	}
	const bitmap = "encodeTimeslotDescriptionStructMSTIMESLOTALLOCATIONChoiceMSTIMESLOTALLOCATIONMSTIMESLOTALLOCATION"
	fn := functions[bitmap]
	if fn == nil || !strings.Contains(render(fn.Type), "v uint8") || strings.Count(render(fn.Body), "w.RecordAggregate(sourceAggregate0, uint64(v))") != 1 {
		return false
	}
	// The alternate description has at most one presence record for each of
	// GAMMA_TN0..7; its selector dispatch contains no iteration or recursion.
	alpha := functions["encodeTimeslotDescriptionStructMSTIMESLOTALLOCATIONChoiceALPHA"]
	if alpha == nil {
		return false
	}
	producers := map[string]bool{bitmap: true}
	for i := 0; i < 8; i++ {
		suffix := fmt.Sprint(i)
		producer := "encodeTimeslotDescriptionStructMSTIMESLOTALLOCATIONChoiceALPHAGAMMATN" + suffix + "Value"
		selector := "encodeTimeslotDescriptionStructMSTIMESLOTALLOCATIONChoiceALPHAGAMMATN" + suffix + "Selector"
		pf, sf := functions[producer], functions[selector]
		if pf == nil || sf == nil || strings.Count(render(pf.Body), "w.RecordAggregate(sourceAggregate0, 1)") != 1 || strings.Count(render(sf.Body), producer+"(w, *v)") != 1 || strings.Count(render(alpha.Body), selector+"(w, v.GAMMATN"+suffix+")") != 1 {
			return false
		}
		producers[producer] = true
	}
	for name, fn := range functions {
		count := strings.Count(render(fn.Body), "w.RecordAggregate(sourceAggregate0,")
		if count > 0 && (!producers[name] || count != 1) {
			return false
		}
	}
	loops := false
	ast.Inspect(alpha.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			loops = true
		}
		return true
	})
	if loops {
		return false
	}
	// Pin the checked primitive contract before its emission loop.
	runtimeFile, err := parser.ParseFile(fset, "runtime.go", primitive, 0)
	if err != nil {
		return false
	}
	for _, decl := range runtimeFile.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "WriteBitString" || fn.Recv == nil {
			continue
		}
		if len(fn.Body.List) < 2 {
			return false
		}
		guard, ok := fn.Body.List[0].(*ast.IfStmt)
		if !ok || !strings.Contains(render(guard.Cond), "width != value.BitLength") || !strings.Contains(render(guard.Body), "return w.operationFailure(") || !strings.Contains(render(guard.Body), "InvalidValue") {
			return false
		}
		return true
	}
	return false
}
