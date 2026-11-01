package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestPrintedErrorLabelsReplay(t *testing.T) {
	// The printed labels are case-sensitive source designations.
	labels := []ErrorLabel{
		"Unknown message type", "Distribution part error", "Address information part error",
		"Non-distribution part error", "DBPSCH message part error", "Message escape", "Ignore",
		"PDCH pairs configuration error", "Frequency parameters error",
	}
	for _, label := range labels {
		for phase := 0; phase < 8; phase++ {
			for _, width := range []int{0, 1, -1} {
				for _, collect := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/phase%d/width%d/collect%t", label, phase, width, collect), func(t *testing.T) {
						input := []byte{0xa5, 0x80}
						var collector *Collector
						if collect {
							collector = NewReceiverCollector()
						}
						r, err := NewReaderBits(input, 9, collector)
						if err != nil {
							t.Fatal(err)
						}
						prefix, err := r.ReadUint(phase)
						if err != nil {
							t.Fatal(err)
						}
						if err = r.Enter("Body"); err != nil {
							t.Fatal(err)
						}
						region, err := r.ReadErrorRegion(label, width, fillSource)
						if err != nil {
							t.Fatal(err)
						}
						wantWidth := width
						if width == -1 {
							wantWidth = 9 - phase
						}
						wantLocation := Location{Path: "Body", BitOffset: phase, Width: wantWidth, Source: fillSource}
						if region.Label != label || region.Location != wantLocation || region.Bits.BitLength != wantWidth {
							t.Fatalf("region metadata: %+v", region)
						}
						report := collector.TakeReceiverReport()
						if collect {
							if len(report.Regions) != 1 || report.Regions[0].Label != label || report.Regions[0].Location != wantLocation || len(report.ZeroFill) != 0 || report.Dropped != 0 {
								t.Fatalf("receiver report: %+v", report)
							}
						} else if len(report.Regions) != 0 {
							t.Fatal("nil collector retained region")
						}
						suffixWidth := 9 - phase - wantWidth
						suffix, err := r.ReadUint(suffixWidth)
						if err != nil {
							t.Fatal(err)
						}
						w := NewWriter()
						if err = w.WriteUint(prefix, phase); err != nil {
							t.Fatal(err)
						}
						if err = w.WriteErrorRegion(region); err != nil {
							t.Fatal(err)
						}
						if err = w.WriteUint(suffix, suffixWidth); err != nil {
							t.Fatal(err)
						}
						got, err := w.FinishBits(BitString{})
						if err != nil || got.BitLength != 9 || !bytes.Equal(got.Bytes, input) {
							t.Fatalf("replay differs: %+v %v", got, err)
						}
					})
				}
			}
		}
	}
}

func TestPrintedErrorLabelsRefuseNearMisses(t *testing.T) {
	for _, label := range []ErrorLabel{
		"", "arbitrary label", "pdch pairs configuration error", "PDCH Pairs configuration error",
		"PDCH pairs configuration Error", " PDCH pairs configuration error", "PDCH pairs configuration error ",
		"PDCH  pairs configuration error", "PDCH pair configuration error", "PDCH pairs configurations error",
		"frequency parameters error", "Frequency Parameters error", "Frequency parameters Error",
		" Frequency parameters error", "Frequency parameters error ", "Frequency  parameters error",
		"Frequency parameter error", "Frequency parameters errors", "Ignore ", "ignore",
	} {
		t.Run(string(label), func(t *testing.T) {
			c := NewReceiverCollector()
			r := NewReaderWithCollector([]byte{0x80}, c)
			_, err := r.ReadErrorRegion(label, 1, fillSource)
			var decode *DecodeError
			if !errors.As(err, &decode) || decode.Kind != InvalidSchema {
				t.Fatalf("read accepted unknown label %q: %v", label, err)
			}
			if r.Position() != 0 || len(c.TakeReceiverReport().Regions) != 0 {
				t.Fatal("invalid label consumed or reported bits")
			}
			w := NewWriter()
			err = w.WriteErrorRegion(ErrorRegion{Label: label, Location: Location{Width: 1}, Bits: BitString{Bytes: []byte{0x80}, BitLength: 1}})
			var operation *OperationError
			if !errors.As(err, &operation) || operation.Kind != InvalidValue {
				t.Fatalf("write accepted unknown label %q: %v", label, err)
			}
		})
	}
}

func TestPrintedErrorLabelConstants(t *testing.T) {
	if PDCHPairsConfigurationError != ErrorLabel("PDCH pairs configuration error") || FrequencyParametersError != ErrorLabel("Frequency parameters error") {
		t.Fatal("printed label constant changed")
	}
	existing := []ErrorLabel{UnknownMessageType, DistributionPartError, AddressInformationPartError, NonDistributionPartError, DBPSCHMessagePartError, MessageEscape, Ignore}
	want := []string{"Unknown message type", "Distribution part error", "Address information part error", "Non-distribution part error", "DBPSCH message part error", "Message escape", "Ignore"}
	for i, label := range existing {
		if string(label) != want[i] {
			t.Fatalf("existing label %d changed: %q", i, label)
		}
	}
}
