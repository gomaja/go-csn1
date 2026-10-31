package controlblock

import (
	"fmt"

	"github.com/gomaja/go-csn1/runtime"
)

// SelectorOwnership is authenticated root shape, not an inference from a title.
// TS 44.060 V19.0.0 §11.2.34 Physical Information prints its own MESSAGE_TYPE;
// the content roots of §11.2.0.1 generally require an external selector.
type SelectorOwnership uint8

const (
	ExternalSelector SelectorOwnership = iota + 1
	InternalSelector
)

// ResolutionKind distinguishes a source-unknown selector from a known body
// whose codec is unavailable. Only the former may use unknown-message syntax.
type ResolutionKind uint8

const (
	UnknownMessage ResolutionKind = iota + 1
	KnownMessage
	DefaultMessage // An explicitly source-selected default-content root.
)

// Binding is supplied by an authenticated catalogue. Callbacks use the original
// reader/writer and context, retaining L/H phase. An internal-selector encoder
// must validate the selector in its typed root. A known unavailable body leaves
// its callbacks nil and is refused; it never becomes opaque bytes.
type Binding struct {
	Kind       ResolutionKind
	Ownership  SelectorOwnership
	DecodeFrom func(*runtime.Reader, runtime.RLCMACContext) (any, error)
	EncodeTo   func(*runtime.Writer, any, runtime.RLCMACContext) error
}

// Resolver selects by the complete context. It must classify every source-known
// selector as KnownMessage or DefaultMessage, even before its codec is present.
// No generated table or leaf codec is imported by this package.
type Resolver interface {
	Resolve(runtime.RLCMACContext, uint64) (Binding, error)
}

// Message contains one selected typed body. Selector excludes physical headers.
type Message struct {
	Selector uint64
	Body     any
}

// DefaultDownlink is §11.2.0.1's PAGE_MODE (ordinary) or
// USED_DL_COVERAGE_CLASS (EC), followed by the exact unspecified remainder.
type DefaultDownlink struct {
	Distribution uint64
	Remainder    runtime.BitString
}

// SelectorWidth applies the distinct maps of TS 44.060 V19.0.0 §§11.1.1.3,
// 11.2.0.1–2. EC uplink uses five bits for CC1..4 and three for CC5.
func SelectorWidth(ctx runtime.RLCMACContext) (int, error) {
	if err := ctx.Validate(); err != nil {
		return 0, err
	}
	if ctx.Mode == runtime.RLCMACEC {
		if ctx.Direction == runtime.RLCMACUplink {
			if ctx.CoverageClass == nil {
				return 0, &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "11.2.0.2", Name: "uplink coverage class"}
			}
			if *ctx.CoverageClass == 5 {
				return 3, nil
			}
		}
		return 5, nil
	}
	return 6, nil
}

func resolve(ctx runtime.RLCMACContext, selector uint64, resolver Resolver) (Binding, error) {
	if resolver == nil {
		return Binding{}, &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "11.2.0", Name: "source-backed message resolver"}
	}
	return resolver.Resolve(ctx, selector)
}

func unsupportedMessage(ctx runtime.RLCMACContext, selector uint64) error {
	return fmt.Errorf("TS 44.060 V19.0.0 §11.2.0 mode %d direction %d selector %d: %w", ctx.Mode, ctx.Direction, selector, runtime.ErrUnsupported)
}

func defaultBinding(ctx runtime.RLCMACContext, b Binding) bool {
	return (b.Kind == UnknownMessage || b.Kind == DefaultMessage) && ctx.Direction == runtime.RLCMACDownlink && ctx.Mode != runtime.RLCMACDBPSCH
}

// DecodeMessageFrom peeks at the selector, resolves it with all caller facts,
// then consumes it exactly once according to root ownership. A containing bit
// extent is required; unknown downlink remainder expands to that exact end.
// Typed partial values and original syntax/region errors are returned directly.
func DecodeMessageFrom(r *runtime.Reader, ctx runtime.RLCMACContext, resolver Resolver) (Message, error) {
	var m Message
	width, err := SelectorWidth(ctx)
	if err != nil {
		return m, err
	}
	if err := r.Check(); err != nil {
		return m, err
	}
	if r.BoundedRemaining() < 0 {
		return m, &runtime.ContextRequiredError{Standard: "TS 44.060 V19.0.0", Clause: "10.0b.2.1", Name: "message bit extent"}
	}
	peek := r.Fork()
	selector, err := peek.ReadUint(width)
	if err != nil {
		return m, err
	}
	b, err := resolve(ctx, selector, resolver)
	if err != nil {
		return m, err
	}
	m.Selector = selector
	if defaultBinding(ctx, b) {
		if _, err = r.ReadUint(width); err != nil {
			return m, err
		}
		distribution, e := r.ReadUint(2)
		if e != nil {
			return m, e
		}
		rest, e := r.ReadBitString(r.Remaining())
		m.Body = DefaultDownlink{Distribution: distribution, Remainder: rest}
		return m, e
	}
	if b.Kind != KnownMessage || b.DecodeFrom == nil {
		return m, unsupportedMessage(ctx, selector)
	}
	if b.Ownership != ExternalSelector && b.Ownership != InternalSelector {
		return m, invalidHeader("resolver omitted selector ownership (§11.2.0)")
	}
	if b.Ownership == ExternalSelector {
		if _, err = r.ReadUint(width); err != nil {
			return m, err
		}
	}
	m.Body, err = b.DecodeFrom(r, ctx)
	return m, err
}

// EncodeMessageTo writes one selector and the typed body using the resolver's
// authenticated root shape. It performs no application protocol-state action.
func EncodeMessageTo(w *runtime.Writer, ctx runtime.RLCMACContext, m Message, resolver Resolver) error {
	width, err := SelectorWidth(ctx)
	if err != nil {
		return err
	}
	if m.Selector >= uint64(1)<<width {
		return invalidHeader("selector exceeds mode-specific width (§11.2.0)")
	}
	b, err := resolve(ctx, m.Selector, resolver)
	if err != nil {
		return err
	}
	if defaultBinding(ctx, b) {
		v, ok := m.Body.(DefaultDownlink)
		if !ok {
			return invalidHeader("default downlink body has incorrect type (§11.2.0.1)")
		}
		if err := w.WriteUint(m.Selector, width); err != nil {
			return err
		}
		if err := w.WriteUint(v.Distribution, 2); err != nil {
			return err
		}
		return w.WriteBitString(v.Remainder, v.Remainder.BitLength)
	}
	if b.Kind != KnownMessage || b.EncodeTo == nil {
		return unsupportedMessage(ctx, m.Selector)
	}
	if b.Ownership != ExternalSelector && b.Ownership != InternalSelector {
		return invalidHeader("resolver omitted selector ownership (§11.2.0)")
	}
	if b.Ownership == ExternalSelector {
		if err := w.WriteUint(m.Selector, width); err != nil {
			return err
		}
	}
	return b.EncodeTo(w, m.Body, ctx)
}
