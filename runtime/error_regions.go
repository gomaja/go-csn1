package runtime

import (
	"errors"
	"strings"
)

// ErrorLabel identifies a printed receiver error arm, not an application
// protocol action. TS 44.060 V19.0.0 §§11.1.2–11.1.3.3.
type ErrorLabel string

const (
	UnknownMessageType          ErrorLabel = "Unknown message type"
	DistributionPartError       ErrorLabel = "Distribution part error"
	AddressInformationPartError ErrorLabel = "Address information part error"
	NonDistributionPartError    ErrorLabel = "Non-distribution part error"
	DBPSCHMessagePartError      ErrorLabel = "DBPSCH message part error"
	MessageEscape               ErrorLabel = "Message escape"
	Ignore                      ErrorLabel = "Ignore"
)

func (label ErrorLabel) valid() bool {
	switch label {
	case UnknownMessageType, DistributionPartError, AddressInformationPartError, NonDistributionPartError, DBPSCHMessagePartError, MessageEscape, Ignore:
		return true
	}
	return false
}

// ErrorRegion is an owned error-arm value. Location.BitOffset identifies the
// end of the accepted enclosing prefix; Bits retains only the selected arm.
// The generated value retains its accepted typed prefix separately.
type ErrorRegion struct {
	Label    ErrorLabel
	Location Location
	Bits     BitString
}

// RegionDiagnostic describes a selected error arm. Replay bits belong to the
// typed ErrorRegion, not to a diagnostic collector or an ordinary WireInfo.
type RegionDiagnostic struct {
	Label    ErrorLabel
	Location Location
}

// ReceiverReport owns committed error-region and inferred-zero locations.
// ZeroFill widths identify every inferred bit in logical order; they are
// never received reserved observations. Dropped explicitly marks omissions.
type ReceiverReport struct {
	Regions  []RegionDiagnostic
	ZeroFill []Location
	Dropped  uint64
}

// ReadErrorRegion reads a source-selected error arm at the current accepted
// prefix. width -1 means the active containing end; a nonnegative width leaves
// subsequent fields available. Selection and region nesting belong to the
// generated decoder. TS 44.060 V19.0.0 §§11.1.2–11.1.3 preserve correctly
// recognized prefixes and interpret the selected error label.
func (r *Reader) ReadErrorRegion(label ErrorLabel, width int, source Source) (ErrorRegion, error) {
	if err := r.Check(); err != nil {
		return ErrorRegion{}, err
	}
	if !label.valid() {
		return ErrorRegion{}, r.fail(InvalidSchema, "unknown receiver error label")
	}
	if width < -1 {
		return ErrorRegion{}, r.fail(InvalidSchema, "invalid receiver error-region width")
	}
	if width > maxBits {
		return ErrorRegion{}, r.fail(Limit, "receiver error-region width exceeds bit limit")
	}
	if width == -1 {
		width = r.end - r.pos
	}
	if width < 0 || width > r.end-r.pos {
		return ErrorRegion{}, r.fail(Truncated, "error region exceeds containing extent")
	}
	start := r.pos
	bits, err := r.ReadBitString(width)
	if err != nil {
		return ErrorRegion{}, err
	}
	region := ErrorRegion{Label: label, Location: Location{Path: strings.Join(r.path, "/"), BitOffset: start, Width: width, Source: source}, Bits: bits}
	r.recordReceiver(label, start, width, source, nil)
	return region, nil
}

// WriteErrorRegion replays the owned bits of a selected receiver error arm.
// A canonical sender must instead construct a source-valid semantic arm.
func (w *Writer) WriteErrorRegion(region ErrorRegion) error {
	if !region.Label.valid() || region.Location.Width != region.Bits.BitLength {
		return w.operationFailure(errors.New("inconsistent receiver error region"), 0, InvalidValue)
	}
	return w.WriteBitString(region.Bits, region.Bits.BitLength)
}

// IsRegionSyntaxError recognizes wire syntax failures eligible for a printed
// error arm. Resource, schema and required-context failures stay errors.
// Opaque choice handles may be tested while their choice scope is active.
func IsRegionSyntaxError(err error) bool {
	if f, ok := err.(*deferredFailure); ok && f.lh {
		return false
	}
	kind, _, ok := choiceFailurePosition(err)
	if !ok {
		return false
	}
	if _, scoped := err.(*deferredFailure); !scoped && (errors.Is(err, ErrContextRequired) || errors.Is(err, ErrLHPositionUndefined)) {
		return false
	}
	return kind == Truncated || kind == InvalidBranch || kind == InvalidValue
}

