package assignmentmessages_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	prr "github.com/gomaja/go-csn1/ts44060/assignmentmessages"
)

type requestVector struct {
	Name      string `json:"name"`
	Bits      string `json:"bits"`
	Address   int    `json:"address"`
	Release   int    `json:"release"`
	Mask      int    `json:"mask"`
	Remainder bool   `json:"remainder"`
}

func requestBytes(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

// TS 44.060 V19.0.0 §11.2.16 pp16479–16527. The imported
// fields follow §§12.7, 12.7a, 12.7b, 12.5.3–12.5.4, 12.5a.2–12.5a.3.
// This builds printed fields directly, independently of the generated encoder.
func requestBits(address, release, mask int, remainder bool) string {
	var b strings.Builder
	field := func(v, n int) { fmt.Fprintf(&b, "%0*b", n, v) }
	optional := func(on bool, value string) {
		if on {
			b.WriteByte('1')
			b.WriteString(value)
		} else {
			b.WriteByte('0')
		}
	}
	set := func(n int) bool { return mask&(1<<uint(n)) != 0 }
	optional(set(0), "10")
	if address == 2 {
		b.WriteByte('1')
		field(0x10203040, 32)
	} else {
		b.WriteByte('0')
		field(address, 1)
		field(21, 5)
	}
	// §12.30 omits outer spares; 4+7+73+1 = 85 bits per ordinary entry.
	optional(set(1), "0001"+fmt.Sprintf("%07b", 73)+"001"+strings.Repeat("0", 70)+"0")
	b.WriteString("01011010")
	field(0x1234, 16)
	optional(set(2), "11")
	field(31, 6)
	optional(set(3), "101010")
	for i := 0; i < 8; i++ {
		optional(set(4+i), fmt.Sprintf("%04b", i+1))
	}
	quality := func(count int) {
		for i := 0; i < count; i++ {
			optional(set(2), fmt.Sprintf("%08b", 0x51+i))
		}
	}
	timeslots := func() {
		optional(set(3), strings.Repeat("0", 8))
		optional(set(4), strings.Repeat("0", 8))
	}
	for level := 0; level < 6; level++ {
		if level == release {
			if remainder {
				b.WriteString("0101")
			}
			return b.String()
		}
		b.WriteByte('1')
		switch level {
		case 0:
			if set(0) {
				b.WriteByte('1')
				quality(2)
			} else {
				b.WriteByte('0')
			}
			if set(1) {
				b.WriteByte('1')
				timeslots()
			} else {
				b.WriteByte('0')
			}
			optional(set(2), "1010101")
			b.WriteString("10")
		case 1:
			if set(0) {
				b.WriteByte('1')
				optional(set(1), "1010")
				b.WriteString("1000110")
				optional(set(2), "10100101")
				b.WriteByte('0')
			} else {
				b.WriteByte('0')
			}
			optional(set(3), "1")
		case 2:
			if set(0) {
				b.WriteByte('1')
				b.WriteString("1010101101")
				optional(set(1), "1")
				b.WriteByte('0')
			} else {
				b.WriteByte('0')
			}
		case 3:
			b.WriteByte('1')
			if set(0) {
				b.WriteByte('1')
				quality(7)
			} else {
				b.WriteByte('0')
			}
			if set(1) {
				b.WriteByte('1')
				timeslots()
			} else {
				b.WriteByte('0')
			}
		case 4:
			b.WriteByte('1')
		case 5:
			optional(set(0), "101")
			// §11 p12938: literal zero precedes the L padding sequence.
			b.WriteByte('0')
			// L is evaluated at the block coordinate, starting at bit 14.
			for (14+b.Len())%8 != 0 {
				field((0x2b>>uint(7-(14+b.Len())%8))&1, 1)
			}
		}
	}
	return b.String()
}

func requestVectors() []requestVector {
	masks := []int{0, 4095}
	for bit := 0; bit < 12; bit++ {
		masks = append(masks, 1<<uint(bit))
	}
	var out []requestVector
	for address := 0; address < 3; address++ {
		for release := 0; release <= 6; release++ {
			for _, mask := range masks {
				for _, remainder := range []bool{false, true} {
					if release == 6 && remainder {
						continue
					}
					out = append(out, requestVector{fmt.Sprintf("address%d-release%d-mask%d-remainder%t", address, release, mask, remainder), requestBits(address, release, mask, remainder), address, release, mask, remainder})
				}
			}
		}
	}
	return out
}

func requestRead(t testing.TB, bits string, position int) (prr.PacketResourceRequestMessageContent, runtime.WireInfo) {
	t.Helper()
	r, err := runtime.NewChoiceReaderBits(requestBytes(strings.Repeat("1", position)+bits+"101"), position+len(bits)+3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(position); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(len(bits))
	if err != nil {
		t.Fatal(err)
	}
	v, err := prr.DecodePacketResourceRequestMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position() != position+len(bits) {
		t.Fatalf("extent %d, want %d", r.Position(), position+len(bits))
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: requestBytes(strings.Repeat("1", position) + bits), BitLength: position + len(bits)}, position+len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	if sentinel, e := r.ReadUint(3); e != nil || sentinel != 5 {
		t.Fatalf("sentinel %d: %v", sentinel, e)
	}
	return v, wire
}

func requestWrite(t testing.TB, v prr.PacketResourceRequestMessageContent, wire runtime.WireInfo, position int) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if err := w.WriteLiteral(strings.Repeat("1", position)); err != nil {
		t.Fatal(err)
	}
	if err := prr.EncodePacketResourceRequestMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestClause11216PrintedBranches(t *testing.T) {
	vectors := requestVectors()
	for _, tc := range vectors {
		t.Run(tc.Name, func(t *testing.T) {
			for _, position := range []int{6, 14, 22, 30} {
				v, wire := requestRead(t, tc.Bits, position)
				if v.CVALUE != 31 || v.ChannelRequestDescription.RLCOCTETCOUNT != 0x1234 || v.ChannelRequestDescription.PEAKTHROUGHPUTCLASS != 5 || v.ChannelRequestDescription.RADIOPRIORITY != 2 || v.ChannelRequestDescription.RLCMODE != 1 || v.ChannelRequestDescription.LLCPDUTYPE != 0 {
					t.Fatal("typed mandatory fields differ")
				}
				optional := []*uint8{v.ACCESSTYPE, v.CHANGEMARK, v.SIGNVAR, v.ILEVELTN0, v.ILEVELTN1, v.ILEVELTN2, v.ILEVELTN3, v.ILEVELTN4, v.ILEVELTN5, v.ILEVELTN6, v.ILEVELTN7}
				optionalWant := []uint8{2, 3, 42, 1, 2, 3, 4, 5, 6, 7, 8}
				for i, field := range optional {
					bit := i + 1
					if i == 0 {
						bit = 0
					}
					want := optionalWant[i]
					if (field != nil) != (tc.Mask&(1<<uint(bit)) != 0) || field != nil && *field != want {
						t.Fatalf("optional field %d differs", i)
					}
				}
				if tc.Address == 2 {
					if v.GlobalTFIChoice.TLLIGRNTI == nil || v.GlobalTFIChoice.TLLIGRNTI.TLLIGRNTI != 0x10203040 {
						t.Fatal("TLLI choice differs")
					}
				} else if v.GlobalTFIChoice.GlobalTFI == nil {
					t.Fatal("Global TFI choice lost")
				}
				if (v.MSRadioAccessCapability2 != nil) != (tc.Mask&2 != 0) {
					t.Fatal("capability presence differs")
				}
				if v.MSRadioAccessCapability2 != nil {
					entry := v.MSRadioAccessCapability2.MSRACapability.Entries
					if len(entry) != 1 || entry[0].AccessTechnologyTypeChoice.AccessCapabilities == nil || entry[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessTechnologyType != 1 {
						t.Fatal("typed §12.30 capability lost")
					}
				}
				out := requestWrite(t, v, wire, position)
				want := strings.Repeat("1", position) + tc.Bits
				if out.BitLength != len(want) || !bytes.Equal(out.Bytes, requestBytes(want)) {
					t.Fatal("exact replay differs")
				}
				v.CVALUE = 17
				edited := requestWrite(t, v, wire, position)
				r, err := runtime.NewChoiceReaderBits(edited.Bytes, edited.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = r.ReadUint(position); err != nil {
					t.Fatal(err)
				}
				again, err := prr.DecodePacketResourceRequestMessageContentFrom(r)
				if err != nil || again.CVALUE != 17 {
					t.Fatalf("typed edit: %v", err)
				}
				fresh := runtime.Canonical(v)
				freshWire := requestWrite(t, fresh, runtime.WireInfo{}, position)
				fr, err := runtime.NewChoiceReaderBits(freshWire.Bytes, freshWire.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = fr.ReadUint(position); err != nil {
					t.Fatal(err)
				}
				fv, err := prr.DecodePacketResourceRequestMessageContentFrom(fr)
				if err != nil || !runtime.SemanticallyEqual(fv, fresh) {
					t.Fatalf("fresh fields: %v", err)
				}
			}
			// Byte helpers include octet completion, independently of caller limits.
			raw := requestBytes(tc.Bits)
			d, err := prr.DecodePacketResourceRequestMessageContent(raw)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := prr.EncodePacketResourceRequestMessageContent(d.Value)
			if err != nil || !bytes.Equal(encoded, raw) {
				t.Fatalf("byte replay: %v", err)
			}
			if tc.Release == 6 {
				canonical, err := prr.EncodePacketResourceRequestMessageContentCanonicalAtLength(d.Value, len(raw))
				if err != nil {
					t.Fatal(err)
				}
				cd, err := prr.DecodePacketResourceRequestMessageContent(canonical)
				if err != nil || !runtime.SemanticallyEqual(cd.Value, d.Value) {
					t.Fatalf("canonical fields: %v", err)
				}
			}
		})
	}
	if file := os.Getenv("CSN1_SYNTHETIC_REQUEST_VECTORS"); file != "" {
		data, err := json.Marshal(vectors)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(file, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("vectors=%d bounded_phase_cases=%d", len(vectors), 4*len(vectors))
}

func TestClause11216BoundsAndInvalidFields(t *testing.T) {
	bits := requestBits(2, 6, 4095, false)
	mandatory := len(requestBits(2, 0, 4095, false))
	for cut := 0; cut < len(bits); cut++ {
		r, err := runtime.NewChoiceReaderBits(requestBytes(strings.Repeat("0", 14)+bits), 14+cut, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(14); err != nil {
			t.Fatal(err)
		}
		v, e := prr.DecodePacketResourceRequestMessageContentFrom(r)
		if cut < mandatory && e == nil {
			t.Fatalf("accepted mandatory fields truncated at %d of %d bits", cut, mandatory)
		}
		// Printed null alternatives intentionally accept release boundaries.
		if e == nil {
			_ = requestWrite(t, v, r.Wire(), 14)
		}
	}
	v, _ := requestRead(t, bits, 14)
	v.CVALUE = 64
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := prr.EncodePacketResourceRequestMessageContentTo(w, v); err == nil {
		t.Fatal("accepted overflowing C_VALUE")
	}
	v, _ = requestRead(t, bits, 14)
	v.GlobalTFIChoice.GlobalTFI = &prr.PacketResourceRequestMessageContentGlobalTFIChoiceGlobalTFI{}
	w = runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := prr.EncodePacketResourceRequestMessageContentTo(w, v); err == nil {
		t.Fatal("accepted two address arms")
	}
	for phase := 0; phase < 8; phase++ {
		if phase == 6 {
			continue
		}
		r, err := runtime.NewChoiceReaderBits(requestBytes(bits), len(bits), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		_, err = prr.DecodePacketResourceRequestMessageContentFrom(r)
		var located *prr.ContentPhaseError
		if !errors.As(err, &located) || r.Position() != phase {
			t.Fatal("wrong-phase reader consumed input")
		}
	}
	_, err := prr.EncodePacketResourceRequestMessageContentCanonical(v)
	var extent *runtime.ExtentError
	if !errors.As(err, &extent) {
		t.Fatalf("missing caller extent: %v", err)
	}
}

func TestClause11216CapabilitySenderKeepsReplay(t *testing.T) {
	base := requestBits(0, 0, 2, false)
	ordinary := "0001" + fmt.Sprintf("%07b", 73) + "001" + strings.Repeat("0", 70) + "0"
	received := "0001" + fmt.Sprintf("%07b", 77) + "001" + strings.Repeat("0", 70) + "11110"
	// A null release boundary avoids unrelated padding when checking the import.
	bits := strings.Replace(base, ordinary, received, 1)
	v, wire := requestRead(t, bits, 14)
	before := runtime.Canonical(v)
	replay := requestWrite(t, v, wire, 14)
	if replay.BitLength != 14+len(bits) || !bytes.Equal(replay.Bytes, requestBytes(strings.Repeat("1", 14)+bits)) {
		t.Fatal("received nested spares lost")
	}
	fresh := requestWrite(t, runtime.Canonical(v), runtime.WireInfo{}, 14)
	if fresh.BitLength != 14+len(base) || !bytes.Equal(fresh.Bytes, requestBytes(strings.Repeat("1", 14)+base)) {
		t.Fatal("PRR sender bound to ordinary capability spare policy")
	}
	if !reflect.DeepEqual(before, runtime.Canonical(v)) {
		t.Fatal("sender mutated received capability")
	}
	// TS 24.008 V20.1.0 §10.5.5.12a: discriminator 1111 uses the
	// second seven-bit envelope around a 10-bit-per-entry technology list.
	for _, count := range []int{0, 1, 12} {
		list := strings.Repeat("1"+"0010"+"011"+"01", count) + "0"
		prefix := strings.TrimSuffix(ordinary, "0") + "1" + "1111"
		receivedAdditional := prefix + fmt.Sprintf("%07b", len(list)+3) + list + "111" + "0"
		sentAdditional := prefix + fmt.Sprintf("%07b", len(list)) + list + "0"
		receivedBody := strings.Replace(base, ordinary, receivedAdditional, 1)
		sentBody := strings.Replace(base, ordinary, sentAdditional, 1)
		value, receipt := requestRead(t, receivedBody, 14)
		entries := value.MSRadioAccessCapability2.MSRACapability.Entries
		if len(entries) != 2 || entries[1].AccessTechnologyTypeChoice.Length == nil {
			t.Fatal("additional technology arm lost")
		}
		replayed := requestWrite(t, value, receipt, 14)
		if replayed.BitLength != 14+len(receivedBody) || !bytes.Equal(replayed.Bytes, requestBytes(strings.Repeat("1", 14)+receivedBody)) {
			t.Fatal("additional technology replay differs")
		}
		sent := requestWrite(t, runtime.Canonical(value), runtime.WireInfo{}, 14)
		if sent.BitLength != 14+len(sentBody) || !bytes.Equal(sent.Bytes, requestBytes(strings.Repeat("1", 14)+sentBody)) {
			t.Fatal("additional technology sender kept spare bits")
		}
	}
}

func TestClause11216GrowingCapability(t *testing.T) {
	base := requestBits(0, 0, 2, false)
	entry := "0001" + fmt.Sprintf("%07b", 73) + "001" + strings.Repeat("0", 70)
	var previous float64
	for _, count := range []int{4, 16, 64, 256} {
		capability := strings.Repeat(entry+"1", count-1) + entry + "0"
		bits := strings.Replace(base, entry+"0", capability, 1)
		data := requestBytes(bits)
		allocations := testing.AllocsPerRun(1, func() {
			d, err := prr.DecodePacketResourceRequestMessageContent(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Value.MSRadioAccessCapability2.MSRACapability.Entries) != count {
				t.Fatal("growing capability lost entries")
			}
		})
		if !raceEnabled && previous > 0 && allocations > previous*6+1024 {
			t.Fatalf("nonlinear allocations: %.0f after %.0f", allocations, previous)
		}
		previous = allocations
		t.Logf("entries=%d octets=%d allocations=%.0f", count, len(data), allocations)
	}
}

// TS 44.060 V19.0.0 §11 p12938: the printed Ignore alternative
// is received data, with no fresh or canonical sender representation.
func TestClause11216IgnoredPadding(t *testing.T) {
	base := requestBits(0, 6, 0, false)
	// The minimal Rel-12 request has no downlink eTFI; its padding selector
	// is at offset 49+6+3+2+4+2+2 = 68 from the content start.
	const paddingStart = 68
	if base[paddingStart] != '0' {
		t.Fatal("padding calculation differs from the clause")
	}
	bits := base[:paddingStart] + "110101"
	v, wire := requestRead(t, bits, 14)
	out := requestWrite(t, v, wire, 14)
	if out.BitLength != 14+len(bits) || !bytes.Equal(out.Bytes, requestBytes(strings.Repeat("1", 14)+bits)) {
		t.Fatal("Ignore padding replay differs")
	}
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := prr.EncodePacketResourceRequestMessageContentTo(w, runtime.Canonical(v)); err == nil {
		t.Fatal("canonical sender accepted received Ignore padding")
	}
}

func FuzzPacketResourceRequest(f *testing.F) {
	for _, mask := range []int{0, 2, 4095} {
		f.Add(requestBytes(requestBits(0, 6, mask, false)))
	}
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1024 {
			return
		}
		d, err := prr.DecodePacketResourceRequestMessageContent(data)
		if err != nil {
			return
		}
		out, err := prr.EncodePacketResourceRequestMessageContent(d.Value)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("accepted request cannot replay: %v", err)
		}
	})
}
