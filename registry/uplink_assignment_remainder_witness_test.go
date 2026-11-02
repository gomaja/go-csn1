package registry_test

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	subject "github.com/gomaja/go-csn1/ts44060/packetuplinkassignment"
)

// TS 44.060 V19.0.0 §11.2.29 p18405; §11.1 p12944. The four
// generator families each have an executed witness, rather than an exemption.
func uplinkAssignmentRemainderErrorFamilies(t *testing.T) map[string]bool {
	t.Helper()
	decoded, e := subject.DecodeDLMCULCarrierInfoStruct([]byte{0xd0})
	if e != nil {
		t.Fatal(e)
	}
	base := decoded.Value
	changed := base
	choice := changed.UPLINKTIMESLOTALLOCATIONChoice
	arm := *choice.Alt11
	arm.Ignored.BitLength--
	choice.Alt11 = &arm
	changed.UPLINKTIMESLOTALLOCATIONChoice = choice
	fresh := base
	fresh.Wire = runtime.WireInfo{}
	w := runtime.NewWriter()
	w.WithWire(base.Wire)
	if _, e = w.PushLimit(7); e != nil {
		t.Fatal(e)
	}
	cases := map[string]func() error{
		"reserved remainder has no sender encoding": func() error {
			cw := runtime.NewWriter()
			cw.WithWire(runtime.CanonicalTo(base, 8).Wire)
			return subject.EncodeDLMCULCarrierInfoStructTo(cw, runtime.Canonical(base))
		},
		"reserved remainder requires received wire":          func() error { return subject.EncodeDLMCULCarrierInfoStructTo(runtime.NewWriter(), fresh) },
		"reserved remainder width changed":                   func() error { return subject.EncodeDLMCULCarrierInfoStructTo(runtime.NewWriter(), changed) },
		"reserved remainder must fill its containing extent": func() error { return subject.EncodeDLMCULCarrierInfoStructTo(w, fresh) },
	}
	out := map[string]bool{}
	for family, run := range cases {
		e := run()
		found := false
		for e != nil {
			if e.Error() == family {
				found = true
				break
			}
			e = errors.Unwrap(e)
		}
		if !found {
			t.Fatalf("missing executed family %q", family)
		}
		out[family] = true
	}
	return out
}
func TestUplinkAssignmentRemainderErrorFamilies(t *testing.T) {
	uplinkAssignmentRemainderErrorFamilies(t)
}
