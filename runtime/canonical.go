package runtime

import (
	"fmt"
	"reflect"
)

var wireInfoType = reflect.TypeFor[WireInfo]()

type canonicalPointer struct {
	typ reflect.Type
	ptr uintptr
}

// Canonical makes an independent copy of a generated value and removes all
// received wire layout from it, including nested fields. Generated canonical
// encoders use the copy so that a caller can also keep encoding the original
// value byte for byte.
func Canonical[T any](value T) T {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return value
	}
	copy := canonicalValue(reflected, make(map[canonicalPointer]reflect.Value))
	return copy.Interface().(T)
}

// CanonicalTo gives a fresh value an explicit transmitted length in bits.
// Generated encoders use it for truncated rest octets and receiver-inferred
// zero extensions. The target is independent of any received wire layout.
func CanonicalTo[T any](value T, bits int) T {
	copy := Canonical(value)
	rv := reflect.ValueOf(&copy).Elem()
	if rv.Kind() == reflect.Struct {
		field := rv.FieldByName("Wire")
		if field.IsValid() && field.CanSet() && field.Type() == wireInfoType {
			field.Set(reflect.ValueOf(WireInfo{canonical: true, targetSet: true, targetBits: bits}))
		}
	}
	return copy
}

func SemanticallyEqual[T any](a, b T) bool {
	return reflect.DeepEqual(Canonical(a), Canonical(b))
}

// CanonicalEncode tries a fresh encoding, then the permitted octet extents
// when source-defined truncation or inferred zeros require a shorter layout.
// Each candidate must fit the source-defined extent and decode to the same
// typed semantic value. A zero maximum requires CanonicalEncodeAtLength.
func CanonicalEncode[T any](value T, minOctets, maxOctets int, shortest bool, encode func(T) ([]byte, error), decode func([]byte) (Decoded[T], error), equivalent func(T, T) bool) ([]byte, error) {
	if maxOctets == 0 {
		return nil, &ExtentError{Minimum: minOctets, Required: true}
	}
	if minOctets < 0 || maxOctets < minOctets || maxOctets > maxBits/8 {
		return nil, fmt.Errorf("invalid canonical extent range")
	}
	if equivalent == nil {
		equivalent = SemanticallyEqual[T]
	}
	try := func(fresh T) ([]byte, error) {
		out, err := encode(fresh)
		if err != nil {
			return nil, err
		}
		if len(out) < minOctets || len(out) > maxOctets {
			return nil, &ExtentError{Actual: len(out), Minimum: minOctets, Maximum: maxOctets}
		}
		again, err := decode(out)
		if err != nil {
			return nil, fmt.Errorf("canonical bytes do not decode: %w", err)
		}
		if !equivalent(value, again.Value) {
			return nil, fmt.Errorf("canonical bytes change typed semantics")
		}
		return out, nil
	}
	search := func() ([]byte, bool) {
		for octets := minOctets; octets <= maxOctets; octets++ {
			out, err := try(CanonicalTo(value, octets*8))
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
	out, firstErr := try(Canonical(value))
	if firstErr == nil {
		return out, nil
	}
	if !shortest {
		if out, ok := search(); ok {
			return out, nil
		}
	}
	return nil, firstErr
}

// CanonicalEncodeAtLength uses an explicit containing-message value length.
// A candidate is accepted only if it fills that extent and decodes to the
// original typed value. The caller supplies the length prescribed by its
// enclosing message (TS 44.018 V19.0.0 §8.9). A zero maximum means that
// the definition has no standalone extent, so the supplied length is the
// bound (subject to the runtime bit limit).
func CanonicalEncodeAtLength[T any](value T, octets, minOctets, maxOctets int, encode func(T) ([]byte, error), decode func([]byte) (Decoded[T], error), equivalent func(T, T) bool) ([]byte, error) {
	if maxOctets == 0 {
		maxOctets = maxBits / 8
	}
	if octets < minOctets || octets > maxOctets || maxOctets > maxBits/8 {
		return nil, &ExtentError{Actual: octets, Minimum: minOctets, Maximum: maxOctets}
	}
	if equivalent == nil {
		equivalent = SemanticallyEqual[T]
	}
	out, err := encode(CanonicalTo(value, octets*8))
	if err != nil {
		return nil, err
	}
	if len(out) != octets {
		return nil, &ExtentError{Actual: len(out), Minimum: octets, Maximum: octets}
	}
	again, err := decode(out)
	if err != nil {
		return nil, fmt.Errorf("canonical bytes do not decode: %w", err)
	}
	if !equivalent(value, again.Value) {
		return nil, fmt.Errorf("canonical bytes change typed semantics")
	}
	return out, nil
}

func canonicalValue(value reflect.Value, seen map[canonicalPointer]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if value.Type() == wireInfoType {
		return reflect.ValueOf(WireInfo{canonical: true})
	}
	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		key := canonicalPointer{value.Type(), value.Pointer()}
		if prior, ok := seen[key]; ok {
			return prior
		}
		out := reflect.New(value.Type().Elem())
		seen[key] = out
		out.Elem().Set(canonicalValue(value.Elem(), seen))
		return out
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.New(value.Type()).Elem()
		out.Set(canonicalValue(value.Elem(), seen))
		return out
	case reflect.Struct:
		out := reflect.New(value.Type()).Elem()
		out.Set(value)
		for i := range value.NumField() {
			if out.Field(i).CanSet() && value.Field(i).CanInterface() {
				out.Field(i).Set(canonicalValue(value.Field(i), seen))
			}
		}
		return out
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := range value.Len() {
			out.Index(i).Set(canonicalValue(value.Index(i), seen))
		}
		return out
	case reflect.Array:
		out := reflect.New(value.Type()).Elem()
		for i := range value.Len() {
			out.Index(i).Set(canonicalValue(value.Index(i), seen))
		}
		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			out.SetMapIndex(iter.Key(), canonicalValue(iter.Value(), seen))
		}
		return out
	default:
		return value
	}
}
