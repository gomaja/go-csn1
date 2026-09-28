// Package arithaudit inventories host-integer operations at decoder boundaries.
package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Finding struct {
	File       string
	Line       int
	Column     int
	Kind       string
	Expression string
}

func (f Finding) Key() string {
	return fmt.Sprintf("%s:%d:%d:%s:%s", f.File, f.Line, f.Column, f.Kind, f.Expression)
}

// Signature is stable when another generated function shifts a hit's line.
func (f Finding) Signature() string {
	return fmt.Sprintf("%s:%s:%s", f.File, f.Kind, f.Expression)
}

type Classification struct {
	Count  int
	Reason string
}

func CheckAllowlist(findings []Finding, allowed map[string]Classification) error {
	seen := make(map[string]int, len(allowed))
	for _, finding := range findings {
		signature := finding.Signature()
		classification := allowed[signature]
		reason := strings.TrimSpace(classification.Reason)
		if reason == "" {
			return fmt.Errorf("unclassified integer operation %s", finding.Key())
		}
		seen[signature]++
		if !strings.HasPrefix(reason, "BOUNDED: ") && !strings.HasPrefix(reason, "FIXED: ") && !strings.HasPrefix(reason, "FINDING: ") {
			return fmt.Errorf("invalid arithmetic classification %s: %s", finding.Key(), reason)
		}
		if !checkCitation.MatchString(reason) {
			return fmt.Errorf("arithmetic classification lacks guard or finding citation %s: %s", finding.Key(), reason)
		}
	}
	for signature, classification := range allowed {
		if classification.Count <= 0 || seen[signature] != classification.Count {
			return fmt.Errorf("arithmetic classification count for %s is %d, want %d", signature, seen[signature], classification.Count)
		}
	}
	return nil
}

var integerConversions = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "byte": true, "rune": true,
}

var wireTemplateName = regexp.MustCompile(`\b(offset|length|count|index|bitPos|bits|total|n)\b`)
var checkCitation = regexp.MustCompile(`[a-zA-Z0-9_./-]+\.go:[0-9]+`)

func ScanSource(name string, source []byte) ([]Finding, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, source, 0)
	if err != nil {
		return nil, err
	}
	info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue), Uses: make(map[*ast.Ident]types.Object)}
	// The scanner can inspect a single file from a multi-file module. Missing
	// sibling declarations are tolerated; fully resolved expressions still use
	// go/types, and unresolved ones are treated conservatively below.
	checker := types.Config{Importer: importer.Default(), Error: func(error) {}}
	_, _ = checker.Check(file.Name.Name, fset, []*ast.File{file}, info)
	var findings []Finding
	add := func(node ast.Node, kind string) {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return
		}
		position := fset.Position(node.Pos())
		findings = append(findings, Finding{File: name, Line: position.Line, Column: position.Column, Kind: kind, Expression: buf.String()})
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		decoder := isDecoderName(fn.Name.Name)
		// Runtime methods carry wire positions in their receiver even when they
		// have no explicit parameters; other helpers may receive wire lengths.
		runtimeHelper := strings.Contains(name, "runtime/") && (fn.Recv != nil || fn.Type.Params != nil && len(fn.Type.Params.List) != 0)
		// Generated encoders can receive a value produced by a decoder, so do
		// not infer safety from a Marshal/Encode name.
		generated := strings.HasPrefix(name, "telecom/") || strings.Contains(name, "/generated.go")
		possiblyWireDerived := decoder || runtimeHelper || generated
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.AssignStmt:
				if value.Tok != token.ADD_ASSIGN && value.Tok != token.SUB_ASSIGN && value.Tok != token.MUL_ASSIGN && value.Tok != token.SHL_ASSIGN {
					break
				}
				if len(value.Lhs) != 1 || !integerExpr(info, value.Lhs[0]) {
					break
				}
				if len(value.Rhs) != 0 && hasWireSource(value.Rhs[0]) || possiblyWireDerived {
					add(value, "arithmetic")
				}
			case *ast.IncDecStmt:
				if integerExpr(info, value.X) && (hasWireSource(value.X) || possiblyWireDerived) {
					add(value, "arithmetic")
				}
			case *ast.BinaryExpr:
				if value.Op != token.ADD && value.Op != token.SUB && value.Op != token.MUL && value.Op != token.SHL {
					break
				}
				if !integerExpr(info, value) {
					break
				}
				if hasWireSource(value) || possiblyWireDerived && hasNonconstantOperand(value) {
					add(value, "arithmetic")
				}
			case *ast.CallExpr:
				ident, ok := value.Fun.(*ast.Ident)
				if !ok || !integerConversions[ident.Name] || len(value.Args) != 1 {
					break
				}
				if hasWireSource(value.Args[0]) || possiblyWireDerived && hasNonconstantOperand(value.Args[0]) {
					add(value, "conversion")
				}
			case *ast.BasicLit:
				if value.Kind != token.STRING || !strings.Contains(name, "gen/") {
					break
				}
				literal, err := strconv.Unquote(value.Value)
				if err != nil {
					break
				}
				for _, line := range strings.Split(literal, "\n") {
					line = strings.ReplaceAll(strings.TrimSpace(line), "\t", " ")
					wireLine := strings.Contains(line, "len(") || strings.Contains(line, "cap(") || wireTemplateName.MatchString(line)
					arithmeticLine := strings.ContainsAny(line, "+-*") || strings.Contains(line, "<<")
					if wireLine && arithmeticLine {
						position := fset.Position(value.Pos())
						findings = append(findings, Finding{File: name, Line: position.Line, Column: position.Column, Kind: "template", Expression: line})
					}
				}
			}
			return true
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Column != findings[j].Column {
			return findings[i].Column < findings[j].Column
		}
		if findings[i].Kind != findings[j].Kind {
			return findings[i].Kind < findings[j].Kind
		}
		return findings[i].Expression < findings[j].Expression
	})
	return findings, nil
}

func isDecoderName(name string) bool {
	for _, prefix := range []string{"Decode", "Unmarshal", "Read", "Parse", "decode", "unmarshal", "read", "parse"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func integerExpr(info *types.Info, expr ast.Expr) bool {
	if tv, ok := info.Types[expr]; ok && tv.Type != nil {
		basic, ok := tv.Type.Underlying().(*types.Basic)
		return ok && basic.Info()&types.IsInteger != 0
	}
	return true // unresolved imports or sibling declarations must not hide a hit
}

func hasWireSource(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if ident, ok := call.Fun.(*ast.Ident); ok && (ident.Name == "len" || ident.Name == "cap" || isDecoderName(ident.Name)) {
				found = true
			}
		}
		return !found
	})
	return found
}

func hasNonconstantOperand(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if ident, ok := node.(*ast.Ident); ok && ident.Name != "nil" && ident.Name != "true" && ident.Name != "false" {
			found = true
		}
		return !found
	})
	return found
}
