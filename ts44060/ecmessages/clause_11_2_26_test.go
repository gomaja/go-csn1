package ecmessages_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/ecmessages"
)

// TS 44.060 V19.0.0 §11.2.26 table 11.2.26.1, pp17892–17901:
// 2 page bits + 1 address selector + 6 Global TFI bits + 2 release bits
// + 4 cause bits. §11.2.0.1 selects this downlink message with 001000.
// §10.3.1.1 figure 10.3.1.1.1 supplies the 184-bit CS-1 carrier.
func releaseBlock(t testing.TB, body string, ignored bool) []byte {
	t.Helper()
	bits := "01000000001000" + body
	if len(bits) > 184 {
		t.Fatal("synthetic content exceeds CS-1")
	}
	if ignored {
		bits += "1"
	} else {
		bits += "0"
	}
	for len(bits) < 184 {
		if ignored || (byte(0x2b)>>uint(7-len(bits)%8))&1 != 0 {
			bits += "1"
		} else {
			bits += "0"
		}
	}
	out := make([]byte, 23)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func releaseReader(t testing.TB, data []byte, bits int) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(data, bits, c)
	if err != nil {
		t.Fatal(err)
	}
	releaseHeader(t, r)
	return r, c
}
func releaseHeader(t testing.TB, r *runtime.Reader) {
	t.Helper()
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, err := controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); err != nil {
		t.Fatal(err)
	}
	tag, err := r.ReadUint(6)
	if err != nil || tag != 8 || r.Position() != 14 {
		t.Fatal("downlink TBF selector", tag, err)
	}
}
func releaseEncode(v ecmessages.PacketTBFReleaseMessageContent, wire runtime.WireInfo, canonical bool) (runtime.BitString, error) {
	w := runtime.NewWriter()
	if canonical {
		v = runtime.Canonical(v)
		wire = runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
	}
	w.WithWire(wire)
	if _, err := w.PushLimit(184); err != nil {
		return runtime.BitString{}, err
	}
	if err := w.WriteUint(0x40, 8); err != nil {
		return runtime.BitString{}, err
	}
	if err := w.WriteUint(8, 6); err != nil {
		return runtime.BitString{}, err
	}
	if err := ecmessages.EncodePacketTBFReleaseMessageContentTo(w, v); err != nil {
		return runtime.BitString{}, err
	}
	return w.FinishBits(runtime.BitString{})
}
func releaseWire(t testing.TB, data []byte, r *runtime.Reader) runtime.WireInfo {
	t.Helper()
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	return wire
}

