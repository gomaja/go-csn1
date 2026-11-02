package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const senderSequence = `package sample
func decodeSequence(r *runtime.Reader) (Value,error) {
 if err:=r.Enter("Value");err!=nil{return Value{},err}
 defer r.Leave()
 var v Value
 senderStart:=r.Position()
 senderVirtual:=r.VirtualBits()
 x0,err:=decodeField(r)
 if err!=nil{return Value{},err}
 v.Field=x0
 v.Wire.BitsConsumed=r.Position()-senderStart
 v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual
 return v,nil
}
func decodeField(r *runtime.Reader)(uint8,error){value,err:=r.ReadUint(3);return uint8(value),err}`

const continuationEncoder = `package sample
func encodeOctets(w *runtime.Writer,v []uint8)error {
 remaining:=w.RemainingLimit()
 if remaining<0 {
  if bits,received:=w.ReceivedBits();received {if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("invalid received continuation extent"))};remaining=bits-w.Position()} else if target,ok:=w.RemainingCanonicalTarget();ok {remaining=target}
 }
 if remaining<0 {return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("%w: continuation octets require a containing extent",runtime.ErrContextRequired))}
 oldBound,err:=w.PushLimit(remaining)
 if err!=nil{return err}
 if remaining%8!=0||len(v)!=remaining/8 {return w.LengthBoundError("continuation octets must fill the containing extent")}
 for _,item:=range v {if err:=encodeOctet(w,item);err!=nil{return err}}
 return w.PopLimit(oldBound)
}`

func TestSenderSequenceSpanProof(t *testing.T) {
	for _, expression := range []string{"r.Position() - senderStart", "r.VirtualBits() - senderVirtual"} {
		for _, tc := range []struct{ name, old, new string }{
			{name: "generated sequence"},
			{"reversed subtraction", "r.Position()-senderStart", "senderStart-r.Position()"},
			{"reversed virtual subtraction", "r.VirtualBits()-senderVirtual", "senderVirtual-r.VirtualBits()"},
			{"reassigned start", "v.Field=x0", "v.Field=x0;senderStart=0"},
			{"reassigned virtual start", "v.Field=x0", "v.Field=x0;senderVirtual=0"},
			{"unrelated initial position", "senderStart:=r.Position()", "senderStart:=other.Position()"},
			{"unrelated final position", "r.Position()-senderStart", "other.Position()-senderStart"},
			{"unrelated virtual position", "senderVirtual:=r.VirtualBits()", "senderVirtual:=other.VirtualBits()"},
			{"reader reassigned", "v.Field=x0", "v.Field=x0;r=other"},
			{"reader rewound", "v.Field=x0", "v.Field=x0;r.Commit(other)"},
			{"unchecked entry", `if err:=r.Enter("Value");err!=nil{return Value{},err}`, ``},
			{"rewinding child", "value,err:=r.ReadUint(3)", "r.Commit(other);value,err:=r.ReadUint(3)"},
			{"aliased child reader", "value,err:=r.ReadUint(3)", "alias:=r;alias.Commit(other);value,err:=r.ReadUint(3)"},
			{"unknown child", "decodeField(r)", "unknown(r)"},
			{"unchecked child", "if err!=nil{return Value{},err}", ""},
			{"start address escapes", "v.Field=x0", "v.Field=x0;escape(&senderStart)"},
			{"virtual address escapes", "v.Field=x0", "v.Field=x0;escape(&senderVirtual)"},
		} {
			t.Run(expression+"/"+tc.name, func(t *testing.T) {
				source := senderSequence
				if tc.old != "" {
					source = strings.Replace(source, tc.old, tc.new, 1)
				}
				err := residualSourceProof(t, source, expression)
				if (err == nil) != (tc.old == "") {
					t.Fatalf("proof: %v", err)
				}
			})
		}
	}
}

