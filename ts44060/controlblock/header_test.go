package controlblock

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestControlHeaderModes(t *testing.T) {
	// Figure 10.3.1.1.1: 10|10|1|101, 0|10001|0|1,
	// 10|10101|1 = ad 45 ab. Extended: 1|10001|0|1 = c5;
	// 011|1|1010 = 7a. Nonzero spare bits must replay unchanged.
	for _, raw := range [][]byte{{0xad, 0x45, 0xab}, {0xad, 0xc5, 0xab, 0x7a}, {0x6d}} {
		r := runtime.NewReader(raw)
		h, err := DecodeDownlinkHeader(r)
		if err != nil {
			t.Fatal(err)
		}
		if h.MAC.RRBP != 2 || h.MAC.SP != 1 || h.MAC.USF != 5 {
			t.Fatalf("MAC: %+v", h)
		}
		if len(raw) > 1 && (h.Control.RTI != 17 || h.Control.Address.TFI != 21 || h.Control.Address.D != 1 || h.Control.Address.PR != 2) {
			t.Fatalf("control: %+v", h.Control)
		}
		if len(raw) == 4 && (h.Control.Extension.RBSNe != 3 || h.Control.Extension.FSe != 1 || h.Control.Extension.Spare != 10) {
			t.Fatal(h.Control.Extension)
		}
		w := runtime.NewWriter()
		if err := EncodeDownlinkHeader(w, h); err != nil {
			t.Fatal(err)
		}
		got, err := paddedHeaderBytes(w)
		if err != nil || !bytes.Equal(got, raw) || r.Position() != len(raw)*8 {
			t.Fatalf("%x => %x, %v", raw, got, err)
		}
	}
	// Figure 10.3.2.1: 01|10101|1 = 6b.
	r := runtime.NewReader([]byte{0x6b})
	u, err := DecodeUplinkHeader(r)
	if err != nil || u.Spare != 21 || u.R != 1 {
		t.Fatalf("uplink: %+v %v", u, err)
	}
	w := runtime.NewWriter()
	if err := EncodeUplinkHeader(w, u); err != nil {
		t.Fatal(err)
	}
	got, err := paddedHeaderBytes(w)
	if err != nil || !bytes.Equal(got, []byte{0x6b}) {
		t.Fatal(got, err)
	}
	// Figures 10.3.1.3.1/2: PR|PT|RRBP|SP = 1|0|10|1;
	// extended 1|1|10|1|0|0|1|10101 occupies 13 bits.
	for _, tc := range []struct {
		raw []byte
		n   int
		ext bool
	}{{[]byte{0xa8}, 5, false}, {[]byte{0xe9, 0xa8}, 13, true}} {
		r := runtime.NewReader(tc.raw)
		h, err := DecodeECDownlinkHeader(r)
		if err != nil || r.Position() != tc.n || h.PR != 1 || h.RRBP != 2 || h.SP != 1 || (h.Extended != nil) != tc.ext {
			t.Fatalf("EC: %+v %v pos=%d", h, err, r.Position())
		}
		w := runtime.NewWriter()
		if err := EncodeECDownlinkHeader(w, h); err != nil {
			t.Fatal(err)
		}
		got, err := paddedHeaderBytes(w)
		if err != nil || !bytes.Equal(got, tc.raw) {
			t.Fatalf("EC replay %x %v", got, err)
		}
	}
}

