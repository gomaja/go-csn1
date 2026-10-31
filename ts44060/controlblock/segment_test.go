package controlblock

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func ordinaryContext() runtime.RLCMACContext {
	return runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
}

func fragments(n int) []Segment {
	s := make([]Segment, n)
	for i := range s {
		c := &ControlHeader{RTI: 7}
		if i > 0 {
			c.RBSN = 1
		}
		if n > 2 && i > 0 {
			c.Extension = &ControlExtension{RBSNe: uint64(i - 1)}
			if i == n-1 {
				c.Extension.FSe = 1
			}
		} else if i == n-1 {
			c.FS = 1
		}
		s[i] = Segment{Format: CS1, Header: c, Contents: runtime.BitString{Bytes: bytes.Repeat([]byte{byte(i + 1)}, 21), BitLength: 168}}
		if c.Extension != nil {
			s[i].Contents.BitLength = 160
			s[i].Contents.Bytes = s[i].Contents.Bytes[:20]
		}
	}
	return s
}

func TestSegmentation(t *testing.T) {
	ctx := ordinaryContext()
	for _, n := range []int{1, 2, 3, 9} {
		s := fragments(n)
		got, err := Reassemble(ctx, "11.2.47", s)
		want := []byte{}
		for _, f := range s {
			want = append(want, f.Contents.Bytes...)
		}
		if err != nil || got.BitLength != len(want)*8 || !bytes.Equal(got.Bytes, want) {
			t.Fatalf("%d blocks: %x %v", n, got.Bytes, err)
		}
		if n == 1 {
			s[0].Contents.Bytes[0] ^= 255
			if got.Bytes[0] != 1 {
				t.Fatal("output aliases input")
			}
		}
	}
	for _, change := range []func([]Segment){
		func(s []Segment) { s[1].Header.RTI = 8 }, func(s []Segment) { s[1].Header.RBSN = 0 }, func(s []Segment) { s[0].Header.FS = 1 }, func(s []Segment) { s[2].Header.Extension.FSe = 0 }, func(s []Segment) { s[1].Header.Extension.RBSNe = 1 }, func(s []Segment) { s[1].Header.Extension.FSe = 1 }, func(s []Segment) { s[1].Contents.BitLength = 159 },
	} {
		s := fragments(3)
		change(s)
		_, err := Reassemble(ctx, "11.2.47", s)
		var e *SegmentError
		if !errors.As(err, &e) || e.Block < 0 || e.Offset < 0 {
			t.Fatalf("missing location: %v", err)
		}
	}
	for _, n := range []int{0, 10} {
		if _, err := Reassemble(ctx, "11.2.47", fragments(n)); err == nil {
			t.Fatalf("accepted %d", n)
		}
	}
	if _, err := Reassemble(ctx, "11.2.1", fragments(3)); err == nil {
		t.Fatal("extended segmentation without permission")
	}
	ctx.Direction = runtime.RLCMACUplink
	if _, err := Reassemble(ctx, "11.2.1", fragments(2)); err == nil {
		t.Fatal("segmented uplink")
	}
}

func TestSegmentationProhibitions(t *testing.T) {
	for _, clause := range []string{"11.2.9e", "11.2.17a", "11.2.17c", "11.2.18", "11.2.19", "11.2.20", "11.2.21", "11.2.21a", "11.2.21b", "11.2.23", "11.2.23a", "11.2.23b", "11.2.24", "11.2.25", "11.2.25c", "11.2.40"} {
		if _, err := Reassemble(ordinaryContext(), clause, fragments(2)); err == nil {
			t.Fatalf("%s accepted segmentation", clause)
		}
		if _, err := Reassemble(ordinaryContext(), clause, fragments(1)); err != nil {
			t.Fatalf("%s: %v", clause, err)
		}
	}
}

func TestExtendedSegmentationPermissions(t *testing.T) {
	yes, no := true, false
	for _, clause := range []string{"11.2.7", "11.2.7a", "11.2.29", "11.2.29a", "11.2.31", "11.2.31a", "11.2.37"} {
		ctx := ordinaryContext()
		if _, err := Reassemble(ctx, clause, fragments(3)); !errors.Is(err, runtime.ErrContextRequired) {
			t.Fatalf("%s missing capability: %v", clause, err)
		}
		ctx.DLDC = &yes
		if _, err := Reassemble(ctx, clause, fragments(3)); err != nil {
			t.Fatalf("%s DLDC: %v", clause, err)
		}
		ctx.DLDC = &no
		if _, err := Reassemble(ctx, clause, fragments(3)); err == nil {
			t.Fatalf("%s DLDC false", clause)
		}
		ctx.DLMC = &yes
		_, err := Reassemble(ctx, clause, fragments(3))
		allowed := clause == "11.2.7" || clause == "11.2.29" || clause == "11.2.31"
		if (err == nil) != allowed {
			t.Fatalf("%s DLMC: %v", clause, err)
		}
	}
	for _, clause := range []string{"11.2.36", "11.2.43", "11.2.46", "11.2.47"} {
		if _, err := Reassemble(ordinaryContext(), clause, fragments(9)); err != nil {
			t.Fatalf("%s: %v", clause, err)
		}
	}
}