// senderEndingSequence models the emitted receiver-ending concatenation. The
// predicate receives only the just-decoded value, so it cannot rewind or replace
// the reader whose counters the two receipt pairs measure.
const senderEndingSequence = `package sample
func decodeSequence(r *runtime.Reader) (Value,error) {
 if err:=r.Enter("Value");err!=nil{return Value{},err}
 defer r.Leave()
 var v Value
 senderStart:=r.Position()
 senderVirtual:=r.VirtualBits()
 x0,err:=decodeField(r)
 if err!=nil{return Value{},err}
 v.Field=x0
 if fieldEnds(x0) {
  v.Wire.BitsConsumed=r.Position()-senderStart
  v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual
  return v,nil
 }
 v.Wire.BitsConsumed=r.Position()-senderStart
 v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual
 return v,nil
}
func decodeField(r *runtime.Reader)(uint8,error){value,err:=r.ReadUint(3);return uint8(value),err}
func fieldEnds(value uint8)bool{return value==0}`

func TestSenderEarlyReceiptSpanProof(t *testing.T) {
	for _, tc := range []struct{ name, old, new string }{
		{name: "generated early receipt"},
		{"reversed early subtraction", "r.Position()-senderStart", "senderStart-r.Position()"},
		{"reversed early virtual subtraction", "r.VirtualBits()-senderVirtual", "senderVirtual-r.VirtualBits()"},
		{"receipt after early return", "return v,nil\n }", "return v,nil\n v.Wire.BitsConsumed=r.Position()-senderStart\n }"},
		{"missing early return", "return v,nil\n }", "\n }"},
		{"missing early physical receipt", "v.Wire.BitsConsumed=r.Position()-senderStart\n  ", ""},
		{"missing early virtual receipt", "v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual\n  ", ""},
		{"reader mutation in early branch", "if fieldEnds(x0) {", "if fieldEnds(x0) {r.Commit(other)"},
		{"reader mutation before early branch", "v.Field=x0", "v.Field=x0;r.Commit(other)"},
		{"reader mutation after early branch", "\n }\n v.Wire.BitsConsumed", "\n }\n r.Commit(other)\n v.Wire.BitsConsumed"},
		{"rewinding child", "value,err:=r.ReadUint(3)", "r.Commit(other);value,err:=r.ReadUint(3)"},
		{"aliased child reader", "value,err:=r.ReadUint(3)", "alias:=r;alias.Commit(other);value,err:=r.ReadUint(3)"},
		{"unchecked child", "if err!=nil{return Value{},err}", ""},
		{"reassigned start", "v.Field=x0", "v.Field=x0;senderStart=0"},
		{"reassigned virtual start", "v.Field=x0", "v.Field=x0;senderVirtual=0"},
		{"reader passed to predicate", "fieldEnds(x0)", "fieldEnds(r)"},
		{"reader passed as extra predicate argument", "fieldEnds(x0)", "fieldEnds(x0,r)"},
		{"reader mutated in predicate init", "if fieldEnds(x0)", "if err:=r.Commit(other);fieldEnds(x0)"},
		{"predicate closure captures reader", "if fieldEnds(x0)", "if func()bool{r.Commit(other);return true}()"},
		{"predicate variable captures reader", "if fieldEnds(x0)", "predicate:=func(value uint8)bool{r.Commit(other);return true};if predicate(x0)"},
		{"predicate uses another value", "fieldEnds(x0)", "fieldEnds(v.Field)"},
		{"predicate with else branch", "\n }\n v.Wire.BitsConsumed", "\n }else{r.Commit(other)}\n v.Wire.BitsConsumed"},
		{"reversed final subtraction", "\n v.Wire.BitsConsumed=r.Position()-senderStart", "\n v.Wire.BitsConsumed=senderStart-r.Position()"},
		{"reversed final virtual subtraction", "\n v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual", "\n v.Wire.ImplicitZeros=senderVirtual-r.VirtualBits()"},
		{"early virtual receipt uses another reader", "r.VirtualBits()-senderVirtual", "other.VirtualBits()-senderVirtual"},
		{"early physical receipt uses another reader", "r.Position()-senderStart", "other.Position()-senderStart"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := senderEndingSequence
			if tc.old != "" {
				if !strings.Contains(source, tc.old) {
					t.Fatal("mutation did not apply")
				}
				source = strings.Replace(source, tc.old, tc.new, 1)
			}
			checkSenderReceiptSource(t, source, tc.old == "")
		})
	}
}

