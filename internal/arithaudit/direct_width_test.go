package arithaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectWidthNarrowingProof(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		ok   bool
	}{
		{"literal", `width:=8;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{"too wide", `width:=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"widened", `width:=8;width=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"unknown", `width:=8;width=n;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"later write preserves read", `width:=8;v,_:=r.ReadUint(width);width=9;return uint8(v)`, true},
		{"offset", `width:=3;v,_:=r.ReadUint(width);v+=1;return uint8(v)`, true},
		{"plain reassignment", `width:=9;width=8;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{"eval declaration", `width,err:=r.Eval("8");_ = err;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{"eval reassignment", `width:=9;var err error;width,err=r.Eval("8");_ = err;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{"unrelated eval preserves proof", `width:=8;x,_:=r.Eval("9");_ = x;v,_:=r.ReadUint(width);return uint8(v)`, true},
		{"compound assignment", `width:=8;width+=1;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"increment", `width:=8;width++;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"decrement", `width:=0;width--;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"second lhs", `width:=8;var x int;x,width=1,9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"second eval lhs", `width:=8;x,width:=r.Eval("8");_ = x;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"literal multi lhs", `width,x:=8,9;_ = x;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"literal multi lhs single rhs", `width,_:=8;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"address escapes", `width:=8;p:=&width;*p=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"parenthesized address", `width:=8;p:=&(width);*p=9;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"for later iteration", `width:=8;for j:=0;j<2;j++{v,_:=r.ReadUint(width);_ = uint8(v);width=9};return 0`, false},
		{"range later iteration", `width:=8;for range []int{0,1}{v,_:=r.ReadUint(width);_ = uint8(v);width=9};return 0`, false},
		{"for write before read", `width:=8;for j:=0;j<2;j++{width=8;v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, false},
		{"range write before read", `width:=8;for range []int{0,1}{width=8;v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, false},
		{"for unchanged width", `width:=8;for j:=0;j<2;j++{v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, true},
		{"range unchanged width", `width:=8;for range []int{0,1}{v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, true},
		{"unrelated eval", `width:=9;x,_:=r.Eval("8");_ = x;v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"discarded eval", `width:=9;r.Eval("8");v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"wrong receiver", `width,_:=x.Eval("8");v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"unknown eval", `width,_:=r.Eval("n");v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"range key assign", `width:=8;for width=range []int{0,1}{};v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"range value assign", `width:=8;for _,width=range []int{9}{};v,_:=r.ReadUint(width);return uint8(v)`, false},
		{"range key declaration", `width:=8;for width:=range []int{0,1}{v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, false},
		{"range value declaration", `width:=8;for _,width:=range []int{9}{v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, false},
		{"var shadow", `width:=8;{var width=9;v,_:=r.ReadUint(width);_ = uint8(v)};return 0`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := CheckGeneratedNarrowing("sample.go", []byte("package sample;func decode(r reader,n int) uint8{"+tc.body+"}"))
			if (err == nil) != tc.ok || tc.ok && n != 1 {
				t.Fatalf("checked=%d err=%v, want accepted=%t: %s", n, err, tc.ok, tc.body)
			}
		})
	}
}

func TestDirectProofsRejectCompoundOperators(t *testing.T) {
	for _, operator := range []string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "|=", "^=", "&^="} {
		t.Run(operator, func(t *testing.T) {
			width := fmt.Appendf(nil, "package sample;func decode(r reader) uint8{width:=8;width%s1;v,_:=r.ReadUint(width);return uint8(v)}", operator)
			if _, err := CheckGeneratedNarrowing("sample.go", width); err == nil {
				t.Fatal("compound width assignment accepted")
			}
			count := fmt.Appendf(nil, "package sample;func decode(){count:=1048576;count%s1;for i:=0;i<count;i++{}}", operator)
			if _, _, err := CheckGeneratedLoopArithmetic("sample.go", count); err == nil {
				t.Fatal("compound count assignment accepted")
			}
		})
	}
}

func TestDirectCountWritesFailClosed(t *testing.T) {
	loop := `;for i:=0;i<count;i++{}`
	for _, tc := range []struct {
		name string
		body string
		ok   bool
	}{
		{"compound assignment", `count:=1048576;count*=1048576` + loop, false},
		{"increment", `count:=1048576;count++` + loop, false},
		{"decrement", `count:=0;count--` + loop, false},
		{"second lhs", `count:=1;var x int;x,count=0,n;_ = x` + loop, false},
		{"second call lhs", `count:=1;x,count:=r.Eval("1");_ = x` + loop, false},
		{"literal multi lhs", `count,x:=1,2;_ = x` + loop, false},
		{"literal multi lhs single rhs", `count,_:=1` + loop, false},
		{"later outer iteration", `count:=1;for o:=0;o<2;o++{for i:=0;i<count;i++{};count=n}`, false},
		{"later write", `count:=1` + loop + `;count=n`, false},
		{"address escapes", `count:=1;p:=&count;*p=n` + loop, false},
		{"parenthesized address", `count:=1;p:=&(count);*p=n` + loop, false},
		{"range key assign", `count:=1;for count=range []int{1}{}` + loop, false},
		{"range value assign", `count:=1;for _,count=range []int{n}{}` + loop, false},
		{"range key declaration", `count:=1;for count:=range []int{n}{for i:=0;i<count;i++{}}`, false},
		{"range value declaration", `count:=1;for _,count:=range []int{n}{for i:=0;i<count;i++{}}`, false},
		{"var shadow", `count:=1;{var count=n;for i:=0;i<count;i++{}}`, false},
		{"unknown eval receiver", `count,_:=x.Eval("1")` + loop, false},
		{"wrong eval arity", `count,_:=r.Eval()` + loop, false},
		{"wrong spare fill arity", `count:=w.SpareFillCount(n)` + loop, false},
		{"no preceding bounded write", `for i:=0;i<count;i++{};count:=1`, false},
		{"plain reassignment", `count:=1048576;count=2` + loop, true},
		{"later bounded write", `count:=1` + loop + `;count=2`, true},
		{"bounded outer iteration", `count:=1;for o:=0;o<2;o++{for i:=0;i<count;i++{};count=2}`, true},
		{"eval", `count,err:=r.Eval("2");_ = err` + loop, true},
		{"spare count", `count,recorded:=w.SpareCount("spare");_ = recorded` + loop, true},
		{"spare fill count", `count:=w.SpareFillCount()` + loop, true},
		{"expression result", `count,err:=r.ExpressionResult(r.ExpressionVariable("n").Operand());_ = err` + loop, true},
		{"bounded remaining", `remaining:=r.BoundedRemaining();count:=remaining/8` + loop, true},
		{"choice counter without loop", `count:=0;count++;_ = count`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := CheckGeneratedLoopArithmetic("sample.go", []byte("package sample;func decode(r reader,w writer,n int){"+tc.body+"}"))
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v, want accepted=%t: %s", err, tc.ok, tc.body)
			}
		})
	}
}

func TestDirectCountLoopProof(t *testing.T) {
	for _, tc := range []struct {
		count string
		ok    bool
	}{
		{`count:=2`, true},
		{`count:=1048576`, true},
		{`count:=1048577`, false},
		{`count:=-1`, false},
		{`count:=n`, false},
		{`count,err:=r.ExpressionResult(r.ExpressionVariable("n").Operand());if err!=nil{return}`, true},
		{`count,err:=r.ExpressionResult(r.ExpressionVariable("n").Operand());if err!=nil{return};count=n`, false},
	} {
		_, _, err := CheckGeneratedLoopArithmetic("sample.go", []byte("package sample;func decode(r reader,n int){"+tc.count+";for i:=0;i<count;i++{}}"))
		if (err == nil) != tc.ok {
			t.Fatalf("%s: %v", tc.count, err)
		}
	}
}

func TestDirectLHWidthProofRejectsLongerLiterals(t *testing.T) {
	for _, width := range []string{"1", "10", "11", "16", "1+1", "1*16", "1; width=2", "1; width++"} {
		source := fmt.Appendf(nil, "package sample;func encodeField(w *runtime.Writer){width:=%s;_ = width;_ = uint64(runtime.LHBit('L',w.Position()))}", width)
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "sample"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "sample", "generated.go"), source, 0o600); err != nil {
			t.Fatal(err)
		}
		findings, err := ScanSource("sample/generated.go", source)
		if err != nil {
			t.Fatal(err)
		}
		checker := &citationChecker{root: dir, files: make(map[string]citationFile)}
		checked := 0
		for _, finding := range findings {
			if finding.Expression != "uint64(runtime.LHBit('L', w.Position()))" {
				continue
			}
			checked++
			if err := checker.checkGeneratedResidual(finding); (err == nil) != (width == "1") {
				t.Fatalf("literal %s: %v", width, err)
			}
		}
		if checked != 1 {
			t.Fatalf("literal %s: checked %d expressions: %+v", width, checked, findings)
		}
	}
}
