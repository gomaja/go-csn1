package controlmessages_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/controlmessages"
)

// TS 44.060 V19.0.0 §11.2.1 paragraphs 13464–13491, §§12.10–12.11:
// synthetic identities use all printed selectors, with source widths 32,
// 11+16 and 1+5 bits. The optional wait contributes 1+8+1 bits.
func accessRejectIdentity(kind int, wait int) string {
	var bits string
	switch kind {
	case 0:
		bits = fmt.Sprintf("0%032b", uint32(0x10203040))
	case 1:
		bits = fmt.Sprintf("10%011b%016b", 0x5a5, 0x9abc)
	case 2:
		bits = fmt.Sprintf("110%05b", 19)
	case 3:
		bits = fmt.Sprintf("111%05b", 27)
	default:
		panic("invalid synthetic identity")
	}
	if wait < 0 {
		return bits + "0"
	}
	return bits + fmt.Sprintf("1%08b%01b", wait, wait&1)
}

func accessRejectBits(t testing.TB, bits string) []byte {
	t.Helper()
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b == '1' {
			out[i/8] |= 1 << uint(7-i%8)
		} else if b != '0' {
			t.Fatal("invalid synthetic bit")
		}
	}
	return out
}

// TS 44.060 V19.0.0 §11: padding starts with literal 0 and then L symbols;
// the 00101011 phase is anchored at the caller's absolute bit position.
func accessRejectPadding(t testing.TB, prefix string, extent int) []byte {
	t.Helper()
	if len(prefix) > extent {
		t.Fatal("synthetic prefix exceeds extent")
	}
	if len(prefix) < extent {
		prefix += "0"
	}
	for len(prefix) < extent {
		prefix += fmt.Sprint((byte(0x2b) >> uint(7-len(prefix)%8)) & 1)
	}
	return accessRejectBits(t, prefix)
}

func accessRejectReader(t testing.TB, data []byte, extent, offset int) (*runtime.Reader, *runtime.Collector) {
	t.Helper()
	c := runtime.NewReceiverCollector()
	r, err := runtime.NewChoiceReaderBits(data, extent, c)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < offset; i++ {
		if _, err = r.ReadUint(1); err != nil {
			t.Fatal(err)
		}
	}
	return r, c
}

func accessRejectEncode(t testing.TB, v controlmessages.PacketAccessRejectMessageContent, data []byte, extent, offset int, r *runtime.Reader, canonical bool) runtime.BitString {
	t.Helper()
	w := runtime.NewWriter()
	if canonical {
		w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, extent).Wire)
		var err error
		if _, err = w.PushLimit(extent); err != nil {
			t.Fatal(err)
		}
		v = runtime.Canonical(v)
	} else {
		wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: extent}, extent, runtime.BitString{}, r.Wire())
		if err != nil {
			t.Fatal(err)
		}
		w.WithWire(wire)
	}
	for i := 0; i < offset; i++ {
		if err := w.WriteUint(uint64((data[i/8]>>uint(7-i%8))&1), 1); err != nil {
			t.Fatal(err)
		}
	}
	if err := controlmessages.EncodePacketAccessRejectMessageContentTo(w, v); err != nil {
		t.Fatal(err)
	}
	out, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func accessRejectEqual(t testing.TB, got runtime.BitString, want []byte, extent int) {
	t.Helper()
	if got.BitLength != extent || !bytes.Equal(got.Bytes, want) {
		t.Fatalf("wire %x/%d, want %x/%d", got.Bytes, got.BitLength, want, extent)
	}
}

func accessRejectCheck(t testing.TB, reject controlmessages.RejectStruct, kind, wait int) {
	t.Helper()
	k := reject.TLLIGRNTIGroupFallback.Known
	if k == nil || reject.TLLIGRNTIGroupFallback.Ignored != nil {
		t.Fatalf("identity lost: %+v", reject)
	}
	switch kind {
	case 0:
		if k.TLLIGRNTIChoice.TLLIGRNTI == nil || k.TLLIGRNTIChoice.TLLIGRNTI.TLLIGRNTI != 0x10203040 {
			t.Fatalf("TLLI %+v", k.TLLIGRNTIChoice)
		}
	case 1:
		arm := k.TLLIGRNTIChoice.PacketRequestReference
		if arm == nil || arm.PacketRequestReferenceChoice.PacketRequestReference == nil {
			t.Fatal("request reference arm missing")
		}
		pr := arm.PacketRequestReferenceChoice.PacketRequestReference.PacketRequestReference
		if pr.RANDOMACCESSINFORMATIONValue != 0x5a5 || pr.FRAMENUMBER != 0x9abc {
			t.Fatalf("request reference %+v", pr)
		}
	case 2, 3:
		arm := k.TLLIGRNTIChoice.PacketRequestReference
		if arm == nil || arm.PacketRequestReferenceChoice.GlobalTFI == nil {
			t.Fatal("global TFI arm missing")
		}
		tf := arm.PacketRequestReferenceChoice.GlobalTFI.GlobalTFI.UPLINKTFIChoice
		if kind == 2 {
			if tf.UPLINKTFI == nil || tf.UPLINKTFI.UPLINKTFI != 19 {
				t.Fatalf("uplink TFI %+v", tf)
			}
		} else if tf.DOWNLINKTFI == nil || tf.DOWNLINKTFI.DOWNLINKTFI != 27 {
			t.Fatalf("downlink TFI %+v", tf)
		}
	}
	if wait < 0 {
		if k.WAITINDICATIONGroup != nil {
			t.Fatal("unexpected wait")
		}
	} else if k.WAITINDICATIONGroup == nil || k.WAITINDICATIONGroup.WAITINDICATION != uint8(wait) || k.WAITINDICATIONGroup.WAITINDICATIONSIZE != uint8(wait&1) {
		t.Fatalf("wait %+v", k.WAITINDICATIONGroup)
	}
}