func TestSenderPredicateSideEffectsRefused(t *testing.T) {
	for _, replacement := range []string{
		`var current,stale *runtime.Reader
func fieldEnds(value uint8)bool{current.Commit(stale);return true}`,
		`var current,stale *runtime.Reader
func fieldEnds(value uint8)bool{return hidden(value)}
func hidden(value uint8)bool{current.Commit(stale);return true}`,
		`var current,stale *runtime.Reader
func fieldEnds(value uint8)bool{return hidden()}
func hidden()bool{current.Commit(stale);return true}`,
		`var current,stale *runtime.Reader
func fieldEnds(value uint8)bool{defer current.Commit(stale);return true}`,
		`var mutate func()bool
func fieldEnds(value uint8)bool{return mutate()}`,
	} {
		source := strings.Replace(senderEndingSequence, "func fieldEnds(value uint8)bool{return value==0}", replacement, 1)
		checkSenderReceiptSource(t, source, false)
	}
}

func TestSenderChildGlobalMutationRefused(t *testing.T) {
	for _, tc := range []struct{ name, operation, globals string }{
		{"direct reader global", "current.Commit(stale)", "var current,stale *runtime.Reader"},
		{"reader alias global", "current.Commit(stale)", "type ReaderAlias = runtime.Reader;var current,stale *ReaderAlias"},
		{"reader wrapper global", "holder.current.Commit(holder.stale)", "var holder struct{current,stale *runtime.Reader}"},
		{"reader interface global", "current.Commit(stale)", "var current interface{Commit(*runtime.Reader)};var stale *runtime.Reader"},
		{"reader hidden by helper", "hidden()", "var current,stale *runtime.Reader;func hidden(){current.Commit(stale)}"},
		{"reader hidden by callback", "hidden()", "var callback func();func hidden(){callback()}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(senderEndingSequence, "value,err:=r.ReadUint(3)", tc.operation+";value,err:=r.ReadUint(3)", 1) + "\n" + tc.globals + "\n"
			checkSenderReceiptSource(t, source, false)
		})
	}
}

func TestSenderFormattingCallbackSideEffectsRefused(t *testing.T) {
	for _, method := range []string{"GoString", "String", "Error"} {
		t.Run(method, func(t *testing.T) {
			source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport \"fmt\"", 1)
			source = strings.Replace(source, "value,err:=r.ReadUint(3)", `fmt.Sprintf("%#v",hiddenValue{});value,err:=r.ReadUint(3)`, 1)
			source += "\ntype hiddenValue struct{};var current,stale *runtime.Reader\nfunc(hiddenValue) " + method + "()string{current.Commit(stale);return \"\"}\n"
			checkSenderReceiptSource(t, source, false)
		})
	}
}

func TestSenderPackageSourceCompleteness(t *testing.T) {
	for _, tc := range []struct {
		name, extra  string
		removeSource bool
		want         bool
	}{
		{name: "declaration free documentation", extra: "package sample\n", want: true},
		{name: "hidden callback method", extra: "package sample\ntype Hidden struct{};func (Hidden) String()string{current.Commit(stale);return \"\"}\nvar current,stale *runtime.Reader"},
		{name: "builtin shadow", extra: "package sample\nfunc len(value any)int{current.Commit(stale);return 0}\nvar current,stale *runtime.Reader"},
		{name: "additional global reader", extra: "package sample\nvar current,stale *runtime.Reader"},
		{name: "cached source removed", removeSource: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			generated := filepath.Join(root, "generated.go")
			if err := os.WriteFile(generated, []byte(senderEndingSequence), 0600); err != nil {
				t.Fatal(err)
			}
			checker := &citationChecker{root: root, files: make(map[string]citationFile)}
			if _, err := checker.source("generated.go"); err != nil {
				t.Fatal(err)
			}
			if tc.extra != "" {
				if err := os.WriteFile(filepath.Join(root, "extra.go"), []byte(tc.extra), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.removeSource {
				if err := os.Remove(generated); err != nil {
					t.Fatal(err)
				}
			}
			for i, line := range strings.Split(senderEndingSequence, "\n") {
				if !strings.Contains(line, "v.Wire.BitsConsumed=") {
					continue
				}
				err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: i + 1, Expression: "r.Position() - senderStart"})
				if (err == nil) != tc.want {
					t.Errorf("line %d: proof=%v, want acceptance=%v", i+1, err, tc.want)
				}
			}
		})
	}
}

