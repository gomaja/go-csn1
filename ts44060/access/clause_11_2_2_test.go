package access_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

func accessBits(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func accessReader(t testing.TB, bits string, c *runtime.Collector) *runtime.Reader {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(accessBits(bits), len(bits), c)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func accessEncode[T any](t testing.TB, v T, encode func(*runtime.Writer, T) error) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	if err := encode(w, v); err != nil {
		t.Fatal(err)
	}
	b, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func accessEqual(t testing.TB, b runtime.BitString, bits string) {
	t.Helper()
	if b.BitLength != len(bits) || !bytes.Equal(b.Bytes, accessBits(bits)) {
		t.Fatalf("encoded %x/%d, want %s", b.Bytes, b.BitLength, bits)
	}
}
func accessTruncations[T any](t testing.TB, bits string, decode func(*runtime.Reader) (T, error)) {
	t.Helper()
	for n := 0; n < len(bits); n++ {
		r := accessReader(t, bits[:n], nil)
		if _, err := decode(r); err == nil {
			t.Fatalf("accepted truncation at %d/%d", n, len(bits))
		}
	}
}
func accessSentinel[T any](t testing.TB, bits string, decode func(*runtime.Reader) (T, error)) {
	t.Helper()
	r := accessReader(t, bits+"10101101", nil)
	old, err := r.PushLimit(len(bits))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decode(r); err != nil {
		t.Fatal(err)
	}
	if r.Position() != len(bits) {
		t.Fatalf("extent %d want %d", r.Position(), len(bits))
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	v, err := r.ReadUint(8)
	if err != nil || v != 0xad {
		t.Fatalf("sentinel %x: %v", v, err)
	}
}
func accessRoundTrip[T any](t testing.TB, bits string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, canonical func(T) ([]byte, error)) T {
	t.Helper()
	r := accessReader(t, bits, nil)
	v, err := decode(r)
	if err != nil {
		t.Fatalf("decode %s: %v", bits, err)
	}
	if r.Position() != len(bits) {
		t.Fatalf("consumed %d, want %d", r.Position(), len(bits))
	}
	accessEqual(t, accessEncode(t, v, encode), bits)
	accessEqual(t, accessEncode(t, runtime.Canonical(v), encode), bits)
	out, err := canonical(v)
	if err != nil || !bytes.Equal(out, accessBits(bits)) {
		t.Fatalf("canonical %x: %v, want %s", out, err, bits)
	}
	return v
}

// TS 44.060 V19.0.0 §11.2.2 pp13514–13516 gives 8/11-bit access
// bursts. Table 11.2.2.1 pp13532–13542 puts CTRL_ACK
// after both message-type alternatives. Access bursts start at bit zero.
func TestClause1122AccessBursts(t *testing.T) {
	for ack := 0; ack < 4; ack++ {
		for tn := -1; tn < 8; tn++ {
			b11 := fmt.Sprintf("110111%03b%02b", max(tn, 0), ack)
			b8 := fmt.Sprintf("000%03b%02b", max(tn, 0), ack)
			if tn < 0 {
				b11 = fmt.Sprintf("111111001%02b", ack)
				b8 = fmt.Sprintf("011111%02b", ack)
			}
			v11 := accessRoundTrip(t, b11, access.DecodePacketControlAcknowledgement11BitMessageFrom, access.EncodePacketControlAcknowledgement11BitMessageTo, access.EncodePacketControlAcknowledgement11BitMessageCanonical)
			v8 := accessRoundTrip(t, b8, access.DecodePacketControlAcknowledgement8BitMessageFrom, access.EncodePacketControlAcknowledgement8BitMessageTo, access.EncodePacketControlAcknowledgement8BitMessageCanonical)
			if int(v11.CTRLACK) != ack || int(v8.CTRLACK) != ack {
				t.Fatal("CTRL_ACK lost")
			}
			if tn < 0 {
				if v11.MESSAGETYPEChoice.MESSAGETYPE == nil || *v11.MESSAGETYPEChoice.MESSAGETYPE != 505 || v8.MESSAGETYPEChoice.MESSAGETYPE == nil || *v8.MESSAGETYPEChoice.MESSAGETYPE != 31 {
					t.Fatal("fixed message type lost")
				}
			} else {
				if v11.MESSAGETYPEChoice.TNRRBP == nil || int(v11.MESSAGETYPEChoice.TNRRBP.TNRRBP) != tn || v8.MESSAGETYPEChoice.TNRRBP == nil || int(v8.MESSAGETYPEChoice.TNRRBP.TNRRBP) != tn {
					t.Fatal("TN_RRBP lost")
				}
			}
			v11.CTRLACK ^= 1
			accessEqual(t, accessEncode(t, v11, access.EncodePacketControlAcknowledgement11BitMessageTo), b11[:9]+fmt.Sprintf("%02b", ack^1))
			v8.CTRLACK ^= 1
			accessEqual(t, accessEncode(t, v8, access.EncodePacketControlAcknowledgement8BitMessageTo), b8[:6]+fmt.Sprintf("%02b", ack^1))
			accessTruncations(t, b11, access.DecodePacketControlAcknowledgement11BitMessageFrom)
			accessTruncations(t, b8, access.DecodePacketControlAcknowledgement8BitMessageFrom)
			accessSentinel(t, b11, access.DecodePacketControlAcknowledgement11BitMessageFrom)
			accessSentinel(t, b8, access.DecodePacketControlAcknowledgement8BitMessageFrom)
		}
	}
	for _, bits := range []string{"11111100000", "11011000000"} {
		if _, err := access.DecodePacketControlAcknowledgement11BitMessageFrom(accessReader(t, bits, nil)); err == nil {
			t.Fatal("invalid type accepted")
		}
	}
	for _, bits := range []string{"01111000", "00100000"} {
		if _, err := access.DecodePacketControlAcknowledgement8BitMessageFrom(accessReader(t, bits, nil)); err == nil {
			t.Fatal("invalid type accepted")
		}
	}
}

// bit 14 is the content origin: eight MAC bits, six external selector bits.
// TS 44.060 V19.0.0 §§10.3.2, 11.2.0.2, 11.2.2 pp13520–13530.
func TestClause1122NormalBlock(t *testing.T) {
	type suffixCase struct {
		bits    string
		padding bool
	}
	cases := []suffixCase{{bits: ""}, {bits: "0101011"}}
	for tn := -1; tn < 8; tn++ {
		for _, gr := range []int{-1, 0, 15} {
			prefix := "1"
			if tn < 0 {
				prefix += "0"
			} else {
				prefix += "1" + fmt.Sprintf("%03b", tn)
			}
			if gr < 0 {
				prefix += "0"
			} else {
				prefix += "1" + fmt.Sprintf("%04b", gr)
			}
			cases = append(cases, suffixCase{bits: prefix}, suffixCase{bits: prefix + "010101"}, suffixCase{bits: prefix + "10", padding: true})
			for _, ext := range []int{0, 341, 511} {
				cases = append(cases, suffixCase{bits: prefix + "11" + fmt.Sprintf("%09b", ext), padding: true})
			}
		}
	}
	for _, sc := range cases {
		suffix := sc.bits
		for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 16} {
			prefix := strings.Repeat("1", phase) + "01000000000001"
			body := fmt.Sprintf("%032b%02b", uint32(0x12345678), 3) + suffix
			bits := prefix + body
			// Nonempty explicit release-6 arm ends with the shared positional padding.
			if sc.padding {
				bits += "0"
				for len(bits)%8 != 0 {
					bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
				}
			}
			r := accessReader(t, bits, nil)
			if _, err := r.ReadUint(len(prefix)); err != nil {
				t.Fatal(err)
			}
			v, err := access.DecodePacketControlAcknowledgementMessageContentFrom(r)
			if len(prefix)%8 != 6 {
				var phaseError *access.ContentPhaseError
				if !errors.As(err, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != len(prefix) || r.Position() != len(prefix) {
					t.Fatalf("reader phase %d changed cursor or error: %v", len(prefix), err)
				}
				target := runtime.NewWriter()
				if e := target.WriteUint(0, len(prefix)); e != nil {
					t.Fatal(e)
				}
				writerErr := access.EncodePacketControlAcknowledgementMessageContentTo(target, v)
				if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != len(prefix) || target.Position() != len(prefix) {
					t.Fatalf("writer phase %d changed cursor or error: %v", len(prefix), writerErr)
				}
				continue
			}
			if err != nil || v.TLLIGRNTI != 0x12345678 || v.CTRLACK != 3 || r.Position() != len(bits) {
				t.Fatalf("suffix=%s phase=%d: %+v %v pos=%d", suffix, phase, v, err, r.Position())
			}
			if strings.HasPrefix(suffix, "1") {
				optional := v.TNRRBPChoice.TNRRBP
				if optional == nil {
					t.Fatal("release-5 fields missing")
				}
				offset := 1
				if suffix[offset] == '1' {
					offset++
					want := uint8(0)
					for _, b := range suffix[offset : offset+3] {
						want = want*2 + uint8(b-'0')
					}
					offset += 3
					if optional.TNRRBP == nil || *optional.TNRRBP != want {
						t.Fatal("TN_RRBP lost")
					}
				} else {
					offset++
					if optional.TNRRBP != nil {
						t.Fatal("unexpected TN_RRBP")
					}
				}
				if suffix[offset] == '1' {
					offset++
					want := uint8(0)
					for _, b := range suffix[offset : offset+4] {
						want = want*2 + uint8(b-'0')
					}
					offset += 4
					if optional.GRNTIExtension == nil || *optional.GRNTIExtension != want {
						t.Fatal("G-RNTI extension lost")
					}
				} else {
					offset++
					if optional.GRNTIExtension != nil {
						t.Fatal("unexpected G-RNTI extension")
					}
				}
				if len(suffix) > offset && suffix[offset] == '1' {
					offset++
					ext := optional.CTRLACKEXTENSIONChoice.CTRLACKEXTENSION
					if ext == nil {
						t.Fatal("release-6 choice missing")
					}
					if suffix[offset] == '1' {
						offset++
						want := uint16(0)
						for _, b := range suffix[offset : offset+9] {
							want = want*2 + uint16(b-'0')
						}
						if ext.CTRLACKEXTENSION == nil || *ext.CTRLACKEXTENSION != want {
							t.Fatal("CTRL_ACK_EXTENSION lost")
						}
					} else if ext.CTRLACKEXTENSION != nil {
						t.Fatal("unexpected CTRL_ACK_EXTENSION")
					}
				}
			}
			wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: accessBits(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			w := runtime.NewWriter()
			w.WithWire(wire)
			if err = w.WriteBitString(runtime.BitString{Bytes: accessBits(prefix), BitLength: len(prefix)}, len(prefix)); err != nil {
				t.Fatal(err)
			}
			if err = access.EncodePacketControlAcknowledgementMessageContentTo(w, v); err != nil {
				t.Fatal(err)
			}
			got, err := w.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			accessEqual(t, got, bits)
			v.CTRLACK = 2
			w = runtime.NewWriter()
			if err = w.WriteBitString(runtime.BitString{Bytes: accessBits(prefix), BitLength: len(prefix)}, len(prefix)); err != nil {
				t.Fatal(err)
			}
			if err = access.EncodePacketControlAcknowledgementMessageContentTo(w, runtime.Canonical(v)); err != nil {
				t.Fatal(err)
			}
			got, err = w.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			rr := accessReader(t, stringBits(got), nil)
			if _, err = rr.ReadUint(len(prefix)); err != nil {
				t.Fatal(err)
			}
			edited, err := access.DecodePacketControlAcknowledgementMessageContentFrom(rr)
			if err != nil || edited.CTRLACK != 2 {
				t.Fatalf("edit: %+v %v", edited, err)
			}
		}
	}
}
func stringBits(b runtime.BitString) string {
	var s strings.Builder
	for i := 0; i < b.BitLength; i++ {
		s.WriteByte('0' + (b.Bytes[i/8]>>uint(7-i%8))&1)
	}
	return s.String()
}

type accessArm struct {
	prefix, name string
	fields       []string
	widths       []int
}

func checkAccessChoices[T any](t *testing.T, n int, arms []accessArm, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, canonical func(T) ([]byte, error)) {
	t.Helper()
	counts := make([]int, len(arms))
	for value := 0; value < 1<<uint(n); value++ {
		bits := fmt.Sprintf("%0*b", n, value)
		which := -1
		for i, a := range arms {
			if strings.HasPrefix(bits, a.prefix) {
				which = i
				break
			}
		}
		r := accessReader(t, bits, nil)
		v, err := decode(r)
		if which < 0 {
			if err == nil {
				t.Fatalf("reserved selector accepted %s", bits)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", bits, err)
		}
		if r.Position() != n {
			t.Fatalf("extent %d/%d", r.Position(), n)
		}
		counts[which]++
		a := arms[which]
		choice := reflect.ValueOf(v).FieldByName("OnePhaseAccessRequestChoice")
		if int(choice.FieldByName("Alternative").Uint()) != which {
			t.Fatalf("wrong arm for %s", bits)
		}
		selected := choice.FieldByName(a.name)
		if selected.IsNil() {
			t.Fatalf("missing typed %s", a.name)
		}
		offset := len(a.prefix)
		for i, name := range a.fields {
			want := uint64(0)
			for _, b := range bits[offset : offset+a.widths[i]] {
				want = want*2 + uint64(b-'0')
			}
			offset += a.widths[i]
			if got := selected.Elem().FieldByName(name).Uint(); got != want {
				t.Fatalf("%s=%d want %d", name, got, want)
			}
		}
		accessEqual(t, accessEncode(t, v, encode), bits)
		out, e := canonical(v)
		if e != nil || !bytes.Equal(out, accessBits(bits)) {
			t.Fatalf("canonical %s: %x %v", bits, out, e)
		}
		field := selected.Elem().FieldByName("RandomBits")
		if !field.CanSet() {
			t.Fatal("random bits unavailable")
		}
		field.SetUint(field.Uint() ^ 1)
		accessEqual(t, accessEncode(t, v, encode), fmt.Sprintf("%0*b", n, value^1))
		if counts[which] == 1 {
			accessTruncations(t, bits, decode)
			accessSentinel(t, bits, decode)
		}
	}
	for i, n := range counts {
		if n == 0 {
			t.Fatalf("uncovered arm %s", arms[i].name)
		}
	}
	t.Logf("exhaustive %d-bit inputs=%d accepted=%v", n, 1<<uint(n), counts)
}

func fuzzAccessCodec[T any](t *testing.T, data []byte, extent int, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error) {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(data, extent, nil)
	if err != nil {
		t.Fatal(err)
	}
	v, err := decode(r)
	if err != nil {
		return
	}
	bits := stringBits(runtime.BitString{Bytes: data, BitLength: r.Position()})
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: accessBits(bits), BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err = encode(w, v); err != nil {
		t.Fatalf("accepted decode cannot replay %s: %v", bits, err)
	}
	got, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	accessEqual(t, got, bits)
}
func FuzzAccessForms(f *testing.F) {
	for kind := byte(0); kind < 12; kind++ {
		f.Add(kind, []byte{0xff, 0xff, 0xff, 0xff, 0xfc, 0}, uint16(48))
		f.Add(kind, []byte{0, 0}, uint16(11))
	}
	f.Fuzz(func(t *testing.T, kind byte, data []byte, n uint16) {
		if len(data) > 64 {
			return
		}
		extent := 0
		if len(data) > 0 {
			extent = int(n) % (len(data)*8 + 1)
		}
		switch kind % 12 {
		case 0:
			fuzzAccessCodec(t, data, extent, access.DecodePacketControlAcknowledgement11BitMessageFrom, access.EncodePacketControlAcknowledgement11BitMessageTo)
		case 1:
			fuzzAccessCodec(t, data, extent, access.DecodePacketControlAcknowledgement8BitMessageFrom, access.EncodePacketControlAcknowledgement8BitMessageTo)
		case 2:
			fuzzAccessCodec(t, data, extent, access.DecodePacketControlAcknowledgementMessageContentFrom, access.EncodePacketControlAcknowledgementMessageContentTo)
		case 3:
			fuzzAccessCodec(t, data, extent, access.DecodePacketChannelRequest11BitMessageContentFrom, access.EncodePacketChannelRequest11BitMessageContentTo)
		case 4:
			fuzzAccessCodec(t, data, extent, access.DecodePacketChannelRequest8BitMessageContentFrom, access.EncodePacketChannelRequest8BitMessageContentTo)
		case 5:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSPacketChannelRequestMessageContentFrom, access.EncodeEGPRSPacketChannelRequestMessageContentTo)
		case 6:
			fuzzAccessCodec(t, data, extent, access.DecodeMPRACHPacketChannelRequestMessageContentFrom, access.EncodeMPRACHPacketChannelRequestMessageContentTo)
		case 7:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSPACKETCHANNELREQUESTMessageContentForPEOOnePhaseAccessRequestFrom, access.EncodeEGPRSPACKETCHANNELREQUESTMessageContentForPEOOnePhaseAccessRequestTo)
		case 8:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventFrom, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentRLCDataBlockMethodOrPageResponseForPositioningEventTo)
		case 9:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentAccessBurstMethodFrom, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentAccessBurstMethodTo)
		case 10:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1From, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart1To)
		case 11:
			fuzzAccessCodec(t, data, extent, access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2From, access.EncodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2To)
		}
	})
}

