package registry

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func sourceReader(t testing.TB, d runtime.Descriptor, data []byte) *runtime.Reader {
	t.Helper()
	origin := SourceOrigin(d)
	if origin == 0 {
		return runtime.NewReader(data)
	}
	bits := len(data) * 8
	if d.Extent.Fixed && d.Extent.MaximumBits%8 != 0 && bits == (d.Extent.MaximumBits+7)/8*8 {
		bits = d.Extent.MaximumBits
	}
	w := runtime.NewWriter()
	if err := w.WriteUint(0, origin); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteBitString(runtime.BitString{Bytes: data, BitLength: bits}, bits); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := runtime.NewReaderBits(raw.Bytes, raw.BitLength, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	return r
}
