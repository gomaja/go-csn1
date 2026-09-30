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
	Layout        struct {
		RestOctetsLength int `json:"rest_octets_length"`
	} `json:"layout"`
}

type geranResult struct {
	bits                   int
	tail                   int
	encoded                []byte
	err                    error
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
		got := decodeGERANRecord(name, record.Type, wire)
		if why := compareGERANExpected(record, wire, got); why != "" {
			mismatches = append(mismatches, fmt.Sprintf("%d: %s", records, why))
		} else {
			matched++
		}
		if oldGERANMismatch(record) && (got.err == nil || strings.HasPrefix(record.Expected, "Rejected as truncated")) {
			changed++
		}
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("records=%d matched=%d mismatched=%d changed_vs_old=%d", records, matched, len(mismatches), changed)
	for _, why := range mismatches {
		t.Error(why)
	}
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
		}
	case strings.HasSuffix(name, "geran-ps.jsonl"):
		d, err := uecapability.DecodeGERANPS(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.encoded, out.err = uecapability.EncodeGERANPS(d.Value)
		}
	case strings.Contains(name, "classmark2"):
		d, err := uecapability.DecodeClassmark2ValuePart(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.spare3, out.spare4, out.spare5 = d.Value.Spare3, d.Value.Spare4, d.Value.Spare5
			out.encoded, out.err = uecapability.EncodeClassmark2ValuePart(d.Value)
		}
	case strings.HasSuffix(name, "classmark3.jsonl"):
		d, err := classmark.DecodeClassmark3ValuePart(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.encoded, out.err = classmark.EncodeClassmark3ValuePart(d.Value)
		}
	case strings.Contains(typ, "REJECT") || strings.Contains(typ, "IAR Rest"):
		d, err := restoctets.DecodeIARRestOctets(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.iar = &d.Value
			out.encoded, out.err = restoctets.EncodeIARRestOctets(d.Value)
		}
	default:
		d, err := restoctets.DecodeIARestOctets(wire)
		out.err = err
		if err == nil {
			out.bits, out.tail = d.BitsConsumed, d.Tail.BitLength
			out.ia = &d.Value
			out.encoded, out.err = restoctets.EncodeIARestOctets(d.Value)
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
