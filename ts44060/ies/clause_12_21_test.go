package ies

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func frameNumberPack(text string) []byte {
	b := make([]byte, (len(text)+7)/8)
	for i, c := range text {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}

// TS 44.060 V19.0.0 Tables 12.21.1–12.21.3, paragraphs
// 23298–23309: one selector followed by the quoted 16-bit or 13-bit field.
func TestStartingFrameNumberDescription(t *testing.T) {
	for arm, width := range []int{16, 13} {
		for _, number := range []uint16{0, 1, 21, uint16((1 << width) - 1)} {
			t.Run(fmt.Sprintf("arm-%d-value-%d", arm, number), func(t *testing.T) {
				text := fmt.Sprintf("%01b%0*b", arm, width, number)
				wire := frameNumberPack(text)
				var fresh StartingFrameNumberDescriptionIE
				if arm == 0 {
					fresh.AbsoluteFrameNumberEncodingChoice.AbsoluteFrameNumberEncoding = &StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoiceAbsoluteFrameNumberEncoding{AbsoluteFrameNumberEncoding: number}
				} else {
					fresh.AbsoluteFrameNumberEncodingChoice.Alternative = StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoiceAlternativeRelativeFrameNumberEncoding
					fresh.AbsoluteFrameNumberEncodingChoice.RelativeFrameNumberEncoding = &StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoiceRelativeFrameNumberEncoding{RelativeFrameNumberEncoding: number}
				}
				got, err := EncodeStartingFrameNumberDescriptionIE(fresh)
				if err != nil || !bytes.Equal(got, wire) {
					t.Fatalf("fresh %x want %x: %v", got, wire, err)
				}
				decoded, err := DecodeStartingFrameNumberDescriptionIE(wire)
				if err != nil || decoded.BitsConsumed != len(text) {
					t.Fatalf("decode extent %d: %v", decoded.BitsConsumed, err)
				}
				choice := decoded.Value.AbsoluteFrameNumberEncodingChoice
				if int(choice.Alternative) != arm {
					t.Fatalf("typed alternative: %+v", choice)
				}
				if arm == 0 && (choice.AbsoluteFrameNumberEncoding == nil || choice.AbsoluteFrameNumberEncoding.AbsoluteFrameNumberEncoding != number || choice.RelativeFrameNumberEncoding != nil) {
					t.Fatalf("absolute: %+v", choice)
				}
				if arm == 1 && (choice.RelativeFrameNumberEncoding == nil || choice.RelativeFrameNumberEncoding.RelativeFrameNumberEncoding != number || choice.AbsoluteFrameNumberEncoding != nil) {
					t.Fatalf("relative: %+v", choice)
				}
				replay, err := EncodeStartingFrameNumberDescriptionIE(decoded.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x: %v", replay, err)
				}
				canonical, err := EncodeStartingFrameNumberDescriptionIECanonical(decoded.Value)
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
				}
				edited := decoded.Value
				edited.Wire = runtime.WireInfo{}
				next := number ^ 1
				if arm == 0 {
					v := *choice.AbsoluteFrameNumberEncoding
					v.AbsoluteFrameNumberEncoding = next
					edited.AbsoluteFrameNumberEncodingChoice.AbsoluteFrameNumberEncoding = &v
				} else {
					v := *choice.RelativeFrameNumberEncoding
					v.RelativeFrameNumberEncoding = next
					edited.AbsoluteFrameNumberEncodingChoice.RelativeFrameNumberEncoding = &v
				}
				changed, err := EncodeStartingFrameNumberDescriptionIE(edited)
				if err != nil || !bytes.Equal(changed, frameNumberPack(fmt.Sprintf("%01b%0*b", arm, width, next))) {
					t.Fatalf("typed edit %x: %v", changed, err)
				}
				for phase := 0; phase < 8; phase++ {
					r := runtime.NewReader(frameNumberPack(strings.Repeat("0", phase) + text + "10101"))
					if _, err := r.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(len(text))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := DecodeStartingFrameNumberDescriptionIEFrom(r); err != nil || r.Remaining() != 0 {
						t.Fatalf("phase %d: %v", phase, err)
					}
					if err := r.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					sentinel, err := r.ReadUint(5)
					if err != nil || sentinel != 21 {
						t.Fatalf("sentinel %d: %v", sentinel, err)
					}
				}
				for cut := 0; cut < len(text); cut++ {
					r := runtime.NewReader(wire)
					if _, err := r.PushLimit(cut); err != nil {
						t.Fatal(err)
					}
					if _, err := DecodeStartingFrameNumberDescriptionIEFrom(r); err == nil {
						t.Fatalf("strict prefix %d accepted", cut)
					}
					if r.Position() > cut {
						t.Fatalf("crossed prefix %d", cut)
					}
				}
			})
		}
	}
	invalid := StartingFrameNumberDescriptionIE{AbsoluteFrameNumberEncodingChoice: StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoice{Alternative: 2}}
	if _, err := EncodeStartingFrameNumberDescriptionIE(invalid); err == nil {
		t.Fatal("unknown arm encoded")
	}
	relative := StartingFrameNumberDescriptionIE{AbsoluteFrameNumberEncodingChoice: StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoice{Alternative: StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoiceAlternativeRelativeFrameNumberEncoding, RelativeFrameNumberEncoding: &StartingFrameNumberDescriptionIEAbsoluteFrameNumberEncodingChoiceRelativeFrameNumberEncoding{RelativeFrameNumberEncoding: 8192}}}
	if _, err := EncodeStartingFrameNumberDescriptionIE(relative); err == nil {
		t.Fatal("14-bit relative value encoded")
	}
}

func FuzzStartingFrameNumberReplay(f *testing.F) {
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{0xff, 0xff})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		v, err := DecodeStartingFrameNumberDescriptionIE(wire)
		if err != nil {
			return
		}
		got, err := EncodeStartingFrameNumberDescriptionIE(v.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x want %x: %v", got, wire, err)
		}
	})
}
