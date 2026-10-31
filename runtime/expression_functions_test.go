package runtime

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestSourceFunctionSyntax(t *testing.T) {
	for _, raw := range []string{
		"1+max(val(CELL_PARAMS_POINTER))",
		"expanded_EPDAN_extension_length(val(EPD A N Extension length index))",
	} {
		if err := ValidateExpression(raw); err != nil {
			t.Errorf("source-defined expression %q: %v", raw, err)
		}
	}
}

func TestSourceFunctionSyntaxRefusal(t *testing.T) {
	for _, raw := range []string{
		"max()", "max(X)", "max(val())", "max(val(X)+1)", "max(val(X * 2))", "max(val(X)",
		"expanded_EPDAN_extension_length()", "expanded_EPDAN_extension_length(1,2)",
		"expanded_EPDAN_extension_length(1", "unknown(1)", "q(1", "q()",
	} {
		if err := ValidateExpression(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
}

// TS 44.060 V19.0.0 §11.2.6a table 11.2.6a.1a pp14501–14515,
// verbatim function body from the authenticated archive.
const epdanDefinition = `unsigned char expanded_EPDAN_extension_length (unsigned char index)
{

/*Indicates whether the mobile station is assigned a DLDC configuration (1) or not (0) */
extern unsigned char DLDC_configuration;

unsigned char L;

if ((index >= 6) OR (DLDC_configuration == 0))
L = index + 1;
else
L = index + 65;

return (L);
}`

// TS 44.018 V19.0.0 §9.1.54 table 9.1.54.1b pp11173–11232.
// Verbatim cell text in table row order, separated by tabs.
const qDefinition = "m\tq\tm\tq\tm\tq\tm\tq\tm\tq\n" +
	"0\t0\t5\t39\t10\t71\t15\t101\t20\t126\n" +
	"1\t9\t6\t46\t11\t77\t16\t106\t21-31\t0\n" +
	"2\t17\t7\t53\t12\t83\t17\t111\t\t\n" +
	"3\t25\t8\t59\t13\t89\t18\t116\t\t\n" +
	"4\t32\t9\t65\t14\t95\t19\t121\t\t"

func sourceFunctions(t testing.TB) (*ExpressionContext, [][]uint64, []uint64) {
	t.Helper()
	// Extract the threshold and offsets from the quoted source, rather than
	// maintaining a second handwritten table of function values.
	threshold := regexp.MustCompile(`index >= ([0-9]+)`).FindStringSubmatch(epdanDefinition)
	offsets := regexp.MustCompile(`L = index \+ ([0-9]+)`).FindAllStringSubmatch(epdanDefinition, -1)
	if len(threshold) != 2 || len(offsets) != 2 || !strings.Contains(epdanDefinition, "DLDC_configuration == 0") {
		t.Fatal("source function shape changed")
	}
	boundary, _ := strconv.Atoi(threshold[1])
	small, _ := strconv.Atoi(offsets[0][1])
	large, _ := strconv.Atoi(offsets[1][1])
	rows := [][]uint64{make([]uint64, 64), make([]uint64, 64)}
	for d := range rows {
		for i := range rows[d] {
			offset := small
			if d != 0 && i < boundary {
				offset = large
			}
			rows[d][i] = uint64(i + offset)
		}
	}
	q := make([]uint64, 32)
	seen := make([]bool, 32)
	for _, line := range strings.Split(qDefinition, "\n")[1:] {
		cells := strings.Split(line, "\t")
		for i := 0; i+1 < len(cells); i += 2 {
			if cells[i] == "" {
				continue
			}
			bounds := strings.Split(cells[i], "-")
			lo, err := strconv.Atoi(bounds[0])
			if err != nil {
				t.Fatal(err)
			}
			hi := lo
			if len(bounds) == 2 {
				hi, err = strconv.Atoi(bounds[1])
				if err != nil {
					t.Fatal(err)
				}
			}
			n, err := strconv.ParseUint(cells[i+1], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			for index := lo; index <= hi; index++ {
				if index < 0 || index >= len(q) || seen[index] {
					t.Fatal("invalid or repeated source index")
				}
				q[index], seen[index] = n, true
			}
		}
	}
	for i, present := range seen {
		if !present {
			t.Fatalf("source index %d missing", i)
		}
	}
	f, err := NewExpressionFunction("expanded_EPDAN_extension_length", epdanSource, "DLDC_configuration", rows)
	if err != nil {
		t.Fatal(err)
	}
	g, err := NewExpressionFunction("q", Source{Standard: "TS 44.018", Version: "19.0.0", Clause: "9.1.54", Table: "9.1.54.1b pp11173–11232"}, "", [][]uint64{q})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := NewExpressionContext([]*ExpressionFunction{f, g}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, rows, q
}

func TestExpandedEPDANLength(t *testing.T) {
	ctx, rows, _ := sourceFunctions(t)
	for d := range 2 {
		for i := range 64 {
			r, w := NewReader([]byte{0}), NewWriter()
			r.Set("DLDC_configuration", uint64(d))
			w.Set("DLDC_configuration", uint64(d))
			raw := "expanded_EPDAN_extension_length(val(index))"
			r.Set("index", uint64(i))
			w.Set("index", uint64(i))
			tokens, err := expressionTokens(raw)
			if err != nil {
				t.Fatal(err)
			}
			values := []struct {
				n   int
				err error
			}{}
			n, e := r.EvalWithContext(raw, ctx)
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			n, e = w.EvalWithContext(raw, ctx)
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			n, e = r.EvalTokensWithContext(tokens, ctx)
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			n, e = w.EvalTokensWithContext(tokens, ctx)
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			n, e = r.ExpressionResultWithContext(r.ExpressionFunction("expanded_EPDAN_extension_length", r.ExpressionVariable("index"), ctx))
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			n, e = w.ExpressionResultWithContext(w.ExpressionFunction("expanded_EPDAN_extension_length", w.ExpressionVariable("index"), ctx))
			values = append(values, struct {
				n   int
				err error
			}{n, e})
			for path, v := range values {
				if v.err != nil || v.n != int(rows[d][i]) {
					t.Errorf("DLDC=%d index=%d path=%d: %d %v want %d", d, i, path, v.n, v.err, rows[d][i])
				}
			}
		}
	}
	// Worked boundary values, TS 44.060 table 11.2.6a.1a pp14487–14490.
	for _, tc := range []struct{ d, i, want uint64 }{{0, 0, 1}, {1, 0, 65}, {1, 5, 70}, {1, 6, 7}, {0, 63, 64}, {1, 63, 64}} {
		if got := rows[tc.d][tc.i]; got != tc.want {
			t.Errorf("%+v got %d", tc, got)
		}
	}
}

func TestSourceQTable(t *testing.T) {
	ctx, _, q := sourceFunctions(t)
	r, w := NewReader(nil), NewWriter()
	for i, want := range q {
		raw := fmt.Sprintf("q(%d)", i)
		a, e := r.EvalWithContext(raw, ctx)
		b, f := r.ExpressionResultWithContext(r.ExpressionFunction("q", ExpressionConstant(int64(i)), ctx))
		old, oldErr := w.Eval(raw)
		if a != int(want) || b != a || old != a || e != nil || f != nil || oldErr != nil {
			t.Errorf("%s: interpreted %d/%v compiled %d/%v existing %d/%v want %d", raw, a, e, b, f, old, oldErr, want)
		}
	}
}

func TestSourceFunctionFailures(t *testing.T) {
	ctx, _, _ := sourceFunctions(t)
	for _, tc := range []struct {
		raw, name string
		value     ExpressionValue
		d         uint64
		bind      bool
		ctx       *ExpressionContext
	}{
		{"expanded_EPDAN_extension_length(0)", "expanded_EPDAN_extension_length", ExpressionConstant(0), 0, false, ctx},
		{"expanded_EPDAN_extension_length(0)", "expanded_EPDAN_extension_length", ExpressionConstant(0), 2, true, ctx},
		{"expanded_EPDAN_extension_length(64)", "expanded_EPDAN_extension_length", ExpressionConstant(64), 0, true, ctx},
		{"expanded_EPDAN_extension_length(-1)", "expanded_EPDAN_extension_length", ExpressionConstant(-1), 0, true, ctx},
		{"expanded_EPDAN_extension_length(1048577)", "expanded_EPDAN_extension_length", ExpressionConstant(maxBits + 1), 0, true, ctx},
		{"expanded_EPDAN_extension_length(missing)", "expanded_EPDAN_extension_length", expressionVariable("missing", nil), 0, true, ctx},
		{"expanded_EPDAN_extension_length(0)", "expanded_EPDAN_extension_length", ExpressionConstant(0), 0, true, nil},
		{"q(32)", "q", ExpressionConstant(32), 0, true, ctx},
		{"q(-1)", "q", ExpressionConstant(-1), 0, true, ctx},
	} {
		r, w := NewReader([]byte{0}), NewWriter()
		if err := r.Enter("field"); err != nil {
			t.Fatal(err)
		}
		if err := w.Enter("field"); err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(3); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, 3); err != nil {
			t.Fatal(err)
		}
		if tc.bind {
			r.Set("DLDC_configuration", tc.d)
			w.Set("DLDC_configuration", tc.d)
		}
		_, re := r.EvalWithContext(tc.raw, tc.ctx)
		_, rc := r.ExpressionResultWithContext(r.ExpressionFunction(tc.name, tc.value, tc.ctx))
		_, we := w.EvalWithContext(tc.raw, tc.ctx)
		_, wc := w.ExpressionResultWithContext(w.ExpressionFunction(tc.name, tc.value, tc.ctx))
		if re == nil || rc == nil || we == nil || wc == nil {
			t.Fatalf("accepted %s: %v %v %v %v", tc.raw, re, rc, we, wc)
		}
		if re.Error() != rc.Error() || we.Error() != wc.Error() {
			t.Errorf("%s parity: reader %v / %v writer %v / %v", tc.raw, re, rc, we, wc)
		}
		for _, e := range []error{re, rc} {
			var de *DecodeError
			var oe *OperationError
			if errors.As(e, &oe) {
				if oe.Location.Path != "field" || oe.Location.BitOffset != 3 {
					t.Errorf("location %+v", oe.Location)
				}
			} else if errors.As(e, &de) {
				if de.Path != "field" || de.Offset != 3 {
					t.Errorf("location %+v", de)
				}
			} else {
				t.Errorf("unlocated %v", e)
			}
		}
		for _, e := range []error{we, wc} {
			var oe *OperationError
			if !errors.As(e, &oe) || oe.Location.Path != "field" || oe.Location.BitOffset != 3 {
				t.Errorf("writer location %v", e)
			}
		}
		if (!tc.bind || tc.ctx == nil) && !errors.Is(re, ErrContextRequired) {
			t.Errorf("missing context identity: %v", re)
		}
	}
}

func TestExpressionTableOwnershipAndValidation(t *testing.T) {
	ctx, rows, _ := sourceFunctions(t)
	rows[1][0] = 999
	r := NewReader(nil)
	r.Set("DLDC_configuration", 1)
	if n, e := r.EvalWithContext("expanded_EPDAN_extension_length(0)", ctx); n != 65 || e != nil {
		t.Fatalf("borrowed input %d %v", n, e)
	}
	valid := make([]uint64, 32)
	for _, tc := range []struct {
		name, context string
		source        Source
		rows          [][]uint64
	}{
		{"unknown", "", epdanSource, [][]uint64{valid}},
		{"q", "DLDC_configuration", epdanSource, [][]uint64{valid}},
		{"q", "", Source{}, [][]uint64{valid}},
		{"q", "", epdanSource, nil},
		{"q", "", epdanSource, [][]uint64{valid[:31]}},
		{"expanded_EPDAN_extension_length", "", epdanSource, rows},
		{"expanded_EPDAN_extension_length", "DLDC_configuration", epdanSource, rows[:1]},
	} {
		if _, err := NewExpressionFunction(tc.name, tc.source, tc.context, tc.rows); err == nil {
			t.Errorf("accepted %+v", tc)
		}
	}
	valid[31] = maxBits + 1
	if _, err := NewExpressionFunction("q", epdanSource, "", [][]uint64{valid}); err == nil {
		t.Fatal("accepted oversized output")
	}
	f := ctx.functions["q"]
	if _, err := NewExpressionContext([]*ExpressionFunction{f, f}, nil); err == nil {
		t.Fatal("duplicate function")
	}
	if _, err := NewExpressionContext([]*ExpressionFunction{nil}, nil); err == nil {
		t.Fatal("nil function")
	}
	if _, err := NewExpressionContext([]*ExpressionFunction{{}}, nil); err == nil {
		t.Fatal("zero function")
	}
}

func TestBoundEPDANLabel(t *testing.T) {
	ctx, _, _ := sourceFunctions(t)
	r, w := NewReader(nil), NewWriter()
	for _, v := range []interface{ Set(string, uint64) }{r, w} {
		v.Set("DLDC_configuration", 1)
		v.Set("EPD_A_N_Extension_length_index", 5)
	}
	raw := "expanded_EPDAN_extension_length(val(EPD_A_N_Extension_length_index))"
	a, e := r.EvalWithContext(raw, ctx)
	b, f := w.EvalWithContext(raw, ctx)
	if a != 70 || b != 70 || e != nil || f != nil {
		t.Fatal(a, e, b, f)
	}
}

func TestContextArithmeticFailures(t *testing.T) {
	ctx, _, _ := sourceFunctions(t)
	for _, tc := range []struct {
		raw   string
		value ExpressionValue
	}{
		{"q(0)+1048576+1", ExpressionConstant(0).Add(ExpressionConstant(maxBits)).Add(ExpressionConstant(1))},
		{"q(0)-1", ExpressionConstant(0).Subtract(ExpressionConstant(1))},
		{"q(1)*1048576", ExpressionConstant(9).Multiply(ExpressionConstant(maxBits))},
		{"q(1)/0", ExpressionConstant(9).Divide(ExpressionConstant(0))},
	} {
		r, w := NewReader(nil), NewWriter()
		_, a := r.EvalWithContext(tc.raw, ctx)
		_, b := r.ExpressionResultWithContext(tc.value)
		_, c := w.EvalWithContext(tc.raw, ctx)
		_, d := w.ExpressionResultWithContext(tc.value)
		if a == nil || b == nil || c == nil || d == nil || a.Error() != b.Error() || c.Error() != d.Error() {
			t.Errorf("%s: %v %v %v %v", tc.raw, a, b, c, d)
		}
	}
}

func FuzzExpressionFunctionParity(f *testing.F) {
	for _, s := range []struct {
		i    int64
		d    uint64
		bind bool
	}{{0, 0, true}, {0, 1, true}, {5, 1, true}, {6, 1, true}, {63, 1, true}, {64, 0, true}, {-1, 0, true}, {0, 2, true}, {0, 0, false}, {maxBits + 1, 1, true}} {
		f.Add(s.i, s.d, s.bind)
	}
	ctx, _, _ := sourceFunctions(f)
	f.Fuzz(func(t *testing.T, index int64, d uint64, bind bool) {
		if index == -1<<63 {
			t.Skip()
		}
		r, w := NewReader(nil), NewWriter()
		if bind {
			r.Set("DLDC_configuration", d)
			w.Set("DLDC_configuration", d)
		}
		raw := fmt.Sprintf("expanded_EPDAN_extension_length(%d)", index)
		a, e := r.EvalWithContext(raw, ctx)
		b, f := r.ExpressionResultWithContext(r.ExpressionFunction("expanded_EPDAN_extension_length", ExpressionConstant(index), ctx))
		c, g := w.EvalWithContext(raw, ctx)
		h, j := w.ExpressionResultWithContext(w.ExpressionFunction("expanded_EPDAN_extension_length", ExpressionConstant(index), ctx))
		if a != b || c != h || fmt.Sprint(e) != fmt.Sprint(f) || fmt.Sprint(g) != fmt.Sprint(j) {
			t.Fatalf("%s d=%d bind=%v reader %d,%v / %d,%v writer %d,%v / %d,%v", raw, d, bind, a, e, b, f, c, g, h, j)
		}
	})
}

func TestSourceFunctionNamesRemainScalarBindings(t *testing.T) {
	r, w := NewReader(nil), NewWriter()
	for _, name := range []string{"max", "expanded_EPDAN_extension_length"} {
		r.Set(name, 7)
		w.Set(name, 7)
		for _, raw := range []string{name, "val(" + name + ")", name + "+1"} {
			want := 7
			if strings.HasSuffix(raw, "+1") {
				want = 8
			}
			for _, codec := range []interface{ Eval(string) (int, error) }{r, w} {
				if n, e := codec.Eval(raw); n != want || e != nil {
					t.Errorf("%s = %d %v", raw, n, e)
				}
			}
		}
	}
}

func TestExpressionContextPreservesDivision(t *testing.T) {
	ctx, e := NewExpressionContext(nil, nil)
	if e != nil {
		t.Fatal(e)
	}
	r, w := NewReader(nil), NewWriter()
	for _, codec := range []interface{ Set(string, uint64) }{r, w} {
		codec.Set("a", 8)
		codec.Set("b", 2)
		codec.Set("a/b", 6)
	}
	tokens, e := expressionTokens("val(a / b)")
	if e != nil {
		t.Fatal(e)
	}
	a, e := r.EvalWithContext("val(a / b)", ctx)
	b, f := w.EvalWithContext("val(a / b)", ctx)
	c, g := r.EvalTokensWithContext(tokens, ctx)
	d, h := w.EvalTokensWithContext(tokens, ctx)
	for _, v := range []struct {
		n int
		e error
	}{{a, e}, {b, f}, {c, g}, {d, h}} {
		if v.n != 4 || v.e != nil {
			t.Errorf("division = %d %v", v.n, v.e)
		}
	}
}
