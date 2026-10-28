package registry_test

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

func compareEntryResult(t *testing.T, form string, want any, wantErr error, got any, gotErr error) {
	t.Helper()
	if fmt.Sprint(wantErr) != fmt.Sprint(gotErr) {
		t.Fatalf("%s error: got %v, want %v", form, gotErr, wantErr)
	}
	// Error returns do not carry a decoded value; implementations may return nil
	// or their typed zero value through the descriptor's any slot.
	if wantErr == nil && !reflect.DeepEqual(want, got) {
		t.Fatalf("%s value differs:\n got %#v\nwant %#v", form, got, want)
	}
}

func compareDescriptorEntries(t *testing.T, d runtime.Descriptor, data []byte) {
	t.Helper()
	want, wantErr := d.Decode(data)
	got, _, err := d.DecodeWithReport(data)
	compareEntryResult(t, "report", want, wantErr, got, err)
	got, err = d.DecodeWithCollector(data, nil)
	compareEntryResult(t, "nil collector", want, wantErr, got, err)
	// The hand-written descriptors are the reader parity contract here.
	// Generated descriptors have separate reader framing/error conventions;
	// TestReaderEntryMatchesByteEntry covers their received byte behavior.
	if d.Name == "Mobile Station Classmark 2 value part" || d.Name == "geran-cs" || d.Name == "geran-ps" {
		got, err = d.DecodeFrom(readerBoundTo(t, data))
		compareEntryResult(t, "reader", want, wantErr, got, err)
	}
	if d.DecodeWithContext == nil {
		return
	}
	for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
		want, wantErr = d.DecodeWithContext(data, acs)
		got, _, err = d.DecodeWithReportWithContext(data, acs)
		compareEntryResult(t, fmt.Sprintf("report ACS %d", acs), want, wantErr, got, err)
		got, err = d.DecodeWithCollectorWithContext(data, nil, acs)
		compareEntryResult(t, fmt.Sprintf("nil collector ACS %d", acs), want, wantErr, got, err)
	}
}

func TestManualDecodeReportParity(t *testing.T) {
	for _, tc := range []struct{ standard, name, input string }{
		{"TS 24.008", "Mobile Station Classmark 2 value part", "fe51f6"},
		{"TS 24.008", "Mobile Station Classmark 2 value part", "6f"},
		{"TS 36.331", "geran-cs", "3303a930a49be5"},
		{"TS 36.331", "geran-cs", "20"},
		{"TS 36.331", "geran-ps", "c088"},
		{"TS 36.331", "geran-ps", "dc"},
	} {
		t.Run(tc.name+"/"+tc.input, func(t *testing.T) {
			d, err := registry.Lookup(tc.standard, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			data, err := hex.DecodeString(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			compareDescriptorEntries(t, d, data)
			want, wantErr := registry.Decode(tc.standard, tc.name, data)
			got, _, err := registry.DecodeWithReport(tc.standard, tc.name, data)
			compareEntryResult(t, "registry report", want, wantErr, got, err)
		})
	}
}

func TestCatalogueDecodeReportParity(t *testing.T) {
	rng := rand.New(rand.NewSource(20261027))
	for _, d := range registry.Descriptors() {
		t.Run(d.Standard+"/"+d.Name, func(t *testing.T) {
			// Match the issue's synthetic sweep: zero through eight random octets.
			// The extent and reader parity suites exercise canonical ceilings separately.
			for range 2000 {
				data := make([]byte, rng.Intn(9))
				_, _ = rng.Read(data)
				compareDescriptorEntries(t, d, data)
			}
		})
	}
}

func TestManualFramedDecodeReportParity(t *testing.T) {
	rng := rand.New(rand.NewSource(20261028))
	for _, name := range []string{"geran-cs", "geran-ps"} {
		t.Run(name, func(t *testing.T) {
			d, err := registry.Lookup("TS 36.331", name)
			if err != nil {
				t.Fatal(err)
			}
			for range 2000 {
				data := make([]byte, 6+rng.Intn(55))
				_, _ = rng.Read(data)
				if name == "geran-cs" {
					data[0], data[1] = 0x33, 3
				}
				compareDescriptorEntries(t, d, data)
			}
		})
	}
}
