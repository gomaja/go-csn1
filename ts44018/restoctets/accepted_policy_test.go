package restoctets

import (
	"bytes"
	"testing"
)

func TestNCOrderThreeRemainsAccepted(t *testing.T) {
	// TS 44.018 table 10.5.2.33b.2 delegates to TS 44.060 table 11.2.23.2.
	// SI13 table 10.5.2.37b.2 and PSI13 table 11.2.25.2 interpret code 3 as NC0.
	for code := uint8(0); code < 4; code++ {
		wire := []byte{code << 6}
		d, err := DecodeNCMeasurementParametersStruct(wire)
		if err != nil || d.Value.NETWORKCONTROLORDER != code {
			t.Fatalf("NC %d: %v", code, err)
		}
		plain, err := EncodeNCMeasurementParametersStruct(d.Value)
		if err != nil || !bytes.Equal(plain, wire) {
			t.Fatalf("NC %d plain %x: %v", code, plain, err)
		}
		canonical, err := EncodeNCMeasurementParametersStructCanonicalAtLength(d.Value, 1)
		if err != nil || !bytes.Equal(canonical, wire) {
			t.Fatalf("NC %d canonical %x: %v", code, canonical, err)
		}
	}
}
