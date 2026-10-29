package runtime

import (
	"reflect"
	"testing"
)

func TestCanonicalEmptyBitmapSemantics(t *testing.T) {
	// TS 44.060 V19.0.0 §12.29 paragraphs 23515–23520: a zero
	// bitmap length transmits no bitmap bits and means no reduced MA.
	empty := BitString{Bytes: []byte{}, BitLength: 0}
	if !SemanticallyEqual(BitString{}, empty) {
		t.Fatal("nil and empty zero-width bitmaps differ")
	}
	nested := struct{ Bitmap *BitString }{&empty}
	zero := BitString{}
	if !SemanticallyEqual(nested, struct{ Bitmap *BitString }{&zero}) {
		t.Fatal("nested empty bitmaps differ")
	}
	if !SemanticallyEqual(Canonical(empty), BitString{}) {
		t.Fatal("canonical empty bitmap differs")
	}
	if empty.Bytes == nil {
		t.Fatal("canonical normalization mutated the source")
	}
	for _, other := range []BitString{{Bytes: []byte{0}, BitLength: 0}, {Bytes: []byte{}, BitLength: 1}, {Bytes: []byte{0}, BitLength: 1}} {
		if SemanticallyEqual(BitString{}, other) {
			t.Fatalf("accepted a different or malformed bitmap: %+v", other)
		}
	}
	// Other slices can carry application state; this rule is limited to BitString.
	if SemanticallyEqual([]uint8(nil), []uint8{}) {
		t.Fatal("unrelated nil and empty slices merged")
	}
	if reflect.DeepEqual(empty.Bytes, Canonical(empty).Bytes) {
		t.Fatal("canonical copy did not normalize its zero-width bitmap")
	}
}
