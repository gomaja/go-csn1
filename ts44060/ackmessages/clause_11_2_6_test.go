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

func acknowledgementBits(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

// TS 44.060 V19.0.0 §11 padding starts with zero, then positional L bits.
func acknowledgementPadding(bits string, end int) string {
	if len(bits) < end {
		bits += "0"
	}
	for len(bits) < end {
		bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
	}
	return bits
}

func acknowledgementReader(t testing.TB, raw []byte, end, origin int) *runtime.Reader {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(raw, end, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(origin); err != nil {
		t.Fatal(err)
	}
	return r
}

func acknowledgementWrite(t testing.TB, bits string, origin int, wire runtime.WireInfo, encode func(*runtime.Writer) error) ([]byte, error) {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, err := w.PushLimit(len(bits)); err != nil {
		t.Fatal(err)
	}
	for _, b := range bits[:origin] {
		if err := w.WriteUint(uint64(b-'0'), 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := encode(w); err != nil {
		return nil, err
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err == nil && out.BitLength != len(bits) {
		t.Fatalf("extent %d want %d", out.BitLength, len(bits))
	}
	return out.Bytes, err
}

func checkAcknowledgement[T any](t *testing.T, bits string, origin, minimum, editBit int, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, inspect func(T), edit func(*T), sendable bool) {
	t.Helper()
	raw := acknowledgementBits(bits)
	r := acknowledgementReader(t, raw, len(bits), origin)
	value, err := decode(r)
	if err != nil || r.Position() != len(bits) {
		t.Fatalf("decode position %d want %d: %v", r.Position(), len(bits), err)
	}
	inspect(value)
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	replay, err := acknowledgementWrite(t, bits, origin, wire, func(w *runtime.Writer) error { return encode(w, value) })
	if err != nil || !bytes.Equal(replay, raw) {
		t.Fatalf("byte-exact replay differs: %v", err)
	}
	// A same-width semantic edit must retain received ignored and padding bits.
	edited := value
	edit(&edited)
	changedReplay, err := acknowledgementWrite(t, bits, origin, wire, func(w *runtime.Writer) error { return encode(w, edited) })
	expectedEdit := bytes.Clone(raw)
	expectedEdit[editBit/8] ^= 1 << uint(7-editBit%8)
	if err != nil || !bytes.Equal(changedReplay, expectedEdit) {
		t.Fatalf("received edit not transmitted: %v", err)
	}
	editedValue, err := decode(acknowledgementReader(t, changedReplay, len(bits), origin))
	if err != nil || !runtime.SemanticallyEqual(editedValue, edited) {
		t.Fatalf("received edit differs: %v", err)
	}
	if sendable {
		fresh := runtime.Canonical(value)
		cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(bits)).Wire
		canonical, err := acknowledgementWrite(t, bits, origin, cw, func(w *runtime.Writer) error { return encode(w, fresh) })
		if err != nil || !bytes.Equal(canonical, raw) {
			t.Fatalf("fresh canonical differs: %v", err)
		}
		edit(&fresh)
		changed, err := acknowledgementWrite(t, bits, origin, cw, func(w *runtime.Writer) error { return encode(w, fresh) })
		if err != nil || bytes.Equal(changed, raw) {
			t.Fatalf("typed edit not transmitted: %v", err)
		}
		again, err := decode(acknowledgementReader(t, changed, len(bits), origin))
		if err != nil || !runtime.SemanticallyEqual(again, fresh) {
			t.Fatalf("edited value differs: %v", err)
		}
	}
	// Every incomplete mandatory prefix must fail inside its exact bound.
	for end := origin; end < minimum; end++ {
		tr := acknowledgementReader(t, raw, end, origin)
		if _, err := decode(tr); err == nil {
			t.Fatalf("accepted truncated mandatory prefix %d", end)
		} else {
			var located *runtime.OperationError
			if !errors.As(err, &located) {
				t.Fatalf("unlocated truncation at %d: %v", end, err)
			}
		}
		if tr.Position() > end {
			t.Fatalf("crossed bit bound %d at %d", end, tr.Position())
		}
	}
	// The containing five-bit sentinel must remain available, including all phases.
	for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		frame := strings.Repeat("1", phase) + bits[origin:] + "10101"
		rd := acknowledgementReader(t, acknowledgementBits(frame), len(frame), phase)
		old, err := rd.PushLimit(len(bits) - origin)
		if err != nil {
			t.Fatal(err)
		}
		// Padding depends on the absolute coordinate. Replaying the body at a different
		// phase can enter the receiver's ignored-padding arm; it must still stop.
		_, err = decode(rd)
		if origin == 14 && phase%8 != 6 {
			var pe *ContentPhaseError
			if !errors.As(err, &pe) || pe.SourceOrigin() != 14 || pe.BitPosition() != phase || rd.Position() != phase {
				t.Fatalf("reader phase %d: %v", phase, err)
			}
			target := runtime.NewWriter()
			if e := target.WriteUint(0, phase); e != nil {
				t.Fatal(e)
			}
			writerErr := encode(target, value)
			if !errors.As(writerErr, &pe) || pe.SourceOrigin() != 14 || pe.BitPosition() != phase || target.Position() != phase {
				t.Fatalf("writer phase %d: %v", phase, writerErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("phase %d: %v", phase, err)
		}
		if rd.Position() != phase+len(bits)-origin {
			t.Fatalf("phase %d consumed sentinel", phase)
		}
		if err = rd.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := rd.ReadUint(5)
		if err != nil || sentinel != 21 {
			t.Fatalf("sentinel %d: %v", sentinel, err)
		}
	}
}

// TS 44.060 V19.0.0 §§10.3.2, 11.2.0.2 and 11.2.6, pp14399–14431.
// The five-bit TFI, 72-bit acknowledgement, optional request and quality report
// precede four distinct release gates. All vectors are complete uplink blocks.
func TestClause1126DownlinkAcknowledgement(t *testing.T) {
	for _, request := range []bool{false, true} {
		for mask := range 256 {
			bits := "01000000" + "000010" + fmt.Sprintf("%05b", 19) + "1" + fmt.Sprintf("%07b", 85) + strings.Repeat("10100101", 8)
			if request {
				bits += "1" + fmt.Sprintf("%04b%02b%01b%01b%016b", 9, 2, 1, 0, 321)
			} else {
				bits += "0"
			}
			bits += fmt.Sprintf("%06b%03b%06b", 37, 5, 42)
			for tn := range 8 {
				if mask&(1<<tn) == 0 {
					bits += "0"
				} else {
					bits += fmt.Sprintf("1%04b", tn+3)
				}
			}
			minimum := len(bits)
			// An explicit release-1999 zero accepts the rest as an ignored release tail.
			bits = acknowledgementPadding(bits+"0", 184)
			checkAcknowledgement(t, bits, 14, minimum, 18, DecodePacketDownlinkAckNackMessageContentFrom, EncodePacketDownlinkAckNackMessageContentTo, func(v PacketDownlinkAckNackMessageContent) {
				if v.DOWNLINKTFI != 19 || v.AckNackDescription.FINALACKINDICATION != 1 || v.AckNackDescription.STARTINGSEQUENCENUMBER != 85 || v.AckNackDescription.RECEIVEDBLOCKBITMAP != 0xa5a5a5a5a5a5a5a5 || (v.ChannelRequestDescription != nil) != request || v.ChannelQualityReport.CVALUE != 37 || v.ChannelQualityReport.RXQUAL != 5 || v.ChannelQualityReport.SIGNVAR != 42 || v.PFIChoice.Alternative != 1 {
					t.Fatal("base fields or release gate differ")
				}
				levels := []*uint8{v.ChannelQualityReport.ILEVELTN0, v.ChannelQualityReport.ILEVELTN1, v.ChannelQualityReport.ILEVELTN2, v.ChannelQualityReport.ILEVELTN3, v.ChannelQualityReport.ILEVELTN4, v.ChannelQualityReport.ILEVELTN5, v.ChannelQualityReport.ILEVELTN6, v.ChannelQualityReport.ILEVELTN7}
				for tn, p := range levels {
					if (p != nil) != (mask&(1<<tn) != 0) || p != nil && *p != uint8(tn+3) {
						t.Fatalf("timeslot %d differs", tn)
					}
				}
			}, func(v *PacketDownlinkAckNackMessageContent) { v.DOWNLINKTFI ^= 1 }, false)
		}
	}
	// The original MAC header is decoded through the production framing core.
	r := acknowledgementReader(t, acknowledgementBits("01000000"+strings.Repeat("0", 176)), 184, 0)
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err := controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil || r.Position() != 8 {
		t.Fatalf("uplink header: %v", err)
	}
}

// Each release gate has the printed null, ignored-tail and extension arms.
// TS 44.060 V19.0.0 §11.2.6 pp14404–14418. PFI belongs to Release 1999;
// the Iu request, RB and timeslot to Release 5; the extended request to Release 6.
func TestClause1126ReleaseGates(t *testing.T) {
	base := "01000000000010" + fmt.Sprintf("%05b", 19) + "1" + fmt.Sprintf("%07b", 85) + strings.Repeat("10100101", 8) + "0" + fmt.Sprintf("%06b%03b%06b", 37, 5, 42) + "00000000"
	for gate := range 4 {
		prefix := base
		if gate > 0 {
			prefix += "10"
		}
		if gate > 1 {
			prefix += "1000"
		}
		if gate > 2 {
			prefix += "10"
		}
		// End-selected null is exactly the containing bound, not a received zero.
		for _, tail := range []string{"", "0" + strings.Repeat("1", 184-len(prefix)-1)} {
			bits := prefix + tail
			r := acknowledgementReader(t, acknowledgementBits(bits), len(bits), 14)
			v, err := DecodePacketDownlinkAckNackMessageContentFrom(r)
			if err != nil || r.Position() != len(bits) {
				t.Fatalf("gate %d null=%v at %d: %v", gate, tail == "", r.Position(), err)
			}
			want := 0
			if tail != "" {
				want = 1
			}
			if gate == 0 {
				if int(v.PFIChoice.Alternative) != want {
					t.Fatalf("gate %d selected %d want %d", gate, v.PFIChoice.Alternative, want)
				}
			} else {
				p := v.PFIChoice.PFI
				if v.PFIChoice.Alternative != 2 || p == nil || p.PFI != nil {
					t.Fatal("lost Release 1999 prefix")
				}
				if gate == 1 {
					if int(p.IuModeChannelRequestDescriptionChoice.Alternative) != want {
						t.Fatal("wrong Release 5 end arm")
					}
				} else {
					i := p.IuModeChannelRequestDescriptionChoice.IuModeChannelRequestDescription
					if p.IuModeChannelRequestDescriptionChoice.Alternative != 2 || i == nil || i.IuModeChannelRequestDescription != nil || i.RBId != nil || i.TimeslotNumber != nil {
						t.Fatal("lost Release 5 prefix")
					}
					if gate == 2 {
						if int(i.ExtendedChannelRequestDescriptionChoice.Alternative) != want {
							t.Fatal("wrong Release 6 end arm")
						}
					} else {
						e := i.ExtendedChannelRequestDescriptionChoice.ExtendedChannelRequestDescription
						if i.ExtendedChannelRequestDescriptionChoice.Alternative != 2 || e == nil || e.ExtendedChannelRequestDescription != nil || int(i.EARLYTBFESTABLISHMENTChoice.Alternative) != want {
							t.Fatal("lost Release 6 prefix or wrong Release 7 end arm")
						}
					}
				}
			}
			fresh := runtime.Canonical(v)
			if tail == "" {
				cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(bits)).Wire
				canonical, e := acknowledgementWrite(t, bits, 14, cw, func(w *runtime.Writer) error { return EncodePacketDownlinkAckNackMessageContentTo(w, fresh) })
				if e != nil || !bytes.Equal(canonical, acknowledgementBits(bits)) {
					t.Fatalf("gate %d null canonical: %v", gate, e)
				}
			} else {
				// The unlabelled bit**=<no string> emits no remainder on send. Its
				// preceding zero survives; a requested longer byte extent must refuse.
				w := runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(prefix)+1).Wire)
				if _, e := w.PushLimit(len(prefix) + 1); e != nil {
					t.Fatal(e)
				}
				for _, b := range bits[:14] {
					if e := w.WriteUint(uint64(b-'0'), 1); e != nil {
						t.Fatal(e)
					}
				}
				if e := EncodePacketDownlinkAckNackMessageContentTo(w, fresh); e != nil {
					t.Fatal(e)
				}
				out, e := w.FinishBits(runtime.BitString{})
				if e != nil || out.BitLength != len(prefix)+1 || !bytes.Equal(out.Bytes, acknowledgementBits(bits[:len(prefix)+1])) {
					t.Fatalf("gate %d no-string sender extent %d: %v", gate, out.BitLength, e)
				}
				emitted, e := EncodePacketDownlinkAckNackMessageContentCanonicalAtLength(v, 23)
				if emitted != nil || e == nil {
					t.Fatalf("gate %d long canonical extent was accepted", gate)
				}
				if gate == 2 {
					// The following Release 7 null cannot fill a longer target.
					var bound *runtime.BoundError
					if !errors.As(e, &bound) || bound.Kind != runtime.CanonicalTarget {
						t.Fatalf("gate %d canonical target: %v", gate, e)
					}
				} else {
					var extent *runtime.ExtentError
					if !errors.As(e, &extent) {
						t.Fatalf("gate %d long canonical extent: %v", gate, e)
					}
				}
			}
			wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: acknowledgementBits(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			out, err := acknowledgementWrite(t, bits, 14, wire, func(w *runtime.Writer) error { return EncodePacketDownlinkAckNackMessageContentTo(w, v) })
			if err != nil || !bytes.Equal(out, acknowledgementBits(bits)) {
				t.Fatalf("gate %d exact replay: %v", gate, err)
			}
			v.DOWNLINKTFI ^= 1
			changed, err := acknowledgementWrite(t, bits, 14, wire, func(w *runtime.Writer) error { return EncodePacketDownlinkAckNackMessageContentTo(w, v) })
			expected := acknowledgementBits(bits)
			expected[2] ^= 1 << 5
			if err != nil || !bytes.Equal(changed, expected) {
				t.Fatalf("gate %d retained edit differs: %v", gate, err)
			}

		}
	}
	for mask := range 32 {
		for early := range 2 {
			bits := base + "1"
			if mask&1 != 0 {
				bits += fmt.Sprintf("1%07b", 53)
			} else {
				bits += "0"
			}
			bits += "1"
			if mask&2 != 0 {
				bits += "1" + fmt.Sprintf("%05b%02b", 17, 2) + "1" + fmt.Sprintf("%08b", 23) + "0"
			} else {
				bits += "0"
			}
			if mask&4 != 0 {
				bits += fmt.Sprintf("1%05b", 21)
			} else {
				bits += "0"
			}
			if mask&8 != 0 {
				bits += fmt.Sprintf("1%03b", 5)
			} else {
				bits += "0"
			}
			bits += "1"
			if mask&16 != 0 {
				bits += "1" + fmt.Sprintf("%07b%02b%01b", 37, 2, 1) + "11" + "0"
			} else {
				bits += "0"
			}
			bits += fmt.Sprintf("1%01b", early)
			minimum := len(base)
			bits = acknowledgementPadding(bits, 184)
			checkAcknowledgement(t, bits, 14, minimum, 18, DecodePacketDownlinkAckNackMessageContentFrom, EncodePacketDownlinkAckNackMessageContentTo, func(v PacketDownlinkAckNackMessageContent) {
				p := v.PFIChoice.PFI
				if v.PFIChoice.Alternative != 2 || p == nil || (p.PFI != nil) != (mask&1 != 0) || p.PFI != nil && *p.PFI != 53 {
					t.Fatal("Release 1999 differs")
				}
				i := p.IuModeChannelRequestDescriptionChoice.IuModeChannelRequestDescription
				if p.IuModeChannelRequestDescriptionChoice.Alternative != 2 || i == nil || (i.IuModeChannelRequestDescription != nil) != (mask&2 != 0) || (i.RBId != nil) != (mask&4 != 0) || (i.TimeslotNumber != nil) != (mask&8 != 0) {
					t.Fatal("Release 5 differs")
				}
				if i.RBId != nil && *i.RBId != 21 || i.TimeslotNumber != nil && *i.TimeslotNumber != 5 {
					t.Fatal("RB or timeslot differs")
				}
				if i.IuModeChannelRequestDescription != nil {
					es := i.IuModeChannelRequestDescription.Entries
					if len(es) != 1 || es[0].RBId != 17 || es[0].RADIOPRIORITY != 2 || es[0].RLCBLOCKCOUNT == nil || *es[0].RLCBLOCKCOUNT != 23 {
						t.Fatal("Iu request differs")
					}
				}
				e := i.ExtendedChannelRequestDescriptionChoice.ExtendedChannelRequestDescription
				if i.ExtendedChannelRequestDescriptionChoice.Alternative != 2 || e == nil || (e.ExtendedChannelRequestDescription != nil) != (mask&16 != 0) {
					t.Fatal("Release 6 differs")
				}
				if e.ExtendedChannelRequestDescription != nil {
					es := e.ExtendedChannelRequestDescription.Entries
					if len(es) != 1 || es[0].PFI != 37 || es[0].RADIOPRIORITY != 2 || es[0].RLCMODE != 1 || es[0].LLCPDUTYPE == nil || *es[0].LLCPDUTYPE != 1 {
						t.Fatal("extended request differs")
					}
				}
				last := i.EARLYTBFESTABLISHMENTChoice.EARLYTBFESTABLISHMENT
				if i.EARLYTBFESTABLISHMENTChoice.Alternative != 2 || last == nil || last.EARLYTBFESTABLISHMENT != uint8(early) {
					t.Fatal("Release 7 differs")
				}
			}, func(v *PacketDownlinkAckNackMessageContent) { v.DOWNLINKTFI ^= 1 }, true)
		}
	}
}

func FuzzClause1126DownlinkAcknowledgementReplay(f *testing.F) {
	f.Add(acknowledgementBits(acknowledgementPadding("01000000000010"+fmt.Sprintf("%05b", 19)+"1"+fmt.Sprintf("%07b", 85)+strings.Repeat("10100101", 8)+"0"+fmt.Sprintf("%06b%03b%06b", 37, 5, 42)+"00000000"+"0", 184)))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r := acknowledgementReader(t, raw, 184, 14)
		v, err := DecodePacketDownlinkAckNackMessageContentFrom(r)
		if err != nil {
			return
		}
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		prefix := fmt.Sprintf("%08b%06b", raw[0], raw[1]>>2) + strings.Repeat("0", 170)
		out, err := acknowledgementWrite(t, prefix, 14, wire, func(w *runtime.Writer) error { return EncodePacketDownlinkAckNackMessageContentTo(w, v) })
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("replay differs: %v", err)
		}
	})
}
