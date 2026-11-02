package ackmessages

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

func ackMessagePack(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

// TS 44.060 V19.0.0 §11.2.6a p14456 specifies mobile-to-network
// direction. The MAC header is §10.3.2.1; selector 001000 is §11.2.0.2
// p13433. Table 11.2.6a.1a pp14483–14515 supplies both DLDC rows.
func ackMessageFrame(index, dual int, extension string, option int) string {
	bits := "01000000" + "001000" + "10101" + "1"
	quality := "00" + "101010" + "00" // §12.5.1: absent BEP and timeslot measurements, C_VALUE=42.
	if option&1 != 0 {
		bits += "1" + quality
	} else {
		bits += "0"
	}
	if option&2 != 0 {
		bits += "1" + "101010100001001000110100"
	} else {
		bits += "0"
	}
	if option&4 != 0 {
		bits += "1" + "1010101"
	} else {
		bits += "0"
	}
	if index >= 0 {
		width := index + 1
		if dual == 1 && index < 6 {
			width = index + 65
		}
		if extension == "" {
			extension = strings.Repeat("0", width)
		}
		bits += "1" + fmt.Sprintf("%06b", index) + extension
	} else {
		bits += "0"
	}
	// §12.3.1: length-bounded 15-bit body, FAI/BOW/EOW=101,
	// SSN=0x555, absent compressed bitmap and empty URBB.
	bits += "1" + "00001111" + "101" + "10101010101" + "0"
	bits += "0" // §11 padding's known arm precedes positioned L bits.
	for len(bits) < 184 {
		bits += string("00101011"[len(bits)%8])
	}
	return bits
}

func readAckMessage(t *testing.T, bits string, context int) (EGPRSPacketDownlinkAckNackMessageContent, *runtime.Reader, error) {
	t.Helper()
	r, err := runtime.NewReaderBits(ackMessagePack(bits), len(bits), nil)
	if err != nil {
		t.Fatal(err)
	}
	if context >= 0 {
		r.Set("DLDC_configuration", uint64(context))
	}
	h, err := controlblock.DecodeUplinkHeader(r)
	if err != nil {
		return EGPRSPacketDownlinkAckNackMessageContent{}, r, err
	}
	if h.Spare != 0 || h.R != 0 {
		t.Fatal("uplink MAC fields")
	}
	selector, err := r.ReadUint(6)
	if err != nil {
		return EGPRSPacketDownlinkAckNackMessageContent{}, r, err
	}
	if selector != 8 {
		t.Fatalf("selector %d", selector)
	}
	v, err := DecodeEGPRSPacketDownlinkAckNackMessageContentFrom(r)
	if err == nil {
		v.Wire, err = runtime.SealBits(v, runtime.BitString{Bytes: ackMessagePack(bits), BitLength: len(bits)}, r.Position(), runtime.BitString{}, r.Wire())
	}
	return v, r, err
}

func writeAckMessage(v EGPRSPacketDownlinkAckNackMessageContent, context int, canonical bool) (runtime.BitString, error) {
	if canonical {
		v = runtime.Canonical(v)
	}
	w := runtime.NewWriter()
	if canonical {
		w.WithWire(runtime.CanonicalTo(v, 184).Wire)
	} else {
		w.WithWire(v.Wire)
		v.Wire = runtime.WireInfo{}
	}
	if _, err := w.PushLimit(184); err != nil {
		return runtime.BitString{}, err
	}
	if context >= 0 {
		w.Set("DLDC_configuration", uint64(context))
	}
	if err := controlblock.EncodeUplinkHeader(w, controlblock.UplinkHeader{}); err != nil {
		return runtime.BitString{}, err
	}
	if err := w.WriteUint(8, 6); err != nil {
		return runtime.BitString{}, err
	}
	if err := EncodeEGPRSPacketDownlinkAckNackMessageContentTo(w, v); err != nil {
		return runtime.BitString{}, err
	}
	return w.FinishBits(runtime.BitString{})
}

func TestEGPRSAckMessageExtensionTable(t *testing.T) {
	for dual := 0; dual < 2; dual++ {
		for index := 0; index < 64; index++ {
			t.Run(fmt.Sprintf("dual%d/index%d", dual, index), func(t *testing.T) {
				bits := ackMessageFrame(index, dual, "", 0)
				if len(bits) != 184 {
					t.Fatal("frame size")
				}
				v, r, err := readAckMessage(t, bits, dual)
				if err != nil {
					t.Fatal(err)
				}
				if r.Position() != 184 || v.DOWNLINKTFI != 21 || v.MSOUTOFMEMORY != 1 {
					t.Fatalf("root %+v position %d", v, r.Position())
				}
				if v.EPDANExtensionLengthIndexGroup == nil || v.EPDANExtensionLengthIndexGroup.EPDANExtensionLengthIndex != uint8(index) {
					t.Fatal("extension index lost")
				}
				ack := v.EGPRSAckNackDescription.EGPRSAckNackDescriptionStructChoice.LengthL
				if ack == nil || ack.LengthL != 15 || ack.Content.FINALACKINDICATION != 1 || ack.Content.BEGINNINGOFWINDOW != 0 || ack.Content.ENDOFWINDOW != 1 || ack.Content.STARTINGSEQUENCENUMBER != 1365 || len(ack.Content.UNCOMPRESSEDRECEIVEDBLOCKBITMAP) != 0 {
					t.Fatalf("following Ack/Nack shifted: %+v", ack)
				}
				encoded, err := writeAckMessage(v, dual, false)
				if err != nil || encoded.BitLength != 184 || !bytes.Equal(encoded.Bytes, ackMessagePack(bits)) {
					t.Fatalf("replay %x/%d: %v", encoded.Bytes, encoded.BitLength, err)
				}
				canonical, err := writeAckMessage(v, dual, true)
				if err != nil || !bytes.Equal(canonical.Bytes, ackMessagePack(bits)) {
					t.Fatalf("contextual canonical %x: %v", canonical.Bytes, err)
				}
				if _, _, err := readAckMessage(t, bits, -1); !errors.Is(err, runtime.ErrContextRequired) {
					t.Fatalf("missing decode context: %v", err)
				}
				if _, err := writeAckMessage(v, -1, false); !errors.Is(err, runtime.ErrContextRequired) {
					t.Fatalf("missing encode context: %v", err)
				}
				if _, err := EncodeEGPRSPacketDownlinkAckNackMessageContentCanonicalAtLength(v, 23); !errors.Is(err, runtime.ErrContextRequired) {
					t.Fatalf("contextless canonical: %v", err)
				}
				v.MSOUTOFMEMORY = 0
				edited, err := writeAckMessage(v, dual, false)
				want := bits[:19] + "0" + bits[20:]
				if err != nil || !bytes.Equal(edited.Bytes, ackMessagePack(want)) {
					t.Fatalf("edit %x: %v", edited.Bytes, err)
				}
			})
		}
	}
}

func TestEGPRSAckMessageOptionalArms(t *testing.T) {
	for option := 0; option < 8; option++ {
		bits := ackMessageFrame(-1, 0, "", option)
		v, _, err := readAckMessage(t, bits, -1)
		if err != nil {
			t.Fatal(err)
		}
		if (v.EGPRSChannelQualityReport != nil) != (option&1 != 0) || (v.ChannelRequestDescription != nil) != (option&2 != 0) || (v.PFI != nil) != (option&4 != 0) || v.EPDANExtensionLengthIndexGroup != nil {
			t.Fatalf("optional fields %d", option)
		}
		if v.EGPRSChannelQualityReport != nil && v.EGPRSChannelQualityReport.CVALUE != 42 {
			t.Fatal("channel quality")
		}
		if v.ChannelRequestDescription != nil && v.ChannelRequestDescription.RLCOCTETCOUNT != 0x1234 {
			t.Fatal("channel request")
		}
		if v.PFI != nil && *v.PFI != 85 {
			t.Fatal("PFI")
		}
		for _, canonical := range []bool{false, true} {
			encoded, err := writeAckMessage(v, -1, canonical)
			if err != nil || !bytes.Equal(encoded.Bytes, ackMessagePack(bits)) {
				t.Fatalf("optional %d canonical=%v: %x %v", option, canonical, encoded.Bytes, err)
			}
		}
	}
}

func TestEGPRSAckMessageBounds(t *testing.T) {
	bits := ackMessageFrame(5, 1, "", 0)
	// Cuts before completion of the length-bounded Ack/Nack cannot borrow
	// following bits from the backing buffer; padding may end early.
	required := 14 + 16 + 70 + 24
	for cut := 14; cut < required; cut++ {
		if _, r, err := readAckMessage(t, bits[:cut], 1); err == nil {
			t.Fatalf("accepted truncation %d position %d", cut, r.Position())
		}
	}
	r := runtime.NewReader(append(ackMessagePack(bits), 0xa5))
	old, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	r.Set("DLDC_configuration", 1)
	if _, err := controlblock.DecodeUplinkHeader(r); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(6); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeEGPRSPacketDownlinkAckNackMessageContentFrom(r); err != nil {
		t.Fatal(err)
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(8)
	if err != nil || sentinel != 0xa5 {
		t.Fatalf("sentinel %x: %v", sentinel, err)
	}
}

func FuzzEGPRSAckMessage(f *testing.F) {
	f.Add(ackMessagePack(ackMessageFrame(0, 0, "", 0)), uint8(0))
	f.Add(ackMessagePack(ackMessageFrame(5, 1, "", 0)), uint8(1))
	f.Fuzz(func(t *testing.T, data []byte, dual uint8) {
		if len(data) != 23 {
			return
		}
		r, err := runtime.NewReaderBits(data, 184, nil)
		if err != nil {
			return
		}
		r.Set("DLDC_configuration", uint64(dual&1))
		if _, err := r.ReadUint(14); err != nil {
			return
		}
		v, err := DecodeEGPRSPacketDownlinkAckNackMessageContentFrom(r)
		if err != nil {
			return
		}
		layout, err := runtime.SealBits(v, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(layout)
		if _, err := w.PushLimit(184); err != nil {
			t.Fatal(err)
		}
		w.Set("DLDC_configuration", uint64(dual&1))
		prefix := uint64(data[0])<<6 | uint64(data[1]>>2)
		if err := w.WriteUint(prefix, 14); err != nil {
			t.Fatal(err)
		}
		if err := EncodeEGPRSPacketDownlinkAckNackMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		encoded, err := w.FinishBits(runtime.BitString{})
		if err != nil || encoded.BitLength != 184 || !bytes.Equal(encoded.Bytes, data) {
			t.Fatalf("replay %x want %x: %v", encoded.Bytes, data, err)
		}
	})
}

func TestEPDANExtensionInfoArms(t *testing.T) {
	// §11.2.6a pp14469–14480: three Rel-5 optionals, the Rel-6
	// request, and the Rel-7 early-establishment and secondary report.
	for mask := 0; mask < 64; mask++ {
		bits := ""
		if mask&1 != 0 {
			bits += "1" + "10101" + "10" + "0" + "0"
		} else {
			bits += "0"
		}
		if mask&2 != 0 {
			bits += "1" + "10101"
		} else {
			bits += "0"
		}
		if mask&4 != 0 {
			bits += "1" + "101"
		} else {
			bits += "0"
		}
		if mask&8 != 0 {
			bits += "1" + "1010101" + "10" + "1" + "0" + "0"
		} else {
			bits += "0"
		}
		bits += fmt.Sprintf("%01b", (mask>>4)&1)
		if mask&32 != 0 {
			bits += "1" + "00" + "101010" + "00"
		} else {
			bits += "0"
		}
		for phase := 0; phase < 8; phase++ {
			prefix := strings.Repeat("1", phase)
			full := prefix + bits + "10101"
			r := runtime.NewReader(ackMessagePack(full))
			if _, err := r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(len(bits))
			if err != nil {
				t.Fatal(err)
			}
			v, err := DecodeEPDANExtensionInfoFrom(r)
			if err != nil {
				t.Fatalf("mask %d: %v", mask, err)
			}
			if (v.Group.IuModeChannelRequestDescription != nil) != (mask&1 != 0) || (v.Group.RBId != nil) != (mask&2 != 0) || (v.Group.TimeslotNumber != nil) != (mask&4 != 0) || (v.ExtendedChannelRequestDescription != nil) != (mask&8 != 0) || v.Group2.EARLYTBFESTABLISHMENT != uint8((mask>>4)&1) || (v.Group2.SecondaryDualCarrierChannelReport != nil) != (mask&32 != 0) {
				t.Fatalf("lost extension field mask %d", mask)
			}
			if v.Group.IuModeChannelRequestDescription != nil && v.Group.IuModeChannelRequestDescription.Entries[0].RBId != 21 {
				t.Fatal("Iu mode RB Id")
			}
			if v.Group.RBId != nil && *v.Group.RBId != 21 {
				t.Fatal("RB Id")
			}
			if v.Group.TimeslotNumber != nil && *v.Group.TimeslotNumber != 5 {
				t.Fatal("timeslot")
			}
			if v.ExtendedChannelRequestDescription != nil && v.ExtendedChannelRequestDescription.Entries[0].PFI != 85 {
				t.Fatal("extended request PFI")
			}
			if v.Group2.SecondaryDualCarrierChannelReport != nil && v.Group2.SecondaryDualCarrierChannelReport.CVALUE != 42 {
				t.Fatal("secondary channel quality")
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(5)
			if err != nil || sentinel != 21 {
				t.Fatalf("extension sentinel %d: %v", sentinel, err)
			}
			for _, canonical := range []bool{false, true} {
				value := v
				if canonical {
					value = runtime.Canonical(value)
				}
				w := runtime.NewWriter()
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				if _, err := w.PushLimit(len(bits)); err != nil {
					t.Fatal(err)
				}
				if err := EncodeEPDANExtensionInfoTo(w, value); err != nil {
					t.Fatal(err)
				}
				got, err := w.FinishBits(runtime.BitString{})
				if err != nil || got.BitLength != phase+len(bits) || !bytes.Equal(got.Bytes, ackMessagePack(prefix+bits)) {
					t.Fatalf("mask %d phase %d canonical %v: %x %v", mask, phase, canonical, got.Bytes, err)
				}
			}
		}
	}
}

func TestEGPRSAckMessageExtensionTypes(t *testing.T) {
	// §11.2.6a pp14469–14480: complete release groups end after
	// 3, 4 and 6 bits. Cuts inside a group select the printed ignored arm.
	for dual := 0; dual < 2; dual++ {
		for index := 0; index < 64; index++ {
			width := index + 1
			if dual == 1 && index < 6 {
				width = index + 65
			}
			bits := ackMessageFrame(index, dual, "", 0)
			v, _, err := readAckMessage(t, bits, dual)
			if err != nil {
				t.Fatal(err)
			}
			content := v.EPDANExtensionLengthIndexGroup.Content
			known := width == 3 || width == 4 || width >= 6
			if known {
				if content.Known == nil || content.Ignored != nil || content.Alternative != EGPRSPacketDownlinkAckNackMessageContentEPDANExtensionLengthIndexGroupEPDANExtensionInfoEPDANExtensionInfoFallbackAlternativeKnown {
					t.Fatalf("dual%d/index%d: valid extension degraded to raw bits", dual, index)
				}
				value := content.Known
				if value.Group.IuModeChannelRequestDescription != nil || value.Group.RBId != nil || value.Group.TimeslotNumber != nil || value.ExtendedChannelRequestDescription != nil || value.Group2.EARLYTBFESTABLISHMENT != 0 || value.Group2.SecondaryDualCarrierChannelReport != nil {
					t.Fatalf("dual%d/index%d: zero extension acquired fields", dual, index)
				}
			} else {
				if content.Ignored == nil || content.Known != nil || content.Alternative != EGPRSPacketDownlinkAckNackMessageContentEPDANExtensionLengthIndexGroupEPDANExtensionInfoEPDANExtensionInfoFallbackAlternativeIgnored || content.Ignored.BitLength != width || !bytes.Equal(content.Ignored.Bytes, make([]byte, (width+7)/8)) {
					t.Fatalf("dual%d/index%d: ignored extension lost its exact extent", dual, index)
				}
			}
		}
	}
}
