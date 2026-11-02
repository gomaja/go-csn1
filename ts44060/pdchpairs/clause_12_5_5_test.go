package pdchpairs

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.5.5 tables 12.5.5.1–2, pp22497–22525.
// This IE has no independent length. A caller supplies the containing bit
// extent; a reserved 11 consumes that remainder (§§11.1.2–11.1.3.2).
// The codec reports the printed label; application procedures stay with callers.
func TestClause12_5_5(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		for carrier := 0; carrier < 2; carrier++ {
			for selector := 0; selector < 4; selector++ {
				for _, tail := range []string{"", "1", "10101", strings.Repeat("101", 11)} {
					code := fmt.Sprintf("%d%02b", carrier, selector)
					payload := ""
					want := map[string]uint64{}
					if selector == 2 {
						want = map[string]uint64{"DOWNLINKPDCHPAIRSC1": 0xa5, "UPLINKPDCHPAIRSC1": 0x3c}
						payload = "1010010100111100"
						if carrier == 1 {
							payload = "10100101001111001000000101111110"
							want = map[string]uint64{"DOWNLINKPDCHPAIRSC1": 0xa5, "DOWNLINKPDCHPAIRSC2": 0x3c, "UPLINKPDCHPAIRSC1": 0x81, "UPLINKPDCHPAIRSC2": 0x7e}
						}
					}
					body := code + payload
					if selector == 3 {
						body += tail
					}
					prefix := strings.Repeat("0", phase)
					input := pdchBits(prefix + body + "101")
					c := runtime.NewReceiverCollector()
					r, err := runtime.NewChoiceReaderBits(input, phase+len(body)+3, c)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := r.ReadUint(phase); err != nil {
						t.Fatal(err)
					}
					old, err := r.PushLimit(len(body))
					if err != nil {
						t.Fatal(err)
					}
					v, err := DecodePDCHPairsDescriptionIEFrom(r)
					if err != nil {
						t.Fatalf("carrier %d selector %d phase %d: %v", carrier, selector, phase, err)
					}
					if r.Position() != phase+len(body) {
						t.Fatalf("end %d want %d", r.Position(), phase+len(body))
					}
					report := c.TakeReceiverReport()
					if selector == 3 {
						if !v.ReceiverRegionEnds() || len(report.Regions) != 1 || string(report.Regions[0].Label) != "PDCH pairs configuration error" || report.Regions[0].Location.BitOffset != phase+3 || report.Regions[0].Location.Width != len(tail) {
							t.Fatalf("error designation %+v %+v", v, report)
						}
					} else {
						if v.ReceiverRegionEnds() || len(report.Regions) != 0 {
							t.Fatalf("ordinary selector became error region: %+v", report)
						}
						got := map[string]uint64{}
						pdchFields(reflect.ValueOf(v), got)
						if !reflect.DeepEqual(got, want) {
							t.Fatalf("typed fields %+v want %+v", got, want)
						}
					}
					if err := r.PopLimit(old); err != nil {
						t.Fatal(err)
					}
					sentinel, err := r.ReadUint(3)
					if err != nil || sentinel != 5 {
						t.Fatalf("sentinel %d %v", sentinel, err)
					}
					d, err := Lookup("PDCH Pairs Description IE")
					if err != nil {
						t.Fatal(err)
					}
					decoded, err := d.DecodeBits(pdchBits(body), len(body))
					if err != nil {
						t.Fatal(err)
					}
					value := decoded.(runtime.Decoded[PDCHPairsDescriptionIE]).Value
					w := runtime.NewWriter()
					if err := EncodePDCHPairsDescriptionIETo(w, value); err != nil {
						t.Fatal(err)
					}
					out, err := w.FinishBits(runtime.BitString{})
					if err != nil || out.BitLength != len(body) || !bytes.Equal(out.Bytes, pdchBits(body)) {
						t.Fatalf("replay %+v %v", out, err)
					}
					canonical := runtime.Canonical(value)
					w = runtime.NewWriter()
					err = EncodePDCHPairsDescriptionIETo(w, canonical)
					if selector == 3 {
						var op *runtime.OperationError
						if !errors.As(err, &op) || op.Kind != runtime.InvalidValue {
							t.Fatalf("sender reserved refusal: %v", err)
						}
						if _, err := EncodePDCHPairsDescriptionIECanonicalAtLength(value, (len(body)+7)/8); !errors.As(err, &op) {
							t.Fatalf("canonical reserved refusal: %v", err)
						}
						canonical.Wire = runtime.WireInfo{}
						if err := EncodePDCHPairsDescriptionIETo(runtime.NewWriter(), canonical); !errors.As(err, &op) {
							t.Fatalf("fresh reserved refusal: %v", err)
						}
					} else {
						if err != nil {
							t.Fatal(err)
						}
						out, err = w.FinishBits(runtime.BitString{})
						if err != nil || out.BitLength != len(body) || !bytes.Equal(out.Bytes, pdchBits(body)) {
							t.Fatalf("fresh %+v %v", out, err)
						}
						if _, err := EncodePDCHPairsDescriptionIECanonicalAtLength(value, (len(body)+7)/8); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		}
	}
	if _, err := DecodePDCHPairsDescriptionIEFrom(runtime.NewReader([]byte{0xff})); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatalf("missing caller extent: %v", err)
	}
}

func pdchBits(s string) []byte {
	b := make([]byte, (len(s)+7)/8)
	for i, c := range s {
		if c == '1' {
			b[i/8] |= 1 << uint(7-i%8)
		}
	}
	return b
}
func pdchFields(v reflect.Value, out map[string]uint64) {
	if v.Kind() == reflect.Pointer {
		if !v.IsNil() {
			pdchFields(v.Elem(), out)
		}
		return
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		field := v.Field(i)
		if strings.Contains(name, "PDCHPAIRSC") && field.Kind() == reflect.Uint8 {
			out[name] = field.Uint()
		} else if name != "Wire" {
			pdchFields(field, out)
		}
	}
}

func FuzzClause12_5_5(f *testing.F) {
	f.Add([]byte{0x75}, uint16(8))
	f.Add([]byte{0xf5}, uint16(8))
	f.Add([]byte{0x54, 0xa7, 0x80}, uint16(19))
	f.Fuzz(func(t *testing.T, data []byte, length uint16) {
		if len(data) > 1024 {
			return
		}
		bits := int(length)
		if bits > len(data)*8 {
			bits = len(data) * 8
		}
		d, err := Lookup("PDCH Pairs Description IE")
		if err != nil {
			t.Fatal(err)
		}
		plain, pe := d.DecodeBits(data, bits)
		reported, _, _, re := d.DecodeBitsWithReport(data, bits)
		if fmt.Sprint(pe) != fmt.Sprint(re) || !reflect.DeepEqual(plain, reported) {
			t.Fatalf("report parity %v / %v", pe, re)
		}
		if pe != nil {
			return
		}
		v := plain.(runtime.Decoded[PDCHPairsDescriptionIE]).Value
		w := runtime.NewWriter()
		if err := EncodePDCHPairsDescriptionIETo(w, v); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		want := append([]byte(nil), data[:(bits+7)/8]...)
		if bits%8 != 0 {
			want[len(want)-1] &= 0xff << uint(8-bits%8)
		}
		if err != nil || out.BitLength != bits || !bytes.Equal(out.Bytes, want) {
			t.Fatalf("bit replay %+v want %x/%d: %v", out, want, bits, err)
		}
	})
}

// Increasing ignored suffixes retain one region with input-proportional
// storage. This allocation-byte bound does not make a timing or RSS claim.
func TestPDCHErrorRegionGrowth(t *testing.T) {
	d, err := Lookup("PDCH Pairs Description IE")
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{256, 4096, 65536, 262144} {
		body := "011" + strings.Repeat("1", size)
		input := pdchBits(body)
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		_, _, report, err := d.DecodeBitsWithReport(input, len(body))
		goruntime.ReadMemStats(&after)
		if err != nil || len(report.Regions) != 1 || report.Regions[0].Location.Width != size {
			t.Fatalf("region size %d: %+v %v", size, report, err)
		}
		bound := uint64(len(input))*64 + 128*1024
		t.Logf("ignored region %d bits: %d allocated bytes; bound %d", size, after.TotalAlloc-before.TotalAlloc, bound)
		if allocated := after.TotalAlloc - before.TotalAlloc; !raceEnabled && allocated > bound {
			t.Fatalf("region %d bits allocated %d bytes, bound %d", size, allocated, bound)
		}
	}
}

func TestPDCHRequiresCallerExtentBeforeSelector(t *testing.T) {
	for carrier := 0; carrier < 2; carrier++ {
		for selector := 0; selector < 4; selector++ {
			input := pdchBits(fmt.Sprintf("%d%02b", carrier, selector) + strings.Repeat("1", 40))
			r := runtime.NewReader(input)
			if _, err := DecodePDCHPairsDescriptionIEFrom(r); !errors.Is(err, runtime.ErrContextRequired) || r.Position() != 0 {
				t.Fatalf("carrier %d selector %d progressed to %d: %v", carrier, selector, r.Position(), err)
			}
		}
	}
}

func TestPDCHExplicitEditsAndTruncations(t *testing.T) {
	d, err := Lookup("PDCH Pairs Description IE")
	if err != nil {
		t.Fatal(err)
	}
	for carrier := 0; carrier < 2; carrier++ {
		payload := "1010010100111100"
		if carrier == 1 {
			payload = "10100101001111001000000101111110"
		}
		body := fmt.Sprintf("%d10", carrier) + payload
		for cut := 0; cut < len(body); cut++ {
			_, _, report, err := d.DecodeBitsWithReport(pdchBits(body), cut)
			if err == nil || len(report.Regions) != 0 {
				t.Fatalf("carrier %d truncated at %d selected receiver error region: %+v %v", carrier, cut, report, err)
			}
		}
		decoded, err := d.DecodeBits(pdchBits(body), len(body))
		if err != nil {
			t.Fatal(err)
		}
		received := decoded.(runtime.Decoded[PDCHPairsDescriptionIE]).Value
		edited := runtime.Canonical(received)
		edited.Wire = received.Wire
		if !pdchEdit(reflect.ValueOf(&edited).Elem(), "DOWNLINKPDCHPAIRSC1", 0x5a) {
			t.Fatal("typed downlink field absent")
		}
		w := runtime.NewWriter()
		if err := EncodePDCHPairsDescriptionIETo(w, edited); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		want := body[:3] + "01011010" + body[11:]
		if err != nil || out.BitLength != len(want) || !bytes.Equal(out.Bytes, pdchBits(want)) {
			t.Fatalf("typed edit %+v want %x: %v", out, pdchBits(want), err)
		}
		canonical, err := EncodePDCHPairsDescriptionIECanonicalAtLength(edited, (len(want)+7)/8)
		if err != nil || !bytes.Equal(canonical, pdchBits(want)) {
			t.Fatalf("canonical edit %x want %x: %v", canonical, pdchBits(want), err)
		}
	}
}

func TestPDCHReservedArmIntegrity(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"label", "receiver error label mismatch"},
		{"extent", "receiver error region must fill its containing extent"},
		{"missing arm", "receiver fallback requires exactly one arm"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := DecodePDCHPairsDescriptionIE([]byte{0x75})
			if err != nil {
				t.Fatal(err)
			}
			v := decoded.Value
			arm := v.DOWNLINKPDCHPAIRSC1Choice.PDCHPairsConfigurationError.DOWNLINKPDCHPAIRSC1Choice.DOWNLINKPDCHPAIRSC1
			switch tc.name {
			case "label":
				arm.Ignored.Group2.Label = runtime.ErrorLabel("Ignore")
			case "extent":
				arm.Ignored.Group2.Bits.BitLength--
			case "missing arm":
				arm.Ignored = nil
			}
			w := runtime.NewWriter()
			if _, err := w.PushLimit(8); err != nil {
				t.Fatal(err)
			}
			err = EncodePDCHPairsDescriptionIETo(w, v)
			var operation *runtime.OperationError
			if !errors.As(err, &operation) || operation.Kind != runtime.InvalidValue || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s: %v", tc.name, err)
			}
		})
	}
}

