package registry

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// This list is kept equal to the README's explicit unresolved-carrier table.
// Membership records an open source question, never a wire-size permission.
var unresolvedCarrierLimits = []string{
	"TS 44.060/12.10a/ARFCN index list struct",
	"TS 44.060/12.10a/GPRS Mobile Allocation IE",
	"TS 44.060/12.10a/RFL number list struct",
	"TS 44.060/12.3.2/FLO Ack/Nack Description IE",
	"TS 44.060/12.3.2/FLO Ack/Nack Description struct",
	"TS 44.060/12.36/MBMS Channel Parameters IE",
	"TS 44.060/12.37/MBMS p-t-m channel description IE",
	"TS 44.060/12.38/MPRACH description IE",
	"TS 44.060/12.39/MBMS Session List IE",
	"TS 44.060/12.40/MBMS Session Parameters List IE",
	"TS 44.060/12.50/3G Individual Priority Parameters Description struct",
	"TS 44.060/12.50/E-UTRAN Individual Priority Parameters Description struct",
	"TS 44.060/12.50/Individual priorities IE",
	"TS 44.060/12.50/Repeated Individual E-UTRAN Priority Parameters struct",
	"TS 44.060/12.50/Repeated Individual UTRAN Priority Parameters struct",
	"TS 44.060/12.53/E-UTRAN Parameters IE",
	"TS 44.060/12.53/Repeated E-UTRAN Neighbour Cells struct",
	"TS 44.060/12.53/Repeated E-UTRAN Not Allowed Cells struct",
	"TS 44.060/12.53/Repeated E-UTRAN PCID to TA mapping struct",
	"TS 44.060/12.57/PCID Group IE",
	"TS 44.060/12.58/PSC Group IE",
	"TS 44.060/12.59/Enhanced Cell Reselection Parameters IE",
	"TS 44.060/12.59/Repeated E-UTRAN Enhanced Cell Reselection Parameters struct",
	"TS 44.060/12.65/Dynamic Allocation 3 IE",
	"TS 44.060/12.69/E-UTRAN IPP with extended EARFCNs IE",
	"TS 44.060/12.69/Repeated Individual E-UTRAN PP with extended EARFCNs struct",
	"TS 44.060/12.70/E-UTRAN NC with extended EARFCNs IE",
	"TS 44.060/12.70/Repeated E-UTRAN NC with extended EARFCNs struct",
	"TS 44.060/12.7a/Iu mode Channel Request Description IE",
	"TS 44.060/12.7b/Extended Channel Request Description IE",
	"TS 44.060/12.8.2/Dual Carrier Direct encoding 1 struct",
	"TS 44.060/12.8.2/Dual Carrier Frequency Parameters IE",
	"TS 44.060/12.8.4/DLMC Direct encoding 1 struct",
	"TS 44.060/12.8.4/DLMC Frequency Parameters IE",
	"TS 44.060/12.8/Direct encoding 1 struct",
	"TS 44.060/12.8/Frequency Parameters IE",
}

func TestUnresolvedCarrierLimitsMatchREADME(t *testing.T) {
	data, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(data), "<!-- unresolved-carrier-limits -->\n")
	if !ok {
		t.Fatal("missing unresolved carrier table")
	}
	section, _, ok = strings.Cut(section, "<!-- /unresolved-carrier-limits -->")
	if !ok {
		t.Fatal("unterminated unresolved carrier table")
	}
	var got []string
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "| `") {
			parts := strings.Split(line, "`")
			got = append(got, parts[1])
		}
	}
	want := append([]string(nil), unresolvedCarrierLimits...)
	sort.Strings(want)
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("README unresolved carriers=%q want %q", got, want)
	}
}

