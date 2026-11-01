package ies_test

import (
	"bytes"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"testing"
)

// TS 44.060 V19.0.0 §12.51 table 12.51.1, pp25304–25309:
// the five fields occupy 3+4+4+2+2 = 15 bits, in this order.
func TestClause12_51PrintedFields(t *testing.T) {
	for code := uint16(0); ; code++ {
		raw := []byte{byte(code >> 7), byte(code << 1)}
		decoded, err := ies.DecodeGSMPriorityParametersIE(raw)
		if err != nil {
			t.Fatal(err)
		}
		v := decoded.Value
		want := ies.GSMPriorityParametersIE{GERANPRIORITY: uint8(code >> 12), THRESHPrioritySearch: uint8(code>>8) & 15, THRESHGSMLow: uint8(code>>4) & 15, HPRIO: uint8(code>>2) & 3, TReselection: uint8(code) & 3}
		if v.GERANPRIORITY != want.GERANPRIORITY || v.THRESHPrioritySearch != want.THRESHPrioritySearch || v.THRESHGSMLow != want.THRESHGSMLow || v.HPRIO != want.HPRIO || v.TReselection != want.TReselection {
			t.Fatalf("printed fields for %d: %+v", code, v)
		}
		got, err := ies.EncodeGSMPriorityParametersIE(v)
		if err != nil || !bytes.Equal(raw, got) {
			t.Fatalf("replay for %d: %v", code, err)
		}
		got, err = ies.EncodeGSMPriorityParametersIECanonical(want)
		if err != nil || !bytes.Equal(raw, got) {
			t.Fatalf("canonical for %d: %v", code, err)
		}
		if code == 32767 {
			break
		}
	}
}

func TestClause12_51CallerCoordinateAndBounds(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		v := ies.GSMPriorityParametersIE{GERANPRIORITY: 5, THRESHPrioritySearch: 9, THRESHGSMLow: 6, HPRIO: 2, TReselection: 3}
		if err := ies.EncodeGSMPriorityParametersIETo(w, v); err != nil {
			t.Fatal(err)
		}
		b, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		for _, extra := range []int{0, 1, 8, 64, 1024, 4096} {
			raw := append(append([]byte(nil), b.Bytes...), bytes.Repeat([]byte{0xa5}, extra)...)
			r, err := runtime.NewReaderBits(raw, phase+15, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			got, err := ies.DecodeGSMPriorityParametersIEFrom(r)
			if err != nil || got.GERANPRIORITY != 5 || got.THRESHPrioritySearch != 9 || got.THRESHGSMLow != 6 || got.HPRIO != 2 || got.TReselection != 3 || r.Position() != phase+15 {
				t.Fatalf("coordinate %d, sentinel %d: %+v %v", phase, extra, got, err)
			}
		}
		// Editing the last typed field changes exactly its two-bit value.
		r, err := runtime.NewReaderBits(b.Bytes, phase+15, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		edited, err := ies.DecodeGSMPriorityParametersIEFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		edited.TReselection ^= 1
		ew := runtime.NewWriter()
		if err = ew.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		if err = ies.EncodeGSMPriorityParametersIETo(ew, edited); err != nil {
			t.Fatal(err)
		}
		eb, err := ew.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		er, err := runtime.NewReaderBits(eb.Bytes, phase+15, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = er.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		ev, err := ies.DecodeGSMPriorityParametersIEFrom(er)
		if err != nil || ev.TReselection != 2 || ev.GERANPRIORITY != 5 {
			t.Fatalf("edited field: %+v %v", ev, err)
		}
		for end := phase; end < phase+15; end++ {
			r, err := runtime.NewReaderBits(b.Bytes, end, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			if _, err = ies.DecodeGSMPriorityParametersIEFrom(r); err == nil {
				t.Fatalf("truncation at %d accepted", end)
			}
		}
	}
}

func FuzzClause12_51Replay(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{0xff, 0xfe})
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 2 {
			return
		}
		decoded, err := ies.DecodeGSMPriorityParametersIE(raw)
		if err != nil {
			return
		}
		got, err := ies.EncodeGSMPriorityParametersIE(decoded.Value)
		if err != nil || !bytes.Equal(raw, got) {
			t.Fatalf("replay: %v", err)
		}
	})
}
