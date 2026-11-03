package runtime

import "testing"

func TestErrorRegionReceiptStorageBoundaries(t *testing.T) {
	for _, size := range []int{0, 1, 32, 64, 128, 129, 1024} {
		input := make([]byte, size)
		if size > 0 {
			input[size-1] = 0xa5
		}
		region, err := NewReader(input).ReadErrorRegion(Ignore, -1, fillSource)
		if err != nil {
			t.Fatal(err)
		}
		if len(region.Bits.Bytes) != cap(region.Bits.Bytes) {
			t.Fatal("public slice exposes private receipt")
		}
		if err := NewWriter().WriteErrorRegion(region); err != nil {
			t.Fatal(err)
		}
		if size > 0 {
			region.Bits.Bytes[size-1] ^= 1
			if err := NewWriter().WriteErrorRegion(region); err == nil {
				t.Fatal("edited last byte accepted")
			}
		} else {
			region.Label = MessageEscape
			if err := NewWriter().WriteErrorRegion(region); err == nil {
				t.Fatal("edited empty region accepted")
			}
		}
	}
}

// Receiving the receipt must not allocate another object for up to 128
// octets, including an empty arm. Inputs and readers are outside the window.
func TestErrorRegionReceiptAllocationCount(t *testing.T) {
	for _, size := range []int{0, 1, 32, 64, 128, 129, 1024} {
		input := make([]byte, size)
		r := NewReader(input)
		count := testing.AllocsPerRun(100, func() {
			r.pos = 0
			*r.steps = 0
			v, err := r.ReadErrorRegion(Ignore, -1, Source{})
			if err != nil {
				panic(err)
			}
			regionStorageSink = v
		})
		want := float64(1)
		if size == 0 {
			want = 0
		}
		if size > 128 {
			want = 2
		}
		if !raceEnabled && count != want {
			t.Errorf("%d octets: %g allocations, want %g", size, count, want)
		}
	}
}
