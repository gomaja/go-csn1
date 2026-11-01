package arithaudit

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

// checkEnclosingCountArithmetic checks each complete canonical count block.
// The block has a guarded length subtraction, a literal narrowing bound, a
// sibling equality guard and two assignments. It cannot modify the input list.
func checkEnclosingCountArithmetic(file citationFile, fn *ast.FuncDecl, f Finding) error {
	text := func(n ast.Node) string {
		var b bytes.Buffer
		if err := format.Node(&b, file.set, n); err != nil {
			return ""
		}
		return b.String()
	}
	valid := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok || file.set.Position(block.Pos()).Line > f.Line || file.set.Position(block.End()).Line < f.Line || len(block.List) != 6 && len(block.List) != 5 {
			return true
		}
		offset := len(block.List) - 5
		assignment, ok := block.List[offset].(*ast.AssignStmt)
		if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || !isName(assignment.Lhs[0], "count") || len(assignment.Rhs) != 1 {
			return true
		}
		conversion, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok || !isName(conversion.Fun, "uint64") || len(conversion.Args) != 1 {
			return true
		}
		subtract, ok := conversion.Args[0].(*ast.BinaryExpr)
		if !ok || subtract.Op != token.SUB {
			return true
		}
		length, ok := subtract.X.(*ast.CallExpr)
		if !ok || !isName(length.Fun, "len") || len(length.Args) != 1 {
			return true
		}
		access := text(length.Args[0])
		if access == "" {
			return true
		}
		// Every expression is a typed lvalue path, never a call or a side effect.
		pure := true
		ast.Inspect(length.Args[0], func(n ast.Node) bool {
			switch n.(type) {
			case *ast.CallExpr, *ast.FuncLit, *ast.SliceExpr:
				pure = false
			}
			return pure
		})
		if !pure {
			return true
		}
		set, ok := block.List[offset+3].(*ast.AssignStmt)
		if !ok || set.Tok != token.ASSIGN || len(set.Lhs) != 1 || len(set.Rhs) != 1 {
			return true
		}
		cast, ok := set.Rhs[0].(*ast.CallExpr)
		if !ok || len(cast.Args) != 1 || !isName(cast.Args[0], "count") {
			return true
		}
		kind, ok := cast.Fun.(*ast.Ident)
		if !ok || (kind.Name != "uint8" && kind.Name != "uint16" && kind.Name != "uint32" && kind.Name != "uint64") {
			return true
		}
		target := text(set.Lhs[0])
		if target == "" {
			return true
		}
		seen, ok := block.List[offset+4].(*ast.AssignStmt)
		if !ok || seen.Tok != token.ASSIGN || len(seen.Lhs) != 1 || len(seen.Rhs) != 1 || !isName(seen.Rhs[0], "true") {
			return true
		}
		flag, ok := seen.Lhs[0].(*ast.Ident)
		if !ok || !strings.HasPrefix(flag.Name, "countSeen") {
			return true
		}
		conflict, ok := block.List[offset+2].(*ast.IfStmt)
		if !ok || conflict.Init != nil || conflict.Else != nil || !returnsError(conflict.Body) {
			return true
		}
		want := flag.Name + "&&uint64(" + generatedWhitespace.ReplaceAllString(target, "") + ")!=count"
		if generatedWhitespace.ReplaceAllString(text(conflict.Cond), "") != want {
			return true
		}
		// Reuse the independent setter proof on this block, substituting only the
		// unchanged typed list path. Complete statement count prevents extra writes.
		code := strings.ReplaceAll(text(block), "len("+access+")", "len(items)")
		parsed, err := parser.ParseFile(token.NewFileSet(), "count.go", "package p;func(v *T)SetCount(items []uint8)error"+code, 0)
		if err != nil {
			return true
		}
		synthetic := parsed.Decls[0].(*ast.FuncDecl)
		expression := strings.ReplaceAll(f.Expression, "len("+access+")", "len(items)")
		if expression == "uint64("+target+")" {
			expression = kind.Name + "(count)"
		}
		if err := checkGeneratedCountSetter(synthetic, expression); err == nil {
			valid = true
		}
		return true
	})
	if !valid {
		return fmt.Errorf("unverified enclosing count arithmetic %s: %s", fn.Name.Name, f.Expression)
	}
	return nil
}
