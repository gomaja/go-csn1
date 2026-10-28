package runtime

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func inferPrefix(t *testing.T, r *Reader) {
	t.Helper()
	r.SetZeroExtension(true)
	old, err := r.PushLimit(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
}

func TestLHTransmittedAfterInferred(t *testing.T) {
	for _, b := range []byte{0x2b, 0x56} {
		r := NewReader([]byte{b})
		inferPrefix(t, r)
		numeric := r.Fork()
		if _, err := numeric.ReadUint(2); err != nil {
			t.Fatal(err)
		}
		if bit, err := numeric.LHBit('L'); err != nil || bit != 1 {
			t.Fatalf("numeric transmitted L at position 2: %d %v", bit, err)
		}
		match, matchErr := r.LookaheadLiteral("LLLLLLLL")
		if matchErr != nil {
			t.Fatal(matchErr)
		}
		err := r.Expect("LLLLLLLL")
		if match != (b == 0x2b) || (err == nil) != (b == 0x2b) {
			t.Errorf("%x: match=%t Expect=%v", b, match, err)
		}
	}
	for _, mode := range []string{"literal", "complete", "received", "padding", "fixed-padding"} {
		t.Run(mode, func(t *testing.T) {
			wire := Seal(struct{}{}, []byte{0x2b}, 8, BitString{}, WireInfo{ImplicitZeros: 1, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}})
			w := NewWriter()
			w.WithWire(wire)
			if err := w.WriteUint(0, 1); err != nil {
				t.Fatal(err)
			}
			var err error
			switch mode {
			case "literal":
				err = w.WriteLiteral("LLLLLLLL")
			case "complete":
				err = w.WithCompleteWireValue(WireInfo{}, 8, func() error { return w.WriteLiteral("LLLLLLLL") })
			case "received":
				err = w.WithWireValue(Seal(struct{}{}, []byte{0x2b}, 8, BitString{}, WireInfo{}), func() error { return w.WriteLiteral("LLLLLLLL") })
			case "padding":
				w.wire = WireInfo{}
				if err = w.WriteLiteral("L"); err == nil {
					err = w.WritePadding()
				}
			case "fixed-padding":
				w.wire = WireInfo{}
				err = w.WritePaddingTo(8)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(w.bytes, []byte{0x2b}) {
				t.Fatalf("wrote %x", w.bytes)
			}
			if mode == "literal" || mode == "complete" || mode == "received" {
				if _, err := w.Bytes(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLHLaterInferredRunRefused(t *testing.T) {
	r := NewReader([]byte{0})
	inferPrefix(t, r)
	if _, err := r.ReadUint(2); err != nil {
		t.Fatal(err)
	}
	old, err := r.PushLimit(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("later"); err != nil {
		t.Fatal(err)
	}
	err = r.Expect("L")
	var de *DecodeError
	if !errors.As(err, &de) || de.Kind != InvalidSchema || de.Offset != 2 || de.Path != "later" || !strings.Contains(de.Detail, "no source assigns") {
		t.Fatalf("later inferred literal: %v", err)
	}
	r.Leave()
	if err := r.PopLimit(old); err != nil {
		t.Fatal(err)
	}
}

func TestLHFirstInferredRunAtEnd(t *testing.T) {
	r := NewReader([]byte{0})
	if _, err := r.ReadUint(8); err != nil {
		t.Fatal(err)
	}
	r.SetZeroExtension(true)
	if err := r.Expect("LLH"); err != nil {
		t.Fatal(err)
	}
	wire := Seal(struct{}{}, []byte{0}, 8, BitString{}, r.Wire())
	w := NewWriter()
	w.WithWire(wire)
	if err := w.WriteUint(0, 8); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLiteral("LLH"); err != nil {
		t.Fatal(err)
	}
	if got, err := w.Bytes(); err != nil || !bytes.Equal(got, []byte{0}) {
		t.Fatalf("replay=%x %v", got, err)
	}
}

func TestLHLookaheadAndWriterRefuseLaterInferredRun(t *testing.T) {
	r := NewReader([]byte{0})
	inferPrefix(t, r)
	if _, err := r.ReadUint(2); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(0); err != nil {
		t.Fatal(err)
	}
	if err := r.Enter("second"); err != nil {
		t.Fatal(err)
	}
	before := r.VirtualBits()
	for _, pattern := range []string{"L", "H"} {
		matched, err := r.LookaheadLiteral(pattern)
		if matched || err != nil {
			t.Fatalf("%s: matched=%t error=%v", pattern, matched, err)
		}
	}
	if r.VirtualBits() != before {
		t.Fatal("lookahead consumed inferred bits")
	}
	// Numeric zero extension in later runs remains supported. Only L/H has
	// no assigned position after transmission resumes.
	if _, err := r.ReadUint(1); err != nil {
		t.Fatal(err)
	}
	if r.VirtualBits() != 2 {
		t.Fatal("later numeric run was refused")
	}
	wire := WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}, {At: 3, Count: 1}}}
	w := NewWriter()
	w.WithWire(wire)
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 2); err != nil {
		t.Fatal(err)
	}
	if err := w.Enter("second"); err != nil {
		t.Fatal(err)
	}
	var oe *OperationError
	if err := w.WriteLiteral("L"); !errors.As(err, &oe) || oe.Kind != InvalidSchema || oe.Location.BitOffset != 2 || oe.Location.Path != "second" || !strings.Contains(err.Error(), "no source assigns") {
		t.Fatalf("writer refusal: %v", err)
	}
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLiteral("LLLLLL"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w.bytes, []byte{0x2b}) {
		t.Fatalf("after two inferred runs wrote %x", w.bytes)
	}
}

func TestLHFirstInferredLookaheadAndNumericCoordinate(t *testing.T) {
	r := NewReader(nil)
	r.SetZeroExtension(true)
	for _, pattern := range []string{"LLH", "LLHLHLHH"} {
		matched, err := r.LookaheadLiteral(pattern)
		if err != nil || matched || r.VirtualBits() != 0 {
			t.Fatalf("first lookahead %s: %t %v", pattern, matched, err)
		}
	}
	if err := r.ExpectZeros(2); err != nil {
		t.Fatal(err)
	}
	if value, err := r.LHBit('L'); err != nil || value != 1 {
		t.Fatalf("numeric L after two inferred zeros: %d %v", value, err)
	}
	if err := r.Expect("H"); err != nil {
		t.Fatal(err)
	}
	wire := Seal(struct{}{}, nil, 0, BitString{}, r.Wire())
	w := NewWriter()
	w.WithWire(wire)
	if err := w.WriteZeros(2); err != nil {
		t.Fatal(err)
	}
	if value, err := w.LHBit('L'); err != nil || value != 1 {
		t.Fatalf("numeric writer L: %d %v", value, err)
	}
	if err := w.WriteLiteral("H"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Bytes(); err != nil {
		t.Fatal(err)
	}
}

func TestLHReceivedScopesDoNotResetInferredCoordinate(t *testing.T) {
	w := NewWriter()
	for _, symbol := range []string{"L", "L", "H"} {
		wire := Seal(struct{}{}, nil, 0, BitString{}, WireInfo{ImplicitZeros: 1, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}})
		if err := w.WithWireValue(wire, func() error { return w.WriteLiteral(symbol) }); err != nil {
			t.Fatalf("%s: %v", symbol, err)
		}
	}
	if w.Position() != 0 || w.virtual != 3 {
		t.Fatalf("scope inferred state %d+%d", w.Position(), w.virtual)
	}
	if err := w.WithCompleteWireValue(WireInfo{}, 8, func() error { return w.WriteLiteral("LLLLLLLL") }); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w.bytes, []byte{0x2b}) {
		t.Fatalf("complete scope after inferred run: %x", w.bytes)
	}
	wire := Seal(struct{}{}, nil, 0, BitString{}, WireInfo{ImplicitZeros: 1, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}})
	var oe *OperationError
	if err := w.WithWireValue(wire, func() error { return w.WriteLiteral("L") }); !errors.As(err, &oe) || oe.Kind != InvalidSchema || oe.Location.BitOffset != 8 {
		t.Fatalf("received scope bypassed refusal: %v", err)
	}
}

func TestLHIgnoredFallbackRetainsRefusal(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}}})
	w.SetZeroExtension(true)
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 2); err != nil {
		t.Fatal(err)
	}
	mark := w.MarkFallback()
	err := w.CheckIgnoredFallback(mark, false, func(r *Reader) error {
		if err := r.Enter("known"); err != nil {
			return err
		}
		return r.Expect("L")
	})
	var de *DecodeError
	if !errors.As(err, &de) || de.Kind != InvalidSchema || de.Offset != 2 || de.Path != "known" || !strings.Contains(de.Detail, "no source assigns") {
		t.Fatalf("fallback hid refusal: %v", err)
	}
}

