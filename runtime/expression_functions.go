package runtime

import (
	"errors"
	"fmt"
	"strings"
)

// ExpressionFunction is an immutable, source-supplied width table. Rows select
// context values, columns select arguments. The runtime checks shape and bounds;
// the compiler must authenticate the values against Source before construction.
// TS 44.060 V19.0.0 §11.2.6a table 11.2.6a.1a (pp14483–14514)
// defines the two DLDC rows; TS 44.018 V19.0.0 §9.1.54 tables
// 9.1.54.1a/b define p and q. No values are inferred for missing entries.
type ExpressionFunction struct {
	name, context string
	source        Source
	rows          [][]uint64
}

var epdanSource = Source{Standard: "TS 44.060", Version: "19.0.0", Clause: "11.2.6a", Table: "11.2.6a.1a pp14483–14514"}

// NewExpressionFunction validates and copies an authenticated table. Only p, q
// and expanded_EPDAN_extension_length are supported. p/q require one 32-entry
// row and no context; EPDAN requires two 64-entry rows and the binding name
// DLDC_configuration. Every result must be within the runtime width limit.
func NewExpressionFunction(name string, source Source, context string, rows [][]uint64) (*ExpressionFunction, error) {
	nrows, ncols := 1, 32
	switch name {
	case "p", "q":
		if context != "" {
			return nil, fmt.Errorf("%s does not take context", name)
		}
	case "expanded_EPDAN_extension_length":
		nrows, ncols = 2, 64
		if context != "DLDC_configuration" {
			return nil, fmt.Errorf("%s requires DLDC_configuration", name)
		}
	default:
		return nil, fmt.Errorf("unsupported expression function %q", name)
	}
	if source.Standard == "" || source.Version == "" || source.Clause == "" || source.Table == "" {
		return nil, fmt.Errorf("%s requires a source citation", name)
	}
	if len(rows) != nrows {
		return nil, fmt.Errorf("%s requires %d table rows", name, nrows)
	}
	for _, row := range rows {
		if len(row) != ncols {
			return nil, fmt.Errorf("%s requires %d table columns", name, ncols)
		}
		for _, n := range row {
			if n > maxBits {
				return nil, fmt.Errorf("%s table value outside 0..%d", name, maxBits)
			}
		}
	}
	f := &ExpressionFunction{name: name, context: context, source: source, rows: make([][]uint64, len(rows))}
	for i, row := range rows {
		f.rows[i] = append([]uint64(nil), row...)
	}
	return f, nil
}

// ExpressionContext holds immutable table and aggregate identities. Mutable
// values live exclusively in a reader's or writer's existing owned bindings.
// A context can be shared by concurrent codecs after construction.
type ExpressionContext struct {
	functions  map[string]*ExpressionFunction
	aggregates map[string]Aggregate
}

// NewExpressionContext copies identities, refusing duplicate names rather than
// selecting an arbitrary table or aggregate. It does not retain the input slices.
func NewExpressionContext(functions []*ExpressionFunction, aggregates []Aggregate) (*ExpressionContext, error) {
	c := &ExpressionContext{functions: make(map[string]*ExpressionFunction), aggregates: make(map[string]Aggregate)}
	for _, f := range functions {
		if f == nil || f.name == "" {
			return nil, fmt.Errorf("uninitialized expression function")
		}
		if _, ok := c.functions[f.name]; ok {
			return nil, fmt.Errorf("duplicate expression function %q", f.name)
		}
		c.functions[f.name] = f
	}
	for _, a := range aggregates {
		if a.count == "" {
			return nil, fmt.Errorf("uninitialized aggregate")
		}
		if _, ok := c.aggregates[a.label]; ok {
			return nil, fmt.Errorf("duplicate aggregate label %q", a.label)
		}
		c.aggregates[a.label] = a
	}
	return c, nil
}

type expressionFunctionError struct {
	source Source
	detail string
	cause  error
}

func (e *expressionFunctionError) Error() string {
	return fmt.Sprintf("%s V%s §%s table %s: %s", e.source.Standard, e.source.Version, e.source.Clause, e.source.Table, e.detail)
}
func (e *expressionFunctionError) Unwrap() error { return e.cause }

func functionError(source Source, detail string, cause error) ExpressionValue {
	return ExpressionValue{err: &expressionFunctionError{source: source, detail: detail, cause: cause}}
}

func (p *expression) applyFunction(name string, arg ExpressionValue) ExpressionValue {
	arg = arg.Operand()
	if arg.err != nil {
		return arg
	}
	var f *ExpressionFunction
	if p.context != nil {
		f = p.context.functions[name]
	}
	if f == nil {
		if name == "p" || name == "q" {
			return arg.CellWidth(name)
		}
		if name != "expanded_EPDAN_extension_length" {
			return ExpressionValue{err: fmt.Errorf("unsupported expression function %q", name)}
		}
		return functionError(epdanSource, "missing authenticated "+name+" table", ErrContextRequired)
	}
	if arg.n < 0 || arg.n >= int64(len(f.rows[0])) {
		return functionError(f.source, name+" argument outside table", nil)
	}
	row := uint64(0)
	if f.context != "" {
		n, ok, err := p.resolveVar(f.context)
		if err != nil {
			return functionError(f.source, err.Error(), err)
		}
		if !ok {
			return functionError(f.source, "missing "+f.context, ErrContextRequired)
		}
		if n >= uint64(len(f.rows)) {
			return functionError(f.source, f.context+" outside table", nil)
		}
		row = n
	}
	return ExpressionValue{n: int64(f.rows[row][arg.n])}
}