// All descriptors participate, including those with nonzero MaxOctets.
// Successful explicit encoding at the runtime resource ceiling requires a named
// caller-bounded control component. Unresolved shared IE records do not authorize
// padding growth. A valid synthetic decode is required for each
// descriptor, so a failing sample search cannot silently hide a new gap.
func TestCanonicalRuntimeExtentRequiresUnresolvedCarrier(t *testing.T) {
	const limit = (1 << 20) / 8
	if err := runtime.CheckInput(make([]byte, limit)); err != nil {
		t.Fatal(err)
	}
	if err := runtime.CheckInput(make([]byte, limit+1)); err == nil {
		t.Fatal("runtime ceiling no longer matches the audit extent")
	}
	counts := map[string]int{}
	if len(Descriptors()) != 632 {
		t.Fatalf("catalogue has %d descriptors, want 632", len(Descriptors()))
	}
	for _, d := range Descriptors() {
		key := d.Standard + "/" + d.Clause + "/" + d.Name
		value, wire := paddingLimitSample(t, d)
		largestAccepted := 0
		for _, n := range []int{1, 2, 3, 4, 5, 8, 10, 16, 18, 20, 21, 22, 23, 24, 30, 32, 40, 50, 64, 100, 128, 200, 251, 253, 255, 256, 257, 300, 1000, 20000} {
			if _, err := d.CanonicalAtLength(value, n); err == nil {
				largestAccepted = n
			}
		}
		out, err := d.CanonicalAtLength(value, limit)
		class := "exact-or-restricted"
		if err == nil {
			class = "unresolved-padding"
			largestAccepted = limit
			if len(out) != limit {
				t.Errorf("%s: accepted extent produced %d octets", key, len(out))
			}
			if callerBoundedControlComponent(d) {
				// TS 44.060 V19.0.0 §11 paragraph 12937 supplies the
				// message extent externally. These are content/rule roots;
				// MAC framing and the selector are outside their codecs.
				_, contextErr := d.Canonical(value)
				var required *runtime.ExtentError
				if !errors.As(contextErr, &required) || !required.Required {
					t.Errorf("%s: missing caller extent preflight: %v", key, contextErr)
				}
				class = "caller-bounded-control"
			} else {
				t.Errorf("%s: grows by padding without a numerical carrier record or caller-bounded classification", key)
			}
		} else {
			var extent *runtime.ExtentError
			if errors.As(err, &extent) && extent.Maximum > 0 && extent.Maximum < limit && extent.Actual == limit {
				class = "source-capped"
			}
			if d.DecodeWithContext != nil {
				class = "requires-ACS"
			}
		}
		counts[class]++
		t.Logf("%s\t%s\tlargest-ladder-extent=%d\tsample=%x", class, key, largestAccepted, wire)
	}
	if counts["unresolved-padding"] != 0 {
		t.Errorf("unresolved-padding class must be empty: %d", counts["unresolved-padding"])
	}
	t.Logf("classification: %v; descriptors=%d", counts, len(Descriptors()))
}

func paddingLimitSample(t *testing.T, d runtime.Descriptor) (any, []byte) {
	t.Helper()
	decode := d.Decode
	if d.DecodeWithContext != nil {
		decode = func(wire []byte) (any, error) { return d.DecodeWithContext(wire, runtime.SI4ACSZero) }
	}
	if d.Standard == "TS 44.018" && d.Clause == "10.5.2.16" && d.Name == "Frequency Parameters, before time" {
		// TS 44.018 V19.0.0 §10.5.2.16 supplies this preceding field.
		r := runtime.NewReader(nil)
		r.Set("Length of frequency parameters", 0)
		if _, err := r.PushLimit(0); err != nil {
			t.Fatal(err)
		}
		v, err := d.DecodeFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		return v, nil
	}
	seeds := [][]byte{{0x33, 3, 0, 0, 0, 0}, {0x20, 0}}
	if d.Context != "" {
		// The access rows contain fixed prefix and suffix bits around the body.
		// Try zero-valued fields with each printed prefix and suffix alternative.
		for _, prefix := range []byte{0, 0x40, 0x80} {
			for _, suffix := range []byte{0, 0x20} {
				seeds = append(seeds, []byte{prefix, suffix}, []byte{prefix, suffix, 0, 0})
			}
		}
	}
	for _, encoded := range canonicalSeeds[d.Clause+"\x00"+d.Name] {
		wire, err := hex.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		seeds = append(seeds, wire)
	}
	for _, wire := range seeds {
		if v, err := decode(wire); err == nil {
			return v, wire
		}
	}
	for _, fill := range []byte{0x2b, 0, 0xff, 0x55, 0x80, 0x40} {
		for n := 0; n <= 64; n++ {
			wire := bytes.Repeat([]byte{fill}, n)
			if v, err := decode(wire); err == nil {
				return v, wire
			}
		}
	}
	for n := 1; n <= 64; n++ {
		for first := 0; first < 256; first++ {
			wire := make([]byte, n)
			wire[0] = byte(first)
			if v, err := decode(wire); err == nil {
				return v, wire
			}
		}
	}
	t.Fatalf("%s §%s %s: missing accepted synthetic sample", d.Standard, d.Clause, d.Name)
	return nil, nil
}