func TestSenderEarlyReceiptPathsProof(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"renamed predicate", strings.ReplaceAll(senderEndingSequence, "fieldEnds", "finished")},
		{"another checked child after early path", strings.Replace(senderEndingSequence, "\n }\n v.Wire.BitsConsumed", `
 }
 x1,err:=decodeField(r)
 if err!=nil{return Value{},err}
 v.Second=x1
 if fieldEnds(x1) {
  v.Wire.BitsConsumed=r.Position()-senderStart
  v.Wire.ImplicitZeros=r.VirtualBits()-senderVirtual
  return v,nil
 }
 v.Wire.BitsConsumed`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkSenderReceiptSource(t, tc.source, true)
		})
	}
}

func TestSenderReceiptFindingsUseTheirOwnPosition(t *testing.T) {
	source := senderEndingSequence
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	for i, line := range strings.Split(source, "\n") {
		if strings.Contains(line, "v.Wire.BitsConsumed=") {
			column := strings.Index(line, "r.Position()") + 1
			if err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: i + 1, Column: column, Expression: "r.Position() - senderStart"}); err != nil {
				t.Fatalf("own receipt position refused: %v", err)
			}
			// Neither the virtual receipt nor a nearby statement owns this line.
			for _, finding := range []Finding{
				{File: "generated.go", Line: i + 1, Expression: "r.VirtualBits() - senderVirtual"},
				{File: "generated.go", Line: i, Expression: "r.Position() - senderStart"},
				{File: "generated.go", Line: i + 1, Column: 1, Expression: "r.Position() - senderStart"},
			} {
				if err := checker.checkGeneratedResidual(finding); err == nil {
					t.Fatalf("unrelated receipt accepted: %+v", finding)
				}
			}
		}
	}
}

func checkSenderReceiptSource(t *testing.T, source string, want bool) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/gomaja/go-csn1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	receipts := 0
	for i, line := range strings.Split(source, "\n") {
		var expression string
		switch {
		case strings.Contains(line, "v.Wire.BitsConsumed="):
			expression = "r.Position() - senderStart"
		case strings.Contains(line, "v.Wire.ImplicitZeros="):
			expression = "r.VirtualBits() - senderVirtual"
		default:
			continue
		}
		receipts++
		err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: i + 1, Expression: expression})
		if (err == nil) != want {
			t.Errorf("receipt line %d (%s): proof=%v, want acceptance=%v", i+1, expression, err, want)
		}
	}
	if receipts == 0 {
		t.Fatal("fixture contains no receipts")
	}
}

func TestContinuationExtentDifferenceProof(t *testing.T) {
	for _, tc := range []struct{ name, old, new string }{
		{name: "generated continuation"},
		{"reversed subtraction", "bits-w.Position()", "w.Position()-bits"},
		{"unrelated position", "bits-w.Position()", "bits-other.Position()"},
		{"reassigned extent", "remaining=bits-w.Position()", "bits=1<<30;remaining=bits-w.Position()"},
		{"missing received guard", `if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("invalid received continuation extent"))};`, ``},
		{"reversed received guard", "w.Position()>bits", "w.Position()<bits"},
		{"missing upper bound", "bits>1048576||", ""},
		{"missing physical bound", "oldBound,err:=w.PushLimit(remaining)", "oldBound,err:=unbounded()"},
		{"unchecked physical bound", "if err!=nil{return err}", ""},
		{"missing restore", "return w.PopLimit(oldBound)", "return nil"},
		{"unbounded extent", "w.ReceivedBits()", "unbounded()"},
		{"unchecked received flag", ";received {", ";true {"},
		{"missing negative guard", `if remaining<0 {return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("%w: continuation octets require a containing extent",runtime.ErrContextRequired))}`, ``},
		{"reassigned writer", "remaining:=w.RemainingLimit()", "remaining:=w.RemainingLimit();w=other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := continuationEncoder
			if tc.old != "" {
				source = strings.Replace(source, tc.old, tc.new, 1)
			}
			err := residualSourceProof(t, source, "bits - w.Position()")
			if (err == nil) != (tc.old == "") {
				t.Fatalf("proof: %v", err)
			}
		})
	}
}

