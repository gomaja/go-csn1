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
	for _, n := range []int{1, 2, 4, 8, 16} {
		wire, err := EncodeECDummyMessageContentCanonicalAtLength(value, n)
		if err != nil {
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
