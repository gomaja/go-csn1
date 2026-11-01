package radioaccesscapability_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts44060/radioaccesscapability"
)

func capabilityString(tags ...int) string {
	var bits strings.Builder
	for i, tag := range tags {
		// TS 24.008 V20.1.0 §10.5.5.12a pp30776–30883: one ordinary
		// access-technology entry, 73-bit Content through Release 15, and a
		// recursive continuation selector. The outer spare repetition is absent.
		fmt.Fprintf(&bits, "%04b%07b001%s", tag, 73, strings.Repeat("0", 70))
		if i+1 == len(tags) {
			bits.WriteByte('0')
		} else {
			bits.WriteByte('1')
		}
	}
	return bits.String()
}

func capabilityBytes(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}

func capabilityEmit(t testing.TB, v radioaccesscapability.MSRadioAccessCapability2IE, phase int, wire runtime.WireInfo, canonical bool) runtime.BitString {
	t.Helper()
	if canonical {
		v = runtime.Canonical(v)
	}
	w := runtime.NewWriter()
	if !canonical {
		w.WithWire(wire)
	}
	if err := w.WriteLiteral(strings.Repeat("1", phase)); err != nil {
		t.Fatal(err)
	}
	if err := radioaccesscapability.EncodeMSRadioAccessCapability2IETo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TS 44.060 V19.0.0 §12.30 pp23543–23547 requires the typed
// TS 24.008 value-part struct and suppresses its outer spare repetition.
func TestClause1230TypedCapability(t *testing.T) {
	cases := [][]int{{1}, {2}, {1, 2}, {2, 1, 3}}
	for _, tags := range cases {
		body := capabilityString(tags...)
		for phase := 0; phase < 8; phase++ {
			t.Run(fmt.Sprintf("tags%v-phase%d", tags, phase), func(t *testing.T) {
				raw := capabilityBytes(strings.Repeat("1", phase) + body + "101")
				r, err := runtime.NewChoiceReaderBits(raw, phase+len(body)+3, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = r.ReadUint(phase); err != nil {
					t.Fatal(err)
				}
				old, err := r.PushLimit(len(body))
				if err != nil {
					t.Fatal(err)
				}
				v, err := radioaccesscapability.DecodeMSRadioAccessCapability2IEFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				typed := v.MSRACapability
				if len(typed.Entries) != len(tags) {
					t.Fatalf("typed entries %d", len(typed.Entries))
				}
				for i, tag := range tags {
					cap := typed.Entries[i].AccessTechnologyTypeChoice.AccessCapabilities
					if cap == nil || cap.AccessTechnologyType != uint8(tag) || cap.AccessCapabilities.Length != 73 || cap.AccessCapabilities.Content.AccessCapabilities.RFPowerCapability != 1 {
						t.Fatal("typed capability fields differ")
					}
				}
				if r.Position() != phase+len(body) {
					t.Fatalf("wrapper consumed spare bits: %d", r.Position())
				}
				wire := r.Wire()
				if err = r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				sentinel, err := r.ReadUint(3)
				if err != nil || sentinel != 5 {
					t.Fatalf("sentinel %d: %v", sentinel, err)
				}
				for _, canonical := range []bool{false, true} {
					out := capabilityEmit(t, v, phase, wire, canonical)
					want := capabilityBytes(strings.Repeat("1", phase) + body)
					if out.BitLength != phase+len(body) || !bytes.Equal(out.Bytes, want) {
						t.Fatal("exact wrapper extent changed")
					}
				}
				// Editing a typed access-technology value preserves its exact extent.
				edited := v
				edited.MSRACapability.Entries = append([]msrac.MSRACapabilityValuePartStructElement(nil), v.MSRACapability.Entries...)
				first := *edited.MSRACapability.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities
				first.AccessTechnologyType = 3
				edited.MSRACapability.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities = &first
				out := capabilityEmit(t, edited, phase, wire, false)
				rr, _ := runtime.NewChoiceReaderBits(out.Bytes, out.BitLength, nil)
				_, _ = rr.ReadUint(phase)
				changed, err := radioaccesscapability.DecodeMSRadioAccessCapability2IEFrom(rr)
				if err != nil {
					t.Fatal(err)
				}
				if changed.MSRACapability.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessTechnologyType != 3 {
					t.Fatal("typed edit lost")
				}
			})
		}
		// The byte API preserves the backing octet's tail while the component
		// reader above consumes exactly the spare-suppressed semantic extent.
		raw := capabilityBytes(body)
		got, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(raw)
		if err != nil {
			t.Fatal(err)
		}
		out, err := radioaccesscapability.EncodeMSRadioAccessCapability2IE(got.Value)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("byte replay: %v", err)
		}
		if got.BitsConsumed != len(body) {
			t.Fatal("byte semantic boundary differs")
		}
		if dir := os.Getenv("CSN1_SYNTHETIC_VECTORS"); dir != "" {
			data, err := json.Marshal(map[string]any{"tags": tags, "bits": len(body), "hex": fmt.Sprintf("%x", raw)})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("capability-%d.json", len(tags)*10+tags[0])), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestClause1230CapabilityBoundaries(t *testing.T) {
	body := capabilityString(1)
	raw := capabilityBytes(body + "101")
	// The four-bit technology discriminator and seven-bit length are mandatory;
	// a declared Content may not consume the sentinel beyond its containing end.
	for cut := 0; cut < len(body); cut++ {
		r, _ := runtime.NewChoiceReaderBits(raw, cut, nil)
		if _, err := radioaccesscapability.DecodeMSRadioAccessCapability2IEFrom(r); err == nil {
			t.Fatalf("accepted truncated wrapper at %d", cut)
		}
	}
	v, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(capabilityBytes(body))
	if err != nil {
		t.Fatal(err)
	}
	clean := runtime.Canonical(v.Value)
	w := runtime.NewWriter()
	if err = radioaccesscapability.EncodeMSRadioAccessCapability2IETo(w, clean); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	r, _ := runtime.NewChoiceReaderBits(out.Bytes, out.BitLength, nil)
	decoded, err := radioaccesscapability.DecodeMSRadioAccessCapability2IEFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runtime.Canonical(decoded), clean) {
		t.Fatal("fresh typed fields differ")
	}
}

func FuzzRadioAccessCapability2(f *testing.F) {
	f.Add(capabilityBytes(capabilityString(1)))
	f.Add(capabilityBytes(capabilityString(1, 2)))
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			return
		}
		decoded, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(data)
		if err != nil {
			return
		}
		out, err := radioaccesscapability.EncodeMSRadioAccessCapability2IE(decoded.Value)
		if err != nil {
			t.Fatalf("accepted wrapper cannot replay: %v", err)
		}
		if !bytes.Equal(out, data) {
			t.Fatal("wrapper replay differs")
		}
	})
}

func TestClause1230CapabilityLinearAllocationGrowth(t *testing.T) {
	var previous float64
	for _, count := range []int{8, 32, 128, 512} {
		tags := make([]int, count)
		for i := range tags {
			tags[i] = 1 + i%2
		}
		data := capabilityBytes(capabilityString(tags...))
		allocations := testing.AllocsPerRun(1, func() {
			decoded, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(data)
			if err != nil || len(decoded.Value.MSRACapability.Entries) != count {
				t.Fatalf("growing typed capability %d: %v", count, err)
			}
		})
		// Four times the input may use at most six times the allocations. This
		// permits bounded setup costs while rejecting quadratic list growth.
		if !raceEnabled && previous > 0 && allocations > previous*6+1024 {
			t.Fatalf("nonlinear allocation growth at %d entries: %.0f after %.0f", count, allocations, previous)
		}
		t.Logf("entries=%d input_octets=%d decode_allocations=%.0f", count, len(data), allocations)
		previous = allocations
	}
}
