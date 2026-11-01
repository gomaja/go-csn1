package radioaccesscapability_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts44060/radioaccesscapability"
)

// TS 44.060 V19.0.0 §12.30 p23547 suppresses every sender spare.
// TS 24.008 V20.1.0 §10.5.5.12a pp30776–30803 includes spares
// in both seven-bit lengths. Received spares remain part of exact replay.
func TestCapabilityNestedSpareSenderAndReplay(t *testing.T) {
	for tag := 0; tag < 15; tag++ {
		for _, spares := range []int{0, 1, 4, 54} {
			body := fmt.Sprintf("%04b%07b001%s%s0", tag, 73+spares, strings.Repeat("0", 70), strings.Repeat("1", spares))
			want := fmt.Sprintf("%04b%07b001%s0", tag, 73, strings.Repeat("0", 70))
			checkCapabilitySpareProfile(t, body, want)
		}
	}
	for _, count := range []int{0, 1, 12} {
		for _, spares := range []int{0, 1, 4, 6} {
			prefix := strings.TrimSuffix(capabilityString(1), "0") + "1"
			list := strings.Repeat("1"+"0010"+"011"+"01", count) + "0"
			body := prefix + "1111" + fmt.Sprintf("%07b", len(list)+spares) + list + strings.Repeat("1", spares) + "0"
			want := prefix + "1111" + fmt.Sprintf("%07b", len(list)) + list + "0"
			checkCapabilitySpareProfile(t, body, want)
		}
	}
}

func checkCapabilitySpareProfile(t *testing.T, body, want string) {
	t.Helper()
	for phase := 0; phase < 8; phase++ {
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
		value, err := radioaccesscapability.DecodeMSRadioAccessCapability2IEFrom(r)
		if err != nil {
			t.Fatalf("phase %d decode: %v", phase, err)
		}
		if r.Position() != phase+len(body) {
			t.Fatal("received boundary changed")
		}
		wire := r.Wire()
		if err = r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(3)
		if err != nil || sentinel != 5 {
			t.Fatal("containing sentinel consumed")
		}
		replay := capabilityEmit(t, value, phase, wire, false)
		expectedReplay := capabilityBytes(strings.Repeat("1", phase) + body)
		if replay.BitLength != phase+len(body) || !bytes.Equal(replay.Bytes, expectedReplay) {
			t.Fatal("received spare lost on replay")
		}
		canonical := capabilityEmit(t, value, phase, runtime.WireInfo{}, true)
		expected := capabilityBytes(strings.Repeat("1", phase) + want)
		if canonical.BitLength != phase+len(want) || !bytes.Equal(canonical.Bytes, expected) {
			t.Fatalf("canonical retains a spare: phase %d bits %d, want %d", phase, canonical.BitLength, phase+len(want))
		}
		again := capabilityEmit(t, value, phase, wire, false)
		if again.BitLength != replay.BitLength || !bytes.Equal(again.Bytes, replay.Bytes) {
			t.Fatal("canonical mutated received state")
		}
	}
	raw := capabilityBytes(body)
	decoded, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(raw)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := radioaccesscapability.EncodeMSRadioAccessCapability2IECanonicalAtLength(decoded.Value, (len(want)+7)/8)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(canonical, capabilityBytes(want)) {
		t.Fatal("canonical byte API differs")
	}
	fresh := runtime.Canonical(decoded.Value)
	// A plain fresh sender uses the same policy without an explicit bit target.
	fresh.Wire = runtime.WireInfo{}
	for i := range fresh.MSRACapability.Entries {
		arm := &fresh.MSRACapability.Entries[i].AccessTechnologyTypeChoice
		if arm.AccessCapabilities != nil {
			arm.AccessCapabilities.AccessCapabilities.Length = 0
		}
		if arm.Length != nil {
			arm.Length.Length = 0
		}
	}

	out, err := radioaccesscapability.EncodeMSRadioAccessCapability2IE(fresh)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, capabilityBytes(want)) {
		t.Fatal("plain fresh sender retained spare bits")
	}
}

func TestCapabilityOrdinaryContextKeepsDeclaredSpares(t *testing.T) {
	raw := capabilityBytes("0001" + fmt.Sprintf("%07b", 77) + "001" + strings.Repeat("0", 70) + "1111" + "0")
	d, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(raw)
	if err != nil {
		t.Fatal(err)
	}
	cap := runtime.Canonical(d.Value.MSRACapability.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities)
	w := runtime.NewWriter()
	if err = msrac.EncodeAccessCapabilitiesStructTo(w, cap); err != nil {
		t.Fatal(err)
	}
	if w.Position() != 84 {
		t.Fatalf("ordinary TS 24.008 length changed: %d", w.Position())
	}
}

func TestCapabilitySenderRetainsCurrentReleaseFields(t *testing.T) {
	// TS 24.008 V20.1.0 §10.5.5.12a pp30875–30883: the
	// optional four-bit sync accuracy and three final features are Content.
	content := "001" + strings.Repeat("0", 66) + "1" + "1001" + "111"
	body := "0001" + fmt.Sprintf("%07b", len(content)+4) + content + "1111" + "0"
	want := "0001" + fmt.Sprintf("%07b", len(content)) + content + "0"
	checkCapabilitySpareProfile(t, body, want)
	d, err := radioaccesscapability.DecodeMSRadioAccessCapability2IE(capabilityBytes(want))
	if err != nil {
		t.Fatal(err)
	}
	c := d.Value.MSRACapability.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessCapabilities.Content.AccessCapabilities
	if c.MSSyncAccuracy == nil || *c.MSSyncAccuracy != 9 || c.ECUplinkCoverageEnhancementSupport != 1 || c.MTAAccessSecuritySupport != 1 || c.ECPagingIndicationChannelMonitoringSupport != 1 {
		t.Fatal("current release fields lost")
	}
}
