package messages

import (
	"bytes"
	"testing"

	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// TS 44.018 V19.0.0 §9.1.50a, DOCX paragraphs 10646–10650.
func TestSI10bisFixedCanonicalCarrier(t *testing.T) {
	value := SYSTEMINFORMATIONTYPE10bis{MessageType: 10, SI10bisRestOctets: restoctets.SI10bisRestOctets{}}
	wire := append([]byte{0x28}, bytes.Repeat([]byte{0x2b}, 20)...)
	checkSI10FixedCarrier(t, "9.1.50a", "SYSTEM INFORMATION TYPE 10bis", value, wire)
	for sequence := uint8(1); sequence < 4; sequence++ {
		value.SI10bisRestOctets.SI10bisSequence = sequence
		wire[1] = (wire[1] & 0x3f) | (sequence << 6)
		checkSI10FixedCarrier(t, "9.1.50a", "SYSTEM INFORMATION TYPE 10bis", value, wire)
	}
	value.SI10bisRestOctets.SI10bisSequence = 4
	if _, err := EncodeSYSTEMINFORMATIONTYPE10bisCanonical(value); err == nil {
		t.Fatal("invalid sequence accepted")
	}
}
