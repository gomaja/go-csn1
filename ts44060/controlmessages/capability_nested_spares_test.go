package controlmessages_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

// TS 44.060 V19.0.0 §11.2.32 p19914 selects §12.30 p23547's
// all-spare sender rule, including the nested TS 24.008 declared length.
func TestAdditionalCapabilitiesNestedSpareProfile(t *testing.T) {
	for address := 0; address < 2; address++ {
		for _, spares := range []int{1, 4} {
			addr := "0010101"
			if address == 1 {
				addr = fmt.Sprintf("1%032b", uint32(0x10203040))
			}
			cap := func(n int) string {
				return "0001" + fmt.Sprintf("%07b", 73+n) + "001" + strings.Repeat("0", 70) + strings.Repeat("1", n) + "0"
			}
			prefix := "01000000001011" + addr
			raw := simpleControlBits(prefix+cap(spares)+"10", 184)
			want := simpleControlBits(prefix+cap(0)+"10", 184)
			r, _ := contentReader(t, raw, 184, runtime.RLCMACUplink, 11)
			value, err := controlmessages.DecodeAdditionalMSRadioAccessCapabilitiesMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, value)
			})
			out := simpleControlCanonical(t, raw, 14, 184, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, runtime.Canonical(value))
			})
			if !bytes.Equal(out.Bytes, want) {
				t.Fatal("nested spares retained in selected additional-capability sender")
			}
			canonical, err := controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentCanonicalAtLength(value, 22)
			if err != nil {
				t.Fatal(err)
			}
			expanded := simpleControlBits(prefix+cap(0)+"10", 190)
			rr, e := runtime.NewReaderBits(expanded, 190, nil)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = rr.ReadUint(14); e != nil {
				t.Fatal(e)
			}
			expected, e := rr.ReadBitString(176)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(canonical, expected.Bytes) {
				t.Fatal("canonical byte API retained nested spares")
			}
			simpleControlReplay(t, raw, 184, 14, r, func(w *runtime.Writer) error {
				return controlmessages.EncodeAdditionalMSRadioAccessCapabilitiesMessageContentTo(w, value)
			})
		}
	}
}