// receiverEvent is immutable after publication. Forks share immutable prefixes
// rather than mutable slices; appending and committing never copy the prefix.
type receiverEvent struct {
	previous *receiverEvent
	label    ErrorLabel
	location Location
}

type receiverSnapshot struct {
	head           *receiverEvent
	regions, fills int
	dropped        uint64
}

type receiverBudget struct{ limit, visits, pathBudget, pathUsed int }

type receiverState struct {
	current, checkpoint receiverSnapshot
	budget              *receiverBudget
	reserved            *Collector
}

// NewReceiverCollector enables receiver diagnostics as well as the independent
// reserved-value collector. An ordinary Collector collects reserved values only.
// Complete or discard every fork before taking either report or reusing it.
func NewReceiverCollector() *Collector {
	c := new(Collector)
	c.receiver = &receiverState{reserved: c, budget: new(receiverBudget)}
	return c
}

func (c *Collector) observationCollector() *Collector {
	if c.receiver != nil {
		return c.receiver.reserved
	}
	return c
}

func (c *Collector) forkReceiver() *Collector {
	slot := new(struct {
		collector Collector
		state     receiverState
	})
	slot.state = *c.receiver
	slot.state.checkpoint = slot.state.current
	slot.collector.receiver = &slot.state
	return &slot.collector
}

func (r *Reader) commitReceiver(other *Reader) {
	c := r.collector
	state := other.collector.receiver.current
	start, dropped := r.observationStart, r.observationDropped
	other.observationStart = -1
	*r = *other
	r.collector = c
	c.receiver.current = state
	r.observationStart, r.observationDropped = start, dropped
}

// TakeReceiverReport copies immutable committed locations into owned slices and
// releases their event chain. It never consumes reserved-value observations.
func (c *Collector) TakeReceiverReport() ReceiverReport {
	if c == nil || c.receiver == nil {
		return ReceiverReport{}
	}
	state := c.receiver
	snapshot := state.current
	report := ReceiverReport{Dropped: snapshot.dropped}
	if snapshot.regions != 0 {
		report.Regions = make([]RegionDiagnostic, snapshot.regions)
	}
	if snapshot.fills != 0 {
		report.ZeroFill = make([]Location, snapshot.fills)
	}
	regions, fills := snapshot.regions, snapshot.fills
	for event := snapshot.head; event != nil; event = event.previous {
		if event.label == "" {
			fills--
			report.ZeroFill[fills] = event.location
		} else {
			regions--
			report.Regions[regions] = RegionDiagnostic{Label: event.label, Location: event.location}
		}
	}
	state.current = receiverSnapshot{}
	state.checkpoint = receiverSnapshot{}
	if state.budget != nil {
		state.budget = &receiverBudget{limit: state.budget.limit, pathBudget: state.budget.pathBudget}
	}
	return report
}

func (r *Reader) recordReceiver(label ErrorLabel, offset, width int, source Source, field []string) {
	if r.collector == nil || r.collector.receiver == nil {
		return
	}
	state := r.collector.receiver
	budget := state.budget
	// The root constructor supplies the input bound through the existing
	// reserved collector counters; only this opt-in path initializes diagnostics.
	budget.limit = int(state.reserved.eventLimit) + 128
	budget.pathBudget = budget.limit * 128
	if budget.visits >= budget.limit {
		state.current.dropped++
		return
	}
	budget.visits++
	path, ok := r.receiverPath(field, budget)
	if !ok {
		state.current.dropped++
		return
	}
	location := Location{Path: path, BitOffset: offset, Width: width, Source: source}
	state.current.head = &receiverEvent{previous: state.current.head, label: label, location: location}
	if label == "" {
		state.current.fills++
	} else {
		state.current.regions++
	}
}

func (r *Reader) receiverPath(field []string, budget *receiverBudget) (string, bool) {
	c := r.collector.observationCollector()
	if len(field) > 1 || c.indexDepth > 128 {
		return "", false
	}
	length := 0
	for _, part := range r.path {
		if len(part) > maxBits-length-1 {
			return "", false
		}
		length += len(part) + 1
	}
	for _, part := range field {
		if len(part) > maxBits-length-1 {
			return "", false
		}
		length += len(part) + 1
	}
	for _, scope := range c.indices {
		if scope.index < -1 || len(scope.path) > maxBits-length-24 {
			return "", false
		}
		length += len(scope.path) + 24
	}
	if length > budget.pathBudget-budget.pathUsed {
		return "", false
	}
	budget.pathUsed += length
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
	for _, part := range field {
		appendObservationPart(&path, part, -1)
	}
	return path.String(), true
}
