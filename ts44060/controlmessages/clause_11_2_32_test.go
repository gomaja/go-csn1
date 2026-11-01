package controlmessages_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/msrac"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

func checkAdditionalCapabilityType(t testing.TB, capability msrac.MSRACapabilityValuePartStruct) {
	t.Helper()
	if len(capability.Entries) != 1 {
		t.Fatalf("typed capability entries %d", len(capability.Entries))
	}
}

func additionalCapabilityPrefix(address, extension int) string {
	return additionalCapabilityPrefixBits(address, extension, 73)
}

func additionalCapabilityPrefixBits(address, extension, contentBits int) string {
	addr := "0010101" // address selector 0, Global TFI uplink 0, TFI 10101
	if address == 1 {
		addr = fmt.Sprintf("1%032b", uint32(0x10203040))
	}
	// TS 44.060 V19.0.0 §12.30 pp23543–23547 delegates to the
	// TS 24.008 V20.1.0 §10.5.5.12a struct, excluding outer spare bits.
	// With optional groups absent, Content has 69 bits through Release 13.
	// Release 14 adds one absent selector and one EC-uplink bit; Release 15
	// adds MTA and EC-paging bits: 69+1+1+1+1=73 bits.
	// TS 24.008 V20.1.0 §10.5.5.12a pp30772–30883.
	capability := fmt.Sprintf("0001%07b", contentBits) + "001" + strings.Repeat("0", contentBits-3) + "0"
	tail := "10"
	if extension >= 0 {
		tail = fmt.Sprintf("11%03b", extension)
	}
	return "01000000001011" + addr + capability + tail
}

// TS 44.060 V19.0.0 §11.2.32 pp19911–19918; selector 001011
// belongs to §11.2.0.2. The typed capability is the explicit §12.30 dependency.
func TestClause11232AdditionalCapabilities(t *testing.T) {
	for address := 0; address < 2; address++ {
		for _, extension := range []int{-1, 0, 7} {
			raw := simpleControlBits(additionalCapabilityPrefix(address, extension), 184)
			r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 11)
			v, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			checkAdditionalCapabilityType(t, v.MSRadioAccessCapability2)
			simpleControlWant(t, v, "AccessTechnologyType", 1)
			simpleControlWant(t, v, "RFPowerCapability", 1)
			simpleControlWant(t, v, "ECUplinkCoverageEnhancementSupport", 0)
			simpleControlWant(t, v, "MTAAccessSecuritySupport", 0)
			simpleControlWant(t, v, "ECPagingIndicationChannelMonitoringSupport", 0)
			if address == 0 {
				simpleControlWant(t, v, "UPLINKTFI", 21)
			} else {
				simpleControlWant(t, v, "TLLI", 0x10203040)
			}
			if extension < 0 {
				simpleControlWant(t, v, "DownlinkETFI")
			} else {
				simpleControlWant(t, v, "DownlinkETFI", uint64(extension))
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, v)
			})
			cv := runtime.Canonical(v)
			out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, cv)
			})
			if !bytes.Equal(out.Bytes, raw) {
				t.Fatal("canonical capability content changed")
			}
			v.MSRadioAccessCapability2.Entries[0].AccessTechnologyTypeChoice.AccessCapabilities.AccessTechnologyType = 2
			edited := simpleControlWrite(t, raw, 14, 184, r.Wire(), func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, v)
			})
			read, _ := contentReader(t, edited.Bytes, 184, runtime.RLCMACUplink, 11)
			changed, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(read)
			if err != nil {
				t.Fatal(err)
			}
			simpleControlWant(t, changed, "AccessTechnologyType", 2)
			simpleControlVector(t, "11.2.32", fmt.Sprintf("address%d-extension%d", address, extension), raw)
		}
	}
}

func TestClause11232CapabilityBounds(t *testing.T) {
	raw := simpleControlBits(additionalCapabilityPrefix(0, 7), 184)
	for cut := 14; cut < len(additionalCapabilityPrefix(0, -1))-2; cut++ {
		r, _ := runtime.NewChoiceReaderBits(raw, cut, nil)
		_, _ = r.ReadUint(14)
		if _, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r); err == nil {
			t.Fatalf("accepted incomplete address/capability at %d", cut)
		}
	}
	// Neither a declared capability length nor a received compatibility suffix
	// may read the sentinel after the exact message extent.
	withSentinel := append(append([]byte(nil), raw...), 0xa0)
	r, _ := runtime.NewChoiceReaderBits(withSentinel, 187, nil)
	old, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = r.ReadUint(14)
	if _, err = controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r); err != nil {
		t.Fatal(err)
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadUint(3)
	if err != nil || got != 5 {
		t.Fatalf("sentinel %d %v", got, err)
	}
	// A null release tail is accepted at the supplied boundary; a zero release
	// marker instead consumes only the receiver-compatible remainder.
	for _, tail := range []string{"", "0" + strings.Repeat("1", 31)} {
		prefix := additionalCapabilityPrefix(0, -1)
		prefix = prefix[:len(prefix)-2] + tail
		wire := simpleControlBits(prefix, len(prefix))
		r, _ := runtime.NewChoiceReaderBits(wire, len(prefix), nil)
		_, _ = r.ReadUint(14)
		v, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		simpleControlReplay(t, wire, len(prefix), 14, r, func(w *runtime.Writer) error {
			return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, v)
		})
	}
}

// A received earlier capability prefix keeps its declared length and replay.
func TestClause11232EarlierCapabilityPrefix(t *testing.T) {
	raw := simpleControlBits(additionalCapabilityPrefixBits(0, 7, 12), 184)
	r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 11)
	v, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	simpleControlWant(t, v, "Length", 12)
	simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
		return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, v)
	})
}

func FuzzClause11232AdditionalCapabilities(f *testing.F) {
	f.Add(simpleControlBits(additionalCapabilityPrefix(0, 7), 184))
	f.Add(simpleControlBits(additionalCapabilityPrefix(1, -1), 184))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		_, _ = r.ReadUint(14)
		v, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r)
		if err == nil {
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, v)
			})
		}
	})
}
