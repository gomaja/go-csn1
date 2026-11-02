package arithaudit

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
)

// The selected sender's exact template measures Content into a bounded Writer,
// rejects bits outside 0..127, and rejects count>12 before count*10+1. Its
// reflect loops start at zero and stop at the unchanged NumField/Len bound.
// TS 44.060 V19.0.0 §12.30; TS 24.008 V20.1.0 §10.5.5.12a.
// Pinning the complete function prevents an unrelated or shadowed guard from
// authenticating an arithmetic operation. Template edits require renewed audit.
func checkGeneratedCapabilitySender(file citationFile, fn *ast.FuncDecl) error {
	start, end := file.set.Position(fn.Pos()).Offset, file.set.Position(fn.End()).Offset
	text := generatedWhitespace.ReplaceAllString(string(file.source[start:end]), "")
	// The second exact form uses the collision-free capabilitymsrac import
	// alias; only its three qualified type/function selectors differ.
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	if digest != "21f5134e8df2caa5e68eed77ad0956c225d7a5e9392b33e07c4e5f0ce7417268" && digest != "d134afb297deed8741263c91bdfbff55cca2af61cf84f4b4b695c7ca7ff2a84d" {
		return fmt.Errorf("capability sender arithmetic template changed")
	}
	return nil
}

// The foreign decoder advances the same bounded Reader; no fork is made and
// the two snapshots remain unchanged until the successful receipt is stored.
func checkGeneratedCapabilitySpan(file citationFile, fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || generatedNodeText(file, fn.Type.Params.List[0]) != "r*runtime.Reader" {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 2 || generatedNodeText(file, fn.Type.Results.List[0]) != "msrac.MSRACapabilityValuePartStruct" || generatedNodeText(file, fn.Type.Results.List[1]) != "error" {
		return false
	}
	const expected = "{senderStart:=r.Position()senderVirtual:=r.VirtualBits()v,err:=msrac.DecodeMSRACapabilityValuePartStructFrom(r)iferr!=nil{returnv,err}v.Wire.BitsConsumed=r.Position()-senderStartv.Wire.ImplicitZeros=r.VirtualBits()-senderVirtualreturnv,nil}"
	return generatedNodeText(file, fn.Body) == expected
}
