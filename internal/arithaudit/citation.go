package arithaudit

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	operandCitation   = regexp.MustCompile(`\boperand=([A-Za-z_][A-Za-z_0-9.]*)\b`)
	guardCitation     = regexp.MustCompile(`\bguard=([A-Za-z_0-9./-]+\.go):([0-9]+)\b`)
	postCitation      = regexp.MustCompile(`\bpost=([A-Za-z_0-9./-]+\.go):([0-9]+)\b`)
	calleeCitation    = regexp.MustCompile(`\bcallee=([A-Za-z_][A-Za-z_0-9]*)\b`)
	intrinsicCitation = regexp.MustCompile(`\bintrinsic=([A-Za-z_][A-Za-z_0-9]*)\b`)
)

type citationFile struct {
	set    *token.FileSet
	file   *ast.File
	lines  []string
	source []byte
}

type citationChecker struct {
	root  string
	files map[string]citationFile
}

// CheckBoundCitation checks citation locality and operand identity. This is a
// lightweight dominance heuristic, not a control-flow proof: reviewers must
// still verify that the cited comparison rejects the unsafe path.
func CheckBoundCitation(root string, finding Finding, reason string) error {
	return (&citationChecker{root: root, files: make(map[string]citationFile)}).check(finding, reason)
}

func (c *citationChecker) source(path string) (citationFile, error) {
	if file, ok := c.files[path]; ok {
		return file, nil
	}
	contents, err := os.ReadFile(filepath.Join(c.root, filepath.FromSlash(path)))
	if err != nil {
		return citationFile{}, err
	}
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, contents, 0)
	if err != nil {
		return citationFile{}, err
	}
	result := citationFile{set: set, file: file, lines: strings.Split(string(contents), "\n"), source: contents}
	c.files[path] = result
	return result, nil
}

func functionAt(file citationFile, line int) *ast.FuncDecl {
	for _, decl := range file.file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && file.set.Position(fn.Pos()).Line <= line && line <= file.set.Position(fn.End()).Line {
			return fn
		}
	}
	return nil
}

