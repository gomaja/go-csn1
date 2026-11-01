package ies

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func lsaPack(text string) []byte {
	b := make([]byte, (len(text)+7)/8)
	for i, c := range text {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}

func lsaGroup(index int, pattern string) (LSAIDInformationStruct, string) {
	var v LSAIDInformationStruct
	text := ""
	count := 1
	if pattern == "empty" {
		count = 0
	}
	if pattern == "mixed" {
		count = 3
	}
	for j := 0; j < count; j++ {
		short := pattern == "short" || pattern == "mixed" && j%2 == 1
		value := uint32(index*37 + j*11 + 1)
		var entry LSAIDInformationStructLSAIDListEntry
		if short {
			value &= 1023
			entry.LSAIDChoice.Alternative = LSAIDInformationStructLSAIDListEntryLSAIDChoiceAlternativeShortLSAID
			entry.LSAIDChoice.ShortLSAID = &LSAIDInformationStructLSAIDListEntryLSAIDChoiceShortLSAID{ShortLSAID: uint16(value)}
			text += "11" + fmt.Sprintf("%010b", value)
		} else {
			entry.LSAIDChoice.LSAID = &LSAIDInformationStructLSAIDListEntryLSAIDChoiceLSAID{LSAID: value}
			text += "10" + fmt.Sprintf("%024b", value)
		}
		v.LSAIDList = append(v.LSAIDList, entry)
	}
	return v, text + "0"
}

// TS 44.060 V19.0.0 Table 12.28.1, paragraphs 23502–23505.
// The approved paragraph-23503 receipt removes only the stray colon:
// the five-bit count repeats terminated LSA lists, whose arms have 24/10 bits.
func TestLSAParameters(t *testing.T) {
	for _, count := range []int{0, 1, 2, 7, 31} {
		for _, pattern := range []string{"empty", "long", "short", "mixed"} {
			t.Run(fmt.Sprintf("count-%d-%s", count, pattern), func(t *testing.T) {
				var fresh LSAParametersIE
				text := fmt.Sprintf("%05b", count)
				for i := 0; i < count; i++ {
					v, s := lsaGroup(i, pattern)
					fresh.LSAIDInformationList = append(fresh.LSAIDInformationList, v)
					text += s
				}
				if err := fresh.SetLSAIDInformationList(fresh.LSAIDInformationList); err != nil {
					t.Fatal(err)
				}
				wire := lsaPack(text)
				encoded, err := EncodeLSAParametersIE(fresh)
				if err != nil || !bytes.Equal(encoded, wire) {
					t.Fatalf("fresh %x want %x: %v", encoded, wire, err)
				}
				decoded, err := DecodeLSAParametersIE(wire)
				if err != nil || decoded.BitsConsumed != len(text) || int(decoded.Value.NROFFREQORCELLS) != count || len(decoded.Value.LSAIDInformationList) != count {
					t.Fatalf("typed count/extent: %+v %v", decoded, err)
				}
				for i, group := range decoded.Value.LSAIDInformationList {
					expected, _ := lsaGroup(i, pattern)
					if len(group.LSAIDList) != len(expected.LSAIDList) {
						t.Fatal("typed LSA list length")
					}
					for j, item := range group.LSAIDList {
						want := expected.LSAIDList[j].LSAIDChoice
						got := item.LSAIDChoice
						if got.Alternative != want.Alternative {
							t.Fatal("typed selector")
						}
						if want.LSAID != nil && (got.LSAID == nil || got.LSAID.LSAID != want.LSAID.LSAID) {
							t.Fatal("24-bit LSA value")
						}
						if want.ShortLSAID != nil && (got.ShortLSAID == nil || got.ShortLSAID.ShortLSAID != want.ShortLSAID.ShortLSAID) {
							t.Fatal("10-bit short LSA value")
						}
					}
				}
				replay, err := EncodeLSAParametersIE(decoded.Value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("replay %x: %v", replay, err)
				}
				canonical, err := EncodeLSAParametersIECanonicalAtLength(decoded.Value, len(wire))
				if err != nil || !bytes.Equal(canonical, wire) {
					t.Fatalf("canonical %x: %v", canonical, err)
				}
				edit := decoded.Value
				edit.Wire = runtime.WireInfo{}
				if err := edit.SetLSAIDInformationList(nil); err != nil {
					t.Fatal(err)
				}
				changed, err := EncodeLSAParametersIE(edit)
				if err != nil || !bytes.Equal(changed, []byte{0}) {
					t.Fatalf("typed count edit %x: %v", changed, err)
				}
				for phase := 0; phase < 8; phase++ {
					r := runtime.NewReader(lsaPack(strings.Repeat("1", phase) + text + "10101"))
					if _, err := r.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(len(text))
					if err != nil {
						t.Fatal(err)
					}
					if _, err := DecodeLSAParametersIEFrom(r); err != nil || r.Remaining() != 0 {
						t.Fatalf("bounded phase %d: %v", phase, err)
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
					if _, err := DecodeLSAParametersIEFrom(r); err == nil {
						t.Fatalf("strict prefix %d/%d accepted", cut, len(text))
					}
					if r.Position() > cut {
						t.Fatalf("crossed bound %d", cut)
					}
				}
			})
		}
	}
	v := LSAParametersIE{}
	if err := v.SetLSAIDInformationList(make([]LSAIDInformationStruct, 32)); err == nil {
		t.Fatal("32-entry count accepted")
	}
	group, _ := lsaGroup(0, "short")
	group.LSAIDList[0].LSAIDChoice.ShortLSAID.ShortLSAID = 1024
	if _, err := EncodeLSAIDInformationStruct(group); err == nil {
		t.Fatal("11-bit short LSA encoded")
	}
	group, _ = lsaGroup(0, "long")
	group.LSAIDList[0].LSAIDChoice.LSAID.LSAID = 1 << 24
	if _, err := EncodeLSAIDInformationStruct(group); err == nil {
		t.Fatal("25-bit LSA encoded")
	}
	if _, err := DecodeLSAIDInformationStruct([]byte{0xff}); err == nil {
		t.Fatal("unterminated/truncated entry accepted")
	}
}

func TestLSAListGrowth(t *testing.T) {
	var before float64
	for _, count := range []int{8, 16, 32, 64} {
		text := strings.Repeat("11"+strings.Repeat("1", 10), count) + "0"
		wire := lsaPack(text)
		allocs := testing.AllocsPerRun(3, func() {
			if _, err := DecodeLSAIDInformationStruct(wire); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && before != 0 && allocs > before*2+8 {
			t.Fatalf("nonlinear list allocations: %g after %g", allocs, before)
		}
		before = allocs
	}
}

func TestLSAParametersIECanonicalRequiresExtent(t *testing.T) {
	_, err := EncodeLSAParametersIECanonical(LSAParametersIE{})
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || !extent.Required {
		t.Fatalf("required canonical extent: %v", err)
	}
}

func FuzzLSAParametersReplay(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{8, 0})
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		v, err := DecodeLSAParametersIE(wire)
		if err != nil {
			return
		}
		got, err := EncodeLSAParametersIE(v.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x want %x: %v", got, wire, err)
		}
	})
}
