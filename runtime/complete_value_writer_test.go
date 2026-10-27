package runtime

import (
	"errors"
	"reflect"
	"testing"
)

func TestWithCompleteWireValueScopesFreshLayout(t *testing.T) {
	for _, fail := range []bool{false, true} {
		parent := Seal(struct{}{}, []byte{0xa5}, 1, BitString{Bytes: []byte{0x4a}, BitLength: 7}, WireInfo{Padding: []BitString{{Bytes: []byte{0xaa}, BitLength: 8}}})
		w := NewWriter()
		w.WithWire(parent)
		before := *w
		sentinel := errors.New("complete value failure")
		err := w.WithCompleteWireValue(WireInfo{}, 16, func() error {
			if _, received := w.ReceivedBits(); received {
				t.Fatal("fresh value inherited received layout")
			}
			if fail {
				if err := w.WriteUint(0, 1); err != nil {
					t.Fatal(err)
				}
				return sentinel
			}
			return w.WritePaddingTo(w.BoundEndOr(0))
		})
		if fail && !errors.Is(err, sentinel) || !fail && err != nil {
			t.Fatalf("fail=%t: %v", fail, err)
		}
		if !reflect.DeepEqual(w.wire, before.wire) || w.spare != before.spare || w.padding != before.padding || w.terminal != before.terminal || w.bounded != before.bounded || w.limit != before.limit {
			t.Fatal("complete scope replaced parent state")
		}
	}
}

func TestWithCompleteWireValuePreservesBoundsAndTargets(t *testing.T) {
	for _, bound := range []int{8, 16, 24} {
		w := NewWriter()
		old, err := w.PushLimit(bound)
		if err != nil {
			t.Fatal(err)
		}
		err = w.WithCompleteWireValue(WireInfo{}, 16, func() error { return w.WritePaddingTo(w.BoundEndOr(0)) })
		if bound < 16 {
			if err == nil {
				t.Fatal("accepted a too-small parent bound")
			}
			continue
		}
		if err != nil || w.Position() != 16 || w.RemainingLimit() != bound-16 {
			t.Fatalf("bound=%d position=%d remaining=%d: %v", bound, w.Position(), w.RemainingLimit(), err)
		}
		if err := w.WritePaddingTo(bound); err != nil {
			t.Fatal(err)
		}
		if err := w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []int{8, 16, 24} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: target})
		err := w.WithCompleteWireValue(WireInfo{}, 16, func() error { return w.WritePaddingTo(w.BoundEndOr(0)) })
		if target < 16 {
			var failure *BoundError
			if !errors.As(err, &failure) || failure.Kind != CanonicalTarget {
				t.Fatalf("small target: %v", err)
			}
			continue
		}
		if err != nil || w.Position() != 16 {
			t.Fatalf("target=%d end=%d: %v", target, w.Position(), err)
		}
		if err := w.WritePadding(); err != nil {
			t.Fatal(err)
		}
		if w.Position() != target {
			t.Fatal("parent target was replaced")
		}
	}
}

func TestWithCompleteWireValueReplaysReceivedExtent(t *testing.T) {
	wire := Seal(struct{}{}, []byte{0}, 8, BitString{}, WireInfo{})
	w := NewWriter()
	if err := w.WithCompleteWireValue(wire, 168, func() error { return w.WriteUint(0, 8) }); err != nil {
		t.Fatal(err)
	}
	if w.Position() != 8 {
		t.Fatal("received short image expanded")
	}
	var nilWriter *Writer
	if err := nilWriter.WithCompleteWireValue(WireInfo{}, 168, func() error { return nil }); err == nil {
		t.Fatal("nil writer accepted")
	}
	if err := w.WithCompleteWireValue(WireInfo{}, 168, nil); err == nil {
		t.Fatal("nil callback accepted")
	}
	w = NewWriter()
	if err := w.WithCompleteWireValue(WireInfo{canonical: true, targetSet: true, targetBits: 167}, 168, func() error { return nil }); err == nil {
		t.Fatal("wrong explicit complete extent accepted")
	}
}

func TestWithCompleteWireValueResumesParentInferredSpans(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}, {At: 17, Count: 1}}})
	if err := w.WithCompleteWireValue(WireInfo{}, 16, func() error { return w.WriteUint(0, 16) }); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(1, 1); err == nil {
		t.Fatal("resumed parent failed to reject an edit to its later inferred span")
	}
}
