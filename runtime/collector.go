package runtime

import (
	"strconv"
	"strings"
)

// Observation records an accepted, received scalar reserved by its source.
// Receiver interpretation and canonical sender validation remain separate.
type Observation struct {
	Location Location
	Value    uint64
}

// Report owns the observations committed during one decode operation.
type Report struct {
	Observations []Observation
	// Dropped counts committed observations omitted by collection bounds.
	// A nonzero count explicitly marks the report as incomplete.
	Dropped uint64
}

// Collector stores an operation's transactional observations; its zero value is
// ready. NewReaderWithCollector sets its bounds from that reader's input, while
// visits accumulate until TakeReport. Readers sharing it must run serially.
type Collector struct {
	observations                             []Observation
	dropped                                  uint64
	eventLimit, visits, pathBudget, pathUsed int
	indices                                  []observationIndex
	indexDepth                               int
}

type observationIndex struct {
	path         string
	index, depth int
}

// TakeReport transfers the committed slice without copying and detaches its
// storage. Subsequent collection cannot change the returned report.
func (c *Collector) TakeReport() Report {
	if c == nil {
		return Report{}
	}
	report := Report{Observations: c.observations, Dropped: c.dropped}
	c.observations = nil
	c.dropped, c.visits, c.pathUsed = 0, 0, 0
	return report
}

// NewReaderWithCollector creates a bounded reader with optional observations.
// A nil collector has the same behavior as NewReader.
func NewReaderWithCollector(data []byte, c *Collector) *Reader {
	r := NewReader(data)
	r.attachCollector(c)
	return r
}

func (r *Reader) attachCollector(c *Collector) {
	r.collector = c
	if c != nil {
		c.eventLimit = r.end
		c.pathBudget = c.eventLimit * 128
	}
}

// Collecting reports whether observation collection is enabled. Generated
// field decoders test it before doing any reservation or location work.
// State after ChoiceErrors.Release is undefined for a borrowed trial.
func (r *Reader) Collecting() bool { return r != nil && r.collector != nil }

// Discard abandons a fork's observations, clearing references to their paths.
// It does not refund collection work. It is a no-op on a root or a completed transaction.
func (r *Reader) Discard() {
	if r == nil || r.collector == nil || r.observationStart < 0 {
		return
	}
	c := r.collector
	if int(r.observationStart) <= len(c.observations) {
		clear(c.observations[r.observationStart:])
		c.observations = c.observations[:r.observationStart]
		c.dropped = r.observationDropped
	}
	r.observationStart = -1
}

// Observe appends a received reserved scalar at an absolute bit position.
// Generated calls follow a successful read and an authenticated source-range
// test. An optional borrowed leaf name avoids modifying the reader path.
// An inferred or partly inferred field does not describe a received span.
func (r *Reader) Observe(offset, width int, value uint64, source Source, field ...string) error {
	if !r.Collecting() {
		return nil
	}
	// Observation failures never affect decoder state or errors.
	if err := r.Check(); err != nil {
		return nil
	}
	if offset < 0 || width <= 0 || offset > r.end || width > r.end-offset || offset+width > r.pos {
		return nil
	}
	c := r.collector
	if c.visits >= c.eventLimit {
		c.dropped++
		return nil
	}
	c.visits++
	if len(field) > 1 || c.indexDepth > 128 {
		c.dropped++
		return nil
	}
	length := 0
	for _, part := range r.path {
		if len(part) >= maxBits-length {
			c.dropped++
			return nil
		}
		length += len(part) + 1
	}
	if len(field) == 1 {
		if len(field[0]) >= maxBits-length {
			c.dropped++
			return nil
		}
		length += len(field[0]) + 1
	}
	for _, scope := range c.indices {
		if scope.index < -1 || len(scope.path) > maxBits-length-24 {
			c.dropped++
			return nil
		}
		if scope.index == -1 {
			length += len(scope.path) + 1
		} else {
			length += len(scope.path) + 24
		}
	}
	// Each received bit can contribute at most one scalar occurrence. Bound
	// speculative visits by input bits and total path materialization by 128
	// bytes per input bit, independently of the decoder's work/depth budget.
	if length > maxBits || length > c.pathBudget-c.pathUsed {
		c.dropped++
		return nil
	}
	c.pathUsed += length
	var path strings.Builder
	path.Grow(length)
	for depth := 0; depth <= len(r.path); depth++ {
		for _, scope := range c.indices {
			if scope.depth == depth {
				appendObservationPart(&path, scope.path, scope.index)
			}
		}
		if depth < len(r.path) {
			appendObservationPart(&path, r.path[depth], -1)
		}
	}
	if len(field) == 1 {
		appendObservationPart(&path, field[0], -1)
	}
	c.observations = append(c.observations, Observation{Location: Location{Path: path.String(), BitOffset: offset, Width: width, Source: source}, Value: value})
	return nil
}

