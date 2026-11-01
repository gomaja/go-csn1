package packetdownlinkassignment_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	subject "github.com/gomaja/go-csn1/ts44060/packetdownlinkassignment"
	"reflect"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.7 pp14955–14958 supplies the approved
// Additional PFC dependency of §11.2.31. This does not publish the message root.
func TestAdditionalPFCDependency(t *testing.T) {
	for tfi := uint8(0); tfi < 32; tfi++ {
		for _, pfi := range []uint8{0, 1, 127} {
			for _, withNPM := range []bool{false, true} {
				v := subject.AdditionalPFCsStruct{DOWNLINKTFIASSIGNMENT: tfi, PFI: pfi}
				if withNPM {
					npm := uint8(31)
					v.NPMTransferTime = &npm
				}
				for phase := 0; phase < 8; phase++ {
					w := runtime.NewWriter()
					if err := w.WriteUint(0, phase); err != nil {
						t.Fatal(err)
					}
					start := w.Position()
					if err := subject.EncodeAdditionalPFCsStructTo(w, v); err != nil {
						t.Fatal(err)
					}
					end := w.Position()
					want := 13
					if withNPM {
						want = 18
					}
					if end-start != want {
						t.Fatal("source width")
					}
					if err := w.WriteUint(6, 3); err != nil {
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
					if _, err = r.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(want)
					if err != nil {
						t.Fatal(err)
					}
					got, err := subject.DecodeAdditionalPFCsStructFrom(r)
					if err != nil || got.DOWNLINKTFIASSIGNMENT != tfi || got.PFI != pfi || (got.NPMTransferTime == nil) != (!withNPM) {
						t.Fatalf("typed dependency: %+v %v", got, err)
					}
					if withNPM && *got.NPMTransferTime != 31 {
						t.Fatal("NPM value")
					}
					if err = r.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					sentinel, err := r.ReadUint(3)
					if err != nil || sentinel != 6 {
						t.Fatal("dependency exceeded containing bound")
					}
				}
			}
		}
	}
}

func FuzzAdditionalPFCDependency(f *testing.F) {
	f.Add([]byte{0, 0}, uint16(13))
	f.Add([]byte{0xff, 0xff, 0xc0}, uint16(18))
	f.Fuzz(func(t *testing.T, data []byte, length uint16) {
		if len(data) > 64 {
			return
		}
		bits := int(length)
		if bits > len(data)*8 {
			return
		}
		for _, d := range subject.Descriptors() {
			plain, pe := d.DecodeBits(data, bits)
			reported, _, _, re := d.DecodeBitsWithReport(data, bits)
			if fmt.Sprint(pe) != fmt.Sprint(re) || !reflect.DeepEqual(plain, reported) {
				t.Fatalf("collector changes dependency: %v / %v", pe, re)
			}
			if pe == nil && bits%8 == 0 {
				out, err := d.Encode(plain)
				if err != nil || !bytes.Equal(out, data[:bits/8]) {
					t.Fatalf("received dependency replay: %v", err)
				}
			}
		}
	})
}
