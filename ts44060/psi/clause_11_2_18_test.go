package psi_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/psi"
)

type psiVector struct {
	name, body              string
	extent, canonicalExtent int
	padded                  bool
	expected                map[string][]uint64
}
type psiCodec struct {
	tag         uint64
	mandatory   int
	decode      func(*runtime.Reader) (any, error)
	encode      func(*runtime.Writer, any) error
	canonical   func(any) any
	canonicalAt func(any, int) ([]byte, error)
	page        func(any) (uint8, bool)
	edit        func(any, uint8) any
}

func psiPack(bits string) []byte {
	b := make([]byte, (len(bits)+7)/8)
	for i, v := range bits {
		if v == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func psiImage(tag uint64, body string, extent int, pad bool) []byte {
	bits := fmt.Sprintf("01000000%06b", tag) + body
	if pad && len(bits) < extent {
		bits += "0"
		for len(bits) < extent {
			bits += string("00101011"[len(bits)%8])
		}
	}
	return psiPack(bits)
}
func psiNormalize(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, strings.TrimSuffix(s, "List"))
}
func psiFields(value any) map[string][]uint64 {
	out := map[string][]uint64{}
	var walk func(reflect.Value, string)
	walk = func(v reflect.Value, name string) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if !v.IsNil() {
				walk(v.Elem(), name)
			}
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			if v.Type() == reflect.TypeFor[runtime.BitString]() {
				return
			}
			for i := 0; i < v.NumField(); i++ {
				f := v.Type().Field(i)
				if f.IsExported() && f.Name != "Wire" && f.Name != "Alternative" {
					walk(v.Field(i), f.Name)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i), name)
			}
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			out[psiNormalize(name)] = append(out[psiNormalize(name)], v.Uint())
		}
	}
	walk(reflect.ValueOf(value), "")
	return out
}
func psiRead(t testing.TB, raw []byte, extent int, tag uint64) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	rd, e := runtime.NewChoiceReaderBits(raw, extent, c)
	if e != nil {
		t.Fatal(e)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, e = controlblock.DecodeBlockHeaderFrom(rd, ctx, controlblock.CS1); e != nil {
		t.Fatal(e)
	}
	got, e := rd.ReadUint(6)
	if e != nil || got != tag {
		t.Fatalf("selector %d: %v", got, e)
	}
	return rd, c
}
func psiEncode(t testing.TB, codec psiCodec, value any, raw []byte, extent int, replay *runtime.Reader) runtime.BitString {
	t.Helper()
	raw = append([]byte(nil), raw[:(extent+7)/8]...)
	if extent%8 != 0 {
		raw[len(raw)-1] &= 0xff << uint(8-extent%8)
	}
	w := runtime.NewWriter()
	if replay != nil {
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, replay.Wire())
		if e != nil {
			t.Fatal(e)
		}
		w.WithWire(wire)
	} else {
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
	}
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(0x40, 8); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(codec.tag, 6); e != nil {
		t.Fatal(e)
	}
	if e := codec.encode(w, value); e != nil {
		t.Fatal(e)
	}
	out, e := w.FinishBits(runtime.BitString{})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func psiEqualBits(t testing.TB, got runtime.BitString, want []byte, extent int) {
	t.Helper()
	want = append([]byte(nil), want[:(extent+7)/8]...)
	if extent%8 != 0 {
		want[len(want)-1] &= 0xff << uint(8-extent%8)
	}
	if got.BitLength != extent || !bytes.Equal(got.Bytes, want) {
		t.Fatalf("wire %s/%d, want %x/%d", hex.EncodeToString(got.Bytes), got.BitLength, want, extent)
	}
}

