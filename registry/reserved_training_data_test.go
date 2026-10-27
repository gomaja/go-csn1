package registry_test

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

// Captured bytes remain outside the repository. Only aggregate binding counts
// are logged; no record bytes or values derived from a capture are fixtures.
func TestReportsGERANTrainingData(t *testing.T) {
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
	counts, weighted := map[string]int{}, map[string]int{}
	total, withEvents, events := 0, 0, 0
	for _, name := range []string{"ue-capability/geran-cs.jsonl", "ue-capability/geran-ps.jsonl", "a-interface/classmark2.jsonl", "a-interface/classmark2-vendor-altered.jsonl", "a-interface/classmark3.jsonl", "um-ccch-bcch/rest-octets.jsonl", "um-ccch-bcch/immediate-assignment-messages.jsonl", "um-sacch/enhanced-measurement-report-synthetic.jsonl"} {
		file, err := os.Open(filepath.Join(root, "geran", filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		scan := bufio.NewScanner(file)
		row := 0
		for scan.Scan() {
			row++
			total++
			var record geranRecord
			if err := json.Unmarshal(scan.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			text := record.Hex
			if record.RestOctets != "" {
				text = record.RestOctets
			}
			data, err := hex.DecodeString(text)
			if err != nil {
				t.Fatalf("%s row %d: invalid hex", name, row)
			}
			standard, definition := "TS 24.008", "Mobile Station Classmark 2 value part"
			switch {
			case strings.Contains(name, "geran-cs"):
				standard, definition = "TS 36.331", "geran-cs"
			case strings.Contains(name, "geran-ps"):
				standard, definition = "TS 36.331", "geran-ps"
			case strings.Contains(name, "classmark3"):
				definition = "Classmark 3 Value part"
			case strings.Contains(name, "enhanced-measurement"):
				standard, definition = "TS 44.018", "Enhanced Measurement report"
			case strings.Contains(name, "um-ccch"):
				standard, definition = "TS 44.018", "IA Rest Octets"
				if strings.Contains(record.Type, "REJECT") || strings.Contains(record.Type, "IAR Rest") {
					definition = "IAR Rest Octets"
				}
			}
			d, err := registry.Lookup(standard, definition)
			if err != nil {
				t.Fatal(err)
			}
			before, be := d.Decode(data)
			after, report, ae := d.DecodeWithReport(data)
			if report.Dropped != 0 {
				t.Fatalf("%s row %d: unexpectedly incomplete report (%d dropped)", name, row, report.Dropped)
			}
			if observeDecode(d, before, be) != observeDecode(d, after, ae) {
				t.Fatalf("%s row %d: report changes decode/replay/canonical outcome", name, row)
			}
			if be == nil && !runtime.SemanticallyEqual(before, after) {
				t.Fatalf("%s row %d: report changes typed value", name, row)
			}
			var bde, ade *runtime.DecodeError
			if errors.As(be, &bde) && errors.As(ae, &ade) && (bde.Kind != ade.Kind || bde.Offset != ade.Offset) {
				t.Fatalf("%s row %d: report changes error kind/offset", name, row)
			}
			if len(report.Observations) > 0 {
				withEvents++
			}
			for _, event := range report.Observations {
				if event.Location.BitOffset < 0 || event.Location.Width <= 0 || event.Location.BitOffset+event.Location.Width > len(data)*8 || event.Location.Path == "" || event.Location.Source.Standard == "" {
					t.Fatalf("%s row %d: incomplete observation metadata", name, row)
				}
				leaf := event.Location.Path[strings.LastIndex(event.Location.Path, "/")+1:]
				key := event.Location.Source.Standard + " V" + event.Location.Source.Version + " §" + event.Location.Source.Clause + " table " + event.Location.Source.Table + " / " + leaf
				counts[key]++
				weighted[key] += record.Count
				events++
			}
		}
		if err := scan.Err(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if total != 1812 {
		t.Fatalf("records=%d, want 1812", total)
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	t.Logf("records=%d records_with_observations=%d observations=%d bindings=%d", total, withEvents, events, len(keys))
	for _, key := range keys {
		t.Logf("%s: record_events=%d weighted_events=%d", key, counts[key], weighted[key])
	}
}