// Both the first Reject and additional Reject position exercise every identity
// alternative and each optional-wait policy, against calculated wire bytes.
func TestClause1121IdentityWaitAndAdditional(t *testing.T) {
	for kind := 0; kind < 4; kind++ {
		for _, wait := range []int{-1, 0, 0xa5} {
			for _, additional := range []bool{false, true} {
				t.Run(fmt.Sprintf("kind%d/wait%d/additional%t", kind, wait, additional), func(t *testing.T) {
					prefix := "0100000010000110"
					if additional {
						prefix += accessRejectIdentity(0, -1) + "1"
					}
					prefix += accessRejectIdentity(kind, wait) + "01010"
					data := accessRejectPadding(t, prefix, 184)
					t.Logf("input=%x", data)
					r, c := accessRejectReader(t, data, 184, 14)
					v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
					if err != nil {
						t.Fatal(err)
					}
					k := v.PAGEMODEGroupFallback.Known
					if k == nil || k.PAGEMODE != 2 || r.Position() != 184 {
						t.Fatalf("message %+v at %d", v, r.Position())
					}
					reject := k.Reject
					if additional {
						accessRejectCheck(t, k.Reject, 0, -1)
						if len(k.Group.AdditionalRejectList) != 1 {
							t.Fatalf("additional count %d", len(k.Group.AdditionalRejectList))
						}
						reject = k.Group.AdditionalRejectList[0].AdditionalReject
					} else if len(k.Group.AdditionalRejectList) != 0 {
						t.Fatal("unexpected additional Reject")
					}
					accessRejectCheck(t, reject, kind, wait)
					if report := c.TakeReceiverReport(); len(report.Regions) != 0 || len(report.ZeroFill) != 0 {
						t.Fatalf("unexpected report %+v", report)
					}
					accessRejectEqual(t, accessRejectEncode(t, v, data, 184, 14, r, false), data, 184)
					accessRejectEqual(t, accessRejectEncode(t, v, data, 184, 14, r, true), data, 184)
					edited := runtime.Canonical(v)
					edited.PAGEMODEGroupFallback.Known.PAGEMODE = 3
					editReject := &edited.PAGEMODEGroupFallback.Known.Reject
					if additional {
						editReject = &edited.PAGEMODEGroupFallback.Known.Group.AdditionalRejectList[0].AdditionalReject
					}
					editKnown := editReject.TLLIGRNTIGroupFallback.Known
					var editBits string
					switch kind {
					case 0:
						editKnown.TLLIGRNTIChoice.TLLIGRNTI.TLLIGRNTI ^= 1
						editBits = fmt.Sprintf("0%032b", uint32(0x10203041))
					case 1:
						editKnown.TLLIGRNTIChoice.PacketRequestReference.PacketRequestReferenceChoice.PacketRequestReference.PacketRequestReference.FRAMENUMBER ^= 1
						editBits = fmt.Sprintf("10%011b%016b", 0x5a5, 0x9abd)
					case 2:
						editKnown.TLLIGRNTIChoice.PacketRequestReference.PacketRequestReferenceChoice.GlobalTFI.GlobalTFI.UPLINKTFIChoice.UPLINKTFI.UPLINKTFI ^= 1
						editBits = fmt.Sprintf("110%05b", 18)
					case 3:
						editKnown.TLLIGRNTIChoice.PacketRequestReference.PacketRequestReferenceChoice.GlobalTFI.GlobalTFI.UPLINKTFIChoice.DOWNLINKTFI.DOWNLINKTFI ^= 1
						editBits = fmt.Sprintf("111%05b", 26)
					}
					if wait < 0 {
						editBits += "0"
					} else {
						editKnown.WAITINDICATIONGroup.WAITINDICATION ^= 1
						editKnown.WAITINDICATIONGroup.WAITINDICATIONSIZE ^= 1
						editBits += fmt.Sprintf("1%08b%01b", wait^1, (wait^1)&1)
					}
					editPrefix := "0100000010000111"
					if additional {
						editPrefix += accessRejectIdentity(0, -1) + "1"
					}
					want := accessRejectPadding(t, editPrefix+editBits+"01010", 184)
					accessRejectEqual(t, accessRejectEncode(t, edited, data, 184, 14, r, true), want, 184)
				})
			}
		}
	}
}

