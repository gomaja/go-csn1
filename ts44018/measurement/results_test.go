package measurement

import (
	"bytes"
	"encoding/hex"
	"errors"
	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"reflect"
	"strings"
	"testing"
)

func resultsServing() MeasurementResultsFields {
	return MeasurementResultsFields{BAUSED: 1, DTXUSED: 0, RXLEVFULLSERVINGCELL: 21, N3GBAUSED: 1, MEASVALID: 0, RXLEVSUBSERVINGCELL: 42, SI23BAUSED: 1, RXQUALFULLSERVINGCELL: 3, RXQUALSUBSERVINGCELL: 5}
}
func resultsCSG(plmn bool) *ies.UTRANCSGMeasurementReportIE {
	v := &ies.UTRANCSGMeasurementReportIE{UTRANCGI: 0x123456, CSGID: 0x1234, AccessMode: 1, REPORTINGQUANTITY: 50}
	if plmn {
		v.PLMNID = &ies.PLMNIDStruct{MCC: 0x25f, MNC: 2}
	}
	return v
}

// TS 44.018 V19.0.0 table 10.5.2.20.0, shared serving-cell prefix
// approved on 2026-10-26; TS 44.060 V19.0.0 §12.61 for CSG.
func TestMeasurementResultsAllCountsAndCSG(t *testing.T) {
	for count := 0; count <= 7; count++ {
		for _, form := range []int{0, 1, 2} {
			if form == 1 && count > 2 || form == 2 && count > 0 {
				continue
			}
			reports := make([]NCELLReportStruct, count)
			for i := range reports {
				reports[i] = NCELLReportStruct{RXLEVNCELL: 45, BCCHFREQNCELL: 17, BSICNCELL: 37}
			}
			var v MeasurementResultsContents
			var err error
			var csg *ies.UTRANCSGMeasurementReportIE
			if form > 0 {
				csg = resultsCSG(form == 2)
			}
			if count == 7 {
				v, err = resultsBuildUnavailable(resultsServing())
			} else {
				v, err = resultsBuildReports(resultsServing(), reports, csg)
			}
			if err != nil {
				t.Fatalf("build %d/%d: %v", count, form, err)
			}
			wire, err := EncodeMeasurementResultsContents(v)
			if err != nil || len(wire) != 16 {
				t.Fatalf("encode %d/%d: %x %v", count, form, wire, err)
			}
			if wire[0] != 0x95 || wire[1] != 0xaa || wire[2]&0xfe != 0xba || int((uint16(wire[2]&1)<<2)|uint16(wire[3]>>6)) != count {
				t.Fatalf("shared field positions %d/%d: %x", count, form, wire)
			}
			d, err := DecodeMeasurementResultsContents(wire)
			if err != nil || d.BitsConsumed != 128 || d.Tail.BitLength != 0 || !resultsSameServing(d.Value.Content, resultsServing()) {
				t.Fatalf("decode %d/%d: %+v %v", count, form, d, err)
			}
			if count == 7 {
				if d.Value.Content.Count.Alternative != MeasurementResultsCountChoiceAlternativeUnavailable || d.Value.Content.Count.Unavailable == nil || d.Value.Content.Count.Unavailable.NONCELLM != 7 || d.Value.Content.Count.Reports != nil {
					t.Fatal("unavailable arm")
				}
			} else {
				r := d.Value.Content.Count.Reports
				if d.Value.Content.Count.Alternative != MeasurementResultsCountChoiceAlternativeReports || r == nil || d.Value.Content.Count.Unavailable != nil || int(r.NONCELLM) != count || len(r.NCELLReports) != count {
					t.Fatal("reports arm/count")
				}
				for _, n := range r.NCELLReports {
					if n.RXLEVNCELL != 45 || n.BCCHFREQNCELL != 17 || n.BSICNCELL != 37 {
						t.Fatal("neighbour fields")
					}
				}
				if (r.Suffix.CSG != nil) != (form > 0) {
					t.Fatal("CSG presence")
				}
				if form > 0 {
					got := r.Suffix.CSG.Report
					if got.UTRANCGI != 0x123456 || got.CSGID != 0x1234 || got.AccessMode != 1 || got.REPORTINGQUANTITY != 50 || (got.PLMNID != nil) != (form == 2) {
						t.Fatalf("CSG fields %+v", got)
					}
					if form == 2 && (got.PLMNID.MCC != 0x25f || got.PLMNID.MNC != 2) {
						t.Fatal("PLMN fields")
					}
				}
			}
			for _, encode := range []func(MeasurementResultsContents) ([]byte, error){EncodeMeasurementResultsContents, EncodeMeasurementResultsContentsCanonical, func(v MeasurementResultsContents) ([]byte, error) {
				return EncodeMeasurementResultsContentsCanonicalAtLength(v, 16)
			}} {
				again, err := encode(d.Value)
				if err != nil || !bytes.Equal(again, wire) {
					t.Fatalf("replay/canonical %d/%d: %x %v", count, form, again, err)
				}
			}
			// Every source-defined zero after the semantic fields is literal padding.
			end := 26 + 17*count
			if count == 7 {
				end = 26
			}
			if form == 1 {
				end += 64
			}
			if form == 2 {
				end += 88
			}
			for bit := end; bit < 128; bit++ {
				// With no CSG, the first suffix bit is the valid present selector.
				if count != 7 && form == 0 && bit == end {
					continue
				}
				bad := append([]byte(nil), wire...)
				bad[bit/8] |= 1 << uint(7-bit%8)
				if _, err := DecodeMeasurementResultsContents(bad); err == nil {
					t.Fatalf("accepted nonzero literal padding %d/%d bit %d", count, form, bit)
				}
			}
		}
	}
}