func residualSourceProof(t *testing.T, source, expression string) error {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/gomaja/go-csn1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	// Locate the source expression rather than borrowing another occurrence's proof.
	compact := strings.ReplaceAll(expression, " ", "")
	line := 0
	for i, text := range strings.Split(source, "\n") {
		if strings.Contains(strings.ReplaceAll(text, " ", ""), compact) {
			line = i + 1
			break
		}
	}
	if line == 0 {
		line = 2
	} // Reversed-source mutations must fail closed too.
	return checker.checkGeneratedResidual(Finding{File: "generated.go", Line: line, Expression: expression})
}

func TestContinuationLoopExtentProof(t *testing.T) {
	const source = `package sample
func decodeOctets(r *runtime.Reader)([]uint8,error){
 remaining:=r.BoundedRemaining()
 if remaining<0{return nil,err}
 if remaining%8!=0{return nil,err}
 var out []uint8
 for i:=0;i<remaining/8;i++ {out=append(out,0)}
 return out,nil
}`
	for _, tc := range []struct{ name, old, new string }{
		{name: "bounded remaining"},
		{"unbounded remaining", "r.BoundedRemaining()", "unbounded()"},
		{"reassigned before loop", "var out []uint8", "remaining=1<<30;var out []uint8"},
		{"reassigned in loop", "out=append(out,0)", "remaining=1<<30;out=append(out,0)"},
		{"reassigned after loop", "return out,nil", "remaining=1<<30;return out,nil"},
		{"address escapes", "var out []uint8", "escape(&remaining);var out []uint8"},
		{"unrelated source", "r.BoundedRemaining()", "other.BoundedRemaining()"},
		{"mixed assignment", "remaining:=r.BoundedRemaining()", "remaining,other:=unbounded(),r.BoundedRemaining()"},
		{"missing rejection", "if remaining<0{return nil,err}", ""},
		{"reassigned index", "out=append(out,0)", "i=-1;out=append(out,0)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := source
			if tc.old != "" {
				text = strings.Replace(text, tc.old, tc.new, 1)
			}
			increments, _, err := CheckGeneratedLoopArithmetic("generated.go", []byte(text))
			if (err == nil) != (tc.old == "") {
				t.Fatalf("increments=%d error=%v", increments, err)
			}
			if tc.old == "" && increments != 1 {
				t.Fatalf("increments=%d", increments)
			}
		})
	}
}

