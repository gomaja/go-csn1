package arithaudit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// CheckGeneratedChoiceCounters proves the emitted choice counters are local
// zero-based sums over a finite number of static alternatives. The generator
// emits one conditional increment per arm: csn1/gen/direct.go:1023 `matches++`
// and csn1/gen/direct.go:1037 `if v.%s!=nil{count++};`.
func CheckGeneratedChoiceCounters(name string, source []byte) (int, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, name, source, 0)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		hasChoiceCounter := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			stmt, ok := node.(*ast.IncDecStmt)
			if !ok {
				return true
			}
			id, ok := stmt.X.(*ast.Ident)
			hasChoiceCounter = hasChoiceCounter || ok && (id.Name == "count" || id.Name == "matches")
			return true
		})
		if !hasChoiceCounter {
			continue
		}
		initialized := map[string]bool{}
		increments := map[string]int{}
		var loops []ast.Node
		var fault error
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.ForStmt, *ast.RangeStmt:
				loops = append(loops, node)
			case *ast.AssignStmt:
				for i, lhs := range stmt.Lhs {
					id, ok := lhs.(*ast.Ident)
					if !ok || id.Name != "count" && id.Name != "matches" {
						continue
					}
					if stmt.Tok != token.DEFINE || i >= len(stmt.Rhs) || !isZeroLiteral(stmt.Rhs[i]) {
						fault = fmt.Errorf("%s: choice counter %s has nonzero assignment", name, id.Name)
						return false
					}
					initialized[id.Name] = true
				}
			case *ast.IncDecStmt:
				id, ok := stmt.X.(*ast.Ident)
				if !ok || id.Name != "count" && id.Name != "matches" {
					break
				}
				if stmt.Tok != token.INC || !initialized[id.Name] {
					fault = fmt.Errorf("%s: choice counter %s is not zero-based", name, id.Name)
					return false
				}
				for _, loop := range loops {
					if loop.Pos() <= stmt.Pos() && stmt.End() <= loop.End() {
						fault = fmt.Errorf("%s: choice counter %s increments in a loop", name, id.Name)
						return false
					}
				}
				increments[id.Name]++
				total++
			}
			return fault == nil
		})
		if fault != nil {
			return 0, fault
		}
		for counter, n := range increments {
			if n > 64 {
				return 0, fmt.Errorf("%s: choice counter %s has %d static arms", name, counter, n)
			}
		}
	}
	return total, nil
}

func isZeroLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}
