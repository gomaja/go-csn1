package runtime

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const maxBits = 1 << 20

type expression struct {
	tokens   []string
	position int
	vars     map[string]uint64
}

func eval(raw string, vars map[string]uint64) (int, error) {
	var tokens []string
	for i := 0; i < len(raw); {
		r := rune(raw[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if strings.ContainsRune("()+-*/", r) {
			tokens = append(tokens, string(r))
			i++
			continue
		}
		start := i
		for i < len(raw) && (unicode.IsLetter(rune(raw[i])) || unicode.IsDigit(rune(raw[i])) || raw[i] == '_') {
			i++
		}
		if i == start {
			return 0, fmt.Errorf("invalid width expression %q", raw)
		}
		tokens = append(tokens, raw[start:i])
	}
	p := expression{tokens: tokens, vars: vars}
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
		if op == "+" {
			n += v
		} else {
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
	for p.peek() == "*" || p.peek() == "/" {
		op := p.take()
		v, err := p.primary()
		if err != nil {
			return 0, err
		}
		if op == "*" {
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
	if t == "-" {
		n, err := p.primary()
		return -n, err
	}
	if n, err := strconv.ParseInt(t, 10, 64); err == nil {
		return n, nil
	}
	if n, ok := p.vars[key(t)]; ok {
		return int64(n), nil
	}
	return 0, fmt.Errorf("unresolved width variable %q", t)
}
