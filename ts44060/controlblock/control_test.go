package controlblock

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

type resolverFunc func(runtime.RLCMACContext, uint64) (Binding, error)

func (f resolverFunc) Resolve(c runtime.RLCMACContext, s uint64) (Binding, error) { return f(c, s) }

func TestDispatchContext(t *testing.T) {
	for _, mode := range []runtime.RLCMACMode{runtime.RLCMACOrdinary, runtime.RLCMACEC, runtime.RLCMACDBPSCH} {
		for _, direction := range []runtime.RLCMACDirection{runtime.RLCMACDownlink, runtime.RLCMACUplink} {
			for _, cc := range []uint8{1, 5} {
				dldc := false
				ctx := ordinaryContext()
				ctx.Mode = mode
				ctx.Direction = direction
				ctx.CoverageClass = &cc
				ctx.DLDC = &dldc
				ctx.FLO = &dldc
				ctx.PriorAllocation = &runtime.RLCMACAllocation{Timeslots: 0x81}
				ctx.CodingScheme = runtime.RLCMACCS1
				if mode == runtime.RLCMACEC {
					ctx.CodingScheme = runtime.RLCMACECCoding
				}
				if mode == runtime.RLCMACDBPSCH {
					ctx.Channel = runtime.RLCMACFACCH
				}
				width, err := SelectorWidth(ctx)
				if err != nil {
					t.Fatal(err)
				}
				wantWidth := 6
				if mode == runtime.RLCMACEC {
					wantWidth = 5
					if direction == runtime.RLCMACUplink && cc == 5 {
						wantWidth = 3
					}
				}
				if width != wantWidth {
					t.Fatalf("width %d want %d", width, wantWidth)
				}
				res := resolverFunc(func(got runtime.RLCMACContext, s uint64) (Binding, error) {
					if !reflect.DeepEqual(got, ctx) || s != 2 {
						t.Fatalf("context lost: %+v selector %d", got, s)
					}
					return Binding{Kind: KnownMessage, Ownership: ExternalSelector, DecodeFrom: func(r *runtime.Reader, c runtime.RLCMACContext) (any, error) {
						if r.Position() != width {
							t.Fatalf("position %d", r.Position())
						}
						return r.ReadUint(2)
					}}, nil
				})
				w := runtime.NewWriter()
				if err := w.WriteUint(2, width); err != nil {
					t.Fatal(err)
				}
				if err := w.WriteUint(3, 2); err != nil {
					t.Fatal(err)
				}
				bits, err := w.FinishBits(runtime.BitString{})
				if err != nil {
					t.Fatal(err)
				}
				r, err := runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
				if err != nil {
					t.Fatal(err)
				}
				msg, err := DecodeMessageFrom(r, ctx, res)
				if err != nil || msg.Body != uint64(3) || r.Remaining() != 0 {
					t.Fatal(msg, err)
				}
			}
		}
	}
	ctx := ordinaryContext()
	ctx.Mode = runtime.RLCMACEC
	ctx.Direction = runtime.RLCMACUplink
	if _, err := SelectorWidth(ctx); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatal("missing uplink coverage", err)
	}
	// Selector 000010 alone does not distinguish DBPSCH acknowledgements.
	ctx = ordinaryContext()
	ctx.Mode = runtime.RLCMACDBPSCH
	ctx.Channel = runtime.RLCMACFACCH
	ctx.Direction = runtime.RLCMACUplink
	res := resolverFunc(func(c runtime.RLCMACContext, _ uint64) (Binding, error) {
		if c.FLO == nil {
			return Binding{}, &runtime.ContextRequiredError{Standard: "TS 44.060", Clause: "11.2.6b; 11.2.6c", Name: "FLO"}
		}
		return Binding{}, nil
	})
	r, _ := runtime.NewReaderBits([]byte{8}, 6, nil)
	if _, err := DecodeMessageFrom(r, ctx, res); !errors.Is(err, runtime.ErrContextRequired) {
		t.Fatal(err)
	}
}

func TestUnknownDispatch(t *testing.T) {
	unknown := resolverFunc(func(runtime.RLCMACContext, uint64) (Binding, error) { return Binding{Kind: UnknownMessage}, nil })
	for _, mode := range []runtime.RLCMACMode{runtime.RLCMACOrdinary, runtime.RLCMACEC} {
		ctx := ordinaryContext()
		ctx.Mode = mode
		width, _ := SelectorWidth(ctx)
		w := runtime.NewWriter()
		_ = w.WriteUint(0, width)
		_ = w.WriteUint(2, 2)
		_ = w.WriteUint(5, 3)
		bits, _ := w.FinishBits(runtime.BitString{})
		r, _ := runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
		m, err := DecodeMessageFrom(r, ctx, unknown)
		v, ok := m.Body.(DefaultDownlink)
		if err != nil || !ok || v.Distribution != 2 || v.Remainder.BitLength != 3 || r.Remaining() != 0 {
			t.Fatal(m, err)
		}
		ctx.Direction = runtime.RLCMACUplink
		cc := uint8(1)
		ctx.CoverageClass = &cc
		r, _ = runtime.NewReaderBits(bits.Bytes, bits.BitLength, nil)
		if _, err := DecodeMessageFrom(r, ctx, unknown); !errors.Is(err, runtime.ErrUnsupported) {
			t.Fatal("unknown uplink accepted", err)
		}
	}
	known := resolverFunc(func(runtime.RLCMACContext, uint64) (Binding, error) { return Binding{Kind: KnownMessage}, nil })
	r, _ := runtime.NewReaderBits([]byte{0}, 8, nil)
	if _, err := DecodeMessageFrom(r, ordinaryContext(), known); !errors.Is(err, runtime.ErrUnsupported) {
		t.Fatal("known body degraded", err)
	}
	missing := resolverFunc(func(runtime.RLCMACContext, uint64) (Binding, error) { return Binding{}, nil })
	r, _ = runtime.NewReaderBits([]byte{0}, 8, nil)
	if _, err := DecodeMessageFrom(r, ordinaryContext(), missing); !errors.Is(err, runtime.ErrUnsupported) {
		t.Fatal("unclassified binding accepted as unknown", err)
	}
	// The catalogue may explicitly bind PSI7 to the source-default shape.
	psi7 := resolverFunc(func(_ runtime.RLCMACContext, s uint64) (Binding, error) {
		if s != 56 {
			t.Fatal(s)
		}
		return Binding{Kind: DefaultMessage}, nil
	})
	r, _ = runtime.NewReaderBits([]byte{0xe2}, 8, nil)
	m, err := DecodeMessageFrom(r, ordinaryContext(), psi7)
	if _, ok := m.Body.(DefaultDownlink); err != nil || !ok {
		t.Fatal(m, err)
	}
}