// TS 44.060 V19.0.0 §§10.0b.2.1, 10.3.2 and 11.2.0.2:
// a CS-1 uplink control block has 184 bits; its content starts at bit 14.
func TestClause1122CompleteControlBlocks(t *testing.T) {
	for ack := 0; ack < 4; ack++ {
		for _, tn := range []int{-1, 0, 7} {
			for _, gr := range []int{-1, 0, 15} {
				for _, ext := range []int{-1, 0, 341, 511} {
					suffix := "1"
					if tn < 0 {
						suffix += "0"
					} else {
						suffix += "1" + fmt.Sprintf("%03b", tn)
					}
					if gr < 0 {
						suffix += "0"
					} else {
						suffix += "1" + fmt.Sprintf("%04b", gr)
					}
					suffix += "1"
					if ext < 0 {
						suffix += "0"
					} else {
						suffix += "1" + fmt.Sprintf("%09b", ext)
					}
					bits := "01000000000001" + fmt.Sprintf("%032b%02b", uint32(0x12345678), ack) + suffix + "0"
					for len(bits) < 184 {
						bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
					}
					r := accessReader(t, bits, nil)
					ctx := runtime.RLCMACContext{Direction: runtime.RLCMACUplink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
					if _, err := controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
						t.Fatal(err)
					}
					tag, err := r.ReadUint(6)
					if err != nil || tag != 1 || r.Position() != 14 {
						t.Fatal("content origin differs")
					}
					v, err := access.DecodePacketControlAcknowledgementMessageContentFrom(r)
					if err != nil || r.Position() != 184 || int(v.CTRLACK) != ack || v.TLLIGRNTI != 0x12345678 {
						t.Fatalf("full block: %+v %v", v, err)
					}
					wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: accessBits(bits), BitLength: 184}, 184, runtime.BitString{}, r.Wire())
					if err != nil {
						t.Fatal(err)
					}
					for _, canonical := range []bool{false, true} {
						w := runtime.NewWriter()
						cv := v
						if canonical {
							wire = runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
							cv = runtime.Canonical(v)
						}
						w.WithWire(wire)
						if _, err = w.PushLimit(184); err != nil {
							t.Fatal(err)
						}
						if err = w.WriteUint(0x1001, 14); err != nil {
							t.Fatal(err)
						}
						if err = access.EncodePacketControlAcknowledgementMessageContentTo(w, cv); err != nil {
							t.Fatal(err)
						}
						out, err := w.FinishBits(runtime.BitString{})
						if err != nil {
							t.Fatal(err)
						}
						accessEqual(t, out, bits)
					}
				}
			}
		}
	}
}

