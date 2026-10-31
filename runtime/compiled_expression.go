package runtime

import "fmt"

// ExpressionValue holds a generated width expression's intermediate value or
// its first error. Its zero value is the constant zero. Arithmetic methods
// validate their operands as well as their intermediate results.
type ExpressionValue struct {
	n   int64
	err error
}

// ExpressionConstant represents a parsed int64 literal before operand checks.
func ExpressionConstant(n int64) ExpressionValue { return ExpressionValue{n: n} }

func expressionVariable(name string, vars map[string]uint64) ExpressionValue {
	return expressionVariableFrom(name, vars, nil)
}

func expressionVariableFrom(name string, vars map[string]uint64, reader *Reader) ExpressionValue {
	p := expression{vars: vars, reader: reader}
	n, ok, err := p.resolveVar(name)
	if err != nil {
		return ExpressionValue{err: err}
	}
	if !ok {
		return ExpressionValue{err: fmt.Errorf("unresolved width variable %q", name)}
	}
	if n > maxBits {
		return ExpressionValue{err: fmt.Errorf("width variable %q outside 0..%d", name, maxBits)}
	}
	return ExpressionValue{n: int64(n)}
}

// ExpressionVariable resolves a single-token reference using Eval's exact and
// compact name matching, including its ambiguity and variable-bound checks.
// State after ChoiceErrors.Release is undefined for a borrowed trial.
func (r *Reader) ExpressionVariable(name string) ExpressionValue {
	return expressionVariableFrom(name, r.vars, r)
}

// ExpressionVariable resolves a single-token reference using Eval's exact and
// compact name matching, including its ambiguity and variable-bound checks.
func (w *Writer) ExpressionVariable(name string) ExpressionValue {
	return expressionVariable(name, w.vars)
}

// Operand applies the product parser's operand bound before any multiplication
// or division, and before returning even a product containing just one primary.
func (v ExpressionValue) Operand() ExpressionValue {
	if v.err == nil && (v.n < -maxBits || v.n > maxBits) {
		v.err = fmt.Errorf("expression operand outside limits")
	}
	return v
}

// Negate applies the unary primary operator before its enclosing operand check.
func (v ExpressionValue) Negate() ExpressionValue {
	v = v.Operand()
	if v.err != nil {
		return v
	}
	v.n = -v.n
	return v
}

// Add adds two checked products, preserving the leftmost error.
func (v ExpressionValue) Add(right ExpressionValue) ExpressionValue {
	v, right = v.Operand(), right.Operand()
	if v.err != nil {
		return v
	}
	if right.err != nil {
		return right
	}
	if v.n > maxBits-right.n || v.n < -maxBits-right.n {
		return ExpressionValue{err: fmt.Errorf("expression result outside limits")}
	}
	v.n += right.n
	return v.resultBound()
}

// Subtract subtracts two checked products, preserving the leftmost error.
func (v ExpressionValue) Subtract(right ExpressionValue) ExpressionValue {
	v, right = v.Operand(), right.Operand()
	if v.err != nil {
		return v
	}
	if right.err != nil {
		return right
	}
	if v.n > maxBits+right.n || v.n < -maxBits+right.n {
		return ExpressionValue{err: fmt.Errorf("expression result outside limits")}
	}
	v.n -= right.n
	return v.resultBound()
}

// Multiply multiplies two checked operands, rejecting each intermediate result.
func (v ExpressionValue) Multiply(right ExpressionValue) ExpressionValue {
	v, right = v.Operand(), right.Operand()
	if v.err != nil {
		return v
	}
	if right.err != nil {
		return right
	}
	// Both operands have passed Operand, exactly as in expression.product.
	// Their product is at most maxBits squared, which fits in int64.
	v.n *= right.n
	return v.resultBound()
}

// Divide divides two checked operands, rejecting a zero divisor at this step.
func (v ExpressionValue) Divide(right ExpressionValue) ExpressionValue {
	v, right = v.Operand(), right.Operand()
	if v.err != nil {
		return v
	}
	if right.err != nil {
		return right
	}
	if right.n == 0 {
		return ExpressionValue{err: fmt.Errorf("division by zero")}
	}
	v.n /= right.n
	return v.resultBound()
}

func (v ExpressionValue) resultBound() ExpressionValue {
	if v.n < -maxBits || v.n > maxBits {
		v.err = fmt.Errorf("expression result outside limits")
	}
	return v
}

// CellWidth applies Eval's p or q table, including its zero-valued tail entries
// and index checks. Generated code passes only the table names "p" and "q".
func (v ExpressionValue) CellWidth(name string) ExpressionValue {
	if v.err != nil {
		return v
	}
	if v.n < 0 || v.n > 31 {
		return ExpressionValue{err: fmt.Errorf("%s index outside 0..31", name)}
	}
	table := cellInformationWidths[name]
	if v.n >= int64(len(table)) {
		v.n = 0
	} else {
		v.n = table[v.n]
	}
	return v
}

func (v ExpressionValue) width() (int, error) {
	if v.err != nil {
		return 0, v.err
	}
	if v.n < 0 || v.n > maxBits {
		return 0, fmt.Errorf("width %d outside 0..%d", v.n, maxBits)
	}
	return int(v.n), nil
}

// ExpressionResult applies Eval's final width check and reader error location.
func (r *Reader) ExpressionResult(v ExpressionValue) (int, error) {
	n, err := v.width()
	if err != nil {
		return 0, r.fail(InvalidSchema, err.Error())
	}
	return n, nil
}

// ExpressionResult applies Eval's final width check and writer error behavior.
func (w *Writer) ExpressionResult(v ExpressionValue) (int, error) { return v.width() }

// ExpressionResultWithContext applies final bounds and locates every compiled
// expression failure, including arithmetic and unresolved scalar bindings.
func (r *Reader) ExpressionResultWithContext(v ExpressionValue) (int, error) {
	n, err := v.width()
	if err != nil {
		return 0, r.expressionFailure(err)
	}
	return n, nil
}

// ExpressionResultWithContext is the writer counterpart of
// Reader.ExpressionResultWithContext. It retains the writer path and bit offset.
func (w *Writer) ExpressionResultWithContext(v ExpressionValue) (int, error) {
	n, err := v.width()
	if err != nil {
		return 0, w.expressionFailure(err)
	}
	return n, nil
}
