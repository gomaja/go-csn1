package runtime

import (
	"bytes"
	"testing"
)

// TS 44.060 V19.0.0 §§11.1.2–11.1.3: error labels are applied by
// receivers to received bits. A sender cannot fabricate or edit such a region.
func TestErrorRegionProvenance(t *testing.T) {
	fabricated := ErrorRegion{Label: Ignore, Location: Location{Width: 2}, Bits: BitString{Bytes: []byte{0x80}, BitLength: 2}}
	if err := NewWriter().WriteErrorRegion(fabricated); err == nil {
		t.Error("fabricated region accepted")
	}
	edits := []struct {
		name string
		edit func(*ErrorRegion)
	}{
		{"label", func(v *ErrorRegion) { v.Label = MessageEscape }},
		{"offset", func(v *ErrorRegion) { v.Location.BitOffset++ }},
		{"path", func(v *ErrorRegion) { v.Location.Path += "/changed" }},
		{"width", func(v *ErrorRegion) { v.Location.Width++ }},
		{"standard", func(v *ErrorRegion) { v.Location.Source.Standard += " changed" }},
		{"version", func(v *ErrorRegion) { v.Location.Source.Version += " changed" }},
		{"clause", func(v *ErrorRegion) { v.Location.Source.Clause += " changed" }},
		{"table", func(v *ErrorRegion) { v.Location.Source.Table += " changed" }},
		{"bits", func(v *ErrorRegion) { v.Bits.Bytes[0] ^= 0x80 }},
		{"length", func(v *ErrorRegion) { v.Bits.BitLength--; v.Location.Width-- }},
	}
	for _, tc := range edits {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReader([]byte{0xa5})
			v, err := r.ReadErrorRegion(Ignore, -1, fillSource)
			if err != nil {
				t.Fatal(err)
			}
			w := NewWriter()
			if err = w.WriteErrorRegion(v); err != nil {
				t.Fatal(err)
			}
			out, err := w.Bytes()
			if err != nil || !bytes.Equal(out, []byte{0xa5}) {
				t.Fatalf("replay %v", err)
			}
			tc.edit(&v)
			if err := NewWriter().WriteErrorRegion(v); err == nil {
				t.Fatal("edited receiver region accepted")
			}
		})
	}
}

func TestErrorRegionEmptyReceiptAndCopies(t *testing.T) {
	r := NewReader(nil)
	v, err := r.ReadErrorRegion(Ignore, 0, fillSource)
	if err != nil {
		t.Fatal(err)
	}
	copy := v
	if err := NewWriter().WriteErrorRegion(copy); err != nil {
		t.Fatal("received empty region", err)
	}
	fresh := ErrorRegion{Label: v.Label, Location: v.Location, Bits: v.Bits}
	if err := NewWriter().WriteErrorRegion(fresh); err == nil {
		t.Fatal("copied public fields fabricated provenance")
	}
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true})
	if err := w.WriteErrorRegion(v); err == nil {
		t.Fatal("canonical sender retained region")
	}
}

func FuzzErrorRegionProvenance(f *testing.F) {
	f.Add([]byte{0x2b}, uint8(0))
	f.Add([]byte{}, uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, index uint8) {
		if len(data) > 4096 {
			return
		}
		r := NewReader(data)
		v, err := r.ReadErrorRegion(Ignore, -1, Source{})
		if err != nil {
			t.Fatal(err)
		}
		w := NewWriter()
		if err = w.WriteErrorRegion(v); err != nil {
			t.Fatal(err)
		}
		out, err := w.Bytes()
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("received replay %v", err)
		}
		if len(v.Bits.Bytes) > 0 {
			v.Bits.Bytes[int(index)%len(v.Bits.Bytes)] ^= 1
			if err := NewWriter().WriteErrorRegion(v); err == nil {
				t.Fatal("mutated receiver bits accepted")
			}
		}
	})
}
