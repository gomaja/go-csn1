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
	"TS 44.018/10.5.2.1c/BA List Pref",
	"TS 44.018/9.1.43p/EC System Information Type 1",
	"TS 44.018/9.1.43r/EC System Information Type 3",
	"TS 44.018/9.1.43s/EC System Information Type 4",
	"TS 44.018/9.1.58/NOTIFY APPLICATION DATA",
	"TS 44.018/9.1.62/EC-Dummy message content",
	"TS 44.060/12.58/PSC Group IE",
	"TS 44.060/12.36/MBMS Channel Parameters IE",
	"TS 44.060/12.37/MBMS p-t-m channel description IE",
	"TS 44.060/12.38/MPRACH description IE",
	"TS 44.060/12.39/MBMS Session List IE",
	"TS 44.060/12.40/MBMS Session Parameters List IE",
	"TS 44.060/12.8/Frequency Parameters IE",
	"TS 44.060/12.8/Direct encoding 1 struct",
	"TS 44.060/12.10a/GPRS Mobile Allocation IE",
	"TS 44.060/12.10a/RFL number list struct",
	"TS 44.060/12.10a/ARFCN index list struct",
	"TS 44.060/12.57/PCID Group IE",
	"TS 44.060/12.59/Enhanced Cell Reselection Parameters IE",
	"TS 44.060/12.59/Repeated E-UTRAN Enhanced Cell Reselection Parameters struct",
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
// Successful explicit encoding at the runtime resource ceiling must have an
// unresolved carrier record. A valid synthetic decode is required for each
// descriptor, so a failing sample search cannot silently hide a new gap.
func TestCanonicalRuntimeExtentRequiresUnresolvedCarrier(t *testing.T) {
	unresolved := map[string]bool{}
	for _, key := range unresolvedCarrierLimits {
		unresolved[key] = true
	}
	const limit = (1 << 20) / 8
	if err := runtime.CheckInput(make([]byte, limit)); err != nil {
		t.Fatal(err)
	}
	if err := runtime.CheckInput(make([]byte, limit+1)); err == nil {
		t.Fatal("runtime ceiling no longer matches the audit extent")
	}
	counts := map[string]int{}
	if len(Descriptors()) != 317 {
		t.Fatalf("catalogue has %d descriptors, want 317", len(Descriptors()))
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
			if !unresolved[key] {
				t.Errorf("%s: accepts runtime extent without unresolved carrier record", key)
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
		{"../ts44018/messages/doc.go", []string{"no numerical EC-BCCH carrier maximum is established", "no numerical maximum is recorded from the pinned source"}},
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
