package controlmessages_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

// TS 44.060 V19.0.0 §11 paragraphs 12937–12940: padding is null,
// or zero followed by the positional bits of the 00101011 spare pattern.
func simpleControlBits(prefix string, extent int) []byte {
	out := make([]byte, (extent+7)/8)
	for i := 0; i < extent; i++ {
		var bit bool
		if i < len(prefix) {
			bit = prefix[i] == '1'
		} else if i > len(prefix) {
			bit = (byte(0x2b)>>uint(7-i%8))&1 != 0
		}
		if bit {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func simpleControlWrite(t testing.TB, data []byte, origin, extent int, wire runtime.WireInfo, encode func(*runtime.Writer) error) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	for i := 0; i < origin; i++ {
		if e := w.WriteUint(uint64((data[i/8]>>uint(7-i%8))&1), 1); e != nil {
			t.Fatal(e)
		}
	}
	if e := encode(w); e != nil {
		t.Fatal(e)
	}
	out, e := w.FinishBits(runtime.BitString{})
	if e != nil {
		t.Fatal(e)
	}
	return out
}

func simpleControlReplay(t testing.TB, data []byte, extent, origin int, r *runtime.Reader, encode func(*runtime.Writer) error) {
	t.Helper()
	raw := runtime.BitString{Bytes: append([]byte(nil), data[:(extent+7)/8]...), BitLength: extent}
	if extent%8 != 0 {
		raw.Bytes[len(raw.Bytes)-1] &= byte(0xff << uint(8-extent%8))
	}
	wire, e := runtime.SealBits(struct{}{}, raw, extent, runtime.BitString{}, r.Wire())
	if e != nil {
		t.Fatal(e)
	}
	out := simpleControlWrite(t, data, origin, extent, wire, encode)
	if out.BitLength != extent || !bytes.Equal(out.Bytes, raw.Bytes) {
		t.Fatalf("replay %x/%d want %x/%d", out.Bytes, out.BitLength, raw.Bytes, extent)
	}
}

func simpleControlCanonical(t testing.TB, data []byte, origin, extent int, encode func(*runtime.Writer) error) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
	if _, e := w.PushLimit(extent); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < origin; i++ {
		if e := w.WriteUint(uint64((data[i/8]>>uint(7-i%8))&1), 1); e != nil {
			t.Fatal(e)
		}
	}
	if e := encode(w); e != nil {
		t.Fatal(e)
	}
	out, e := w.FinishBits(runtime.BitString{})
	if e != nil || out.BitLength != extent {
		t.Fatalf("canonical %x/%d: %v", out.Bytes, out.BitLength, e)
	}
	return out
}

func simpleControlFresh(t testing.TB, data []byte, origin int, encode func(*runtime.Writer) error) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	for i := 0; i < origin; i++ {
		if err := w.WriteUint(uint64((data[i/8]>>uint(7-i%8))&1), 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := encode(w); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Inspect exported generated fields, excluding replay state. The decode call at
// each use has a concrete generated return type; these checks cover nested
// fields without duplicating the generated choice declarations in test code.
func simpleControlFields(value any, name string) []uint64 {
	var out []uint64
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.Kind() == reflect.Pointer {
			if !v.IsNil() {
				walk(v.Elem())
			}
			return
		}
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				f := v.Type().Field(i)
				if !f.IsExported() || f.Name == "Wire" {
					continue
				}
				x := v.Field(i)
				if f.Name == name {
					if x.Kind() == reflect.Pointer && !x.IsNil() {
						x = x.Elem()
					}
					if x.Kind() >= reflect.Uint && x.Kind() <= reflect.Uint64 {
						out = append(out, x.Uint())
					}
				}
				walk(v.Field(i))
			}
		} else if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		}
	}
	walk(reflect.ValueOf(value))
	return out
}
func simpleControlWant(t testing.TB, value any, name string, want ...uint64) {
	t.Helper()
	if got := simpleControlFields(value, name); !reflect.DeepEqual(got, want) {
		t.Fatalf("%s=%v want %v", name, got, want)
	}
}
func simpleControlVector(t testing.TB, clause, name string, raw []byte) {
	t.Helper()
	b, e := json.Marshal(struct{ Clause, Name, Hex string }{clause, name, fmt.Sprintf("%x", raw)})
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("SYNTHETIC %s", b)
}

