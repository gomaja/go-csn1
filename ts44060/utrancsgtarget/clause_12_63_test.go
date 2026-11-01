package utrancsgtarget

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

var clause1263Codec = clauseCodec[UTRANCSGTargetCellIE]{DecodeUTRANCSGTargetCellIE, DecodeUTRANCSGTargetCellIEFrom, EncodeUTRANCSGTargetCellIE, EncodeUTRANCSGTargetCellIETo, EncodeUTRANCSGTargetCellIECanonicalAtLength}
var clause1263Vectors = []clauseVector[UTRANCSGTargetCellIE]{
	{name: "plmn-0-values-0-access-0", hex: "00000000", bits: 29, value: UTRANCSGTargetCellIE{UTRANCI: 0}},
	{name: "plmn-0-values-1-access-0", hex: "12345670", bits: 29, value: UTRANCSGTargetCellIE{UTRANCI: 19088743}},
	{name: "plmn-0-values-2-access-0", hex: "fffffff0", bits: 29, value: UTRANCSGTargetCellIE{UTRANCI: 268435455}},
	{name: "plmn-1-values-0-access-0", hex: "00000008000000", bits: 53, value: UTRANCSGTargetCellIE{UTRANCI: 0, PLMNID: &PLMNIDStruct{MCC: 0, MNC: 0}}},
	{name: "plmn-1-values-1-access-0", hex: "1234567891a2b0", bits: 53, value: UTRANCSGTargetCellIE{UTRANCI: 19088743, PLMNID: &PLMNIDStruct{MCC: 291, MNC: 1110}}},
	{name: "plmn-1-values-2-access-0", hex: "fffffffffffff8", bits: 53, value: UTRANCSGTargetCellIE{UTRANCI: 268435455, PLMNID: &PLMNIDStruct{MCC: 4095, MNC: 4095}}},
}

// TS 44.060 V19.0.0 §12.63 table 12.63.1, DOCX paragraphs 25589–25595.
// Synthetic field concatenations follow the printed widths and selectors.
func TestClause12_63(t *testing.T) {
	for _, vector := range clause1263Vectors {
		t.Run(vector.name, func(t *testing.T) {
			checkClauseVector(t, clause1263Codec, vector, vector.bits)
			d, err := clause1263Codec.decode(clauseBytes(t, vector.hex))
			if err != nil || d.BitsConsumed != vector.bits {
				t.Fatalf("consumed %d want %d: %v", d.BitsConsumed, vector.bits, err)
			}
		})
	}
	checkClauseInvalid(t, clause1263Codec, UTRANCSGTargetCellIE{UTRANCI: 1 << 28}, 32)
	checkClauseInvalid(t, clause1263Codec, UTRANCSGTargetCellIE{PLMNID: &PLMNIDStruct{MCC: 1 << 12}}, 32)
	checkClauseInvalid(t, clause1263Codec, UTRANCSGTargetCellIE{PLMNID: &PLMNIDStruct{MNC: 1 << 12}}, 32)
}

// An edit changes exactly the cited scalar bit and retains the received tail.
func TestClause12_63Edit(t *testing.T) {
	vector := clause1263Vectors[1]
	wire := clauseBytes(t, vector.hex)
	d, err := clause1263Codec.decode(wire)
	if err != nil {
		t.Fatal(err)
	}
	d.Value.UTRANCI ^= 1
	want := bytes.Clone(wire)
	want[27/8] ^= 1 << uint(7-27%8)
	got, err := clause1263Codec.encode(d.Value)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("edit %x want %x: %v", got, want, err)
	}
	fresh, err := clause1263Codec.decode(got)
	if err != nil || !runtime.SemanticallyEqual(fresh.Value, d.Value) {
		t.Fatalf("edited typed value lost: %v", err)
	}
}
func FuzzClause12_63Replay(f *testing.F) {
	for _, v := range clause1263Vectors {
		f.Add(clauseBytes(f, v.hex))
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 2048 {
			return
		}
		d, err := clause1263Codec.decode(wire)
		if err != nil {
			return
		}
		got, err := clause1263Codec.encode(d.Value)
		if err != nil || !bytes.Equal(got, wire) {
			t.Fatalf("replay %x: %v", got, err)
		}
	})
}