// TS 44.060 V19.0.0 §11.2.1 paragraphs 13484–13491: a leading 0
// selects the first printed all-requests policy; nonempty RB/PFI lists use
// repeated 1-tagged entries followed by 0, under TS 24.007 V20.0.0 Annex B §B.1.1.4.
func TestClause1121ReleaseLists(t *testing.T) {
	cases := []struct {
		name, suffix string
		agb          bool
		selected     bool
	}{
		{"release5-empty", "100", false, false},
		{"release5-all-and-selected", "1" + "1" + "0000" + "0" + "1" + "1111" + "1000001111110" + "0" + "0", false, true},
		{"release6-empty", "1010", true, false},
		{"release6-all-and-selected", "1" + "1" + "1001" + "1000110" + "0" + "1" + "1" + "0" + "1" + "10000000111111110" + "0", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prefix := "0100000010000110" + accessRejectIdentity(0, -1) + "0" + tc.suffix
			data := accessRejectPadding(t, prefix, 184)
			t.Logf("input=%x", data)
			r, c := accessRejectReader(t, data, 184, 14)
			v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if v.PAGEMODEGroupFallback.Known == nil {
				t.Fatalf("message lost %+v", v)
			}
			release := v.PAGEMODEGroupFallback.Known.Group.IuModeRejectChoice.IuModeReject
			if release == nil {
				t.Fatal("release5 arm missing")
			}
			switch tc.name {
			case "release5-empty", "release6-empty":
				if len(release.IuModeRejectList) != 0 {
					t.Fatal("unexpected Iu entry")
				}
			case "release5-all-and-selected":
				if len(release.IuModeRejectList) != 2 {
					t.Fatalf("Iu count %d", len(release.IuModeRejectList))
				}
				all := release.IuModeRejectList[0].IuModeReject
				selected := release.IuModeRejectList[1].IuModeReject
				if all.GRNTIExtension != 0 || all.RBIdChoice.Alt0 == nil || all.RBIdChoice.RBId != nil {
					t.Fatalf("Iu all policy %+v", all)
				}
				if selected.GRNTIExtension != 15 || selected.RBIdChoice.RBId == nil || len(*selected.RBIdChoice.RBId) != 2 || (*selected.RBIdChoice.RBId)[0].RBId != 0 || (*selected.RBIdChoice.RBId)[1].RBId != 31 {
					t.Fatalf("Iu selected policy %+v", selected)
				}
			case "release6-all-and-selected":
				if len(release.IuModeRejectList) != 1 {
					t.Fatal("missing Iu entry")
				}
				selected := release.IuModeRejectList[0].IuModeReject
				if selected.GRNTIExtension != 9 || selected.RBIdChoice.RBId == nil || len(*selected.RBIdChoice.RBId) != 1 || (*selected.RBIdChoice.RBId)[0].RBId != 3 {
					t.Fatalf("Iu RB %+v", selected)
				}
			}
			if tc.agb {
				agb := release.AGbModeRejectChoice.AGbModeReject
				if agb == nil {
					t.Fatal("release6 arm missing")
				}
				if !tc.selected {
					if len(agb.AGbModeRejectList) != 0 {
						t.Fatal("unexpected A/Gb entry")
					}
				} else {
					if len(agb.AGbModeRejectList) != 2 {
						t.Fatalf("A/Gb count %d", len(agb.AGbModeRejectList))
					}
					all := agb.AGbModeRejectList[0].AGbModeReject.PFIChoice
					selected := agb.AGbModeRejectList[1].AGbModeReject.PFIChoice
					if all.Alt0 == nil || all.PFI != nil {
						t.Fatalf("A/Gb all policy %+v", all)
					}
					if selected.PFI == nil || len(*selected.PFI) != 2 || (*selected.PFI)[0].PFI != 0 || (*selected.PFI)[1].PFI != 127 {
						t.Fatalf("A/Gb selected policy %+v", selected)
					}
				}
			} else if release.AGbModeRejectChoice.NoString == nil {
				t.Fatal("legacy A/Gb extension arm missing")
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("unexpected report %+v", report)
			}
			accessRejectEqual(t, accessRejectEncode(t, v, data, 184, 14, r, false), data, 184)
			if tc.agb {
				accessRejectEqual(t, accessRejectEncode(t, v, data, 184, 14, r, true), data, 184)
			} else {
				// A fresh earlier-release sender drops its ignored tail and ends
				// at the exact source-valid prefix, with no padding arm selected.
				want := accessRejectBits(t, prefix)
				accessRejectEqual(t, accessRejectEncode(t, v, want, len(prefix), 14, r, true), want, len(prefix))
			}
		})
	}
}

