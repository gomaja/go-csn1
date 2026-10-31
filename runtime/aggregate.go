package runtime

import (
	"fmt"
	"math/bits"
	"strings"
)

// Aggregate is an immutable identity for one field in one generated scope.
// Its count, maximum and bitmap population live in private expression bindings,
// so Reader.Fork and choice trials isolate changes through existing copy-on-write
// storage. Reset at each enclosing list/TBF; use distinct identities for nested
// scopes. TS 44.060 V19.0.0 §11.2.21 p17216 uses a list maximum;
// §§11.2.29a pp18719–18729 and 12.42 pp23778–23792 use allocation populations.
// Aggregate never retains repeated values or pointers to a reader or writer.
type Aggregate struct {
	label, count, maximum, population string
	source                            Source
}

// NewAggregate constructs a private scope/field identity, with an exact source
// citation. Scope and label must be nonempty and cannot contain NUL. A generated
// scope identifies the containing production and node path, not a global field.
func NewAggregate(scope, label string, source Source) (Aggregate, error) {
	if scope == "" || label == "" || strings.ContainsRune(scope, 0) || strings.ContainsRune(label, 0) {
		return Aggregate{}, fmt.Errorf("aggregate requires nonempty scope and label without NUL")
	}
	if source.Standard == "" || source.Version == "" || source.Clause == "" || source.Table == "" {
		return Aggregate{}, fmt.Errorf("aggregate requires a source citation")
	}
	for _, part := range []string{source.Standard, source.Version, source.Clause, source.Table} {
		if strings.ContainsRune(part, 0) {
			return Aggregate{}, fmt.Errorf("aggregate source cannot contain NUL")
		}
	}
	label = key(label)
	if label == "" {
		return Aggregate{}, fmt.Errorf("aggregate requires a field label")
	}
	prefix := "\x00aggregate\x00" + source.Standard + "\x00" + source.Version + "\x00" + source.Clause + "\x00" + source.Table + "\x00" + scope + "\x00" + label
	return Aggregate{label: label, count: prefix + "\x00count", maximum: prefix + "\x00max", population: prefix + "\x00population", source: source}, nil
}

// ResetAggregate starts an empty aggregate in this reader only. A reset in a
// discarded trial is discarded too. A zero Aggregate is refused.
func (r *Reader) ResetAggregate(a Aggregate) error {
	if a.count == "" {
		return r.expressionFailure(missingAggregate().err)
	}
	r.Set(a.count, 0)
	r.Set(a.maximum, 0)
	r.Set(a.population, 0)
	return nil
}

// ResetAggregate starts an empty aggregate in this writer only.
func (w *Writer) ResetAggregate(a Aggregate) error {
	if a.count == "" {
		return w.expressionFailure(missingAggregate().err)
	}
	w.Set(a.count, 0)
	w.Set(a.maximum, 0)
	w.Set(a.population, 0)
	return nil
}

func missingAggregate() ExpressionValue {
	return functionError(Source{Standard: "TS 44.060", Version: "19.0.0", Clause: "11.2.21", Table: "p17216"}, "missing scoped aggregate", ErrContextRequired)
}

func (p *expression) aggregateBinding(name string) (uint64, bool) {
	if p.reader != nil {
		return p.reader.binding(name)
	}
	n, ok := p.vars[name]
	return n, ok
}

func (p *expression) aggregateNext(a Aggregate, value uint64) (uint64, uint64, uint64, error) {
	if a.count == "" {
		return 0, 0, 0, missingAggregate().err
	}
	count, initialized := p.aggregateBinding(a.count)
	maximum, haveMaximum := p.aggregateBinding(a.maximum)
	population, havePopulation := p.aggregateBinding(a.population)
	if !initialized || !haveMaximum || !havePopulation {
		return 0, 0, 0, functionError(a.source, "aggregate has not been reset", ErrContextRequired).err
	}
	add := uint64(bits.OnesCount64(value))
	if count >= maxBits || population > maxBits-add {
		return 0, 0, 0, functionError(a.source, "aggregate count or population outside limits", nil).err
	}
	// One record is at most 64 bitmap bits. Both counters are checked against
	// maxBits before addition; the maximum is kept as uint64 until evaluated.
	return count + 1, max(maximum, value), population + add, nil
}

