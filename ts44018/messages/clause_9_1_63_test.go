package messages

import (
	"bytes"
	"encoding/hex"
	"github.com/gomaja/go-csn1/runtime"
	"testing"
)

// TS 44.018 V19.0.0 §9.1.63 pp11878-11895: each IMSI has
// 4*(Number of IMSI Digits+1) bits; Rel-14 uses positional L/H.
var ecPagingRequestVectors = []struct {
	first, second, ext, page, mandatory, content int
	wire                                         string
}{
	{-1, -2, 1, 0, 41, 42, "98123456782b2b2b2b2b2b"},
	{-1, -2, 1, 1, 45, 46, "9b61234567832b2b2b2b2b"},
	{-1, -2, 2, 0, 41, 46, "9812345678672b2b2b2b2b"},
	{-1, -2, 2, 1, 45, 50, "9b61234567866b2b2b2b2b"},
	{-1, -1, 1, 0, 74, 75, "9812345678848d159e2b2b"},
	{-1, -1, 1, 1, 78, 79, "9b612345678848d159e32b"},
	{-1, -1, 2, 0, 74, 79, "9812345678848d159e132b"},
	{-1, -1, 2, 1, 78, 83, "9b612345678848d159e12b"},
	{-1, 0, 1, 0, 50, 51, "9812345678c0eb2b2b2b2b"},
	{-1, 0, 1, 1, 54, 55, "9b612345678c0f2b2b2b2b"},
	{-1, 0, 2, 0, 50, 55, "9812345678c0d32b2b2b2b"},
	{-1, 0, 2, 1, 54, 59, "9b612345678c0d2b2b2b2b"},
	{0, -2, 1, 0, 17, 18, "99032b2b2b2b2b2b2b2b2b"},
	{0, -2, 1, 1, 21, 22, "9b70332b2b2b2b2b2b2b2b"},
	{0, -2, 2, 0, 17, 22, "9903672b2b2b2b2b2b2b2b"},
	{0, -2, 2, 1, 21, 26, "9b70366b2b2b2b2b2b2b2b"},
	{0, -1, 1, 0, 50, 51, "9903848d159e2b2b2b2b2b"},
	{0, -1, 1, 1, 54, 55, "9b703848d159e32b2b2b2b"},
	{0, -1, 2, 0, 50, 55, "9903848d159e132b2b2b2b"},
	{0, -1, 2, 1, 54, 59, "9b703848d159e12b2b2b2b"},
	{0, 0, 1, 0, 26, 27, "9903c0eb2b2b2b2b2b2b2b"},
	{0, 0, 1, 1, 30, 31, "9b703c0f2b2b2b2b2b2b2b"},
	{0, 0, 2, 0, 26, 31, "9903c0d32b2b2b2b2b2b2b"},
	{0, 0, 2, 1, 30, 35, "9b703c0d2b2b2b2b2b2b2b"},
	{0, 15, 1, 0, 86, 87, "9903fccccccccccccccccf"},
	{1, -2, 1, 0, 21, 22, "9913332b2b2b2b2b2b2b2b"},
	{1, -2, 1, 1, 25, 26, "9b71332b2b2b2b2b2b2b2b"},
	{1, -2, 2, 0, 21, 26, "9913366b2b2b2b2b2b2b2b"},
	{1, -2, 2, 1, 25, 30, "9b7133672b2b2b2b2b2b2b"},
	{1, -1, 1, 0, 54, 55, "99133848d159e32b2b2b2b"},
	{1, -1, 1, 1, 58, 59, "9b7133848d159e2b2b2b2b"},
	{1, -1, 2, 0, 54, 59, "99133848d159e12b2b2b2b"},
	{1, -1, 2, 1, 58, 63, "9b7133848d159e132b2b2b"},
	{1, 0, 1, 0, 30, 31, "99133c0f2b2b2b2b2b2b2b"},
	{1, 0, 1, 1, 34, 35, "9b7133c0eb2b2b2b2b2b2b"},
	{1, 0, 2, 0, 30, 35, "99133c0d2b2b2b2b2b2b2b"},
	{1, 0, 2, 1, 34, 39, "9b7133c0d32b2b2b2b2b2b"},
	{2, -2, 1, 0, 25, 26, "9923332b2b2b2b2b2b2b2b"},
	{2, -2, 1, 1, 29, 30, "9b7233332b2b2b2b2b2b2b"},
	{2, -2, 2, 0, 25, 30, "992333672b2b2b2b2b2b2b"},
	{2, -2, 2, 1, 29, 34, "9b7233366b2b2b2b2b2b2b"},
	{2, -1, 1, 0, 58, 59, "992333848d159e2b2b2b2b"},
	{2, -1, 1, 1, 62, 63, "9b72333848d159e32b2b2b"},
	{2, -1, 2, 0, 58, 63, "992333848d159e132b2b2b"},
	{2, -1, 2, 1, 62, 67, "9b72333848d159e12b2b2b"},
	{2, 0, 1, 0, 34, 35, "992333c0eb2b2b2b2b2b2b"},
	{2, 0, 1, 1, 38, 39, "9b72333c0f2b2b2b2b2b2b"},
	{2, 0, 2, 0, 34, 39, "992333c0d32b2b2b2b2b2b"},
	{2, 0, 2, 1, 38, 43, "9b72333c0d2b2b2b2b2b2b"},
	{3, -2, 1, 0, 29, 30, "993333332b2b2b2b2b2b2b"},
	{3, -2, 1, 1, 33, 34, "9b7333332b2b2b2b2b2b2b"},
	{3, -2, 2, 0, 29, 34, "993333366b2b2b2b2b2b2b"},
	{3, -2, 2, 1, 33, 38, "9b733333672b2b2b2b2b2b"},
	{3, -1, 1, 0, 62, 63, "9933333848d159e32b2b2b"},
	{3, -1, 1, 1, 66, 67, "9b733333848d159e2b2b2b"},
	{3, -1, 2, 0, 62, 67, "9933333848d159e12b2b2b"},
	{3, -1, 2, 1, 66, 71, "9b733333848d159e132b2b"},
	{3, 0, 1, 0, 38, 39, "9933333c0f2b2b2b2b2b2b"},
	{3, 0, 1, 1, 42, 43, "9b733333c0eb2b2b2b2b2b"},
	{3, 0, 2, 0, 38, 43, "9933333c0d2b2b2b2b2b2b"},
	{3, 0, 2, 1, 42, 47, "9b733333c0d32b2b2b2b2b"},
	{4, -2, 1, 0, 33, 34, "994333332b2b2b2b2b2b2b"},
	{4, -2, 1, 1, 37, 38, "9b743333332b2b2b2b2b2b"},
	{4, -2, 2, 0, 33, 38, "99433333672b2b2b2b2b2b"},
	{4, -2, 2, 1, 37, 42, "9b743333366b2b2b2b2b2b"},
	{4, -1, 1, 0, 66, 67, "99433333848d159e2b2b2b"},
	{4, -1, 1, 1, 70, 71, "9b7433333848d159e32b2b"},
	{4, -1, 2, 0, 66, 71, "99433333848d159e132b2b"},
	{4, -1, 2, 1, 70, 75, "9b7433333848d159e12b2b"},
	{4, 0, 1, 0, 42, 43, "99433333c0eb2b2b2b2b2b"},
	{4, 0, 1, 1, 46, 47, "9b7433333c0f2b2b2b2b2b"},
	{4, 0, 2, 0, 42, 47, "99433333c0d32b2b2b2b2b"},
	{4, 0, 2, 1, 46, 51, "9b7433333c0d2b2b2b2b2b"},
	{5, -2, 1, 0, 37, 38, "99533333332b2b2b2b2b2b"},
	{5, -2, 1, 1, 41, 42, "9b753333332b2b2b2b2b2b"},
	{5, -2, 2, 0, 37, 42, "99533333366b2b2b2b2b2b"},
	{5, -2, 2, 1, 41, 46, "9b75333333672b2b2b2b2b"},
	{5, -1, 1, 0, 70, 71, "995333333848d159e32b2b"},
	{5, -1, 1, 1, 74, 75, "9b75333333848d159e2b2b"},
	{5, -1, 2, 0, 70, 75, "995333333848d159e12b2b"},
	{5, -1, 2, 1, 74, 79, "9b75333333848d159e132b"},
	{5, 0, 1, 0, 46, 47, "995333333c0f2b2b2b2b2b"},
	{5, 0, 1, 1, 50, 51, "9b75333333c0eb2b2b2b2b"},
	{5, 0, 2, 0, 46, 51, "995333333c0d2b2b2b2b2b"},
	{5, 0, 2, 1, 50, 55, "9b75333333c0d32b2b2b2b"},
	{6, -2, 1, 0, 41, 42, "99633333332b2b2b2b2b2b"},
	{6, -2, 1, 1, 45, 46, "9b76333333332b2b2b2b2b"},
	{6, -2, 2, 0, 41, 46, "9963333333672b2b2b2b2b"},
	{6, -2, 2, 1, 45, 50, "9b76333333366b2b2b2b2b"},
	{6, -1, 1, 0, 74, 75, "9963333333848d159e2b2b"},
	{6, -1, 1, 1, 78, 79, "9b763333333848d159e32b"},
	{6, -1, 2, 0, 74, 79, "9963333333848d159e132b"},
	{6, -1, 2, 1, 78, 83, "9b763333333848d159e12b"},
	{6, 0, 1, 0, 50, 51, "9963333333c0eb2b2b2b2b"},
	{6, 0, 1, 1, 54, 55, "9b763333333c0f2b2b2b2b"},
	{6, 0, 2, 0, 50, 55, "9963333333c0d32b2b2b2b"},
	{6, 0, 2, 1, 54, 59, "9b763333333c0d2b2b2b2b"},
	{7, -2, 1, 0, 45, 46, "9973333333332b2b2b2b2b"},
	{7, -2, 1, 1, 49, 50, "9b77333333332b2b2b2b2b"},
	{7, -2, 2, 0, 45, 50, "9973333333366b2b2b2b2b"},
	{7, -2, 2, 1, 49, 54, "9b7733333333672b2b2b2b"},
	{7, -1, 1, 0, 78, 79, "99733333333848d159e32b"},
	{7, -1, 1, 1, 82, 83, "9b7733333333848d159e2b"},
	{7, -1, 2, 0, 78, 83, "99733333333848d159e12b"},
	{7, -1, 2, 1, 82, 87, "9b7733333333848d159e13"},
	{7, 0, 1, 0, 54, 55, "99733333333c0f2b2b2b2b"},
	{7, 0, 1, 1, 58, 59, "9b7733333333c0eb2b2b2b"},
	{7, 0, 2, 0, 54, 59, "99733333333c0d2b2b2b2b"},
	{7, 0, 2, 1, 58, 63, "9b7733333333c0d32b2b2b"},
	{8, -2, 1, 0, 49, 50, "9983333333332b2b2b2b2b"},
	{8, -2, 1, 1, 53, 54, "9b7833333333332b2b2b2b"},
	{8, -2, 2, 0, 49, 54, "998333333333672b2b2b2b"},
	{8, -2, 2, 1, 53, 58, "9b7833333333366b2b2b2b"},
	{8, -1, 1, 0, 82, 83, "998333333333848d159e2b"},
	{8, -1, 1, 1, 86, 87, "9b78333333333848d159e3"},
	{8, -1, 2, 0, 82, 87, "998333333333848d159e13"},
	{8, 0, 1, 0, 58, 59, "998333333333c0eb2b2b2b"},
	{8, 0, 1, 1, 62, 63, "9b78333333333c0f2b2b2b"},
	{8, 0, 2, 0, 58, 63, "998333333333c0d32b2b2b"},
	{8, 0, 2, 1, 62, 67, "9b78333333333c0d2b2b2b"},
	{9, -2, 1, 0, 53, 54, "999333333333332b2b2b2b"},
	{9, -2, 1, 1, 57, 58, "9b7933333333332b2b2b2b"},
	{9, -2, 2, 0, 53, 58, "999333333333366b2b2b2b"},
	{9, -2, 2, 1, 57, 62, "9b793333333333672b2b2b"},
	{9, -1, 1, 0, 86, 87, "9993333333333848d159e3"},
	{9, 0, 1, 0, 62, 63, "9993333333333c0f2b2b2b"},
	{9, 0, 1, 1, 66, 67, "9b793333333333c0eb2b2b"},
	{9, 0, 2, 0, 62, 67, "9993333333333c0d2b2b2b"},
	{9, 0, 2, 1, 66, 71, "9b793333333333c0d32b2b"},
	{10, -2, 1, 0, 57, 58, "99a333333333332b2b2b2b"},
	{10, -2, 1, 1, 61, 62, "9b7a3333333333332b2b2b"},
	{10, -2, 2, 0, 57, 62, "99a33333333333672b2b2b"},
	{10, -2, 2, 1, 61, 66, "9b7a3333333333366b2b2b"},
	{10, 0, 1, 0, 66, 67, "99a33333333333c0eb2b2b"},
	{10, 0, 1, 1, 70, 71, "9b7a33333333333c0f2b2b"},
	{10, 0, 2, 0, 66, 71, "99a33333333333c0d32b2b"},
	{10, 0, 2, 1, 70, 75, "9b7a33333333333c0d2b2b"},
	{11, -2, 1, 0, 61, 62, "99b33333333333332b2b2b"},
	{11, -2, 1, 1, 65, 66, "9b7b3333333333332b2b2b"},
	{11, -2, 2, 0, 61, 66, "99b33333333333366b2b2b"},
	{11, -2, 2, 1, 65, 70, "9b7b333333333333672b2b"},
	{11, 0, 1, 0, 70, 71, "99b333333333333c0f2b2b"},
	{11, 0, 1, 1, 74, 75, "9b7b333333333333c0eb2b"},
	{11, 0, 2, 0, 70, 75, "99b333333333333c0d2b2b"},
	{11, 0, 2, 1, 74, 79, "9b7b333333333333c0d32b"},
	{12, -2, 1, 0, 65, 66, "99c33333333333332b2b2b"},
	{12, -2, 1, 1, 69, 70, "9b7c333333333333332b2b"},
	{12, -2, 2, 0, 65, 70, "99c3333333333333672b2b"},
	{12, -2, 2, 1, 69, 74, "9b7c333333333333366b2b"},
	{12, 0, 1, 0, 74, 75, "99c3333333333333c0eb2b"},
	{12, 0, 1, 1, 78, 79, "9b7c3333333333333c0f2b"},
	{12, 0, 2, 0, 74, 79, "99c3333333333333c0d32b"},
	{12, 0, 2, 1, 78, 83, "9b7c3333333333333c0d2b"},
	{13, -2, 1, 0, 69, 70, "99d3333333333333332b2b"},
	{13, -2, 1, 1, 73, 74, "9b7d333333333333332b2b"},
	{13, -2, 2, 0, 69, 74, "99d3333333333333366b2b"},
	{13, -2, 2, 1, 73, 78, "9b7d33333333333333672b"},
	{13, 0, 1, 0, 78, 79, "99d33333333333333c0f2b"},
	{13, 0, 1, 1, 82, 83, "9b7d33333333333333c0eb"},
	{13, 0, 2, 0, 78, 83, "99d33333333333333c0d2b"},
	{13, 0, 2, 1, 82, 87, "9b7d33333333333333c0d3"},
	{14, -2, 1, 0, 73, 74, "99e3333333333333332b2b"},
	{14, -2, 1, 1, 77, 78, "9b7e33333333333333332b"},
	{14, -2, 2, 0, 73, 78, "99e333333333333333672b"},
	{14, -2, 2, 1, 77, 82, "9b7e33333333333333366b"},
	{14, 0, 1, 0, 82, 83, "99e333333333333333c0eb"},
	{14, 0, 1, 1, 86, 87, "9b7e333333333333333c0f"},
	{14, 0, 2, 0, 82, 87, "99e333333333333333c0d3"},
	{15, -2, 1, 0, 77, 78, "99f333333333333333332b"},
	{15, -2, 1, 1, 81, 82, "9b7f33333333333333332b"},
	{15, -2, 2, 0, 77, 82, "99f333333333333333366b"},
	{15, -2, 2, 1, 81, 86, "9b7f333333333333333367"},
	{15, 0, 1, 0, 86, 87, "99f3333333333333333c0f"},
}