func TestLHCanonicalInferredCoordinates(t *testing.T) {
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 0})
	w.SetZeroExtension(true)
	if err := w.WriteZeros(2); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLiteral("H"); err != nil {
		t.Fatal(err)
	}
	if got, err := w.Bytes(); err != nil || len(got) != 0 {
		t.Fatalf("canonical terminal %x %v", got, err)
	}
	w = NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 8})
	w.SetZeroExtension(true)
	old, err := w.PushLimit(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.PopLimit(old); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := w.PushLimit(0); err != nil {
		t.Fatal(err)
	}
	var oe *OperationError
	if err := w.WriteLiteral("L"); !errors.As(err, &oe) || oe.Kind != InvalidSchema || oe.Location.BitOffset != 2 {
		t.Fatalf("canonical later run %v", err)
	}
}

func TestLHCanonicalRefusalSurvivesDeferredLocation(t *testing.T) {
	for _, raw := range []bool{false, true} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 8, rawErrors: raw})
		w.SetZeroExtension(true)
		old, err := w.PushLimit(0)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, 1); err != nil {
			t.Fatal(err)
		}
		if err := w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, 2); err != nil {
			t.Fatal(err)
		}
		if _, err := w.PushLimit(0); err != nil {
			t.Fatal(err)
		}
		if err := w.Enter("later"); err != nil {
			t.Fatal(err)
		}
		err = w.WriteLiteral("L")
		// Canonical candidates postpone location normalization until all candidates
		// fail. Their cause must retain the more precise field facts meanwhile.
		err = operationFailure(err, "encode", "candidate", 0, 1, InvalidValue)
		var operation *OperationError
		if !errors.As(err, &operation) || operation.Kind != InvalidSchema || operation.Location.Path != "later" || operation.Location.BitOffset != 2 || !strings.Contains(err.Error(), "no source assigns") {
			t.Fatalf("raw=%t: %+v %v", raw, operation, err)
		}
	}
}

