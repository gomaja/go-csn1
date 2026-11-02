package controlmessages_test

import (
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
	"testing"
)

// TS 44.060 V19.0.0 §§11.2.9c p16053, 11.2.12 p16369 and
// 11.2.17 p16576 permit an earlier-release null suffix. That sender path
// cannot fill every caller extent merely because another arm has padding.
func TestContentCanonicalUnfilledExtent(t *testing.T) {
	for _, name := range []string{"Packet Mobile TBF Status message content", "Packet Polling Request message content", "Packet PSI Status message content"} {
		t.Run(name, func(t *testing.T) {
			d, err := controlmessages.Lookup(name)
			if err != nil {
				t.Fatal(err)
			}
			value, err := d.Decode([]byte{0, 0})
			if err != nil {
				t.Fatal(err)
			}
			if out, err := d.Encode(value); err != nil || len(out) != 2 || out[0] != 0 || out[1] != 0 {
				t.Fatalf("replay %x: %v", out, err)
			}
			out, err := d.CanonicalAtLength(value, 3)
			var extent *runtime.ExtentError
			if len(out) != 0 || !errors.As(err, &extent) || extent.Required || extent.Actual != 2 || extent.Minimum != 3 || extent.Maximum != 3 {
				t.Fatalf("unfilled target %x: %v (%+v)", out, err, extent)
			}
		})
	}
}
