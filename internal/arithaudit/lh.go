package arithaudit

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
)

// checkLHUnsignedSource admits only a single direct binding from the pinned
// runtime's uint8 result. Reject every shadow, reassignment or address escape
// in the function rather than infer types from an identifier's spelling.
func checkLHUnsignedSource(file citationFile, fn *ast.FuncDecl, f Finding) error {
	if !reviewedLHBinding(file, fn) {
		return fmt.Errorf("L/H conversion lacks an unchanged unsigned runtime result: %s", f.Key())
	}
	proven := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || file.set.Position(call.Pos()).Line != f.Line || !isName(call.Fun, "uint64") || len(call.Args) != 1 {
			return true
		}
		if f.Column > 0 && file.set.Position(call.Pos()).Column != f.Column {
			return true
		}
		operand := call.Args[0]
		if xor, ok := operand.(*ast.BinaryExpr); ok {
			literal, ok := xor.Y.(*ast.BasicLit)
			if xor.Op != token.XOR || !ok || literal.Kind != token.INT || literal.Value != "1" {
				return true
			}
			operand = xor.X
		}
		if isName(operand, "lhL") {
			proven = true
		}
		return true
	})
	if !proven {
		return fmt.Errorf("L/H conversion differs from its reviewed operand: %s", f.Key())
	}
	return nil
}

func reviewedLHBinding(file citationFile, fn *ast.FuncDecl) bool {
	imported := false
	for _, spec := range file.file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/gomaja/go-csn1/runtime" && (spec.Name == nil || spec.Name.Name == "runtime") {
			imported = true
		}
	}
	if !imported {
		return false
	}
	var binding *ast.AssignStmt
	for _, statement := range fn.Body.List {
		assign, ok := statement.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != 2 || !isName(assign.Lhs[0], "lhL") || len(assign.Rhs) != 1 {
			continue
		}
		if binding != nil {
			return false
		}
		binding = assign
	}
	if binding == nil {
		return false
	}
	call, ok := binding.Rhs[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	symbol, ok := call.Args[0].(*ast.BasicLit)
	if !ok || symbol.Kind != token.CHAR || symbol.Value != "'L'" {
		return false
	}
	method, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || method.Sel.Name != "LHBit" {
		return false
	}
	receiver, ok := method.X.(*ast.Ident)
	if !ok {
		return false
	}
	typedReceiver := false
	for _, field := range fn.Type.Params.List {
		pointer, ok := field.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		typ, ok := pointer.X.(*ast.SelectorExpr)
		if !ok || !isName(typ.X, "runtime") || (typ.Sel.Name != "Reader" && typ.Sel.Name != "Writer") {
			continue
		}
		for _, parameter := range field.Names {
			if parameter.Name == receiver.Name {
				typedReceiver = true
			}
		}
	}
	if !typedReceiver {
		return false
	}
	valid := true
	ast.Inspect(fn, func(node ast.Node) bool {
		if node == binding {
			return true
		}
		if writesName(node, "lhL") || writesName(node, receiver.Name) {
			valid = false
		}
		switch value := node.(type) {
		case *ast.Field:
			for _, name := range value.Names {
				if name.Name == "lhL" {
					valid = false
				}
			}
		case *ast.UnaryExpr:
			if value.Op == token.AND && isName(value.X, "lhL") {
				valid = false
			}
		}
		return true
	})
	return valid
}

// reviewedLHMaskWrites preserves the one-bit mask used by the narrowing
// proof. The whole function must retain the runtime result unchanged.
func reviewedLHMaskWrites(file citationFile, fn *ast.FuncDecl) bool {
	valid := true
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || assign.Tok != token.XOR_ASSIGN || len(assign.Lhs) != 1 || !isName(assign.Lhs[0], "v") || len(assign.Rhs) != 1 {
			return true
		}
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || !isName(call.Fun, "uint64") || !isName(call.Args[0], "lhL") {
			return true
		}
		valid = valid && reviewedLHBinding(file, fn)
		return true
	})
	return valid
}
