package registry_test

import (
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	subject "github.com/gomaja/go-csn1/ts44060/pshandoverradioresources3"
)

// TS 44.060 V19.0.0 §12.42b p24022; §11.1 p12944. The four
// generator families each have an executed witness, rather than an exemption.
func reservedRemainderErrorFamilies(t *testing.T) map[string]bool {
	t.Helper()
	decoded, e := subject.DecodeUFPSStruct([]byte{0xd0})
	if e != nil {
		t.Fatal(e)
	}
	base := decoded.Value
	changed := base
	choice := changed.DLMCFrequencyParametersChoice
	arm := *choice.Alt11
	arm.Ignored.BitLength--
	choice.Alt11 = &arm
	changed.DLMCFrequencyParametersChoice = choice
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
			return subject.EncodeUFPSStructTo(cw, runtime.Canonical(base))
		},
		"reserved remainder requires received wire":          func() error { return subject.EncodeUFPSStructTo(runtime.NewWriter(), fresh) },
		"reserved remainder width changed":                   func() error { return subject.EncodeUFPSStructTo(runtime.NewWriter(), changed) },
		"reserved remainder must fill its containing extent": func() error { return subject.EncodeUFPSStructTo(w, fresh) },
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
func TestReservedRemainderErrorFamilies(t *testing.T) { reservedRemainderErrorFamilies(t) }