// TS 44.060 V19.0.0 §§10.0b.2.1, 10.3.1.1 and 11.2.0.1:
// each 184-bit downlink block carries an eight-bit MAC header and a six-bit selector.
// Short release variants use an explicit sub-bound and a following sentinel.
func psiCheckVectors(t *testing.T, codec psiCodec, rows []psiVector) {
	t.Helper()
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			raw := psiImage(codec.tag, row.body, row.extent, row.padded)
			carrier := raw
			if row.extent < 184 {
				bits := fmt.Sprintf("01000000%06b", codec.tag) + row.body + "101"
				for len(bits) < 184 {
					bits += "0"
				}
				carrier = psiPack(bits)
			}
			rd, c := psiRead(t, carrier, 184, codec.tag)
			old, e := rd.PushLimit(row.extent - 14)
			if e != nil {
				t.Fatal(e)
			}
			v, e := codec.decode(rd)
			if e != nil {
				t.Fatal(e)
			}
			page, known := codec.page(v)
			if !known || rd.Position() != row.extent {
				t.Fatalf("typed content %v at %d/%d", v, rd.Position(), row.extent)
			}
			fields := psiFields(v)
			for name, want := range row.expected {
				if !reflect.DeepEqual(fields[name], want) {
					t.Fatalf("field %s = %v, want %v", name, fields[name], want)
				}
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("unexpected error region %+v", report)
			}
			psiEqualBits(t, psiEncode(t, codec, v, raw, row.extent, rd), raw, row.extent)
			// TS 44.060 V19.0.0 paragraphs 16838, 17693 and 17773: no-string remainders carry no sender bits.
			psiEqualBits(t, psiEncode(t, codec, codec.canonical(v), raw, row.canonicalExtent, nil), raw, row.canonicalExtent)
			edited := codec.edit(v, (page+1)%4)
			out := psiEncode(t, codec, edited, raw, row.extent, rd)
			want := append([]byte(nil), raw...)
			want[1] = (want[1] & 0xfc) | ((page + 1) % 4)
			psiEqualBits(t, out, want, row.extent)
			if e = rd.PopLimit(old); e != nil {
				t.Fatal(e)
			}
			if row.extent < 184 {
				sentinel, e := rd.ReadUint(3)
				if e != nil || sentinel != 5 {
					t.Fatalf("sentinel %d: %v", sentinel, e)
				}
			}
		})
	}
}
func psiCheckMandatoryCuts(t *testing.T, codec psiCodec, row psiVector) {
	t.Helper()
	raw := psiImage(codec.tag, row.body, row.extent, row.padded)
	for n := 0; n < codec.mandatory; n++ {
		c := runtime.NewReceiverCollector()
		rd, e := runtime.NewChoiceReaderBits(raw, 14+n, c)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = rd.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := codec.decode(rd)
		if e != nil {
			t.Fatalf("mandatory cut %d: %v", n, e)
		}
		if _, known := codec.page(v); known {
			t.Fatalf("mandatory cut %d accepted as known", n)
		}
		report := c.TakeReceiverReport()
		if len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || report.Regions[0].Location.Width != n {
			t.Fatalf("mandatory cut %d region %+v", n, report)
		}
	}
}
func psiFuzz(f *testing.F, codec psiCodec, row psiVector) {
	f.Add(psiImage(codec.tag, row.body, 184, true))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		rd, e := runtime.NewChoiceReaderBits(raw, 184, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = rd.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := codec.decode(rd)
		if e == nil {
			extent := rd.Position()
			out := psiEncode(t, codec, v, raw, extent, rd)
			out.Bytes[0] = raw[0]
			out.Bytes[1] = (out.Bytes[1] & 3) | (raw[1] & 0xfc)
			psiEqualBits(t, out, raw, extent)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.18; exact extracted paragraph span is recorded in the source intake.
// TS 44.060 V19.0.0 §11.2.0.1 paragraph 13371: external selector 110001.
var psi1Codec = psiCodec{tag: 49, mandatory: 109,
	decode: func(r *runtime.Reader) (any, error) { return psi.DecodePSI1MessageContentFrom(r) },
	encode: func(w *runtime.Writer, v any) error {
		return psi.EncodePSI1MessageContentTo(w, v.(psi.PSI1MessageContent))
	},
	canonical: func(v any) any { return runtime.Canonical(v.(psi.PSI1MessageContent)) },
	canonicalAt: func(v any, n int) ([]byte, error) {
		return psi.EncodePSI1MessageContentCanonicalAtLength(v.(psi.PSI1MessageContent), n)
	},
	page: func(v any) (uint8, bool) {
		p := v.(psi.PSI1MessageContent).PAGEMODEGroupFallback.Known
		if p == nil {
			return 0, false
		}
		return p.PAGEMODE, true
	},
	edit: func(v any, page uint8) any {
		p := v.(psi.PSI1MessageContent)
		p.PAGEMODEGroupFallback.Known.PAGEMODE = page
		return p
	},
}
var psi1Vectors = []psiVector{
	{name: "page-0-hr-None-null", body: "0010110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 123, canonicalExtent: 123, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-None-zero", body: "00101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 124, canonicalExtent: 124, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-None-r99-null", body: "00101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-None-r99-zero", body: "001011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-None-r6-no-power", body: "0010110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-None-r6-power", body: "001011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-null", body: "00101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-zero", body: "001011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-r99-null", body: "001011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 131, canonicalExtent: 131, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-r99-zero", body: "0010110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 132, canonicalExtent: 132, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-r6-no-power", body: "00101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-0-hr-11-r6-power", body: "0010110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {0}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-null", body: "0110110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 123, canonicalExtent: 123, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-zero", body: "01101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 124, canonicalExtent: 124, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-r99-null", body: "01101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-r99-zero", body: "011011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-r6-no-power", body: "0110110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-None-r6-power", body: "011011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-null", body: "01101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-zero", body: "011011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-r99-null", body: "011011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 131, canonicalExtent: 131, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-r99-zero", body: "0110110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 132, canonicalExtent: 132, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-r6-no-power", body: "01101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-1-hr-11-r6-power", body: "0110110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {1}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-null", body: "1010110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 123, canonicalExtent: 123, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-zero", body: "10101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 124, canonicalExtent: 124, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-r99-null", body: "10101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-r99-zero", body: "101011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-r6-no-power", body: "1010110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-None-r6-power", body: "101011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-null", body: "10101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-zero", body: "101011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-r99-null", body: "101011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 131, canonicalExtent: 131, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-r99-zero", body: "1010110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 132, canonicalExtent: 132, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-r6-no-power", body: "10101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-2-hr-11-r6-power", body: "1010110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {2}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-null", body: "1110110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 123, canonicalExtent: 123, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-zero", body: "11101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 124, canonicalExtent: 124, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-r99-null", body: "11101101010011001010101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-r99-zero", body: "111011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-r6-no-power", body: "1110110101001100101010101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-None-r6-power", body: "111011010100110010101010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-null", body: "11101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011", extent: 127, canonicalExtent: 127, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-zero", body: "111011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100110", extent: 128, canonicalExtent: 128, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-r99-null", body: "111011010100110010111011101010011100101001001001011000111100000110110110101001101001010101110101110011011010100111101", extent: 131, canonicalExtent: 131, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-r99-zero", body: "1110110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111010", extent: 132, canonicalExtent: 132, padded: false, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-r6-no-power", body: "11101101010011001011101110101001110010100100100101100011110000011011011010100110100101010111010111001101101010011110110", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
	{name: "page-3-hr-11-r6-power", body: "1110110101001100101110111010100111001010010010010110001111000001101101101010011010010101011101011100110110101001111011111011", extent: 184, canonicalExtent: 184, padded: true, expected: map[string][]uint64{"acccontrclass": {38460}, "accessbursttype": {1}, "alpha": {7}, "bandindicator": {1}, "bscvmax": {9}, "bspagblksres": {9}, "bspbcchblks": {2}, "bspccrel": {1}, "bsprachblks": {5}, "controlacktype": {0}, "drxtimermax": {4}, "lbmstxpwrmaxcch": {27}, "maxretrans": {0, 1, 2, 3}, "measurementorder": {1}, "mscr": {1}, "navgi": {9}, "nmo": {1}, "pagemode": {3}, "pb": {6}, "pbcchchangemark": {5}, "pcmeaschan": {1}, "psi1repeatperiod": {9}, "psichangefield": {10}, "psicounthr": {11}, "psicountlr": {37}, "psistatusind": {1}, "s": {6}, "sgsnr": {0}, "t3168": {2}, "t3192": {3}, "tavgt": {19}, "tavgw": {11}, "txint": {10}}},
}

func TestClause11218TypedDistribution(t *testing.T) {
	psiCheckVectors(t, psi1Codec, psi1Vectors)
	psiCheckMandatoryCuts(t, psi1Codec, psi1Vectors[0])
}
func FuzzPSI1Distribution(f *testing.F) { psiFuzz(f, psi1Codec, psiCompleteVector(psi1Vectors)) }

// TS 44.060 V19.0.0 §11 defines positional spare padding. Keep the
// content at phase 6, including non-octet bounds; reject other phases before I/O.
func psiCheckPhases(t *testing.T, codec psiCodec, row psiVector) {
	t.Helper()
	for _, origin := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		extent := origin + 170
		bits := strings.Repeat("0", origin) + row.body + "0"
		for len(bits) < extent {
			bits += string("00101011"[len(bits)%8])
		}
		if len(bits) > extent {
			t.Fatal("phase fixture exceeds its containing extent")
		}
		raw := psiPack(bits)
		rd, e := runtime.NewChoiceReaderBits(raw, extent, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = rd.ReadUint(origin); e != nil {
			t.Fatal(e)
		}
		v, e := codec.decode(rd)
		if origin%8 != 6 {
			var phaseError *psi.ContentPhaseError
			if !errors.As(e, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != origin || rd.Position() != origin {
				t.Fatalf("reader phase %d changed cursor or error: %v", origin, e)
			}
			target := runtime.NewWriter()
			if e := target.WriteUint(0, origin); e != nil {
				t.Fatal(e)
			}
			writerErr := codec.encode(target, v)
			if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != origin || target.Position() != origin {
				t.Fatalf("writer phase %d changed cursor or error: %v", origin, writerErr)
			}
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		if _, known := codec.page(v); !known || rd.Position() != extent {
			t.Fatalf("phase %d: %v at %d", origin, v, rd.Position())
		}
		fields := psiFields(v)
		for name, want := range row.expected {
			if !reflect.DeepEqual(fields[name], want) {
				t.Fatalf("phase %d field %s=%v, want %v", origin, name, fields[name], want)
			}
		}
		for _, canonical := range []bool{false, true} {
			w := runtime.NewWriter()
			value := v
			if canonical {
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
				value = codec.canonical(v)
			} else {
				wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: extent}, extent, runtime.BitString{}, rd.Wire())
				if e != nil {
					t.Fatal(e)
				}
				w.WithWire(wire)
			}
			if _, e = w.PushLimit(extent); e != nil {
				t.Fatal(e)
			}
			if e = w.WriteUint(0, origin); e != nil {
				t.Fatal(e)
			}
			if e = codec.encode(w, value); e != nil {
				t.Fatal(e)
			}
			got, e := w.FinishBits(runtime.BitString{})
			if e != nil {
				t.Fatal(e)
			}
			psiEqualBits(t, got, raw, extent)
		}
	}
}
func TestDistributionPaddingPhases(t *testing.T) {
	for _, entry := range []struct {
		name  string
		codec psiCodec
		row   psiVector
	}{
		{"PSI1", psi1Codec, psiCompleteVector(psi1Vectors)},
		{"PSI2", psi2Codec, psiCompleteVector(psi2Vectors)},
		{"PSI3ter", psi3terCodec, psiCompleteVector(psi3terVectors)},
		{"PSI13", psi13Codec, psiCompleteVector(psi13Vectors)},
		{"PSI14", psi14Codec, psiCompleteVector(psi14Vectors)},
		{"PSI15", psi15Codec, psiCompleteVector(psi15Vectors)},
	} {
		t.Run(entry.name, func(t *testing.T) { psiCheckPhases(t, entry.codec, entry.row) })
	}
}

// TS 44.060 V19.0.0 §11.2.18 and §12.9: the bit immediately before
// N_AVG_I is printed as 0, so 1 selects the Distribution part error region.
func TestPSI1MalformedDistribution(t *testing.T) {
	row := psiCompleteVector(psi1Vectors)
	raw := psiImage(psi1Codec.tag, row.body, 184, true)
	const reservedBit = 14 + 107
	raw[reservedBit/8] |= 1 << uint(7-reservedBit%8)
	rd, c := psiRead(t, raw, 184, psi1Codec.tag)
	v, e := psi1Codec.decode(rd)
	if e != nil {
		t.Fatal(e)
	}
	if _, known := psi1Codec.page(v); known {
		t.Fatal("nonzero printed zero accepted as known PSI1")
	}
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || report.Regions[0].Location.Width != 170 {
		t.Fatalf("distribution error report %+v", report)
	}
	if _, e = psi1Codec.canonicalAt(v, 22); e == nil {
		t.Fatal("receiver error arm has a canonical sender encoding")
	}
	psiEqualBits(t, psiEncode(t, psi1Codec, v, raw, 184, rd), raw, 184)
}

func psiCompleteVector(rows []psiVector) psiVector {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].padded && rows[i].canonicalExtent == 184 {
			return rows[i]
		}
	}
	panic("no complete sender vector")
}

func TestPublicPSICanonicalAtLength(t *testing.T) {
	for _, entry := range []struct {
		name  string
		codec psiCodec
		rows  []psiVector
	}{
		{"PSI1", psi1Codec, psi1Vectors}, {"PSI2", psi2Codec, psi2Vectors}, {"PSI3ter", psi3terCodec, psi3terVectors}, {"PSI13", psi13Codec, psi13Vectors}, {"PSI14", psi14Codec, psi14Vectors}, {"PSI15", psi15Codec, psi15Vectors},
	} {
		t.Run(entry.name, func(t *testing.T) {
			row := psiCompleteVector(entry.rows)
			raw := psiImage(entry.codec.tag, row.body, 184, true)
			rd, _ := psiRead(t, raw, 184, entry.codec.tag)
			v, e := entry.codec.decode(rd)
			if e != nil {
				t.Fatal(e)
			}
			// The byte API translates its octet extent to source origin 14.
			// The block tests above also cover the exact 170-bit content extent.
			out, e := entry.codec.canonicalAt(v, 22)
			if e != nil {
				t.Fatal(e)
			}
			bits := strings.Repeat("0", 14) + row.body + "0"
			for len(bits) < 190 {
				bits += string("00101011"[len(bits)%8])
			}
			if !bytes.Equal(out, psiPack(bits[14:])) {
				t.Fatalf("canonical bytes %x, want %x", out, psiPack(bits[14:]))
			}
		})
	}
}