func TestHeaderErrors(t *testing.T) {
	for _, raw := range [][]byte{nil, {0xad}, {0xad, 0x45}, {0xad, 0xc5, 0xab}} {
		_, err := DecodeDownlinkHeader(runtime.NewReader(raw))
		var located *runtime.DecodeError
		if err == nil || !errors.As(err, &located) {
			t.Fatalf("%x: %v", raw, err)
		}
	}
	for _, pt := range []byte{0, 0xc0} {
		if _, err := DecodeDownlinkHeader(runtime.NewReader([]byte{pt})); err == nil {
			t.Fatalf("accepted payload type %x", pt)
		}
	}
	for _, pt := range []byte{0, 0x80, 0xc0} {
		if _, err := DecodeUplinkHeader(runtime.NewReader([]byte{pt})); err == nil {
			t.Fatalf("accepted UL payload type %x", pt)
		}
	}
	for _, h := range []DownlinkHeader{{MAC: DownlinkMAC{PayloadType: 2}}, {MAC: DownlinkMAC{PayloadType: 1}, Control: &ControlHeader{}}, {MAC: DownlinkMAC{PayloadType: 2}, Control: &ControlHeader{RTI: 32}}} {
		if err := EncodeDownlinkHeader(runtime.NewWriter(), h); err == nil {
			t.Fatalf("accepted inconsistent %+v", h)
		}
	}
}

func TestHeaderCanonicalSpareBits(t *testing.T) {
	h, err := DecodeDownlinkHeader(runtime.NewReader([]byte{0xad, 0xc5, 0xab, 0x7a}))
	if err != nil {
		t.Fatal(err)
	}
	w := runtime.NewWriter()
	if err = CanonicalDownlinkHeader(w, h); err != nil {
		t.Fatal(err)
	}
	b, err := w.Bytes()
	if err != nil || !bytes.Equal(b, []byte{0xad, 0xc5, 0xab, 0x70}) {
		t.Fatal(b, err)
	}
	if h.Control.Extension.Spare != 10 {
		t.Fatal("canonical mutated received header")
	}
	w = runtime.NewWriter()
	if err = CanonicalControlHeader(w, *h.Control); err != nil {
		t.Fatal(err)
	}
	b, err = w.Bytes()
	if err != nil || !bytes.Equal(b, []byte{0xc5, 0xab, 0x70}) {
		t.Fatal(b, err)
	}
	w = runtime.NewWriter()
	u, err := DecodeUplinkHeader(runtime.NewReader([]byte{0x7f}))
	if err != nil {
		t.Fatal(err)
	}
	if err = CanonicalUplinkHeader(w, u); err != nil {
		t.Fatal(err)
	}
	b, err = w.Bytes()
	if err != nil || !bytes.Equal(b, []byte{0x41}) {
		t.Fatal(b, err)
	}
}

func TestFreshHeaderSpareBits(t *testing.T) {
	w := runtime.NewWriter()
	if err := EncodeUplinkHeader(w, UplinkHeader{Spare: 31, R: 1}); err != nil {
		t.Fatal(err)
	}
	b, err := w.Bytes()
	if err != nil || !bytes.Equal(b, []byte{0x41}) {
		t.Fatal(b, err)
	}
	c := ControlHeader{RBSN: 1, RTI: 17, FS: 0, Address: &ControlAddress{PR: 2, TFI: 21, D: 1}, Extension: &ControlExtension{RBSNe: 3, FSe: 1, Spare: 10}}
	w = runtime.NewWriter()
	if err = EncodeControlHeader(w, c); err != nil {
		t.Fatal(err)
	}
	b, err = w.Bytes()
	if err != nil || !bytes.Equal(b, []byte{0xc5, 0xab, 0x70}) {
		t.Fatal(b, err)
	}
}

func FuzzHeaders(f *testing.F) {
	for _, b := range [][]byte{{0xad, 0x45, 0xab}, {0xad, 0xc5, 0xab, 0x7a}, {0x6b}, {0xe9, 0xa8}} {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		r := runtime.NewReader(b)
		h, e := DecodeDownlinkHeader(r)
		if e == nil {
			w := runtime.NewWriter()
			if e = EncodeDownlinkHeader(w, h); e != nil {
				t.Fatal(e)
			}
			raw, e := paddedHeaderBytes(w)
			if e != nil || !bytes.Equal(raw, b[:r.Position()/8]) {
				t.Fatal("header replay", e)
			}
			h2, e := DecodeDownlinkHeader(runtime.NewReader(raw))
			if e != nil || !reflect.DeepEqual(h, h2) {
				t.Fatal("header value", e)
			}
		}
		r = runtime.NewReader(b)
		u, e := DecodeUplinkHeader(r)
		if e == nil {
			w := runtime.NewWriter()
			if e = EncodeUplinkHeader(w, u); e != nil {
				t.Fatal(e)
			}
		}
		r = runtime.NewReader(b)
		ec, e := DecodeECDownlinkHeader(r)
		if e == nil {
			w := runtime.NewWriter()
			if e = EncodeECDownlinkHeader(w, ec); e != nil {
				t.Fatal(e)
			}
			raw, e := paddedHeaderBytes(w)
			if e != nil {
				t.Fatal(e)
			}
			ec2, e := DecodeECDownlinkHeader(runtime.NewReader(raw))
			if e != nil || !reflect.DeepEqual(ec, ec2) {
				t.Fatal("EC value", e)
			}
		}
	})
}

