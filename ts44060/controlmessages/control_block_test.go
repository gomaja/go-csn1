package controlmessages_test

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"reflect"
	"testing"
)

// TS 44.060 V19.0.0 §10.0b.2.1 paragraphs 7388–7389 and
// §10.3.1.1 figure 10.3.1.1.1 paragraphs 7665–7730 define the
// 184-bit CS-1 block. §10.3.2 paragraphs 7887–7957 defines the uplink
// MAC octet; §11.2.0.1/§11.2.0.2 supplies the six-bit message selector.
// Content starts at bit 14 on the same reader, after the production header.
func simpleBlock(t testing.TB, selector uint64, body string) []byte {
	t.Helper()
	bits := fmt.Sprintf("01000000%06b", selector) + body
	if len(bits) > 184 {
		t.Fatal("prefix exceeds CS-1 block")
	}
	if len(bits) < 184 {
		bits += "0"
	}
	for len(bits) < 184 {
		if (byte(0x2b)>>uint(7-len(bits)%8))&1 != 0 {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	b := make([]byte, 23)
	for i, c := range bits {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func simpleReader(t testing.TB, b []byte, selector uint64, uplink bool) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(b, 184, c)
	if err != nil {
		t.Fatal(err)
	}
	simpleHeader(t, r, selector, uplink)
	return r, c
}
func simpleHeader(t testing.TB, r *runtime.Reader, selector uint64, uplink bool) {
	t.Helper()
	direction := runtime.RLCMACDownlink
	if uplink {
		direction = runtime.RLCMACUplink
	}
	ctx := runtime.RLCMACContext{Direction: direction, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if !uplink && (selector == 36 || selector == 6) {
		ctx.Channel = runtime.RLCMACPCCCH
	}
	if _, err := controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	tag, err := r.ReadUint(6)
	if err != nil || tag != selector || r.Position() != 14 {
		t.Fatalf("selector=%d origin=%d: %v", tag, r.Position(), err)
	}
}

func simpleField(t testing.TB, root any, name string) reflect.Value {
	t.Helper()
	var found []reflect.Value
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if v.Kind() == reflect.Pointer {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		if v.Kind() != reflect.Struct {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if f.Name == "Wire" {
				continue
			}
			x := v.Field(i)
			typ := x.Type()
			if typ.Kind() == reflect.Pointer {
				typ = typ.Elem()
			}
			scalar := typ.Kind() >= reflect.Uint && typ.Kind() <= reflect.Uint64
			list := typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8
			if f.Name == name && (scalar || list) {
				found = append(found, x)
			} else {
				walk(x)
			}
		}
	}
	walk(reflect.ValueOf(root))
	if len(found) != 1 {
		t.Fatalf("field %s has %d occurrences in %T", name, len(found), root)
	}
	return found[0]
}
func simpleUint(t testing.TB, root any, name string, want uint64) {
	t.Helper()
	v := simpleField(t, root, name)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			t.Fatalf("field %s absent", name)
		}
		v = v.Elem()
	}
	if v.Uint() != want {
		t.Fatalf("field %s=%d want %d", name, v.Uint(), want)
	}
}
func simpleEncoded[T any](t testing.TB, b []byte, v T, encode func(*runtime.Writer, T) error, wire runtime.WireInfo, canonical bool) []byte {
	t.Helper()
	w := runtime.NewWriter()
	if canonical {
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
		v = runtime.Canonical(v)
	} else {
		w.WithWire(wire)
	}
	if _, err := w.PushLimit(184); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(uint64(b[0]), 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(uint64(b[1]>>2), 6); err != nil {
		t.Fatal(err)
	}
	if err := encode(w, v); err != nil {
		t.Fatal(err)
	}
	result, err := w.FinishBits(runtime.BitString{})
	if err != nil || result.BitLength != 184 {
		t.Fatalf("encode extent %d: %v", result.BitLength, err)
	}
	return result.Bytes
}
func simpleExercise[T any](t testing.TB, selector uint64, uplink bool, body string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, check func(testing.TB, T), editField string) {
	t.Helper()
	b := simpleBlock(t, selector, body)
	r, c := simpleReader(t, b, selector, uplink)
	v, err := decode(r)
	if err != nil {
		t.Fatal(err)
	}
	check(t, v)
	if r.Position() != 184 {
		t.Fatalf("content end %d", r.Position())
	}
	if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
		t.Fatalf("unexpected region %+v", report)
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: b, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if got := simpleEncoded(t, b, v, encode, wire, false); !bytes.Equal(got, b) {
		t.Fatalf("replay %x want %x", got, b)
	}
	if got := simpleEncoded(t, b, v, encode, runtime.WireInfo{}, true); !bytes.Equal(got, b) {
		t.Fatalf("canonical %x want %x", got, b)
	}
	f := simpleField(t, &v, editField)
	old := f.Uint()
	next := uint64(0)
	if old == 0 {
		next = 1
	}
	f.SetUint(next)
	edited := simpleEncoded(t, b, v, encode, runtime.WireInfo{}, true)
	er, _ := simpleReader(t, edited, selector, uplink)
	ev, err := decode(er)
	if err != nil {
		t.Fatal(err)
	}
	simpleUint(t, ev, editField, next)
	if !reflect.DeepEqual(runtime.Canonical(v), runtime.Canonical(ev)) {
		t.Fatal("typed edit changed another semantic field")
	}
	// A containing limit protects a following three-bit sentinel.
	raw := append(append([]byte(nil), b...), 0xa0)
	sr, err := runtime.NewChoiceReaderBits(raw, 187, nil)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := sr.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	simpleHeader(t, sr, selector, uplink)
	if _, err = decode(sr); err != nil {
		t.Fatal(err)
	}
	if err = sr.PopLimit(previous); err != nil {
		t.Fatal(err)
	}
	sentinel, err := sr.ReadUint(3)
	if err != nil || sentinel != 5 {
		t.Fatalf("sentinel %d: %v", sentinel, err)
	}
}
func simpleFuzz[T any](f *testing.F, selector uint64, uplink bool, body string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error) {
	f.Add(simpleBlock(f, selector, body))
	f.Add(make([]byte, 23))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 23 {
			return
		}
		data = bytes.Clone(data)
		data[0] = 0x40
		data[1] = (data[1] & 3) | byte(selector<<2)
		r, _ := simpleReader(t, data, selector, uplink)
		v, err := decode(r)
		if err != nil {
			return
		}
		if r.Position() > 184 {
			t.Fatal("crossed containing bound")
		}
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, r.Position(), runtime.BitString{}, r.Wire())
		if err != nil {
			return
		}
		got := simpleEncoded(t, data, v, encode, wire, false)
		if !bytes.Equal(got, data) {
			t.Fatalf("replay differs: %x %x", got, data)
		}
	})
}

