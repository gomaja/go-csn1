package access_test

import (
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/access"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.44 p20779 defines 8/11-bit access
// bursts; table 11.2.44.1 pp20783–20787 prints 8 reference bits
// followed, only in the 11-bit form, by 3 spares. Table 11.2.44.2
// pp20791–20792 require fresh sender spares = 000.
func TestClause11244HandoverAccess(t *testing.T) {
	for reference := 0; reference < 256; reference++ {
		bits := fmt.Sprintf("%08b", reference)
		v8 := accessRoundTrip(t, bits, access.DecodePSHandoverAccessMessageContent8BitMessageFrom, access.EncodePSHandoverAccessMessageContent8BitMessageTo, access.EncodePSHandoverAccessMessageContent8BitMessageCanonical)
		if int(v8.HandoverReference) != reference {
			t.Fatal("reference lost")
		}
		v8.HandoverReference ^= 1
		accessEqual(t, accessEncode(t, v8, access.EncodePSHandoverAccessMessageContent8BitMessageTo), fmt.Sprintf("%08b", reference^1))
		accessTruncations(t, bits, access.DecodePSHandoverAccessMessageContent8BitMessageFrom)
		accessSentinel(t, bits, access.DecodePSHandoverAccessMessageContent8BitMessageFrom)
		fresh := access.PSHandoverAccessMessageContent11BitMessage{HandoverReference: uint8(reference)}
		accessEqual(t, accessEncode(t, fresh, access.EncodePSHandoverAccessMessageContent11BitMessageTo), bits+"000")
		for spare := 0; spare < 8; spare++ {
			full := bits + fmt.Sprintf("%03b", spare)
			c := new(runtime.Collector)
			r := accessReader(t, full, c)
			v, err := access.DecodePSHandoverAccessMessageContent11BitMessageFrom(r)
			if err != nil || int(v.HandoverReference) != reference || r.Position() != 11 {
				t.Fatalf("handover %+v %v", v, err)
			}
			report := c.TakeReport()
			if (len(report.Observations) > 0) != (spare != 0) {
				t.Fatalf("spare %d: %+v", spare, report)
			}
			// The bare typed reader shares its layout stream with the caller.
			// Seal the exact standalone extent before using a new writer.
			v.Wire, err = runtime.SealBits(v, runtime.BitString{Bytes: accessBits(full), BitLength: len(full)}, r.Position(), runtime.BitString{}, r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			accessEqual(t, accessEncode(t, v, access.EncodePSHandoverAccessMessageContent11BitMessageTo), full)
			d, err := access.Lookup("PS Handover Access message content 11 bit message")
			if err != nil {
				t.Fatal(err)
			}
			dv, err := d.DecodeBits(accessBits(full), len(full))
			if err != nil {
				t.Fatal(err)
			}
			accessEqual(t, accessEncode(t, dv.(runtime.Decoded[access.PSHandoverAccessMessageContent11BitMessage]).Value, access.EncodePSHandoverAccessMessageContent11BitMessageTo), full)
			cv := runtime.Canonical(v)
			accessEqual(t, accessEncode(t, cv, access.EncodePSHandoverAccessMessageContent11BitMessageTo), bits+"000")
			cv.HandoverReference ^= 1
			accessEqual(t, accessEncode(t, cv, access.EncodePSHandoverAccessMessageContent11BitMessageTo), fmt.Sprintf("%08b000", reference^1))
			if reference == 0 {
				accessTruncations(t, full, access.DecodePSHandoverAccessMessageContent11BitMessageFrom)
				accessSentinel(t, full, access.DecodePSHandoverAccessMessageContent11BitMessageFrom)
			}
		}
	}
}
