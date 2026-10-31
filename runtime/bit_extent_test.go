package runtime

import (
	"bytes"
	"testing"
)

func TestBitExtentRejectsInvalidLength(t *testing.T) {
	for _, n := range []int{-1, 9, maxBits + 1} {
		if _, err := NewReaderBits([]byte{0xff}, n, nil); err == nil {
			t.Fatalf("accepted %d bits", n)
		}
	}
	for _, n := range []int{0, 8, 11, 15, 16} {
		r, err := NewReaderBits([]byte{0xa5, 0xff}, n, nil)
		if err != nil {
			t.Fatal(err)
		}
		if r.BoundedRemaining() != n {
			t.Fatalf("extent %d became %d", n, r.BoundedRemaining())
		}
		got, err := r.ReadBitString(n)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(1); err == nil {
			t.Fatal("read past significant end")
		}
		w := NewWriter()
		if err = w.WriteBitString(got, n); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(BitString{})
		if err != nil || out.BitLength != n || !bytes.Equal(out.Bytes, got.Bytes) {
			t.Fatalf("extent replay: %+v %v", out, err)
		}
	}
	for phase := 0; phase < 8; phase++ {
		r, err := NewReaderBits([]byte{0xff, 0xff}, phase+4, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(3)
		if err != nil {
			t.Fatal(err)
		}
		region, err := r.ReadErrorRegion(Ignore, -1, Source{})
		if err != nil || region.Location.BitOffset != phase || region.Bits.BitLength != 3 {
			t.Fatalf("phase %d: %+v %v", phase, region, err)
		}
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if v, err := r.ReadUint(1); err != nil || v != 1 {
			t.Fatalf("sentinel lost: %d %v", v, err)
		}
	}
}

func FuzzBitExtent(f *testing.F) {
	f.Add([]byte{0xa5, 0xff}, 11)
	f.Add([]byte{}, 0)
	f.Add([]byte{1}, -1)
	f.Fuzz(func(t *testing.T, data []byte, n int) {
		if len(data) > 4096 {
			t.Skip()
		}
		r, err := NewReaderBits(data, n, nil)
		valid := n >= 0 && n <= len(data)*8
		if (err == nil) != valid {
			t.Fatalf("valid=%v error=%v", valid, err)
		}
		if err != nil {
			return
		}
		value, err := r.ReadBitString(n)
		if err != nil {
			t.Fatal(err)
		}
		if r.Position() != n || r.Remaining() != 0 {
			t.Fatal("rounded extent")
		}
		w := NewWriter()
		if err = w.WriteBitString(value, n); err != nil {
			t.Fatal(err)
		}
		output, err := w.FinishBits(BitString{})
		if err != nil || output.BitLength != n || !bytes.Equal(output.Bytes, value.Bytes) {
			t.Fatal("bit replay differs")
		}
	})
}

func TestBitExtentLimitAndSealValidation(t *testing.T) {
	for _, construct := range []func([]byte, int, *Collector) (*Reader, error){NewReaderBits, NewChoiceReaderBits} {
		if _, err := construct(make([]byte, maxBits/8+1), 1, nil); err == nil {
			t.Fatal("oversize backing input accepted")
		}
		r, err := construct(make([]byte, maxBits/8), maxBits, nil)
		if err != nil || r.Remaining() != maxBits {
			t.Fatalf("maximum extent: %v", err)
		}
	}
	for _, tc := range []struct {
		raw      BitString
		consumed int
		tail     BitString
	}{
		{BitString{Bytes: []byte{0}, BitLength: 9}, 0, BitString{}},
		{BitString{Bytes: []byte{0}, BitLength: 3}, 4, BitString{}},
		{BitString{Bytes: []byte{0}, BitLength: 3}, 1, BitString{BitLength: 2}},
		{BitString{Bytes: []byte{0}, BitLength: 3}, 1, BitString{Bytes: []byte{0}, BitLength: 1}},
	} {
		if _, err := SealBits(struct{}{}, tc.raw, tc.consumed, tc.tail, WireInfo{}); err == nil {
			t.Fatalf("bad extent sealed: %+v", tc)
		}
	}
}