// TS 44.060 V19.0.0 §11.2.1 paragraph 13483 and §§11.1.2–11.1.3:
// the inner indefinite Ignore owns only the failed Reject. It retains the
// PAGE_MODE and all preceding Rejects and stops the enclosing list/sequence.
func TestClause1121NestedIgnoreKeepsPrefixAndBound(t *testing.T) {
	for _, preceding := range []int{0, 1, 3, 8, 129} {
		for _, bad := range []string{"0", "10", "1101", "11111111", "0" + strings.Repeat("1", 32) + "111"} {
			t.Run(fmt.Sprintf("preceding%d/bad%s", preceding, bad), func(t *testing.T) {
				prefix := strings.Repeat("0", 14) + "10"
				if preceding > 0 {
					prefix += accessRejectIdentity(0, -1)
					for i := 1; i < preceding; i++ {
						prefix += "1" + accessRejectIdentity(i%4, -1)
					}
					prefix += "1"
				}
				start := len(prefix)
				bits := prefix + bad
				data := accessRejectBits(t, bits)
				// A distinct octet follows the explicit caller extent in the same backing input.
				bounded := accessRejectBits(t, bits+"11010101")
				r, c := accessRejectReader(t, bounded, len(bits)+8, 14)
				old, err := r.PushLimit(len(bits) - 14)
				if err != nil {
					t.Fatal(err)
				}
				v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
				if err != nil {
					t.Fatal(err)
				}
				k := v.PAGEMODEGroupFallback.Known
				if k == nil || k.PAGEMODE != 2 || r.Position() != len(bits) {
					t.Fatalf("prefix lost %+v at %d", v, r.Position())
				}
				var region *runtime.ErrorRegion
				if preceding == 0 {
					region = k.Reject.TLLIGRNTIGroupFallback.Ignored
				} else {
					accessRejectCheck(t, k.Reject, 0, -1)
					if len(k.Group.AdditionalRejectList) != preceding {
						t.Fatalf("additional count %d, want %d", len(k.Group.AdditionalRejectList), preceding)
					}
					for i := 0; i < preceding-1; i++ {
						accessRejectCheck(t, k.Group.AdditionalRejectList[i].AdditionalReject, (i+1)%4, -1)
					}
					region = k.Group.AdditionalRejectList[preceding-1].AdditionalReject.TLLIGRNTIGroupFallback.Ignored
				}
				if region == nil || region.Label != runtime.Ignore || region.Location.BitOffset != start || region.Location.Width != len(bad) || region.Bits.BitLength != len(bad) || !bytes.Equal(region.Bits.Bytes, accessRejectBits(t, bad)) {
					t.Fatalf("Ignore region %+v, want offset %d width %d", region, start, len(bad))
				}
				report := c.TakeReceiverReport()
				if len(report.Regions) != 1 || report.Regions[0].Label != runtime.Ignore || report.Regions[0].Location.BitOffset != start || report.Regions[0].Location.Width != len(bad) {
					t.Fatalf("receiver report %+v", report)
				}
				if err = r.PopLimit(old); err != nil {
					t.Fatal(err)
				}
				sentinel, err := r.ReadUint(8)
				if err != nil || sentinel != 0xd5 {
					t.Fatalf("sentinel %x %v", sentinel, err)
				}
				// Replay through a second reader excludes the sentinel and checks every owned bit.
				rr, _ := accessRejectReader(t, data, len(bits), 14)
				replay, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(rr)
				if err != nil {
					t.Fatal(err)
				}
				accessRejectEqual(t, accessRejectEncode(t, replay, data, len(bits), 14, rr, false), data, len(bits))
				fresh := runtime.Canonical(v)
				w := runtime.NewWriter()
				w.WithWire(runtime.CanonicalTo(struct{ Wire runtime.WireInfo }{}, len(bits)).Wire)
				if _, err = w.PushLimit(len(bits)); err != nil {
					t.Fatal(err)
				}
				if err = w.WriteUint(0, 14); err != nil {
					t.Fatal(err)
				}
				if err = controlmessages.EncodePacketAccessRejectMessageContentTo(w, fresh); err == nil || !strings.Contains(err.Error(), "receiver error region") {
					t.Fatalf("canonical Ignore refusal: %v", err)
				}
			})
		}
	}
}

func TestClause1121EarlierReleaseAndNullTails(t *testing.T) {
	base := strings.Repeat("0", 14) + "10" + accessRejectIdentity(0, -1) + "0"
	for _, suffix := range []string{"", "0", "010101101", "10", "10011100101"} {
		t.Run("suffix"+suffix, func(t *testing.T) {
			bits := base + suffix
			data := accessRejectBits(t, bits)
			r, c := accessRejectReader(t, data, len(bits), 14)
			v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			k := v.PAGEMODEGroupFallback.Known
			if k == nil || k.PAGEMODE != 2 || r.Position() != len(bits) {
				t.Fatalf("tail lost typed prefix %+v", v)
			}
			accessRejectCheck(t, k.Reject, 0, -1)
			switch {
			case suffix == "":
				// TS 44.060 V19.0.0 §§11.1.4.4, 11.2.1: the enclosing
				// truncated group ends before interpreting the release choice.
				points := r.Wire().TruncatedAt["PacketAccessRejectMessageContentPAGEMODEGroupFallbackKnownAdditionalRejectGroup"]
				if !runtime.IsZero(k.Group.IuModeRejectChoice) || len(points) != 1 || points[0] != 1 {
					t.Fatalf("release5 omitted at group boundary: %+v, points=%v", k.Group.IuModeRejectChoice, points)
				}
			case suffix[0] == '0':
				if k.Group.IuModeRejectChoice.NoString == nil {
					t.Fatal("release5 earlier-release arm missing")
				}
			default:
				release := k.Group.IuModeRejectChoice.IuModeReject
				if release == nil || len(release.IuModeRejectList) != 0 {
					t.Fatal("release5 empty list missing")
				}
				if suffix == "10" {
					if release.AGbModeRejectChoice.AltUnlabeled == nil {
						t.Fatal("release6 null arm missing")
					}
				} else if release.AGbModeRejectChoice.NoString == nil {
					t.Fatal("release6 earlier-release arm missing")
				}
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("ordinary legacy tail became an error region: %+v", report)
			}
			accessRejectEqual(t, accessRejectEncode(t, v, data, len(bits), 14, r, false), data, len(bits))
			sender := base
			switch {
			case suffix == "":
			case suffix[0] == '0':
				sender += "0"
			case suffix == "10":
				sender += "10"
			default:
				sender += "100"
			}
			want := accessRejectBits(t, sender)
			accessRejectEqual(t, accessRejectEncode(t, v, want, len(sender), 14, r, true), want, len(sender))
		})
	}
}