func TestMeasurementResultsSyntheticOracleVectors(t *testing.T) {
	for _, tc := range []struct {
		wire  string
		count int
	}{
		{"95aabbc0000000000000000000000000", 7},
		{"95aabaad8ca625200000000000000000", 2},
	} {
		wire, err := hex.DecodeString(tc.wire)
		if err != nil {
			t.Fatal(err)
		}
		d, err := DecodeMeasurementResultsContents(wire)
		if err != nil {
			t.Fatal(err)
		}
		if !resultsSameServing(d.Value.Content, resultsServing()) {
			t.Fatal("serving oracle")
		}
		if tc.count == 2 {
			if len(d.Value.Content.Count.Reports.NCELLReports) != 2 {
				t.Fatal("count")
			}
			a, b := d.Value.Content.Count.Reports.NCELLReports[0], d.Value.Content.Count.Reports.NCELLReports[1]
			if a.RXLEVNCELL != 45 || a.BCCHFREQNCELL != 17 || a.BSICNCELL != 37 || b.RXLEVNCELL != 12 || b.BCCHFREQNCELL != 9 || b.BSICNCELL != 18 {
				t.Fatal("NCELL field oracle")
			}
		}
		replay, err := EncodeMeasurementResultsContents(d.Value)
		if err != nil || !bytes.Equal(replay, wire) {
			t.Fatal("oracle replay", err)
		}
	}
}

