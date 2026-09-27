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
	// TS 44.018 V19.0.0 §10.5.2.16 references printed field labels
	// containing spaces inside val(...). Match the longest bound label so
	// arithmetic after it remains a separate expression.
	parts := []string{t}
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
		if part == "(" || part == ")" || part == "+" || part == "-" || part == "*" || part == "/" {
			break
		}
		parts = append(parts, part)
		n, ok, err := p.resolveVar(strings.Join(parts, " "))
		if err != nil {
			return 0, err
		}
		if ok {
			value, found, last = n, true, i+1
		}
	}
	if found {
		p.position = last
		return int64(value), nil
	}
	return 0, fmt.Errorf("unresolved width variable %q", t)
}

func (p *expression) resolveVar(name string) (uint64, bool, error) {
	canonical := key(name)
	if n, ok := p.vars[canonical]; ok {
		return n, true, nil
	}
	// Word extraction can omit spaces inside a val(...) reference while
	// preserving them in its printed field declaration. Ambiguous compact
	// spellings fail closed instead of selecting whichever map entry wins.
	compact := strings.ReplaceAll(canonical, " ", "")
	var value uint64
	found := false
	for label, n := range p.vars {
		if strings.ReplaceAll(label, " ", "") != compact {
			continue
		}
		if found {
			return 0, false, fmt.Errorf("ambiguous width variable %q", name)
		}
		value, found = n, true
	}
	return value, found, nil
}