// Table 11.2.2.1 p13523 consumes an ignored suffix to the caller's bound.
// These component extents exercise storage growth, not the 184-bit MAC carrier.
func TestClause1122IgnoredSuffixGrowth(t *testing.T) {
	for _, octets := range []int{32, 128, 512, 2048} {
		bits := strings.Repeat("0", 34) + "0" + strings.Repeat("1", octets*8-35)
		raw := accessBits(bits)
		v, err := access.DecodePacketControlAcknowledgementMessageContent(raw)
		if err != nil || v.BitsConsumed != len(bits) {
			t.Fatalf("%d octets: %v", octets, err)
		}
		replay, err := access.EncodePacketControlAcknowledgementMessageContent(v.Value)
		if err != nil || !bytes.Equal(replay, raw) {
			t.Fatalf("%d octet replay: %v", octets, err)
		}
		allocations := testing.AllocsPerRun(3, func() {
			if _, e := access.DecodePacketControlAcknowledgementMessageContent(raw); e != nil {
				t.Fatal(e)
			}
		})
		if !raceEnabled && allocations > 128+float64(octets)/8 {
			t.Fatalf("%d octets: %.0f allocations exceed linear bound", octets, allocations)
		}
	}
}

// TS 44.060 V19.0.0 §11.2.2 pp13520–13530: the identity and
// acknowledgement are mandatory; a present extension has all nine bits.
func TestClause1122NormalBlockTruncations(t *testing.T) {
	prefix := "01000000000001"
	mandatory := fmt.Sprintf("%032b%02b", uint32(0x12345678), 3)
	var truncated []string
	for n := 0; n < 34; n++ {
		truncated = append(truncated, mandatory[:n])
	}
	for n := 0; n < 9; n++ {
		truncated = append(truncated, mandatory+"10011"+strings.Repeat("1", n))
	}
	for _, body := range truncated {
		r := accessReader(t, prefix+body, nil)
		if _, err := r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		if _, err := access.DecodePacketControlAcknowledgementMessageContentFrom(r); err == nil {
			t.Fatalf("accepted incomplete content %s", body)
		}
	}
	// An invalid padding selector belongs to the printed receiver error arm.
	bits := prefix + mandatory + "10010" + "1"
	bits += strings.Repeat("1", 184-len(bits))
	c := runtime.NewReceiverCollector()
	r := accessReader(t, bits, c)
	if _, err := r.ReadUint(14); err != nil {
		t.Fatal(err)
	}
	v, err := access.DecodePacketControlAcknowledgementMessageContentFrom(r)
	if err != nil || v.TLLIGRNTI != 0x12345678 || v.CTRLACK != 3 || r.Position() != 184 || len(c.TakeReceiverReport().Regions) == 0 {
		t.Fatalf("receiver error arm lost known fields: %+v %v", v, err)
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: accessBits(bits), BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err = w.WriteUint(0x1001, 14); err != nil {
		t.Fatal(err)
	}
	if err = access.EncodePacketControlAcknowledgementMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	accessEqual(t, out, bits)
}