func TestClause1121PaddingPhasesAndGrowingInput(t *testing.T) {
	for _, phase := range []int{6, 14, 22} {
		for _, extent := range []int{64, 65, 96, 184, 1024} {
			prefix := strings.Repeat("0", phase) + "10" + accessRejectIdentity(2, -1) + "01010"
			data := accessRejectPadding(t, prefix, extent)
			r, c := accessRejectReader(t, data, extent, phase)
			v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if v.PAGEMODEGroupFallback.Known == nil || v.PAGEMODEGroupFallback.Known.PAGEMODE != 2 || r.Position() != extent {
				t.Fatalf("phase %d extent %d prefix %+v", phase, extent, v)
			}
			if report := c.TakeReceiverReport(); len(report.Regions) != 0 {
				t.Fatalf("phase %d report %+v", phase, report)
			}
			accessRejectEqual(t, accessRejectEncode(t, v, data, extent, phase, r, false), data, extent)
			accessRejectEqual(t, accessRejectEncode(t, v, data, extent, phase, r, true), data, extent)
		}
	}
}

// Every finite truncated prefix remains within the authenticated caller bound.
// Error-region fallback may accept malformed content, but must report ownership.
func TestClause1121EveryTruncatedPrefix(t *testing.T) {
	bits := strings.Repeat("0", 14) + "10" + accessRejectIdentity(1, 0xa5) + "1" + accessRejectIdentity(3, 14) + "01010"
	data := accessRejectPadding(t, bits, 184)
	for width := 14; width < 184; width++ {
		r, c := accessRejectReader(t, data, width, 14)
		_, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
		if r.Position() > width {
			t.Fatalf("width %d crossed bound at %d", width, r.Position())
		}
		if err == nil && r.Position() != width {
			t.Fatalf("width %d left unexplained bits at %d", width, r.Position())
		}
		report := c.TakeReceiverReport()
		for _, region := range report.Regions {
			if region.Location.BitOffset < 0 || region.Location.BitOffset+region.Location.Width > width {
				t.Fatalf("width %d report beyond bound %+v", width, report)
			}
		}
	}
}

func FuzzClause1121PacketAccessReject(f *testing.F) {
	for _, data := range [][]byte{{}, {0}, {0x80}, {0xff}, {0x80, 0, 0, 0, 0}, {0xb5, 0x9a, 0xbc, 0xff}} {
		f.Add(data, uint16(len(data)*8))
	}
	for kind := 0; kind < 4; kind++ {
		data := accessRejectPadding(f, "10"+accessRejectIdentity(kind, 0xa5)+"01010", 184)
		f.Add(data, uint16(184))
	}
	for _, count := range []int{1, 8, 129} {
		var b strings.Builder
		b.WriteString("10")
		b.WriteString(accessRejectIdentity(0, -1))
		for i := 1; i < count; i++ {
			b.WriteString("1")
			b.WriteString(accessRejectIdentity(0, -1))
		}
		b.WriteString("1110")
		bits := b.String()
		f.Add(accessRejectBits(f, bits), uint16(len(bits)))
	}
	f.Fuzz(func(t *testing.T, data []byte, n uint16) {
		if len(data) > 4096 {
			return
		}
		extent := min(len(data)*8, int(n))
		data, extent = positionContentFixture(t, data, extent)
		r, err := runtime.NewChoiceReaderBits(data, extent, nil)
		if err == nil {
			_, err = r.ReadUint(14)
		}
		if err != nil {
			t.Fatal(err)
		}
		v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
		if r.Position() > extent {
			t.Fatal("decoder crossed explicit extent")
		}
		if err == nil {
			if r.Position() != extent {
				t.Fatal("successful content decoder left unexplained input")
			}
			raw := append([]byte(nil), data[:(extent+7)/8]...)
			if extent%8 != 0 {
				raw[len(raw)-1] &= 0xff << uint(8-extent%8)
			}
			accessRejectEqual(t, accessRejectEncode(t, v, raw, extent, 14, r, false), raw, extent)
		}
	})
}