// Table 11.2.26.2 p17910 permits only 0000 and 0010 for sending.
// Every other code has abnormal-release receive meaning, and is retained raw.
func TestClause11226ReleaseCauses(t *testing.T) {
	count := 0
	for page := 0; page < 4; page++ {
		for direction := 0; direction < 2; direction++ {
			for _, tfi := range []int{0, 31} {
				for flags := 0; flags < 4; flags++ {
					for cause := 0; cause < 16; cause++ {
						for _, ignored := range []bool{false, true} {
							body := fmt.Sprintf("%02b0%01b%05b%02b%04b", page, direction, tfi, flags, cause)
							data := releaseBlock(t, body, ignored)
							r, c := releaseReader(t, data, 184)
							v, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(r)
							if err != nil {
								t.Fatal(err)
							}
							regions := c.TakeReceiverReport().Regions
							if r.Position() != 184 || v.ReceiverRegionEnds() != ignored || (!ignored && len(regions) != 0) || (ignored && (len(regions) != 1 || regions[0].Label != runtime.Ignore)) {
								t.Fatalf("valid release ignored=%v at %d regions=%+v", ignored, r.Position(), regions)
							}

							outer := v.PAGEMODEGroupFallback.Known
							if outer == nil || int(outer.PAGEMODE) != page {
								t.Fatal("page lost")
							}
							addr := outer.GLOBALTFIGroupFallback.Known
							if addr == nil {
								t.Fatal("address lost")
							}
							tf := addr.GLOBALTFI.UPLINKTFIChoice
							if direction == 0 {
								if tf.UPLINKTFI == nil || int(tf.UPLINKTFI.UPLINKTFI) != tfi {
									t.Fatal("uplink TFI")
								}
							} else if tf.DOWNLINKTFI == nil || int(tf.DOWNLINKTFI.DOWNLINKTFI) != tfi {
								t.Fatal("downlink TFI")
							}
							fields := addr.UPLINKRELEASEGroupFallback.Known
							if fields == nil || int(fields.UPLINKRELEASE) != flags>>1 || int(fields.DOWNLINKRELEASE) != flags&1 || int(fields.TBFRELEASECAUSE) != cause {
								t.Fatal("release fields lost")
							}
							allowed := cause == 0 || cause == 2
							report := c.TakeReport()
							if allowed {
								if len(report.Observations) != 0 {
									t.Fatal("assigned cause reported")
								}
							} else {
								if len(report.Observations) != 1 || report.Observations[0].Value != uint64(cause) || report.Observations[0].Location.BitOffset != 25 || report.Observations[0].Location.Width != 4 || report.Observations[0].Location.Source.Standard != "TS 44.060" || report.Observations[0].Location.Source.Clause != "11.2.26" {
									t.Fatalf("reserved cause %d report %+v", cause, report)
								}
							}
							wire := releaseWire(t, data, r)
							got, err := releaseEncode(v, wire, false)
							if err != nil || got.BitLength != 184 || !bytes.Equal(got.Bytes, data) {
								t.Fatalf("reserved/plain replay cause=%d: %v", cause, err)
							}
							_, explicitErr := ecmessages.EncodePacketTBFReleaseMessageContentCanonicalAtLength(v, 23)
							if (explicitErr == nil) != (allowed && !ignored) {
								t.Fatalf("explicit canonical sender cause=%d ignored=%v: %v", cause, ignored, explicitErr)
							}
							_, err = releaseEncode(v, runtime.WireInfo{}, true)
							if (err == nil) != (allowed && !ignored) {
								t.Fatalf("canonical sender cause=%d: %v", cause, err)
							}
							fresh := runtime.Canonical(v)
							_, err = releaseEncode(fresh, runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire, false)
							if (err == nil) != (allowed && !ignored) {
								t.Fatalf("fresh sender cause=%d: %v", cause, err)
							}
							// The two assigned causes can be edited without changing another field.
							if allowed && !ignored {
								fields.TBFRELEASECAUSE ^= 2
								edited, err := releaseEncode(v, runtime.WireInfo{}, true)
								if err != nil {
									t.Fatal(err)
								}
								er, _ := releaseReader(t, edited.Bytes, 184)
								ev, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(er)
								if err != nil || ev.PAGEMODEGroupFallback.Known.GLOBALTFIGroupFallback.Known.UPLINKRELEASEGroupFallback.Known.TBFRELEASECAUSE != uint8(cause^2) {
									t.Fatal("cause edit", err)
								}
							}
							count++
						}
					}
				}
			}
		}
	}
	if count != 2048 {
		t.Fatal("vector count", count)
	}
}

func TestClause11226BoundAndErrors(t *testing.T) {
	data := releaseBlock(t, "100111111100010", false)
	// A complete control block cannot consume the following three-bit sentinel.
	raw := append(bytes.Clone(data), 0xa0)
	r, err := runtime.NewChoiceReaderBits(raw, 187, nil)
	if err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(184)
	if err != nil {
		t.Fatal(err)
	}
	releaseHeader(t, r)
	if _, err = ecmessages.DecodePacketTBFReleaseMessageContentFrom(r); err != nil {
		t.Fatal(err)
	}
	if err = r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	sentinel, err := r.ReadUint(3)
	if err != nil || sentinel != 5 {
		t.Fatal("sentinel", sentinel, err)
	}
	// Every cut in the required content selects its printed receiver error arm.
	for cut := 0; cut < 15; cut++ {
		r, c := releaseReader(t, data, 184)
		if _, err = r.PushLimit(cut); err != nil {
			t.Fatal(err)
		}
		v, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		report := c.TakeReceiverReport()
		label := runtime.NonDistributionPartError
		offset := 23
		if cut < 2 {
			label = runtime.DistributionPartError
			offset = 14
		} else if cut < 9 {
			label = runtime.AddressInformationPartError
			offset = 16
		}
		if len(report.Regions) != 1 || report.Regions[0].Label != label || report.Regions[0].Location.BitOffset != offset || r.Position() != 14+cut || !v.ReceiverRegionEnds() {
			t.Fatalf("cut %d: %+v end=%d", cut, report, r.Position())
		}
		// The exact truncated received extent must replay without adding a sender suffix.
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: 14 + cut}, 14+cut, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		w.WithWire(wire)
		if _, err = w.PushLimit(14 + cut); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(0x40, 8); err != nil {
			t.Fatal(err)
		}
		if err = w.WriteUint(8, 6); err != nil {
			t.Fatal(err)
		}
		if err = ecmessages.EncodePacketTBFReleaseMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		got, err := w.FinishBits(runtime.BitString{})
		if err != nil || got.BitLength != 14+cut {
			t.Fatalf("truncated replay cut=%d: %v", cut, err)
		}
		for i := 0; i < got.BitLength; i++ {
			if (got.Bytes[i/8]>>uint(7-i%8))&1 != (data[i/8]>>uint(7-i%8))&1 {
				t.Fatal("truncated replay bit", i)
			}
		}

	}
	// An invalid address selector selects the address-information error region.
	malformed := releaseBlock(t, "101111111100010", false)
	mr, mc := releaseReader(t, malformed, 184)
	mv, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(mr)
	if err != nil || !mv.ReceiverRegionEnds() {
		t.Fatal("address error", err)
	}
	report := mc.TakeReceiverReport()
	if len(report.Regions) != 1 || report.Regions[0].Label != runtime.AddressInformationPartError {
		t.Fatal("address error report")
	}
	got, err := releaseEncode(mv, releaseWire(t, malformed, mr), false)
	if err != nil || !bytes.Equal(got.Bytes, malformed) {
		t.Fatal("address error replay", err)
	}
}

