package registry_test

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts24008/classmark"
	"github.com/gomaja/go-csn1/ts36331/uecapability"
	"github.com/gomaja/go-csn1/ts44018/measurement"
	"github.com/gomaja/go-csn1/ts44018/restoctets"
)

// These GERAN corpus records include live traces. Keep their bytes outside the
// repository and opt in explicitly for local validation.
func TestGERANTrainingData(t *testing.T) {
	root := os.Getenv("GO_CSN1_TRAINING_DATA")
	if root == "" {
		t.Skip("set GO_CSN1_TRAINING_DATA to run the local GERAN corpus")
	}
	if root == "~" || strings.HasPrefix(root, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		root = filepath.Join(home, strings.TrimPrefix(root, "~/"))
	}
	for _, name := range []string{
		"ue-capability/geran-cs.jsonl", "ue-capability/geran-ps.jsonl",
		"a-interface/classmark2.jsonl", "a-interface/classmark2-vendor-altered.jsonl",
		"a-interface/classmark3.jsonl", "um-ccch-bcch/rest-octets.jsonl",
		"um-ccch-bcch/immediate-assignment-messages.jsonl",
		"um-sacch/enhanced-measurement-report-synthetic.jsonl",
	} {
		t.Run(name, func(t *testing.T) { checkGERANFile(t, filepath.Join(root, "geran", name), name) })
	}
}

type geranRecord struct {
	Hex           string `json:"hex"`
	RestOctets    string `json:"rest_octets"`
	Count         int    `json:"count"`
	Type          string `json:"type"`
	Expected      string `json:"expected"`
	LibraryResult string `json:"library_result"`
	Name          string `json:"name"`
	Source        string `json:"source"`
	Layout        struct {
		RestOctetsLength int `json:"rest_octets_length"`
	} `json:"layout"`
}

type geranResult struct {
	bits                   int
	tail                   int
	encoded                []byte
	err                    error
	canonicalErr           error
	spare3, spare4, spare5 bool
	ia                     *restoctets.IARestOctets
	iar                    *restoctets.IARRestOctets
}

func checkGERANFile(t *testing.T, path, name string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 4096), 4<<20)
	var records, matched, changed int
	var mismatches []string
	for scan.Scan() {
		records++
		var record geranRecord
		if err := json.Unmarshal(scan.Bytes(), &record); err != nil {
			t.Fatalf("row %d: %v", records, err)
		}
		wireText := record.Hex
		if record.RestOctets != "" {
			wireText = record.RestOctets
		}
		wire, err := hex.DecodeString(wireText)
		if err != nil {
			t.Fatalf("row %d: invalid hex: %v", records, err)
		}
		if record.RestOctets != "" {
			full, err := hex.DecodeString(record.Hex)
			if err != nil || !bytes.HasSuffix(full, wire) || len(wire) != record.Layout.RestOctetsLength {
				mismatches = append(mismatches, fmt.Sprintf("%d: rest octets do not match message suffix or declared length", records))
				continue
			}
		}
		var why string
		var got geranResult
		if strings.HasSuffix(name, "enhanced-measurement-report-synthetic.jsonl") {
			why = compareEMRExpected(records, record, wire)
		} else {
			got = decodeGERANRecord(name, record.Type, wire)
			why = compareGERANExpected(record, wire, got)
		}
		if why != "" {
			mismatches = append(mismatches, fmt.Sprintf("%d: %s", records, why))
		} else {
			matched++
		}
		if !strings.HasSuffix(name, "enhanced-measurement-report-synthetic.jsonl") && oldGERANMismatch(record) && (got.err == nil || strings.HasPrefix(record.Expected, "Rejected as truncated")) {
			changed++
		}
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(name, "enhanced-measurement-report-synthetic.jsonl") && records != 7 {
		t.Errorf("EMR records=%d, want 7", records)
	}
	t.Logf("records=%d matched=%d mismatched=%d changed_vs_old=%d", records, matched, len(mismatches), changed)
	for _, why := range mismatches {
		t.Error(why)
	}
}

