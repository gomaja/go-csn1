package mbmsmessages_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/mbmsmessages"
)

// TS 44.060 V19.0.0 §11.2.38 pp20312–20317 and §11.2.0.2 p13441:
// uplink ordinary CS-1 block, eight MAC-header bits and external type 001111.
func serviceRequestBits(tlli uint32, service uint32, plmn, session, request, ignored bool) (string, int) {
	bits := fmt.Sprintf("01000000001111%032b", tlli)
	if plmn {
		bits += fmt.Sprintf("1%024b%012b%012b", service, 0x123, 0x456)
	} else {
		bits += fmt.Sprintf("0%024b", service)
	}
	if session {
		bits += "1" + fmt.Sprintf("%08b", 0x9a)
	} else {
		bits += "0"
	}
	if request {
		bits += "1"
	} else {
		bits += "0"
	}
	required := len(bits)
	if len(bits) < 184 {
		if ignored {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	for len(bits) < 184 {
		if ignored {
			bits += "1"
		} else {
			bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
		}
	}
	return bits, required
}
func mbmsBytes(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func mbmsReader(t testing.TB, data []byte, extent int, direction runtime.RLCMACDirection, selector uint64) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, e := runtime.NewChoiceReaderBits(data, extent, c)
	if e != nil {
		t.Fatal(e)
	}
	ctx := runtime.RLCMACContext{Direction: direction, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, e = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); e != nil {
		t.Fatal(e)
	}
	tag, e := r.ReadUint(6)
	if e != nil || tag != selector || r.Position() != 14 {
		t.Fatalf("type %d offset %d: %v", tag, r.Position(), e)
	}
	return r, c
}
func mbmsEncode(t testing.TB, data []byte, extent int, wire runtime.WireInfo, encode func(*runtime.Writer) error) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(uint64(data[0]), 8); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(uint64(data[1]>>2), 6); e != nil {
		t.Fatal(e)
	}
	if e := encode(w); e != nil {
		t.Fatal(e)
	}
	out, e := w.FinishBits(runtime.BitString{})
	if e != nil || out.BitLength != extent {
		t.Fatalf("extent %d: %v", out.BitLength, e)
	}
	return out.Bytes
}
func mbmsReplay(t testing.TB, data []byte, r *runtime.Reader, encode func(*runtime.Writer) error) {
	t.Helper()
	wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if e != nil {
		t.Fatal(e)
	}
	if got := mbmsEncode(t, data, 184, wire, encode); !bytes.Equal(got, data) {
		t.Fatalf("replay %x want %x", got, data)
	}
}
func TestClause11238ServiceRequest(t *testing.T) {
	for _, tlli := range []uint32{0, 0x10203040, 0xffffffff} {
		for _, service := range []uint32{0, 0xabcdef, 0xffffff} {
			for _, plmn := range []bool{false, true} {
				for _, session := range []bool{false, true} {
					for _, request := range []bool{false, true} {
						for _, ignored := range []bool{false, true} {
							bits, required := serviceRequestBits(tlli, service, plmn, session, request, ignored)
							raw := mbmsBytes(bits)
							t.Logf("input=%x", raw)
							r, c := mbmsReader(t, raw, 184, runtime.RLCMACUplink, 15)
							v, e := mbmsmessages.DecodeMBMSServiceRequestMessageContentFrom(r)
							if e != nil || v.TLLI != tlli || r.Position() != 184 || v.MSIDRequestIndication != map[bool]uint8{false: 0, true: 1}[request] {
								t.Fatalf("typed %+v offset %d: %v", v, r.Position(), e)
							}
							choice := v.TMGI.MBMSSERVICEIDChoice
							if plmn {
								if choice.MCC == nil || choice.MBMSSERVICEID != nil || choice.MCC.MBMSSERVICEID != service || choice.MCC.MCC != 0x123 || choice.MCC.MNC != 0x456 {
									t.Fatalf("PLMN %+v", choice)
								}
							} else {
								if choice.MBMSSERVICEID == nil || choice.MCC != nil || choice.MBMSSERVICEID.MBMSSERVICEID != service {
									t.Fatalf("service %+v", choice)
								}
							}
							if (v.MBMSSessionIdentity != nil) != session {
								t.Fatalf("session %+v", v)
							}
							if session && *v.MBMSSessionIdentity != 0x9a {
								t.Fatal("session value")
							}
							if report := c.TakeReceiverReport(); (len(report.Regions) > 0) != ignored {
								t.Fatalf("receiver report %+v", report)
							}
							mbmsReplay(t, raw, r, func(w *runtime.Writer) error { return mbmsmessages.EncodeMBMSServiceRequestMessageContentTo(w, v) })
							if !ignored {
								cv := runtime.Canonical(v)
								wire := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
								if got := mbmsEncode(t, raw, 184, wire, func(w *runtime.Writer) error { return mbmsmessages.EncodeMBMSServiceRequestMessageContentTo(w, cv) }); !bytes.Equal(got, raw) {
									t.Fatalf("canonical %x want %x", got, raw)
								}
								cv.TLLI ^= 0x80000000
								edit := mbmsEncode(t, raw, 184, wire, func(w *runtime.Writer) error { return mbmsmessages.EncodeMBMSServiceRequestMessageContentTo(w, cv) })
								er, _ := mbmsReader(t, edit, 184, runtime.RLCMACUplink, 15)
								ev, err := mbmsmessages.DecodeMBMSServiceRequestMessageContentFrom(er)
								if err != nil || ev.TLLI != tlli^0x80000000 {
									t.Fatalf("edit %+v %v", ev, err)
								}
							}
							for end := 14; end < required; end++ {
								tr, _ := runtime.NewChoiceReaderBits(raw, end, nil)
								if _, e = tr.ReadUint(14); e != nil {
									t.Fatal(e)
								}
								if _, e = mbmsmessages.DecodeMBMSServiceRequestMessageContentFrom(tr); e == nil {
									t.Fatalf("truncation accepted at %d", end)
								}
							}
							// The exact complete content bound must leave the following sentinel untouched.
							extended := mbmsBytes(bits + "101")
							sr, _ := runtime.NewChoiceReaderBits(extended, 187, nil)
							if _, e = sr.ReadUint(14); e != nil {
								t.Fatal(e)
							}
							limit, e := sr.PushLimit(170)
							if e != nil {
								t.Fatal(e)
							}
							if _, e = mbmsmessages.DecodeMBMSServiceRequestMessageContentFrom(sr); e != nil {
								t.Fatal(e)
							}
							if e = sr.PopLimit(limit); e != nil {
								t.Fatal(e)
							}
							sentinel, e := sr.ReadUint(3)
							if e != nil || sentinel != 5 {
								t.Fatalf("sentinel %d %v", sentinel, e)
							}
						}
					}
				}
			}
		}
	}
}
func FuzzClause11238ServiceRequest(f *testing.F) {
	for _, plmn := range []bool{false, true} {
		for _, session := range []bool{false, true} {
			bits, _ := serviceRequestBits(0x10203040, 0xabcdef, plmn, session, true, false)
			f.Add(mbmsBytes(bits))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		raw = append([]byte(nil), raw...)
		raw[0] = 0x40
		raw[1] = (raw[1] & 3) | 60
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		if _, e := r.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := mbmsmessages.DecodeMBMSServiceRequestMessageContentFrom(r)
		if e == nil {
			mbmsReplay(t, raw, r, func(w *runtime.Writer) error { return mbmsmessages.EncodeMBMSServiceRequestMessageContentTo(w, v) })
		}
	})
}
