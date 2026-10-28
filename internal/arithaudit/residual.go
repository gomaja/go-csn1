package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
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
	if fn.Name.Name == "DecodeClassmark2ValuePartFrom" && (f.Expression == "origin + offset" || f.Expression == "uint64(value)") {
		observation := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			literal, ok := node.(*ast.FuncLit)
			if !ok || file.set.Position(literal.Pos()).Line > f.Line || file.set.Position(literal.End()).Line < f.Line {
				return true
			}
			start, end := file.set.Position(literal.Pos()).Offset, file.set.Position(literal.End()).Offset
			text := strings.ReplaceAll(generatedWhitespace.ReplaceAllString(string(file.source[start:end]), ""), ";", "")
			observation = text == "func(fieldstring,offset,widthint,valueuint8)error{returnr.Observe(origin+offset,width,uint64(value),source,field)}"
			return true
		})
		if observation {
			for _, guard := range []string{"r.Check()", "r.Remaining()<24", "start:=r.Fork()", "origin:=start.Position()"} {
				if !strings.Contains(body, guard) {
					return fmt.Errorf("manual observation origin guard absent: %s", f.Key())
				}
			}
			calls := 0
			var callErr error
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				id, ok := call.Fun.(*ast.Ident)
				if !ok || id.Name != "observe" {
					return true
				}
				calls++
				if len(call.Args) != 4 {
					callErr = fmt.Errorf("manual observation arguments differ")
					return true
				}
				offset, ok := call.Args[1].(*ast.BasicLit)
				if !ok {
					callErr = fmt.Errorf("manual observation offset is not fixed")
					return true
				}
				width, ok := call.Args[2].(*ast.BasicLit)
				if !ok {
					callErr = fmt.Errorf("manual observation width is not fixed")
					return true
				}
				at, err := strconv.Atoi(offset.Value)
				if err != nil || at < 0 || at > 24 {
					callErr = fmt.Errorf("manual observation offset outside received value")
					return true
				}
				bits, err := strconv.Atoi(width.Value)
				if err != nil || bits < 1 || bits > 24-at {
					callErr = fmt.Errorf("manual observation width outside received value")
				}
				return true
			})
			if callErr != nil {
				return fmt.Errorf("%s: %w", f.Key(), callErr)
			}
			if calls != 3 {
				return fmt.Errorf("manual observation calls differ: %s", f.Key())
			}
			return nil
		}
		// Existing writer conversions in this function retain their own proof below.
		if f.Expression == "origin + offset" {
			return fmt.Errorf("manual observation helper differs: %s", f.Key())
		}
	}
	// These new operations are emitted only in fixed descriptor and location
	// helpers. Check the complete local guard shape before accepting their
	// classified arithmetic; other generated functions still use the rules below.
	if guards, ok := reviewedHelperGuards[fn.Name.Name]; ok && reviewedHelperExpressions[fn.Name.Name][f.Expression] {
		for _, guard := range guards {
			if !strings.Contains(body, guard) {
				return fmt.Errorf("generated helper guard %q absent for %s", guard, f.Key())
			}
		}
		return nil
	}
	if fn.Name.Name == "indexedPath" && (f.Expression == "start + len(component)" || f.Expression == "end + 1") {
		// Cut advances through disjoint components of full. The separator
		// check precedes end+1, so both offsets stay within len(full).
		// Pin the complete loop, including its initial state, to reject
		// shadowed variables and extra writes to the running offsets.
		const prefix = `funcindexedPath(full,pathstring,indexint)string{parent:=indexParent(path)start:=0forrest:=fullrest!=""{component,tail,more:=strings.Cut(rest,"/")end:=start+len(component)ifcomponent==parent{ifparent==path{returnfull[:end]+"["+strconv.Itoa(index)+"]"+full[end:]}next,_,_:=strings.Cut(tail,"/")ifnext==path{returnfull[:end]+"/"+path+"["+strconv.Itoa(index)+"]"+strings.TrimPrefix(tail,path)}returnfull[:end]+"/"+path+"["+strconv.Itoa(index)+"]"+full[end:]}if!more{break}start=end+1rest=tail}`
		if !strings.HasPrefix(strings.ReplaceAll(body, ";", ""), prefix) {
			return fmt.Errorf("generated indexed path loop changed: %s", f.Key())
		}
		return nil
	}
	if fn.Recv != nil && strings.HasPrefix(fn.Name.Name, "Set") {
		return checkGeneratedCountSetter(fn, f.Expression)
	}
	var ordered []string
	switch f.Expression {
	case "uint64(value)", "uint8(value)":
		// The Classmark 2 helpers pin an unsigned byte writer parameter
		// and a reader width of at most eight bits on every target.
		want := "func(valueuint8,widthint){ifwriteErr==nil{writeErr=w.WriteUint(uint64(value),width)}}"
		if f.Expression == "uint8(value)" {
			want = "func(widthint)uint8{ifwidth<0||width>8{readErr=fmt.Errorf(\"invalidClassmark2readwidth\");return0};value,err:=r.ReadUint(width);iferr!=nil{readErr=err};returnuint8(value)}"
			if fn.Name.Name == "DecodeClassmark2ValuePartFrom" {
				want = "func(widthint)uint8{value,err:=r.ReadUint(width);iferr!=nil{readErr=err};returnuint8(value)}"
				calls := regexp.MustCompile(`read\(([0-9]+)\)`).FindAllStringSubmatch(body, -1)
				if len(calls) == 0 {
					return fmt.Errorf("classmark 2 reader has no fixed widths: %s", f.Key())
				}
				for _, call := range calls {
					if len(call[1]) > 1 || call[1] == "0" || call[1][0] > '8' {
						return fmt.Errorf("classmark 2 reader width exceeds eight: %s", f.Key())
					}
				}
			}
		}
		proven := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			literal, ok := node.(*ast.FuncLit)
			if !ok || file.set.Position(literal.Pos()).Line > f.Line || file.set.Position(literal.End()).Line < f.Line {
				return true
			}
			start, end := file.set.Position(literal.Pos()).Offset, file.set.Position(literal.End()).Offset
			// Pin the complete helper body so an inner shadow or changed
			// source of value cannot inherit this proof.
			proven = strings.ReplaceAll(generatedWhitespace.ReplaceAllString(string(file.source[start:end]), ""), ";", "") == strings.ReplaceAll(want, ";", "")
			return true
		})
		if !proven {
			return fmt.Errorf("generated byte conversion lacks the bounded helper: %s", f.Key())
		}
		return nil
	case "w.Position() - before":
		ordered = []string{"before:=w.Position()", "iferr:=encode", "ifw.Position()-before!=length"}
	case "r.Position() - 1", "uint64(runtime.LHBit('H', r.Position()-1))", "uint64(runtime.LHBit('L', r.Position()-1))":
		ordered = []string{"r.Eval(\"1\")", "r.ReadUint(width)", "iferr!=nil{return", "r.Position()-1"}
		if directLiteralWidth(fn, "1") {
			ordered[0] = "width:=1"
		}
	case "uint64(runtime.LHBit('H', w.Position()))", "uint64(runtime.LHBit('L', w.Position()))":
		ordered = []string{"w.Eval(\"1\")", "runtime.LHBit(", "w.Position()"}
		if directLiteralWidth(fn, "1") {
			ordered[0] = "width:=1"
		}
	case "(len(data) - 3) * 8", "len(data) - 3":
		ordered = []string{"runtime.CheckInput(data)", "iflen(data)<3", "len(data)-3"}
	case "40 + cm3.BitsConsumed":
		ordered = []string{"runtime.CheckInput(data)", "iflen(data)<6", "classmark.DecodeClassmark3ValuePart(data[5:])", "40+cm3.BitsConsumed"}
	case "40 + v.Classmark3.Wire.BitsConsumed":
		ordered = []string{"runtime.CheckInput(cm3)", "v.Classmark3.Wire.BitsConsumed>len(cm3)*8", "runtime.CheckInput(out)", "40+v.Classmark3.Wire.BitsConsumed"}
	case "r.Position() + r.Remaining()":
		ordered = []string{"decodeMSRACapabilityValuePartMSRACapabilityValuePartStruct(r)", "iferr!=nil{return", "knownEnd:=r.Position()+r.Remaining()"}
	case "knownEnd - valueStart":
		ordered = []string{"valueStart:=r.Position()", "decodeMSRACapabilityValuePartMSRACapabilityValuePartStruct(r)", "iferr!=nil{return", "knownEnd:=r.Position()+r.Remaining()", "ifknownEnd-valueStart>400"}
	case "valueStart + 400":
		ordered = []string{"valueStart:=r.Position()", "knownEnd:=r.Position()+r.Remaining()", "ifknownEnd-valueStart>400{knownEnd=valueStart+400}"}
	case "knownEnd - r.Position()":
		ordered = []string{"knownEnd:=r.Position()+r.Remaining()", "ifknownEnd-valueStart>400{knownEnd=valueStart+400}", "ifknownEnd<r.Position(){knownEnd=r.Position()}", "r.PushLimit(knownEnd-r.Position())"}
	case "len(cm3) * 8":
		ordered = []string{"runtime.CheckInput(cm3)", "len(cm3)*8"}
	case "v.RevisionLevel << 5", "v.SSScreeningIndicator << 4":
		ordered = []string{"v.RevisionLevel>3", "v.SSScreeningIndicator>3", generatedWhitespace.ReplaceAllString(f.Expression, "")}
	case "^uint64(0) - 1":
		ordered = []string{"^uint64(0)-1"}
	case "v += 1":
		ordered = []string{"r.Eval(\"3\")", "r.ReadUint(width)", "v+=1"}
		if directLiteralWidth(fn, "3") {
			ordered[0] = "width:=3"
		}
	case "uint64(v) - 1":
		ordered = []string{"ifuint64(v)<1", "raw:=uint64(v)-1"}
	case "len(out.Entries) - 1":
		ordered = []string{"for{before:=r.Position()", "out.Entries=append(out.Entries,element)", "continuation,err:=r.ReadUint(1)", "iferr!=nil{return", "len(out.Entries)-1"}
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

