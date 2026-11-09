package ies

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§11.1.4.4, 12.24: the complete R99 component
// survives an extension ending inside Rel-4. The cut component is ignored.
func TestGPRSCellOptionsRetainsCompleteExtensionComponents(t *testing.T) {
	for _, bits := range []string{"1", "11", "110", "110101", "110101111", "1101011111"} {
		t.Run(bits, func(t *testing.T) {
			w := runtime.NewWriter()
			if err := w.WriteLiteral(bits); err != nil {
				t.Fatal(err)
			}
			raw, err := w.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			wire := gprsCellOptionsWithIgnored(raw)
			d, err := DecodeGPRSCellOptionsIE(wire)
			if err != nil {
				t.Fatal(err)
			}
			if d.Value.ExtensionLengthGroup == nil || d.Value.ExtensionLengthGroup.Content.Known == nil || d.Value.ExtensionLengthGroup.Content.Ignored != nil {
				t.Fatal("truncation discarded the complete prefix")
			}
			ext := d.Value.ExtensionLengthGroup.Content.Known
			if len(bits) < 9 {
				if !runtime.IsZero(ext.Group) {
					t.Fatal("partial R99 component was retained")
				}
			} else {
				r99 := ext.Group.EGPRSPACKETCHANNELREQUESTGroup
				if r99 == nil || r99.EGPRSPACKETCHANNELREQUEST != 1 || r99.BEPPERIOD != 5 || ext.Group.PFCFEATUREMODE != 1 || ext.Group.DTMSUPPORT != 1 || ext.Group.BSSPAGINGCOORDINATION != 1 {
					t.Fatal("complete R99 values were lost")
				}
			}
			if !runtime.IsZero(ext.Group2) || !runtime.IsZero(ext.Group3) || ext.REDUCEDLATENCYACCESS != 0 || ext.NMOIALTERNATE != 0 {
				t.Fatal("cut or later components retained values")
			}
			replay, err := EncodeGPRSCellOptionsIE(d.Value)
			if err != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("byte-exact replay: %v", err)
			}
		})
	}
}
