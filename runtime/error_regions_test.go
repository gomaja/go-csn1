package runtime

import (
	"reflect"
	"strings"
	"testing"
	"unsafe"
)

func TestErrorRegionOwnership(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xa5, 0x80}, 9, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(2); err != nil {
		t.Fatal(err)
	}
	if err = r.Enter("Body"); err != nil {
		t.Fatal(err)
	}
	loser := r.Fork()
	if _, err = loser.ReadErrorRegion(DistributionPartError, 2, fillSource); err != nil {
		t.Fatal(err)
	}
	loser.Discard()
	winner := r.Fork()
	region, err := winner.ReadErrorRegion(NonDistributionPartError, -1, fillSource)
	if err != nil {
		t.Fatal(err)
	}
	if region.Label != NonDistributionPartError || region.Location.BitOffset != 2 || region.Location.Width != 7 || region.Location.Path != "Body" || region.Location.Source != fillSource {
		t.Fatalf("region: %+v", region)
	}
	r.Commit(winner)
	report := c.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != NonDistributionPartError || report.Regions[0].Location != region.Location {
		t.Fatalf("report: %+v", report)
	}
	if len(report.ZeroFill) != 0 || report.Dropped != 0 {
		t.Fatalf("report contamination: %+v", report)
	}
	region.Bits.Bytes[0] = 0
	r2, err := NewReaderBits([]byte{0xff}, 8, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r2.ReadErrorRegion(Ignore, -1, Source{}); err != nil {
		t.Fatal(err)
	}
	c.TakeReceiverReport()
	if report.Regions[0].Location.Path != "Body" {
		t.Fatal("report changed on reuse")
	}
}

func TestReceiverForkSnapshot(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xff}, 8, c)
	if err != nil {
		t.Fatal(err)
	}
	first := r.Fork()
	if _, err = first.ReadErrorRegion(Ignore, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	snapshot := first.Fork()
	first.Discard()
	if _, err = snapshot.ReadErrorRegion(MessageEscape, 2, Source{}); err != nil {
		t.Fatal(err)
	}
	r.Commit(snapshot)
	got := c.TakeReceiverReport()
	if len(got.Regions) != 2 || got.Regions[0].Label != Ignore || got.Regions[1].Label != MessageEscape {
		t.Fatalf("aliased fork: %+v", got)
	}
}

func TestDiagnosticGrowth(t *testing.T) {
	for _, size := range []int{0, 1, 16, 256, 4096} {
		c := NewReceiverCollector()
		r, err := NewReaderBits(make([]byte, size), size*8, c)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < size*8+256; i++ {
			arm := r.Fork()
			if _, err = arm.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
				t.Fatal(err)
			}
			r.Commit(arm)
		}
		got := c.TakeReceiverReport()
		if len(got.Regions) > size*8+128 || got.Dropped == 0 {
			t.Fatalf("unbounded report size=%d count=%d dropped=%d", size, len(got.Regions), got.Dropped)
		}
		if *r.steps != 0 {
			t.Fatal("diagnostics consumed decode steps")
		}
	}
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xff}, 8, c)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.Enter(strings.Repeat("x", 1<<18)); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
		t.Fatal(err)
	}
	if got := c.TakeReceiverReport(); len(got.Regions) != 0 || got.Dropped != 1 {
		t.Fatal("oversize path accepted")
	}
}

func TestOrdinaryDecodeAllocationBaseline(t *testing.T) {
	// These layout controls preserve existing allocation size classes on amd64.
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("amd64 allocation-size control")
	}
	if got := unsafe.Sizeof(Reader{}); got != 344 {
		t.Fatalf("Reader layout grew: %d", got)
	}
	if got := unsafe.Sizeof(Collector{}); got != 96 {
		t.Fatalf("Collector layout grew: %d", got)
	}
	if got := unsafe.Sizeof(WireInfo{}); got != 184 {
		t.Fatalf("WireInfo layout grew: %d", got)
	}
	plain := testing.AllocsPerRun(20, func() { r := NewReader([]byte{0xff}); _, _ = r.ReadUint(8) })
	disabled := testing.AllocsPerRun(20, func() { r := NewReaderWithCollector([]byte{0xff}, nil); _, _ = r.ReadUint(8) })
	if disabled != plain {
		t.Fatalf("nil collector changed ordinary allocations: %v versus %v", disabled, plain)
	}

	if !reflect.DeepEqual((*Collector)(nil).TakeReceiverReport(), ReceiverReport{}) {
		t.Fatal("nil collector report")
	}
}

func TestErrorRegionLabelsAndReplay(t *testing.T) {
	for _, label := range []ErrorLabel{UnknownMessageType, DistributionPartError, AddressInformationPartError, NonDistributionPartError, DBPSCHMessagePartError, MessageEscape, Ignore} {
		for phase := 0; phase < 8; phase++ {
			r, err := NewReaderBits([]byte{0xa5, 0x80}, 9, NewReceiverCollector())
			if err != nil {
				t.Fatal(err)
			}
			prefix, err := r.ReadUint(phase)
			if err != nil {
				t.Fatal(err)
			}
			region, err := r.ReadErrorRegion(label, -1, fillSource)
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
			out, err := w.FinishBits(BitString{})
			if err != nil || out.BitLength != 9 || !reflect.DeepEqual(out.Bytes, []byte{0xa5, 0x80}) {
				t.Fatalf("phase %d %s replay %+v %v", phase, label, out, err)
			}
		}
	}
	r := NewReader([]byte{0})
	if _, err := r.ReadErrorRegion(ErrorLabel("bad"), 0, Source{}); err == nil {
		t.Fatal("unknown label accepted")
	}
	if _, err := r.ReadErrorRegion(Ignore, -2, Source{}); err == nil {
		t.Fatal("invalid width accepted")
	}
	r.SetZeroExtension(true)
	if _, err := r.ReadErrorRegion(Ignore, 9, Source{}); err == nil {
		t.Fatal("ignored received region inferred")
	}
}