func ecPagingIdentity(n int) MobileIdentityStruct {
	if n < 0 {
		return MobileIdentityStruct{PTMSIChoice: MobileIdentityStructPTMSIChoice{Alternative: MobileIdentityStructPTMSIChoiceAlternativePTMSI, PTMSI: &MobileIdentityStructPTMSIChoicePTMSI{PTMSI: 0x12345678}}}
	}
	b := bytes.Repeat([]byte{0x33}, (n+2)/2)
	if (n+1)%2 != 0 {
		b[len(b)-1] = 0x30
	}
	return MobileIdentityStruct{PTMSIChoice: MobileIdentityStructPTMSIChoice{Alternative: MobileIdentityStructPTMSIChoiceAlternativeNumberOfIMSIDigits, NumberOfIMSIDigits: &MobileIdentityStructPTMSIChoiceNumberOfIMSIDigits{NumberOfIMSIDigits: uint8(n), IMSIDigits: runtime.BitString{Bytes: b, BitLength: 4 * (n + 1)}}}}
}
func ecPagingValue(first, second, ext, page int) ECPagingRequestMessageContent {
	v := ECPagingRequestMessageContent{MessageType: 9, UsedDLCoverageClass: 2, MobileIdentity1: ecPagingIdentity(first)}
	if second != -2 {
		p := ecPagingIdentity(second)
		v.MobileIdentity2 = &p
	}
	if page != 0 {
		p := uint8(11)
		v.ECPageExtension = &p
	}
	switch ext {
	case 0:
		v.PositioningEventPendingIndicatorChoice = ECPagingRequestMessageContentPositioningEventPendingIndicatorChoice{Alternative: ECPagingRequestMessageContentPositioningEventPendingIndicatorChoiceAlternativeAltUnlabeled, AltUnlabeled: &struct{}{}}
	case 1:
		v.PositioningEventPendingIndicatorChoice = ECPagingRequestMessageContentPositioningEventPendingIndicatorChoice{Alternative: ECPagingRequestMessageContentPositioningEventPendingIndicatorChoiceAlternativeAltL, AltL: &struct{}{}}
	case 2:
		v.PositioningEventPendingIndicatorChoice = ECPagingRequestMessageContentPositioningEventPendingIndicatorChoice{Alternative: ECPagingRequestMessageContentPositioningEventPendingIndicatorChoiceAlternativePositioningEventPendingIndicator, PositioningEventPendingIndicator: &ECPagingRequestMessageContentPositioningEventPendingIndicatorChoicePositioningEventPendingIndicator{PositioningEventPendingIndicator: 2, EnhancedCoverageRestrictionIndicator: 1}}
	}
	return v
}
func TestECPagingRequestSource(t *testing.T) {
	for _, tc := range ecPagingRequestVectors {
		wire := ecRRWire(t, tc.wire)
		want := ecPagingValue(tc.first, tc.second, tc.ext, tc.page)
		got, err := DecodeECPagingRequestMessageContent(wire)
		if err != nil || !runtime.SemanticallyEqual(got.Value, want) {
			t.Fatalf("identities %d/%d ext %d page %d: %#v %v", tc.first, tc.second, tc.ext, tc.page, got, err)
		}
		ecRRCheck(t, "9.1.63", "EC Paging Request message content", wire, want, tc.content)
		ecRRBound(t, wire, tc.mandatory, func(r *runtime.Reader) error { _, err := DecodeECPagingRequestMessageContentFrom(r); return err })
		got.Value.UsedDLCoverageClass = 1
		edited, err := EncodeECPagingRequestMessageContent(got.Value)
		if err != nil {
			t.Fatal(err)
		}
		again, err := DecodeECPagingRequestMessageContent(edited)
		if err != nil || again.Value.UsedDLCoverageClass != 1 {
			t.Fatal("edit lost", err)
		}
		if tc.ext == 2 {
			for cut := tc.mandatory + 1; cut < tc.content; cut++ {
				r := runtime.NewReader(wire)
				if _, err := r.PushLimit(cut); err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeECPagingRequestMessageContentFrom(r); err == nil {
					t.Fatalf("accepted incomplete H arm at %d", cut)
				}
			}
		}
		// The null alternative is available only at the exact received end.
		r := runtime.NewReader(wire)
		if _, err := r.PushLimit(tc.mandatory); err != nil {
			t.Fatal(err)
		}
		short, err := DecodeECPagingRequestMessageContentFrom(r)
		if err != nil || short.PositioningEventPendingIndicatorChoice.Alternative != ECPagingRequestMessageContentPositioningEventPendingIndicatorChoiceAlternativeAltUnlabeled || r.Position() != tc.mandatory {
			t.Fatalf("null at exact bound: %#v %v", short, err)
		}
	}
}
func TestECPagingRequestInvalidFields(t *testing.T) {
	// A null arm at a non-octet end cannot be padded to an 88-bit sender block
	// without selecting a different printed alternative.
	if _, err := EncodeECPagingRequestMessageContentCanonicalAtLength(ecPagingValue(0, -2, 0, 0), 11); err == nil {
		t.Fatal("canonical null invented a completed alternative")
	}

	for _, edit := range []func(*ECPagingRequestMessageContent){func(v *ECPagingRequestMessageContent) { v.MessageType = 8 }, func(v *ECPagingRequestMessageContent) {
		v.MobileIdentity1.PTMSIChoice.NumberOfIMSIDigits.NumberOfIMSIDigits = 16
	}, func(v *ECPagingRequestMessageContent) {
		v.MobileIdentity1.PTMSIChoice.NumberOfIMSIDigits.IMSIDigits.BitLength = 3
	}, func(v *ECPagingRequestMessageContent) {
		v.PositioningEventPendingIndicatorChoice.PositioningEventPendingIndicator.PositioningEventPendingIndicator = 4
	}} {
		v := ecPagingValue(0, -2, 2, 0)
		edit(&v)
		if _, err := EncodeECPagingRequestMessageContentCanonicalAtLength(v, 11); err == nil {
			t.Fatal("accepted invalid field")
		}
	}
}
func FuzzECPagingRequestReplay(f *testing.F) {
	for _, tc := range ecPagingRequestVectors {
		b, err := hex.DecodeString(tc.wire)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 256 {
			return
		}
		helper, helperErr := DecodeMobileIdentityStruct(wire)
		if helperErr == nil {
			encoded, err := EncodeMobileIdentityStruct(helper.Value)
			if err != nil || !bytes.Equal(encoded, wire) {
				t.Fatalf("helper replay %x: %v", encoded, err)
			}
		}
		d, err := DecodeECPagingRequestMessageContent(wire)
		if err != nil {
			return
		}
		b, err := EncodeECPagingRequestMessageContent(d.Value)
		if err != nil || !bytes.Equal(b, wire) {
			t.Fatalf("replay %x %v", b, err)
		}
	})
}

