package messages

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §9.1.49a pp10573–10586 and §10.4
// table 10.4.2 pp13201–13206: the complete first octet includes
// RR short PD, message type 01000 and the two-bit short L2 header.
func vgcsIdentity(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(17 + 29*i)
	}
	return data
}

func vgcsWire(n, filler, octets int) []byte {
	data := append([]byte{0x20, byte(n<<3 | filler)}, vgcsIdentity(n)...)
	return append(data, bytes.Repeat([]byte{0x2b}, octets-len(data))...)
}

func vgcsValue(n, filler int) VGCSADDITIONALINFO {
	return VGCSADDITIONALINFO{MessageType: 8, TalkerIdentityLength: uint8(n), FillerBits: uint8(filler), TalkerIdentity: runtime.BitString{Bytes: vgcsIdentity(n), BitLength: 8 * n}}
}

func TestVGCSAdditionalInformationTypedVectors(t *testing.T) {
	for _, n := range []int{0, 1, 2, 16, 17} {
		for _, filler := range []int{0, 1, 7} {
			for _, octets := range []int{n + 2, 23} {
				t.Run(fmt.Sprintf("length%d/filler%d/extent%d", n, filler, octets), func(t *testing.T) {
					wire := vgcsWire(n, filler, octets)
					got, err := DecodeVGCSADDITIONALINFO(wire)
					if err != nil {
						t.Fatal(err)
					}
					v := got.Value
					if got.BitsConsumed != 8*octets || v.RRShortPD != 0 || v.MessageType != 8 || v.ShortLayer2Header != 0 || v.TalkerIdentityLength != uint8(n) || v.FillerBits != uint8(filler) || v.TalkerIdentity.BitLength != 8*n || !bytes.Equal(v.TalkerIdentity.Bytes, vgcsIdentity(n)) {
						t.Fatalf("typed fields or extent differ: %+v/%d", v, got.BitsConsumed)
					}
					replay, err := EncodeVGCSADDITIONALINFO(v)
					if err != nil || !bytes.Equal(replay, wire) {
						t.Fatalf("replay %x want %x: %v", replay, wire, err)
					}
					for _, value := range []VGCSADDITIONALINFO{v, vgcsValue(n, filler)} {
						encoded, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(value, octets)
						if err != nil || !bytes.Equal(encoded, wire) {
							t.Fatalf("canonical %x want %x: %v", encoded, wire, err)
						}
					}
					if _, err := EncodeVGCSADDITIONALINFOCanonical(v); err == nil {
						t.Fatal("missing containing extent accepted")
					}
					for _, invalid := range []int{-1, 0, n + 1, 24, 256} {
						if _, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(v, invalid); err == nil {
							t.Fatalf("accepted invalid extent %d", invalid)
						}
					}
					// Editing a value must reach the encoded identity instead of replaying stale bytes.
					if n > 0 {
						changed := runtime.Canonical(v)
						changed.TalkerIdentity.Bytes[0] ^= 0xff
						expected := append([]byte(nil), wire...)
						expected[2] ^= 0xff
						encoded, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(changed, octets)
						if err != nil || !bytes.Equal(encoded, expected) {
							t.Fatalf("identity edit %x: %v", encoded, err)
						}
					}
				})
			}
		}
	}
}

func TestVGCSAdditionalInformationBoundsAndGrowth(t *testing.T) {
	wire := vgcsWire(2, 3, 8)
	for cut := 0; cut < 32; cut++ {
		r, err := runtime.NewReaderBits(wire, cut, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeVGCSADDITIONALINFOFrom(r); err == nil {
			t.Fatalf("accepted mandatory truncation at bit %d", cut)
		}
		if r.Position() > cut {
			t.Fatal("decoder crossed containing limit")
		}
	}
	for msg := 0; msg < 32; msg++ {
		if msg == 8 {
			continue
		}
		wrong := append([]byte(nil), wire...)
		wrong[0] = byte(msg << 2)
		if _, err := DecodeVGCSADDITIONALINFO(wrong); err == nil {
			t.Fatalf("accepted message type %d", msg)
		}
	}
	for _, change := range []func(*VGCSADDITIONALINFO){
		func(v *VGCSADDITIONALINFO) { v.RRShortPD = 2 },
		func(v *VGCSADDITIONALINFO) { v.MessageType = 9 },
		func(v *VGCSADDITIONALINFO) { v.ShortLayer2Header = 4 },
		func(v *VGCSADDITIONALINFO) { v.TalkerIdentityLength = 32 },
		func(v *VGCSADDITIONALINFO) { v.FillerBits = 8 },
		func(v *VGCSADDITIONALINFO) { v.TalkerIdentity.BitLength = 15 },
		func(v *VGCSADDITIONALINFO) { v.TalkerIdentityLength = 3 },
	} {
		v := vgcsValue(2, 3)
		change(&v)
		if _, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(v, 8); err == nil {
			t.Fatal("invalid typed field accepted")
		}
	}
	got, err := DecodeVGCSADDITIONALINFO(wire)
	if err != nil {
		t.Fatal(err)
	}
	growth := got.Value
	growth.TalkerIdentityLength = 3
	growth.TalkerIdentity = runtime.BitString{Bytes: vgcsIdentity(3), BitLength: 24}
	if _, err := EncodeVGCSADDITIONALINFO(growth); err == nil {
		t.Fatal("plain encode silently changed the received semantic boundary")
	}
	encoded, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(growth, 8)
	if err != nil || !bytes.Equal(encoded, vgcsWire(3, 3, 8)) {
		t.Fatalf("canonical growth %x: %v", encoded, err)
	}
	_, err = EncodeVGCSADDITIONALINFOCanonicalAtLength(vgcsValue(2, 3), 24)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) || extent.Maximum != 23 {
		t.Fatalf("missing source carrier cap: %v", err)
	}
}

