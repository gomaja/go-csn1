package access_test

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
)

// TS 44.060 V19.0.0 table 11.2.5a.4 pp14181–14187 supplies
// prefix 100 and the first eight field bits. Table 11.2.5a.4.1
// pp14203–14210 supplies the remaining nineteen field bits.
func TestExtendedAccessBurst(t *testing.T) {
	v := access.Table11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2{
		RandomIDLow: 5, MSSyncAccuracy: 10, RandomIDHigh: 0xabc, ShortBSSID: 5, FinalMTAAccess: 1, Spare: 3,
	}
	word := uint32(0x45a)<<19 | 0xabc<<7 | 5<<4 | 1<<3 | 3
	input := extendedAccessWire(word)
	w := runtime.NewWriter()
	if err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2To(w, v); err != nil {
		t.Fatal(err)
	}
	if w.Position() != 30 {
		t.Fatal("wrong writer extent")
	}
	wire, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire.Bytes, input) {
		t.Fatal("printed continuation field order changed")
	}
	decoded, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(input)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.BitsConsumed != 30 {
		t.Fatal("wrong decoded extent")
	}
	got := decoded.Value
	if got.RandomIDLow != 5 || got.MSSyncAccuracy != 10 || got.RandomIDHigh != 0xabc || got.ShortBSSID != 5 || got.FinalMTAAccess != 1 || got.Spare != 3 {
		t.Fatal("typed fields changed")
	}
	replay, err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(got)
	if err != nil || !bytes.Equal(replay, input) {
		t.Fatal("replay changed", err)
	}
	canonical, err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2Canonical(got)
	if err != nil || !bytes.Equal(canonical, input) {
		t.Fatal("canonical changed", err)
	}
	got.RandomIDHigh = 0x123
	edited, err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(edited, extendedAccessWire(uint32(0x45a)<<19|0x123<<7|5<<4|1<<3|3)) {
		t.Fatal("continuation edit changed unrelated fields")
	}
	again, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(edited)
	if err != nil || again.Value.RandomIDHigh != 0x123 {
		t.Fatal("continuation edit lost", err)
	}
	for n := 0; n < 30; n++ {
		r, e := runtime.NewReaderBits(input, n, nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2From(r); e == nil {
			t.Fatalf("accepted truncation %d", n)
		}
	}
	r, err := runtime.NewReaderBits(input, 30, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2From(r); err != nil || r.Position() != 30 {
		t.Fatal("exact reader extent", err)
	}
	for bit := 0; bit < 3; bit++ {
		bad := append([]byte(nil), input...)
		bad[0] ^= 1 << uint(7-bit)
		if _, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(bad); err == nil {
			t.Fatal("accepted wrong literal prefix")
		}
	}
	name := "EGPRS MULTILATERATION REQUEST message content (Extended Access Burst Method – part 2)"
	context := `{"table":"11.2.5a.4","training_sequence":"4 (Note 1)","prefix":"100","suffix":""}`
	full, err := access.LookupContext("11.2.5a", name, context)
	if err != nil {
		t.Fatal(err)
	}
	body, err := access.LookupContext("11.2.5a", name, "")
	if err != nil {
		t.Fatal(err)
	}
	if full.Context != context || full.Extent.MinimumBits != 30 || full.Extent.MaximumBits != 30 || body.Context != "" || body.Extent.MinimumBits != 8 || body.Extent.MaximumBits != 8 {
		t.Fatal("full access replaced original body identity")
	}
	bodyValue, err := access.DecodeEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2([]byte{0x5a})
	if err != nil || bodyValue.BitsConsumed != 8 || bodyValue.Value.RandomIDLow != 5 || bodyValue.Value.MSSyncAccuracy != 10 {
		t.Fatal("original eight-bit typed API changed", err)
	}
	descriptorValue, err := full.Decode(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := descriptorValue.(runtime.Decoded[access.Table11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2]); !ok {
		t.Fatal("wrong full-message descriptor type")
	}
}

func extendedAccessWire(word uint32) []byte {
	word <<= 2
	return []byte{byte(word >> 24), byte(word >> 16), byte(word >> 8), byte(word)}
}

func FuzzExtendedAccessBurst(f *testing.F) {
	f.Add(extendedAccessWire(uint32(0x45a)<<19 | 0xabc<<7 | 5<<4 | 1<<3 | 3))
	f.Add(extendedAccessWire(uint32(4) << 27))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 8 {
			return
		}
		decoded, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(input)
		if err != nil {
			return
		}
		if decoded.BitsConsumed != 30 {
			t.Fatal("wrong fuzz extent")
		}
		replay, err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(decoded.Value)
		if err != nil || !bytes.Equal(replay, input) {
			t.Fatal("fuzz replay changed", err)
		}
		canonical, err := access.EncodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2Canonical(decoded.Value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := access.DecodeTable11_2_5a_4TrainingSequenceN4Note1PrefixN100SuffixNEGPRSMULTILATERATIONREQUESTMessageContentExtendedAccessBurstMethodPart2(canonical)
		if err != nil {
			t.Fatal(err)
		}
		a, b := runtime.Canonical(decoded.Value), runtime.Canonical(again.Value)
		if a.RandomIDLow != b.RandomIDLow || a.MSSyncAccuracy != b.MSSyncAccuracy || a.RandomIDHigh != b.RandomIDHigh || a.ShortBSSID != b.ShortBSSID || a.FinalMTAAccess != b.FinalMTAAccess || a.Spare != b.Spare {
			t.Fatal("canonical fuzz changed typed fields")
		}
	})
}
