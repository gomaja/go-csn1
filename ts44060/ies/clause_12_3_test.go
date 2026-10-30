package ies

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type clauseCodec[T any] struct {
	decode    func([]byte) (runtime.Decoded[T], error)
	from      func(*runtime.Reader) (T, error)
	encode    func(T) ([]byte, error)
	to        func(*runtime.Writer, T) error
	canonical func(T, int) ([]byte, error)
}

type clauseVector[T any] struct {
	name, hex string
	bits      int
	value     T
}

func clauseBytes(t testing.TB, text string) []byte {
	t.Helper()
	data, err := hex.DecodeString(text)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func clauseBitText(data []byte, bits int) string {
	var out strings.Builder
	for i := 0; i < bits; i++ {
		out.WriteByte('0' + ((data[i/8] >> uint(7-i%8)) & 1))
	}
	return out.String()
}

func clausePack(text string) []byte {
	out := make([]byte, (len(text)+7)/8)
	for i, c := range text {
		if c == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

// Every fixture is synthetic and comes from the cited printed grammar.
// Prefixes use a live enclosing bit limit, with nonzero data beyond it.
func checkClauseVector[T any](t *testing.T, codec clauseCodec[T], vector clauseVector[T], minimum int) {
	t.Helper()
	wire := clauseBytes(t, vector.hex)
	input, err := codec.decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := codec.encode(input.Value)
	if err != nil || !bytes.Equal(replay, wire) {
		t.Fatalf("byte replay %x: %v", replay, err)
	}
	fresh, err := codec.encode(runtime.Canonical(vector.value))
	if err != nil || !bytes.Equal(fresh, wire) {
		t.Fatalf("fresh typed encoding %x want %x: %v", fresh, wire, err)
	}
	canonical, err := codec.canonical(input.Value, len(wire))
	if err != nil || !bytes.Equal(canonical, wire) {
		t.Fatalf("canonical %x want %x: %v", canonical, wire, err)
	}
	again, err := codec.decode(canonical)
	if err != nil {
		t.Fatal(err)
	}
	stable, err := codec.canonical(again.Value, len(wire))
	if err != nil || !bytes.Equal(stable, canonical) {
		t.Fatalf("canonical stability %x: %v", stable, err)
	}
	if !runtime.SemanticallyEqual(input.Value, again.Value) {
		t.Fatal("canonical changed typed fields")
	}
	if input.BitsConsumed == vector.bits && vector.bits%8 != 0 {
		decorated := bytes.Clone(wire)
		decorated[len(decorated)-1] |= byte((1 << uint(8-vector.bits%8)) - 1)
		value, decodeErr := codec.decode(decorated)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		output, encodeErr := codec.encode(value.Value)
		if encodeErr != nil || !bytes.Equal(output, decorated) {
			t.Fatalf("nonzero tail replay %x: %v", output, encodeErr)
		}
		output, encodeErr = codec.canonical(value.Value, len(wire))
		if encodeErr != nil || !bytes.Equal(output, wire) {
			t.Fatalf("nonzero tail canonical %x: %v", output, encodeErr)
		}
	}
	payload := clauseBitText(wire, vector.bits)
	framed := clausePack("101" + payload + "10101")
	r := runtime.NewReader(framed)
	if _, err := r.ReadUint(3); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(vector.bits)
	if err != nil {
		t.Fatal(err)
	}
	value, err := codec.from(r)
	if err != nil {
		t.Fatal(err)
	}
	if r.Position() != 3+vector.bits || r.Remaining() != 0 {
		t.Fatalf("bounded position %d remaining %d", r.Position(), r.Remaining())
	}
	if !runtime.SemanticallyEqual(value, vector.value) {
		t.Fatalf("bounded typed fields differ: got %#v want %#v", runtime.Canonical(value), runtime.Canonical(vector.value))
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(5)
	if err != nil || sentinel != 21 {
		t.Fatalf("sentinel %d: %v", sentinel, err)
	}
	// Canonical encoding at the same bit offset preserves non-octet FLO extents.
	w := runtime.NewWriter()
	if err := w.WriteUint(5, 3); err != nil {
		t.Fatal(err)
	}
	bound, err := w.PushLimit(vector.bits)
	if err != nil {
		t.Fatal(err)
	}
	if err := codec.to(w, runtime.Canonical(value)); err != nil {
		t.Fatal(err)
	}
	if w.Position() != 3+vector.bits {
		t.Fatalf("bounded encoded position %d", w.Position())
	}
	if err := w.PopLimit(bound); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(21, 5); err != nil {
		t.Fatal(err)
	}
	output, err := w.Finish(runtime.BitString{})
	if err != nil || !bytes.Equal(output, framed) {
		t.Fatalf("bounded canonical %x want %x: %v", output, framed, err)
	}
	for n := 0; n < minimum; n++ {
		reader := runtime.NewReader(wire)
		if _, err := reader.PushLimit(n); err != nil {
			t.Fatal(err)
		}
		_, err := codec.from(reader)
		if err == nil {
			t.Fatalf("accepted strict prefix of %d bits", n)
		}
		var detail *runtime.DecodeError
		if !errors.As(err, &detail) {
			t.Fatalf("prefix %d error is not typed: %v", n, err)
		}
		if reader.Position() > n || detail.Offset > n {
			t.Fatalf("prefix %d crossed limit: position %d error offset %d", n, reader.Position(), detail.Offset)
		}
	}
}

func checkClauseInvalid[T any](t *testing.T, codec clauseCodec[T], value T, octets int) {
	t.Helper()
	if _, err := codec.encode(value); err == nil {
		t.Fatal("invalid typed field encoded")
	}
	if _, err := codec.canonical(value, octets); err == nil {
		t.Fatal("invalid typed field canonically encoded")
	}
}

var clause123Codec = clauseCodec[AckNackDescriptionIE]{DecodeAckNackDescriptionIE, DecodeAckNackDescriptionIEFrom, EncodeAckNackDescriptionIE, EncodeAckNackDescriptionIETo, EncodeAckNackDescriptionIECanonicalAtLength}
var clause123Vectors = []clauseVector[AckNackDescriptionIE]{
	{name: "ack-0-0", hex: "000000000000000000", bits: 72, value: AckNackDescriptionIE{FINALACKINDICATION: 0, STARTINGSEQUENCENUMBER: 0, RECEIVEDBLOCKBITMAP: 0}},
	{name: "ack-1-127", hex: "ffffffffffffffffff", bits: 72, value: AckNackDescriptionIE{FINALACKINDICATION: 1, STARTINGSEQUENCENUMBER: 127, RECEIVEDBLOCKBITMAP: 18446744073709551615}},
	{name: "ack-0-85", hex: "550123456789abcdef", bits: 72, value: AckNackDescriptionIE{FINALACKINDICATION: 0, STARTINGSEQUENCENUMBER: 85, RECEIVEDBLOCKBITMAP: 81985529216486895}},
	{name: "ack-1-1", hex: "818000000000000001", bits: 72, value: AckNackDescriptionIE{FINALACKINDICATION: 1, STARTINGSEQUENCENUMBER: 1, RECEIVEDBLOCKBITMAP: 9223372036854775809}},
}

// TS 44.060 V19.0.0 §12.3: exact printed field widths and all selectors.
func TestClause12_3(t *testing.T) {
	for _, vector := range clause123Vectors {
		t.Run(vector.name, func(t *testing.T) {
			minimum := vector.bits
			checkClauseVector(t, clause123Codec, vector, minimum)
		})
	}
	checkClauseInvalid(t, clause123Codec, AckNackDescriptionIE{FINALACKINDICATION: 2}, 9)
	checkClauseInvalid(t, clause123Codec, AckNackDescriptionIE{STARTINGSEQUENCENUMBER: 128}, 9)
}

func FuzzAcknowledgementAndECReplay(f *testing.F) {
	roots := []struct{ clause, name string }{
		{"12.3", "Ack/Nack Description IE"},
		{"12.3.2", "FLO Ack/Nack Description IE"},
		{"12.72", "EC Ack/Nack Description IE"},
		{"12.73", "EC Primary Ack/Nack Description IE"},
		{"12.74", "EC Packet Timing Advance IE"},
		{"12.76", "EC Channel Quality Report Higher CC IE"},
	}
	seeds := []string{"ff0123456789abcdef", "955800", "fd2ad0", "fd28", "a8", "fffe"}
	descriptors := make([]runtime.Descriptor, len(roots))
	for i, root := range roots {
		descriptor, err := LookupClause(root.clause, root.name)
		if err != nil {
			f.Fatal(err)
		}
		descriptors[i] = descriptor
		f.Add(uint8(i), clauseBytes(f, seeds[i]), uint16(0))
	}
	f.Fuzz(func(t *testing.T, index uint8, wire []byte, limit uint16) {
		if len(wire) > 512 {
			return
		}
		descriptor := descriptors[int(index)%len(descriptors)]
		value, err := descriptor.Decode(wire)
		if err == nil {
			output, encodeErr := descriptor.Encode(value)
			if encodeErr != nil || !bytes.Equal(output, wire) {
				t.Fatalf("replay %x: %v", output, encodeErr)
			}
		}
		reader := runtime.NewReader(wire)
		width := int(limit) % (len(wire)*8 + 1)
		if _, err := reader.PushLimit(width); err != nil {
			t.Fatal(err)
		}
		_, _ = descriptor.DecodeFrom(reader)
		if reader.Position() > width {
			t.Fatalf("bounded decode crossed %d bits", width)
		}
	})
}
