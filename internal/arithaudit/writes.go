package arithaudit

import (
	"go/ast"
	"go/token"
)

// writesName also treats taking an address as a write: subsequent writes
// through that alias cannot be bounded by these syntax proofs. Identifiers
// with the same spelling in nested scopes are conservatively included.
// Go specification, Assignments, IncDec statements, Address operators,
// and For statements with range clauses: https://go.dev/ref/spec.
func writesName(node ast.Node, name string) bool {
	match := func(expr ast.Expr) bool { return expr != nil && isName(ast.Unparen(expr), name) }
	switch stmt := node.(type) {
	case *ast.AssignStmt:
		for _, lhs := range stmt.Lhs {
			if match(lhs) {
				return true
			}
		}
	case *ast.IncDecStmt:
		return match(stmt.X)
	case *ast.UnaryExpr:
		return stmt.Op == token.AND && match(stmt.X)
	case *ast.RangeStmt:
		return match(stmt.Key) || match(stmt.Value)
	case *ast.ValueSpec:
		for _, id := range stmt.Names {
			if id.Name == name {
				return true
			}
		}
	}
	return false
}

func hasNameWrite(node ast.Node, name string) bool {
	found := false
	ast.Inspect(node, func(child ast.Node) bool {
		found = found || writesName(child, name)
		return !found
	})
	return found
}

// plainAssignmentValue binds a proof to its destination, rather than to an
// arbitrary call or right-hand literal. Callers separately review whether a
// multi-value source may establish a proof for its first result.
// Go specification, Assignments and Short variable declarations:
// https://go.dev/ref/spec#Assignments and #Short_variable_declarations.
func plainAssignmentValue(assign *ast.AssignStmt, name string) (ast.Expr, bool) {
	if assign.Tok != token.DEFINE && assign.Tok != token.ASSIGN ||
		len(assign.Lhs) == 0 || !isName(assign.Lhs[0], name) || len(assign.Rhs) != 1 {
		return nil, false
	}
	return assign.Rhs[0], true
}
