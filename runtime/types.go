// Package runtime supplies bounded bit operations for generated CSN.1 codecs.
package runtime

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyValue means a named rest-octet value is absent from the containing
// message. TS 44.018 V19.0.0 §10.5.2.16 permits a zero-octet IE length,
// while its printed CSN.1 grammar begins with a required discriminator.
var ErrEmptyValue = errors.New("empty CSN.1 value")

// ErrUnsupported identifies a named value whose printed grammar cannot be
// selected from its bytes without information outside that value.
var ErrUnsupported = errors.New("unsupported CSN.1 value")

// ErrContextRequired identifies a value whose alternatives are selected by
// a field in its containing message (TS 44.018 V19.0.0 §10.5.2.35).
var ErrContextRequired = errors.New("CSN.1 context required")

type ContextRequiredError struct{ Standard, Clause, Name string }

func (e *ContextRequiredError) Error() string {
	return fmt.Sprintf("%s §%s <%s>: %s", e.Standard, e.Clause, e.Name, ErrContextRequired)
}

func (e *ContextRequiredError) Unwrap() error { return ErrContextRequired }

// SI4ACS is the ACS bit in the containing SI4 message. TS 44.018 V19.0.0
// §10.5.2.35 selects SI7/SI8's O+S layout for one and S-only for zero.
type SI4ACS uint8

const (
	SI4ACSZero SI4ACS = 0
	SI4ACSOne  SI4ACS = 1
)

func (c SI4ACS) Valid() bool { return c == SI4ACSZero || c == SI4ACSOne }

// BitString is an MSB-first bit sequence. Unused low bits in the last byte
// are ignored; BitLength is the exact number of significant bits.
type BitString struct {
	Bytes     []byte
	BitLength int
}

type Decoded[T any] struct {
	Value        T
	BitsConsumed int
	Tail         BitString
}

// Descriptor identifies a definition by published specification, clause and
// printed name. Version is metadata and never part of the lookup key.
type Descriptor struct {
	Standard, Version, Clause, Name string
	// MaxOctets is zero when a definition has no standalone source bound.
	MaxOctets            int
	Decode               func([]byte) (any, error)
	DecodeFrom           func(*Reader) (any, error)
	Encode               func(any) ([]byte, error)
	Canonical            func(any) ([]byte, error)
	CanonicalAtLength    func(any, int) ([]byte, error)
	DecodeWithContext    func([]byte, SI4ACS) (any, error)
	EncodeWithContext    func(any, SI4ACS) ([]byte, error)
	CanonicalWithContext func(any, SI4ACS) ([]byte, error)
}

type ErrorKind string

const (
	Truncated     ErrorKind = "truncated"
	InvalidBranch ErrorKind = "invalid-branch"
	InvalidValue  ErrorKind = "invalid-value"
	Limit         ErrorKind = "limit"
	InvalidSchema ErrorKind = "invalid-schema"
)

type DecodeError struct {
	Kind         ErrorKind
	Offset       int
	Path, Detail string
}

// ExtentError reports a missing caller-supplied extent or an encoding that
// does not fit its source-defined or caller-supplied octet extent. Maximum
// zero means the source does not define a standalone maximum.
type ExtentError struct {
	Actual, Minimum, Maximum int
	Required                 bool
}

func (e *ExtentError) Error() string {
	if e.Required {
		return "CSN.1 canonical encoding requires an explicit containing-message length"
	}
	return fmt.Sprintf("CSN.1 extent %d outside %d..%d octets", e.Actual, e.Minimum, e.Maximum)
}

// ErrBoundConflict identifies a typed value that cannot be encoded within a
// bound enclosing it. Every *BoundError unwraps to it.
var ErrBoundConflict = errors.New("CSN.1 value conflicts with its enclosing bound")

// BoundKind names the bound that a typed value conflicts with.
type BoundKind string

const (
	// LengthBound is a length-delimited value such as TS 44.060 V19.0.0
	// §12.24 "< bit (val(Extension Length) + 1) & … >", or a fixed-size
	// value such as the TS 44.018 V19.0.0 §10.5.2.37h 20-octet SI 18 value.
	LengthBound BoundKind = "length"
	// CanonicalTarget is the explicit extent of a canonical encoding.
	CanonicalTarget BoundKind = "canonical-target"
	// ReceivedTruncation is the truncation point recorded when a value was
	// decoded (TS 44.060 V19.0.0 §11.1.4.4, TS 44.018 V19.0.0 §8.9).
	ReceivedTruncation BoundKind = "received-truncation"
)

// BoundError reports a typed value that does not fit a bound enclosing it.
// For example, a GPRS Cell Options Extension Length that is too short for a
// nonzero later field (TS 44.060 V19.0.0 §12.24): the truncated
// concatenation must end at the length, and only zero components may be
// omitted. Path is the encoder path at the conflict, Field the conflicting
// field or component, Limit the bound and Position the bits written so far,
// both counted from the start of the encoding.
type BoundError struct {
	Kind            BoundKind
	Path, Field     string
	Detail          string
	Limit, Position int
}

func (e *BoundError) Error() string {
	return fmt.Sprintf("CSN.1 %s bound at bit %d conflicts with %s at %s (bit %d): %s", e.Kind, e.Limit, e.Field, e.Path, e.Position, e.Detail)
}

func (e *BoundError) Unwrap() error { return ErrBoundConflict }

// PreferTruncation retains the deepest truncated choice arm. TS 24.007
// V20.0.0 Annex B §B.1.2.2 permits alternative decoding; when no arm
// matches because input ends inside one, that boundary is the useful error.
func PreferTruncation(current, candidate error) error {
	var next *DecodeError
	if !errors.As(candidate, &next) || next.Kind != Truncated {
		return current
	}
	var previous *DecodeError
	if !errors.As(current, &previous) || next.Offset > previous.Offset {
		return candidate
	}
	return current
}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("CSN.1 %s at bit %d in %s: %s", e.Kind, e.Offset, e.Path, e.Detail)
}

func key(name string) string { return strings.ToLower(strings.Join(strings.Fields(name), " ")) }

func cloneVars(input map[string]uint64) map[string]uint64 {
	out := make(map[string]uint64, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