func (p *expression) sourceFunction(name string) (int64, error) {
	if p.take() != "(" {
		return 0, fmt.Errorf("%s needs (", name)
	}
	n, err := p.sum()
	if err != nil {
		return 0, err
	}
	if p.take() != ")" {
		return 0, fmt.Errorf("unclosed %s", name)
	}
	if p.syntaxOnly {
		return 0, nil
	}
	v := p.applyFunction(name, ExpressionValue{n: n})
	return v.n, v.err
}

func (p *expression) maximum() (int64, error) {
	if p.take() != "(" || p.take() != "val" || p.take() != "(" {
		return 0, fmt.Errorf("max requires max(val(label))")
	}
	var label strings.Builder
	for p.peek() != "" && p.peek() != ")" {
		t := p.take()
		if t == "(" || t == "+" || t == "*" || t == "/" {
			return 0, fmt.Errorf("max requires a field label")
		}
		if label.Len() > 0 && t != "-" && p.tokens[p.position-2] != "-" {
			label.WriteByte(' ')
		}
		label.WriteString(t)
	}
	if label.Len() == 0 || p.take() != ")" || p.take() != ")" {
		return 0, fmt.Errorf("max requires max(val(label))")
	}
	if p.syntaxOnly {
		return 0, nil
	}
	var a Aggregate
	if p.context != nil {
		a = p.context.aggregates[key(label.String())]
	}
	v := p.aggregateValue(a, "maximum")
	return v.n, v.err
}

// EvalWithContext evaluates source-defined functions with explicit immutable
// tables and aggregate identities, retaining the reader's absolute location.
func (r *Reader) EvalWithContext(raw string, context *ExpressionContext) (int, error) {
	n, err := evalContext(raw, context, r.vars, r)
	if err != nil {
		return 0, r.expressionFailure(err)
	}
	return n, nil
}

// EvalWithContext is the writer counterpart of Reader.EvalWithContext.
func (w *Writer) EvalWithContext(raw string, context *ExpressionContext) (int, error) {
	n, err := evalContext(raw, context, w.vars, nil)
	if err != nil {
		return 0, w.expressionFailure(err)
	}
	return n, nil
}

func evalContext(raw string, context *ExpressionContext, vars map[string]uint64, reader *Reader) (int, error) {
	tokens, err := expressionTokens(raw)
	if err != nil {
		return 0, err
	}
	return evalContextTokens(tokens, context, vars, reader)
}

func evalContextTokens(tokens []string, context *ExpressionContext, vars map[string]uint64, reader *Reader) (int, error) {
	p := expression{tokens: tokens, context: context, vars: vars, reader: reader}
	n, err := p.sum()
	if err != nil {
		return 0, err
	}
	if p.position != len(tokens) {
		return 0, fmt.Errorf("unexpected expression token %q", tokens[p.position])
	}
	return (ExpressionValue{n: n}).width()
}

// EvalTokensWithContext has EvalTokens' borrowing contract and evaluates the
// same functions as EvalWithContext.
func (r *Reader) EvalTokensWithContext(tokens []string, context *ExpressionContext) (int, error) {
	n, err := evalContextTokens(tokens, context, r.vars, r)
	if err != nil {
		return 0, r.expressionFailure(err)
	}
	return n, nil
}

// EvalTokensWithContext is the writer counterpart of Reader.EvalTokensWithContext.
func (w *Writer) EvalTokensWithContext(tokens []string, context *ExpressionContext) (int, error) {
	n, err := evalContextTokens(tokens, context, w.vars, nil)
	if err != nil {
		return 0, w.expressionFailure(err)
	}
	return n, nil
}

// ExpressionFunction applies a source-supplied table to a compiled operand.
func (r *Reader) ExpressionFunction(name string, arg ExpressionValue, context *ExpressionContext) ExpressionValue {
	p := expression{vars: r.vars, reader: r, context: context}
	return p.applyFunction(name, arg)
}

// ExpressionFunction is the writer counterpart of Reader.ExpressionFunction.
func (w *Writer) ExpressionFunction(name string, arg ExpressionValue, context *ExpressionContext) ExpressionValue {
	p := expression{vars: w.vars, context: context}
	return p.applyFunction(name, arg)
}

func (r *Reader) expressionFailure(err error) error {
	var f *expressionFunctionError
	if errors.As(err, &f) {
		return &OperationError{Operation: "decode", Kind: InvalidSchema,
			Location: Location{Path: strings.Join(r.path, "/"), BitOffset: r.Position(), Source: f.source}, Cause: err}
	}
	return r.fail(InvalidSchema, err.Error())
}

func (w *Writer) expressionFailure(err error) error {
	var f *expressionFunctionError
	source := Source{}
	if errors.As(err, &f) {
		source = f.source
	}
	return &OperationError{Operation: "encode", Kind: InvalidSchema,
		Location: Location{Path: w.path, BitOffset: w.Position(), Source: source}, Cause: err}
}
