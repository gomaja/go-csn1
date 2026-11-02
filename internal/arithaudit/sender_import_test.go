package arithaudit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const forkedSenderChild = `package child
import runtime "example.test/codec/runtime"
func Decode(r *runtime.Reader)(uint8,error){
 var failures runtime.ChoiceErrors
 defer failures.Release()
 candidate:=failures.ForkScoped(r)
 value,err:=decodeValue(candidate)
 if err!=nil{return 0,failures.Keep(r,err)}
 r.Commit(candidate)
 return value,nil
}
func decodeValue(r *runtime.Reader)(uint8,error){value,err:=r.ReadUint(3);return uint8(value),err}`

func TestSenderImportedReaderProof(t *testing.T) {
	for _, tc := range []struct {
		name, old, new string
		want           bool
	}{
		{name: "source resolved scoped fork", want: true},
		{"selected scoped reader alias", "r.Commit(candidate)", "var chosen *runtime.Reader;chosen=candidate;r.Commit(chosen)", true},
		{"grouped alias identities", "r.Commit(candidate)", "var chosen,unrelated *runtime.Reader;chosen=candidate;decodeValue(chosen);r.Commit(unrelated)", false},
		{"shadowed alias identities", "r.Commit(candidate)", "chosen:=candidate;{var chosen *runtime.Reader;_=chosen};r.Commit(chosen)", true},
		{"renamed choice state", "failures", "trials", true},
		{"foreign runtime choice implementation", "example.test/codec/runtime", "example.test/foreign/runtime", false},
		{"missing runtime import", `import runtime "example.test/codec/runtime"`, "", false},
		{"rewinding imported child", "value,err:=r.ReadUint(3)", "r.Commit(other);value,err:=r.ReadUint(3)", false},
		{"unrelated committed reader", "r.Commit(candidate)", "r.Commit(other)", false},
		{"root advanced after fork", "r.Commit(candidate)", "r.ReadUint(1);r.Commit(candidate)", false},
		{"commit before further operation", "r.Commit(candidate)", "r.Commit(candidate);r.ReadUint(1)", false},
		{"fork from unrelated reader", "failures.ForkScoped(r)", "failures.ForkScoped(other)", false},
		{"candidate reassigned", "value,err:=decodeValue(candidate)", "candidate=other;value,err:=decodeValue(candidate)", false},
		{"reader address escapes", "value,err:=r.ReadUint(3)", "escape(&r);value,err:=r.ReadUint(3)", false},
		{"reader captured by escaping closure", "value,err:=r.ReadUint(3)", "escape(func(){r.ReadUint(1)});value,err:=r.ReadUint(3)", false},
		{"concurrent reader mutation", "value,err:=r.ReadUint(3)", "go decodeValue(r);value,err:=r.ReadUint(3)", false},
		{"unknown decoder", "decodeValue(candidate)", "unknown(candidate)", false},
		{"foreign fork receiver", "failures.ForkScoped(r)", "other.ForkScoped(r)", false},
		{"missing imported callee", "func Decode(", "func Other(", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			child := forkedSenderChild
			if tc.old != "" {
				if !strings.Contains(child, tc.old) {
					t.Fatal("mutation did not apply")
				}
				if tc.name == "renamed choice state" {
					child = strings.ReplaceAll(child, tc.old, tc.new)
				} else {
					child = strings.Replace(child, tc.old, tc.new, 1)
				}
			}
			source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport dep \"example.test/codec/child\"", 1)
			source = strings.Replace(source, "value,err:=r.ReadUint(3);return uint8(value),err", "return dep.Decode(r)", 1)
			checkImportedSenderSource(t, source, child, tc.want)
		})
	}
}

func TestSenderUnsafeAddressAliasesRefused(t *testing.T) {
	source := strings.Replace(senderEndingSequence, "package sample", `package sample
import runtime "example.test/codec/runtime"
import "unsafe"
type UPointer = unsafe.Pointer
type UAddress = uintptr
type RPointer = *runtime.Reader
var currentAddress uint64=0
var staleAddress uint64=0`, 1)
	source = strings.Replace(source, "value,err:=r.ReadUint(3)", "var p,q *runtime.Reader;p=RPointer(UPointer(UAddress(currentAddress)));q=RPointer(UPointer(UAddress(staleAddress)));p.Commit(q);value,err:=r.ReadUint(3)", 1)
	checkImportedSenderSource(t, source, forkedSenderChild, false)
}

