package runtime

import (
	"errors"
	"fmt"
	"reflect"
)

var wireInfoType = reflect.TypeFor[WireInfo]()
var bitStringType = reflect.TypeFor[BitString]()

type canonicalPointer struct {
	typ reflect.Type
	ptr uintptr
}

// Canonical makes an independent copy of a generated value and removes all
// received wire layout from it, including nested fields. Generated canonical
// encoders use the copy so that a caller can also keep encoding the original
// value byte for byte.
func Canonical[T any](value T) T { return canonicalCopy(value, false) }
func canonicalCopy[T any](value T, rawErrors bool) T {
	input := any(value)
	if input == nil {
		return value
	}
	return canonicalCopyValue(input, rawErrors).(T)
}

func canonicalCopyValue(value any, rawErrors bool) any {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return value
	}
	return canonicalValue(reflected, make(map[canonicalPointer]reflect.Value), rawErrors).Interface()
}

// CanonicalTo gives a fresh value an explicit transmitted length in bits.
// Generated encoders use it for truncated rest octets and receiver-inferred
// zero extensions. The target is independent of any received wire layout.
func CanonicalTo[T any](value T, bits int) T { return canonicalTo(value, bits, false) }
func canonicalTo[T any](value T, bits int, rawErrors bool) T {
	copy := canonicalCopy(value, rawErrors)
	setCanonicalTarget(reflect.ValueOf(&copy).Elem(), bits, rawErrors)
	return copy
}

func setCanonicalTarget(rv reflect.Value, bits int, rawErrors bool) {
	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName("Wire")
		if field.IsValid() && field.CanSet() && field.Type() == wireInfoType {
			field.Set(reflect.ValueOf(WireInfo{canonical: true, targetSet: true, targetBits: bits, rawErrors: rawErrors}))
		}
	}
}

func SemanticallyEqual[T any](a, b T) bool {
	return reflect.DeepEqual(canonicalCopyValue(a, false), canonicalCopyValue(b, false))
}

// CanonicalEncode tries a fresh encoding, then the permitted octet extents
// when source-defined truncation or inferred zeros require a shorter layout.
// Each candidate must fit the source-defined extent and decode to the same
// typed semantic value. A zero maximum returns *ExtentError with Required
// set: the caller must use CanonicalEncodeAtLength. Otherwise, if no
// candidate succeeds, the fresh encoding's error is returned:
//   - *ExtentError when it falls outside minOctets..maxOctets;
//   - the encoder's own error, which for generated encoders is *BoundError
//     with Kind LengthBound when content does not fit a bound inside the
//     value, or an error wrapping ErrContextRequired;
//   - an error wrapping the decode error when the bytes do not decode.
//
// Bytes that decode to a different typed value give an untyped error.
func CanonicalEncode[T any](value T, minOctets, maxOctets int, shortest bool, encode func(T) ([]byte, error), decode func([]byte) (Decoded[T], error), equivalent func(T, T) bool) ([]byte, error) {
	if equivalent == nil {
		equivalent = SemanticallyEqual[T]
	}
	return canonicalEncode(minOctets, maxOctets, shortest,
		func(bits int, target bool) ([]byte, error) {
			if target {
				return encode(canonicalTo(value, bits, true))
			}
			return encode(canonicalCopy(value, true))
		},
		func(out []byte) error {
			again, err := decode(out)
			if err != nil {
				return fmt.Errorf("canonical bytes do not decode: %w", err)
			}
			if !equivalent(value, again.Value) {
				return errCanonicalMismatch
			}
			return nil
		},
		func(bits int, target bool) error {
			if target {
				_, err := encode(CanonicalTo(value, bits))
				return err
			}
			_, err := encode(Canonical(value))
			return err
		})
}

