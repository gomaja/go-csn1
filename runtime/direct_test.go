package runtime

import (
	"bytes"
	"testing"
)

func TestDirectBitIOBoundsAndPaddingOrigin(t *testing.T) {
	if LHBit('L', 0) != 0 || LHBit('H', 0) != 1 || LHBit('L', 2) != 1 || LHBit('H', 2) != 0 {
		t.Fatal("L/H constraint must follow the absolute padding-pattern position")
	}
	r := NewReader([]byte{0x2b})
	for _, symbol := range []string{"L", "L", "L", "L", "L", "L", "L", "L"} {
		if err := r.Expect(symbol); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.ReadUint(1); err == nil {
		t.Fatal("read beyond input")
	}
	w := NewWriter()
	if err := w.WriteLiteral("LLLLLLLL"); err != nil {
		t.Fatal(err)
	}
	if got, err := w.Bytes(); err != nil || !bytes.Equal(got, []byte{0x2b}) {
		t.Fatalf("bytes=%x err=%v", got, err)
	}
}

func TestDirectReaderChoiceSnapshot(t *testing.T) {
	r := NewReader([]byte{0x80})
	a := r.Fork()
	if err := a.Expect("0"); err == nil {
		t.Fatal("wrong branch accepted")
	}
	b := r.Fork()
	if err := b.Expect("1"); err != nil {
		t.Fatal(err)
	}
	r.Commit(b)
	if r.Position() != 1 {
		t.Fatalf("position=%d", r.Position())
	}
}

func TestWireReplayRejectsModifiedOriginal(t *testing.T) {
	value := struct{ Field uint8 }{Field: 3}
	wire := Seal(value, []byte{0x30}, 4, BitString{}, WireInfo{})
	wire.Original[0] = 0xff
	if _, ok := OriginalIfUnchanged(value, wire); ok {
		t.Fatal("modified original replayed")
	}
}
