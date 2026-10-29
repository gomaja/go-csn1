package runtime

import (
	"reflect"
	"testing"
)

func TestReleasedChoiceReaderDropsReferences(t *testing.T) {
	root := NewReader([]byte{1, 2, 3})
	root.Set("retained", 1)
	if err := root.Enter("root"); err != nil {
		t.Fatal(err)
	}
	var choices ChoiceErrors
	arm := choices.ForkScoped(root)
	state := arm.trial
	arm.collector = new(Collector)
	arm.wire = WireInfo{Tail: BitString{Bytes: []byte{1}}, Spare: []BitString{{Bytes: []byte{2}}}, Padding: []BitString{{Bytes: []byte{3}}}, Terminal: []BitString{{Bytes: []byte{4}}}, TruncatedAt: map[string][]int{"x": {1}}, SpareCounts: map[string][]int{"x": {1}}, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}}
	arm.Set("overlay", 2)
	choices.Release()
	var check func(reflect.Value, string)
	check = func(v reflect.Value, path string) {
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				field := v.Type().Field(i)
				if path == "Reader" && field.Name == "trial" {
					continue
				}
				check(v.Field(i), path+"."+field.Name)
			}
			return
		}
		switch v.Kind() {
		case reflect.Map, reflect.Slice, reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan:
			if !v.IsNil() {
				t.Fatalf("released reader retains %s", path)
			}
		}
	}
	check(reflect.ValueOf(*arm), "Reader")
	if arm.trial != state || state.base != nil || state.next != nil || state.reader != nil || len(state.values) != 0 {
		t.Fatal("pooled metadata retains a live trial")
	}
}

func BenchmarkChoiceReaderRelease(b *testing.B) {
	r := NewReader([]byte{1})
	r.Set("source", 1)
	b.ReportAllocs()
	for b.Loop() {
		var c ChoiceErrors
		arm := c.ForkScoped(r)
		arm.Set("trial", 2)
		c.Release()
	}
}