func compareEMRExpected(row int, record geranRecord, wire []byte) string {
	// TS 44.018 V19.0.0 §9.1.55. The seventh file record (10022b)
	// was produced by an old encoder with 0x2b inside the pre-Rel-8
	// bitmap. It is truncated at bit 24, not a conforming round trip.
	expected := []struct{ name, result string }{
		{"flags only", "valid: decodes, no reports"},
		{"serving cell and invalid BSIC", "valid: decodes"},
		{"bitmap to the end of the message", "valid: decodes; the reporting bitmap runs to the end of the message"},
		{"release 8 bitmap", "valid: decodes"},
		{"release 9 UTRAN CSG and release 11 SI23_BA_USED", "valid: decodes, including the release 9 and 11 additions"},
		{"release 8 E-UTRAN measurement report", "valid: decodes to bitmap length 1 (report 33) and two E-UTRAN cells"},
		{"encoder output of a two-entry absent bitmap", "must round-trip: the encoder's own output for a value with two absent bitmap positions should decode back to that value"},
	}
	if row < 1 || row > len(expected) || record.Name != expected[row-1].name || record.Expected != expected[row-1].result || record.Source != "synthetic" || !strings.Contains(record.Type, "TS 44.018 V19.0.0 §9.1.55") {
		return "unexpected EMR training record or expectation"
	}
	d, err := measurement.DecodeEnhancedMeasurementReport(wire)
	if row == 7 {
		var decodeErr *runtime.DecodeError
		if !errors.As(err, &decodeErr) || decodeErr.Kind != runtime.Truncated || decodeErr.Offset != 24 {
			return fmt.Sprintf("obsolete encoder output: want Truncated at bit 24, got %v", err)
		}
		return ""
	}
	if err != nil {
		return fmt.Sprintf("decode: %v", err)
	}
	encoded, err := measurement.EncodeEnhancedMeasurementReport(d.Value)
	if err != nil || !bytes.Equal(encoded, wire) || d.BitsConsumed != len(wire)*8 || d.Tail.BitLength != 0 {
		return fmt.Sprintf("wire round trip or boundary: %x -> %x, bits=%d tail=%d: %v", wire, encoded, d.BitsConsumed, d.Tail.BitLength, err)
	}
	canonical, err := measurement.EncodeEnhancedMeasurementReportCanonical(d.Value)
	if err != nil {
		return fmt.Sprintf("canonical encode: %v", err)
	}
	again, err := measurement.DecodeEnhancedMeasurementReport(canonical)
	if err != nil || !equivalentEMR(d.Value, again.Value) {
		return fmt.Sprintf("canonical typed round trip: %v", err)
	}
	v := d.Value
	switch row {
	case 1:
		if v.ServingCellData != nil || len(v.RepeatedInvalidBSICInformationList) != 0 || v.REPORTINGQUANTITYList != nil || v.BITMAPLENGTHChoice.BITMAPLENGTH != nil {
			return "flags-only message contains a report"
		}
	case 2:
		s := v.ServingCellData
		b := v.RepeatedInvalidBSICInformationList
		if s == nil || s.DTXUSED != 1 || s.RXLEVVAL != 40 || s.RXQUALFULL != 2 || s.MEANBEP != 20 || s.CVBEP != 3 || s.NBRRCVDBLOCKS != 17 || len(b) != 2 || b[0].RepeatedInvalidBSICInformation.BCCHFREQNCELL != 3 || b[0].RepeatedInvalidBSICInformation.BSIC != 42 || b[0].RepeatedInvalidBSICInformation.RXLEVNCELL != 30 || b[1].RepeatedInvalidBSICInformation.BCCHFREQNCELL != 17 || b[1].RepeatedInvalidBSICInformation.BSIC != 5 || b[1].RepeatedInvalidBSICInformation.RXLEVNCELL != 12 {
			return "serving cell or invalid BSIC values differ"
		}
	case 3:
		if v.REPORTINGQUANTITYList == nil || len(*v.REPORTINGQUANTITYList) != 96 || v.BITMAPLENGTHChoice.BITMAPLENGTH != nil {
			return "pre-Rel-8 bitmap does not occupy 96 typed positions"
		}
	case 4, 5, 6:
		b := v.BITMAPLENGTHChoice.BITMAPLENGTH
		if b == nil || b.BITMAPLENGTH != 1 || len(b.REPORTINGQUANTITYList) != 2 || b.REPORTINGQUANTITYList[0] != nil || b.REPORTINGQUANTITYList[1] == nil {
			return "Rel-8 bitmap length or positions differ"
		}
		want := uint8(33)
		if row == 5 {
			want = 12
		}
		if *b.REPORTINGQUANTITYList[1] != want {
			return "Rel-8 reporting quantity differs"
		}
		if row == 5 {
			u := b.UTRANCSGMeasurementReportChoice.UTRANCSGMeasurementReport
			if u == nil || u.UTRANCSGMeasurementReport == nil || u.UTRANCSGMeasurementReport.UTRANCGI != 1193046 || u.UTRANCSGMeasurementReport.PLMNID.MCC != 607 || u.UTRANCSGMeasurementReport.PLMNID.MNC != 2 || u.UTRANCSGMeasurementReport.CSGID != 4660 || u.UTRANCSGMeasurementReport.AccessMode != 1 || u.UTRANCSGMeasurementReport.REPORTINGQUANTITY != 50 || u.SI23BAUSEDChoice.SI23BAUSED == nil || u.SI23BAUSEDChoice.SI23BAUSED.SI23BAUSED != 1 {
				return "Rel-9 UTRAN CSG or Rel-11 SI23 values differ"
			}
		}
		if row == 6 {
			e := b.EUTRANMeasurementReport
			if e == nil || e.NEUTRAN != 1 || len(e.EUTRANFREQUENCYINDEXGroupList) != 2 || e.EUTRANFREQUENCYINDEXGroupList[0].EUTRANFREQUENCYINDEX != 2 || e.EUTRANFREQUENCYINDEXGroupList[0].CELLIDENTITY != 301 || e.EUTRANFREQUENCYINDEXGroupList[0].REPORTINGQUANTITY != 45 || e.EUTRANFREQUENCYINDEXGroupList[1].EUTRANFREQUENCYINDEX != 5 || e.EUTRANFREQUENCYINDEXGroupList[1].CELLIDENTITY != 17 || e.EUTRANFREQUENCYINDEXGroupList[1].REPORTINGQUANTITY != 20 {
				return "E-UTRAN cell values differ"
			}
		}
	}
	return ""
}