func TestMeasurementResultsRejectsInvalidConstruction(t *testing.T) {
	serving := resultsServing()
	for _, edit := range []func(*MeasurementResultsFields){
		func(v *MeasurementResultsFields) { v.BAUSED = 2 }, func(v *MeasurementResultsFields) { v.DTXUSED = 2 },
		func(v *MeasurementResultsFields) { v.RXLEVFULLSERVINGCELL = 64 }, func(v *MeasurementResultsFields) { v.N3GBAUSED = 2 },
		func(v *MeasurementResultsFields) { v.MEASVALID = 2 }, func(v *MeasurementResultsFields) { v.RXLEVSUBSERVINGCELL = 64 },
		func(v *MeasurementResultsFields) { v.SI23BAUSED = 2 }, func(v *MeasurementResultsFields) { v.RXQUALFULLSERVINGCELL = 8 }, func(v *MeasurementResultsFields) { v.RXQUALSUBSERVINGCELL = 8 },
	} {
		bad := serving
		edit(&bad)
		if _, err := resultsBuildUnavailable(bad); err == nil {
			t.Fatal("accepted serving field overflow")
		}
	}
	if _, err := resultsBuildReports(serving, make([]NCELLReportStruct, 7), nil); err == nil {
		t.Fatal("accepted seventh report")
	}
	badArm := func(choice MeasurementResultsCountChoice) MeasurementResultsContents {
		v := MeasurementResultsContents{Content: serving}
		v.Content.Count = choice
		return v
	}
	for _, v := range []MeasurementResultsContents{
		badArm(MeasurementResultsCountChoice{Alternative: 99}),
		badArm(MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeUnavailable}),
		badArm(MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports}),
		badArm(MeasurementResultsCountChoice{Unavailable: &MeasurementResultsUnavailableArm{NONCELLM: 6}}),
		badArm(MeasurementResultsCountChoice{Unavailable: &MeasurementResultsUnavailableArm{NONCELLM: 7}, Reports: &MeasurementResultsReportsArm{}}),
		badArm(MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports, Reports: &MeasurementResultsReportsArm{NONCELLM: 1}}),
		badArm(MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports, Reports: &MeasurementResultsReportsArm{NCELLReports: make([]NCELLReportStruct, 1)}}),
		badArm(MeasurementResultsCountChoice{Alternative: MeasurementResultsCountChoiceAlternativeReports, Reports: &MeasurementResultsReportsArm{NONCELLM: 7, NCELLReports: make([]NCELLReportStruct, 7)}}),
	} {
		_, err := EncodeMeasurementResultsContents(v)
		var op *runtime.OperationError
		if err == nil || !errors.As(err, &op) || op.Location.Source.Clause != "10.5.2.20" {
			t.Fatalf("invalid construction %+v: %v", v, err)
		}
	}

	for _, n := range []NCELLReportStruct{{RXLEVNCELL: 64}, {BCCHFREQNCELL: 32}, {BSICNCELL: 64}} {
		if _, err := resultsBuildReports(serving, []NCELLReportStruct{n}, nil); err == nil {
			t.Fatal("nested NCELL overflow")
		}
	}
	for _, tc := range []struct {
		count int
		plmn  bool
	}{{3, false}, {1, true}} {
		if _, err := resultsBuildReports(serving, make([]NCELLReportStruct, tc.count), resultsCSG(tc.plmn)); err == nil {
			t.Fatal("CSG exceeded contents bound")
		}
	}
	v, err := resultsBuildUnavailable(serving)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{-1, 0, 15, 17} {
		if _, err := EncodeMeasurementResultsContentsCanonicalAtLength(v, n); err == nil {
			t.Fatalf("accepted extent %d", n)
		}
	}
	if err := EncodeMeasurementResultsContentsTo(nil, v); err == nil {
		t.Fatal("nil writer")
	}
	if _, err := DecodeMeasurementResultsContentsFrom(nil); err == nil {
		t.Fatal("nil reader")
	}
}

func TestMeasurementResultsRepeatedErrorIndex(t *testing.T) {
	v, err := resultsBuildReports(resultsServing(), make([]NCELLReportStruct, 2), nil)
	if err != nil {
		t.Fatal(err)
	}
	v.Content.Count.Reports.NCELLReports[1].RXLEVNCELL = 64
	_, err = EncodeMeasurementResultsContents(v)
	var op *runtime.OperationError
	if !errors.As(err, &op) || strings.Count(op.Location.Path, "[1]") != 1 {
		t.Fatalf("repeated source index: %v", err)
	}
}