func (c *citationChecker) check(finding Finding, reason string) error {
	var failures []error
	for _, part := range strings.Split(reason, ";") {
		if err := c.checkOne(finding, part); err == nil {
			return nil
		} else {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (c *citationChecker) checkOne(finding Finding, reason string) error {
	operands := operandCitation.FindStringSubmatch(reason)
	guard := guardCitation.FindStringSubmatch(reason)
	if len(operands) == 0 || len(guard) == 0 {
		return fmt.Errorf("bound citation needs operand and guard for %s", finding.Key())
	}
	line, err := strconv.Atoi(guard[2])
	if err != nil || line <= 0 {
		return fmt.Errorf("invalid bound guard for %s", finding.Key())
	}
	targetFile, err := c.source(finding.File)
	if err != nil {
		return err
	}
	targetFn := functionAt(targetFile, finding.Line)
	if targetFn == nil {
		return fmt.Errorf("finding outside function: %s", finding.Key())
	}
	guardFile, err := c.source(guard[1])
	if err != nil {
		return err
	}
	guardFn := functionAt(guardFile, line)
	if guardFn == nil {
		return fmt.Errorf("guard outside function: %s:%d", guard[1], line)
	}
	if intrinsic := intrinsicCitation.FindStringSubmatch(reason); len(intrinsic) != 0 {
		if guard[1] != finding.File || guardFn != targetFn || line > finding.Line || line > len(guardFile.lines) {
			return fmt.Errorf("intrinsic proof is not at finding: %s", finding.Key())
		}
		sourceLine := guardFile.lines[line-1]
		switch intrinsic[1] {
		case "mod8":
			if line != finding.Line || !regexp.MustCompile(`%\s*8\b`).MatchString(finding.Expression) || !regexp.MustCompile(`%\s*8\b`).MatchString(sourceLine) || !strings.Contains(sourceLine, operands[1]) {
				return fmt.Errorf("modulo-eight proof does not compare operand %q: %s", operands[1], finding.Key())
			}
		case "lenMinusOne":
			if line != finding.Line || !regexp.MustCompile(`len\([^)]*\)\s*-\s*1\b`).MatchString(finding.Expression) || !strings.Contains(finding.Expression, operands[1]) || !regexp.MustCompile(`len\([^)]*\)\s*-\s*1\b`).MatchString(sourceLine) {
				return fmt.Errorf("length-minus-one proof does not match %s", finding.Key())
			}
		case "constant":
			if line != finding.Line || strings.ReplaceAll(finding.Expression, " ", "") != "maxBits+8" || !strings.Contains(strings.ReplaceAll(sourceLine, " ", ""), "maxBits+8") || operands[1] != "maxBits" {
				return fmt.Errorf("constant proof does not match %s", finding.Key())
			}
		case "maskOne":
			if line != finding.Line || !regexp.MustCompile(`&\s*1\b`).MatchString(finding.Expression) || !regexp.MustCompile(`&\s*1\b`).MatchString(sourceLine) || !strings.Contains(finding.Expression, operands[1]) {
				return fmt.Errorf("one-bit mask proof does not match %s", finding.Key())
			}
		case "staticDescriptors":
			if finding.File != "registry/registry.go" || line != finding.Line || !strings.HasPrefix(finding.Expression, "len(classmark.Definitions()) + len(msrac.Definitions())") || !strings.Contains(sourceLine, "make([]runtime.Descriptor, 0,") {
				return fmt.Errorf("static descriptor proof does not match %s", finding.Key())
			}
		case "uint8Widen":
			if line != finding.Line || finding.Expression != "uint64(b)" || !strings.Contains(sourceLine, "uint64(b)") {
				return fmt.Errorf("uint8 widening proof does not match %s", finding.Key())
			}
		case "hostIntWiden":
			if line != finding.Line || finding.Expression != "int64(len(table))" || !strings.Contains(sourceLine, finding.Expression) {
				return fmt.Errorf("host int widening proof does not match %s", finding.Key())
			}
		case "bitAccumulator":
			if finding.Expression != "value << 1" || operands[1] != "width" || !strings.Contains(sourceLine, "width > 64") || !strings.Contains(finding.Expression, "value") {
				return fmt.Errorf("bit accumulator proof does not match %s", finding.Key())
			}
		case "implicitSpan":
			if operands[1] != "last.Count" || !strings.Contains(sourceLine, "last.Count >= maxBits") || !strings.Contains(sourceLine, "last.At > maxBits-last.Count") || !strings.Contains(finding.Expression, "ImplicitSpans") && !strings.Contains(finding.Expression, "spans") {
				return fmt.Errorf("implicit span proof does not match %s", finding.Key())
			}
		case "matchedRange":
			if finding.Expression != "r.pos + i" || operands[1] != "r.pos" || !strings.Contains(sourceLine, "r.Check()") || !strings.Contains(sourceLine, "len(pattern) > r.Remaining()") {
				return fmt.Errorf("matched range proof does not match %s", finding.Key())
			}
		default:
			return fmt.Errorf("unknown intrinsic proof %q", intrinsic[1])
		}
		return nil
	}
	callee := calleeCitation.FindStringSubmatch(reason)
	if len(callee) == 0 {
		if guard[1] != finding.File || guardFn != targetFn || line > finding.Line {
			return fmt.Errorf("guard is not earlier in finding function: %s", finding.Key())
		}
		if !strings.Contains(finding.Expression, operands[1]) {
			return fmt.Errorf("guard operand %q absent from %s", operands[1], finding.Key())
		}
	} else {
		if guardFn.Name.Name != callee[1] || !callsFunctionBefore(targetFile, targetFn.Body, callee[1], finding) {
			return fmt.Errorf("named callee %q does not guard %s", callee[1], finding.Key())
		}
	}
	if !hasComparisonAt(guardFile, guardFn.Body, line, operands[1]) {
		return fmt.Errorf("cited line has no comparison of %q: %s:%d", operands[1], guard[1], line)
	}
	if post := postCitation.FindStringSubmatch(reason); len(post) != 0 {
		postLine, err := strconv.Atoi(post[2])
		if err != nil || post[1] != guard[1] || functionAt(guardFile, postLine) != guardFn || !hasComparisonAt(guardFile, guardFn.Body, postLine, operands[1]) {
			return fmt.Errorf("postcondition does not compare %q in named callee: %s", operands[1], finding.Key())
		}
	}
	return nil
}

func callsFunctionBefore(file citationFile, body *ast.BlockStmt, name string, finding Finding) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		position := file.set.Position(call.Pos())
		if position.Line > finding.Line || position.Line == finding.Line && finding.Column > 0 && position.Column >= finding.Column {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			found = found || fn.Name == name
		case *ast.SelectorExpr:
			found = found || fn.Sel.Name == name
		}
		return !found
	})
	return found
}

func hasComparisonAt(file citationFile, body *ast.BlockStmt, line int, operand string) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}
		var condition ast.Expr
		switch stmt := node.(type) {
		case *ast.IfStmt:
			condition = stmt.Cond
		case *ast.ForStmt:
			condition = stmt.Cond
		}
		if condition == nil || file.set.Position(condition.Pos()).Line > line || file.set.Position(condition.End()).Line < line {
			return true
		}
		ast.Inspect(condition, func(part ast.Node) bool {
			binary, ok := part.(*ast.BinaryExpr)
			if !ok || binary.Op != token.LSS && binary.Op != token.LEQ && binary.Op != token.GTR && binary.Op != token.GEQ && binary.Op != token.EQL && binary.Op != token.NEQ {
				return true
			}
			var printed bytes.Buffer
			if format.Node(&printed, file.set, binary) == nil {
				pattern := regexp.MustCompile(`(^|[^A-Za-z_0-9.])` + regexp.QuoteMeta(operand) + `($|[^A-Za-z_0-9.])`)
				found = found || pattern.MatchString(printed.String())
			}
			return !found
		})
		return !found
	})
	return found
}