func TestRegionSyntaxErrorBoundary(t *testing.T) {
	for _, kind := range []ErrorKind{Truncated, InvalidBranch, InvalidValue, Limit, InvalidSchema} {
		err := &DecodeError{Kind: kind}
		want := kind != Limit && kind != InvalidSchema
		if IsRegionSyntaxError(err) != want {
			t.Fatalf("kind %s", kind)
		}
		var choice ChoiceErrors
		arm := choice.ForkScoped(NewReader(nil))
		scoped := arm.Error(kind, "failure")
		choice.Track(scoped)
		if IsRegionSyntaxError(scoped) != want {
			t.Fatalf("scoped kind %s", kind)
		}
		choice.Release()
	}
	if IsRegionSyntaxError(ErrContextRequired) || IsRegionSyntaxError(ErrLHPositionUndefined) || IsRegionSyntaxError(nil) {
		t.Fatal("non-syntax failure caught")
	}
}

func FuzzReceiverDiagnostics(f *testing.F) {
	f.Add([]byte{0xff, 0x80}, uint8(3), uint8(11))
	f.Add([]byte{}, uint8(0), uint8(0))
	f.Fuzz(func(t *testing.T, data []byte, phase, width uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		extent := min(len(data)*8, int(width))
		c := NewReceiverCollector()
		r, err := NewChoiceReaderBits(data, extent, c)
		if err != nil {
			t.Fatal(err)
		}
		prefix := min(extent, int(phase)%8)
		if _, err = r.ReadUint(prefix); err != nil {
			t.Fatal(err)
		}
		var choices ChoiceErrors
		trial := choices.ForkScoped(r)
		trial.SetZeroExtension(true)
		if _, err = trial.ReadUintWithFill(int(width)%65, fillSource, "Trial"); err != nil {
			t.Fatal(err)
		}
		trial.Discard()
		choices.Release()
		if got := c.TakeReceiverReport(); len(got.ZeroFill) != 0 {
			t.Fatal("discarded trial leaked")
		}
		arm, err := r.ReadErrorRegion(Ignore, -1, Source{})
		if err != nil || arm.Bits.BitLength != extent-prefix {
			t.Fatal("region extent differs")
		}
		r.SetZeroExtension(true)
		if _, err = r.ReadBitStringWithFill(int(width)%65, fillSource, "Suffix"); err != nil {
			t.Fatal(err)
		}
		if err = r.ExpectWithFill(strings.Repeat("0", int(width)%8), fillSource, "Literal"); err != nil {
			t.Fatal(err)
		}
		got := c.TakeReceiverReport()
		wantFills := 0
		if width%65 != 0 {
			wantFills++
		}
		if width%8 != 0 {
			wantFills++
		}
		if len(got.Regions) != 1 || len(got.ZeroFill) != wantFills {
			t.Fatalf("diagnostics lost: %+v", got)
		}
		if received := c.TakeReport(); len(received.Observations) != 0 {
			t.Fatal("inferred zeros reported as received")
		}
	})
}

func TestReceiverReservedAndNestedOwnership(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xff}, 8, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if err = r.Observe(0, 1, 1, Source{}, "Prefix"); err != nil {
		t.Fatal(err)
	}
	var outer, inner ChoiceErrors
	loser := outer.ForkScoped(r)
	if _, err = loser.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if err = loser.Observe(1, 1, 1, Source{}, "Rejected"); err != nil {
		t.Fatal(err)
	}
	if _, err = loser.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
		t.Fatal(err)
	}
	loser.Discard()
	winner := outer.ForkScoped(r)
	if _, err = winner.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if err = winner.Observe(1, 1, 1, Source{}, "Selected"); err != nil {
		t.Fatal(err)
	}
	if _, err = winner.ReadErrorRegion(DistributionPartError, 0, Source{}); err != nil {
		t.Fatal(err)
	}
	child := inner.ForkScoped(winner)
	if _, err = child.ReadErrorRegion(Ignore, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	snapshot := child.Fork()
	winner.Commit(child)
	inner.Release()
	r.Commit(winner)
	outer.Release()
	if _, err = snapshot.ReadErrorRegion(MessageEscape, 1, Source{}); err != nil {
		t.Fatal(err)
	}
	r.Commit(snapshot)
	reserved := c.TakeReport()
	regions := c.TakeReceiverReport()
	if len(reserved.Observations) != 2 || reserved.Observations[1].Location.Path != "Selected" || len(regions.Regions) != 3 || regions.Regions[2].Label != MessageEscape {
		t.Fatalf("reserved=%+v regions=%+v", reserved, regions)
	}

}

func TestReceiverDiscardDoesNotRefundBudget(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xff}, 8, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		arm := r.Fork()
		if _, err = arm.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
			t.Fatal(err)
		}
		arm.Discard()
	}
	if _, err = r.ReadErrorRegion(Ignore, 0, Source{}); err != nil {
		t.Fatal(err)
	}
	got := c.TakeReceiverReport()
	if len(got.Regions) != 0 || got.Dropped != 1 {
		t.Fatalf("discard refunded collection work: %+v", got)
	}
}
