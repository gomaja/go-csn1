package mbmsassignment_test

import (
	"bytes"
	"fmt"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlblock"
	"github.com/gomaja/go-csn1/ts44060/mbmsassignment"
	"strings"
	"testing"
)

// TS 44.060 V19.0.0 §11.2.39 pp20333–20372. The exact punctuation
// receipts at pp20355,20364,20365 retain the printed wire fields.
// Ordinary downlink selector 010100 is external (§11.2.0.1 p13366).
func assignmentBits(address, identity, flags, arm, mask, length int) (string, int) {
	bits := "01000000010100" + "10"
	if address == 0 {
		bits += "0" + "1" + "10001"
	} else {
		bits += "10" + fmt.Sprintf("%032b", 0x10203040)
	}
	if identity == 0 {
		bits += "0"
	} else {
		bits += "1" + fmt.Sprintf("%02b%0*b", identity-1, identity, (1<<uint(identity))-1) + "00"
		if identity >= 3 {
			bits += "1" + "1001"
			if identity == 4 {
				bits += "1" + "10101"
			} else {
				bits += "0"
			}
		} else {
			bits += "0"
		}
	}
	if flags&1 != 0 {
		bits += "1" + "0" + fmt.Sprintf("%024b", 0xabcdef)
	} else {
		bits += "0"
	}
	if flags&2 != 0 {
		bits += "1" + fmt.Sprintf("%08b", 0x9a)
	} else {
		bits += "0"
	}
	switch arm {
	case 0:
		bits += "00" + fmt.Sprintf("%02b", mask&3)
		if mask&4 != 0 {
			bits += "1" + fmt.Sprintf("%08b", 0xa5)
		} else {
			bits += "0"
		}
	case 1:
		bits += "01"
		if mask&1 != 0 {
			bits += "1" + fmt.Sprintf("%016b", 0x9abc)
		} else {
			bits += "0"
		}
		if mask&2 != 0 {
			bits += "1" + fmt.Sprintf("%03b00%010b", 5, 0x2aa)
		} else {
			bits += "0"
		}
		bits += fmt.Sprintf("%08b%03b", 0xa5, length) + strings.Repeat("1", length)
		if mask&4 != 0 {
			bits += "1" + fmt.Sprintf("%05b", 17)
		} else {
			bits += "0"
		}
		if mask&8 != 0 {
			bits += "1" + "101"
		} else {
			bits += "0"
		}
		if mask&16 != 0 {
			bits += "1" + "10111"
		} else {
			bits += "0"
		}
		bits += fmt.Sprintf("%08b1", 0xc3)
	case 2:
		bits += "10"
		if mask&1 != 0 {
			bits += "1" + fmt.Sprintf("%016b", 0x9abc)
		} else {
			bits += "0"
		}
		bits += fmt.Sprintf("%03b", length) + strings.Repeat("1", length)
	case 3:
		bits += "11"
	}
	end := len(bits)
	bits += "0"
	for len(bits) < 184 {
		bits += fmt.Sprint((0x2b >> uint(7-len(bits)%8)) & 1)
	}
	return bits, end
}
func assignmentBytes(bits string) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		}
	}
	return out
}
func assignmentReader(t testing.TB, data []byte, extent int) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, e := runtime.NewChoiceReaderBits(data, extent, c)
	if e != nil {
		t.Fatal(e)
	}
	ctx := runtime.RLCMACContext{Direction: runtime.RLCMACDownlink, Mode: runtime.RLCMACOrdinary, Channel: runtime.RLCMACPACCH}
	if _, e = controlblock.DecodeBlockHeaderFrom(r, ctx, controlblock.CS1); e != nil {
		t.Fatal(e)
	}
	tag, e := r.ReadUint(6)
	if e != nil || tag != 20 {
		t.Fatalf("selector %d: %v", tag, e)
	}
	return r, c
}
func assignmentEncode(t testing.TB, raw []byte, wire runtime.WireInfo, encode func(*runtime.Writer) error) []byte {
	t.Helper()
	w := runtime.NewWriter()
	w.WithWire(wire)
	if _, e := w.PushLimit(184); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(uint64(raw[0]), 8); e != nil {
		t.Fatal(e)
	}
	if e := w.WriteUint(20, 6); e != nil {
		t.Fatal(e)
	}
	if e := encode(w); e != nil {
		t.Fatal(e)
	}
	b, e := w.FinishBits(runtime.BitString{})
	if e != nil || b.BitLength != 184 {
		t.Fatalf("extent %d: %v", b.BitLength, e)
	}
	return b.Bytes
}

