package messages

import (
	"bytes"
	"encoding/hex"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.61 pp11817-11827 and §10.4 table 10.4.4.
// TS 45.003 V19.0.0 §5.2b.1: the EC-CCCH/D block carries 88 bits.
var ecRejectVectors = []struct {
	mask, dl, content int
	wire              string
}{
	{0, 0, 30, "21234ad32b2b2b2b2b2b2b"},
	{0, 1, 30, "25234ad32b2b2b2b2b2b2b"},
	{0, 2, 30, "29234ad32b2b2b2b2b2b2b"},
	{0, 3, 30, "2d234ad32b2b2b2b2b2b2b"},
	{1, 0, 34, "237234ad2b2b2b2b2b2b2b"},
	{1, 1, 34, "277234ad2b2b2b2b2b2b2b"},
	{1, 2, 34, "2b7234ad2b2b2b2b2b2b2b"},
	{1, 3, 34, "2f7234ad2b2b2b2b2b2b2b"},
	{2, 0, 51, "21234adffb7fcb2b2b2b2b"},
	{2, 1, 51, "25234adffb7fcb2b2b2b2b"},
	{2, 2, 51, "29234adffb7fcb2b2b2b2b"},
	{2, 3, 51, "2d234adffb7fcb2b2b2b2b"},
	{3, 0, 55, "237234adffb7fd2b2b2b2b"},
	{3, 1, 55, "277234adffb7fd2b2b2b2b"},
	{3, 2, 55, "2b7234adffb7fd2b2b2b2b"},
	{3, 3, 55, "2f7234adffb7fd2b2b2b2b"},
	{4, 0, 51, "21234ad400200b2b2b2b2b"},
	{4, 1, 51, "25234ad400200b2b2b2b2b"},
	{4, 2, 51, "29234ad400200b2b2b2b2b"},
	{4, 3, 51, "2d234ad400200b2b2b2b2b"},
	{5, 0, 55, "237234ad4002012b2b2b2b"},
	{5, 1, 55, "277234ad4002012b2b2b2b"},
	{5, 2, 55, "2b7234ad4002012b2b2b2b"},
	{5, 3, 55, "2f7234ad4002012b2b2b2b"},
	{6, 0, 72, "21234adffb7fe001002b2b"},
	{6, 1, 72, "25234adffb7fe001002b2b"},
	{6, 2, 72, "29234adffb7fe001002b2b"},
	{6, 3, 72, "2d234adffb7fe001002b2b"},
	{7, 0, 76, "237234adffb7fe00100b2b"},
	{7, 1, 76, "277234adffb7fe00100b2b"},
	{7, 2, 76, "2b7234adffb7fe00100b2b"},
	{7, 3, 76, "2f7234adffb7fe00100b2b"},
}

func ecRejectValue(mask, dl int) ECImmediateAssignmentRejectMessageContent {
	v := ECImmediateAssignmentRejectMessageContent{MessageType: 2, UsedDLCoverageClass: uint8(dl), ECRequestReference1: 0x1234, ECWaitTimer1: 173}
	if mask&1 != 0 {
		page := uint8(11)
		v.ECPageExtension = &page
	}
	if mask&2 != 0 {
		v.ECRequestReference2Group = &ECImmediateAssignmentRejectMessageContentECRequestReference2Group{ECRequestReference2: 0x1fed, ECWaitTimer2: 255}
	}
	if mask&4 != 0 {
		v.ECRequestReference3Group = &ECImmediateAssignmentRejectMessageContentECRequestReference3Group{ECRequestReference3: 1}
	}
	return v
}
func TestECImmediateAssignmentRejectSource(t *testing.T) {
	for _, tc := range ecRejectVectors {
		wire := ecRRWire(t, tc.wire)
		want := ecRejectValue(tc.mask, tc.dl)
		got, err := DecodeECImmediateAssignmentRejectMessageContent(wire)
		if err != nil || !runtime.SemanticallyEqual(got.Value, want) {
			t.Fatalf("mask %d dl %d: %#v %v", tc.mask, tc.dl, got, err)
		}
		ecRRCheck(t, "9.1.61", "EC Immediate Assignment Reject message content", wire, want, tc.content)
		ecRRBound(t, wire, tc.content, func(r *runtime.Reader) error {
			_, err := DecodeECImmediateAssignmentRejectMessageContentFrom(r)
			return err
		})
		got.Value.ECWaitTimer1 = 7
		edited, err := EncodeECImmediateAssignmentRejectMessageContent(got.Value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeECImmediateAssignmentRejectMessageContent(edited)
		if err != nil || again.Value.ECWaitTimer1 != 7 {
			t.Fatal("edit lost", err)
		}
	}
}
func TestECRejectInvalidFields(t *testing.T) {
	v := ecRejectValue(7, 3)
	for _, edit := range []func(*ECImmediateAssignmentRejectMessageContent){func(v *ECImmediateAssignmentRejectMessageContent) { v.MessageType = 1 }, func(v *ECImmediateAssignmentRejectMessageContent) { v.UsedDLCoverageClass = 4 }, func(v *ECImmediateAssignmentRejectMessageContent) { *v.ECPageExtension = 16 }, func(v *ECImmediateAssignmentRejectMessageContent) { v.ECRequestReference1 = 8192 }, func(v *ECImmediateAssignmentRejectMessageContent) {
		v.ECRequestReference2Group.ECRequestReference2 = 8192
	}, func(v *ECImmediateAssignmentRejectMessageContent) {
		v.ECRequestReference3Group.ECRequestReference3 = 8192
	}} {
		bad := ecRejectValue(7, 3)
		edit(&bad)
		if _, err := EncodeECImmediateAssignmentRejectMessageContentCanonicalAtLength(bad, 11); err == nil {
			t.Fatal("accepted invalid field")
		}
	}
	if _, err := EncodeECImmediateAssignmentRejectMessageContentCanonical(v); err == nil {
		t.Fatal("invented default extent")
	}
	for _, n := range []int{32, 128, 512} {
		wire := append(ecRRWire(t, ecRejectVectors[0].wire), bytes.Repeat([]byte{0xa5}, n)...)
		decoded, err := DecodeECImmediateAssignmentRejectMessageContent(wire)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := EncodeECImmediateAssignmentRejectMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(wire, replay) {
			t.Fatal("growing replay", err)
		}
	}
}
func FuzzECImmediateAssignmentRejectReplay(f *testing.F) {
	for _, tc := range ecRejectVectors {
		b, err := hex.DecodeString(tc.wire)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 256 {
			return
		}
		d, err := DecodeECImmediateAssignmentRejectMessageContent(wire)
		if err != nil {
			return
		}
		b, err := EncodeECImmediateAssignmentRejectMessageContent(d.Value)
		if err != nil || !bytes.Equal(b, wire) {
			t.Fatalf("replay %x %v", b, err)
		}
	})
}

func ecRRWire(t *testing.T, s string) []byte {
	t.Helper()
	wire, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return wire
}
func ecRRCheck(t *testing.T, clause, name string, wire []byte, want any, minimum int) {
	t.Helper()
	d, err := LookupClause(clause, name)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := d.Decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := d.Encode(decoded)
	if err != nil || !bytes.Equal(replay, wire) {
		t.Fatalf("replay %x: %v", replay, err)
	}
	canonical, err := d.CanonicalAtLength(decoded, len(wire))
	if err != nil || !bytes.Equal(canonical, wire) {
		t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
	}
	fresh, err := d.CanonicalAtLength(want, len(wire))
	if err != nil || !bytes.Equal(fresh, wire) {
		t.Fatalf("fresh %x want %x: %v", fresh, wire, err)
	}
	for tag := 0; tag < 16; tag++ {
		if tag == int(wire[0]>>4) {
			continue
		}
		bad := bytes.Clone(wire)
		bad[0] = bad[0]&15 | byte(tag<<4)
		if _, err := d.Decode(bad); err == nil {
			t.Fatalf("accepted message type %d", tag)
		}
	}
	for _, n := range []int{-1, 0, 12, 23, 1024, 1<<20 + 1} {
		if _, err := d.CanonicalAtLength(want, n); err == nil {
			t.Fatalf("accepted carrier extent %d", n)
		}
	}
	if minimum > 8 {
		if _, err := d.CanonicalAtLength(want, (minimum-1)/8); err == nil {
			t.Fatal("accepted extent below mandatory contents")
		}
	}
}
func ecRRBound(t *testing.T, wire []byte, minimum int, decode func(*runtime.Reader) error) {
	t.Helper()
	for cut := 0; cut < minimum; cut++ {
		r := runtime.NewReader(wire)
		if _, err := r.PushLimit(cut); err != nil {
			t.Fatal(err)
		}
		if err := decode(r); err == nil {
			t.Fatalf("accepted truncation %d", cut)
		}
		if r.Position() > cut {
			t.Fatal("crossed bound")
		}
	}
	r := runtime.NewReader(append(bytes.Clone(wire), 0xa5))
	old, err := r.PushLimit(len(wire) * 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := decode(r); err != nil {
		t.Fatal(err)
	}
	if r.Position() != len(wire)*8 {
		t.Fatalf("consumed %d", r.Position())
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	v, err := r.ReadUint(8)
	if err != nil || v != 0xa5 {
		t.Fatalf("sentinel %x %v", v, err)
	}
}
