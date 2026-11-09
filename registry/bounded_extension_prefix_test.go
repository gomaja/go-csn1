package registry

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover3"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/pshandover2"
)

// TS 44.060 V19.0.0 §11.1.4.4 applies to the five length-delimited
// descriptions in §§11.2.6a, 12.24, 12.42a, 12.48 and 12.64. Each vector
// ends inside the next component, after a prefix with nonzero field values.
func TestBoundedExtensionComponents(t *testing.T) {
	for _, tc := range []struct {
		clause, name, bits string
		fields             map[string]uint64
		encode             func(*runtime.Writer, any) error
	}{
		{"12.24", "Extension Information", "1101011111", map[string]uint64{"EGPRSPACKETCHANNELREQUEST": 1, "BEPPERIOD": 5, "PFCFEATUREMODE": 1, "DTMSUPPORT": 1, "BSSPAGINGCOORDINATION": 1}, func(w *runtime.Writer, value any) error {
			return ies.EncodeExtensionInformationTo(w, value.(runtime.Decoded[ies.ExtensionInformation]).Value)
		}},
		{"11.2.6a", "EPD A/N Extension Info", "011010111011", map[string]uint64{"RBId": 21, "TimeslotNumber": 5}, func(w *runtime.Writer, value any) error {
			return ackmessages.EncodeEPDANExtensionInfoTo(w, value.(runtime.Decoded[ackmessages.EPDANExtensionInfo]).Value)
		}},
		{"12.42a", "Extension Information", "111", map[string]uint64{"MTTIDOWNLINKASSIGNMENTC1": 1}, func(w *runtime.Writer, value any) error {
			return pshandover2.EncodeExtensionInformationTo(w, value.(runtime.Decoded[pshandover2.ExtensionInformation]).Value)
		}},
		{"12.48", "Extension Information", "011100011", map[string]uint64{"EMSTNWCapability": 1, "MTTIDOWNLINKASSIGNMENTC1": 1, "EGPRSPacketDownlinkAckNackType3Support": 1}, func(w *runtime.Writer, value any) error {
			return dtmhandover2.EncodeExtensionInformationTo(w, value.(runtime.Decoded[dtmhandover2.ExtensionInformation]).Value)
		}},
		{"12.64", "Extension Information", "01110000011", map[string]uint64{"EMSTNWCapability": 1, "MTTIDOWNLINKASSIGNMENTC1": 1, "EGPRSPacketDownlinkAckNackType3Support": 1}, func(w *runtime.Writer, value any) error {
			return dtmhandover3.EncodeExtensionInformationTo(w, value.(runtime.Decoded[dtmhandover3.ExtensionInformation]).Value)
		}},
	} {
		t.Run(tc.clause, func(t *testing.T) {
			d, err := LookupClause("TS 44.060", tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			wire := runtime.NewWriter()
			if err := wire.WriteLiteral(tc.bits); err != nil {
				t.Fatal(err)
			}
			bits, err := wire.FinishBits(runtime.BitString{})
			if err != nil {
				t.Fatal(err)
			}
			r, err := runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(bits.BitLength); err != nil {
				t.Fatal(err)
			}
			value, err := d.DecodeFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if r.Remaining() != 0 {
				t.Fatal("cut component did not end at its bound")
			}
			data, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var decoded any
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatal(err)
			}
			found := map[string]bool{}
			var walk func(any)
			walk = func(v any) {
				switch x := v.(type) {
				case map[string]any:
					for key, item := range x {
						if n, ok := item.(float64); ok {
							if want, exists := tc.fields[key]; exists && n == float64(want) {
								found[key] = true
							}
						}
						walk(item)
					}
				case []any:
					for _, item := range x {
						walk(item)
					}
				}
			}
			walk(decoded)
			for key := range tc.fields {
				if !found[key] {
					t.Fatalf("complete field %s lost: %s", key, data)
				}
			}
			out := runtime.NewWriter()
			if _, err := out.PushLimit(bits.BitLength); err != nil {
				t.Fatal(err)
			}
			if err := tc.encode(out, value); err != nil {
				t.Fatal(err)
			}
			replay, err := out.FinishBits(runtime.BitString{})
			if err != nil || replay.BitLength != bits.BitLength || !bytes.Equal(replay.Bytes, bits.Bytes) {
				t.Fatalf("byte-exact component replay: %v", err)
			}
		})
	}
}
