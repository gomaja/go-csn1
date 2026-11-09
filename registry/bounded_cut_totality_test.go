package registry

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
	"github.com/gomaja/go-csn1/ts44060/ackmessages"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover2"
	"github.com/gomaja/go-csn1/ts44060/dtmhandover3"
	"github.com/gomaja/go-csn1/ts44060/ies"
	"github.com/gomaja/go-csn1/ts44060/pshandover2"
)

// TS 44.060 V19.0.0 §11.1.4.4 accepts whole components and ignores a
// component cut by its description bound. The §12.24 note supplies the zero
// default; canonical encoding must still preserve the typed value. A missing
// mandatory choice cannot be synthesized as a complete selected alternative.
func TestBoundedCutTotality(t *testing.T) {
	t.Run("12.24", func(t *testing.T) {
		sweepBoundedCuts(t, ies.DecodeExtensionInformationFrom, ies.EncodeExtensionInformationTo, func(v *ies.ExtensionInformation) *runtime.WireInfo { return &v.Wire }, nil)
	})
	t.Run("11.2.6a", func(t *testing.T) {
		sweepBoundedCuts(t, ackmessages.DecodeEPDANExtensionInfoFrom, ackmessages.EncodeEPDANExtensionInfoTo, func(v *ackmessages.EPDANExtensionInfo) *runtime.WireInfo { return &v.Wire }, nil)
	})
	t.Run("12.42a", func(t *testing.T) {
		sweepBoundedCuts(t, pshandover2.DecodeExtensionInformationFrom, pshandover2.EncodeExtensionInformationTo, func(v *pshandover2.ExtensionInformation) *runtime.WireInfo { return &v.Wire }, nil)
	})
	t.Run("12.48", func(t *testing.T) {
		sweepBoundedCuts(t, dtmhandover2.DecodeExtensionInformationFrom, dtmhandover2.EncodeExtensionInformationTo, func(v *dtmhandover2.ExtensionInformation) *runtime.WireInfo { return &v.Wire }, func(v dtmhandover2.ExtensionInformation) bool { return runtime.IsZero(v.Group) })
	})
	t.Run("12.64", func(t *testing.T) {
		sweepBoundedCuts(t, dtmhandover3.DecodeExtensionInformationFrom, dtmhandover3.EncodeExtensionInformationTo, func(v *dtmhandover3.ExtensionInformation) *runtime.WireInfo { return &v.Wire }, func(v dtmhandover3.ExtensionInformation) bool { return runtime.IsZero(v.Group) })
	})
}

func sweepBoundedCuts[T any](t *testing.T, decode func(*runtime.Reader) (T, error), encode func(*runtime.Writer, T) error, wire func(*T) *runtime.WireInfo, unselected func(T) bool) {
	t.Helper()
	var decoded, replayFailures, canonicalFailures, refusals, backFailures, semanticFailures int
	var first string
	failure := func(n, v int, operation string, err error) {
		if first == "" {
			first = fmt.Sprintf("%s at length %d, synthetic ordinal %d: %v", operation, n, v, err)
		}
	}
	read := func(data runtime.BitString) (T, *runtime.Reader, error) {
		r, err := runtime.NewReaderBits(data.Bytes, data.BitLength, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.PushLimit(data.BitLength); err != nil {
			t.Fatal(err)
		}
		v, err := decode(r)
		return v, r, err
	}
	write := func(v T, n int, canonical bool) (runtime.BitString, error) {
		w := runtime.NewWriter()
		if canonical {
			w.WithWire(*wire(&v))
		}
		if _, err := w.PushLimit(n); err != nil {
			t.Fatal(err)
		}
		if err := encode(w, v); err != nil {
			return runtime.BitString{}, err
		}
		return w.FinishBits(runtime.BitString{})
	}
	for n := 1; n <= 14; n++ {
		for v := 0; v < 1<<n; v++ {
			data := runtime.BitString{Bytes: make([]byte, (n+7)/8), BitLength: n}
			for i := 0; i < n; i++ {
				if v>>(n-1-i)&1 != 0 {
					data.Bytes[i/8] |= 0x80 >> uint(i%8)
				}
			}
			value, r, err := read(data)
			if err != nil {
				continue
			}
			decoded++
			*wire(&value), err = runtime.SealBits(value, data, r.Position(), r.Tail(), r.Wire())
			if err != nil {
				t.Fatal(err)
			}
			replay, err := write(value, n, false)
			if err != nil || replay.BitLength != n || !bytes.Equal(replay.Bytes, data.Bytes) {
				replayFailures++
				failure(n, v, "replay", err)
			}
			out, err := write(runtime.Canonical(value), n, true)
			if err != nil {
				var bound *runtime.BoundError
				if unselected != nil && unselected(value) && errors.As(err, &bound) && bound.Kind == runtime.LengthBound {
					refusals++
					continue
				}
				canonicalFailures++
				failure(n, v, "canonical encode", err)
				continue
			}
			again, _, err := read(out)
			if err != nil || out.BitLength != n {
				backFailures++
				failure(n, v, "canonical decode", err)
			} else if !runtime.SemanticallyEqual(value, again) {
				semanticFailures++
				failure(n, v, "canonical semantics", nil)
			}
		}
	}
	t.Logf("decoded=%d replay_failures=%d canonical_failures=%d typed_refusals=%d decode_back_failures=%d semantic_failures=%d", decoded, replayFailures, canonicalFailures, refusals, backFailures, semanticFailures)
	if decoded == 0 || replayFailures+canonicalFailures+backFailures+semanticFailures != 0 {
		t.Fatal(first)
	}
}