// The fixed-width message reaches its padding at block bit 29. Supplement the
// complete-block matrix with phase-congruent From/To origins and verify that
// every other phase refuses before I/O.
// TS 44.060 §11 p12940 defines the octet-phased 00101011 padding sequence.
func TestClause11226PaddingPhases(t *testing.T) {
	for _, phase := range []int{0, 1, 2, 3, 4, 5, 6, 7, 14, 22} {
		bits := string(bytes.Repeat([]byte{'0'}, phase)) + "100111111100010" + "0"
		for len(bits) < 184 {
			if (byte(0x2b)>>uint(7-len(bits)%8))&1 != 0 {
				bits += "1"
			} else {
				bits += "0"
			}
		}
		data := make([]byte, 23)
		for i, b := range bits {
			if b == '1' {
				data[i/8] |= 1 << uint(7-i%8)
			}
		}
		r, err := runtime.NewReaderBits(data, 184, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.ReadUint(phase); err != nil {
			t.Fatal(err)
		}
		v, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(r)
		if phase%8 != 6 {
			var phaseError *ecmessages.ContentPhaseError
			if !errors.As(err, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || r.Position() != phase {
				t.Fatalf("reader phase %d changed cursor or error: %v", phase, err)
			}
			target := runtime.NewWriter()
			if e := target.WriteUint(0, phase); e != nil {
				t.Fatal(e)
			}
			writerErr := ecmessages.EncodePacketTBFReleaseMessageContentTo(target, v)
			if !errors.As(writerErr, &phaseError) || phaseError.SourceOrigin() != 14 || phaseError.BitPosition() != phase || target.Position() != phase {
				t.Fatalf("writer phase %d changed cursor or error: %v", phase, writerErr)
			}
			continue
		}

		if err != nil || r.Position() != 184 {
			t.Fatalf("phase %d: %v", phase, err)
		}
		w := runtime.NewWriter()
		w.WithWire(releaseWire(t, data, r))
		if err = w.WriteUint(0, phase); err != nil {
			t.Fatal(err)
		}
		if err = ecmessages.EncodePacketTBFReleaseMessageContentTo(w, v); err != nil {
			t.Fatal(err)
		}
		got, err := w.FinishBits(runtime.BitString{})
		if err != nil || !bytes.Equal(got.Bytes, data) {
			t.Fatal("phase replay", phase, err)
		}
	}
}

func FuzzClause11226Replay(f *testing.F) {
	for _, cause := range []int{0, 1, 2, 15} {
		f.Add(releaseBlock(f, fmt.Sprintf("10011111110%04b", cause), false))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) != 23 {
			return
		}
		data = bytes.Clone(data)
		data[0] = 0x40
		data[1] = (data[1] & 3) | (8 << 2)
		r, _ := releaseReader(t, data, 184)
		v, err := ecmessages.DecodePacketTBFReleaseMessageContentFrom(r)
		if err != nil {
			return
		}
		got, err := releaseEncode(v, releaseWire(t, data, r), false)
		if err != nil || got.BitLength != 184 || !bytes.Equal(got.Bytes, data) {
			t.Fatal("accepted input did not replay exactly", err)
		}
	})
}