func TestMBMSSessionIdentityContainingExtent(t *testing.T) {
	d, err := Lookup("TS 44.018", "MBMS Session Identity value part")
	if err != nil {
		t.Fatal(err)
	}
	if d.MaxOctets != 0 || d.Extent.Finite {
		t.Fatalf("standalone extent changed: %+v", d.Extent)
	}
	// TS 44.018 V19.0.0 §10.5.2.62 p21347 and table 10.5.2.62.1:
	// two-octet type 3 IE, one IEI octet and an eight-bit session identity.
	for _, wire := range [][]byte{{0}, {0xff}, {0x42, 0xff, 0x2b}} {
		v, err := d.Decode(wire)
		if err != nil {
			t.Fatal(err)
		}
		replay, err := d.Encode(v)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("replay %x: %x %v", wire, replay, err)
		}
		out, err := d.CanonicalAtLength(v, 1)
		if err != nil || len(out) != 1 {
			t.Fatalf("at cap: %x %v", out, err)
		}
		out, err = d.CanonicalAtLength(v, 2)
		var extent *runtime.ExtentError
		if out != nil || !errors.As(err, &extent) || extent.Maximum != 1 {
			t.Fatalf("past cap: %x %v", out, err)
		}
	}
}

func TestPaddingLimitPackageDocumentation(t *testing.T) {
	for _, tc := range []struct {
		path    string
		phrases []string
	}{
		{"../ts44018/ies/doc.go", []string{"Relating that length field to the total extent is unresolved", "canonical explicit lengths are capped at one octet"}},
		{"../ts44018/messages/doc.go", []string{"authenticated EC-BCCH carrier maximum is 23 octets", "§9.1.47 defines Uplink free. The containing message supplies the exact bit extent; the pinned main-DCCH Bter maximum is 23 octets (TS 44.006 V19.0.0 §8.8.3)", "§9.1.49b defines VGCS SMS Information. The containing message supplies the exact bit extent; the pinned SACCH Bter maximum is 21 octets (TS 44.006 V19.0.0 §8.8.3)", "§9.1.57 defines VGCS Neighbour Cell Information message. The containing message supplies the exact bit extent; the pinned main-DCCH Bter maximum is 23 octets (TS 44.006 V19.0.0 §8.8.3)"}},
		{"../ts44018/notification/doc.go", []string{"§9.1.21a defines Notification/FACCH. The containing message supplies the exact bit extent; the pinned main-DCCH Bter maximum is 23 octets (TS 44.006 V19.0.0 §8.8.3)"}},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, phrase := range tc.phrases {
			if !strings.Contains(string(data), phrase) {
				t.Errorf("%s omits %q", tc.path, phrase)
			}
		}
	}
}