// TS 44.060 V19.0.0 §11.2.11: complete synthetic CS-1 control blocks.

func simpleTruncation[T any](t testing.TB, selector uint64, uplink bool, body string, bits int, label runtime.ErrorLabel, offset int, decode func(*runtime.Reader) (T, error), check func(testing.TB, T)) {
	t.Helper()
	raw := simpleBlock(t, selector, body)
	r, c := simpleReader(t, raw, selector, uplink)
	if _, err := r.PushLimit(bits); err != nil {
		t.Fatal(err)
	}
	v, err := decode(r)
	if err != nil {
		t.Fatal(err)
	}
	check(t, v)
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != label || report.Regions[0].Location.BitOffset != offset || report.Regions[0].Location.Width != 14+bits-offset || r.Position() != 14+bits {
		t.Fatalf("truncated content at %d: %+v", r.Position(), report)
	}
}

func simpleOptional(t testing.TB, v any, name string, want *uint64) {
	t.Helper()
	f := simpleField(t, v, name)
	if f.Kind() != reflect.Pointer {
		t.Fatalf("%s is not optional", name)
	}
	if want == nil {
		if !f.IsNil() {
			t.Fatalf("%s unexpectedly present", name)
		}
		return
	}
	if f.IsNil() || f.Elem().Uint() != *want {
		t.Fatalf("%s=%v want %d", name, f.Interface(), *want)
	}
}
func simpleLegacy[T any](t testing.TB, selector uint64, uplink bool, body string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, check func(testing.TB, T)) {
	t.Helper()
	b := simpleBlock(t, selector, body+"0")
	b[22] = 0xa7
	r, c := simpleReader(t, b, selector, uplink)
	v, err := decode(r)
	if err != nil {
		t.Fatal(err)
	}
	check(t, v)
	if r.Position() != 184 || len(c.TakeReceiverReport().Regions) != 0 {
		t.Fatal("legacy remainder did not fill its exact bound")
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: b, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	got := simpleEncoded(t, b, v, encode, wire, false)
	if !bytes.Equal(got, b) {
		t.Fatalf("legacy replay %x want %x", got, b)
	}
	// The printed null arm is reached at the exact component end, preserving
	// the original coordinate inside the complete block used for this probe.
	nr, nc := simpleReader(t, b, selector, uplink)
	old, err := nr.PushLimit(len(body))
	if err != nil {
		t.Fatal(err)
	}
	nv, err := decode(nr)
	if err != nil {
		t.Fatal(err)
	}
	check(t, nv)
	if nr.Position() != 14+len(body) || len(nc.TakeReceiverReport().Regions) != 0 {
		t.Fatal("null suffix crossed its bound")
	}
	if err = nr.PopLimit(old); err != nil {
		t.Fatal(err)
	}
}

func simpleRegionReplay[T any](t testing.TB, b []byte, selector uint64, uplink bool, label runtime.ErrorLabel, offset int, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, check func(testing.TB, T)) {
	t.Helper()
	r, c := simpleReader(t, b, selector, uplink)
	v, err := decode(r)
	if err != nil {
		t.Fatal(err)
	}
	check(t, v)
	report := c.TakeReceiverReport()
	if r.Position() != 184 || len(report.Regions) != 1 || report.Regions[0].Label != label || report.Regions[0].Location.BitOffset != offset || report.Regions[0].Location.Width != 184-offset {
		t.Fatalf("region %+v at %d", report, r.Position())
	}
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: b, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if got := simpleEncoded(t, b, v, encode, wire, false); !bytes.Equal(got, b) {
		t.Fatalf("region replay %x want %x", got, b)
	}
	w := runtime.NewWriter()
	w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
	if _, err = w.PushLimit(184); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(uint64(b[0]), 8); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(selector, 6); err != nil {
		t.Fatal(err)
	}
	err = encode(w, runtime.Canonical(v))
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || operation.Kind != runtime.InvalidValue {
		t.Fatalf("receiver arm sender refusal: %T %v", err, err)
	}
}
func simpleIgnore[T any](t testing.TB, selector uint64, uplink bool, body string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, check func(testing.TB, T)) {
	t.Helper()
	b := simpleBlock(t, selector, body)
	at := 14 + len(body)
	b[at/8] ^= 1 << uint(7-at%8)
	simpleRegionReplay(t, b, selector, uplink, runtime.Ignore, at+1, decode, encode, check)
	simpleSpareReplay(t, selector, uplink, body, decode, encode, check)
}

// TS 44.060 V19.0.0 §11 paragraph 12938: a leading padding selector 1 selects Ignore.

func simpleSpareReplay[T any](t testing.TB, selector uint64, uplink bool, body string, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, check func(testing.TB, T)) {
	t.Helper()
	expected := simpleBlock(t, selector, body)
	for _, delta := range []int{1, 7} {
		b := bytes.Clone(expected)
		at := 14 + len(body) + delta
		b[at/8] ^= 1 << uint(7-at%8)
		r, c := simpleReader(t, b, selector, uplink)
		v, err := decode(r)
		if err != nil {
			t.Fatal(err)
		}
		check(t, v)
		if r.Position() != 184 || len(c.TakeReceiverReport().Regions) != 0 {
			t.Fatal("nonsemantic spare padding changed the receiver arm")
		}
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: b, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		if got := simpleEncoded(t, b, v, encode, wire, false); !bytes.Equal(got, b) {
			t.Fatal("nonsemantic spare replay changed bytes")
		}
		if got := simpleEncoded(t, b, v, encode, runtime.WireInfo{}, true); !bytes.Equal(got, expected) {
			t.Fatal("canonical spare padding did not restore its positional pattern")
		}
	}
}

func simpleList(t testing.TB, v any, name string, want []uint8) {
	t.Helper()
	f := simpleField(t, v, name)
	if f.Kind() == reflect.Pointer {
		if want == nil {
			if !f.IsNil() {
				t.Fatalf("%s unexpectedly present", name)
			}
			return
		}
		if f.IsNil() {
			t.Fatalf("%s absent", name)
		}
		f = f.Elem()
	}
	if f.Len() != len(want) {
		t.Fatalf("%s length %d want %d", name, f.Len(), len(want))
	}
	for i, w := range want {
		if f.Index(i).Uint() != uint64(w) {
			t.Fatalf("%s[%d]=%d want %d", name, i, f.Index(i).Uint(), w)
		}
	}
}
