package uecapability

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	goruntime "runtime"
	"runtime/debug"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

func checkNestedReaderParity[T any](t *testing.T, raw []byte, decode func([]byte) (runtime.Decoded[T], error), from func(*runtime.Reader) (T, error), encode func(T) ([]byte, error)) {
	t.Helper()
	want, err := decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []int{0, 1, 8, 17} {
		t.Run(fmt.Sprint(offset), func(t *testing.T) {
			w := runtime.NewWriter()
			if err := w.WriteUint(0, offset); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteBitString(runtime.BitString{Bytes: raw, BitLength: len(raw) * 8}, len(raw)*8); err != nil {
				t.Fatal(err)
			}
			if err := w.WriteUint(0xa5, 8); err != nil {
				t.Fatal(err)
			}
			if err := w.WritePaddingTo((w.Position() + 7) / 8 * 8); err != nil {
				t.Fatal(err)
			}
			data, err := w.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			r := runtime.NewReader(data)
			if _, err := r.ReadUint(offset); err != nil {
				t.Fatal(err)
			}
			old, err := r.PushLimit(len(raw) * 8)
			if err != nil {
				t.Fatal(err)
			}
			got, err := from(r)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want.Value) {
				t.Fatal("nested typed value or received wire differs from byte entry")
			}
			if r.Position() != offset+want.BitsConsumed {
				t.Fatal("reader moved past its semantic boundary")
			}
			replay, err := encode(got)
			if err != nil || !bytes.Equal(replay, raw) {
				t.Fatalf("replay %x: %v", replay, err)
			}
			// Skip the value's preserved tail before leaving its enclosing extent.
			if _, err := r.ReadBitString(r.Remaining()); err != nil {
				t.Fatal(err)
			}
			if err := r.PopLimit(old); err != nil {
				t.Fatal(err)
			}
			if suffix, err := r.ReadUint(8); err != nil || suffix != 0xa5 {
				t.Fatalf("suffix %x: %v", suffix, err)
			}
		})
	}
}

func TestManualNestedWireMatchesByteEntry(t *testing.T) {
	// TS 24.008 V20.1.0 §§10.5.1.6, 10.5.1.7, 10.5.5.12a;
	// TS 36.331 V19.4.0 §6.3.6 UE-CapabilityRAT-ContainerList.
	t.Run("Classmark2", func(t *testing.T) {
		checkNestedReaderParity(t, []byte{0xfe, 0x51, 0xf6}, DecodeClassmark2ValuePart, DecodeClassmark2ValuePartFrom, EncodeClassmark2ValuePart)
	})
	for _, input := range []string{"3303a930a49be5", "330320000000"} {
		t.Run("GERANCS/"+input, func(t *testing.T) {
			data, err := hex.DecodeString(input)
			if err != nil {
				t.Fatal(err)
			}
			checkNestedReaderParity(t, data, DecodeGERANCS, DecodeGERANCSFrom, EncodeGERANCS)
		})
	}
	for _, input := range []string{"c088", "1000"} {
		t.Run("GERANPS/"+input, func(t *testing.T) {
			data, err := hex.DecodeString(input)
			if err != nil {
				t.Fatal(err)
			}
			checkNestedReaderParity(t, data, DecodeGERANPS, DecodeGERANPSFrom, EncodeGERANPS)
		})
	}
}

func TestManualNestedContainerErrorPath(t *testing.T) {
	raw := []byte{0x33, 3, 0x20, 0, 0, 0x30}
	_, baseErr := DecodeGERANCS(raw)
	var base *runtime.OperationError
	if !errors.As(baseErr, &base) {
		t.Fatal("missing invalid-branch control", baseErr)
	}
	r := runtime.NewReader(append([]byte{0}, raw...))
	if _, err := r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("Envelope"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(len(raw) * 8); err != nil {
		t.Fatal(err)
	}
	_, err := DecodeGERANCSFrom(r)
	var got *runtime.OperationError
	if !errors.As(err, &got) || got.Kind != base.Kind || got.Location.Path != "Envelope/"+base.Location.Path || got.Location.BitOffset != base.Location.BitOffset+8 {
		t.Fatalf("nested error %v; byte %v", err, baseErr)
	}
}

func TestGERANCSReaderTailStorageGrowsLinearly(t *testing.T) {
	// TS 36.331 V19.4.0 §6.3.6 supplies the containing octet extent.
	// TS 24.008 V20.1.0 §10.5.1.7 limits the known Classmark 3 value;
	// excess receiver octets stay in Tail without expanding typed fields.
	priorP := goruntime.GOMAXPROCS(1)
	defer goruntime.GOMAXPROCS(priorP)
	priorGC := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(priorGC)
	for _, n := range []int{1024, 4096, 16384, 65536} {
		data := make([]byte, n)
		copy(data, []byte{0x33, 3, 0x20, 0, 0, 0})
		decode := func() {
			r := runtime.NewReader(data)
			if _, err := r.PushLimit(len(data) * 8); err != nil {
				t.Fatal(err)
			}
			v, err := DecodeGERANCSFrom(r)
			if err != nil {
				t.Fatal(err)
			}
			if v.Wire.TransmittedBits != len(data)*8 || v.Classmark3.Wire.TransmittedBits != (len(data)-5)*8 {
				t.Fatal("lost containing extent")
			}
		}
		decode()
		var before, after goruntime.MemStats
		goruntime.ReadMemStats(&before)
		for range 8 {
			decode()
		}
		goruntime.ReadMemStats(&after)
		bytes := (after.TotalAlloc - before.TotalAlloc + 7) / 8
		if bytes > uint64(16*n+65536) {
			t.Fatalf("%d input octets allocate %d bytes", n, bytes)
		}
	}
}