func TestMeasurementResultsNestedOriginsAndEveryTruncation(t *testing.T) {
	serving := resultsServing()
	for _, count := range []int{0, 1, 2, 6, 7} {
		var v MeasurementResultsContents
		var err error
		if count == 7 {
			v, err = resultsBuildUnavailable(serving)
		} else {
			v, err = resultsBuildReports(serving, make([]NCELLReportStruct, count), nil)
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, origin := range []int{0, 3, 11} {
			w := runtime.NewWriter()
			if err := w.WriteUint(0, origin); err != nil {
				t.Fatal(err)
			}
			if err := EncodeMeasurementResultsContentsTo(w, v); err != nil {
				t.Fatal(err)
			}
			if w.Position() != origin+128 {
				t.Fatal("writer extent")
			}
			if err := w.PadMinimum((origin+128+7)/8*8, false); err != nil {
				t.Fatal(err)
			}
			wire, err := w.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			for bits := 0; bits <= 128; bits++ {
				r := runtime.NewReader(wire)
				if _, err := r.ReadUint(origin); err != nil {
					t.Fatal(err)
				}
				if _, err := r.PushLimit(bits); err != nil {
					t.Fatal(err)
				}
				got, err := DecodeMeasurementResultsContentsFrom(r)
				if bits == 128 {
					if err != nil || !resultsSameServing(got.Content, serving) || r.Position() != origin+128 {
						t.Fatalf("nested count %d: %+v %v", count, got, err)
					}
					continue
				}
				var de *runtime.DecodeError
				var op *runtime.OperationError
				if !errors.As(err, &de) || de.Kind != runtime.Truncated || de.Offset != origin+bits || !errors.As(err, &op) || op.Location.BitOffset != origin+bits {
					t.Fatalf("count %d origin %d truncated %d: %v", count, origin, bits, err)
				}
			}
		}
	}
	// A CSG presence selector which cannot fit the mandatory nested IE is
	// source truncation at the fixed contents boundary, even with 16 octets.
	for _, count := range []int{3, 4, 5} {
		v, err := resultsBuildReports(serving, make([]NCELLReportStruct, count), nil)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := EncodeMeasurementResultsContents(v)
		if err != nil {
			t.Fatal(err)
		}
		bit := 26 + 17*count
		wire[bit/8] |= 1 << uint(7-bit%8)
		_, err = DecodeMeasurementResultsContents(wire)
		var de *runtime.DecodeError
		if !errors.As(err, &de) || de.Kind != runtime.Truncated || de.Offset != 128 {
			t.Fatalf("nested CSG count %d: %v", count, err)
		}
	}
}

func TestMeasurementResultsNestedCSGOrigins(t *testing.T) {
	for _, tc := range []struct {
		count int
		plmn  bool
	}{{0, false}, {1, false}, {2, false}, {0, true}} {
		v, err := resultsBuildReports(resultsServing(), make([]NCELLReportStruct, tc.count), resultsCSG(tc.plmn))
		if err != nil {
			t.Fatal(err)
		}
		for _, origin := range []int{3, 11} {
			w := runtime.NewWriter()
			if err := w.WriteUint(0, origin); err != nil {
				t.Fatal(err)
			}
			if err := EncodeMeasurementResultsContentsTo(w, v); err != nil {
				t.Fatal(err)
			}
			if err := w.PadMinimum((origin+128+7)/8*8, false); err != nil {
				t.Fatal(err)
			}
			wire, err := w.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			r := runtime.NewReader(wire)
			if _, err := r.ReadUint(origin); err != nil {
				t.Fatal(err)
			}
			if _, err := r.PushLimit(128); err != nil {
				t.Fatal(err)
			}
			got, err := DecodeMeasurementResultsContentsFrom(r)
			if err != nil || got.Content.Count.Reports == nil || got.Content.Count.Reports.Suffix.CSG == nil || got.Content.Count.Reports.Suffix.CSG.Report.UTRANCGI != 0x123456 || (got.Content.Count.Reports.Suffix.CSG.Report.PLMNID != nil) != tc.plmn || r.Position() != origin+128 {
				t.Fatalf("origin %d CSG %+v: %+v %v", origin, tc, got, err)
			}
		}
	}
	// At an origin inside a larger PDU, an incomplete CSG field still names
	// the imported clause and the absolute end of the contents, exactly once.
	for _, origin := range []int{3, 11} {
		v, err := resultsBuildReports(resultsServing(), make([]NCELLReportStruct, 3), nil)
		if err != nil {
			t.Fatal(err)
		}
		w := runtime.NewWriter()
		if err := w.WriteUint(0, origin); err != nil {
			t.Fatal(err)
		}
		if err := EncodeMeasurementResultsContentsTo(w, v); err != nil {
			t.Fatal(err)
		}
		if err := w.PadMinimum((origin+128+7)/8*8, false); err != nil {
			t.Fatal(err)
		}
		wire, err := w.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		at := origin + 26 + 3*17
		wire[at/8] |= 1 << uint(7-at%8)
		r := runtime.NewReader(wire)
		if _, err := r.ReadUint(origin); err != nil {
			t.Fatal(err)
		}
		if _, err := r.PushLimit(128); err != nil {
			t.Fatal(err)
		}
		_, err = DecodeMeasurementResultsContentsFrom(r)
		var de *runtime.DecodeError
		var op *runtime.OperationError
		if !errors.As(err, &de) || de.Offset != origin+128 || !errors.As(err, &op) || op.Location.BitOffset != origin+128 || op.Location.Source.Standard != "TS 44.060" || op.Location.Source.Clause != "12.61" {
			t.Fatalf("absolute imported failure at origin %d: %v", origin, err)
		}
	}
}

// Each field is driven independently, including DTX_USED and MEAS_VALID,
// whose usual all-zero samples cannot distinguish their positions.
func TestMeasurementResultsServingBitPositions(t *testing.T) {
	for _, f := range []struct {
		name          string
		offset, width int
	}{
		{"BAUSED", 0, 1}, {"DTXUSED", 1, 1}, {"RXLEVFULLSERVINGCELL", 2, 6},
		{"N3GBAUSED", 8, 1}, {"MEASVALID", 9, 1}, {"RXLEVSUBSERVINGCELL", 10, 6},
		{"SI23BAUSED", 16, 1}, {"RXQUALFULLSERVINGCELL", 17, 3}, {"RXQUALSUBSERVINGCELL", 20, 3},
	} {
		var serving MeasurementResultsFields
		reflect.ValueOf(&serving).Elem().FieldByName(f.name).SetUint((1 << f.width) - 1)
		expected := make([]byte, 16)
		for bit := f.offset; bit < f.offset+f.width; bit++ {
			expected[bit/8] |= 1 << uint(7-bit%8)
		}
		v, err := resultsBuildReports(serving, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := EncodeMeasurementResultsContents(v)
		if err != nil || !bytes.Equal(wire, expected) {
			t.Fatalf("%s wire %x want %x: %v", f.name, wire, expected, err)
		}
		d, err := DecodeMeasurementResultsContents(expected)
		if err != nil || !resultsSameServing(d.Value.Content, serving) {
			t.Fatalf("%s fields %+v want %+v: %v", f.name, d.Value.Content, serving, err)
		}
	}
}

// Keep each invalid value otherwise fully encodable, so a second independent
// constraint cannot hide a missing count or choice check.
func TestMeasurementResultsConstructionChecksIndependently(t *testing.T) {
	for _, count := range []uint8{0, 2} {
		v, err := resultsBuildReports(resultsServing(), make([]NCELLReportStruct, 1), nil)
		if err != nil {
			t.Fatal(err)
		}
		v.Content.Count.Reports.NONCELLM = count
		if _, err := EncodeMeasurementResultsContents(v); err == nil {
			t.Fatalf("accepted one report with count %d", count)
		}
	}
	unavailable, err := resultsBuildUnavailable(resultsServing())
	if err != nil {
		t.Fatal(err)
	}
	unavailable.Content.Count.Reports = &MeasurementResultsReportsArm{}
	if _, err := EncodeMeasurementResultsContents(unavailable); err == nil {
		t.Fatal("ignored simultaneous reports arm")
	}
	reports, err := resultsBuildReports(resultsServing(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	reports.Content.Count.Unavailable = &MeasurementResultsUnavailableArm{NONCELLM: 7}
	if _, err := EncodeMeasurementResultsContents(reports); err == nil {
		t.Fatal("ignored simultaneous unavailable arm")
	}
}

// TS 24.007 V20.0.0 Annex B §§B.1.1.2, B.1.1.4 and B.2.1:
// null and a bounded literal-zero tail have one typed representative.
func TestMeasurementResultsSuffixHasNoPaddingCount(t *testing.T) {
	for count := 0; count <= 6; count++ {
		v, err := resultsBuildReports(resultsServing(), make([]NCELLReportStruct, count), nil)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := EncodeMeasurementResultsContents(v)
		if err != nil {
			t.Fatal(err)
		}
		d, err := DecodeMeasurementResultsContents(wire)
		if err != nil || d.Value.Content.Count.Reports.Suffix.Alternative != MeasurementResultsReportsSuffixAlternativeNull || d.Value.Content.Count.Reports.Suffix.Null == nil {
			t.Fatalf("count %d: %+v %v", count, d, err)
		}
		if !runtime.SemanticallyEqual(v, d.Value) {
			t.Fatalf("count %d typed value differs", count)
		}
	}
}