// TS 24.007 V20.0.0 Annex B §B.1.2.2 pp4046-4047: L/H follow the actual bit
// position, including a non-octet prefix supplied by a containing reader.
func TestECPagingRequestEveryOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin, ext, length int
		wire                string
	}{
		{0, 0, 17, "99035280"},
		{0, 1, 88, "99032b2b2b2b2b2b2b2b2ba5"},
		{0, 2, 88, "9903672b2b2b2b2b2b2b2ba5"},
		{1, 0, 17, "4c81a940"},
		{1, 1, 88, "4c81ab2b2b2b2b2b2b2b2b5280"},
		{1, 2, 88, "4c81932b2b2b2b2b2b2b2b5280"},
		{2, 0, 17, "2640d4a0"},
		{2, 1, 88, "2640cb2b2b2b2b2b2b2b2b2940"},
		{2, 2, 88, "2640d92b2b2b2b2b2b2b2b2940"},
		{3, 0, 17, "13206a50"},
		{3, 1, 88, "13206b2b2b2b2b2b2b2b2b34a0"},
		{3, 2, 88, "132064ab2b2b2b2b2b2b2b34a0"},
		{4, 0, 17, "09903528"},
		{4, 1, 88, "0990332b2b2b2b2b2b2b2b2a50"},
		{4, 2, 88, "0990366b2b2b2b2b2b2b2b2a50"},
		{5, 0, 17, "04c81a94"},
		{5, 1, 88, "04c81b2b2b2b2b2b2b2b2b2d28"},
		{5, 2, 88, "04c8192b2b2b2b2b2b2b2b2d28"},
		{6, 0, 17, "02640d4a"},
		{6, 1, 88, "02640d2b2b2b2b2b2b2b2b2a94"},
		{6, 2, 88, "02640c9b2b2b2b2b2b2b2b2a94"},
		{7, 0, 17, "013206a5"},
		{7, 1, 88, "0132062b2b2b2b2b2b2b2b2b4a"},
		{7, 2, 88, "013206cb2b2b2b2b2b2b2b2b4a"},
	} {
		wire := ecRRWire(t, tc.wire)
		want := ecPagingValue(0, -2, tc.ext, 0)
		r := runtime.NewReader(wire)
		if _, err := r.ReadUint(tc.origin); err != nil {
			t.Fatal(err)
		}
		old, err := r.PushLimit(tc.length)
		if err != nil {
			t.Fatal(err)
		}
		v, err := DecodeECPagingRequestMessageContentFrom(r)
		if err != nil || !runtime.SemanticallyEqual(v, want) || r.Position() != tc.origin+tc.length {
			t.Fatalf("origin %d ext %d: %#v %v", tc.origin, tc.ext, v, err)
		}
		if err := r.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		sentinel, err := r.ReadUint(8)
		if err != nil || sentinel != 0xa5 {
			t.Fatalf("sentinel origin %d: %x %v", tc.origin, sentinel, err)
		}
		w := runtime.NewWriter()
		if err := w.WriteUint(0, tc.origin); err != nil {
			t.Fatal(err)
		}
		// The containing writer supplies the exact canonical bit extent.
		w.WithWire(runtime.CanonicalTo(want, tc.origin+tc.length).Wire)
		prior, err := w.PushLimit(tc.length)
		if err != nil {
			t.Fatal(err)
		}
		if err := EncodeECPagingRequestMessageContentTo(w, want); err != nil {
			t.Fatal(err)
		}
		if err := w.PopLimit(prior); err != nil {
			t.Fatal(err)
		}
		w.WithWire(runtime.WireInfo{})
		if err := w.WriteUint(0xa5, 8); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, (8-w.Position()%8)%8); err != nil {
			t.Fatal(err)
		}
		encoded, err := w.Bytes()
		if err != nil || !bytes.Equal(encoded, wire) {
			t.Fatalf("writer origin %d ext %d: %x want %x %v", tc.origin, tc.ext, encoded, wire, err)
		}
	}
}