// directLiteralWidth accepts a single literal declaration and rejects later
// assignments or increments to the identifier, including nested blocks.
func directLiteralWidth(fn *ast.FuncDecl, want string) bool {
	found, valid := false, true
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if !valid {
			return false
		}
		switch statement := node.(type) {
		case *ast.AssignStmt:
			for _, lhs := range statement.Lhs {
				if !isName(lhs, "width") {
					continue
				}
				if found || statement.Tok != token.DEFINE || len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
					valid = false
					break
				}
				literal, ok := statement.Rhs[0].(*ast.BasicLit)
				valid = ok && literal.Kind == token.INT && literal.Value == want
				found = true
			}
		case *ast.IncDecStmt:
			if isName(statement.X, "width") {
				valid = false
			}
		}
		return valid
	})
	return found && valid
}

var reviewedHelperExpressions = map[string]map[string]bool{
	"emrBitmapFields":                {"index + 1": true},
	"descriptorRead":                 {"r.Position() + r.Remaining()": true, "r.Position() + maxBits": true, "r.Position() - start": true},
	"constrainedWidth":               {"len(path) - 1": true, "i--": true, "i + 1": true},
	"locatedFailure":                 {"be.Limit - be.Position": true, `"/" + be.Field`: true, `location.Path += "/" + be.Field`: true},
	"shiftOperation":                 {"copy.Location.BitOffset += offset": true, "copy.Offset += offset": true, "copy.Position += offset": true, "copy.Limit += offset": true, `path + "/" + copy.Location.Path`: true, `path + "/" + copy.Path`: true},
	"DecodeClassmark2ValuePartFrom":  {"r.Position() + r.Remaining()": true},
	"DecodeGERANCSFrom":              {"r.Position() + r.Remaining()": true},
	"DecodeGERANPSFrom":              {"r.Position() + r.Remaining()": true},
	"EncodeClassmark2ValuePartTo":    {"spareIndex++": true},
	"EncodeGERANCSCanonicalAtLength": {"octets - 5": true},
	"decodeClassmark2ValuePartBytes": {"len(data) * 8": true},
}