func vgcsPack(bits string) []byte {
	data := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			data[i/8] |= 1 << uint(7-i%8)
		}
	}
	return data
}

func TestVGCSAdditionalInformationBoundedPhases(t *testing.T) {
	// The sentinel is outside the root's bound; every absolute bit origin is tested.
	semantic := "00100000" + "00010011" + fmt.Sprintf("%08b%08b", 17, 46)
	for phase := 0; phase < 8; phase++ {
		for _, padding := range []int{0, 1, 7, 8, 13} {
			body := semantic
			for i := 0; i < padding; i++ {
				body += string("00101011"[(phase+len(semantic)+i)%8])
			}
			prefix := strings.Repeat("1", phase)
			all := prefix + body
			r := runtime.NewReader(vgcsPack(all + "10101"))
			if _, err := r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(len(body))
			if err != nil {
				t.Fatal(err)
			}
			v, err := DecodeVGCSADDITIONALINFOFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if v.TalkerIdentityLength != 2 || v.FillerBits != 3 || !bytes.Equal(v.TalkerIdentity.Bytes, []byte{17, 46}) || r.Position() != len(all) {
				t.Fatal("bounded typed decode differs")
			}
			layout, err := runtime.SealBits(v, runtime.BitString{Bytes: vgcsPack(all), BitLength: len(all)}, len(all), runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(5)
			if err != nil || sentinel != 21 {
				t.Fatal("decoder consumed following sentinel")
			}
			for _, canonical := range []bool{false, true} {
				value := v
				w := runtime.NewWriter()
				w.WithWire(layout)
				if canonical {
					value = runtime.Canonical(v)
					w.WithWire(runtime.CanonicalTo(value, len(all)).Wire)
				}
				if err := w.WriteLiteral(prefix); err != nil {
					t.Fatal(err)
				}
				bound, err := w.PushLimit(len(body))
				if err != nil {
					t.Fatal(err)
				}
				if err := EncodeVGCSADDITIONALINFOTo(w, value); err != nil {
					t.Fatal(err)
				}
				if err := w.PopLimit(bound); err != nil {
					t.Fatal(err)
				}
				encoded, err := w.FinishBits(runtime.BitString{})
				if err != nil || encoded.BitLength != len(all) || !bytes.Equal(encoded.Bytes, vgcsPack(all)) {
					t.Fatalf("phase%d canonical%v: %x/%d %v", phase, canonical, encoded.Bytes, encoded.BitLength, err)
				}
			}
		}
	}
}

func FuzzVGCSAdditionalInformation(f *testing.F) {
	for _, n := range []int{0, 1, 2, 16, 17} {
		f.Add(vgcsWire(n, 0, n+2))
	}
	f.Add([]byte{})
	f.Add([]byte{0xff})
	f.Add([]byte{0x20, 0xf8})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		got, err := DecodeVGCSADDITIONALINFO(data)
		if err != nil {
			return
		}
		encoded, err := EncodeVGCSADDITIONALINFO(got.Value)
		if err != nil || !bytes.Equal(encoded, data) {
			t.Fatalf("accepted input did not replay: %x %v", encoded, err)
		}
		if len(data) <= 23 {
			canonical, err := EncodeVGCSADDITIONALINFOCanonicalAtLength(got.Value, len(data))
			if err != nil {
				return
			}
			again, err := DecodeVGCSADDITIONALINFO(canonical)
			if err != nil || !runtime.SemanticallyEqual(got.Value, again.Value) {
				t.Fatalf("canonical typed fields differ: %v", err)
			}
		}
	})
}