func TestSelectorOwnership(t *testing.T) {
	for _, ownership := range []SelectorOwnership{ExternalSelector, InternalSelector} {
		ctx := ordinaryContext()
		ctx.Mode = runtime.RLCMACDBPSCH
		ctx.Channel = runtime.RLCMACFACCH
		res := resolverFunc(func(runtime.RLCMACContext, uint64) (Binding, error) {
			return Binding{Kind: KnownMessage, Ownership: ownership, DecodeFrom: func(r *runtime.Reader, _ runtime.RLCMACContext) (any, error) {
				if ownership == InternalSelector {
					v, e := r.ReadUint(6)
					if e != nil || v != 18 {
						t.Fatalf("root selector %d %v", v, e)
					}
				}
				return r.ReadUint(2)
			}, EncodeTo: func(w *runtime.Writer, v any, _ runtime.RLCMACContext) error {
				if ownership == InternalSelector {
					if err := w.WriteUint(18, 6); err != nil {
						return err
					}
				}
				return w.WriteUint(v.(uint64), 2)
			}}, nil
		})
		r, _ := runtime.NewReaderBits([]byte{0x4b}, 8, nil)
		m, err := DecodeMessageFrom(r, ctx, res)
		if err != nil || m.Body != uint64(3) || r.Position() != 8 {
			t.Fatal(m, err)
		}
		w := runtime.NewWriter()
		if err := EncodeMessageTo(w, ctx, m, res); err != nil {
			t.Fatal(err)
		}
		bits, err := w.FinishBits(runtime.BitString{})
		if err != nil || bits.BitLength != 8 || bits.Bytes[0] != 0x4b {
			t.Fatal(bits, err)
		}
	}
}

func TestReceiverProcedureBoundary(t *testing.T) {
	sentinel := &runtime.DecodeError{Kind: runtime.InvalidBranch, Offset: 7, Path: "accepted-prefix", Detail: "syntax region"}
	res := resolverFunc(func(runtime.RLCMACContext, uint64) (Binding, error) {
		return Binding{Kind: KnownMessage, Ownership: ExternalSelector, DecodeFrom: func(*runtime.Reader, runtime.RLCMACContext) (any, error) { return nil, sentinel }}, nil
	})
	r, _ := runtime.NewReaderBits([]byte{0}, 8, nil)
	if _, err := DecodeMessageFrom(r, ordinaryContext(), res); err != sentinel {
		t.Fatal("diagnostic replaced", err)
	}
}

func FuzzMessage(f *testing.F) {
	f.Add([]byte{0x06, 0xff}, 16, uint8(1), uint8(1), uint8(1))
	f.Add([]byte{0xff}, 8, uint8(2), uint8(2), uint8(5))
	f.Fuzz(func(t *testing.T, data []byte, n int, mode, direction, cc uint8) {
		if len(data) > 4096 {
			t.Skip()
		}
		r, err := runtime.NewReaderBits(data, n, nil)
		if err != nil {
			return
		}
		ctx := ordinaryContext()
		ctx.Mode = runtime.RLCMACMode(mode)
		ctx.Direction = runtime.RLCMACDirection(direction)
		ctx.CoverageClass = &cc
		if ctx.Mode == runtime.RLCMACDBPSCH {
			ctx.Channel = runtime.RLCMACFACCH
		}
		type body struct{ Bits runtime.BitString }
		res := resolverFunc(func(_ runtime.RLCMACContext, s uint64) (Binding, error) {
			if s != 1 {
				return Binding{Kind: UnknownMessage}, nil
			}
			return Binding{Kind: KnownMessage, Ownership: ExternalSelector, DecodeFrom: func(r *runtime.Reader, _ runtime.RLCMACContext) (any, error) {
				b, e := r.ReadBitString(r.Remaining())
				return body{b}, e
			}, EncodeTo: func(w *runtime.Writer, v any, _ runtime.RLCMACContext) error {
				b := v.(body)
				return w.WriteBitString(b.Bits, b.Bits.BitLength)
			}}, nil
		})
		m, err := DecodeMessageFrom(r, ctx, res)
		if err != nil {
			return
		}
		w := runtime.NewWriter()
		if err = EncodeMessageTo(w, ctx, m, res); err != nil {
			t.Fatal(err)
		}
		out, err := w.FinishBits(runtime.BitString{})
		if err != nil {
			t.Fatal(err)
		}
		input, _ := runtime.NewReaderBits(data, n, nil)
		want, err := input.ReadBitString(n)
		if err != nil || out.BitLength != n || !bytes.Equal(out.Bytes, want.Bytes) {
			t.Fatal("message replay", out, want, err)
		}
	})
}