// canonicalEncode shares the extent search across generated value types. Its
// typed callbacks do not need an intermediate interface box.
func canonicalEncode(minOctets, maxOctets int, shortest bool, encode func(int, bool) ([]byte, error), verify func([]byte) error, locate func(int, bool) error) ([]byte, error) {
	if maxOctets == 0 {
		return nil, &ExtentError{Minimum: minOctets, Required: true}
	}
	if minOctets < 0 || maxOctets < minOctets || maxOctets > maxBits/8 {
		return nil, canonicalFailure(fmt.Errorf("invalid canonical extent range"))
	}
	try := func(bits int, target bool) ([]byte, error, bool) {
		out, err := encode(bits, target)
		if err != nil {
			return nil, err, false
		}
		if len(out) < minOctets || len(out) > maxOctets {
			return nil, &ExtentError{Actual: len(out), Minimum: minOctets, Maximum: maxOctets}, false
		}
		if err := verify(out); err != nil {
			return nil, err, true
		}
		return out, nil, false
	}
	var targetErr error
	targetBits := 0
	search := func() ([]byte, bool) {
		for octets := minOctets; octets <= maxOctets; octets++ {
			out, err, _ := try(octets*8, true)
			if targetErr == nil && err != nil {
				targetErr, targetBits = err, octets*8
			}
			if err == nil && len(out) == octets {
				return out, true
			}
		}
		return nil, false
	}
	if shortest {
		if out, ok := search(); ok {
			return out, nil
		}
	}
	out, firstErr, firstSchema := try(0, false)
	if firstErr == nil {
		return out, nil
	}
	if !shortest {
		if out, ok := search(); ok {
			return out, nil
		}
	}
	if firstSchema {
		return nil, canonicalFailure(firstErr)
	}
	// A null refusal at the fresh octet boundary is only a probe. Report
	// the actual source/caller extent that the targeted attempt requested.
	var bound, targetedBound *BoundError
	if errors.As(firstErr, &bound) && bound.Kind == CanonicalTarget && errors.As(targetErr, &targetedBound) && targetedBound.Kind == CanonicalTarget && targetedBound.Position == bound.Position {
		if located := locate(targetBits, true); located != nil {
			return nil, located
		}
		return nil, targetErr
	}
	if located := locate(0, false); located != nil {
		return nil, located
	}
	return nil, firstErr
}

// CanonicalEncodeAtLength uses an explicit containing-message value length.
// A candidate is accepted only if it fills that extent and decodes to the
// original typed value. The caller supplies the length prescribed by its
// enclosing message (TS 44.018 V19.0.0 §8.9). A zero maximum means that
// the definition has no standalone extent, so the supplied length is the
// bound (subject to the runtime bit limit). It returns:
//   - *ExtentError when octets is outside minOctets..maxOctets, or the
//     encoding does not fill exactly octets;
//   - the encoder's own error otherwise. Generated encoders return
//     *BoundError with Kind CanonicalTarget when content does not fit in
//     octets, or Kind LengthBound when it does not fit a bound inside the
//     value. A typed length within the value keeps its value: content it
//     cannot carry, such as a nonzero field after a TS 44.060 V19.0.0
//     §12.24 Extension Length, is a LengthBound conflict. A value whose
//     extent belongs to an absent containing message wraps
//     ErrContextRequired;
//   - an error wrapping the decode error when the bytes do not decode.
//
// Bytes that decode to a different typed value give an untyped error.
func CanonicalEncodeAtLength[T any](value T, octets, minOctets, maxOctets int, encode func(T) ([]byte, error), decode func([]byte) (Decoded[T], error), equivalent func(T, T) bool) ([]byte, error) {
	if equivalent == nil {
		equivalent = SemanticallyEqual[T]
	}
	return canonicalEncodeAtLength(octets, minOctets, maxOctets,
		func(bits int) ([]byte, error) { return encode(CanonicalTo(value, bits)) },
		func(out []byte) error {
			again, err := decode(out)
			if err != nil {
				return canonicalFailure(fmt.Errorf("canonical bytes do not decode: %w", err))
			}
			if !equivalent(value, again.Value) {
				return canonicalFailure(fmt.Errorf("canonical bytes change typed semantics"))
			}
			return nil
		})
}

