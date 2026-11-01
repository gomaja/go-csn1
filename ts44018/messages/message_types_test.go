package messages

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func TestECDummyAssignedMessageType(t *testing.T) {
	if _, err := DecodeECDummyMessageContent([]byte{0}); err == nil {
		t.Fatal("wrong EC-Dummy message type decoded")
	}
	if _, err := EncodeECDummyMessageContentCanonicalAtLength(ECDummyMessageContent{}, 1); err == nil {
		t.Fatal("wrong EC-Dummy message type encoded")
	}
	value := ECDummyMessageContent{MessageType: 3, UsedDLCoverageClass: 2}
	wire, err := EncodeECDummyMessageContentCanonicalAtLength(value, 1)
	if err != nil || !bytes.Equal(wire, []byte{0x3b}) {
		t.Fatalf("canonical EC-Dummy: %x %v", wire, err)
	}
	if _, err := EncodeECDummyMessageContentCanonical(value); err == nil {
		t.Fatal("unbounded canonical padding accepted without extent")
	} else {
		var extent *runtime.ExtentError
		if !errors.As(err, &extent) || !extent.Required {
			t.Fatalf("missing canonical extent type: %v", err)
		}
	}
	for _, n := range []int{1, 2, 4, 8, 11, 12, 16} {
		wire, err := EncodeECDummyMessageContentCanonicalAtLength(value, n)
		if n > 11 {
			// TS 45.003 V19.0.0 §5.2b.1 p9934–9935: EC-CCCH/D
			// has 88 information bits, including the six-bit EC header.
			var extent *runtime.ExtentError
			if wire != nil || !errors.As(err, &extent) || extent.Maximum != 11 {
				t.Fatalf("oversized sender: %x %v", wire, err)
			}
			wire = bytes.Repeat([]byte{0x2b}, n)
			wire[0] = 0x3b
		} else if err != nil {
			t.Fatal(err)
		}
		reader := runtime.NewReader(append(wire, 0xff))
		old, err := reader.PushLimit(n * 8)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeECDummyMessageContentFrom(reader); err != nil {
			t.Fatal(err)
		}
		if err := reader.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if reader.Position() != n*8 {
			t.Fatalf("carrier crossed: %d", reader.Position())
		}
		tail, err := reader.ReadUint(8)
		if err != nil || tail != 255 {
			t.Fatalf("following byte consumed: %d %v", tail, err)
		}
	}
}