func TestClause11239Assignment(t *testing.T) {
	vectors := 0
	for address := 0; address < 2; address++ {
		for identity := 0; identity <= 4; identity++ {
			for flags := 0; flags < 4; flags++ {
				for arm := 0; arm < 3; arm++ {
					max := 8
					if arm == 1 {
						max = 32
					}
					if arm == 2 {
						max = 2
					}
					for mask := 0; mask < max; mask++ {
						for length := 1; length <= 5; length++ {
							bits, _ := assignmentBits(address, identity, flags, arm, mask, length)
							if len(bits) > 184 {
								continue
							}
							raw := assignmentBytes(bits)
							t.Logf("input=%x", raw)
							r, c := assignmentReader(t, raw, 184)
							v, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
							if e != nil || r.Position() != 184 || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Ignored != nil {
								t.Fatalf("typed %+v at %d %v", v, r.Position(), e)
							}
							p := v.PAGEMODEGroupFallback.Known
							if p.PAGEMODE != 2 || p.GlobalTFIGroupFallback.Known == nil {
								t.Fatalf("page/address %+v", p)
							}
							a := p.GlobalTFIGroupFallback.Known
							ac := a.GlobalTFIChoice
							if address == 0 {
								if ac.GlobalTFI == nil || ac.GlobalTFI.GlobalTFI.UPLINKTFIChoice.DOWNLINKTFI == nil || ac.GlobalTFI.GlobalTFI.UPLINKTFIChoice.DOWNLINKTFI.DOWNLINKTFI != 17 {
									t.Fatalf("Global TFI %+v", ac)
								}
							} else {
								if ac.TLLIGRNTI == nil || ac.TLLIGRNTI.TLLIGRNTI != 0x10203040 {
									t.Fatal("TLLI")
								}
							}
							b := a.LengthIndicatorOfMSIDGroupFallback.Known
							if b == nil {
								t.Fatalf("body %+v", a)
							}
							if (b.LengthIndicatorOfMSIDGroup != nil) != (identity > 0) {
								t.Fatal("MS_ID presence")
							}
							if identity > 0 {
								id := b.LengthIndicatorOfMSIDGroup
								if id.LengthIndicatorOfMSID != uint8(identity-1) || id.MSID.BitLength != identity || (id.ALPHAGroup != nil) != (identity >= 3) {
									t.Fatalf("MS_ID %+v", id)
								}
								if id.ALPHAGroup != nil {
									if id.ALPHAGroup.ALPHA != 9 || (id.ALPHAGroup.GAMMA != nil) != (identity == 4) {
										t.Fatal("power control")
									}
									if identity == 4 && *id.ALPHAGroup.GAMMA != 21 {
										t.Fatal("gamma")
									}
								}
							}
							if (b.TMGI != nil) != (flags&1 != 0) || (b.MBMSSessionIdentity != nil) != (flags&2 != 0) {
								t.Fatal("session/TMGI presence")
							}
							if b.TMGI != nil && (b.TMGI.MBMSSERVICEIDChoice.MBMSSERVICEID == nil || b.TMGI.MBMSSERVICEIDChoice.MBMSSERVICEID.MBMSSERVICEID != 0xabcdef) {
								t.Fatal("TMGI")
							}
							if b.MBMSSessionIdentity != nil && *b.MBMSSessionIdentity != 0x9a {
								t.Fatal("session")
							}
							switch arm {
							case 0:
								q := b.RejectCauseChoice.RejectCause
								if q == nil || q.RejectCause != uint8(mask&3) || (q.EstimatedSessionDuration != nil) != (mask&4 != 0) {
									t.Fatalf("reject %+v", b.RejectCauseChoice)
								}
								if q.EstimatedSessionDuration != nil && *q.EstimatedSessionDuration != 0xa5 {
									t.Fatal("reject duration")
								}
							case 1:
								q := b.RejectCauseChoice.MBMSBearerDescription
								if q == nil || q.EstimatedSessionDuration != 0xc3 || q.MBMSInBandSignallingIndicator.MBMSInBandSignallingIndicator != 1 {
									t.Fatalf("assigned %+v", b.RejectCauseChoice)
								}
								h := q.MBMSBearerDescription
								if h.DOWNLINKTIMESLOTALLOCATION != 0xa5 || h.LengthOfMBMSBearerIdentity != uint8(length) || h.MBMSBearerIdentity.BitLength != length {
									t.Fatal("bearer")
								}
								if (h.MBMSRadioBearerStartingTime != nil) != (mask&1 != 0) || (h.FrequencyParameters != nil) != (mask&2 != 0) || (h.EGPRSWindowSize != nil) != (mask&4 != 0) || (h.TIMESLOTALLOCATIONUPLINKFEEDBACKCHANNEL != nil) != (mask&8 != 0) || (h.NPMTransferTime != nil) != (mask&16 != 0) {
									t.Fatalf("bearer options %+v", h)
								}
								if h.MBMSRadioBearerStartingTime != nil && *h.MBMSRadioBearerStartingTime != 0x9abc {
									t.Fatal("starting time")
								}
								if h.FrequencyParameters != nil && (h.FrequencyParameters.TSC != 5 || h.FrequencyParameters.ARFCNChoice.ARFCN == nil || h.FrequencyParameters.ARFCNChoice.ARFCN.ARFCN != 0x2aa) {
									t.Fatal("frequency")
								}
								if h.EGPRSWindowSize != nil && h.EGPRSWindowSize.EGPRSWindowSize != 17 {
									t.Fatal("window")
								}
								if h.TIMESLOTALLOCATIONUPLINKFEEDBACKCHANNEL != nil && *h.TIMESLOTALLOCATIONUPLINKFEEDBACKCHANNEL != 5 {
									t.Fatal("feedback")
								}
								if h.NPMTransferTime != nil && *h.NPMTransferTime != 23 {
									t.Fatal("NPM")
								}
							case 2:
								q := b.RejectCauseChoice.MBMSRadioBearerStartingTime
								if q == nil || q.Known == nil || q.Ignored != nil {
									t.Fatalf("reassigned %+v", b.RejectCauseChoice)
								}
								h := q.Known
								if (h.MBMSRadioBearerStartingTime != nil) != (mask&1 != 0) || h.LengthOfMBMSBearerIdentity != uint8(length) || h.MBMSBearerIdentity.BitLength != length {
									t.Fatal("reassigned bearer")
								}
								if h.MBMSRadioBearerStartingTime != nil && *h.MBMSRadioBearerStartingTime != 0x9abc {
									t.Fatal("reassignment time")
								}
							}
							if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
								t.Fatalf("unexpected receiver region %+v", report)
							}
							wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
							if e != nil {
								t.Fatal(e)
							}
							if got := assignmentEncode(t, raw, wire, func(w *runtime.Writer) error {
								return mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentTo(w, v)
							}); !bytes.Equal(got, raw) {
								t.Fatal("replay")
							}
							cv := runtime.Canonical(v)
							cw := runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, 184).Wire
							if got := assignmentEncode(t, raw, cw, func(w *runtime.Writer) error {
								return mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentTo(w, cv)
							}); !bytes.Equal(got, raw) {
								t.Fatalf("canonical %x want %x", got, raw)
							}
							cv.PAGEMODEGroupFallback.Known.PAGEMODE ^= 1
							edit := assignmentEncode(t, raw, cw, func(w *runtime.Writer) error {
								return mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentTo(w, cv)
							})
							er, _ := assignmentReader(t, edit, 184)
							ev, err := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(er)
							if err != nil || ev.PAGEMODEGroupFallback.Known == nil || ev.PAGEMODEGroupFallback.Known.PAGEMODE != 3 {
								t.Fatalf("edit %+v %v", ev, err)
							}
							vectors++
						}
					}
				}
			}
		}
	}
	if vectors == 0 {
		t.Fatal("no vectors")
	}
	t.Logf("complete blocks=%d", vectors)
}
func FuzzClause11239Assignment(f *testing.F) {
	for arm := 0; arm < 4; arm++ {
		bits, _ := assignmentBits(0, 4, 3, arm, 31, 5)
		if len(bits) == 184 {
			f.Add(assignmentBytes(bits))
		}
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) != 23 {
			return
		}
		raw = append([]byte(nil), raw...)
		raw[0] = 0x40
		raw[1] = (raw[1] & 3) | 80
		r, _ := runtime.NewChoiceReaderBits(raw, 184, nil)
		if _, e := r.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		v, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
		if e != nil {
			return
		}
		wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
		if e != nil {
			t.Fatal(e)
		}
		if got := assignmentEncode(t, raw, wire, func(w *runtime.Writer) error {
			return mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentTo(w, v)
		}); !bytes.Equal(got, raw) {
			t.Fatal("replay")
		}
	})
}

