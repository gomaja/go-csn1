package ecimmediateassignment2

import (
	"bytes"
	"encoding/hex"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.60 pp11613-11637; the exact notation
// corrections carry authenticated receipts in the source intake.
// EC-CCCH/D: TS 45.003 V19.0.0 §5.2b.1, 88 information bits.
var ecAssignmentVectors = []struct {
	mask, mode, content int
	wire                string
}{
	{0, 0, 62, "1d2348acf59da33b2b2b2b"},
	{0, 1, 67, "1d2348acf59da33d8b2b2b"},
	{0, 2, 64, "1d2348acf59da33e2b2b2b"},
	{0, 3, 69, "1d2348acf59da33db32b2b"},
	{1, 0, 66, "1f72348acf59da33ab2b2b"},
	{1, 1, 71, "1f72348acf59da33d92b2b"},
	{1, 2, 68, "1f72348acf59da33eb2b2b"},
	{1, 3, 73, "1f72348acf59da33db2b2b"},
	{2, 0, 68, "1d2348acfdb6768ceb2b2b"},
	{2, 1, 73, "1d2348acfdb6768cf62b2b"},
	{2, 2, 70, "1d2348acfdb6768cfb2b2b"},
	{2, 3, 75, "1d2348acfdb6768cf6cb2b"},
	{3, 0, 72, "1f72348acfdb6768ce2b2b"},
	{3, 1, 77, "1f72348acfdb6768cf632b"},
	{3, 2, 74, "1f72348acfdb6768cfab2b"},
	{3, 3, 79, "1f72348acfdb6768cf6d2b"},
	{4, 0, 66, "1d2348acf59dea33ab2b2b"},
	{4, 1, 71, "1d2348acf59dea33d92b2b"},
	{4, 2, 68, "1d2348acf59dea33eb2b2b"},
	{4, 3, 73, "1d2348acf59dea33db2b2b"},
	{5, 0, 70, "1f72348acf59dea33b2b2b"},
	{5, 1, 75, "1f72348acf59dea33d8b2b"},
	{5, 2, 72, "1f72348acf59dea33e2b2b"},
	{5, 3, 77, "1f72348acf59dea33db32b"},
	{6, 0, 72, "1d2348acfdb677a8ce2b2b"},
	{6, 1, 77, "1d2348acfdb677a8cf632b"},
	{6, 2, 74, "1d2348acfdb677a8cfab2b"},
	{6, 3, 79, "1d2348acfdb677a8cf6d2b"},
	{7, 0, 76, "1f72348acfdb677a8ceb2b"},
	{7, 1, 81, "1f72348acfdb677a8cf62b"},
	{7, 2, 78, "1f72348acfdb677a8cfb2b"},
	{7, 3, 83, "1f72348acfdb677a8cf6cb"},
	{8, 0, 67, "1d2348acf59da3cdcb2b2b"},
	{8, 1, 72, "1d2348acf59da3cdec2b2b"},
	{8, 2, 69, "1d2348acf59da3cdf32b2b"},
	{8, 3, 74, "1d2348acf59da3cdedab2b"},
	{9, 0, 71, "1f72348acf59da3cdd2b2b"},
	{9, 1, 76, "1f72348acf59da3cdecb2b"},
	{9, 2, 73, "1f72348acf59da3cdf2b2b"},
	{9, 3, 78, "1f72348acf59da3cdedb2b"},
	{10, 0, 73, "1d2348acfdb6768f372b2b"},
	{10, 1, 78, "1d2348acfdb6768f37b32b"},
	{10, 2, 75, "1d2348acfdb6768f37cb2b"},
	{10, 3, 80, "1d2348acfdb6768f37b62b"},
	{11, 0, 77, "1f72348acfdb6768f3732b"},
	{11, 1, 82, "1f72348acfdb6768f37b2b"},
	{11, 2, 79, "1f72348acfdb6768f37d2b"},
	{11, 3, 84, "1f72348acfdb6768f37b6b"},
	{12, 0, 71, "1d2348acf59dea3cdd2b2b"},
	{12, 1, 76, "1d2348acf59dea3cdecb2b"},
	{12, 2, 73, "1d2348acf59dea3cdf2b2b"},
	{12, 3, 78, "1d2348acf59dea3cdedb2b"},
	{13, 0, 75, "1f72348acf59dea3cdcb2b"},
	{13, 1, 80, "1f72348acf59dea3cdec2b"},
	{13, 2, 77, "1f72348acf59dea3cdf32b"},
	{13, 3, 82, "1f72348acf59dea3cdedab"},
	{14, 0, 77, "1d2348acfdb677a8f3732b"},
	{14, 1, 82, "1d2348acfdb677a8f37b2b"},
	{14, 2, 79, "1d2348acfdb677a8f37d2b"},
	{14, 3, 84, "1d2348acfdb677a8f37b6b"},
	{15, 0, 81, "1f72348acfdb677a8f372b"},
	{15, 1, 86, "1f72348acfdb677a8f37b3"},
	{15, 2, 83, "1f72348acfdb677a8f37cb"},
	{15, 3, 88, "1f72348acfdb677a8f37b6"},
}

func ecAssignmentValue(mask, mode int) ECImmediateAssignmentType2MessageContent {
	v := ECImmediateAssignmentType2MessageContent{MessageType: 1, UsedDLCoverageClass: 3, ECRequestReference: 0x1234, ECPacketChannelDescriptionType2: 0x4567, ECFixedUplinkAllocation: ECFixedUplinkAllocationStruct{EnhancedAccessBurst: 1, STARTINGULTIMESLOT: 5, UplinkTFIAssignment: 19, STARTINGDLTIMESLOTOFFSET: 2, OVERLAIDCDMACODE: 3, GAMMA: 17, ALPHAEnable: 1, StartFirstULDataBlock: 7}}
	if mask&1 != 0 {
		p := uint8(11)
		v.ECPageExtension = &p
	}
	if mask&2 != 0 {
		p := uint8(45)
		v.ECFixedUplinkAllocation.TimingAdvance = &p
	}
	if mask&4 != 0 {
		p := uint8(10)
		v.ECFixedUplinkAllocation.AssignedMCS = &p
	}
	if mask&8 != 0 {
		v.ECFixedUplinkAllocation.P0Group = &ECFixedUplinkAllocationStructP0Group{P0: 9, PRMODE: 1}
	}
	if mode == 1 || mode == 3 {
		v.ECFixedUplinkAllocation.StartFNNextDataBlockList = append(v.ECFixedUplinkAllocation.StartFNNextDataBlockList, ECFixedUplinkAllocationStructStartFNNextDataBlockListEntry{StartFNNextDataBlockChoice: ECFixedUplinkAllocationStructStartFNNextDataBlockListEntryStartFNNextDataBlockChoice{Alternative: ECFixedUplinkAllocationStructStartFNNextDataBlockListEntryStartFNNextDataBlockChoiceAlternativeStartFNNextDataBlock, StartFNNextDataBlock: &ECFixedUplinkAllocationStructStartFNNextDataBlockListEntryStartFNNextDataBlockChoiceStartFNNextDataBlock{StartFNNextDataBlock: 6}}})
	}
	if mode == 2 || mode == 3 {
		v.ECFixedUplinkAllocation.StartFNNextDataBlockList = append(v.ECFixedUplinkAllocation.StartFNNextDataBlockList, ECFixedUplinkAllocationStructStartFNNextDataBlockListEntry{StartFNNextDataBlockChoice: ECFixedUplinkAllocationStructStartFNNextDataBlockListEntryStartFNNextDataBlockChoice{Alternative: ECFixedUplinkAllocationStructStartFNNextDataBlockListEntryStartFNNextDataBlockChoiceAlternativeAlt1, Alt1: &struct{}{}}})
	}
	return v
}
func TestECImmediateAssignmentType2Source(t *testing.T) {
	for _, tc := range ecAssignmentVectors {
		wire := ecRRWire(t, tc.wire)
		want := ecAssignmentValue(tc.mask, tc.mode)
		got, err := DecodeECImmediateAssignmentType2MessageContent(wire)
		if err != nil || !runtime.SemanticallyEqual(got.Value, want) {
			t.Fatalf("mask %d mode %d: %#v %v", tc.mask, tc.mode, got, err)
		}
		ecRRCheck(t, "9.1.60", "EC Immediate Assignment Type 2 message content", wire, want, tc.content)
		ecRRBound(t, wire, tc.content, func(r *runtime.Reader) error {
			_, err := DecodeECImmediateAssignmentType2MessageContentFrom(r)
			return err
		})
		got.Value.ECFixedUplinkAllocation.GAMMA = 3
		edited, err := EncodeECImmediateAssignmentType2MessageContent(got.Value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeECImmediateAssignmentType2MessageContent(edited)
		if err != nil || again.Value.ECFixedUplinkAllocation.GAMMA != 3 {
			t.Fatal("nested edit lost", err)
		}
	}
}
func TestECImmediateAssignmentType2Boundaries(t *testing.T) {
	for _, edit := range []func(*ECImmediateAssignmentType2MessageContent){func(v *ECImmediateAssignmentType2MessageContent) { v.MessageType = 2 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECRequestReference = 8192 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECPacketChannelDescriptionType2 = 32768 }, func(v *ECImmediateAssignmentType2MessageContent) { *v.ECFixedUplinkAllocation.TimingAdvance = 64 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECFixedUplinkAllocation.STARTINGULTIMESLOT = 8 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECFixedUplinkAllocation.UplinkTFIAssignment = 32 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECFixedUplinkAllocation.OVERLAIDCDMACODE = 4 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECFixedUplinkAllocation.GAMMA = 32 }, func(v *ECImmediateAssignmentType2MessageContent) { v.ECFixedUplinkAllocation.P0Group.PRMODE = 2 }, func(v *ECImmediateAssignmentType2MessageContent) { *v.ECFixedUplinkAllocation.AssignedMCS = 16 }} {
		v := ecAssignmentValue(15, 0)
		edit(&v)
		if _, err := EncodeECImmediateAssignmentType2MessageContentCanonicalAtLength(v, 11); err == nil {
			t.Fatal("accepted invalid field")
		}
	}
	for _, n := range []int{0, 1, 32, 128} {
		v := ecAssignmentValue(0, 2)
		entry := v.ECFixedUplinkAllocation.StartFNNextDataBlockList[0]
		v.ECFixedUplinkAllocation.StartFNNextDataBlockList = make([]ECFixedUplinkAllocationStructStartFNNextDataBlockListEntry, n)
		for i := range v.ECFixedUplinkAllocation.StartFNNextDataBlockList {
			v.ECFixedUplinkAllocation.StartFNNextDataBlockList[i] = entry
		}
		wire, err := EncodeECImmediateAssignmentType2MessageContent(v)
		if err != nil {
			t.Fatal(err)
		}
		if len(wire) > (63+2*n+7)/8 {
			t.Fatal("nonlinear list wire size")
		}
		got, err := DecodeECImmediateAssignmentType2MessageContent(wire)
		if err != nil || len(got.Value.ECFixedUplinkAllocation.StartFNNextDataBlockList) != n {
			t.Fatal("growing list", err)
		}
		if n > 1 {
			if _, err := EncodeECImmediateAssignmentType2MessageContentCanonicalAtLength(v, 11); err == nil {
				t.Fatal("oversized list fits carrier")
			}
		}
	}
}
func FuzzECImmediateAssignmentType2Replay(f *testing.F) {
	for _, tc := range ecAssignmentVectors {
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
		helper, helperErr := DecodeECFixedUplinkAllocationStruct(wire)
		if helperErr == nil {
			encoded, err := EncodeECFixedUplinkAllocationStruct(helper.Value)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("helper replay %x: %v", encoded, err)
			}
		}
		d, err := DecodeECImmediateAssignmentType2MessageContent(wire)
		if err != nil {
			return
		}
		b, err := EncodeECImmediateAssignmentType2MessageContent(d.Value)
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
