package registry_test

import (
	"errors"
	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

func TestReportEntryPoints(t *testing.T) {
	if _, report, err := registry.DecodeWithReport("TS 24.008", "Mobile Station Classmark 2 value part", []byte{0x20, 0, 0}); err != nil || len(report.Observations) != 0 {
		t.Fatalf("report=%+v error=%v", report, err)
	}
	if _, report, err := registry.DecodeWithReport("TS 24.008", "not a definition", nil); err == nil || len(report.Observations) != 0 {
		t.Fatalf("lookup report=%+v error=%v", report, err)
	}
	if _, report, err := registry.DecodeWithReportWithContext("TS 24.008", "Single Band Support", []byte{0xf0}, runtime.SI4ACSOne); !errors.Is(err, runtime.ErrUnsupported) || len(report.Observations) != 0 {
		t.Fatalf("unsupported context report=%+v error=%v", report, err)
	}

}
