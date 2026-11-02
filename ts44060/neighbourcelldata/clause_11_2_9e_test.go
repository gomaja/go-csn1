package neighbourcelldata

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
)

// TS 44.060 V19.0.0 §§11.2.9e and 11.2.17a distinguish finite records,
// the zero terminator and the 31 continuation. Finite width is 8*CD_LENGTH.
func TestPrintedDynamicContainerLengths(t *testing.T) {
	for pd := byte(0); pd < 8; pd++ {
		for n := byte(1); n < 31; n++ {
			t.Run(fmt.Sprintf("pd%d/length%d", pd, n), func(t *testing.T) {
				data := bytes.Repeat([]byte{0xa5}, int(n))
				raw := append([]byte{pd<<5 | n}, data...)
				raw = append(raw, 0)
				v, err := DecodeContainerRepetitionStruct(raw)
				if err != nil {
					t.Fatal(err)
				}
				if len(v.Value.PDChoiceList) != 1 {
					t.Fatalf("records=%d want 1", len(v.Value.PDChoiceList))
				}
				x := v.Value.PDChoiceList[0].CONTAINERDATA
				if x == nil || x.PD != pd || x.CDLENGTH != n || x.CONTAINERDATA.BitLength != int(n)*8 || !bytes.Equal(x.CONTAINERDATA.Bytes, data) {
					t.Fatalf("typed record %+v", x)
				}
				out, err := EncodeContainerRepetitionStruct(v.Value)
				if err != nil || !bytes.Equal(raw, out) {
					t.Fatalf("replay %x %v", out, err)
				}
			})
		}
	}
}
func TestPrintedDynamicContainerChangesPD(t *testing.T) {
	raw := []byte{1, 0xa5, 0x22, 0x5a, 0x5b, 0}
	v, err := DecodeContainerRepetitionStruct(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Value.PDChoiceList) != 2 {
		t.Fatalf("records=%d", len(v.Value.PDChoiceList))
	}
	a, b := v.Value.PDChoiceList[0].CONTAINERDATA, v.Value.PDChoiceList[1].CONTAINERDATA
	if a == nil || b == nil || a.PD != 0 || a.CDLENGTH != 1 || b.PD != 1 || b.CDLENGTH != 2 || !bytes.Equal(b.CONTAINERDATA.Bytes, []byte{0x5a, 0x5b}) {
		t.Fatalf("records %+v %+v", a, b)
	}
}

