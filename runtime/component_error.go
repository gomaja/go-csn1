package runtime

import "strings"

// ComponentError locates a generated codec failure beneath the writer's current
// path, appending component only when it is not already the final level.
// Like OperationError, it preserves typed causes and defers attachment for
// speculative canonical candidates. OperationError accepts an explicit path.
func (w *Writer) ComponentError(kind ErrorKind, component string, width int, cause error) error {
	if cause == nil || w == nil || w.wire.rawErrors {
		return w.OperationError(kind, component, width, cause)
	}
	return w.OperationError(kind, errorPath(w.path, component), width, cause)
}

// ErrorPath locates a plain generated decoder failure beneath the reader's
// current path, appending component only when it is not already the final
// level. Callers use it only on failure; reader errors already carry their path.
func (r *Reader) ErrorPath(component string) string {
	if r == nil {
		return component
	}
	return errorPath(strings.Join(r.path, "/"), component)
}
