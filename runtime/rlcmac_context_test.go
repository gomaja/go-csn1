package runtime

import (
	"errors"
	"fmt"
	"testing"
)

func TestRLCMACContextChannels(t *testing.T) {
	// TS 44.060 V19.0.0 §§11.2.6b–c, 11.2.28a–b, 11.2.33–34
	// give the DBPSCH carrier union. Message-specific restrictions belong
	// to the resolver, including FLO for the ADCH acknowledgement variants.
	channels := []struct {
		name    string
		channel RLCMACChannel
		valid   [3]bool // ordinary, EC, DBPSCH
	}{
		{"PACCH", RLCMACPACCH, [3]bool{true, true, true}},
		{"PBCCH", RLCMACPBCCH, [3]bool{true, false, false}},
		{"PCCCH", RLCMACPCCCH, [3]bool{true, false, false}},
		{"FACCH", RLCMACFACCH, [3]bool{true, false, true}},
		{"SACCH", RLCMACSACCH, [3]bool{true, false, true}},
		{"SDCCH", RLCMACSDCCH, [3]bool{true, false, true}},
		{"ADCH", RLCMACADCH, [3]bool{true, false, true}},
	}
	for _, ch := range channels {
		for i, mode := range []RLCMACMode{RLCMACOrdinary, RLCMACEC, RLCMACDBPSCH} {
			for _, direction := range []RLCMACDirection{RLCMACDownlink, RLCMACUplink} {
				t.Run(fmt.Sprintf("%s/mode=%d/direction=%d", ch.name, mode, direction), func(t *testing.T) {
					err := (RLCMACContext{Direction: direction, Mode: mode, Channel: ch.channel}).Validate()
					if (err == nil) != ch.valid[i] {
						t.Fatalf("Validate() = %v, want valid=%t", err, ch.valid[i])
					}
					if err != nil {
						var decode *DecodeError
						if !errors.As(err, &decode) || decode.Kind != InvalidValue {
							t.Fatalf("incompatible channel: %v", err)
						}
					}
				})
			}
		}
	}
}

func TestRLCMACContextCoordinatesAndPhysicalFacts(t *testing.T) {
	for _, mode := range []RLCMACMode{RLCMACOrdinary, RLCMACEC, RLCMACDBPSCH} {
		for _, direction := range []RLCMACDirection{RLCMACDownlink, RLCMACUplink} {
			base := RLCMACContext{Direction: direction, Mode: mode, Channel: RLCMACPACCH}
			for _, coverage := range []uint8{0, 1, 2, 3, 4, 5, 6, 255} {
				for _, coding := range []RLCMACCodingScheme{0, RLCMACCS1, RLCMACCS3, RLCMACMCS0, RLCMACECCoding, 5, 255} {
					c := base
					c.CoverageClass, c.CodingScheme = &coverage, coding
					want := coverage >= 1 && coverage <= 5 && coding <= RLCMACECCoding
					if err := c.Validate(); (err == nil) != want {
						t.Errorf("mode=%d direction=%d coverage=%d coding=%d: %v, want valid=%t", mode, direction, coverage, coding, err, want)
					}
				}
			}
			for _, invalid := range []uint8{0, 8, 255} {
				c := base
				c.Channel = RLCMACChannel(invalid)
				requireRLCMACCoordinates(t, c)
			}
			for _, invalid := range []uint8{0, 4, 255} {
				c := base
				c.Mode = RLCMACMode(invalid)
				requireRLCMACCoordinates(t, c)
				c = base
				c.Direction = RLCMACDirection(invalid)
				requireRLCMACCoordinates(t, c)
			}
		}
	}
}

func requireRLCMACCoordinates(t *testing.T, c RLCMACContext) {
	t.Helper()
	var required *ContextRequiredError
	if err := c.Validate(); !errors.As(err, &required) {
		t.Fatalf("invalid coordinates %+v: %v", c, err)
	}
}

func FuzzRLCMACContextValidate(f *testing.F) {
	for _, channel := range []uint8{1, 2, 3, 4, 5, 6, 7} {
		f.Add(uint8(3), uint8(1), channel, uint8(1), uint8(0), false)
		f.Add(uint8(3), uint8(2), channel, uint8(5), uint8(4), true)
	}
	f.Fuzz(func(t *testing.T, mode, direction, channel, coverage, coding uint8, hasCoverage bool) {
		c := RLCMACContext{Mode: RLCMACMode(mode), Direction: RLCMACDirection(direction), Channel: RLCMACChannel(channel), CodingScheme: RLCMACCodingScheme(coding)}
		if hasCoverage {
			c.CoverageClass = &coverage
		}
		// Rows are modes 1–3, columns are channels 1–7, as in the source-derived
		// channel test. Zero coordinates and out-of-range values are invalid.
		allowed := [3][7]bool{
			{true, true, true, true, true, true, true},
			{true, false, false, false, false, false, false},
			{true, false, false, true, true, true, true},
		}
		want := mode >= 1 && mode <= 3 && direction >= 1 && direction <= 2 && channel >= 1 && channel <= 7 && coding <= 4 && (!hasCoverage || coverage >= 1 && coverage <= 5)
		if want {
			want = allowed[mode-1][channel-1]
		}
		if err := c.Validate(); (err == nil) != want {
			t.Fatalf("Validate(%+v) = %v, coverage=%d, want valid=%t", c, err, coverage, want)
		}
	})
}
