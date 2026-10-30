package registry_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	rries "github.com/gomaja/go-csn1/ts44018/ies"
	"github.com/gomaja/go-csn1/ts44018/messages"
)

func TestPublishedRRRootsLookup(t *testing.T) {
	for _, tc := range []struct{ clause, name string }{
		{"10.5.2.61", "Restriction Timer value part"},
		{"10.5.2.83", "EC Request Reference Description"},
		{"9.1.62", "EC-Dummy message content"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			qualified, err := registry.LookupClause("TS 44.018", tc.clause, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			byName, err := registry.Lookup("TS 44.018", tc.name)
			if err != nil || byName.Standard != qualified.Standard || byName.Clause != qualified.Clause || byName.Name != qualified.Name {
				t.Fatalf("name lookup differs from clause lookup: %+v, %v", byName, err)
			}
			found := 0
			for _, descriptor := range registry.Descriptors() {
				if descriptor.Standard == "TS 44.018" && descriptor.Clause == tc.clause && descriptor.Name == tc.name {
					found++
				}
			}
			if found != 1 {
				t.Fatalf("root registered %d times", found)
			}
			if _, err := registry.LookupClause("TS 44.060", tc.clause, tc.name); err == nil {
				t.Fatal("root registered under another standard")
			}
		})
	}
}

func TestPublishedRRRootsTypedOperations(t *testing.T) {
	t.Run("Restriction Timer", func(t *testing.T) {
		checkRegisteredRRRoot(t, "10.5.2.61", "Restriction Timer value part", []byte{0xff}, []byte{0xf0}, func(value rries.RestrictionTimerValuePart) {
			if value.RESTRICTIONTIMER != 15 {
				t.Fatalf("restriction timer=%d", value.RESTRICTIONTIMER)
			}
		})
	})
	t.Run("EC Request Reference Description", func(t *testing.T) {
		checkRegisteredRRRoot(t, "10.5.2.83", "EC Request Reference Description", []byte{0x55, 0x68}, []byte{0x55, 0x68}, func(value rries.ECRequestReferenceDescription) {
			if value.LastTDMAFrame != 341 || value.EchoedRandomBits != 5 {
				t.Fatalf("request reference fields=%+v", value)
			}
		})
	})
	t.Run("EC-Dummy", func(t *testing.T) {
		checkRegisteredRRRoot(t, "9.1.62", "EC-Dummy message content", []byte{0x3b}, []byte{0x3b}, func(value messages.ECDummyMessageContent) {
			if value.MessageType != 3 || value.UsedDLCoverageClass != 2 {
				t.Fatalf("EC-Dummy fields=%+v", value)
			}
		})
		d, err := registry.Lookup("TS 44.018", "EC-Dummy message content")
		if err != nil {
			t.Fatal(err)
		}
		_, err = d.Canonical(messages.ECDummyMessageContent{MessageType: 3, UsedDLCoverageClass: 2})
		var extent *runtime.ExtentError
		if !errors.As(err, &extent) || !extent.Required {
			t.Fatalf("missing canonical extent lost its type: %v", err)
		}
		if _, err := registry.Decode("TS 44.018", "EC-Dummy message content", []byte{0}); err == nil {
			t.Fatal("wrong message type accepted through registry")
		}
	})
}

func checkRegisteredRRRoot[T any](t *testing.T, clause, name string, wire, canonical []byte, check func(T)) {
	t.Helper()
	d, err := registry.LookupClause("TS 44.018", clause, name)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := registry.Decode("TS 44.018", name, wire)
	if err != nil {
		t.Fatal(err)
	}
	typed, ok := decoded.(runtime.Decoded[T])
	if !ok {
		t.Fatalf("registry returned %T, want runtime.Decoded[%T]", decoded, *new(T))
	}
	check(typed.Value)
	for _, value := range []any{typed, typed.Value} {
		replay, err := d.Encode(value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatalf("registry replay of %T=%x, %v", value, replay, err)
		}
		fresh, err := d.CanonicalAtLength(value, len(canonical))
		if err != nil || !bytes.Equal(fresh, canonical) {
			t.Fatalf("registry canonical of %T=%x, %v", value, fresh, err)
		}
	}
	if _, err := d.Decode(nil); err == nil {
		t.Fatal("empty value accepted")
	}
}
