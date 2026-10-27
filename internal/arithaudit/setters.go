package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// checkGeneratedCountSetter proves the new setter arithmetic independently
// of the count's protocol constraints. The only input is a slice length;
// subtraction is guarded before conversion and narrowing has a literal bound.
// The generator is csn1/gen/count_setters.go, countSetters.
func checkGeneratedCountSetter(fn *ast.FuncDecl, expression string) error {
	fail := func() error { return fmt.Errorf("unverified generated count setter %s: %s", fn.Name.Name, expression) }
	var initial *ast.AssignStmt
	var offset uint64
	maxCount := ^uint64(0)
	var countBound token.Pos
	guardedOffset := false
	normalizations := map[*ast.AssignStmt]bool{}
	for _, statement := range fn.Body.List {
		if initial != nil {
			if guard, ok := statement.(*ast.IfStmt); ok {
				if normalization := emptyListNormalization(guard); normalization != nil {
					normalizations[normalization] = true
				}
			}
		}

		if guard, ok := statement.(*ast.IfStmt); ok && guard.Init == nil && guard.Else == nil && returnsError(guard.Body) {
			condition, ok := guard.Cond.(*ast.BinaryExpr)
			if !ok {
				continue
			}
			literal, ok := condition.Y.(*ast.BasicLit)
			if !ok {
				continue
			}
			bound, err := strconv.ParseUint(literal.Value, 0, 64)
			if err != nil {
				continue
			}
			if initial == nil && condition.Op == token.LSS && isSliceLength(condition.X, "items") {
				offset = bound
				guardedOffset = true
			}
			if initial != nil && condition.Op == token.GTR && isName(condition.X, "count") {
				if bound <= maxCount {
					maxCount = bound
					countBound = guard.End()
				}
			}
		}
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || !isName(assignment.Lhs[0], "count") {
			continue
		}
		if initial != nil || assignment.Tok != token.DEFINE || len(assignment.Rhs) != 1 {
			return fail()
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || !isName(call.Fun, "uint64") || len(call.Args) != 1 {
			return fail()
		}
		subtract, ok := call.Args[0].(*ast.BinaryExpr)
		if !ok || subtract.Op != token.SUB || !isSliceLength(subtract.X, "items") {
			return fail()
		}
		literal, ok := subtract.Y.(*ast.BasicLit)
		if !ok {
			return fail()
		}
		k, err := strconv.ParseUint(literal.Value, 0, 64)
		if err != nil || k != 0 && (!guardedOffset || offset != k) {
			return fail()
		}
		offset = k
		initial = assignment
	}
	if initial == nil {
		return fail()
	}
	invalid := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.AssignStmt:
			if n == initial || normalizations[n] {
				return true
			}
			for _, lhs := range n.Lhs {
				if isName(lhs, "count") || isName(lhs, "items") {
					invalid = true
				}
			}
		case *ast.CallExpr:
			if cast, ok := n.Fun.(*ast.Ident); ok && len(n.Args) == 1 && isName(n.Args[0], "count") {
				var maximum uint64
				switch cast.Name {
				case "uint8":
					maximum = 1<<8 - 1
				case "uint16":
					maximum = 1<<16 - 1
				case "uint32":
					maximum = 1<<32 - 1
				default:
					return true
				}
				if maxCount > maximum || countBound == token.NoPos || countBound >= n.Pos() {
					invalid = true
				}
			}
		case *ast.IncDecStmt:
			if isName(n.X, "count") || isName(n.X, "items") {
				invalid = true
			}
		case *ast.ForStmt, *ast.RangeStmt, *ast.GoStmt, *ast.DeferStmt, *ast.BranchStmt:
			invalid = true
		}
		return !invalid
	})
	if invalid {
		return fail()
	}
	simple := generatedWhitespace.ReplaceAllString(expression, "")
	subtract := "len(items)-" + strconv.FormatUint(offset, 10)
	if simple == subtract || simple == "uint64("+subtract+")" {
		return nil
	}
	for _, conversion := range []struct {
		name    string
		maximum uint64
	}{{"uint8", 1<<8 - 1}, {"uint16", 1<<16 - 1}, {"uint32", 1<<32 - 1}} {
		if simple == conversion.name+"(count)" && maxCount <= conversion.maximum {
			return nil
		}
	}
	if strings.HasPrefix(simple, "uint64(") && simple == "uint64(count)" {
		return nil
	}
	return fail()
}

func isSliceLength(expression ast.Expr, name string) bool {
	call, ok := expression.(*ast.CallExpr)
	return ok && isName(call.Fun, "len") && len(call.Args) == 1 && isName(call.Args[0], name)
}

func returnsError(body *ast.BlockStmt) bool {
	if len(body.List) != 1 {
		return false
	}
	result, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		return false
	}
	call, ok := result.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isName(selector.X, "fmt") && selector.Sel.Name == "Errorf"
}

// Changing a zero-length slice to nil preserves every later len(items) bound.
func emptyListNormalization(guard *ast.IfStmt) *ast.AssignStmt {
	if guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
		return nil
	}
	condition, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.EQL || !isSliceLength(condition.X, "items") {
		return nil
	}
	zero, ok := condition.Y.(*ast.BasicLit)
	if !ok || zero.Value != "0" {
		return nil
	}
	assignment, ok := guard.Body.List[0].(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || !isName(assignment.Lhs[0], "items") || !isName(assignment.Rhs[0], "nil") {
		return nil
	}
	return assignment
}