func TestClause1121PublicByteEntryPoints(t *testing.T) {
	for kind := 0; kind < 4; kind++ {
		prefix := "10" + accessRejectIdentity(kind, 0xa5) + "01010"
		data := accessRejectContentPadding(t, prefix, 184)
		decoded, err := controlmessages.DecodePacketAccessRejectMessageContent(data)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Value.PAGEMODEGroupFallback.Known == nil {
			t.Fatalf("public decode lost fields %+v", decoded)
		}
		accessRejectCheck(t, decoded.Value.PAGEMODEGroupFallback.Known.Reject, kind, 0xa5)
		replay, err := controlmessages.EncodePacketAccessRejectMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(replay, data) {
			t.Fatalf("public replay %x, want %x: %v", replay, data, err)
		}
		canonical, err := controlmessages.EncodePacketAccessRejectMessageContentCanonicalAtLength(decoded.Value, 23)
		if err != nil || !bytes.Equal(canonical, data) {
			t.Fatalf("public canonical %x, want %x: %v", canonical, data, err)
		}
	}
}

func TestClause1121MalformedExtensionReportsDistributionError(t *testing.T) {
	base := strings.Repeat("0", 14) + "10" + accessRejectIdentity(0, -1) + "0"
	for _, tail := range []string{"11", "11101", "10111", "1011100000"} {
		bits := base + tail
		data := accessRejectBits(t, bits)
		r, c := accessRejectReader(t, data, len(bits), 14)
		v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
		if err != nil {
			t.Fatal(err)
		}
		if v.PAGEMODEGroupFallback.Ignored == nil || v.PAGEMODEGroupFallback.Ignored.Label != runtime.DistributionPartError {
			t.Fatalf("malformed extension did not select printed distribution error: %+v", v)
		}
		report := c.TakeReceiverReport()
		if len(report.Regions) != 1 || report.Regions[0].Label != runtime.DistributionPartError || report.Regions[0].Location.BitOffset != 14 || report.Regions[0].Location.Width != len(bits)-14 {
			t.Fatalf("distribution report %+v", report)
		}
		accessRejectEqual(t, accessRejectEncode(t, v, data, len(bits), 14, r, false), data, len(bits))
	}
}

func TestClause1121PublicByteIgnoreReplay(t *testing.T) {
	cases := []string{
		"10" + "011111",
		"10" + accessRejectIdentity(0, -1) + "1" + "110",
		"10" + accessRejectIdentity(0, -1) + "1" + accessRejectIdentity(1, -1) + "1" + "1101",
	}
	for _, bits := range cases {
		data := accessRejectBits(t, bits)
		if len(bits)%8 != 0 {
			t.Fatal("public byte test is not octet aligned")
		}
		decoded, err := controlmessages.DecodePacketAccessRejectMessageContent(data)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.BitsConsumed != len(bits) || decoded.Tail.BitLength != 0 || decoded.Value.PAGEMODEGroupFallback.Known == nil || decoded.Value.PAGEMODEGroupFallback.Known.PAGEMODE != 2 {
			t.Fatalf("public Ignore decode %+v", decoded)
		}
		replay, err := controlmessages.EncodePacketAccessRejectMessageContent(decoded.Value)
		if err != nil || !bytes.Equal(replay, data) {
			t.Fatalf("public Ignore replay %x, want %x: %v", replay, data, err)
		}
		if _, err = controlmessages.EncodePacketAccessRejectMessageContentCanonicalAtLength(decoded.Value, len(data)); err == nil {
			t.Fatal("public canonical emitted receiver Ignore")
		}
	}
}

func TestClause1121SenderFieldBounds(t *testing.T) {
	for kind := 0; kind < 4; kind++ {
		data := accessRejectPadding(t, "10"+accessRejectIdentity(kind, 0xa5)+"01010", 184)
		decoded, err := controlmessages.DecodePacketAccessRejectMessageContent(data)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"PAGE_MODE", "WAIT_INDICATION_SIZE", "identity"} {
			if field == "identity" && kind == 0 {
				continue
			} // All uint32 TLLI values fit the printed 32-bit field.
			invalid := runtime.Canonical(decoded.Value)
			k := invalid.PAGEMODEGroupFallback.Known
			switch field {
			case "PAGE_MODE":
				k.PAGEMODE = 4
			case "WAIT_INDICATION_SIZE":
				k.Reject.TLLIGRNTIGroupFallback.Known.WAITINDICATIONGroup.WAITINDICATIONSIZE = 2
			case "identity":
				c := k.Reject.TLLIGRNTIGroupFallback.Known.TLLIGRNTIChoice.PacketRequestReference.PacketRequestReferenceChoice
				switch kind {
				case 1:
					c.PacketRequestReference.PacketRequestReference.RANDOMACCESSINFORMATIONValue = 2048
				case 2:
					c.GlobalTFI.GlobalTFI.UPLINKTFIChoice.UPLINKTFI.UPLINKTFI = 32
				case 3:
					c.GlobalTFI.GlobalTFI.UPLINKTFIChoice.DOWNLINKTFI.DOWNLINKTFI = 32
				}
			}
			if _, err = controlmessages.EncodePacketAccessRejectMessageContentCanonicalAtLength(invalid, 23); err == nil {
				t.Fatalf("kind %d %s exceeded source width", kind, field)
			}
		}
	}
	prefix := "10" + accessRejectIdentity(0, -1) + "01" + "1" + "1001" + "1000110" + "0" + "1" + "1" + "10000000111111110" + "0"
	data := accessRejectPadding(t, prefix, 184)
	decoded, err := controlmessages.DecodePacketAccessRejectMessageContent(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"G-RNTI_extension", "RB Id", "PFI"} {
		invalid := runtime.Canonical(decoded.Value)
		release := invalid.PAGEMODEGroupFallback.Known.Group.IuModeRejectChoice.IuModeReject
		if release == nil || len(release.IuModeRejectList) != 1 || release.AGbModeRejectChoice.AGbModeReject == nil {
			t.Fatalf("synthetic list missing %+v", invalid)
		}
		switch field {
		case "G-RNTI_extension":
			release.IuModeRejectList[0].IuModeReject.GRNTIExtension = 16
		case "RB Id":
			(*release.IuModeRejectList[0].IuModeReject.RBIdChoice.RBId)[0].RBId = 32
		case "PFI":
			(*release.AGbModeRejectChoice.AGbModeReject.AGbModeRejectList[0].AGbModeReject.PFIChoice.PFI)[0].PFI = 128
		}
		if _, err = controlmessages.EncodePacketAccessRejectMessageContentCanonicalAtLength(invalid, 23); err == nil {
			t.Fatalf("%s exceeded source width", field)
		}
	}
}