func equivalentEMR(a, b measurement.EnhancedMeasurementReport) bool {
	// TS 44.018 V19.0.0 §9.1.55: absent pre-Rel-8 no-report
	// positions up to 96 are redundant when the bitmap is present.
	normalize := func(v measurement.EnhancedMeasurementReport) measurement.EnhancedMeasurementReport {
		v = runtime.Canonical(v)
		if v.REPORTINGQUANTITYList != nil {
			list := append([]*uint8(nil), (*v.REPORTINGQUANTITYList)...)
			for len(list) < 96 {
				list = append(list, nil)
			}
			v.REPORTINGQUANTITYList = &list
		}
		return v
	}
	return runtime.SemanticallyEqual(normalize(a), normalize(b))
}

func checkCanonical[T any](value T, encode func(T) ([]byte, error), decode func([]byte) (runtime.Decoded[T], error)) error {
	canonical, err := encode(value)
	if err != nil {
		return err
	}
	again, err := decode(canonical)
	if err != nil {
		return err
	}
	if !runtime.SemanticallyEqual(value, again.Value) {
		return fmt.Errorf("canonical encoding changed typed semantics")
	}
	return nil
}

func decodeGERANRecord(name, typ string, wire []byte) geranResult {
	var out geranResult
	switch {
	case strings.HasSuffix(name, "geran-cs.jsonl"):
		d, err := uecapability.DecodeGERANCS(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.encoded, out.err = uecapability.EncodeGERANCS(d.Value)
			out.canonicalErr = checkCanonical(d.Value, uecapability.EncodeGERANCSCanonical, uecapability.DecodeGERANCS)
		}
	case strings.HasSuffix(name, "geran-ps.jsonl"):
		d, err := uecapability.DecodeGERANPS(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.encoded, out.err = uecapability.EncodeGERANPS(d.Value)
			out.canonicalErr = checkCanonical(d.Value, uecapability.EncodeGERANPSCanonical, uecapability.DecodeGERANPS)
		}
	case strings.Contains(name, "classmark2"):
		d, err := uecapability.DecodeClassmark2ValuePart(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.spare3, out.spare4, out.spare5 = d.Value.Spare3, d.Value.Spare4, d.Value.Spare5
			out.encoded, out.err = uecapability.EncodeClassmark2ValuePart(d.Value)
			out.canonicalErr = checkCanonical(d.Value, uecapability.EncodeClassmark2ValuePartCanonical, uecapability.DecodeClassmark2ValuePart)
		}
	case strings.HasSuffix(name, "classmark3.jsonl"):
		d, err := classmark.DecodeClassmark3ValuePart(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.encoded, out.err = classmark.EncodeClassmark3ValuePart(d.Value)
			out.canonicalErr = checkCanonical(d.Value, classmark.EncodeClassmark3ValuePartCanonical, classmark.DecodeClassmark3ValuePart)
		}
	case strings.Contains(typ, "REJECT") || strings.Contains(typ, "IAR Rest"):
		d, err := restoctets.DecodeIARRestOctets(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.iar = &d.Value
			out.encoded, out.err = restoctets.EncodeIARRestOctets(d.Value)
			out.canonicalErr = checkCanonical(d.Value, restoctets.EncodeIARRestOctetsCanonical, restoctets.DecodeIARRestOctets)
		}
	default:
		d, err := restoctets.DecodeIARestOctets(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.ia = &d.Value
			out.encoded, out.err = restoctets.EncodeIARestOctets(d.Value)
			out.canonicalErr = checkCanonical(d.Value, restoctets.EncodeIARestOctetsCanonical, restoctets.DecodeIARestOctets)
		}
	}
	return out
}

