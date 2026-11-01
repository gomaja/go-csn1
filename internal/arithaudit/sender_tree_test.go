package arithaudit

import (
	"go/ast"
	"path/filepath"
	"testing"
)

// Exercise every emitted sender receipt independently of inventory rows. A new
// proof shape must preserve existing generated decoders as well as its fixture.
func TestGeneratedSenderSpanReceipts(t *testing.T) {
	root := filepath.Join("..", "..")
	checker := &citationChecker{root: root, files: make(map[string]citationFile)}
	files, err := filepath.Glob(filepath.Join(root, "ts*", "*", "generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	receipts := 0
	for _, full := range files {
		name, err := filepath.Rel(root, full)
		if err != nil {
			t.Fatal(err)
		}
		name = filepath.ToSlash(name)
		file, err := checker.source(name)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file.file, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				return true
			}
			field, ok := assignment.Lhs[0].(*ast.SelectorExpr)
			if !ok {
				return true
			}
			wire, ok := field.X.(*ast.SelectorExpr)
			if !ok || !isName(wire.X, "v") || wire.Sel.Name != "Wire" {
				return true
			}
			expression := ""
			switch field.Sel.Name {
			case "BitsConsumed":
				expression = "r.Position() - senderStart"
			case "ImplicitZeros":
				expression = "r.VirtualBits() - senderVirtual"
			default:
				return true
			}
			receipts++
			position := file.set.Position(assignment.Rhs[0].Pos())
			if err := checker.checkGeneratedResidual(Finding{File: name, Line: position.Line, Column: position.Column, Expression: expression}); err != nil {
				t.Error(err)
			}
			return true
		})
	}
	if receipts == 0 {
		t.Fatal("no generated sender receipts found")
	}
	t.Logf("checked %d sender receipts across %d generated files", receipts, len(files))
}
