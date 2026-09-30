package runtime

import "reflect"

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

func canonicalValue(value reflect.Value, seen map[canonicalPointer]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	if value.Type() == wireInfoType {
		return reflect.Zero(wireInfoType)
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