func TestClause1121EditsAfterNestedIgnoreFail(t *testing.T) {
	bits := strings.Repeat("0", 14) + "10" + accessRejectIdentity(0, -1) + "1" + "110"
	data := accessRejectBits(t, bits)
	r, _ := accessRejectReader(t, data, len(bits), 14)
	v, err := controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if v.PAGEMODEGroupFallback.Known == nil {
		t.Fatal("accepted prefix absent")
	}
	accessRejectEqual(t, accessRejectEncode(t, v, data, len(bits), 14, r, false), data, len(bits))
	wire, err := runtime.SealBits(struct{}{}, runtime.BitString{Bytes: data, BitLength: len(bits)}, len(bits), runtime.BitString{}, r.Wire())
	if err != nil {
		t.Fatal(err)
	}
	encode := func() error {
		w := runtime.NewWriter()
		w.WithWire(wire)
		if err := w.WriteUint(0, 14); err != nil {
			return err
		}
		return controlmessages.EncodePacketAccessRejectMessageContentTo(w, v)
	}
	k := v.PAGEMODEGroupFallback.Known
	items := k.Group.AdditionalRejectList
	if len(items) != 1 || items[0].AdditionalReject.TLLIGRNTIGroupFallback.Ignored == nil {
		t.Fatalf("nested Ignore absent %+v", v)
	}
	k.Group.AdditionalRejectList = append(items[:len(items):len(items)], items[0])
	if err = encode(); err == nil || !strings.Contains(err.Error(), "items after receiver error region") {
		t.Fatalf("appended item after Ignore: %v", err)
	}
	k.Group.AdditionalRejectList = items
	k.Group.IuModeRejectChoice.AltUnlabeled = &struct{}{}
	var bound *runtime.BoundError
	if err = encode(); !errors.As(err, &bound) || bound.Kind != runtime.ReceivedTruncation || bound.Field != "IuModeRejectChoice" {
		t.Fatalf("nonzero omitted extension: %v", err)
	}
	k.Group.IuModeRejectChoice.AltUnlabeled = nil
	accessRejectEqual(t, accessRejectEncode(t, v, data, len(bits), 14, r, false), data, len(bits))
}

func accessRejectExpectedTLLI(reject controlmessages.RejectStruct) bool {
	fallback := reject.TLLIGRNTIGroupFallback
	known := fallback.Known
	return fallback.Ignored == nil && known != nil && known.WAITINDICATIONGroup == nil && known.TLLIGRNTIChoice.PacketRequestReference == nil && known.TLLIGRNTIChoice.TLLIGRNTI != nil && known.TLLIGRNTIChoice.TLLIGRNTI.TLLIGRNTI == 0x10203040
}

var (
	accessRejectAllocationValue    controlmessages.PacketAccessRejectMessageContent
	accessRejectAllocationError    error
	accessRejectAllocationPosition int
)

