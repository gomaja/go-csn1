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
	Decode                          func([]byte) (any, error)
	Encode                          func(any) ([]byte, error)
	DecodeWithContext               func([]byte, SI4ACS) (any, error)
	EncodeWithContext               func(any, SI4ACS) ([]byte, error)
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