// TS 44.060 V19.0.0 §11.2.9e: the MAC header and six-bit message
// selector leave an 18-bit fixed content prefix before the first record.
func containerBlockBits(prefix string, extent int) []byte {
	s := prefix
	if len(s) < extent {
		s += "0"
	}
	for len(s) < extent {
		if (byte(0x2b)>>uint(7-len(s)%8))&1 != 0 {
			s += "1"
		} else {
			s += "0"
		}
	}
	out := make([]byte, (extent+7)/8)
	for i, b := range s {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func containerContentPrefix(page, direction, tfi int, identity bool) string {
	s := fmt.Sprintf("01000000%06b%02b0%01b%05b", 12, page, direction, tfi)
	s += "10010001"
	if identity {
		s += "1" + fmt.Sprintf("%010b%06b", 1001, 53)
	} else {
		s += "0"
	}
	return s
}
func readContainerBlock(t testing.TB, data []byte, extent int) (PacketNeighbourCellDataMessageContent, *runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(data, extent, c)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	tag, err := r.ReadUint(6)
	if err != nil || tag != 12 {
		t.Fatalf("tag %d: %v", tag, err)
	}
	v, err := DecodePacketNeighbourCellDataMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	return v, r, c
}
func acceptedContainer(t testing.TB, v PacketNeighbourCellDataMessageContent) *PacketNeighbourCellDataMessageContentPAGEMODEGroupFallbackKnownGlobalTFIGroupFallbackKnownCONTAINERIDGroupFallbackKnown {
	t.Helper()
	a := v.PAGEMODEGroupFallback.Known
	if a == nil || a.GlobalTFIGroupFallback.Known == nil || a.GlobalTFIGroupFallback.Known.CONTAINERIDGroupFallback.Known == nil {
		t.Fatalf("accepted prefix lost: %+v", v)
	}
	return a.GlobalTFIGroupFallback.Known.CONTAINERIDGroupFallback.Known
}
func encodeContainerBlock(t testing.TB, original []byte, r *runtime.Reader, v PacketNeighbourCellDataMessageContent, canonical bool) []byte {
	t.Helper()
	w := runtime.NewWriter()
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: original, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	if canonical {
		wire = runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
		v = runtime.Canonical(v)
	}
	w.WithWire(wire)
	if _, err = w.PushLimit(184); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(0x40, 8); err != nil {
		t.Fatal(err)
	}
	if err = w.WriteUint(12, 6); err != nil {
		t.Fatal(err)
	}
	if err = EncodePacketNeighbourCellDataMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil || out.BitLength != 184 {
		t.Fatalf("encoded %d: %v", out.BitLength, err)
	}
	return out.Bytes
}
func TestClause1129eCompleteDownlinkContainers(t *testing.T) {
	for code := 0; code < 32; code++ {
		for pd := 0; pd < 2; pd++ {
			t.Run(fmt.Sprintf("pd%d/length%d", pd, code), func(t *testing.T) {
				prefix := containerContentPrefix(1, 0, 21, false)
				want := 0
				if code == 0 {
					prefix += "00000000"
				} else {
					prefix += fmt.Sprintf("%03b%05b", pd, code)
					n := code
					if code >= 18 {
						n = 18
					}
					prefix += strings.Repeat("10100101", n)
					if code < 18 {
						prefix += "00000000"
					}
					if code <= 18 || code == 31 {
						want = 1
					}
				}
				raw := containerBlockBits(prefix, 184)
				v, r, _ := readContainerBlock(t, raw, 184)
				known := acceptedContainer(t, v)
				if r.Position() != 184 || len(known.CONTAINER.PDChoiceList) != want {
					t.Fatalf("code%d items %d at %d", code, len(known.CONTAINER.PDChoiceList), r.Position())
				}
				if want == 1 {
					x := known.CONTAINER.PDChoiceList[0]
					if code == 31 {
						if x.CDLENGTH == nil || x.CDLENGTH.PD != uint8(pd) || len(x.CDLENGTH.CONTAINERDATA) != 18 {
							t.Fatalf("continuation %+v", x)
						}
					} else if x.CONTAINERDATA == nil || x.CONTAINERDATA.PD != uint8(pd) || int(x.CONTAINERDATA.CDLENGTH) != code || x.CONTAINERDATA.CONTAINERDATA.BitLength != code*8 {
						t.Fatalf("finite %+v", x)
					}
				}
				if replay := encodeContainerBlock(t, raw, r, v, false); !bytes.Equal(replay, raw) {
					t.Fatalf("replay %x want %x", replay, raw)
				}
				if code <= 18 || code == 31 {
					if canonical := encodeContainerBlock(t, raw, r, v, true); !bytes.Equal(canonical, raw) {
						t.Fatalf("canonical %x want %x", canonical, raw)
					}
				}
			})
		}
	}
}
func TestClause1129eIdentityAndEdit(t *testing.T) {
	for page := 0; page < 4; page++ {
		for d := 0; d < 2; d++ {
			for _, tfi := range []int{0, 31} {
				for _, identity := range []bool{false, true} {
					prefix := containerContentPrefix(page, d, tfi, identity) + "001000011010010100000000"
					raw := containerBlockBits(prefix, 184)
					v, r, c := readContainerBlock(t, raw, 184)
					known := acceptedContainer(t, v)
					if int(v.PAGEMODEGroupFallback.Known.PAGEMODE) != page || known.CONTAINERINDEX != 17 {
						t.Fatal("header fields lost")
					}
					g := v.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Known.GlobalTFI.UPLINKTFIChoice
					if d == 0 {
						if g.UPLINKTFI == nil || int(g.UPLINKTFI.UPLINKTFI) != tfi {
							t.Fatal("uplink TFI lost")
						}
					} else if g.DOWNLINKTFI == nil || int(g.DOWNLINKTFI.DOWNLINKTFI) != tfi {
						t.Fatal("downlink TFI lost")
					}
					if known.CONTAINERID != 2 || (known.ARFCNGroup != nil) != identity {
						t.Fatal("cell identity presence lost")
					}
					if identity && (known.ARFCNGroup.ARFCN != 1001 || known.ARFCNGroup.BSIC != 53) {
						t.Fatal("cell identity value lost")
					}
					if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
						t.Fatalf("unexpected region %+v", report)
					}
					if replay := encodeContainerBlock(t, raw, r, v, false); !bytes.Equal(replay, raw) {
						t.Fatal("identity replay")
					}
					known.CONTAINER.PDChoiceList[0].CONTAINERDATA.CONTAINERDATA.Bytes[0] = 0x5a
					edited := encodeContainerBlock(t, raw, r, v, true)
					ev, _, _ := readContainerBlock(t, edited, 184)
					if got := acceptedContainer(t, ev).CONTAINER.PDChoiceList[0].CONTAINERDATA.CONTAINERDATA.Bytes[0]; got != 0x5a {
						t.Fatalf("edited data=%02x", got)
					}
				}
			}
		}
	}
}
func TestClause1129eTerminatorTruncationAtEveryOrigin(t *testing.T) {
	for phase := 0; phase < 8; phase++ {
		for cut := 0; cut < 8; cut++ {
			prefix := strings.Repeat("0", phase) + "0000000110100101" + strings.Repeat("0", cut)
			extent := len(prefix)
			raw := containerBlockBits(prefix, extent)
			r, err := runtime.NewChoiceReaderBits(raw, extent, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.ReadUint(phase); err != nil {
				t.Fatal(err)
			}
			v, err := DecodeContainerRepetitionStructFrom(r)
			if err != nil {
				t.Fatalf("phase%d/cut%d: %v", phase, cut, err)
			}
			if len(v.PDChoiceList) != 1 || v.PDChoiceList[0].CONTAINERDATA == nil || v.PDChoiceList[0].CONTAINERDATA.CDLENGTH != 1 || r.Position() != phase+16 {
				t.Fatalf("phase%d/cut%d lost prefix or consumed truncated terminator: %+v at%d", phase, cut, v, r.Position())
			}
		}
	}
}

// Content begins at source phase 6; all other phases fail before I/O.
// TS 24.007 V20.0.0 Annex B §B.1.2.2 keeps L/H padding positioned.
func TestPositionedContainerContentPadding(t *testing.T) {
	for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		prefix := strings.Repeat("0", phase) + containerContentPrefix(1, 0, 21, false)[14:] + "000000011010010100000000"
		extent := phase + 80
		raw := containerBlockBits(prefix, extent)
		c := runtime.NewReceiverCollector()
		r, err := runtime.NewChoiceReaderBits(raw, extent, c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		v, err := DecodePacketNeighbourCellDataMessageContentFrom(r)
		if phase%8 != 6 {
			var phaseError *ContentPhaseError
			if !errors.As(err, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || r.Position() != phase {
				t.Fatalf("reader phase %d changed cursor or error: %v", phase, err)
			}
			target := runtime.NewWriter()
			if e := target.WriteUint(0, phase); e != nil {
				t.Fatal(e)
			}
			writerErr := EncodePacketNeighbourCellDataMessageContentTo(target, v)
			if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || target.Position() != phase {
				t.Fatalf("writer phase %d changed cursor or error: %v", phase, writerErr)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		known := acceptedContainer(t, v)
		if len(known.CONTAINER.PDChoiceList) != 1 || r.Position() != extent {
			t.Fatalf("phase%d fields or extent lost", phase)
		}
		if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
			t.Fatalf("phase%d valid padding entered error region: %+v", phase, report)
		}
		w := runtime.NewWriter()
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
		if _, err = w.PushLimit(extent); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		if err = EncodePacketNeighbourCellDataMessageContentTo(w, runtime.Canonical(v)); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil || out.BitLength != extent || !bytes.Equal(out.Bytes, raw) {
			t.Fatalf("phase%d canonical %x/%d want%x/%d: %v", phase, out.Bytes, out.BitLength, raw, extent, err)
		}
	}
}

// TS 44.060 V19.0.0 §11.1.4.4 p13129 preserves complete components
// when the final printed terminator is syntactically incorrect.
func TestContainerMalformedTerminatorPreservesAcceptedRecord(t *testing.T) {
	prefix := containerContentPrefix(1, 0, 21, false) + "00010000" + strings.Repeat("10100101", 16) + "0000001011011110"
	raw := containerBlockBits(prefix, 184)
	v, r, _ := readContainerBlock(t, raw, 184)
	known := acceptedContainer(t, v)
	if len(known.CONTAINER.PDChoiceList) != 1 || known.CONTAINER.PDChoiceList[0].CONTAINERDATA.CDLENGTH != 16 {
		t.Fatal("accepted record was lost")
	}
	if out := encodeContainerBlock(t, raw, r, v, false); !bytes.Equal(out, raw) {
		t.Fatal("malformed terminator replay changed")
	}
	canonical := encodeContainerBlock(t, raw, r, v, true)
	if canonical[21] != 0 || canonical[22] != 0x2b {
		t.Fatalf("canonical terminator/padding=%x", canonical[21:])
	}
}
func TestContainerContinuationCannotConsumeFollowingSentinel(t *testing.T) {
	prefix := containerContentPrefix(1, 0, 21, false) + "00011111" + strings.Repeat("10100101", 18)
	raw := append(containerBlockBits(prefix, 184), 0x5a)
	r, err := runtime.NewChoiceReaderBits(raw, 192, nil)
	if err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	if got, e := r.ReadUint(6); e != nil || got != 12 {
		t.Fatalf("selector %d: %v", got, e)
	}
	v, err := DecodePacketNeighbourCellDataMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	known := acceptedContainer(t, v)
	if len(known.CONTAINER.PDChoiceList) != 1 || known.CONTAINER.PDChoiceList[0].CDLENGTH == nil || len(known.CONTAINER.PDChoiceList[0].CDLENGTH.CONTAINERDATA) != 18 {
		t.Fatal("continuation extent changed")
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(8)
	if err != nil || sentinel != 0x5a {
		t.Fatalf("sentinel %02x: %v", sentinel, err)
	}
}

func TestContainerRepetitionGrowingInput(t *testing.T) {
	previous := float64(0)
	for _, n := range []int{16, 64, 256} {
		raw := append(bytes.Repeat([]byte{1, 0xa5}, n), 0)
		v, err := DecodeContainerRepetitionStruct(raw)
		if err != nil || len(v.Value.PDChoiceList) != n {
			t.Fatalf("records%d: got%d: %v", n, len(v.Value.PDChoiceList), err)
		}
		allocations := testing.AllocsPerRun(1, func() {
			if _, err := DecodeContainerRepetitionStruct(raw); err != nil {
				panic(err)
			}
		})
		if !raceEnabled && previous > 0 && allocations > 5*previous {
			t.Fatalf("fourfold input grew allocations from%.0f to%.0f", previous, allocations)
		}
		previous = allocations
		out, err := EncodeContainerRepetitionStruct(v.Value)
		if err != nil || !bytes.Equal(out, raw) {
			t.Fatalf("records%d replay: %v", n, err)
		}
	}
}

const containerClause = "11.2.9e"
const containerSelector = 12

func encodeContainerContent(w *runtime.Writer, v PacketNeighbourCellDataMessageContent) error {
	return EncodePacketNeighbourCellDataMessageContentTo(w, v)
}

// TS 44.060 V19.0.0 code tables: §§11.2.9e pp16187–16197 and
// 11.2.17a pp16647–16657. Application discard does not change syntax.
func TestPrintedCodeTablePolicy(t *testing.T) {
	for pd := byte(0); pd < 8; pd++ {
		for length := byte(1); length <= 30; length++ {
			t.Run(fmt.Sprintf("finite/pd%d/length%d", pd, length), func(t *testing.T) {
				raw := append([]byte{pd<<5 | length}, bytes.Repeat([]byte{0xa5}, int(length))...)
				raw = append(raw, 0)
				decoded, err := DecodeContainerRepetitionStruct(raw)
				if err != nil {
					t.Fatal(err)
				}
				replay, err := EncodeContainerRepetitionStruct(decoded.Value)
				if err != nil || !bytes.Equal(replay, raw) {
					t.Fatalf("finite replay: %v", err)
				}
				var collector runtime.Collector
				r, err := runtime.NewReaderBits(raw, len(raw)*8, &collector)
				if err != nil {
					t.Fatal(err)
				}
				fresh, err := DecodeContainerRepetitionStructFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				report := collector.TakeReport()
				want := 0
				if pd > 1 {
					want++
				}
				if length > 18 {
					want++
				}
				if len(report.Observations) != want {
					t.Fatalf("reserved observations=%d want %d", len(report.Observations), want)
				}
				i := 0
				for _, o := range report.Observations {
					offset, width, value := 0, 3, uint64(pd)
					if pd <= 1 || i == 1 {
						offset, width, value = 3, 5, uint64(length)
					}
					if o.Value != value || o.Location.BitOffset != offset || o.Location.Width != width || o.Location.Source.Clause != containerClause || !strings.Contains(o.Location.Path, []string{"PD", "CDLENGTH"}[offset/3]) {
						t.Fatalf("wrong observation: %+v", o)
					}
					i++
				}
				fresh = freshContainerValue(fresh)
				_, err = encodeFreshContainer(fresh, len(raw)*8)
				requireContainerSendPolicy(t, err, want > 0)
				_, err = EncodeContainerRepetitionStructCanonicalAtLength(decoded.Value, len(raw))
				requireContainerSendPolicy(t, err, want > 0)
			})
		}
		t.Run(fmt.Sprintf("continuation/pd%d", pd), func(t *testing.T) {
			raw := []byte{pd<<5 | 31, 0xa5, 0x5a}
			decoded, err := DecodeContainerRepetitionStruct(raw)
			if err != nil {
				t.Fatal(err)
			}
			if len(decoded.Value.PDChoiceList) != 1 || decoded.Value.PDChoiceList[0].CDLENGTH == nil || decoded.Value.PDChoiceList[0].CDLENGTH.PD != pd {
				t.Fatal("continuation PD lost")
			}
			replay, err := EncodeContainerRepetitionStruct(decoded.Value)
			if err != nil || !bytes.Equal(replay, raw) {
				t.Fatalf("continuation replay: %v", err)
			}
			var collector runtime.Collector
			r, err := runtime.NewReaderBits(raw, len(raw)*8, &collector)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := DecodeContainerRepetitionStructFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			report := collector.TakeReport()
			want := 0
			if pd > 1 {
				want = 1
			}
			if len(report.Observations) != want {
				t.Fatalf("continuation observations=%d want%d", len(report.Observations), want)
			}
			if want > 0 {
				o := report.Observations[0]
				if o.Value != uint64(pd) || o.Location.BitOffset != 0 || o.Location.Width != 3 || o.Location.Source.Clause != containerClause {
					t.Fatalf("continuation observation %+v", o)
				}
			}
			fresh = freshContainerValue(fresh)
			_, err = encodeFreshContainer(fresh, len(raw)*8)
			requireContainerSendPolicy(t, err, pd > 1)
			_, err = EncodeContainerRepetitionStructCanonicalAtLength(decoded.Value, len(raw))
			requireContainerSendPolicy(t, err, pd > 1)
		})
	}
	// Failed finite/continuation trials at the terminator must not leak PD reports.
	for spare := byte(0); spare < 8; spare++ {
		var collector runtime.Collector
		raw := []byte{spare << 5, 0x2b}
		r, err := runtime.NewReaderBits(raw, 16, &collector)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = DecodeContainerRepetitionStructFrom(r); err != nil {
			t.Fatal(err)
		}
		if len(collector.TakeReport().Observations) != 0 {
			t.Fatal("structural terminator reported as a reserved value")
		}
	}
}
func requireContainerSendPolicy(t *testing.T, err error, reserved bool) {
	t.Helper()
	if !reserved {
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	var operation *runtime.OperationError
	if !errors.As(err, &operation) || (operation.Operation != "encode" && operation.Operation != "canonical") || operation.Kind != runtime.InvalidValue || operation.Location.Source.Clause != containerClause {
		t.Fatalf("reserved send requires a typed located refusal: %v", err)
	}
}

func TestReservedContainersInCompleteBlocks(t *testing.T) {
	for pd := 2; pd <= 7; pd++ {
		for _, code := range []int{1, 18, 31} {
			t.Run(fmt.Sprintf("pd%d/length%d", pd, code), func(t *testing.T) {
				count := code
				if code == 31 {
					count = 18
				}
				prefix := containerContentPrefix(0, 0, 0, false) + fmt.Sprintf("%03b%05b", pd, code) + strings.Repeat("10100101", count)
				if code < 18 {
					prefix += "00000000"
				}
				raw := containerBlockBits(prefix, 184)
				v, r, c := readContainerBlock(t, raw, 184)
				if len(acceptedContainer(t, v).CONTAINER.PDChoiceList) != 1 {
					t.Fatal("reserved record was not retained")
				}
				report := c.TakeReport()
				if len(report.Observations) != 1 || report.Observations[0].Value != uint64(pd) || report.Observations[0].Location.BitOffset != 32 {
					t.Fatalf("complete-block observation %+v", report)
				}
				if replay := encodeContainerBlock(t, raw, r, v, false); !bytes.Equal(replay, raw) {
					t.Fatal("complete-block replay differs")
				}
				w := runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire)
				if err := w.WriteUint(0x40, 8); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteUint(containerSelector, 6); err != nil {
					t.Fatal(err)
				}
				err := encodeContainerContent(w, runtime.Canonical(v))
				requireContainerSendPolicy(t, err, true)
			})
		}
	}
	// A record starts at bit 32: 8 header bits plus 18 octets exhaust 184.
	// Reserved finite lengths 19..30 are incomplete, not complete reserved records.
	for length := 19; length <= 30; length++ {
		prefix := containerContentPrefix(0, 0, 0, false) + fmt.Sprintf("000%05b", length) + strings.Repeat("10100101", 18)
		v, _, c := readContainerBlock(t, containerBlockBits(prefix, 184), 184)
		if len(acceptedContainer(t, v).CONTAINER.PDChoiceList) != 0 {
			t.Fatal("truncated finite record was committed")
		}
		if len(c.TakeReport().Observations) != 0 {
			t.Fatal("truncated trial leaked a reserved observation")
		}
	}
}

func FuzzContainerReplay(f *testing.F) {
	f.Add([]byte{1, 0xa5, 0})
	f.Add([]byte{0x5f, 0xa5})
	f.Add([]byte{0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		v, err := DecodeContainerRepetitionStruct(data)
		if err != nil {
			return
		}
		out, err := EncodeContainerRepetitionStruct(v.Value)
		if err != nil || !bytes.Equal(out, data) {
			t.Fatalf("accepted helper failed exact replay: %v", err)
		}
	})
}

func TestReservedContainerFreshSendRefusal(t *testing.T) {
	for pd := byte(0); pd < 8; pd++ {
		for _, length := range []byte{1, 18, 19, 30, 31} {
			count := length
			if length == 31 {
				count = 2
			}
			raw := append([]byte{pd<<5 | length}, bytes.Repeat([]byte{0xa5}, int(count))...)
			if length != 31 {
				raw = append(raw, 0)
			}
			received, err := DecodeContainerRepetitionStruct(raw)
			if err != nil {
				t.Fatal(err)
			}
			fresh := received.Value
			fresh = freshContainerValue(fresh)
			_, err = encodeFreshContainer(fresh, len(raw)*8)
			requireContainerSendPolicy(t, err, pd > 1 || length > 18 && length < 31)
			_, err = EncodeContainerRepetitionStructCanonicalAtLength(received.Value, len(raw))
			requireContainerSendPolicy(t, err, pd > 1 || length > 18 && length < 31)
		}
	}
}

// Fresh continuation data needs an explicit containing extent, as printed.
func encodeFreshContainer(v ContainerRepetitionStruct, bits int) ([]byte, error) {
	w := runtime.NewWriter()
	if _, err := w.PushLimit(bits); err != nil {
		return nil, err
	}
	if err := EncodeContainerRepetitionStructTo(w, v); err != nil {
		return nil, err
	}
	return w.Finish(runtime.BitString{})
}

// Received sender state lives beside each record, including nested records.
func freshContainerValue(v ContainerRepetitionStruct) ContainerRepetitionStruct {
	v.Wire = runtime.WireInfo{}
	v.PDChoiceList = append([]ContainerRepetitionStructPDChoiceListEntry(nil), v.PDChoiceList...)
	for i := range v.PDChoiceList {
		e := &v.PDChoiceList[i]
		if e.CONTAINERDATA != nil {
			copy := *e.CONTAINERDATA
			clearRecordWire(&copy)
			e.CONTAINERDATA = &copy
		}
		if e.CDLENGTH != nil {
			copy := *e.CDLENGTH
			clearRecordWire(&copy)
			e.CDLENGTH = &copy
		}
	}
	return v
}
func clearRecordWire(v any) {
	field := reflect.ValueOf(v).Elem().FieldByName("Wire")
	if field.IsValid() {
		field.Set(reflect.ValueOf(runtime.WireInfo{}))
	}
}