// TS 44.060 V19.0.0 §11.2.1 paragraphs 13467 and 13476–13483:
// this bounded probe has n accepted TLLI/no-wait Rejects and one final short
// Ignore. Each emitted accepted item owns two arm pointers, one index path,
// fixed-depth choice/binding scratch, and at most one slice-growth allocation.
// Budget 24 allocations per accepted item, plus 256 for the reader/root and
// the single final failed trial/owned error region. These are allocation-count
// bounds for these inputs and the running toolchain; they do not bound bytes,
// elapsed time, RSS, or other identity/list policies.
func TestClause1121IgnoredSuffixAllocationGrowth(t *testing.T) {
	t.Cleanup(func() {
		accessRejectAllocationValue = controlmessages.PacketAccessRejectMessageContent{}
		accessRejectAllocationError = nil
		accessRejectAllocationPosition = 0
	})
	const perRejectAllocations = 24
	const fixedAllocations = 256
	for _, count := range []int{1, 8, 129, 1024} {
		t.Run(fmt.Sprintf("known%d", count), func(t *testing.T) {
			var b strings.Builder
			b.WriteString(strings.Repeat("0", 14) + "10")
			b.WriteString(accessRejectIdentity(0, -1))
			for i := 1; i < count; i++ {
				b.WriteString("1")
				b.WriteString(accessRejectIdentity(0, -1))
			}
			b.WriteString("1110") // Additional tag 1 followed by an incomplete Global TFI identity.
			bits := b.String()
			data := accessRejectBits(t, bits)
			failures := 0
			var firstError error
			allocations := testing.AllocsPerRun(3, func() {
				r, err := runtime.NewChoiceReaderBits(data, len(bits), nil)
				if err != nil {
					accessRejectAllocationError = err
					if firstError == nil {
						firstError = err
					}
					failures++
					return
				}
				if _, err = r.ReadUint(14); err != nil {
					t.Fatal(err)
				}
				accessRejectAllocationValue, accessRejectAllocationError = controlmessages.DecodePacketAccessRejectMessageContentFrom(r)
				accessRejectAllocationPosition = r.Position()
				known := accessRejectAllocationValue.PAGEMODEGroupFallback.Known
				if accessRejectAllocationError != nil || accessRejectAllocationPosition != len(bits) || known == nil || known.PAGEMODE != 2 || len(known.Group.AdditionalRejectList) != count {
					if firstError == nil {
						firstError = accessRejectAllocationError
					}
					failures++
				} else {
					if !accessRejectExpectedTLLI(known.Reject) {
						failures++
					}
					for _, item := range known.Group.AdditionalRejectList[:count-1] {
						if !accessRejectExpectedTLLI(item.AdditionalReject) {
							failures++
							break
						}
					}
					ignored := known.Group.AdditionalRejectList[count-1].AdditionalReject.TLLIGRNTIGroupFallback.Ignored
					expectedStart := len(bits) - 3
					if ignored == nil || ignored.Label != runtime.Ignore || ignored.Bits.BitLength != 3 || ignored.Location.BitOffset != expectedStart || ignored.Location.Width != 3 || len(ignored.Bits.Bytes) != 1 || ignored.Bits.Bytes[0] != 0xc0 {
						failures++
					}
				}
			})
			if failures != 0 {
				t.Fatalf("%d allocation samples failed typed/bound checks: %v", failures, firstError)
			}
			if accessRejectAllocationError != nil {
				t.Fatal(accessRejectAllocationError)
			}
			k := accessRejectAllocationValue.PAGEMODEGroupFallback.Known
			if k == nil || k.PAGEMODE != 2 || len(k.Group.AdditionalRejectList) != count || accessRejectAllocationPosition != len(bits) {
				t.Fatalf("allocation probe lost prefix at count %d", count)
			}
			accessRejectCheck(t, k.Reject, 0, -1)
			region := k.Group.AdditionalRejectList[count-1].AdditionalReject.TLLIGRNTIGroupFallback.Ignored
			if region == nil || region.Label != runtime.Ignore || region.Bits.BitLength != 3 {
				t.Fatalf("allocation probe lost final Ignore: %+v", region)
			}
			bound := float64(fixedAllocations + perRejectAllocations*count)
			t.Logf("known=%d bits=%d allocs/run=%.0f bound=%.0f", count, len(bits), allocations, bound)
			if !raceEnabled && allocations > bound {
				t.Fatalf("allocation count %.0f exceeds bounded linear allowance %.0f", allocations, bound)
			}
		})
	}
}

// TS 44.060 sections 10.3.1-10.3.2 and 11.2.0.1-11.2.0.2: ordinary content starts at bit 14.
func positionContentFixture(t testing.TB, data []byte, bits int) ([]byte, int) {
	t.Helper()
	w := runtime.NewWriter()
	if err := w.WriteUint(0, 14); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteBitString(runtime.BitString{Bytes: data, BitLength: bits}, bits); err != nil {
		t.Fatal(err)
	}
	raw, err := w.FinishBits(runtime.BitString{})
	if err != nil {
		t.Fatal(err)
	}
	return raw.Bytes, raw.BitLength
}
func sourceContentFixtureReader(t testing.TB, data []byte, bits int, c *runtime.Collector) (*runtime.Reader, error) {
	t.Helper()
	data, bits = positionContentFixture(t, data, bits)
	r, err := runtime.NewChoiceReaderBits(data, bits, c)
	if err == nil {
		_, err = r.ReadUint(14)
	}
	return r, err
}
func accessRejectContentPadding(t testing.TB, prefix string, extent int) []byte {
	t.Helper()
	if len(prefix) > extent {
		t.Fatal("synthetic prefix exceeds extent")
	}
	if len(prefix) < extent {
		prefix += "0"
	}
	for len(prefix) < extent {
		prefix += fmt.Sprint((byte(0x2b) >> uint(7-(14+len(prefix))%8)) & 1)
	}
	return accessRejectBits(t, prefix)
}