func callerBoundedControlComponent(d runtime.Descriptor) bool {
	if d.Standard != "TS 44.060" || d.Extent.Finite || !d.Extent.NeedsContext || d.MaxOctets != 0 {
		return false
	}
	switch d.Clause + "/" + d.Name {
	// TS 44.060 V19.0.0 §11.2.26 uses the containing extent.
	case "11.2.26/Packet TBF Release message content":
		return true
	// TS 44.060 V19.0.0: exact source-selected containing bounds.
	case "11.2.28b/Packet DBPSCH Uplink Ack/Nack message content", "11.2.9b/Packet Measurement Order message content":
		return true

	// TS 44.060 V19.0.0 §11.2.39 excludes the external framing and selector.
	case "11.2.39/MBMS Assignment Non-distribution message content":
		return true
	// TS 44.060 V19.0.0: these source-selected definitions require their containing extent.
	case "11.2.18/PSI1 message content", "11.2.19/PSI2 message content", "11.2.21a/PSI3 ter message content", "11.2.25/PSI13 message content", "11.2.25a/PSI14 message content", "11.2.25b/PSI15 message content", "11.2.21b/PSI3 quater message content", "11.2.38/MBMS service request message content", "11.2.6/Packet Downlink Ack/Nack message content", "11.2.6b/Packet DBPSCH Downlink Ack/Nack message", "11.2.6c/Packet DBPSCH Downlink Ack/Nack Type 2 message", "11.2.28a/Packet DBPSCH Uplink Ack/Nack message content":
		return true
	// TS 44.060 V19.0.0 §11 p12937 and each source-selected content root.
	case "11.2.3/Packet Cell Change Failure message content", "11.2.3a/Packet Cell Change Notification message content", "11.2.5b/Packet DBPSCH Assignment message content", "11.2.13/Packet Power Control/Timing Advance message content", "11.2.32/Additional MS Radio Access Capabilities message content", "11.2.34/Physical information message content", "11.2.45/Packet Physical information message content":
		return true
	case "11.2.2/Packet Control Acknowledgement message content", "11.2.20/PSI3 message content", "11.2.24/PSI8 message content", "11.2.9c/Packet Mobile TBF Status message content", "11.2.12/Packet Polling Request message content", "11.2.17/Packet PSI Status message content", "11.2.17b/Packet SI Status message content", "11.2.9/Packet Measurement Report message content", "11.2.1/Packet Access Reject message content", "11.2.2a/Packet Cell Change Continue message content", "11.2.8b/Packet Uplink Dummy Control Block message content", "11.2.11/Packet PDCH Release message content", "11.2.14/Packet PRACH Parameters message content", "11.2.15/Packet Queueing Notification message content", "12.43/NAS Container for PS Handover IE", "12.47/CS Handover Radio Resources IE", "11.2.30a/Packet pause message content", "11.2.35/Packet CS Request message content", "11/padding bits", "11.2.8/Packet Downlink Dummy Control Block message content":
		return true
	}
	return false
}

func TestCallerBoundedControlComponentScope(t *testing.T) {
	d := runtime.Descriptor{Standard: "TS 44.060", Clause: "11", Name: "padding bits", Extent: runtime.Extent{NeedsContext: true}}
	if !callerBoundedControlComponent(d) {
		t.Fatal("padding rule must require its caller's extent")
	}
	for _, change := range []func(*runtime.Descriptor){
		func(d *runtime.Descriptor) { d.Standard = "TS 44.018" },
		func(d *runtime.Descriptor) { d.Name = "unrecorded" },
		func(d *runtime.Descriptor) { d.Clause = "12.36" },
		func(d *runtime.Descriptor) { d.Extent.NeedsContext = false },
		func(d *runtime.Descriptor) { d.Extent.Finite = true },
		func(d *runtime.Descriptor) { d.MaxOctets = 23 },
	} {
		candidate := d
		change(&candidate)
		if callerBoundedControlComponent(candidate) {
			t.Fatalf("unrelated component accepted: %+v", candidate)
		}
	}
}
