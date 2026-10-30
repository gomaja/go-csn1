package messages

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §9.1.58, DOCX paragraphs 11515–11523;
// message assignment: §10.4 table 10.4.2, paragraphs 13231–13236.
// Synthetic application and MSISDN values were checked with an independent codec.
var notifyApplicationVectors = []struct {
	length, octets, contentBits int
	hex                         string
}{
	{-1, 12, 89, "340123456789abcdef01a52b"},
	{-1, 32, 89, "340123456789abcdef01a52b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
	{0, 12, 93, "340123456789abcdef01a583"},
	{0, 32, 93, "340123456789abcdef01a5832b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
	{1, 13, 101, "340123456789abcdef01a58ffb"},
	{1, 32, 101, "340123456789abcdef01a58ffb2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b"},
	{7, 19, 149, "340123456789abcdef01a5bffffffffffffffb"},
	{7, 32, 149, "340123456789abcdef01a5bffffffffffffffb2b2b2b2b2b2b2b2b2b2b2b2b2b"},
	{15, 27, 213, "340123456789abcdef01a5fffffffffffffffffffffffffffffffb"},
	{15, 32, 213, "340123456789abcdef01a5fffffffffffffffffffffffffffffffb2b2b2b2b2b"},
}

func notifyApplicationValue(length int) NOTIFYAPPLICATIONDATA {
	v := NOTIFYAPPLICATIONDATA{MessageType: 13, ApplicationData: runtime.BitString{Bytes: []byte{1, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 1}, BitLength: 72}, DataIdentity: 0xa5}
	if length >= 0 {
		v.MSISDNLengthGroup = &NOTIFYAPPLICATIONDATAMSISDNLengthGroup{MSISDNLength: uint8(length), MSISDN: runtime.BitString{Bytes: bytes.Repeat([]byte{255}, length), BitLength: length * 8}}
	}
	return v
}

func TestNotifyApplicationDataVectors(t *testing.T) {
	d, err := LookupClause("9.1.58", "NOTIFY APPLICATION DATA")
	if err != nil {
		t.Fatal(err)
	}
	for _, vector := range notifyApplicationVectors {
		t.Run(fmt.Sprintf("length-%d/extent-%d", vector.length, vector.octets), func(t *testing.T) {
			wire, err := hex.DecodeString(vector.hex)
			if err != nil {
				t.Fatal(err)
			}
			value := notifyApplicationValue(vector.length)
			canonical, err := EncodeNOTIFYAPPLICATIONDATACanonicalAtLength(value, vector.octets)
			if err != nil || !bytes.Equal(canonical, wire) {
				t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
			}
			decoded, err := d.Decode(wire)
			if err != nil {
				t.Fatal(err)
			}
			replay, err := d.Encode(decoded)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			typed, ok := decoded.(runtime.Decoded[NOTIFYAPPLICATIONDATA])
			if !ok {
				t.Fatalf("decoded type %T", decoded)
			}
			if !runtime.SemanticallyEqual(typed.Value, value) || typed.BitsConsumed != 8*vector.octets {
				t.Fatalf("typed decode %#v", typed)
			}
			stable, err := d.CanonicalAtLength(decoded, vector.octets)
			if err != nil || !bytes.Equal(stable, wire) {
				t.Fatalf("stable %x: %v", stable, err)
			}
			_, err = EncodeNOTIFYAPPLICATIONDATACanonical(value)
			var extent *runtime.ExtentError
			if !errors.As(err, &extent) || !extent.Required {
				t.Fatalf("default canonical %v", err)
			}
			minimum := (vector.contentBits + 7) / 8
			for _, n := range []int{-1, 0, minimum - 1, 1<<17 + 1} {
				if _, err := EncodeNOTIFYAPPLICATIONDATACanonicalAtLength(value, n); err == nil {
					t.Errorf("accepted extent %d", n)
				}
			}
			for cut := 0; cut < vector.contentBits; cut++ {
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeNOTIFYAPPLICATIONDATAFrom(r); err == nil {
					t.Errorf("accepted %d-bit truncation", cut)
				}
				if r.Position() > cut {
					t.Fatalf("crossed reader bound %d", cut)
				}
			}
			r := runtime.NewReader(append(append([]byte{}, wire...), 0xa5))
			old, err := r.PushLimit(8 * len(wire))
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeNOTIFYAPPLICATIONDATAFrom(r)
			if err != nil || !runtime.SemanticallyEqual(got, value) {
				t.Fatalf("bounded decode: %v", err)
			}
			if r.Position() != 8*len(wire) {
				t.Fatal("wrong bounded consumption")
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			sentinel, err := r.ReadUint(8)
			if err != nil || sentinel != 0xa5 {
				t.Fatalf("suffix %x: %v", sentinel, err)
			}
			for selector := 0; selector < 32; selector++ {
				if selector == 13 {
					continue
				}
				bad := append([]byte{}, wire...)
				bad[0] = (bad[0] & 0x83) | byte(selector<<2)
				if _, err := d.Decode(bad); err == nil {
					t.Errorf("accepted message type %d", selector)
				}
			}
		})
	}
}

func TestNotifyApplicationDataFieldBoundaries(t *testing.T) {
	cases := []struct {
		name string
		edit func(*NOTIFYAPPLICATIONDATA)
	}{
		{"short PD", func(v *NOTIFYAPPLICATIONDATA) { v.RRShortPD = 2 }},
		{"message type", func(v *NOTIFYAPPLICATIONDATA) { v.MessageType = 12 }},
		{"short header", func(v *NOTIFYAPPLICATIONDATA) { v.ShortLayer2Header = 4 }},
		{"application width", func(v *NOTIFYAPPLICATIONDATA) { v.ApplicationData.BitLength = 71 }},
		{"MSISDN width", func(v *NOTIFYAPPLICATIONDATA) { v.MSISDNLengthGroup.MSISDN.BitLength = 7 }},
		{"MSISDN length", func(v *NOTIFYAPPLICATIONDATA) { v.MSISDNLengthGroup.MSISDNLength = 16 }},
		{"MSISDN inferred mismatch", func(v *NOTIFYAPPLICATIONDATA) { v.MSISDNLengthGroup.MSISDNLength = 2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := notifyApplicationValue(1)
			tc.edit(&v)
			if _, err := EncodeNOTIFYAPPLICATIONDATACanonicalAtLength(v, 32); err == nil {
				t.Fatal("accepted invalid field")
			}
		})
	}
	for _, length := range []int{-1, 0, 1, 7, 15} {
		for _, pd := range []uint8{0, 1} {
			for header := uint8(0); header < 4; header++ {
				v := notifyApplicationValue(length)
				v.RRShortPD = pd
				v.ShortLayer2Header = header
				wire, err := EncodeNOTIFYAPPLICATIONDATACanonicalAtLength(v, 32)
				if err != nil {
					t.Fatal(err)
				}
				d, err := LookupClause("9.1.58", "NOTIFY APPLICATION DATA")
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := d.Decode(wire)
				if err != nil {
					t.Fatal(err)
				}
				got := decoded.(runtime.Decoded[NOTIFYAPPLICATIONDATA]).Value
				if !runtime.SemanticallyEqual(got, v) {
					t.Fatal("header changed")
				}
			}
		}
	}
}

func FuzzNotifyApplicationDataReplay(f *testing.F) {
	for _, v := range notifyApplicationVectors {
		wire, err := hex.DecodeString(v.hex)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(wire)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 128 {
			return
		}
		d, err := LookupClause("9.1.58", "NOTIFY APPLICATION DATA")
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := d.Decode(wire)
		if err != nil {
			return
		}
		replay, err := d.Encode(decoded)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %v", replay, err)
		}
	})
}