// canonicalEncodeAtLength validates the extent before invoking the typed
// callbacks, and rejects an unfilled target before decoding its bytes.
func canonicalEncodeAtLength(octets, minOctets, maxOctets int, encode func(int) ([]byte, error), verify func([]byte) error) ([]byte, error) {
	if maxOctets == 0 {
		maxOctets = maxBits / 8
	}
	if octets < minOctets || octets > maxOctets || maxOctets > maxBits/8 {
		return nil, &ExtentError{Actual: octets, Minimum: minOctets, Maximum: maxOctets}
	}
	out, err := encode(octets * 8)
	if err != nil {
		return nil, err
	}
	if len(out) != octets {
		return nil, &ExtentError{Actual: len(out), Minimum: octets, Maximum: octets}
	}
	if err := verify(out); err != nil {
		return nil, err
	}
	return out, nil
}

// canonicalWire and canonicalRawErrorsWire are the shared fresh wire records
// of a canonical copy. Callers only copy them into a destination, so sharing
// one boxed record avoids an allocation for every WireInfo field copied.
var (
	canonicalWire          = reflect.ValueOf(WireInfo{canonical: true})
	canonicalRawErrorsWire = reflect.ValueOf(WireInfo{canonical: true, rawErrors: true})
)

func canonicalValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	if !value.IsValid() {
		return value
	}
	// TS 44.060 V19.0.0 §12.29 paragraphs 23515–23520 permits a
	// zero-length bitmap: neither nil nor an empty byte slice transmits
	// any bitmap bits. Normalize only that representation, keeping malformed
	// nonempty storage and all other slice distinctions visible.
	if value.Type() == bitStringType && value.Field(1).Int() == 0 && value.Field(0).Len() == 0 {
		return reflect.Zero(bitStringType)
	}
	if value.Type() == wireInfoType {
		if rawErrors {
			return canonicalRawErrorsWire
		}
		return canonicalWire
	}
	switch value.Kind() {
	case reflect.Pointer:
		return canonicalPointerValue(value, seen, rawErrors)
	case reflect.Interface:
		return canonicalInterfaceValue(value, seen, rawErrors)
	case reflect.Struct:
		return canonicalStructValue(value, seen, rawErrors)
	case reflect.Slice:
		return canonicalSliceValue(value, seen, rawErrors)
	case reflect.Array:
		return canonicalArrayValue(value, seen, rawErrors)
	case reflect.Map:
		return canonicalMapValue(value, seen, rawErrors)
	default:
		return value
	}
}

// Keep each reflection branch in a separate compilation body. Recursive
// copies retain one shared pointer-identity map and the same wire policy.
func canonicalPointerValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}
	key := canonicalPointer{value.Type(), value.Pointer()}
	if prior, ok := seen[key]; ok {
		return prior
	}
	out := reflect.New(value.Type().Elem())
	seen[key] = out
	out.Elem().Set(canonicalValue(value.Elem(), seen, rawErrors))
	return out
}
func canonicalInterfaceValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}
	out := reflect.New(value.Type()).Elem()
	out.Set(canonicalValue(value.Elem(), seen, rawErrors))
	return out
}
func canonicalStructValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	out := reflect.New(value.Type()).Elem()
	out.Set(value)
	for i := range value.NumField() {
		if out.Field(i).CanSet() && value.Field(i).CanInterface() {
			out.Field(i).Set(canonicalValue(value.Field(i), seen, rawErrors))
		}
	}
	return out
}
func canonicalSliceValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}
	out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
	for i := range value.Len() {
		out.Index(i).Set(canonicalValue(value.Index(i), seen, rawErrors))
	}
	return out
}
func canonicalArrayValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	out := reflect.New(value.Type()).Elem()
	for i := range value.Len() {
		out.Index(i).Set(canonicalValue(value.Index(i), seen, rawErrors))
	}
	return out
}
func canonicalMapValue(value reflect.Value, seen map[canonicalPointer]reflect.Value, rawErrors bool) reflect.Value {
	if value.IsNil() {
		return reflect.Zero(value.Type())
	}
	out := reflect.MakeMapWithSize(value.Type(), value.Len())
	iter := value.MapRange()
	for iter.Next() {
		out.SetMapIndex(iter.Key(), canonicalValue(iter.Value(), seen, rawErrors))
	}
	return out
}

func canonicalFailure(cause error) error {
	return &OperationError{Operation: "canonical", Kind: InvalidSchema, Location: Location{BitOffset: -1}, Cause: cause}
}

var errCanonicalMismatch = errors.New("canonical bytes change typed semantics")