// TS 44.060 V19.0.0 §11.2.45 paragraphs 20799–20803 and
// §11.2.0.1 paragraph 13368: ordinary downlink selector 010110.
func TestClause11245PhysicalInformation(t *testing.T) {
	for page := 0; page < 4; page++ {
		for arm := 0; arm < 2; arm++ {
			for _, timing := range []int{0, 165, 255} {
				prefix := fmt.Sprintf("01000000010110%02b%01b%05b%08b", page, arm, 21, timing)
				raw := simpleControlBits(prefix, 184)
				r, _ := contentReader(t, raw, 184, runtime.RLCMACDownlink, 22)
				v, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(r)
				if e != nil || v.PAGEMODE != uint8(page) || v.Group.TIMINGADVANCEVALUE != uint8(timing) || r.Position() != 184 {
					t.Fatalf("typed value %+v position %d: %v", v, r.Position(), e)
				}
				simpleControlWant(t, v, "UPLINKTFI", func() []uint64 {
					if arm == 0 {
						return []uint64{21}
					}
					return nil
				}()...)
				simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
					return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, v)
				})
				cv := runtime.Canonical(v)
				out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
					return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, cv)
				})
				if !bytes.Equal(out.Bytes, raw) {
					t.Fatal("canonical changed normal fields")
				}
				cv.Group.TIMINGADVANCEVALUE ^= 1
				edited := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
					return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, cv)
				})
				er, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACDownlink, 22)
				ev, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(er)
				if e != nil || ev.Group.TIMINGADVANCEVALUE != uint8(timing^1) {
					t.Fatalf("edited value %+v: %v", ev, e)
				}
				simpleControlVector(t, "11.2.45", fmt.Sprintf("page%d-arm%d-timing%d", page, arm, timing), raw)
			}
		}
	}
}

// The reviewed reading authenticates zero inference only after PAGE_MODE and Global TFI
// at paragraph 20803. The surrounding 184-bit block remains present; its bound
// gives the receiver a truncated message followed by an untouched sentinel.
func TestClause11245ZeroFillScope(t *testing.T) {
	raw := simpleControlBits(fmt.Sprintf("01000000010110%02b%01b%05b%08b", 1, 0, 21, 255), 184)
	for n := 0; n <= 8; n++ {
		r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 22)
		end := 22 + n
		old, e := r.PushLimit(end - 14)
		if e != nil {
			t.Fatal(e)
		}
		v, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(r)
		if e != nil {
			t.Fatalf("cut %d: %v", n, e)
		}
		want := uint8(255 << uint(8-n))
		if v.Group.TIMINGADVANCEVALUE != want {
			t.Fatalf("cut %d timing %d want %d", n, v.Group.TIMINGADVANCEVALUE, want)
		}
		report := c.TakeReceiverReport()
		fills := 0
		for _, f := range report.ZeroFill {
			fills += f.Width
			if f.BitOffset < end || f.Source.Clause != "11.2.45" {
				t.Fatalf("fill %+v", f)
			}
		}
		if fills != 8-n {
			t.Fatalf("cut %d fills %d", n, fills)
		}
		simpleControlReplay(t, raw, end, 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, v)
		})
		if e = r.PopLimit(old); e != nil {
			t.Fatal(e)
		}
		sentinel, e := r.ReadUint(1)
		wantBit := uint64((raw[end/8] >> uint(7-end%8)) & 1)
		if e != nil || sentinel != wantBit {
			t.Fatalf("sentinel %d want %d: %v", sentinel, wantBit, e)
		}
		cv := runtime.Canonical(v)
		full := simpleControlCanonical(t, raw, 14, 30, func(w *runtime.Writer) error {
			return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, cv)
		})
		fr, _ := runtime.NewChoiceReaderBits(full.Bytes, 30, nil)
		_, _ = fr.ReadUint(14)
		fv, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(fr)
		if e != nil || fv.Group.TIMINGADVANCEVALUE != want {
			t.Fatalf("fresh sender %+v: %v", fv, e)
		}
	}
	for n := 14; n < 22; n++ {
		r, _ := runtime.NewChoiceReaderBits(raw, n, nil)
		_, _ = r.ReadUint(14)
		if _, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(r); e == nil {
			t.Fatalf("inferred prefix at %d", n)
		}
	}
}
func FuzzClause11245PhysicalInformation(f *testing.F) {
	f.Add(simpleControlBits("01000000010110011010110100101", 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(r)
		if e == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, v)
			})
		}
	})
}

func TestClause11245IgnoredPadding(t *testing.T) {
	prefix := fmt.Sprintf("01000000010110%02b%01b%05b%08b", 1, 0, 21, 165)
	raw := simpleControlBits(prefix+"1", 184)
	r, c := contentReader(t, raw, 184, runtime.RLCMACDownlink, 22)
	v, e := controlmessages.DecodePacketPhysicalInformationMessageContentFrom(r)
	if e != nil || v.PAGEMODE != 1 || v.Group.TIMINGADVANCEVALUE != 165 {
		t.Fatalf("ignored padding value %+v: %v", v, e)
	}
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore {
		t.Fatalf("ignored padding report %+v", report)
	}
	simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
		return controlmessages.EncodePacketPhysicalInformationMessageContentTo(w, v)
	})
}