func TestClause11239EscapeAndAddressRecovery(t *testing.T) {
	bits, _ := assignmentBits(1, 0, 0, 3, 0, 1)
	raw := assignmentBytes(bits)
	r, c := assignmentReader(t, raw, 184)
	v, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
	if e != nil || v.PAGEMODEGroupFallback.Known == nil {
		t.Fatalf("escape prefix %+v %v", v, e)
	}
	p := v.PAGEMODEGroupFallback.Known
	if p.PAGEMODE != 2 || p.GlobalTFIGroupFallback.Known == nil {
		t.Fatal("address prefix lost")
	}
	a := p.GlobalTFIGroupFallback.Known
	if a.GlobalTFIChoice.TLLIGRNTI == nil || a.GlobalTFIChoice.TLLIGRNTI.TLLIGRNTI != 0x10203040 {
		t.Fatal("TLLI prefix lost")
	}
	if report := c.TakeReceiverReport(); len(report.Regions) == 0 {
		t.Fatal("message escape not reported")
	}
	wire, e := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: raw, BitLength: 184}, 184, runtime.BitString{}, r.Wire())
	if e != nil {
		t.Fatal(e)
	}
	if got := assignmentEncode(t, raw, wire, func(w *runtime.Writer) error {
		return mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentTo(w, v)
	}); !bytes.Equal(got, raw) {
		t.Fatal("escape replay")
	}
	if _, e = mbmsassignment.EncodeMBMSAssignmentNonDistributionMessageContentCanonicalAtLength(v, 23); e == nil {
		t.Fatal("received escape accepted by canonical sender")
	}
	// Truncation inside the 32-bit address selects the printed address error arm.
	for end := 17; end < 50; end++ {
		tr, _ := runtime.NewChoiceReaderBits(raw, end, nil)
		if _, e = tr.ReadUint(14); e != nil {
			t.Fatal(e)
		}
		tv, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(tr)
		if e != nil || tv.PAGEMODEGroupFallback.Known == nil || tv.PAGEMODEGroupFallback.Known.PAGEMODE != 2 || tv.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Ignored == nil || tr.Position() != end {
			t.Fatalf("address cut %d: %+v at %d: %v", end, tv, tr.Position(), e)
		}
	}
}

