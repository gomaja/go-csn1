package registry

import (
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestCanonicalImportsUseFileLocalAliases(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   map[string]string
	}{
		{
			`package messages
import (
    "github.com/gomaja/go-csn1/ts44018/ies"
    packet_ies "github.com/gomaja/go-csn1/ts44060/ies"
)`,
			map[string]string{"ies": "ts44018/ies", "packet_ies": "ts44060/ies"},
		},
		{
			`package measurement
import "github.com/gomaja/go-csn1/ts44060/ies"`,
			map[string]string{"ies": "ts44060/ies"},
		},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), "generated.go", tc.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := canonicalImports(t, file); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("import aliases=%v, want %v", got, tc.want)
		}
	}
	call := delegatedCallee.FindStringSubmatch("packet_ies.EncodePacketTimingAdvanceIECanonicalAtLength")
	if len(call) != 3 || call[1] != "packet_ies" || call[2] != "EncodePacketTimingAdvanceIECanonicalAtLength" {
		t.Fatalf("delegation did not retain import alias: %v", call)
	}
}
