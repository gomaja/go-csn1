package ies

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.8b pp14819–14843; the imported
// Channel Request Description is TS 44.060 V19.0.0 §12.7 pp22666–22671.
// Each release suffix has its printed null, L and H alternatives.
func channelRequestPack(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

func channelRequestSuffix(depth int, low bool, origin int) string {
	var b strings.Builder
	layout := func(high bool) {
		bit := "00101011"[(origin+b.Len())%8]
		if high {
			bit ^= 1
		}
		b.WriteByte(bit)
	}
	for i := 0; i < depth; i++ {
		layout(true)
		if i == 0 {
			b.WriteString("101011")
		} else {
			b.WriteByte('1')
		}
	}
	if low {
		layout(false)
	}
	return b.String()
}

func checkChannelRequestCapabilities(t *testing.T, v ChannelRequestDescription2ValuePart, depth int, low bool) {
	t.Helper()
	first := v.RLCNonPersistentModeCapabilityChoice
	if depth == 0 {
		want := ChannelRequestDescription2ValuePartRLCNonPersistentModeCapabilityChoiceAlternativeAltUnlabeled
		if low {
			want = ChannelRequestDescription2ValuePartRLCNonPersistentModeCapabilityChoiceAlternativeAltL
		}
		if first.Alternative != want {
			t.Fatalf("first selector %v want %v", first.Alternative, want)
		}
		return
	}
	if first.RLCNonPersistentModeCapability == nil {
		t.Fatal("missing Rel-7 fields")
	}
	a := first.RLCNonPersistentModeCapability
	if a.Group.RLCNonPersistentModeCapability != 1 || a.Group.ReducedLatencyCapability != 0 || a.Group.UplinkEGPRS2 != 2 || a.Group.DownlinkEGPRS2 != 3 {
		t.Fatalf("Rel-7 fields %+v", a.Group)
	}
	b := a.EMSTMSCapabilityChoice
	if depth == 1 {
		if (b.AltL != nil) != low || (b.AltUnlabeled != nil) == low {
			t.Fatal("Rel-9 terminal selector")
		}
		return
	}
	if b.EMSTMSCapability == nil || b.EMSTMSCapability.EMSTMSCapability != 1 {
		t.Fatal("missing Rel-9 fields")
	}
	c := b.EMSTMSCapability.EMSRMSCapabilityChoice
	if depth == 2 {
		if (c.AltL != nil) != low || (c.AltUnlabeled != nil) == low {
			t.Fatal("Rel-10 terminal selector")
		}
		return
	}
	if c.EMSRMSCapability == nil || c.EMSRMSCapability.EMSRMSCapability != 1 {
		t.Fatal("missing Rel-10 fields")
	}
	d := c.EMSRMSCapability.FANRCapabilityChoice
	if depth == 3 {
		if (d.AltL != nil) != low || (d.AltUnlabeled != nil) == low {
			t.Fatal("Rel-11 terminal selector")
		}
		return
	}
	if d.FANRCapability == nil || d.FANRCapability.FANRCapability != 1 {
		t.Fatal("missing Rel-11 fields")
	}
}

func TestChannelRequestDescription2ReleaseArms(t *testing.T) {
	for cause := 0; cause < 4; cause++ {
		for pfi := 0; pfi < 2; pfi++ {
			for depth := 0; depth <= 4; depth++ {
				for terminal := 0; terminal < 2; terminal++ {
					if depth == 4 && terminal == 1 {
						continue
					}
					for phase := 0; phase < 8; phase++ {
						t.Run(fmt.Sprintf("cause%d/pfi%d/depth%d/low%d/phase%d", cause, pfi, depth, terminal, phase), func(t *testing.T) {
							prefix := strings.Repeat("1", phase)
							body := fmt.Sprintf("%02b", cause) + "101010100001001000110100" + "0" + "1"
							if pfi == 1 {
								body = fmt.Sprintf("%02b", cause) + "101010100001001000110100" + "1" + "1010101" + "1"
							}
							body += channelRequestSuffix(depth, terminal == 1, phase+len(body))
							bits := prefix + body
							wire := channelRequestPack(bits + "10101")
							r := runtime.NewReader(wire)
							if _, err := r.ReadUint(phase); err != nil {
								t.Fatal(err)
							}
							old, err := r.PushLimit(len(body))
							if err != nil {
								t.Fatal(err)
							}
							v, err := DecodeChannelRequestDescription2ValuePartFrom(r)
							if err != nil {
								t.Fatal(err)
							}
							if r.Position() != len(bits) || v.PACKETESTABLISHMENTCAUSE != uint8(cause) || v.MultipleTBFCapability != 1 {
								t.Fatalf("root position=%d value=%+v", r.Position(), v)
							}
							q := v.ChannelRequestDescription
							if q.PEAKTHROUGHPUTCLASS != 10 || q.RADIOPRIORITY != 2 || q.RLCMODE != 1 || q.LLCPDUTYPE != 0 || q.RLCOCTETCOUNT != 0x1234 {
								t.Fatalf("imported fields %+v", q)
							}
							if (v.PFI != nil) != (pfi == 1) || (v.PFI != nil && *v.PFI != 85) {
								t.Fatal("PFI lost")
							}
							checkChannelRequestCapabilities(t, v, depth, terminal == 1)
							layout, sealErr := runtime.SealBits(v, runtime.BitString{Bytes: channelRequestPack(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
							if sealErr != nil {
								t.Fatal(sealErr)
							}
							if err := r.PopLimit(old); err != nil {
								t.Fatal(err)
							}
							sentinel, err := r.ReadUint(5)
							if err != nil || sentinel != 21 {
								t.Fatalf("following sentinel %d %v", sentinel, err)
							}
							for _, canonical := range []bool{false, true} {
								value := v
								if canonical {
									value = runtime.Canonical(v)
								}
								w := runtime.NewWriter()
								w.WithWire(layout)
								if canonical {
									w.WithWire(runtime.CanonicalTo(value, len(bits)).Wire)
								}
								if err := w.WriteLiteral(prefix); err != nil {
									t.Fatal(err)
								}
								bound, err := w.PushLimit(len(body))
								if err != nil {
									t.Fatal(err)
								}
								if err := EncodeChannelRequestDescription2ValuePartTo(w, value); err != nil {
									t.Fatalf("canonical=%v: %v", canonical, err)
								}
								if err := w.PopLimit(bound); err != nil {
									t.Fatal(err)
								}
								encoded, err := w.FinishBits(runtime.BitString{})
								if err != nil || encoded.BitLength != len(bits) || !bytes.Equal(encoded.Bytes, channelRequestPack(bits)) {
									t.Fatalf("canonical=%v got %x/%d %v", canonical, encoded.Bytes, encoded.BitLength, err)
								}
							}
							v.PACKETESTABLISHMENTCAUSE = uint8(cause) ^ 3
							w := runtime.NewWriter()
							w.WithWire(layout)
							if err := w.WriteLiteral(prefix); err != nil {
								t.Fatal(err)
							}
							if _, err := w.PushLimit(len(body)); err != nil {
								t.Fatal(err)
							}
							if err := EncodeChannelRequestDescription2ValuePartTo(w, v); err != nil {
								t.Fatal(err)
							}
							edited, err := w.FinishBits(runtime.BitString{})
							expected := prefix + fmt.Sprintf("%02b", cause^3) + body[2:]
							if err != nil || !bytes.Equal(edited.Bytes, channelRequestPack(expected)) {
								t.Fatalf("edit %x want %x: %v", edited.Bytes, channelRequestPack(expected), err)
							}
						})
					}
				}
			}
		}
	}
}

func TestChannelRequestDescription2Truncation(t *testing.T) {
	// The first 28 bits precede the release-dependent suffix: 2+24+1+1.
	const mandatory = "0010101010000100100011010001"
	for cut := 0; cut < len(mandatory); cut++ {
		r, err := runtime.NewReaderBits(channelRequestPack(mandatory), cut, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeChannelRequestDescription2ValuePartFrom(r); err == nil {
			t.Fatalf("accepted mandatory truncation %d", cut)
		}
	}
	for _, data := range [][]byte{{}, {0}, bytes.Repeat([]byte{255}, 1<<18)} {
		if _, err := DecodeChannelRequestDescription2ValuePart(data); err == nil {
			t.Fatalf("accepted malformed input of %d octets", len(data))
		}
	}
}

func FuzzChannelRequestDescription2(f *testing.F) {
	f.Add([]byte{0x2a, 0x84, 0x8d, 0x01}, uint16(28))
	f.Fuzz(func(t *testing.T, data []byte, extent uint16) {
		if len(data) > 512 {
			return
		}
		bits := int(extent)
		if bits > len(data)*8 {
			return
		}
		r, err := runtime.NewReaderBits(data, bits, nil)
		if err != nil {
			return
		}
		v, err := DecodeChannelRequestDescription2ValuePartFrom(r)
		if err != nil {
			return
		}
		layout, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: bits}, bits, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(layout)
		if _, err := w.PushLimit(bits); err != nil {
			t.Fatal(err)
		}
		if err := EncodeChannelRequestDescription2ValuePartTo(w, v); err != nil {
			t.Fatal(err)
		}
		encoded, err := w.FinishBits(runtime.BitString{})
		if err != nil || encoded.BitLength != bits {
			t.Fatalf("replay extent %d/%d: %v", encoded.BitLength, bits, err)
		}
		for i := 0; i < bits; i++ {
			if (data[i/8]>>uint(7-i%8))&1 != (encoded.Bytes[i/8]>>uint(7-i%8))&1 {
				t.Fatalf("replay bit %d", i)
			}
		}
	})
}

func TestChannelRequestDescription2FieldBounds(t *testing.T) {
	value := ChannelRequestDescription2ValuePart{}
	value.RLCNonPersistentModeCapabilityChoice.Alternative = ChannelRequestDescription2ValuePartRLCNonPersistentModeCapabilityChoiceAlternativeAltL
	value.RLCNonPersistentModeCapabilityChoice.AltL = &struct{}{}
	cases := []struct {
		name string
		edit func(*ChannelRequestDescription2ValuePart)
	}{
		{"cause", func(v *ChannelRequestDescription2ValuePart) { v.PACKETESTABLISHMENTCAUSE = 4 }},
		{"multiple TBF", func(v *ChannelRequestDescription2ValuePart) { v.MultipleTBFCapability = 2 }},
		{"PFI", func(v *ChannelRequestDescription2ValuePart) { n := uint8(128); v.PFI = &n }},
		{"priority", func(v *ChannelRequestDescription2ValuePart) { v.ChannelRequestDescription.RADIOPRIORITY = 4 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := value
			tc.edit(&v)
			if _, err := EncodeChannelRequestDescription2ValuePartCanonicalAtLength(v, 8); err == nil {
				t.Fatal("accepted out-of-width scalar")
			}
		})
	}
	// Terminal padding can grow only with its explicit input extent. Fixed
	// metadata and an unchanged typed body do not become a growing value list.
	var previous float64
	for _, n := range []int{8, 16, 32, 64, 128} {
		wire, err := EncodeChannelRequestDescription2ValuePartCanonicalAtLength(value, n)
		if err != nil {
			t.Fatal(err)
		}
		d, err := DecodeChannelRequestDescription2ValuePart(wire)
		if err != nil || d.BitsConsumed != n*8 || !runtime.SemanticallyEqual(d.Value, value) {
			t.Fatalf("padding extent %d: %v", n, err)
		}
		count := testing.AllocsPerRun(3, func() {
			if _, err := DecodeChannelRequestDescription2ValuePart(wire); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && previous > 0 && count > 2*previous {
			t.Fatalf("padding allocation growth %g => %g", previous, count)
		}
		previous = count
	}
}
