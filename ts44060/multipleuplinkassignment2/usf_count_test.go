package multipleuplinkassignment2

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §12.48a.6 pp25074,25082–25083,25194;
// the repetition reading was approved on 2026-11-07.
func TestBTTIUSFPerAssignedTimeslot(t *testing.T) {
	for count := 0; count < 16; count++ {
		for _, mixed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", count, mixed), func(t *testing.T) {
				w := runtime.NewWriter()
				// PFI absent; RLC mode 1; TFI 9; coding/window/NPM/FANR absent;
				// USF granularity 1; explicit N_TS; all timeslots; distinct USFs.
				if err := w.WriteLiteral("011001000001"); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteUint(uint64(count), 4); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteLiteral("01101"); err != nil {
					t.Fatal(err)
				}
				for i := 0; i < count; i++ {
					if mixed && i%2 == 0 {
						if err := w.WriteLiteral("0"); err != nil {
							t.Fatal(err)
						}
					} else {
						if err := w.WriteLiteral("1"); err != nil {
							t.Fatal(err)
						}
						if err := w.WriteUint(uint64(i%8), 3); err != nil {
							t.Fatal(err)
						}
					}
				}
				wire, err := w.FinishBits(runtime.BitString{})
				if err != nil {
					t.Fatal(err)
				}
				r, err := runtime.NewReaderBits(wire.Bytes, wire.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				v, err := DecodeBTTIUplinkTBFAssignmentStructFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				if int(v.NTS) != count || v.USFC1Choice.USF == nil || v.USFC1Choice.USF.USF != 5 || len(v.USFC1Choice.USF.USFList) != count || r.Remaining() != 0 {
					t.Fatalf("N_TS %d does not yield %d USF entries", count, count+1)
				}
				for i, value := range v.USFC1Choice.USF.USFList {
					if mixed && i%2 == 0 {
						if value != nil {
							t.Fatal("omitted USF retained a value")
						}
					} else if value == nil || int(*value) != i%8 {
						t.Fatalf("USF %d lost its value", i)
					}
				}
				out := runtime.NewWriter()
				if err := EncodeBTTIUplinkTBFAssignmentStructTo(out, v); err != nil {
					t.Fatal(err)
				}
				replay, err := out.FinishBits(runtime.BitString{})
				if err != nil || replay.BitLength != wire.BitLength || !bytes.Equal(replay.Bytes, wire.Bytes) {
					t.Fatalf("round trip: %v", err)
				}
				if err := w.WriteLiteral("1101"); err != nil {
					t.Fatal(err)
				}
				printed, err := w.FinishBits(runtime.BitString{})
				if err != nil {
					t.Fatal(err)
				}
				r, err = runtime.NewReaderBits(printed.Bytes, printed.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				v, err = DecodeBTTIUplinkTBFAssignmentStructFrom(r)
				if err != nil || v.USFC1Choice.USF == nil || len(v.USFC1Choice.USF.USFList) != count || r.Remaining() != 4 {
					t.Fatalf("printed extra USF was consumed: %v", err)
				}
			})
		}
	}
}