// An ordinary arm can leave received tail bits after its semantic fields.
// EncodeTo owns that value-local wire and appends the tail once, including
// when the containing writer starts at a nonzero bit phase.
func TestPDCHReceivedTailReplay(t *testing.T) {
	d, err := Lookup("PDCH Pairs Description IE")
	if err != nil {
		t.Fatal(err)
	}
	for carrier := 0; carrier < 2; carrier++ {
		for selector := 0; selector < 3; selector++ {
			body := fmt.Sprintf("%d%02b", carrier, selector)
			if selector == 2 {
				body += "1010010100111100"
				if carrier == 1 {
					body += "1000000101111110"
				}
			}
			for _, tail := range []string{"1", "10101", "1010101010101"} {
				input := pdchBits(body + tail)
				decoded, err := d.DecodeBits(input, len(body)+len(tail))
				if err != nil {
					t.Fatal(err)
				}
				v := decoded.(runtime.Decoded[PDCHPairsDescriptionIE]).Value
				for phase := 0; phase < 8; phase++ {
					w := runtime.NewWriter()
					if err := w.WriteUint(0, phase); err != nil {
						t.Fatal(err)
					}
					if err := EncodePDCHPairsDescriptionIETo(w, v); err != nil {
						t.Fatal(err)
					}
					out, err := w.FinishBits(runtime.BitString{})
					want := strings.Repeat("0", phase) + body + tail
					if err != nil || out.BitLength != len(want) || !bytes.Equal(out.Bytes, pdchBits(want)) {
						t.Fatalf("carrier %d selector %d phase %d tail %d: replay failed: %v", carrier, selector, phase, len(tail), err)
					}
				}
				if (len(body)+len(tail))%8 == 0 {
					out, err := EncodePDCHPairsDescriptionIE(v)
					if err != nil || !bytes.Equal(out, input) {
						t.Fatalf("public byte replay failed: %v", err)
					}
				}
			}
		}
	}
}

func pdchEdit(v reflect.Value, name string, value uint64) bool {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		return pdchEdit(v.Elem(), name, value)
	}
	if v.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		if fieldName == name && field.Kind() == reflect.Uint8 && field.CanSet() {
			field.SetUint(value)
			return true
		}
		if fieldName != "Wire" && pdchEdit(field, name, value) {
			return true
		}
	}
	return false
}
