package registry

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

func TestReservedBEPPeriodInSI13Extension(t *testing.T) {
	// TS 44.018 V19.0.0 table 10.5.2.37b.1 embeds GPRS Cell Options
	// (TS 44.060 V19.0.0 §12.24). Its delegated BEP_PERIOD table is
	// TS 45.008 V19.0.0 §10.2.3.2.1. These synthetic truncation fixtures
	// carry reserved codes; their controls change only BEP_PERIOD to 10.
	for _, tc := range []struct {
		name, reserved, control string
		code                    uint8
	}{
		{"Rel4", "ea15976ab36992bb4a4df259d63dd40957e08528", "ea15976ab36992ba4a4df259d63dd40957e08528", 11},
		{"Rel6", "dd0011a6fea7fbd89c88ffa33d38bfbdb253df4b", "dd0011a6fea7f5d89c88ffa33d38bfbdb253df4b", 13},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire, err := hex.DecodeString(tc.reserved)
			if err != nil {
				t.Fatal(err)
			}
			t.Run("decode", func(t *testing.T) {
				_, err := restoctets.DecodeSI13RestOctets(wire)
				var de *runtime.DecodeError
				if !errors.As(err, &de) || de.Kind != runtime.InvalidValue || de.Detail != "value reserved by source table" {
					t.Fatalf("reserved BEP_PERIOD: %v", err)
				}
			})
			control, err := hex.DecodeString(tc.control)
			if err != nil {
				t.Fatal(err)
			}
			for _, canonical := range []bool{false, true} {
				name := "plain"
				if canonical {
					name = "canonical"
				}
				t.Run(name, func(t *testing.T) {
					d, err := restoctets.DecodeSI13RestOctets(control)
					if err != nil {
						t.Fatal(err)
					}
					d.Value.BCCHCHANGEMARKGroup.RACChoice.RAC.GPRSCellOptions.ExtensionLengthGroup.Content.Known.Group.EGPRSPACKETCHANNELREQUESTGroup.BEPPERIOD = tc.code
					if canonical {
						_, err = restoctets.EncodeSI13RestOctetsCanonical(d.Value)
					} else {
						_, err = restoctets.EncodeSI13RestOctets(d.Value)
					}
					if err == nil || !strings.Contains(err.Error(), "value reserved by source table") {
						t.Fatalf("reserved BEP_PERIOD: %v", err)
					}
				})
			}
		})
	}
}
