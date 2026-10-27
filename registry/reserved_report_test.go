package registry_test

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
)

func TestReservedSingleBandReport(t *testing.T) {
	d, err := registry.Lookup("TS 24.008", "Single Band Support")
	if err != nil {
		t.Fatal(err)
	}
	value, report, err := d.DecodeWithReport([]byte{0xf0})
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := value.(runtime.Decoded[classmark.SingleBandSupport])
	if !ok || decoded.Value.GSMBand != 15 {
		t.Fatalf("value=%+v", value)
	}
	expected := runtime.Observation{Location: runtime.Location{Path: "SingleBandSupport/SingleBandSupportGSMBand", BitOffset: 0, Width: 4, Source: runtime.Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.7", Table: "10.5.1.7"}}, Value: 15}
	if len(report.Observations) != 1 || report.Observations[0] != expected {
		t.Fatalf("report=%+v", report)
	}
	replay, err := d.Encode(value)
	if err != nil || !bytes.Equal(replay, []byte{0xf0}) {
		t.Fatalf("replay=%x error=%v", replay, err)
	}
	if _, err := d.Canonical(value); err == nil {
		t.Fatal("canonical sender accepted reserved band")
	}
	// TS 24.008 V20.1.0 table 10.5.1.7: 0..9 are allocated,
	// 10..15 reserved. Receipt remains tolerant at both range boundaries.
	for code := byte(0); code < 16; code++ {
		_, report, err := d.DecodeWithReport([]byte{code << 4})
		if err != nil {
			t.Fatal(err)
		}
		want := 0
		if code > 9 {
			want = 1
		}
		if len(report.Observations) != want {
			t.Fatalf("band %d: %+v", code, report)
		}
	}
}

func TestClassmark2ReservedReport(t *testing.T) {
	// TS 24.008 V20.1.0 table 10.5.6a: Revision 0/3 and RF 5/6.
	// TS 24.080 V19.4.0 §3.7.1: SS screening 2/3 interpreted as 01.
	_, report, err := registry.DecodeWithReport("TS 24.008", "Mobile Station Classmark 2 value part", []byte{0x65, 0x20, 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Observations) != 3 {
		t.Fatalf("report=%+v", report)
	}
	for i, want := range []struct {
		path                             string
		offset, width                    int
		value                            uint64
		standard, version, clause, table string
	}{
		{"RevisionLevel", 1, 2, 3, "TS 24.008", "20.1.0", "10.5.1.6", "10.5.6a"},
		{"RFPowerCapability", 5, 3, 5, "TS 24.008", "20.1.0", "10.5.1.6", "10.5.6a"},
		{"SSScreeningIndicator", 10, 2, 2, "TS 24.080", "19.4.0", "3.7.1", ""},
	} {
		got := report.Observations[i]
		if got.Value != want.value || got.Location != (runtime.Location{Path: "Classmark2ValuePart/" + want.path, BitOffset: want.offset, Width: want.width, Source: runtime.Source{Standard: want.standard, Version: want.version, Clause: want.clause, Table: want.table}}) {
			t.Fatalf("event %d=%+v", i, got)
		}
	}
	for _, wire := range [][]byte{{0x20, 0x10, 0}, {0x27, 0x10, 0}} {
		if _, report, err := registry.DecodeWithReport("TS 24.008", "Mobile Station Classmark 2 value part", wire); err != nil || len(report.Observations) != 0 {
			t.Fatalf("defined/context-dependent codes: report=%+v error=%v", report, err)
		}
	}
}

func TestNestedGERANCSObservationOrigin(t *testing.T) {
	// Fresh source-shaped value: the zero-valued prefix takes 24 bits before
	// GSM Band (TS 24.008 V20.1.0 §10.5.1.7). GERAN-CS adds 40 framing bits
	// once (TS 36.331 V19.4.0 §6.3.6 UE-CapabilityRAT-ContainerList).
	value := classmark.Classmark3ValuePart{MultibandSupportedChoice: classmark.Classmark3ValuePartMultibandSupportedChoice{A5Bits: &classmark.Classmark3ValuePartMultibandSupportedChoiceA5Bits{}}, SingleBandSupport: &classmark.SingleBandSupport{GSMBand: 15}}
	raw, err := uecapability.EncodeGERANCS(uecapability.GERANCS{Classmark2: uecapability.Classmark2ValuePart{RevisionLevel: 1}, Classmark3: value})
	if err != nil {
		t.Fatal(err)
	}
	_, report, err := registry.DecodeWithReport("TS 36.331", "geran-cs", raw)
	if err != nil {
		t.Fatal(err)
	}
	expected := runtime.Observation{Location: runtime.Location{Path: "GERANCS/Classmark3ValuePart/SingleBandSupport/SingleBandSupportGSMBand", BitOffset: 64, Width: 4, Source: runtime.Source{Standard: "TS 24.008", Version: "20.1.0", Clause: "10.5.1.7", Table: "10.5.1.7"}}, Value: 15}
	if len(report.Observations) != 1 || report.Observations[0] != expected {
		t.Fatalf("report=%+v", report)
	}
	var c runtime.Collector
	r := runtime.NewReaderWithCollector(append([]byte{0}, raw...), &c)
	if _, err := r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("Envelope"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(len(raw) * 8); err != nil {
		t.Fatal(err)
	}
	if _, err := uecapability.DecodeGERANCSFrom(r); err != nil {
		t.Fatal(err)
	}
	expected.Location.BitOffset = 72
	expected.Location.Path = "Envelope/" + expected.Location.Path
	if report := c.TakeReport(); len(report.Observations) != 1 || report.Observations[0] != expected {
		t.Fatalf("containing origin=%+v", report)
	}
}

func TestClassmark2ReportCodeBoundaries(t *testing.T) {
	for revision := byte(0); revision < 4; revision++ {
		for power := byte(0); power < 8; power++ {
			for screening := byte(0); screening < 4; screening++ {
				wire := []byte{revision<<5 | power, screening << 4, 0}
				value, report, err := registry.DecodeWithReport("TS 24.008", "Mobile Station Classmark 2 value part", wire)
				if err != nil {
					t.Fatal(err)
				}
				want := 0
				if revision == 0 || revision == 3 {
					want++
				}
				if power == 5 || power == 6 {
					want++
				}
				if screening >= 2 {
					want++
				}
				if len(report.Observations) != want {
					t.Fatalf("revision=%d power=%d screening=%d observations=%d want %d", revision, power, screening, len(report.Observations), want)
				}
				d, err := registry.Lookup("TS 24.008", "Mobile Station Classmark 2 value part")
				if err != nil {
					t.Fatal(err)
				}
				replay, err := d.Encode(value)
				if err != nil || !bytes.Equal(replay, wire) {
					t.Fatalf("Classmark 2 replay changed: %v", err)
				}
			}
		}
	}
}

func TestReservedReportLargeReaderAccepts(t *testing.T) {
	const fields = 32768
	wire := bytes.Repeat([]byte{0xff}, fields/2)
	for _, enabled := range []bool{false, true} {
		var c runtime.Collector
		var collector *runtime.Collector
		if enabled {
			collector = &c
		}
		r := runtime.NewReaderWithCollector(wire, collector)
		for i := 0; i < fields; i++ {
			value, err := classmark.DecodeSingleBandSupportFrom(r)
			if err != nil || value.GSMBand != 15 {
				t.Fatalf("enabled=%t field=%d value=%d error=%v", enabled, i, value.GSMBand, err)
			}
		}
		report := c.TakeReport()
		want := 0
		if enabled {
			want = fields
		}
		if r.Position() != fields*4 || len(report.Observations) != want || report.Dropped != 0 {
			t.Fatalf("enabled=%t position=%d events=%d dropped=%d", enabled, r.Position(), len(report.Observations), report.Dropped)
		}
	}
}