func TestClause11239ContainingSentinel(t *testing.T) {
	bits, _ := assignmentBits(1, 4, 3, 1, 31, 5)
	extended := assignmentBytes(bits + "101")
	r, _ := runtime.NewChoiceReaderBits(extended, 187, nil)
	if _, e := r.ReadUint(14); e != nil {
		t.Fatal(e)
	}
	old, e := r.PushLimit(170)
	if e != nil {
		t.Fatal(e)
	}
	v, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
	if e != nil || v.PAGEMODEGroupFallback.Known == nil {
		t.Fatalf("content %+v %v", v, e)
	}
	if e = r.PopLimit(old); e != nil {
		t.Fatal(e)
	}
	sentinel, e := r.ReadUint(3)
	if e != nil || sentinel != 5 {
		t.Fatalf("sentinel %d: %v", sentinel, e)
	}
}

// TS 44.060 V19.0.0 §11.2.39 p20393: length codes 0,6,7 are reserved.
func TestClause11239ReservedBearerLength(t *testing.T) {
	for _, length := range []int{0, 6, 7} {
		for _, arm := range []int{1, 2} {
			bits, _ := assignmentBits(0, 0, 0, arm, 0, length)
			raw := assignmentBytes(bits)
			r, _ := assignmentReader(t, raw, 184)
			v, e := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
			if e != nil || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Known == nil {
				t.Fatalf("known address prefix %+v %v", v, e)
			}
			body := v.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Known.LengthIndicatorOfMSIDGroupFallback
			if body.Known != nil || body.Ignored == nil {
				t.Fatalf("reserved bearer length %d remains a known body", length)
			}
		}
	}
}