func TestSenderImportedAddressAliasesRefused(t *testing.T) {
	source := strings.Replace(senderEndingSequence, "package sample", `package sample
import runtime "example.test/codec/runtime"
import dep "example.test/codec/child"
type UPointer = dep.UPointer
type UAddress = uintptr
type RPointer = *runtime.Reader
var currentAddress uint64=0
var staleAddress uint64=0`, 1)
	source = strings.Replace(source, "value,err:=r.ReadUint(3)", "var p,q *runtime.Reader;p=RPointer(UPointer(UAddress(currentAddress)));q=RPointer(UPointer(UAddress(staleAddress)));p.Commit(q);value,err:=r.ReadUint(3)", 1)
	checkImportedSenderSource(t, source, `package child
import "unsafe"
type UPointer = unsafe.Pointer`, false)
}

func TestSenderMethodValueCallbacksRefused(t *testing.T) {
	const child = `package child
import runtime "example.test/codec/runtime"
var Current,Stale *runtime.Reader
type Trigger struct{}
func(Trigger)normalize(err error,operation,path string,offset,width int,source runtime.Source,kind runtime.ErrorKind)error{Current.Commit(Stale);return err}
func Decode(r *runtime.Reader)(uint8,error){var failures runtime.ChoiceErrors;defer failures.Release();candidate:=failures.ForkScoped(r);value,err:=failed(candidate);failures.Keep(r,err);return value,nil}
func failed(r *runtime.Reader)(uint8,error){value,_:=r.ReadUint(3);err:=r.Error(runtime.InvalidValue,"synthetic");var trigger Trigger;deferred,_:=runtime.DeferOperation(err,"decode","x",0,0,runtime.Source{},runtime.InvalidValue,trigger.normalize);return uint8(value),deferred}`
	source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport dep \"example.test/codec/child\"", 1)
	source = strings.Replace(source, "value,err:=r.ReadUint(3);return uint8(value),err", "return dep.Decode(r)", 1)
	for _, tc := range []struct{ name, prefix, callback string }{
		{"direct method value", "", "trigger.normalize"},
		{"local function variable", "callback:=trigger.normalize;", "callback"},
		{"aggregate callback field", "holder:=struct{callback func(error,string,string,int,int,runtime.Source,runtime.ErrorKind)error}{trigger.normalize};", "holder.callback"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(child, "deferred,_:=runtime.DeferOperation", tc.prefix+"deferred,_:=runtime.DeferOperation", 1)
			changed = strings.Replace(changed, ",trigger.normalize);", ","+tc.callback+");", 1)
			checkImportedSenderSource(t, source, changed, false)
		})
	}
}

func TestSenderCompiledWidthExpressionProof(t *testing.T) {
	const compiled = `width,err:=r.ExpressionResult(runtime.ExpressionConstant(8).Operand().Multiply(r.ExpressionVariable("COUNT").Operand().Operand().Operand()));if err!=nil{return 0,err};value,err:=r.ReadUint(width);return uint8(value),err`
	for _, tc := range []struct {
		name, old, new string
		want           bool
	}{
		{name: "compiled expression", want: true},
		{"nonliteral constant", "runtime.ExpressionConstant(8)", "runtime.ExpressionConstant(hidden())", false},
		{"unproved expression origin", "runtime.ExpressionConstant(8)", "unproved()", false},
		{"unknown fluent method", ".Multiply(", ".Rewind(", false},
		{"operand arguments", ".Operand()", ".Operand(r)", false},
		{"variable address escapes", `r.ExpressionVariable("COUNT")`, `r.ExpressionVariable(escape(&r))`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport runtime \"example.test/codec/runtime\"", 1)
			body := compiled
			if tc.old != "" {
				if !strings.Contains(body, tc.old) {
					t.Fatal("mutation did not apply")
				}
				body = strings.Replace(body, tc.old, tc.new, 1)
			}
			source = strings.Replace(source, "value,err:=r.ReadUint(3);return uint8(value),err", body, 1)
			checkImportedSenderSource(t, source, forkedSenderChild, tc.want)
		})
	}
	source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport runtime \"example.test/codec/runtime\"", 1)
	source = strings.Replace(source, "(uint8,error){value,err:=r.ReadUint(3);return uint8(value),err}", "(runtime.BitString,error){value,err:=r.ReadBitString(8);return runtime.BitString(value),err}", 1)
	checkImportedSenderSource(t, source, forkedSenderChild, true)
}