var reviewedHelperGuards = map[string][]string{
	"emrBitmapFields":                {"ifv.REPORTINGQUANTITYList!=nil", "last:=0", "forindex,report:=range*v.REPORTINGQUANTITYList", "ifreport!=nil{last=index+1}", "entries:=(*v.REPORTINGQUANTITYList)[:last]"},
	"descriptorRead":                 {"r.Check()", "r.Remaining()<minBits", "start:=r.Position()", "err:=decode(r)", "ifmaxBits>0&&r.Remaining()>maxBits"},
	"constrainedWidth":               {"fori:=len(path)-1;i>=0;i--", "path=path[i+1:]"},
	"locatedFailure":                 {"be.Limit>=0&&be.Limit<=1<<20&&be.Position>=0&&be.Position<=1<<20", "location.Width=be.Limit-be.Position"},
	"shiftOperation":                 {"ifcopy.Location.BitOffset>=0", "copy.Cause=shiftOperation(copy.Cause,offset,path)"},
	"DecodeClassmark2ValuePartFrom":  {"r.Check()", "r.Remaining()<24", "r.Position()+r.Remaining()"},
	"DecodeGERANCSFrom":              {"r.Check()", "r.BoundedRemaining()<0", "r.BoundedRemaining()<48", "r.Position()+r.Remaining()"},
	"DecodeGERANPSFrom":              {"r.Check()", "r.BoundedRemaining()<0", "r.BoundedRemaining()<16", "r.Position()+r.Remaining()"},
	"EncodeClassmark2ValuePartTo":    {"spareIndex:=0", "ifspareIndex<len(v.Wire.Spare)", "spareIndex++"},
	"EncodeGERANCSCanonicalAtLength": {"octets<6||octets>37", "octets-5"},
	"decodeClassmark2ValuePartBytes": {"runtime.CheckInput(data)", "len(data)<3", "len(data)*8"},
}