func TestLHPaddingRefusesLaterInferredRun(t *testing.T) {
	for _, mode := range []string{"octet", "fixed", "minimum"} {
		w := NewWriter()
		w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 8})
		w.SetZeroExtension(true)
		old, err := w.PushLimit(0)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, 1); err != nil {
			t.Fatal(err)
		}
		if err := w.PopLimit(old); err != nil {
			t.Fatal(err)
		}
		if err := w.WriteUint(0, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := w.PushLimit(0); err != nil {
			t.Fatal(err)
		}
		switch mode {
		case "octet":
			err = w.WritePadding()
		case "fixed":
			err = w.WritePaddingTo(8)
		case "minimum":
			err = w.PadMinimum(8, true)
		}
		var operation *OperationError
		if !errors.As(err, &operation) || operation.Kind != InvalidSchema || operation.Location.BitOffset != 1 || w.virtual != 1 || !strings.Contains(err.Error(), "no source assigns") {
			t.Fatalf("%s: %+v, inferred=%d, %v", mode, operation, w.virtual, err)
		}
	}
}

func TestMatchesNeverInfersPastReceivedEnd(t *testing.T) {
	for _, pattern := range []string{"0", "1", "L", "H", "LLHLHLHH"} {
		r := NewReader(nil)
		r.SetZeroExtension(true)
		got, err := r.LookaheadLiteral(pattern)
		if got || err != nil || r.VirtualBits() != 0 {
			t.Fatalf("%q: %t %v inferred=%d", pattern, got, err, r.VirtualBits())
		}
	}
}
func TestIgnoredFallbackDoesNotPromoteExpressionFailure(t *testing.T) {
	for _, known := range []func(*Reader) error{
		func(r *Reader) error { _, err := r.Eval("Missing"); return err },
		func(r *Reader) error { _, err := r.ExpressionResult(r.ExpressionVariable("Missing")); return err },
	} {
		w := NewWriter()
		mark := w.MarkFallback()
		if err := w.CheckIgnoredFallback(mark, false, known); err != nil {
			t.Fatalf("ordinary schema error became L/H refusal: %v", err)
		}
	}
}
func TestLHInferredPositionLimitKind(t *testing.T) {
	r := NewReader(nil)
	r.SetZeroExtension(true)
	r.lh.count = maxBits + 1
	_, err := r.LHBit('L')
	var de *DecodeError
	if !errors.As(err, &de) || de.Kind != Limit {
		t.Fatalf("reader: %v", err)
	}
	w := NewWriter()
	w.WithWire(WireInfo{canonical: true, targetSet: true, targetBits: 0})
	w.SetZeroExtension(true)
	w.lh.count = maxBits + 1
	_, err = w.LHBit('L')
	var oe *OperationError
	if !errors.As(err, &oe) || oe.Kind != Limit {
		t.Fatalf("writer: %v", err)
	}
}

