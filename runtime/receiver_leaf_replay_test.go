package runtime_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

type regionLeafVector struct {
	Standard string `json:"standard"`
	Clause   string `json:"clause"`
	Name     string `json:"name"`
	Hex      string `json:"hex"`
	Received string `json:"received"`
	Bits     int    `json:"bits,omitempty"`
}

func publishedRegions(v reflect.Value) []*runtime.ErrorRegion {
	if !v.IsValid() || !v.CanInterface() {
		return nil
	}
	if v.Type() == reflect.TypeFor[runtime.ErrorRegion]() && v.CanAddr() {
		return []*runtime.ErrorRegion{v.Addr().Interface().(*runtime.ErrorRegion)}
	}
	var out []*runtime.ErrorRegion
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() {
			return publishedRegions(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				out = append(out, publishedRegions(v.Field(i))...)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			out = append(out, publishedRegions(v.Index(i))...)
		}
	}
	return out
}

// The vectors are synthetic mutations of the clause fixtures. Each names the
// selected leaf, so replay alone cannot hide a change in the decoded arm.
// TS 44.060 V19.0.0 §11.1.3.2 applies the error label at its received position.
func TestPublishedReceiverLeafReplay(t *testing.T) {
	data, err := os.ReadFile("testdata/receiver_leaf_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []regionLeafVector
	if err = json.Unmarshal(data, &vectors); err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 20 {
		t.Fatalf("got %d vectors, want 20 reachable leaves", len(vectors))
	}
	for _, tc := range vectors {
		t.Run(tc.Received[strings.LastIndexByte(tc.Received, '/')+1:], func(t *testing.T) {
			d, err := registry.LookupClause(tc.Standard, tc.Clause, tc.Name)
			if err != nil {
				t.Fatal(err)
			}
			input, err := hex.DecodeString(tc.Hex)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := d.Decode(input)
			if tc.Bits > 0 {
				decoded, err = d.DecodeBits(input, tc.Bits)
			}
			if err != nil {
				t.Fatal(err)
			}
			value := reflect.New(reflect.TypeOf(decoded))
			value.Elem().Set(reflect.ValueOf(decoded))
			var region *runtime.ErrorRegion
			for _, candidate := range publishedRegions(value.Elem()) {
				if candidate.Location.Path == tc.Received {
					region = candidate
				}
			}
			if region == nil {
				t.Fatal("input did not select the specified region leaf")
			}
			out, err := d.Encode(value.Elem().Interface())
			if err != nil || !bytes.Equal(out, input) {
				t.Fatalf("published replay: %v", err)
			}
			parent := tc.Received[:strings.LastIndexByte(tc.Received, '/')]
			writer := runtime.NewWriter()
			for _, scope := range strings.Split(parent, "/") {
				if err = writer.Enter(scope); err != nil {
					t.Fatal(err)
				}
			}
			if err = writer.WriteErrorRegion(*region); err != nil {
				t.Fatalf("parent-path replay: %v", err)
			}
			if err = writer.Enter("Moved"); err != nil {
				t.Fatal(err)
			}
			if err = writer.WriteErrorRegion(*region); err == nil {
				t.Error("moved region accepted")
			}
			// Receive exactly the same owned bits at a different position, then move
			// that receipt into the published value. Label, width and bits still fit.
			reader := runtime.NewReader(region.Bits.Bytes)
			for _, scope := range strings.Split("Moved/"+tc.Received, "/") {
				if err = reader.Enter(scope); err != nil {
					t.Fatal(err)
				}
			}
			moved, err := reader.ReadErrorRegion(region.Label, region.Bits.BitLength, region.Location.Source)
			if err != nil {
				t.Fatal(err)
			}
			*region = moved
			if _, err = d.Encode(value.Elem().Interface()); err == nil || !strings.Contains(err.Error(), "receiver error region was not received or was edited") {
				t.Fatalf("published value accepted moved receipt or wrong refusal: %v", err)
			}
		})
	}
}

// TS 44.060 V19.0.0 §11.2.5b, paragraph 14299: missing trailing
// components are zero-filled. The channel and response-time definitions use
// only unrestricted bit fields and exhaustive 0/1 choices. Consequently the
// known body cannot produce a wire syntax failure; its printed fallback is
// unreachable. Resource and context failures cannot select an error region.
func TestDBPSCHZeroFilledBodyKeepsKnownArm(t *testing.T) {
	d, err := registry.LookupClause("TS 44.060", "11.2.5b", "Packet DBPSCH Assignment message content")
	if err != nil {
		t.Fatal(err)
	}
	for bits := 11; bits <= 139; bits++ {
		for _, pattern := range []byte{0, 0xff, 0xa5, 0x5a} {
			input := make([]byte, (bits+7)/8)
			for bit := 11; bit < bits; bit++ {
				if pattern&(1<<uint(7-bit%8)) != 0 {
					input[bit/8] |= 1 << uint(7-bit%8)
				}
			}
			decoded, err := d.DecodeBits(input, bits)
			if err != nil {
				t.Fatalf("extent %d pattern %x: %v", bits, pattern, err)
			}
			value := reflect.New(reflect.TypeOf(decoded))
			value.Elem().Set(reflect.ValueOf(decoded))
			for _, region := range publishedRegions(value.Elem()) {
				if strings.HasSuffix(region.Location.Path, "CHANNELDESCRIPTIONGroupFallbackIgnoredContent") {
					t.Fatal("total zero-filled body selected an error region")
				}
			}
			out, err := d.Encode(decoded)
			if err != nil || !bytes.Equal(out, input) {
				t.Fatalf("extent %d pattern %x replay: %v", bits, pattern, err)
			}
		}
	}
}
