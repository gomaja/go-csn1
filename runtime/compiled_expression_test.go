package runtime

import (
	"fmt"
	"math"
	"testing"
)

func TestCompiledExpressionBoundaries(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		value ExpressionValue
	}{
		{"1048577", ExpressionConstant(maxBits + 1).Operand()},
		{"1048576+1", ExpressionConstant(maxBits).Operand().Add(ExpressionConstant(1).Operand())},
		{"0-1", ExpressionConstant(0).Operand().Subtract(ExpressionConstant(1).Operand())},
		{"1048576*2", ExpressionConstant(maxBits).Operand().Multiply(ExpressionConstant(2).Operand())},
		{"1/0", ExpressionConstant(1).Operand().Divide(ExpressionConstant(0).Operand())},
		{"-1", ExpressionConstant(1).Negate().Operand()},
		{"p(32)", ExpressionConstant(32).Operand().CellWidth("p").Operand()},
		{"q(31)", ExpressionConstant(31).Operand().CellWidth("q").Operand()},
		{"-9223372036854775807", ExpressionConstant(math.MaxInt64).Negate().Operand()},
	} {
		want, we := eval(tc.raw, nil)
		got, ge := NewWriter().ExpressionResult(tc.value)
		if got != want || fmt.Sprint(ge) != fmt.Sprint(we) {
			t.Errorf("%s: (%d,%v), want (%d,%v)", tc.raw, got, ge, want, we)
		}
	}
}

func TestCompiledExpressionVariableResolution(t *testing.T) {
	for _, vars := range []map[string]uint64{
		nil, {"x": 0}, {"x": maxBits}, {"x": maxBits + 1}, {"x": math.MaxUint64},
		{"x y": 1}, {"x y": 1, "x  y": 2}, {"xy": 3, "x y": 1},
	} {
		for _, label := range []string{"x", "xy"} {
			r := NewReader(nil)
			w := NewWriter()
			r.vars, w.vars = vars, vars
			want, we := w.Eval(label)
			got, ge := w.ExpressionResult(w.ExpressionVariable(label).Operand())
			if got != want || fmt.Sprint(ge) != fmt.Sprint(we) {
				t.Errorf("%q %v: (%d,%v) want (%d,%v)", label, vars, got, ge, want, we)
			}
			got, ge = r.ExpressionResult(r.ExpressionVariable(label).Operand())
			want, we = r.Eval(label)
			if got != want || fmt.Sprint(ge) != fmt.Sprint(we) {
				t.Errorf("reader %q: (%d,%v) want (%d,%v)", label, got, ge, want, we)
			}
		}
	}
}

func TestCompiledExpressionUncheckedOperands(t *testing.T) {
	operations := map[string]func(ExpressionValue, ExpressionValue) ExpressionValue{
		"add": ExpressionValue.Add, "subtract": ExpressionValue.Subtract,
		"multiply": ExpressionValue.Multiply, "divide": ExpressionValue.Divide,
	}
	for _, n := range []int64{math.MinInt64, math.MinInt64 + 1, -maxBits - 1, maxBits + 1, 1 << 32, math.MaxInt64} {
		for name, operation := range operations {
			for _, operands := range [][2]int64{{n, 0}, {0, n}, {n, n}, {n, -1}} {
				value := operation(ExpressionConstant(operands[0]), ExpressionConstant(operands[1]))
				if _, err := NewWriter().ExpressionResult(value); err == nil || err.Error() != "expression operand outside limits" {
					t.Errorf("%s(%d,%d): %v", name, operands[0], operands[1], err)
				}
			}
		}
		if _, err := NewWriter().ExpressionResult(ExpressionConstant(n).Negate()); err == nil || err.Error() != "expression operand outside limits" {
			t.Errorf("negate(%d): %v", n, err)
		}
	}
}

func BenchmarkExpressionWidth(b *testing.B) {
	for _, compiled := range []bool{false, true} {
		b.Run(fmt.Sprintf("compiled=%v", compiled), func(b *testing.B) {
			r := NewReader(nil)
			r.Set("x", 5)
			b.ReportAllocs()
			for b.Loop() {
				var n int
				var err error
				if compiled {
					n, err = r.ExpressionResult(r.ExpressionVariable("x").Operand().Add(ExpressionConstant(1).Operand()))
				} else {
					n, err = r.Eval("val(x)+1")
				}
				if n != 6 || err != nil {
					b.Fatalf("width %d: %v", n, err)
				}
			}
		})
	}
}
