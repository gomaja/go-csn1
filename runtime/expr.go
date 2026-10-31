package runtime

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const maxBits = 1 << 20

type expression struct {
	tokens     []string
	position   int
	vars       map[string]uint64
	reader     *Reader
	syntaxOnly bool
	context    *ExpressionContext
}

// TS 44.018 V19.0.0 §9.1.54 tables 9.1.54.1a and 9.1.54.1b
// define the FDD p(n) and TDD q(m) cell-information bit widths.
// Both tables set the remaining five-bit count values to zero.
var cellInformationWidths = map[string][]int64{
	"p": {0, 10, 19, 28, 36, 44, 52, 60, 67, 74, 81, 88, 95, 102, 109, 116, 122},
	"q": {0, 9, 17, 25, 32, 39, 46, 53, 59, 65, 71, 77, 83, 89, 95, 101, 106, 111, 116, 121, 126},
}

func eval(raw string, vars map[string]uint64) (int, error) { return evalExpression(raw, vars, nil) }

func evalReader(raw string, r *Reader) (int, error) { return evalExpression(raw, r.vars, r) }

func evalExpression(raw string, vars map[string]uint64, reader *Reader) (int, error) {
	tokens, err := expressionTokens(raw)
	if err != nil {
		return 0, err
	}
	p := expression{tokens: tokens, vars: vars, reader: reader}
	n, err := p.sum()
	if err != nil {
		return 0, err
	}
	if p.position != len(tokens) {
		return 0, fmt.Errorf("unexpected expression token %q", tokens[p.position])
	}
	if n < 0 || n > maxBits {
		return 0, fmt.Errorf("width %d outside 0..%d", n, maxBits)
	}
	return int(n), nil
}

// ValidateExpression checks width and count syntax using the same parser as
// Reader.Eval and Reader.EvalTokens. Labels remain dynamic: no bindings,
// ambiguity, arithmetic limits or division results are evaluated here.
// TS 24.007 V20.0.0 Annex B §§B.1.3, B.1.4 and B.2.1 define bit widths,
// labelled values and repetition expressions.
func ValidateExpression(raw string) error {
	tokens, err := expressionTokens(raw)
	if err != nil {
		return err
	}
	p := expression{tokens: tokens, syntaxOnly: true}
	if _, err := p.sum(); err != nil {
		return err
	}
	if p.position != len(tokens) {
		return fmt.Errorf("unexpected expression token %q", tokens[p.position])
	}
	return nil
}