func TestSegmentationECAndCS3(t *testing.T) {
	ctx := ordinaryContext()
	ctx.Mode = runtime.RLCMACEC
	for _, n := range []int{1, 2} {
		s := make([]Segment, n)
		for i := range s {
			e := &ECExtendedHeader{RBSN: uint64(i)}
			if i == n-1 {
				e.FS = 1
			}
			s[i] = Segment{Format: ECPACCH, EC: e, Contents: runtime.BitString{Bytes: bytes.Repeat([]byte{0xff}, 9), BitLength: 67}}
		}
		got, err := Reassemble(ctx, "11.2.49", s)
		if err != nil || got.BitLength != n*67 || len(got.Bytes) != (n*67+7)/8 {
			t.Fatal(got, err)
		}
		if n == 2 {
			s[1].EC.RBSN = 0
			if _, err := Reassemble(ctx, "11.2.49", s); err == nil {
				t.Fatal("EC sequence")
			}
		}
	}
	ctx.Direction = runtime.RLCMACUplink
	if _, err := Reassemble(ctx, "11.2.50", []Segment{{Format: ECPACCH, Contents: runtime.BitString{Bytes: make([]byte, 8), BitLength: 64}}}); err != nil {
		t.Fatal(err)
	}
	ctx.Mode = runtime.RLCMACOrdinary
	got, err := Reassemble(ctx, "11.2.48", []Segment{{Format: CS3, Contents: runtime.BitString{Bytes: make([]byte, 38), BitLength: 304}}})
	if err != nil || got.BitLength != 304 {
		t.Fatal(got, err)
	}
}

func TestReassemblyCoordinates(t *testing.T) {
	ctx := ordinaryContext()
	ctx.Mode = runtime.RLCMACEC
	// Header at physical bits0..4, selector at5..9, L at10. The
	// 0x2b L bit at physical10 is1; at stripped position5 it would be0.
	s := []Segment{{Format: ECPACCH, Contents: runtime.BitString{Bytes: append([]byte{4}, make([]byte, 9)...), BitLength: 75}}}
	s[0].Contents.Bytes = s[0].Contents.Bytes[:10]
	m, err := Reassemble(ctx, "11.2.49", s)
	if err != nil {
		t.Fatal(err)
	}
	r, err := m.NewReader(nil)
	if err != nil || r.Position() != 5 {
		t.Fatal(r, err)
	}
	if _, err = r.ReadUint(5); err != nil {
		t.Fatal(err)
	}
	if err = r.Expect("L"); err != nil {
		t.Fatal("EC L/H origin lost", err)
	}
	parts := []Segment{
		{Format: ECPACCH, EC: &ECExtendedHeader{RBSN: 0, FS: 0}, Contents: runtime.BitString{Bytes: make([]byte, 9), BitLength: 67}},
		{Format: ECPACCH, EC: &ECExtendedHeader{RBSN: 1, FS: 1}, Contents: runtime.BitString{Bytes: make([]byte, 9), BitLength: 67}},
	}
	m, err = Reassemble(ctx, "11.2.49", parts)
	if err != nil || m.BitLength != 134 {
		t.Fatal(m, err)
	}
	if _, err = m.NewReader(nil); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatal("EC multipart origin convention must be explicit", err)
	}
	ctx = ordinaryContext()
	m, err = Reassemble(ctx, "11.2.47", fragments(3))
	if err != nil {
		t.Fatal(err)
	}
	r, err = m.NewReader(nil)
	if err != nil || r.Position() != 16 || r.Remaining() != m.BitLength {
		t.Fatal(r, err)
	}
}

func TestSegmentationGrowth(t *testing.T) {
	for n := 1; n <= 9; n++ {
		s := fragments(n)
		got, err := Reassemble(ordinaryContext(), "11.2.47", s)
		if err != nil || len(got.Bytes) > n*22 {
			t.Fatalf("unbounded %d %v", n, err)
		}
	}
	s := fragments(1)
	s[0].Contents.BitLength = int(^uint(0) >> 1)
	if _, err := Reassemble(ordinaryContext(), "11.2.47", s); err == nil {
		t.Fatal("huge extent")
	}
}

func TestSegmentationCodingContext(t *testing.T) {
	ctx := ordinaryContext()
	ctx.CodingScheme = runtime.RLCMACCS3
	if _, err := Reassemble(ctx, "11.2.47", fragments(1)); err == nil {
		t.Fatal("contradictory CS-3/CS-1 facts accepted")
	}
	ctx.CodingScheme = runtime.RLCMACCS1
	if _, err := Reassemble(ctx, "11.2.47", fragments(1)); err != nil {
		t.Fatal(err)
	}
}

func FuzzSegmentation(f *testing.F) {
	f.Add([]byte{3, 0, 0, 1, 0, 0, 1, 1, 1})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) == 0 {
			return
		}
		n := int(b[0] % 11)
		s := fragments(n)
		for i := range s {
			if len(b) > i+1 {
				v := b[i+1]
				s[i].Header.RTI = uint64(v & 31)
				s[i].Header.FS = uint64(v >> 7)
				if s[i].Header.Extension != nil {
					s[i].Header.Extension.RBSNe = uint64(v & 7)
					s[i].Header.Extension.FSe = uint64(v >> 6 & 1)
				}
			}
		}
		got, err := Reassemble(ordinaryContext(), "11.2.47", s)
		if err == nil && (got.BitLength > 9*176 || got.BitLength != len(got.Bytes)*8) {
			t.Fatal("unbounded reassembly")
		}
	})
}
