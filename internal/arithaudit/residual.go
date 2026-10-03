package arithaudit

import (
	"fmt"
	"regexp"
	"strings"
)

var generatedWhitespace = regexp.MustCompile(`\s+`)

// checkGeneratedResidual verifies the remaining generator patterns against the
// guard and operation in each emitted function. This is a local pattern check,
// not a full control-flow proof. The allow-list cites the generating statements.
func (c *citationChecker) checkGeneratedResidual(f Finding) error {
	file, err := c.source(f.File)
	if err != nil {
		return err
	}
	fn := functionAt(file, f.Line)
	if fn == nil {
		return fmt.Errorf("generated finding outside function: %s", f.Key())
	}
	start, end := file.set.Position(fn.Pos()).Offset, file.set.Position(fn.End()).Offset
	body := generatedWhitespace.ReplaceAllString(string(file.source[start:end]), "")
	var ordered []string
	switch f.Expression {
	case "w.Position() - before":
		ordered = []string{"before:=w.Position()", "iferr:=encode", "ifw.Position()-before!=length"}
	case "r.Position() - 1", "uint64(runtime.LHBit('H', r.Position()-1))", "uint64(runtime.LHBit('L', r.Position()-1))":
		ordered = []string{"r.Eval(\"1\")", "r.ReadUint(width)", "iferr!=nil{return", "r.Position()-1"}
	case "uint64(runtime.LHBit('H', w.Position()))", "uint64(runtime.LHBit('L', w.Position()))":
		ordered = []string{"w.Eval(\"1\")", "runtime.LHBit(", "w.Position()"}
	case "(len(data) - 3) * 8", "len(data) - 3":
		ordered = []string{"runtime.CheckInput(data)", "iflen(data)<3", "len(data)-3"}
	case "40 + cm3.BitsConsumed":
		ordered = []string{"runtime.CheckInput(data)", "iflen(data)<6", "classmark.DecodeClassmark3ValuePart(data[5:])", "40+cm3.BitsConsumed"}
	case "40 + v.Classmark3.Wire.BitsConsumed":
		ordered = []string{"runtime.CheckInput(cm3)", "v.Classmark3.Wire.BitsConsumed>len(cm3)*8", "runtime.CheckInput(out)", "40+v.Classmark3.Wire.BitsConsumed"}
	case "r.Position() + r.Remaining()":
		ordered = []string{"decodeMSRACapabilityValuePartMSRACapabilityValuePartStruct(r)", "iferr!=nil{return", "knownEnd:=r.Position()+r.Remaining()"}
	case "knownEnd - r.Position()":
		ordered = []string{"knownEnd:=r.Position()+r.Remaining()", "ifknownEnd>400{knownEnd=400}", "ifknownEnd<r.Position(){knownEnd=r.Position()}", "r.PushLimit(knownEnd-r.Position())"}
	case "len(cm3) * 8":
		ordered = []string{"runtime.CheckInput(cm3)", "len(cm3)*8"}
	case "v.RevisionLevel << 5", "v.SSScreeningIndicator << 4":
		ordered = []string{"v.RevisionLevel>3", "v.SSScreeningIndicator>3", generatedWhitespace.ReplaceAllString(f.Expression, "")}
	case "^uint64(0) - 1":
		ordered = []string{"^uint64(0)-1"}
	case "v += 1":
		ordered = []string{"r.Eval(\"3\")", "r.ReadUint(width)", "v+=1"}
	case "uint64(v) - 1":
		ordered = []string{"ifuint64(v)<1", "raw:=uint64(v)-1"}
	default:
		return fmt.Errorf("unverified generated expression: %s", f.Key())
	}
	for _, part := range ordered {
		at := strings.Index(body, part)
		if at < 0 {
			return fmt.Errorf("generated guard %q absent for %s", part, f.Key())
		}
		body = body[at+len(part):]
	}
	return nil
}
