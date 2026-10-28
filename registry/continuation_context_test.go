package registry_test

import (
	"errors"
	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
	"strings"
	"testing"
)

// TS 44.018 V19.0.0 §10.5.2.37h table 10.5.2.37h.2: code 31
// consumes the remaining whole octets of the containing SI value.
func TestContinuationCountRequiresLocatedContext(t *testing.T) {
	d, err := registry.LookupClause("TS 44.018", "10.5.2.37h", "Non-GSM Message struct")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Extent.NeedsContext {
		t.Error("conditional reader context omitted from descriptor")
	}
	_, err = d.Decode([]byte{0x3f, 0xab})
	check := func(err error) {
		t.Helper()
		var op *runtime.OperationError
		if !strings.Contains(err.Error(), "continuation count requires enclosing SI value") || !errors.Is(err, runtime.ErrContextRequired) || !errors.As(err, &op) || op.Operation != "decode" || op.Location.BitOffset != -1 || op.Location.Width != 0 || op.Location.Path == "" || op.Location.Source.Standard != "TS 44.018" || op.Location.Source.Clause != "10.5.2.37h" {
			t.Fatalf("missing located context: %v", err)
		}
	}
	check(err)
	for _, origin := range []int{0, 3, 11} {
		w := runtime.NewWriter()
		if err := w.WriteUint(0, origin); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0x3f, 8); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0xab, 8); err != nil {
			t.Fatal(err)
		}
		if err := w.PadMinimum((origin+16+7)/8*8, false); err != nil {
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
		_, err = d.DecodeFrom(r)
		check(err)
		r = runtime.NewReader(wire)
		if _, err := r.ReadUint(origin); err != nil {
			t.Fatal(err)
		}
		if _, err := r.PushLimit(16); err != nil {
			t.Fatal(err)
		}
		v, err := d.DecodeFrom(r)
		if err != nil || r.Position() != origin+16 {
			t.Fatalf("bounded continuation %T: %v", v, err)
		}
	}
	for _, count := range []byte{0, 1, 18} {
		data := make([]byte, int(count)+1)
		data[0] = 0x20 | count
		if _, err := d.Decode(data); err != nil {
			t.Fatalf("literal count %d requires context: %v", count, err)
		}
	}
	for _, name := range []string{"SI 18 Rest Octets", "SI 20 Rest Octets"} {
		d, err := registry.Lookup("TS 44.018", name)
		if err != nil {
			t.Fatal(err)
		}
		// The fixed grammar supplies continuation-count context, but the
		// descriptor's source size check still needs the received IE extent.
		if !d.Extent.NeedsContext {
			t.Fatalf("source size check for %s omits reader context", name)
		}
	}
}
