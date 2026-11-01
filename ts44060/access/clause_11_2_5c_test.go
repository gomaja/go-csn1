package access_test

import (
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
)

// TS 44.060 V19.0.0 §11.2.5c table 11.2.5c.1 pp14388–14389:
// 0000 + seven RandomBits = eleven significant bits, origin zero.
func TestClause1125cMPRACH(t *testing.T) {
	for random := 0; random < 128; random++ {
		bits := fmt.Sprintf("0000%07b", random)
		v := accessRoundTrip(t, bits, access.DecodeMPRACHPacketChannelRequestMessageContentFrom, access.EncodeMPRACHPacketChannelRequestMessageContentTo, access.EncodeMPRACHPacketChannelRequestMessageContentCanonical)
		if int(v.RandomBits) != random {
			t.Fatal("RandomBits lost")
		}
		v.RandomBits ^= 64
		accessEqual(t, accessEncode(t, v, access.EncodeMPRACHPacketChannelRequestMessageContentTo), fmt.Sprintf("0000%07b", random^64))
		accessTruncations(t, bits, access.DecodeMPRACHPacketChannelRequestMessageContentFrom)
		accessSentinel(t, bits, access.DecodeMPRACHPacketChannelRequestMessageContentFrom)
	}
	for prefix := 1; prefix < 16; prefix++ {
		if _, err := access.DecodeMPRACHPacketChannelRequestMessageContentFrom(accessReader(t, fmt.Sprintf("%04b0000000", prefix), nil)); err == nil {
			t.Fatal("invalid prefix accepted")
		}
	}
}
func FuzzClause1125cMPRACH(f *testing.F) {
	f.Add(uint16(17))
	f.Fuzz(func(t *testing.T, n uint16) {
		bits := fmt.Sprintf("%011b", n&2047)
		r := accessReader(t, bits, nil)
		v, err := access.DecodeMPRACHPacketChannelRequestMessageContentFrom(r)
		if err == nil {
			got := accessEncode(t, v, access.EncodeMPRACHPacketChannelRequestMessageContentTo)
			accessEqual(t, got, bits)
			if got.BitLength != 11 {
				t.Fatal(runtime.BitString{})
			}
		}
	})
}
