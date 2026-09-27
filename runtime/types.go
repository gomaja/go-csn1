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
