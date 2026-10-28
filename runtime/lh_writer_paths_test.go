package runtime

import (
	"bytes"
	"fmt"
	"testing"
)

// Check the transmitted shortcut against the checked coordinate at every
// octet offset, including transitions across canonical and replay boundaries.
func TestLHLiteralWriterPaths(t *testing.T) {
	for _, sealed := range []bool{false, true} {
		for _, zero := range []bool{false, true} {
			for start := 0; start < 8; start++ {
				for boundary := 0; boundary < 10; boundary++ {
					for _, bounded := range []bool{false, true} {
						makeWriter := func() *Writer {
							w := NewWriter()
							if err := w.WriteZeros(start); err != nil {
								t.Fatal(err)
							}
							w.allowZero = zero
							if sealed {
								w.wire = WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: boundary, Count: 3}}}
							} else {
								w.wire = WireInfo{canonical: true, targetSet: !bounded, targetBits: boundary}
								w.bounded = bounded
								w.limit = boundary
							}
							return w
						}
						got, want := makeWriter(), makeWriter()
						pattern := "LLH010HLL"
						gotErr := got.WriteLiteral(pattern)
						var wantErr error
						for _, symbol := range pattern {
							bit := literalBit(symbol, want.bits)
							if symbol == 'L' || symbol == 'H' {
								bit, wantErr = want.LHBit(symbol)
								if wantErr != nil {
									break
								}
							}
							if wantErr = want.put(bit); wantErr != nil {
								wantErr = want.operationFailure(wantErr, 0, InvalidValue)
								break
							}
						}
						if fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || !bytes.Equal(got.bytes, want.bytes) || got.bits != want.bits || got.virtual != want.virtual || got.implicitSpan != want.implicitSpan || got.lh != want.lh {
							t.Fatalf("sealed=%t zero=%t start=%d boundary=%d bounded=%t: got %v, want %v", sealed, zero, start, boundary, bounded, gotErr, wantErr)
						}
					}
				}
			}
		}
	}
}