func paddedHeaderBytes(w *runtime.Writer) ([]byte, error) {
	if n := w.Position() % 8; n != 0 {
		if err := w.WriteUint(0, 8-n); err != nil {
			return nil, err
		}
	}
	return w.Bytes()
}

func TestControlBlockExtents(t *testing.T) {
	for _, tc := range []struct {
		format     BlockFormat
		mode       runtime.RLCMACMode
		direction  runtime.RLCMACDirection
		bits, body int
		first      byte
	}{
		{CS1, runtime.RLCMACOrdinary, runtime.RLCMACDownlink, 184, 176, 0x40},
		{CS1, runtime.RLCMACOrdinary, runtime.RLCMACDownlink, 184, 168, 0x80},
		{CS1, runtime.RLCMACOrdinary, runtime.RLCMACUplink, 184, 176, 0x40},
		{CS3, runtime.RLCMACOrdinary, runtime.RLCMACUplink, 312, 304, 0x40},
		{ECPACCH, runtime.RLCMACEC, runtime.RLCMACDownlink, 80, 75, 0},
		{ECPACCH, runtime.RLCMACEC, runtime.RLCMACDownlink, 80, 67, 0x40},
		{ECPACCH, runtime.RLCMACEC, runtime.RLCMACUplink, 64, 64, 0},
	} {
		ctx := ordinaryContext()
		ctx.Mode = tc.mode
		ctx.Direction = tc.direction
		data := make([]byte, tc.bits/8)
		data[0] = tc.first
		r, err := runtime.NewReaderBits(data, tc.bits, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = DecodeBlockHeaderFrom(r, ctx, tc.format)
		if err != nil || r.Remaining() != tc.body {
			t.Fatalf("%+v remaining=%d %v", tc, r.Remaining(), err)
		}
		for _, delta := range []int{-1, 1} {
			r, err = runtime.NewReaderBits(append(data, 0), tc.bits+delta, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = DecodeBlockHeaderFrom(r, ctx, tc.format); err == nil {
				t.Fatalf("accepted extent %d", tc.bits+delta)
			}
		}
	}
	// A containing non-octet prefix and trailing sentinel remain on the same
	// reader: EC header does not round 75 message bits up to 80.
	for phase := 0; phase < 8; phase++ {
		w := runtime.NewWriter()
		_ = w.WriteUint(0, phase)
		_ = w.WriteUint(0, 5)
		_ = w.WriteUint(0, 64)
		_ = w.WriteUint(0, 11)
		_ = w.WriteUint(1, 1)
		bits, _ := w.FinishBits(runtime.BitString{})
		r, _ := runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
		_, _ = r.ReadUint(phase)
		old, err := r.PushLimit(80)
		if err != nil {
			t.Fatal(err)
		}
		ctx := ordinaryContext()
		ctx.Mode = runtime.RLCMACEC
		_, err = DecodeBlockHeaderFrom(r, ctx, ECPACCH)
		if err != nil || r.Position() != phase+5 || r.Remaining() != 75 {
			t.Fatal(phase, r.Position(), err)
		}
		_, _ = r.ReadBitString(75)
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		v, err := r.ReadUint(1)
		if err != nil || v != 1 {
			t.Fatal("sentinel", err)
		}
	}
}