// RecordAggregate records one repeated field's value. Bitmap population is the
// sum of set bits in each record (e.g. disjoint carriers), never its numeric value.
// Values remain uint64 so population supports every bit; maximum evaluation
// separately refuses values above the expression width limit. Reset is required.
func (r *Reader) RecordAggregate(a Aggregate, value uint64) error {
	p := expression{vars: r.vars, reader: r}
	count, maximum, population, err := p.aggregateNext(a, value)
	if err != nil {
		return r.expressionFailure(err)
	}
	r.Set(a.count, count)
	r.Set(a.maximum, maximum)
	r.Set(a.population, population)
	return nil
}

// RecordAggregate is the writer counterpart of Reader.RecordAggregate.
func (w *Writer) RecordAggregate(a Aggregate, value uint64) error {
	p := expression{vars: w.vars}
	count, maximum, population, err := p.aggregateNext(a, value)
	if err != nil {
		return w.expressionFailure(err)
	}
	w.Set(a.count, count)
	w.Set(a.maximum, maximum)
	w.Set(a.population, population)
	return nil
}

func (p *expression) aggregateValue(a Aggregate, operation string) ExpressionValue {
	if a.count == "" {
		return missingAggregate()
	}
	count, initialized := p.aggregateBinding(a.count)
	if !initialized {
		return functionError(a.source, "aggregate has not been reset", ErrContextRequired)
	}
	name := a.count
	switch operation {
	case "maximum":
		if count == 0 {
			// TS 44.060 V19.0.0 §11.2.21 p17216 does not define the
			// maximum of the empty ** 0 list. Refuse until specified.
			return functionError(a.source, "empty maximum refused (TS 44.060 §11.2.21 p17216)", nil)
		}
		name = a.maximum
	case "population":
		name = a.population
	}
	n, ok := p.aggregateBinding(name)
	if !ok || n > maxBits {
		return functionError(a.source, "aggregate value outside limits or missing", nil)
	}
	return ExpressionValue{n: int64(n)}
}

// ExpressionMaximum evaluates the enclosing list maximum, refusing empty lists.
func (r *Reader) ExpressionMaximum(a Aggregate) ExpressionValue {
	p := expression{vars: r.vars, reader: r}
	return p.aggregateValue(a, "maximum")
}

// ExpressionMaximum is the writer counterpart of Reader.ExpressionMaximum.
func (w *Writer) ExpressionMaximum(a Aggregate) ExpressionValue {
	p := expression{vars: w.vars}
	return p.aggregateValue(a, "maximum")
}

// ExpressionAggregateCount returns the number of records since the last reset.
func (r *Reader) ExpressionAggregateCount(a Aggregate) ExpressionValue {
	p := expression{vars: r.vars, reader: r}
	return p.aggregateValue(a, "count")
}

// ExpressionAggregateCount is the writer counterpart of Reader.ExpressionAggregateCount.
func (w *Writer) ExpressionAggregateCount(a Aggregate) ExpressionValue {
	p := expression{vars: w.vars}
	return p.aggregateValue(a, "count")
}

// ExpressionAggregatePopulation returns the sum of set bits since the last reset.
// Source-bound generator rules select global N or the current TBF's M; this
// primitive does not infer missing prior configuration or a previous TBF's state.
func (r *Reader) ExpressionAggregatePopulation(a Aggregate) ExpressionValue {
	p := expression{vars: r.vars, reader: r}
	return p.aggregateValue(a, "population")
}

// ExpressionAggregatePopulation is the writer counterpart of Reader.ExpressionAggregatePopulation.
func (w *Writer) ExpressionAggregatePopulation(a Aggregate) ExpressionValue {
	p := expression{vars: w.vars}
	return p.aggregateValue(a, "population")
}
