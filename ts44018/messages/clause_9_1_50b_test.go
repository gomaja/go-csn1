package messages

import (
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// TS 44.018 V19.0.0 §9.1.50b, DOCX paragraphs 10660–10665.
func TestSI10terFixedCanonicalCarrier(t *testing.T) {
	value := SYSTEMINFORMATIONTYPE10ter{MessageType: 11, CellChannelDescription: runtime.BitString{Bytes: make([]byte, 16), BitLength: 128}, SI10terRestOctets: restoctets.SI10terRestOctets{PositionInSI5ListChoice: restoctets.SI10terRestOctetsPositionInSI5ListChoice{PositionInSI5List: &restoctets.SI10terRestOctetsPositionInSI5ListChoicePositionInSI5List{}}}}
	wire := append([]byte{0x2c}, make([]byte, 16)...)
	wire = append(wire, 0, 3, 0x2b, 0x2b)
	checkSI10FixedCarrier(t, "9.1.50b", "SYSTEM INFORMATION TYPE 10ter", value, wire)
	value.SI10terRestOctets.PositionInSI5ListChoice.Alternative = 255
	if _, err := EncodeSYSTEMINFORMATIONTYPE10terCanonical(value); err == nil {
		t.Fatal("invalid position selector accepted")
	}
	value.SI10terRestOctets.PositionInSI5ListChoice.Alternative = 0
	value.CellChannelDescription.BitLength = 127
	if _, err := EncodeSYSTEMINFORMATIONTYPE10terCanonical(value); err == nil {
		t.Fatal("short channel description accepted")
	}
}