func TestRemainderExtentDifferenceProof(t *testing.T) {
	const source = `package sample
import runtime "github.com/gomaja/go-csn1/runtime"
func encodeOctets(w *runtime.Writer,v []uint8)error{
 remaining:=w.RemainingLimit()
 if target,present:=w.RemainingCanonicalTarget();present&&(remaining<0||target<remaining){remaining=target}
 if remaining<0{if bits,received:=w.ReceivedBits();received{if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("invalid received whole-octet remainder extent"))};remaining=bits-w.Position()}}
 if remaining<0{return fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired)}
 oldBound,err:=w.PushLimit(remaining/8*8)
 if err!=nil{return err}
 if len(v)!=remaining/8{return w.LengthBoundError("whole-octet remainder must fill every complete octet")}
 for _,element:=range v{if err:=encodeOctet(w,element);err!=nil{return err}}
 return w.PopLimit(oldBound)
}`
	for _, expression := range []string{"bits - w.Position()", "remaining / 8 * 8"} {
		for _, tc := range []struct{ name, old, new string }{
			{name: "generated remainder"},
			{"missing canonical minimum", `if target,present:=w.RemainingCanonicalTarget();present&&(remaining<0||target<remaining){remaining=target}`, ``},
			{"incorrect canonical minimum", `target<remaining`, `target>remaining`},
			{"missing canonical presence", `present&&(remaining<0||target<remaining)`, `remaining<0||target<remaining`},
			{"canonical discards containing limit", `remaining<0||target<remaining`, `true`},
			{"reversed subtraction", "bits-w.Position()", "w.Position()-bits"},
			{"unrelated position", "bits-w.Position()", "bits-other.Position()"},
			{"reassigned extent", "remaining=bits-w.Position()", "bits=1<<30;remaining=bits-w.Position()"},
			{"missing received guard", `if bits<0||bits>1048576||w.Position()<0||w.Position()>bits{return w.ComponentError(runtime.InvalidValue,"Octets",0,fmt.Errorf("invalid received whole-octet remainder extent"))};`, ``},
			{"reversed received guard", "w.Position()>bits", "w.Position()<bits"},
			{"missing upper bound", "bits>1048576||", ""},
			{"missing physical bound", "oldBound,err:=w.PushLimit(remaining/8*8)", "oldBound,err:=unbounded()"},
			{"unchecked physical bound", "if err!=nil{return err}", ""},
			{"missing restore", "return w.PopLimit(oldBound)", "return nil"},
			{"whole residual bound", "w.PushLimit(remaining/8*8)", "w.PushLimit(remaining)"},
			{"unbounded extent", "w.ReceivedBits()", "unbounded()"},
			{"unchecked received flag", ";received{", ";true{"},
			{"missing negative guard", `if remaining<0{return fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired)}`, ``},
			{"reassigned writer", "remaining:=w.RemainingLimit()", "remaining:=w.RemainingLimit();w=other"},
			{"unrelated body", `if len(v)!=remaining/8`, `escape(&bits);if len(v)!=remaining/8`},
		} {
			t.Run(expression+"/"+tc.name, func(t *testing.T) {
				text := source
				if tc.old != "" {
					if !strings.Contains(text, tc.old) {
						t.Fatal("mutation did not apply")
					}
					text = strings.Replace(text, tc.old, tc.new, 1)
				}
				err := residualSourceProof(t, text, expression)
				if (err == nil) != (tc.old == "") {
					t.Fatalf("proof: %v", err)
				}
			})
		}
	}
}

func TestUnguardedReceivedExtentOriginalsRefused(t *testing.T) {
	for _, body := range []string{
		`remaining:=w.RemainingLimit();if remaining<0{if bits,received:=w.ReceivedBits();received{remaining=bits-w.Position()}else if target,ok:=w.RemainingCanonicalTarget();ok{remaining=target}};if remaining<0{return fmt.Errorf("%w: continuation octets require a containing extent",runtime.ErrContextRequired)};if remaining%8!=0||len(v)!=remaining/8{return w.LengthBoundError("continuation octets must fill the containing extent")};for _,item:=range v{if err:=encodeOctet(w,item);err!=nil{return err}};return nil`,
		`remaining:=w.RemainingLimit();if remaining<0{if bits,received:=w.ReceivedBits();received{remaining=bits-w.Position()}};if remaining<0{return fmt.Errorf("%w: whole-octet remainder requires a containing extent",runtime.ErrContextRequired)};if len(v)!=remaining/8{return w.LengthBoundError("whole-octet remainder must fill every complete octet")};for _,element:=range v{if err:=encodeOctet(w,element);err!=nil{return err}};return nil`,
	} {
		source := "package sample\nfunc encodeOctets(w *runtime.Writer,v []uint8)error{" + body + "}"
		if err := residualSourceProof(t, source, "bits - w.Position()"); err == nil {
			t.Fatal("unguarded original accepted")
		}
	}
}