// EnterIndex supplies a repeated element's index without formatting a path.
// Pair a successful call with LeaveIndex after the element decoder returns.
func (r *Reader) EnterIndex(path string, index int) error {
	if !r.Collecting() {
		return nil
	}
	if index < 0 {
		index = -2
	}
	r.enterObservationScope(path, index)
	return nil
}

// EnterObservationPath supplies a containing path without consuming decoder
// work or depth. Pair it with LeaveObservationPath after the nested decode.
func (r *Reader) EnterObservationPath(path string) {
	if r.Collecting() {
		r.enterObservationScope(path, -1)
	}
}

// LeaveObservationPath ends a containing observation path scope.
func (r *Reader) LeaveObservationPath() { r.LeaveIndex() }

func (r *Reader) enterObservationScope(path string, index int) {
	c := r.collector
	if c.indexDepth < 128 {
		if len(c.indices) == cap(c.indices) {
			grown := make([]observationIndex, len(c.indices), min(128, max(1, 2*cap(c.indices))))
			copy(grown, c.indices)
			c.indices = grown
		}
		c.indices = append(c.indices, observationIndex{path: path, index: index, depth: len(r.path)})
	}
	c.indexDepth++
}

// LeaveIndex ends a repeated element's observation scope, including on failure.
func (r *Reader) LeaveIndex() {
	if !r.Collecting() {
		return
	}
	c := r.collector
	if c.indexDepth > 0 {
		c.indexDepth--
		if c.indexDepth < len(c.indices) {
			c.indices[c.indexDepth] = observationIndex{}
			c.indices = c.indices[:c.indexDepth]
		}
	}
}

// DecodeWithReport decodes through the descriptor's reader entry point and
// returns committed observations even when decoding fails.
func (d Descriptor) DecodeWithReport(data []byte) (any, Report, error) {
	if d.DecodeWithCollector == nil {
		return nil, Report{}, ErrUnsupported
	}
	var c Collector
	value, err := d.DecodeWithCollector(data, &c)
	return value, c.TakeReport(), err
}

// DecodeWithReportWithContext supplies the containing SI4 ACS bit, as in the
// descriptor's ordinary context entry point (TS 44.018 V19.0.0 §10.5.2.35).
func (d Descriptor) DecodeWithReportWithContext(data []byte, acs SI4ACS) (any, Report, error) {
	if d.DecodeWithCollectorWithContext == nil {
		return nil, Report{}, ErrUnsupported
	}
	var c Collector
	value, err := d.DecodeWithCollectorWithContext(data, &c, acs)
	return value, c.TakeReport(), err
}

// clipWireBits isolates append storage while keeping Reader.Fork inlineable.
func clipWireBits(a, b, c []BitString) ([]BitString, []BitString, []BitString) {
	return a[:len(a):len(a)], b[:len(b):len(b)], c[:len(c):len(c)]
}

func (r *Reader) commitCollected(other *Reader) {
	start, dropped := r.observationStart, r.observationDropped
	other.observationStart = -1
	*r = *other
	r.observationStart, r.observationDropped = start, dropped
}

// appendObservationPart preserves the established path and index format while
// keeping the builder operations in one compiled body.
//
//go:noinline
func appendObservationPart(path *strings.Builder, part string, index int) {
	if path.Len() != 0 {
		path.WriteByte('/')
	}
	path.WriteString(part)
	if index >= 0 {
		path.WriteByte('[')
		var digits [24]byte
		path.Write(strconv.AppendInt(digits[:0], int64(index), 10))
		path.WriteByte(']')
	}
}
