package runtime

import (
	"reflect"
	"testing"
)

var fillSource = Source{Standard: "TS 44.060", Version: "19.0.0", Clause: "11.2.34"}

func TestZeroFillReport(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xf0}, 3, c)
	if err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	for i := 0; i < 3; i++ {
		if err = r.EnterIndex("Entries", i); err != nil {
			t.Fatal(err)
		}
		v, readErr := r.ReadUintWithFill(5, fillSource, "Timing")
		if readErr != nil {
			t.Fatal(readErr)
		}
		if (i == 0 && v != 28) || (i != 0 && v != 0) {
			t.Fatalf("value[%d]=%d", i, v)
		}
		if err = r.Observe(0, 5, v, fillSource, "Timing"); err != nil {
			t.Fatal(err)
		}
		r.LeaveIndex()
	}
	report := c.TakeReceiverReport()
	if len(report.ZeroFill) != 3 || report.Dropped != 0 {
		t.Fatalf("report: %+v", report)
	}
	for i, want := range []Location{{Path: "Entries[0]/Timing", BitOffset: 3, Width: 2, Source: fillSource}, {Path: "Entries[1]/Timing", BitOffset: 5, Width: 5, Source: fillSource}, {Path: "Entries[2]/Timing", BitOffset: 10, Width: 5, Source: fillSource}} {
		if report.ZeroFill[i] != want {
			t.Fatalf("fill[%d]=%+v want %+v", i, report.ZeroFill[i], want)
		}
	}
	if observed := c.TakeReport(); len(observed.Observations) != 0 || observed.Dropped != 0 {
		t.Fatalf("inferred zeros marked received: %+v", observed)
	}
	if wire := r.Wire(); wire.ImplicitZeros != 12 || !reflect.DeepEqual(wire.ImplicitSpans, []ImplicitSpan{{At: 3, Count: 12}}) {
		t.Fatalf("wire: %+v", wire)
	}
}

func TestDiscardedFillAndChoiceRelease(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits([]byte{0xff}, 1, c)
	if err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	var choice ChoiceErrors
	loser := choice.ForkScoped(r)
	if _, err = loser.ReadUintWithFill(4, fillSource, "Loser"); err != nil {
		t.Fatal(err)
	}
	loser.Discard()
	winner := choice.ForkScoped(r)
	if _, err = winner.ReadUintWithFill(2, fillSource, "Winner"); err != nil {
		t.Fatal(err)
	}
	r.Commit(winner)
	choice.Release()
	report := c.TakeReceiverReport()
	if len(report.ZeroFill) != 1 || report.ZeroFill[0].Path != "Winner" || report.ZeroFill[0].Width != 1 {
		t.Fatalf("trial leak: %+v", report)
	}
	for i := 0; i < 20; i++ {
		var next ChoiceErrors
		arm := next.ForkScoped(r)
		if _, err = arm.ReadUintWithFill(1, fillSource, "Discarded"); err != nil {
			t.Fatal(err)
		}
		next.Release()
	}
	if next := c.TakeReceiverReport(); len(next.ZeroFill) != 0 || next.Dropped != 0 {
		t.Fatalf("release leak: %+v", next)
	}
	if report.ZeroFill[0].Path != "Winner" {
		t.Fatal("returned report changed")
	}
}

func TestDiscardedSnapshotCannotCommitFill(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits(nil, 0, c)
	if err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	arm := r.Fork()
	if _, err = arm.ReadUintWithFill(3, fillSource, "Abandoned"); err != nil {
		t.Fatal(err)
	}
	arm.Discard()
	r.Commit(arm)
	if got := c.TakeReceiverReport(); len(got.ZeroFill) != 0 || got.Dropped != 0 {
		t.Fatalf("discarded fill committed: %+v", got)
	}
}

func TestZeroFillReplayAndStrictScope(t *testing.T) {
	r, err := NewReaderBits([]byte{0xe0}, 3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadUintWithFill(4, fillSource); err == nil {
		t.Fatal("fill enabled outside source scope")
	}
	r, err = NewReaderBits([]byte{0xe0}, 3, NewReceiverCollector())
	if err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	value, err := r.ReadUintWithFill(8, fillSource)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := SealBits(value, BitString{Bytes: []byte{0xe0}, BitLength: 3}, 3, BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	w := NewWriter()
	w.WithWire(wire)
	if err = w.WriteUint(value, 8); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(BitString{})
	if err != nil || out.BitLength != 3 || len(out.Bytes) != 1 || out.Bytes[0] != 0xe0 {
		t.Fatalf("received replay: %+v %v", out, err)
	}
	fresh := NewWriter()
	if err = fresh.WriteUint(value, 8); err != nil {
		t.Fatal(err)
	}
	sent, err := fresh.FinishBits(BitString{})
	if err != nil || sent.BitLength != 8 || sent.Bytes[0] != 0xe0 {
		t.Fatalf("fresh sender: %+v %v", sent, err)
	}
	w = NewWriter()
	w.WithWire(wire)
	if err = w.WriteUint(value|1, 8); err == nil {
		t.Fatal("inferred bit edited")
	}
}

func TestZeroFillLiterals(t *testing.T) {
	c := NewReceiverCollector()
	r, err := NewReaderBits(nil, 0, c)
	if err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	if err = r.ExpectWithFill("000", fillSource, "Selector"); err != nil {
		t.Fatal(err)
	}
	if report := c.TakeReceiverReport(); len(report.ZeroFill) != 1 || report.ZeroFill[0].BitOffset != 0 || report.ZeroFill[0].Width != 3 {
		t.Fatalf("literal fill missing: %+v", report)
	}
	if err = r.ExpectWithFill("1", fillSource, "Incorrect"); err == nil {
		t.Fatal("incorrect inferred literal accepted")
	}
	if report := c.TakeReceiverReport(); len(report.ZeroFill) != 0 {
		t.Fatalf("failed literal reported: %+v", report)
	}
}
