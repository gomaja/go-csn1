package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

// operationCauseDetail renders only facts not already stated by the operation.
// It reads the original causes without changing their fields or unwrap chain.
func operationCauseDetail(cause error, outer *OperationError) string {
	switch e := cause.(type) {
	case *OperationError:
		detail := e.Detail
		if detail == "" && e.Cause != nil {
			detail = operationCauseDetail(e.Cause, e)
		}
		return causeFacts(e.Operation, e.Kind, e.Location, detail, outer)
	case *fieldDecodeError:
		return operationCauseDetail(&e.DecodeError, outer)
	case *DecodeError:
		return causeFacts("", e.Kind, Location{Path: e.Path, BitOffset: e.Offset}, e.Detail, outer)
	case *BoundError:
		path := e.Path
		if e.Field != "" {
			path = fmt.Sprintf("%s/%s", path, e.Field)
		}
		detail := fmt.Sprintf("%s bound (limit %d): %s", e.Kind, e.Limit, e.Detail)
		return causeFacts("", "", Location{Path: path, BitOffset: e.Position}, detail, outer)
	case *FallbackError:
		return causeFacts("", "", Location{Path: e.Path, BitOffset: e.Position}, "ignored fallback bits decode through the known arm", outer)
	case *ContextRequiredError:
		name := ""
		if !causePathIncluded(outer.Location.Path, e.Name) {
			name = fmt.Sprintf(" <%s>", e.Name)
		}
		return fmt.Sprintf("%s §%s%s: %s", e.Standard, e.Clause, name, ErrContextRequired)
	}
	// Preserve a wrapper's own text while composing its trailing typed cause.
	// Only an exact suffix is replaced; other wrapper formats remain verbatim.
	if wrapper, ok := cause.(interface{ Unwrap() error }); ok {
		if inner := wrapper.Unwrap(); inner != nil {
			full, tail := cause.Error(), inner.Error()
			if strings.HasSuffix(full, tail) {
				return strings.TrimSuffix(full, tail) + operationCauseDetail(inner, outer)
			}
		}
	}
	if outer.Operation == "lookup" {
		return lookupCauseDetail(cause.Error(), outer.Location.Path)
	}
	return cause.Error()
}

func causeFacts(operation string, kind ErrorKind, location Location, detail string, outer *OperationError) string {
	facts := ""
	if operation != "" && operation != outer.Operation {
		facts = operation
	}
	if kind != "" && kind != outer.Kind {
		facts = joinCauseFact(facts, string(kind))
	}
	if location.BitOffset >= 0 && location.BitOffset != outer.Location.BitOffset {
		facts = joinCauseFact(facts, fmt.Sprintf("at bit %d", location.BitOffset))
	}
	if !causePathIncluded(outer.Location.Path, location.Path) {
		facts = joinCauseFact(facts, fmt.Sprintf("in %s", location.Path))
	}
	if facts == "" {
		return detail
	}
	if detail == "" {
		return facts
	}
	return facts + ": " + detail
}

func causePathIncluded(outer, inner string) bool {
	return inner == "" || outer == inner || strings.HasSuffix(outer, "/"+inner)
}

func joinCauseFact(facts, fact string) string {
	if facts == "" {
		return fact
	}
	return facts + " " + fact
}

// Lookup causes quote the definition name inside their standalone message.
// Remove that exact component when the surrounding path already names it
// unambiguously: an empty name, or one that needs escaping, stays quoted.
func lookupCauseDetail(message, name string) string {
	quoted := strconv.Quote(name)
	if name == "" || quoted != `"`+name+`"` {
		return message
	}
	for _, prefix := range []string{"unknown CSN.1 definition", "ambiguous CSN.1 definition"} {
		if tail, ok := strings.CutPrefix(message, prefix+" "+quoted); ok && strings.HasPrefix(tail, " in ") {
			return prefix + tail
		}
	}
	return message
}