func compareGERANExpected(record geranRecord, wire []byte, got geranResult) string {
	if strings.HasPrefix(record.Expected, "Rejected as truncated") {
		var de *runtime.DecodeError
		if !errors.As(got.err, &de) || de.Kind != runtime.Truncated || de.Offset != len(wire)*8 {
			return fmt.Sprintf("wanted Truncated at input end, got %v", got.err)
		}
		return ""
	}
	if got.err != nil {
		return fmt.Sprintf("decode or encode: %v", got.err)
	}
	if got.canonicalErr != nil {
		return fmt.Sprintf("canonical typed round trip: %v", got.canonicalErr)
	}
	if !bytes.Equal(got.encoded, wire) {
		return "byte round trip differs"
	}
	if strings.Contains(record.Expected, "400 bits") && got.bits != 400 {
		return fmt.Sprintf("consumed %d bits, want 400", got.bits)
	}
	if record.Expected == "Decode succeeds; spare bits are zero." && (got.spare3 || got.spare4 || got.spare5) {
		return "Classmark 2 spare bit is set"
	}
	if strings.Contains(record.Expected, "Tail should be empty") || strings.Contains(record.Expected, "empty Tail") {
		if got.tail != 0 {
			return fmt.Sprintf("trailing spare bits left as %d-bit Tail", got.tail)
		}
	}
	if strings.Contains(record.Expected, "spare bit(s) set") {
		for _, check := range []struct {
			name   string
			actual bool
		}{{"Spare3", got.spare3}, {"Spare4", got.spare4}, {"Spare5", got.spare5}} {
			want := strings.Contains(record.Expected, check.name)
			if check.actual != want {
				return fmt.Sprintf("%s=%v, want %v", check.name, check.actual, want)
			}
		}
	}
	if got.ia != nil && strings.Contains(record.Expected, "spare padding") {
		v := got.ia
		if v.CompressedInterRATHOINFOINDChoice.Alternative != restoctets.IARestOctetsCompressedInterRATHOINFOINDChoiceAlternativeRCC ||
			v.CompressedInterRATHOINFOINDChoice.RCC == nil || v.CompressedInterRATHOINFOINDChoice.RCC.CompressedInterRATHOINFOIND != 0 ||
			v.MultilaterationInformationRequest != nil || v.PEOIMMCellGroupDetails != nil {
			return "IA padding selected a present release addition or wrong L/H value"
		}
	}
	if got.iar != nil && strings.Contains(record.Expected, "spare padding") {
		v := got.iar
		if v.ExtendedRA3 == nil || *v.ExtendedRA3 != 11 || v.RCCChoice.RCC != nil || v.PEOIMMCellGroupDetails != nil {
			return "IAR padding selected wrong extended RA or present Rel-15 addition"
		}
	}
	return ""
}

func oldGERANMismatch(record geranRecord) bool {
	if strings.HasPrefix(record.LibraryResult, "error:") {
		return true
	}
	if strings.Contains(record.Expected, "Tail should be empty") || strings.Contains(record.Expected, "empty Tail") {
		return !strings.Contains(record.LibraryResult, "Tail 0 bits")
	}
	if strings.Contains(record.Expected, "spare padding") && strings.Contains(record.LibraryResult, "PEOIMMCellGroupDetails {") {
		return true
	}
	return false
}
