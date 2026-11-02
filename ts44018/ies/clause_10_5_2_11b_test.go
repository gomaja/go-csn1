package ies

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.018 V19.0.0 §10.5.2.11b, DOCX paragraphs 15140–15149.
// The reviewed reading gives the clause-local entry a distinct name. Each entry
// occupies 1+4+10+10+7=32 bits; the zero terminator has seven spare bits.
// The phase A type-4 minimum precedent gives 6-1=5 root octets (p15136);
// HANDOVER COMMAND table 9.1.15.1 p6317–6322 caps this LV root at 34-1=33.
func TestDynamicARFCNMappingPublication(t *testing.T) {
	for _, count := range []int{0, 1, 2, 8, 32, 63} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var fields strings.Builder
			want := make([]DynamicARFCNMappingEntry, count)
			for i := range count {
				want[i] = DynamicARFCNMappingEntry{GSMBand: uint8((i*3 + 1) % 16), ARFCNFIRST: uint16((i*73 + 3) % 1024), BANDOFFSET: uint16((i*51 + 5) % 1024), ARFCNRANGE: uint8((i*11 + 7) % 128)}
				v := want[i]
				fmt.Fprintf(&fields, "1%04b%010b%010b%07b", v.GSMBand, v.ARFCNFIRST, v.BANDOFFSET, v.ARFCNRANGE)
			}
			fields.WriteString("01111111")
			if count == 0 {
				fields.WriteString(strings.Repeat("00000000", 3))
			}
			bits := fmt.Sprintf("%08b", max(count*4+1, 4)) + fields.String()
			pack := func(bits string) []byte {
				out := make([]byte, (len(bits)+7)/8)
				for i, c := range bits {
					if c == '1' {
						out[i/8] |= 1 << uint(7-i%8)
					}
				}
				return out
			}
			wire := pack(bits)
			decoded, err := DecodeDynamicARFCNMapping(wire)
			if err != nil {
				t.Fatal(err)
			}
			check := func(v DynamicARFCNMapping) {
				t.Helper()
				if int(v.LENGTHOFDYNAMICARFCNMAPPING) != max(count*4+1, 4) || len(v.DYNAMICARFCNMAPPINGDescription) != count {
					t.Fatalf("length/header mismatch: %+v", v)
				}
				for i, e := range v.DYNAMICARFCNMAPPINGDescription {
					g := e.DynamicARFCNMappingEntry
					w := want[i]
					if g.GSMBand != w.GSMBand || g.ARFCNFIRST != w.ARFCNFIRST || g.BANDOFFSET != w.BANDOFFSET || g.ARFCNRANGE != w.ARFCNRANGE {
						t.Fatalf("entry %d: %+v want %+v", i, g, w)
					}
				}
			}
			check(decoded.Value)
			replay, err := EncodeDynamicARFCNMapping(decoded.Value)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			canonical, err := EncodeDynamicARFCNMappingCanonicalAtLength(decoded.Value, len(wire))
			if len(wire) > 33 {
				var extent *runtime.ExtentError
				if canonical != nil || !errors.As(err, &extent) || extent.Maximum != 33 {
					t.Fatalf("past carrier bound: %x %v", canonical, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			expected := append([]byte(nil), wire...)
			expected[len(expected)-1] = 0
			if count == 0 {
				expected[1] = 0
			}
			if len(wire) <= 33 && !bytes.Equal(canonical, expected) {
				t.Fatalf("canonical %x want %x", canonical, expected)
			}
			fresh, err := DecodeDynamicARFCNMapping(wire)
			if err != nil {
				t.Fatal(err)
			}
			check(fresh.Value)
			if count > 0 {
				edit := decoded.Value
				edit.DYNAMICARFCNMAPPINGDescription[0].DynamicARFCNMappingEntry.BANDOFFSET ^= 1
				encoded, e := EncodeDynamicARFCNMapping(edit)
				if e != nil {
					t.Fatal(e)
				}
				got, e := DecodeDynamicARFCNMapping(encoded)
				if e != nil || got.Value.DYNAMICARFCNMAPPINGDescription[0].DynamicARFCNMappingEntry.BANDOFFSET != want[0].BANDOFFSET^1 {
					t.Fatalf("lost edit: %v", e)
				}
				edit.DYNAMICARFCNMAPPINGDescription[0].DynamicARFCNMappingEntry.GSMBand = 16
				if _, e = EncodeDynamicARFCNMapping(edit); e == nil {
					t.Fatal("accepted overflowing band")
				}
			}
			r := runtime.NewReader(append(append([]byte(nil), wire...), 0xa5))
			old, e := r.PushLimit(len(wire) * 8)
			if e != nil {
				t.Fatal(e)
			}
			v, e := DecodeDynamicARFCNMappingFrom(r)
			if e != nil {
				t.Fatal(e)
			}
			check(v)
			if e = r.PopLimit(old); e != nil {
				t.Fatal(e)
			}
			sentinel, e := r.ReadUint(8)
			if e != nil || sentinel != 0xa5 {
				t.Fatalf("sentinel %x: %v", sentinel, e)
			}
			// Every shorter prefix before the list terminator is malformed. Spare
			// truncation is permitted at the caller's bound and is not asserted invalid.
			for cut := 0; cut < 8+32*count+1; cut++ {
				r := runtime.NewReader(wire)
				if _, e = r.PushLimit(cut); e != nil {
					t.Fatal(e)
				}
				if _, e = DecodeDynamicARFCNMappingFrom(r); e == nil {
					t.Fatalf("accepted %d-bit truncated prefix", cut)
				}
				if r.Position() > cut {
					t.Fatalf("crossed bound %d", cut)
				}
			}
		})
	}
}

func FuzzDynamicARFCNMappingPublication(f *testing.F) {
	f.Add([]byte{1, 0x7f})
	f.Add([]byte{5, 0x88, 0x01, 0x80, 0x87, 0x7f})
	f.Fuzz(func(t *testing.T, data []byte) {
		v, e := DecodeDynamicARFCNMapping(data)
		if e != nil {
			return
		}
		out, e := EncodeDynamicARFCNMapping(v.Value)
		if e != nil || !bytes.Equal(out, data) {
			t.Fatalf("replay %x: %v", out, e)
		}
	})
}

func TestDynamicARFCNMappingMinimumAndCarrier(t *testing.T) {
	// TS 44.018 V19.0.0 §10.5.2.11b p15136; §9.1.15 p6317–6322.
	for _, n := range []int{2, 3, 4, 5} {
		wire := make([]byte, n)
		wire[0] = uint8(n - 1)
		_, err := DecodeDynamicARFCNMapping(wire)
		if (err == nil) != (n >= 5) {
			t.Fatalf("minimum %d: %v", n, err)
		}
	}
	d, err := LookupClause("10.5.2.11b", "Dynamic ARFCN Mapping")
	if err != nil {
		t.Fatal(err)
	}
	boundary := make([]byte, 33)
	boundary[0] = 32
	v, err := d.Decode(boundary)
	if err != nil {
		t.Fatal(err)
	}
	if out, e := d.CanonicalAtLength(v, 33); e != nil || len(out) != 33 {
		t.Fatalf("carrier boundary %x %v", out, e)
	}
	out, e := d.CanonicalAtLength(v, 34)
	var extent *runtime.ExtentError
	if out != nil || !errors.As(e, &extent) || extent.Maximum != 33 {
		t.Fatalf("carrier overflow %x %v", out, e)
	}
}