func TestLHRefusalCauseAndChoicePriority(t *testing.T) {
	r := NewReader([]byte{0})
	inferPrefix(t, r)
	if _, err := r.ReadUint(2); err != nil {
		t.Fatal(err)
	}
	if _, err := r.PushLimit(0); err != nil {
		t.Fatal(err)
	}
	refusal := r.Expect("L")
	if !errors.Is(refusal, ErrLHPositionUndefined) {
		t.Fatalf("missing refusal cause: %v", refusal)
	}
	var operation *OperationError
	if !errors.As(refusal, &operation) || !errors.Is(operation, ErrLHPositionUndefined) {
		t.Fatalf("located refusal lost cause: %v", operation)
	}
	ordinary := r.Error(InvalidSchema, "ordinary expression error")
	if errors.Is(ordinary, ErrLHPositionUndefined) {
		t.Fatal("ordinary schema error matches refusal")
	}
	truncation := r.Error(Truncated, "input ended")
	for _, pair := range [][2]error{{nil, refusal}, {truncation, refusal}, {refusal, truncation}, {refusal, ordinary}} {
		if got := PreferTruncation(pair[0], pair[1]); got != refusal {
			t.Fatalf("lost refusal: %v", got)
		}
	}
	if got := PreferTruncation(truncation, ordinary); got != truncation {
		t.Fatalf("ordinary schema failure replaced truncation: %v", got)
	}
	w := NewWriter()
	w.WithWire(WireInfo{sealed: true, ImplicitSpans: []ImplicitSpan{{At: 0, Count: 1}, {At: 3, Count: 1}}})
	if err := w.WriteUint(0, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteUint(0, 2); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteLiteral("L"); !errors.Is(err, ErrLHPositionUndefined) {
		t.Fatalf("writer missing refusal cause: %v", err)
	}
}

func TestMatchesDelegatesToLookaheadLiteral(t *testing.T) {
	matches := NewReader([]byte{0x2b}).Matches
	if !matches("LLLLLLLL") || matches("HHHHHHHH") {
		t.Fatal("boolean lookahead changed")
	}
	r := NewReader(nil)
	r.pos = -1
	if matched, err := r.LookaheadLiteral("L"); matched || err == nil {
		t.Fatalf("invalid reader: matched=%t error=%v", matched, err)
	}
	if r.Matches("L") {
		t.Fatal("boolean lookahead hid an invalid reader as a match")
	}
}
