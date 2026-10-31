package runtime

import "fmt"

// EvalTokens evaluates already-tokenized width syntax using Eval's parser and
// dynamic label resolution. Tokens must use Eval's lexical grammar: operators
// and parentheses are separate tokens, with whitespace omitted. Within val(),
// unspaced punctuation belongs to the label token (for example "A/N"), while
// operators surrounded by whitespace in the source are separate tokens.
// For compatibility, published tokens that split hyphenated names (such as
// N_E, -, UTRAN) are rejoined through the bindings. A spaced val(N_E - UTRAN)
// token array therefore still selects a bound N_E-UTRAN label; strings retain
// the lexical subtraction rule. New labels should be emitted as single tokens.
// The slice is borrowed only for this call and is neither modified nor retained.
// TS 24.007 V20.0.0 Annex B defines the CSN.1 expression syntax; printed labels
// remain dynamically resolved, including longest matches and ambiguity errors.
func (r *Reader) EvalTokens(tokens []string) (int, error) {
	n, err := evalTokens(tokens, r.vars, r)
	if err != nil {
		return 0, r.fail(InvalidSchema, err.Error())
	}
	return n, nil
}

// EvalTokens evaluates already-tokenized width syntax using Eval's parser and
// dynamic label resolution. It has the same token and borrowing contract as
// Reader.EvalTokens and preserves Writer.Eval's error behavior.
func (w *Writer) EvalTokens(tokens []string) (int, error) {
	return evalTokens(tokens, w.vars, nil)
}

func evalTokens(tokens []string, vars map[string]uint64, reader *Reader) (int, error) {
	p := expression{tokens: tokens, vars: vars, reader: reader, pretokenized: true}
	n, err := p.sum()
	if err != nil {
		return 0, err
	}
	if p.position != len(tokens) {
		return 0, fmt.Errorf("unexpected expression token %q", tokens[p.position])
	}
	return (ExpressionValue{n: n}).width()
}