func TestECPagingRequestNestedEdits(t *testing.T) {
	v := ecPagingValue(0, -1, 2, 0)
	wire, err := EncodeECPagingRequestMessageContentCanonicalAtLength(v, 11)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeECPagingRequestMessageContent(wire)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Value.MobileIdentity1.PTMSIChoice.NumberOfIMSIDigits.IMSIDigits.Bytes[0] = 0x50
	decoded.Value.MobileIdentity2.PTMSIChoice.PTMSI.PTMSI = 0x87654321
	for pending := uint8(0); pending < 4; pending++ {
		for restriction := uint8(0); restriction < 4; restriction++ {
			extension := decoded.Value.PositioningEventPendingIndicatorChoice.PositioningEventPendingIndicator
			extension.PositioningEventPendingIndicator = pending
			extension.EnhancedCoverageRestrictionIndicator = restriction
			encoded, err := EncodeECPagingRequestMessageContent(decoded.Value)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeECPagingRequestMessageContent(encoded)
			if err != nil || !runtime.SemanticallyEqual(again.Value, decoded.Value) {
				t.Fatal("nested edit lost", err)
			}
			canonical, err := EncodeECPagingRequestMessageContentCanonicalAtLength(decoded.Value, 11)
			if err != nil || !bytes.Equal(canonical, encoded) {
				t.Fatal("nested edit canonical", err)
			}
		}
	}
}
