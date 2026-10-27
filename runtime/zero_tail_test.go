package runtime

import (
	"reflect"
	"testing"
)

func TestReaderAllRemainingZero(t *testing.T) {
	for bits := 0; bits <= 168; bits++ {
		data := make([]byte, 22)
		data[21] = 0xff
		r := NewReader(data)
		if _, err := r.PushLimit(bits); err != nil {
			t.Fatal(err)
		}
		before := *r
		steps := *r.steps
		if !r.AllRemainingZero() {
			t.Fatalf("zero extent %d", bits)
		}
		if !reflect.DeepEqual(before, *r) || *r.steps != steps {
			t.Fatal("lookahead changed reader")
		}
		if got := testing.AllocsPerRun(5, func() { r.AllRemainingZero() }); got != 0 {
			t.Fatalf("lookahead allocations %g", got)
		}
		for bit := 0; bit < bits; bit++ {
			data[bit/8] |= 1 << uint(7-bit%8)
			if r.AllRemainingZero() {
				t.Fatalf("extent %d ignored one at %d", bits, bit)
			}
			data[bit/8] = 0
		}
	}
	var invalid Reader
	if invalid.AllRemainingZero() || (*Reader)(nil).AllRemainingZero() {
		t.Fatal("invalid reader accepted")
	}
}

func TestReaderAllRemainingZeroStartsAtCurrentPosition(t *testing.T) {
	r := NewReader([]byte{0xff, 0})
	if _, err := r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	if !r.AllRemainingZero() || r.Position() != 8 {
		t.Fatal("lookahead included consumed prefix")
	}
}

func TestReaderAllRemainingZeroGrowth(t *testing.T) {
	for bits := 16; bits <= 1<<20; bits *= 2 {
		data := make([]byte, bits/8)
		r := NewReader(data)
		if !r.AllRemainingZero() {
			t.Fatalf("extent %d", bits)
		}
		if allocations := testing.AllocsPerRun(3, func() { r.AllRemainingZero() }); allocations != 0 {
			t.Fatalf("extent %d allocates %g", bits, allocations)
		}
		data[len(data)-1] = 1
		if r.AllRemainingZero() {
			t.Fatalf("extent %d missed last bit", bits)
		}
		if r.Position() != 0 || *r.steps != 0 || len(r.vars) != 0 {
			t.Fatal("scan changed reader state")
		}
	}
}