// TS 44.060 V19.0.0 §11.2.39 p20398 delegates NPM Transfer Time
// to §12.45a table 12.45a.1, whose code 31 is reserved.
func TestClause11239ReservedNPMTransferTime(t *testing.T) {
	for _, code := range []int{30, 31} {
		bits, end := assignmentBits(0, 0, 0, 1, 16, 1)
		bits = bits[:end-14] + fmt.Sprintf("%05b", code) + bits[end-9:]
		r, _ := assignmentReader(t, assignmentBytes(bits), 184)
		v, err := mbmsassignment.DecodeMBMSAssignmentNonDistributionMessageContentFrom(r)
		if err != nil || v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Known == nil {
			t.Fatalf("known address prefix for NPM code %d: %v", code, err)
		}
		body := v.PAGEMODEGroupFallback.Known.GlobalTFIGroupFallback.Known.LengthIndicatorOfMSIDGroupFallback
		if code == 31 {
			if body.Known != nil || body.Ignored == nil {
				t.Fatal("reserved NPM code 31 remains a known message body")
			}
		} else if body.Known == nil || body.Known.RejectCauseChoice.MBMSBearerDescription == nil || body.Known.RejectCauseChoice.MBMSBearerDescription.MBMSBearerDescription.NPMTransferTime == nil || *body.Known.RejectCauseChoice.MBMSBearerDescription.MBMSBearerDescription.NPMTransferTime != 30 {
			t.Fatal("valid NPM code 30 lost its typed value")
		}
	}
	code := uint8(31)
	v := mbmsassignment.MBMSBearerDescriptionStruct{DOWNLINKTIMESLOTALLOCATION: 0xa5, LengthOfMBMSBearerIdentity: 1, MBMSBearerIdentity: runtime.BitString{Bytes: []byte{0x80}, BitLength: 1}, NPMTransferTime: &code}
	if _, err := mbmsassignment.EncodeMBMSBearerDescriptionStruct(v); err == nil || !strings.Contains(err.Error(), "reserved by source table") {
		t.Fatalf("reserved NPM plain sender: %v", err)
	}
	if _, err := mbmsassignment.EncodeMBMSBearerDescriptionStructCanonicalAtLength(v, 3); err == nil || !strings.Contains(err.Error(), "reserved by source table") {
		t.Fatalf("reserved NPM canonical sender: %v", err)
	}
}