func expressionTokens(raw string) ([]string, error) {
	var tokens []string
	depth, valDepth := 0, 0
	for i := 0; i < len(raw); {
		r := rune(raw[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if r == '(' || r == ')' || expressionOperator(raw, i, valDepth != 0) {
			switch r {
			case '(':
				depth++
				if valDepth == 0 && len(tokens) > 0 && tokens[len(tokens)-1] == "val" {
					valDepth = depth
				}
			case ')':
				if depth == valDepth {
					valDepth = 0
				}
				depth--
			}
			tokens = append(tokens, string(r))
			i++
			continue
		}
		start := i
		for i < len(raw) {
			r := rune(raw[i])
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' &&
				(valDepth == 0 || !strings.ContainsRune("+-*/", r) || expressionOperator(raw, i, true)) {
				break
			}
			i++
		}
		if i == start {
			return nil, fmt.Errorf("invalid width expression %q", raw)
		}
		tokens = append(tokens, raw[start:i])
	}
	return tokens, nil
}

// TS 44.060 V19.0.0 §11.2.6a p14465 prints EPD A/N as label text;
// §§11.2.3a and 11.2.9 pp13726,15479 print N_E-UTRAN + 1.
// Within val(), only operators surrounded by whitespace are arithmetic.
// Outside val(), the ordinary expression operator grammar is unchanged.
func expressionOperator(raw string, i int, inVal bool) bool {
	if i < 0 || i >= len(raw) {
		return false
	}
	if strings.IndexByte("+-*/", raw[i]) < 0 {
		return false
	}
	if !inVal {
		return true
	}
	if i == 0 || i+1 >= len(raw) {
		return false
	}
	return strings.IndexByte(" \t\r\n\v\f", raw[i-1]) >= 0 && strings.IndexByte(" \t\r\n\v\f", raw[i+1]) >= 0
}

func (p *expression) peek() string {
	if p.position >= len(p.tokens) {
		return ""
	}
	return p.tokens[p.position]
}
func (p *expression) take() string {
	v := p.peek()
	if v != "" {
		p.position++
	}
	return v
}

func (p *expression) sum() (int64, error) {
	n, err := p.product()
	if err != nil {
		return 0, err
	}
	for p.peek() == "+" || p.peek() == "-" {
		op := p.take()
		v, err := p.product()
		if err != nil {
			return 0, err
		}
		if p.syntaxOnly {
			continue
		}
		if op == "+" {
			if n > maxBits-v || n < -maxBits-v {
				return 0, fmt.Errorf("expression result outside limits")
			}
			n += v
		} else {
			if n > maxBits+v || n < -maxBits+v {
				return 0, fmt.Errorf("expression result outside limits")
			}
			n -= v
		}
		if n < -maxBits || n > maxBits {
			return 0, fmt.Errorf("expression result outside limits")
		}
	}
	return n, nil
}

func (p *expression) product() (int64, error) {
	n, err := p.primary()
	if err != nil {
		return 0, err
	}
	if n < -maxBits || n > maxBits {
		return 0, fmt.Errorf("expression operand outside limits")
	}
	for p.peek() == "*" || p.peek() == "/" {
		op := p.take()
		v, err := p.primary()
		if err != nil {
			return 0, err
		}
		if v < -maxBits || v > maxBits {
			return 0, fmt.Errorf("expression operand outside limits")
		}
		if p.syntaxOnly {
			continue
		}
		if op == "*" {
			// Operands are bounded to +/-maxBits before this operation;
			// maxBits² fits in int64. Reject the product at each step.
			n *= v
		} else {
			if v == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			n /= v
		}
		if n < -maxBits || n > maxBits {
			return 0, fmt.Errorf("expression result outside limits")
		}
	}
	return n, nil
}

func (p *expression) primary() (int64, error) {
	t := p.take()
	if t == "" {
		return 0, fmt.Errorf("missing expression operand")
	}
	if t == "(" {
		n, err := p.sum()
		if err != nil {
			return 0, err
		}
		if p.take() != ")" {
			return 0, fmt.Errorf("unclosed expression")
		}
		return n, nil
	}
	if t == "val" {
		if p.take() != "(" {
			return 0, fmt.Errorf("val needs (")
		}
		n, err := p.sum()
		if err != nil {
			return 0, err
		}
		if p.take() != ")" {
			return 0, fmt.Errorf("unclosed val")
		}
		return n, nil
	}
	if t == "max" && p.peek() == "(" {
		return p.maximum()
	}
	if t == "p" || t == "q" || t == "expanded_EPDAN_extension_length" && p.peek() == "(" {
		return p.sourceFunction(t)
	}
	if t == "-" {
		n, err := p.primary()
		return -n, err
	}
	if n, err := strconv.ParseInt(t, 10, 64); err == nil {
		if p.syntaxOnly {
			return 0, nil
		}
		return n, nil
	}
	return p.namedValue(t)
}

func (p *expression) namedValue(t string) (int64, error) {
	// TS 44.018 V19.0.0 §10.5.2.16 references printed field labels
	// containing spaces inside val(...). Match the longest bound label so
	// arithmetic after it remains a separate expression.
	// Syntax checking accepts any possible longest label without constructing
	// each prefix or consulting bindings. Evaluation retains the exact dynamic
	// matching and ambiguity rules below.
	if p.syntaxOnly && (t == ")" || t == "+" || t == "*" || t == "/") {
		return 0, fmt.Errorf("missing expression operand before %q", t)
	}
	name := t
	last := p.position
	var value uint64
	found := false
	if n, ok, err := p.resolveVar(t); err != nil {
		return 0, err
	} else if ok {
		value, found = n, true
	}
	for i := p.position; i < len(p.tokens); i++ {
		part := p.tokens[i]
		if part == "(" || part == ")" || part == "+" || part == "*" || part == "/" {
			break
		}
		if part == "-" {
			if i+1 >= len(p.tokens) || p.tokens[i+1] == "(" || p.tokens[i+1] == ")" {
				break
			}
			if !p.syntaxOnly {
				name += "-" + p.tokens[i+1]
			}
			i++
		} else {
			if !p.syntaxOnly {
				name += " " + part
			}
		}
		// Labels cannot take arguments. Leave a following call to the
		// expression parser instead of absorbing its name into a label.
		if p.syntaxOnly && i+1 < len(p.tokens) && p.tokens[i+1] == "(" {
			break
		}
		n, ok, err := p.resolveVar(name)
		if err != nil {
			return 0, err
		}
		if ok {
			value, found, last = n, true, i+1
		}
	}
	if found {
		p.position = last
		if value > maxBits {
			return 0, fmt.Errorf("width variable %q outside 0..%d", t, maxBits)
		}
		return int64(value), nil
	}
	return 0, fmt.Errorf("unresolved width variable %q", t)
}

func (p *expression) resolveVar(name string) (uint64, bool, error) {
	if p.syntaxOnly {
		return 0, true, nil
	}
	canonical := key(name)
	// Both plain maps and trial overlays prefer exact canonical names and
	// reject ambiguous compact spellings through the same resolver.
	if p.reader != nil {
		return p.reader.resolveBinding(name, canonical)
	}
	plain := Reader{vars: p.vars}
	return plain.resolveBinding(name, canonical)
}
