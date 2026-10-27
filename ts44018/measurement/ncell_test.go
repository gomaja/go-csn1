package measurement

import (
	"bytes"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

// TS 44.018 V19.0.0 table 10.5.2.20.0 defines exactly 6/5/6 bits.
// The vectors were independently checked against the NCELL report in
// pycrate 0.7.11's measurement_results_contents grammar.
func TestNCELLReportStructOperations(t *testing.T) {
	for _, tc := range []struct {
		level, freq, bsic uint8
		wire              []byte
	}{
		{0, 0, 0, []byte{0, 0, 0}}, {63, 31, 63, []byte{0xff, 0xff, 0x80}},
		{45, 17, 37, []byte{0xb6, 0x32, 0x80}}, {12, 9, 18, []byte{0x31, 0x29, 0}},
	} {
		v := NCELLReportStruct{RXLEVNCELL: tc.level, BCCHFREQNCELL: tc.freq, BSICNCELL: tc.bsic}
		for _, encode := range []func(NCELLReportStruct) ([]byte, error){EncodeNCELLReportStruct, EncodeNCELLReportStructCanonical, func(v NCELLReportStruct) ([]byte, error) { return EncodeNCELLReportStructCanonicalAtLength(v, 3) }} {
			b, err := encode(v)
			if err != nil || !bytes.Equal(b, tc.wire) {
				t.Fatalf("fresh %+v: %x want %x: %v", v, b, tc.wire, err)
			}
		}
		received := append([]byte(nil), tc.wire...)
		received[2] |= 0x55
		d, err := DecodeNCELLReportStruct(received)
		if err != nil || d.BitsConsumed != 17 || d.Tail.BitLength != 7 || d.Value.RXLEVNCELL != tc.level || d.Value.BCCHFREQNCELL != tc.freq || d.Value.BSICNCELL != tc.bsic {
			t.Fatalf("decode %+v %v", d, err)
		}
		b, err := EncodeNCELLReportStruct(d.Value)
		if err != nil || !bytes.Equal(b, received) {
			t.Fatalf("replay %x %v", b, err)
		}
		b, err = EncodeNCELLReportStructCanonical(d.Value)
		if err != nil || !bytes.Equal(b, tc.wire) {
			t.Fatalf("canonical %x %v", b, err)
		}
	}
	for _, v := range []NCELLReportStruct{{RXLEVNCELL: 64}, {BCCHFREQNCELL: 32}, {BSICNCELL: 64}} {
		if _, err := EncodeNCELLReportStruct(v); err == nil {
			t.Fatalf("accepted overflow %+v", v)
		}
		if _, err := EncodeNCELLReportStructCanonical(v); err == nil {
			t.Fatal("canonical overflow")
		}
	}
	for _, n := range []int{-1, 0, 1, 2, 4} {
		if _, err := EncodeNCELLReportStructCanonicalAtLength(NCELLReportStruct{}, n); err == nil {
			t.Fatalf("accepted %d octets", n)
		}
	}
}

func TestNCELLReportStructNestedOriginsAndTruncations(t *testing.T) {
	v := NCELLReportStruct{RXLEVNCELL: 45, BCCHFREQNCELL: 17, BSICNCELL: 37}
	for _, origin := range []int{0, 3, 11} {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, origin); err != nil {
			t.Fatal(err)
		}
		if err := EncodeNCELLReportStructTo(w, v); err != nil {
			t.Fatal(err)
		}
		if w.Position() != origin+17 {
			t.Fatalf("writer position %d", w.Position())
		}
		if err := w.PadMinimum((origin+17+7)/8*8, false); err != nil {
			t.Fatal(err)
		}
		wire, err := w.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		for bits := 0; bits <= 17; bits++ {
			r := runtime.NewReader(wire)
			if _, err := r.ReadUint(origin); err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(bits); err != nil {
				t.Fatal(err)
			}
			got, err := DecodeNCELLReportStructFrom(r)
			if bits == 17 {
				if err != nil || got.RXLEVNCELL != 45 || got.BCCHFREQNCELL != 17 || got.BSICNCELL != 37 || r.Position() != origin+17 {
					t.Fatalf("nested %+v %v", got, err)
				}
				continue
			}
			var de *runtime.DecodeError
			var op *runtime.OperationError
			if !errors.As(err, &de) || de.Kind != runtime.Truncated || de.Offset != origin+bits || !errors.As(err, &op) || op.Location.BitOffset != origin+bits || op.Location.Source.Clause != "10.5.2.20" {
				t.Fatalf("origin %d bits %d: %v", origin, bits, err)
			}
		}
	}
}
