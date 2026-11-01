package registry_test

import (
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/registry"
	"github.com/gomaja/go-csn1/runtime"
)

// TS 44.060 V19.0.0 §§10.3.1–10.3.2 position ordinary content
// after the header and six-bit selector. The explicit extent excludes backing
// completion; the byte entry retains its documented completion policy.
func checkDescriptorBitParity(t *testing.T, d runtime.Descriptor, data []byte) {
	t.Helper()
	bits := len(data) * 8
	if d.SourceOrigin != 0 && d.Extent.Fixed && d.Extent.MaximumBits%8 != 0 && bits == (d.Extent.MaximumBits+7)/8*8 {
		bits = d.Extent.MaximumBits
	}
	byteValue, byteReserved, byteErr := d.DecodeWithReport(data)
	bitValue, bitReserved, bitReceiver, bitErr := d.DecodeBitsWithReport(data, bits)
	if !sameReaderError(byteErr, bitErr) {
		t.Fatalf("bit/byte diagnostics differ at %d bits: %v / %v", bits, byteErr, bitErr)
	}
	if !reflect.DeepEqual(byteReserved, bitReserved) {
		t.Fatalf("bit/byte reserved reports differ at %d bits", bits)
	}
	if byteErr == nil && !sameSourceDecoded(d, data, byteValue, bitValue) {
		t.Fatalf("bit/byte decoded result differs at %d bits", bits)
	}
	plain, plainErr := d.DecodeBits(data, bits)
	if !sameReaderError(plainErr, bitErr) {
		t.Fatalf("bit method diagnostics differ at %d bits", bits)
	}
	if plainErr == nil && !reflect.DeepEqual(plain, bitValue) {
		t.Fatalf("bit method values differ at %d bits", bits)
	}
	collector := runtime.NewReceiverCollector()
	reader := readerAtOrigin(t, d, data, bits, collector)
	from, fromErr := d.DecodeFrom(reader)
	if !sameReaderError(fromErr, bitErr) {
		t.Fatalf("bit/caller reader diagnostics differ at %d bits", bits)
	}
	if !reflect.DeepEqual(collector.TakeReport(), bitReserved) || !reflect.DeepEqual(collector.TakeReceiverReport(), bitReceiver) {
		t.Fatalf("bit/caller reader reports differ at %d bits", bits)
	}
	if fromErr == nil && !reflect.DeepEqual(from, bitValue) {
		t.Fatalf("bit/caller reader values differ at %d bits", bits)
	}
}

func TestDescriptorOriginsMatchSource(t *testing.T) {
	for _, d := range registry.Descriptors() {
		if d.SourceOrigin != registry.SourceOrigin(d) {
			t.Fatalf("%s/%s/%s: descriptor origin %d, recorded %d", d.Standard, d.Clause, d.Name, d.SourceOrigin, registry.SourceOrigin(d))
		}
	}
	// TS 24.008 V20.1.0 §10.5.5.12: the existing zero-origin network capability
	// descriptor must keep using an unshifted content reader.
	d, err := registry.Lookup("TS 24.008", "MS network capability value part")
	if err != nil {
		t.Fatal(err)
	}
	if d.SourceOrigin != 0 {
		t.Fatal("zero-origin IE moved")
	}
	for _, data := range [][]byte{nil, {0}, {0x50}, {0xff}} {
		checkDescriptorBitParity(t, d, data)
	}
}
