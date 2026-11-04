package registry

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44018/messages"
	"github.com/gomaja/go-csn1/ts44018/notification"
)

func TestShortHeaderCarrierLimits(t *testing.T) {
	// TS 44.018 V19.0.0 §§9.1.21a, 9.1.47, 9.1.57 select main DCCH;
	// §9.1.49b selects SACCH. TS 44.006 V19.0.0 §§5.2, 6.4a, 8.8.3
	// give 23 and 21 complete Bter octets, including the printed short header.
	t.Run("notification", func(t *testing.T) {
		checkShortHeaderCarrier(t, "NOTIFICATION FACCH", 23, notification.DecodeNOTIFICATIONFACCH, notification.EncodeNOTIFICATIONFACCH, notification.EncodeNOTIFICATIONFACCHCanonical, notification.EncodeNOTIFICATIONFACCHCanonicalAtLength)
	})
	t.Run("uplink-free", func(t *testing.T) {
		checkShortHeaderCarrier(t, "UPLINK FREE", 23, messages.DecodeUPLINKFREE, messages.EncodeUPLINKFREE, messages.EncodeUPLINKFREECanonical, messages.EncodeUPLINKFREECanonicalAtLength)
	})
	t.Run("sms-information", func(t *testing.T) {
		checkShortHeaderCarrier(t, "VGCS SMS INFORMATION", 21, messages.DecodeVGCSSMSINFORMATION, messages.EncodeVGCSSMSINFORMATION, messages.EncodeVGCSSMSINFORMATIONCanonical, messages.EncodeVGCSSMSINFORMATIONCanonicalAtLength)
	})
	t.Run("neighbour-information", func(t *testing.T) {
		checkShortHeaderCarrier(t, "VGCS Neighbour Cell Information", 23, messages.DecodeVGCSNeighbourCellInformation, messages.EncodeVGCSNeighbourCellInformation, messages.EncodeVGCSNeighbourCellInformationCanonical, messages.EncodeVGCSNeighbourCellInformationCanonicalAtLength)
	})
}

func checkShortHeaderCarrier[T any](t *testing.T, name string, maximum int,
	decode func([]byte) (runtime.Decoded[T], error), encode func(T) ([]byte, error),
	canonical func(T) ([]byte, error), atLength func(T, int) ([]byte, error),
) {
	t.Helper()
	d, err := Lookup("TS 44.018", name)
	if err != nil {
		t.Fatal(err)
	}
	_, sample := paddingLimitSample(t, d)
	received, err := decode(sample)
	if err != nil {
		t.Fatal(err)
	}
	out, err := atLength(received.Value, maximum)
	if err != nil || len(out) != maximum {
		t.Fatalf("at cap: length=%d error=%v", len(out), err)
	}
	if _, err = decode(out); err != nil {
		t.Fatalf("at-cap decode: %v", err)
	}
	for _, n := range []int{maximum + 1, 30, (1 << 20) / 8} {
		out, err = atLength(received.Value, n)
		var extent *runtime.ExtentError
		if out != nil || !errors.As(err, &extent) || extent.Maximum != maximum || extent.Actual != n {
			t.Errorf("past cap %d: length=%d error=%v", n, len(out), err)
		}
	}
	out, err = canonical(received.Value)
	var required *runtime.ExtentError
	if out != nil || !errors.As(err, &required) || !required.Required {
		t.Fatalf("standalone extent: %v", err)
	}
	for _, n := range []int{maximum + 1, 30} {
		t.Run(fmt.Sprintf("received-%d", n), func(t *testing.T) {
			// Extend only the terminal padding of a synthetically decoded sample.
			wire := append(bytes.Clone(sample), bytes.Repeat([]byte{0x2b}, n-len(sample))...)
			decoded, decodeErr := decode(wire)
			if decodeErr != nil {
				t.Fatalf("tolerant receive: %v", decodeErr)
			}
			replay, replayErr := encode(decoded.Value)
			if replayErr != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("replay: length=%d error=%v", len(replay), replayErr)
			}
			generic, decodeErr := d.Decode(wire)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			replay, replayErr = d.Encode(generic)
			if replayErr != nil || !bytes.Equal(replay, wire) {
				t.Fatalf("descriptor replay: length=%d error=%v", len(replay), replayErr)
			}
		})
	}
}