func TestSenderDeferredIndexNormalizerProof(t *testing.T) {
	const normalizer = `func locateIndex(err error,path string,index int)error{
 if deferred,ok:=runtime.DeferIndex(err,path,index,locateIndex);ok{return deferred}
 path=strings.TrimPrefix(path,strconv.Itoa(index))
 return err
}`
	source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport runtime \"example.test/codec/runtime\"\nimport \"strconv\"\nimport \"strings\"", 1)
	source = strings.Replace(source, "value,err:=r.ReadUint(3);return uint8(value),err", `value,err:=r.ReadUint(3);if err!=nil{return 0,locateIndex(err,"Field",0)};return uint8(value),nil`, 1)
	for _, tc := range []struct {
		name, old, new string
		want           bool
	}{
		{name: "named index normalizer", want: true},
		{"method normalizer", "runtime.DeferIndex(err,path,index,locateIndex)", "runtime.DeferIndex(err,path,index,trigger.normalize)", false},
		{"normalizer variable", "runtime.DeferIndex(err,path,index,locateIndex)", "runtime.DeferIndex(err,path,index,callback)", false},
		{"aggregate normalizer", "runtime.DeferIndex(err,path,index,locateIndex)", "runtime.DeferIndex(err,path,index,holder.callback)", false},
		{"mutating named normalizer", "path=strings.TrimPrefix", "current.Commit(stale);path=strings.TrimPrefix", false},
		{"foreign integer formatter", "strconv.Itoa(index)", "foreign.Itoa(index)", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := normalizer
			if tc.old != "" {
				if !strings.Contains(body, tc.old) {
					t.Fatal("mutation did not apply")
				}
				body = strings.Replace(body, tc.old, tc.new, 1)
			}
			checkImportedSenderSource(t, source+"\n"+body, forkedSenderChild, tc.want)
		})
	}
}

func TestSenderImportResolutionFailsClosed(t *testing.T) {
	source := strings.Replace(senderEndingSequence, "package sample", "package sample\nimport dep \"example.test/codec/child\"", 1)
	source = strings.Replace(source, "value,err:=r.ReadUint(3);return uint8(value),err", "return dep.Decode(r)", 1)
	for _, tc := range []struct{ name, old, new string }{
		{"missing source", "example.test/codec/child", "example.test/codec/missing"},
		{"foreign dependency", "example.test/codec/child", "example.test/foreign/child"},
		{"missing import", `import dep "example.test/codec/child"`, ""},
		{"shadowed package qualifier", "return dep.Decode(r)", "dep:=other;return dep.Decode(r)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(source, tc.old) {
				t.Fatal("mutation did not apply")
			}
			checkImportedSenderSource(t, strings.Replace(source, tc.old, tc.new, 1), forkedSenderChild, false)
		})
	}
}

func checkImportedSenderSource(t *testing.T, source, child string, want bool) {
	t.Helper()
	root := t.TempDir()
	for path, contents := range map[string]string{"go.mod": "module example.test/codec\n", "generated.go": source, "child/generated.go": child} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	for i, line := range strings.Split(source, "\n") {
		expression := ""
		if strings.Contains(line, "v.Wire.BitsConsumed=") {
			expression = "r.Position() - senderStart"
		}
		if strings.Contains(line, "v.Wire.ImplicitZeros=") {
			expression = "r.VirtualBits() - senderVirtual"
		}
		if expression == "" {
			continue
		}
		err := checker.checkGeneratedResidual(Finding{File: "generated.go", Line: i + 1, Expression: expression})
		if (err == nil) != want {
			t.Errorf("line %d: proof=%v, want acceptance=%v", i+1, err, want)
		}
	}
}
