package runtime

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestValueWireCountPrefixesRemainIndependent(t *testing.T) {
	r := NewReader([]byte{0})
	r.wire.TruncatedAt = map[string][]int{"kept": {1}, "grown": {1, 2}}
	r.wire.SpareCounts = map[string][]int{"kept": {3}, "grown": {3, 4}}
	before := WireInfo{
		TruncatedAt: map[string][]int{"kept": {1}, "grown": {1}},
		SpareCounts: map[string][]int{"kept": {3}, "grown": {3}},
	}
	start := r.Fork()
	wire, err := r.ValueWire(start, before, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(wire.TruncatedAt, map[string][]int{"grown": {2}}) ||
		!reflect.DeepEqual(wire.SpareCounts, map[string][]int{"grown": {4}}) {
		t.Fatalf("count deltas: %v %v", wire.TruncatedAt, wire.SpareCounts)
	}
	wire.TruncatedAt["grown"][0] = 7
	wire.SpareCounts["grown"][0] = 8
	if r.wire.TruncatedAt["grown"][1] != 2 || r.wire.SpareCounts["grown"][1] != 4 {
		t.Fatal("captured counts alias reader storage")
	}
	empty := NewReader(nil)
	got, err := empty.ValueWire(empty.Fork(), WireInfo{}, false)
	if err != nil || got.TruncatedAt == nil || got.SpareCounts == nil {
		t.Fatalf("empty count maps: %#v, %v", got, err)
	}
}

func TestValueWireRejectsCountPrefixOverrun(t *testing.T) {
	for _, name := range []string{"truncation", "spare"} {
		t.Run(name, func(t *testing.T) {
			r := NewReader(nil)
			r.Enter("parent")
			r.wire.TruncatedAt = map[string][]int{"key": {}}
			r.wire.SpareCounts = map[string][]int{"key": {}}
			before := WireInfo{}
			if name == "truncation" {
				before.TruncatedAt = map[string][]int{"key": {1}}
			} else {
				before.SpareCounts = map[string][]int{"key": {1}}
			}
			_, err := r.ValueWire(r.Fork(), before, false)
			var operation *OperationError
			if !errors.As(err, &operation) || operation.Kind != InvalidValue ||
				operation.Operation != "decode" || operation.Location.Path != "parent" ||
				operation.Location.BitOffset != 0 || operation.Location.Width != 0 ||
				!strings.Contains(err.Error(), "value "+name+" prefix exceeds received state") {
				t.Fatalf("prefix failure: %#v, %v", operation, err)
			}
		})
	}
}

func TestValueWireClipsAndValidatesImplicitSpans(t *testing.T) {
	r := NewReader(nil)
	r.virtual = 4
	r.wire.ImplicitSpans = []ImplicitSpan{{At: 0, Count: 1}, {At: 0, Count: 4}}
	got, err := r.ValueWire(r.Fork(), WireInfo{ImplicitZeros: 1}, false)
	if err != nil || !reflect.DeepEqual(got.ImplicitSpans, []ImplicitSpan{{At: 0, Count: 3}}) || got.ImplicitZeros != 3 {
		t.Fatalf("clipped spans: %#v, %v", got, err)
	}
	for _, span := range []ImplicitSpan{
		{At: -1, Count: 1},
		{At: maxBits + 1},
		{At: 0, Count: -1},
		{At: maxBits, Count: 1},
	} {
		r := NewReader(nil)
		r.wire.ImplicitSpans = []ImplicitSpan{span}
		_, err := r.ValueWire(r.Fork(), WireInfo{}, false)
		if err == nil || !strings.Contains(err.Error(), "invalid inferred span in received value") {
			t.Fatalf("accepted invalid span %#v: %v", span, err)
		}
	}
}
