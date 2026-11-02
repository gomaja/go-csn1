package registry

import (
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.2.18–11.2.25b retain Distribution part
// error arms and nested padding Ignore regions in selected typed values.
// The canonical probe still requires the exact located sender refusal.
func psiSelectedReceiverRegion(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		return !v.IsNil() && psiSelectedReceiverRegion(v.Elem())
	}
	if v.Type() == reflect.TypeFor[runtime.ErrorRegion]() {
		r := v.Interface().(runtime.ErrorRegion)
		return r.Label == runtime.DistributionPartError || r.Label == runtime.Ignore
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.IsExported() && f.Name != "Wire" && psiSelectedReceiverRegion(v.Field(i)) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if psiSelectedReceiverRegion(v.Index(i)) {
				return true
			}
		}
	}
	return false
}

func TestPSIReceiverRefusalScope(t *testing.T) {
	expected := &runtime.OperationError{Kind: runtime.InvalidValue, Detail: "receiver error region has no canonical sender encoding"}
	for _, d := range Descriptors() {
		if d.Standard != "TS 44.060" {
			continue
		}
		switch d.Name {
		case "PSI1 message content", "PSI2 message content", "PSI3 ter message content", "PSI13 message content", "PSI14 message content", "PSI15 message content":
		default:
			continue
		}
		decoded, e := d.Decode(nil)
		if e != nil {
			t.Fatal(e)
		}
		value := reflect.ValueOf(decoded).FieldByName("Value").Interface()
		if !canonicalRegionRefusal(value, expected) {
			t.Fatalf("%s receiver arm not recognized", d.Name)
		}
		if canonicalRegionRefusal(value, &runtime.OperationError{Kind: runtime.InvalidSchema, Detail: expected.Detail}) {
			t.Fatal("unrelated schema error accepted")
		}
		fresh := reflect.Zero(reflect.TypeOf(value)).Interface()
		if canonicalRegionRefusal(fresh, expected) {
			t.Fatal("value without a receiver arm exempted")
		}
	}
}
