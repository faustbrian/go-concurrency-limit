package concurrencylimit

import (
	"context"
	"time"
)

// Outcome is the exactly-once terminal classification of an admitted permit.
type Outcome uint8

const (
	// OutcomeSuccess records successful dependency work and contributes a latency sample; it is zero-valued.
	OutcomeSuccess Outcome = iota
	// OutcomeDependencyFailure records dispatched dependency failure and contributes a latency sample.
	OutcomeDependencyFailure
	// OutcomeLocalDrop records local undispatched work without a learning sample.
	OutcomeLocalDrop
	// OutcomeIgnored records local cancellation/ignored work without a learning sample.
	OutcomeIgnored
	// OutcomeOverload records explicit downstream overload and contributes a learning/overload signal.
	OutcomeOverload
)

func (outcome Outcome) valid() bool { return outcome <= OutcomeOverload }

// Metadata is optional bounded diagnostic admission metadata. Priority and
// partition do not alter FIFO admission; authorization remains with callers.
type Metadata struct {
	// Priority is diagnostic in [0, Config.MaxPriority], not a scheduling rank.
	Priority int
	// Partition is empty or a configured exact label. It is copied, not redacted,
	// and does not create a separate concurrency budget.
	Partition string
}

// Completion is the ephemeral input to an execution classifier.
type Completion struct {
	// Context is the original caller context, borrowed only for this classifier call.
	Context context.Context
	// Result is the callback value boxed without deep copying; it remains caller-owned.
	Result any
	// Err is the original callback error; it is never retained in snapshots/events.
	Err error
	// Duration is admission-to-pre-classification elapsed time, excluding queue wait
	// but including admission observer latency; direct completion samples time again.
	Duration time.Duration
}

// OutcomeCounts is a value-only terminal outcome summary.
type OutcomeCounts struct {
	// Success counts that outcome, saturating rather than wrapping.
	Success uint64
	// DependencyFailure counts that outcome, saturating rather than wrapping.
	DependencyFailure uint64
	// LocalDrop counts that outcome, saturating rather than wrapping.
	LocalDrop uint64
	// Ignored counts that outcome, saturating rather than wrapping.
	Ignored uint64
	// Overload counts that outcome, saturating rather than wrapping.
	Overload uint64
}

// Snapshot is a mutex-consistent value-only limiter copy, with no mutable collection.
// Counters saturate at uint64 maximum; lifecycle reset clears selected aggregates,
// not all safety counters. InFlight may exceed a newly decreased Limit.
type Snapshot struct {
	// Limit is the current bounded adaptive admission limit.
	Limit int
	// InFlight counts live accounted permits, not executions still running after reap/reset.
	InFlight int
	// Queued is current retained FIFO node count.
	Queued int
	// Samples counts all eligible completed samples since Reset, not just retained samples.
	Samples uint64
	// RecentSamples is the bounded currently retained window sample count.
	RecentSamples int
	// Baseline is the decaying-minimum quantile estimate, zero before learning.
	Baseline time.Duration
	// Rejections counts recorded capacity/queue/grant failures; prevalidation and
	// all drain/reset refusals are not included.
	Rejections uint64
	// QueueTimeouts counts waiter removals due to the local timer.
	QueueTimeouts uint64
	// ExpiredPermits counts reclaimed permits since Reset; no work cancellation is implied.
	ExpiredPermits uint64
	// ObserverPanics counts contained observer panics and is retained across Reset.
	ObserverPanics uint64
	// ClassifierPanics counts contained classifier panics and is retained across Reset.
	ClassifierPanics uint64
	// AlgorithmErrors counts invalid/panicking algorithm operations and survives Reset.
	AlgorithmErrors uint64
	// ClockErrors counts clock failures/backward completions and survives Reset.
	ClockErrors uint64
	// Generation is a saturating lifecycle counter; internal generation identity
	// continues to change even when this numeric counter saturates.
	Generation uint64
	// Draining rejects new admission; Reset clears it and reopens admission.
	Draining bool
	// Outcomes contains all recorded terminal outcomes since Reset, including reaping.
	Outcomes OutcomeCounts
	// Algorithm is copied bounded diagnostics, not a borrowed mutable algorithm.
	Algorithm AlgorithmState
}

// EventType identifies a stable limiter event.
type EventType uint8

const (
	// EventAdmitted records immediate or queued admission.
	EventAdmitted EventType = iota
	// EventRejected records a reported admission refusal, not every prevalidation error.
	EventRejected
	// EventCompleted records a terminal permit outcome, including ignored TTL reclamation.
	EventCompleted
	// EventLimitChanged records an applied limit change after algorithm/step bounds.
	EventLimitChanged
	// EventReset records lifecycle replacement and has a zero Timestamp.
	EventReset
	// EventDrainStarted records BeginDrain, including repeat calls, and has a zero Timestamp.
	EventDrainStarted
)

// Event is copied value-only metadata, without callback result/context/raw errors.
// Delivery can overlap or lag later state; labels are bounded, not sanitized.
type Event struct {
	// Type identifies the observed event.
	Type EventType
	// Outcome is meaningful only for EventCompleted; zero on other events is not success evidence.
	Outcome Outcome
	// Duration is completion/reap elapsed time for EventCompleted, zero otherwise.
	Duration time.Duration
	// Metadata is copied admission diagnostics; lifecycle/limit events use its zero value.
	Metadata Metadata
	// Before is the captured prior snapshot; some rejection paths capture it after removal.
	Before Snapshot
	// After is the captured post-transition snapshot, not guaranteed current at delivery.
	After Snapshot
	// Timestamp is the event time sample; reset/drain events use zero.
	Timestamp time.Time
}
