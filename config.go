package concurrencylimit

import (
	"fmt"
	"reflect"
	"time"
)

const (
	defaultMinWindow      = 100 * time.Millisecond
	defaultMaxWindow      = time.Second
	defaultMinSamples     = 20
	defaultSampleCapacity = 256
	defaultQuantile       = 0.9
	defaultBaselineSmooth = 0.1
	defaultPermitTTL      = 5 * time.Minute
	// MaxLimit bounds configured limits and portable integer algorithm steps.
	MaxLimit = 1 << 30
	// MaxSampleCapacity bounds retained latency samples, not all completed work.
	MaxSampleCapacity = 1 << 16
	// MaxQueued bounds queued callers per limiter.
	MaxQueued = 1 << 16
	// MaxPartitions bounds explicitly allowed diagnostic partition labels.
	MaxPartitions = 64
	// MaxPartitionBytes bounds one partition label in bytes, without sanitizing secrets.
	MaxPartitionBytes = 128
	// MaxPriority bounds diagnostic priority; it does not reorder FIFO work.
	MaxPriority            = 1024
	maxAlgorithmNameBytes  = 64
	maxAlgorithmStateBytes = 128
)

// SamplingConfig bounds recent latency aggregation and adaptation steps.
// Quantiles are exact over at most Capacity retained samples.
type SamplingConfig struct {
	// MinDuration is the minimum eligible-sample window age; zero defaults to 100 ms.
	MinDuration time.Duration
	// MaxDuration discards an undersampled sparse window on later activity.
	// Zero defaults to one second; it must be at least MinDuration, not a background deadline.
	MaxDuration time.Duration
	// MinSamples is the positive retained count needed to adapt; zero defaults to 20.
	// It must not exceed Capacity.
	MinSamples int
	// Capacity bounds retained durations in a replacement ring; zero defaults to 256
	// and the maximum is MaxSampleCapacity. Outcome totals can exceed retained count.
	Capacity int
	// Quantile is a finite nearest-rank fraction in (0, 1]; zero defaults to 0.9.
	Quantile float64
	// BaselineSmoothing is finite in (0, 1], default 0.1. Lower quantiles replace
	// the baseline immediately; increases move by this fraction.
	BaselineSmoothing float64
	// MaxIncrease bounds one positive limit step; zero defaults to one, maximum MaxLimit.
	MaxIncrease int
	// MaxDecrease bounds one downward step; zero defaults to MaxLimit.
	// Absolute MinLimit still clamps the resulting limit.
	MaxDecrease int
}

// QueueConfig enables bounded FIFO waiting when both fields are positive. A
// zero value selects immediate rejection.
type QueueConfig struct {
	// MaxQueued is zero for no queue, or positive and at most package MaxQueued.
	MaxQueued int
	// MaxWait is positive when queueing is enabled and zero otherwise. Its timer
	// starts after initial admission event dispatch, not at Acquire entry; synchronous
	// callbacks are not forcibly interrupted by it.
	MaxWait time.Duration
}

// Clock supplies monotonic-capable timestamps and queue timers. Shared providers
// must be concurrency-safe and prompt; faults are contained but blocking is not
// forcibly interrupted. Nil Config.Clock selects the standard-library clock.
type Clock interface {
	// Now supplies consistent admission/completion times. Backward completion time
	// is clamped to zero duration and counted as a clock error.
	Now() time.Time
	// NewTimer returns a non-nil live timer with a usable channel. Invalid results
	// or creation/channel panics cause ErrClock on queued acquisition.
	NewTimer(time.Duration) Timer
}

// Timer is supplied by a Clock and owned by one queued Acquire call. A valid
// timer is stopped on return; its notification channel is never closed by the limiter.
type Timer interface {
	// C returns the timeout channel; it is validated and cached once before waiting.
	C() <-chan time.Time
	// Stop promptly releases timer resources and reports whether firing was prevented.
	// A panic is contained but cannot establish successful provider cleanup.
	Stop() bool
}

// Classifier maps an execution completion to exactly one terminal outcome.
// It runs synchronously without limiter locks, may overlap across executions and
// must not retain Completion.Context. Result aliases caller-owned state. Panic
// returns ErrClassifierPanic with attempted ignored completion; invalid outcomes
// also attempt ignored completion, which reap/reset races can make stale.
// Shared mutable classifier state requires caller synchronization.
type Classifier func(Completion) Outcome

// Observer receives copied value-only events synchronously without limiter locks.
// Calls may overlap or deliver out of state order. Panic is counted, not propagated;
// blocking adds to caller latency and reentrant side effects are not rolled back.
// Snapshot reentry is safe; avoid waiting for the current call's own completion.
type Observer func(Event)

// Config contains construction policy; New copies value fields and partition
// membership while borrowing callbacks/clock and taking algorithm ownership.
type Config struct {
	// MinLimit is at least one and no greater than package MaxLimit.
	MinLimit int
	// MaxLimit is between MinLimit and package MaxLimit; limits are process-local.
	MaxLimit int
	// InitialLimit is within MinLimit/MaxLimit and is restored by Reset.
	InitialLimit int
	// Algorithm is required and transfers mutable ownership to this limiter.
	// Do not share the instance with another limiter or call it concurrently yourself.
	Algorithm Algorithm
	// Sampling defines bounded activity-driven adaptation; zero fields normalize
	// to SamplingConfig defaults, rather than disabling learning.
	Sampling SamplingConfig
	// Queue is zero for immediate rejection, or has two positive validated bounds.
	Queue QueueConfig
	// PermitTTL is positive, or zero for five minutes. Expiry reclaims accounting
	// without canceling work, so choose it beyond a known execution-lifetime bound.
	PermitTTL time.Duration
	// Clock is borrowed, nil for standard time, and rejects a typed nil provider.
	Clock Clock
	// Classifier is borrowed; nil selects canceled-context ignored, nil-error
	// success, otherwise dependency failure. It need not map an error to overload.
	Classifier Classifier
	// Observer is optional and borrowed; nil disables event delivery.
	Observer Observer
	// MaxPriority admits metadata priority from zero through this value, at most
	// package MaxPriority. Zero permits only zero; priority does not change FIFO.
	MaxPriority int
	// Partitions is copied into a bounded allowed set, with at most MaxPartitions
	// unique non-empty labels of at most MaxPartitionBytes bytes. Empty metadata
	// partition remains allowed; keys are not redacted or assigned separate capacity.
	Partitions []string
}

// normalizedConfig copies partition membership and policy values, borrows
// callbacks/clock, and retains the exclusively owned mutable algorithm.
type normalizedConfig struct {
	minLimit     int
	maxLimit     int
	initialLimit int
	algorithm    Algorithm
	sampling     SamplingConfig
	queue        QueueConfig
	permitTTL    time.Duration
	clock        Clock
	classifier   Classifier
	observer     Observer
	maxPriority  int
	partitions   map[string]struct{}
}

func normalizeConfig(config Config) (normalizedConfig, error) {
	if config.MinLimit < 1 || config.MinLimit > MaxLimit {
		return normalizedConfig{}, invalidConfig("MinLimit", "must be in [1, MaxLimit]")
	}
	if config.MaxLimit < config.MinLimit || config.MaxLimit > MaxLimit {
		return normalizedConfig{}, invalidConfig("MaxLimit", "must be within MinLimit and MaxLimit")
	}
	if config.InitialLimit < config.MinLimit || config.InitialLimit > config.MaxLimit {
		return normalizedConfig{}, invalidConfig("InitialLimit", "must be within MinLimit and MaxLimit")
	}
	if nilInterface(config.Algorithm) {
		return normalizedConfig{}, invalidConfig("Algorithm", "must not be nil")
	}
	name, panicked := safeAlgorithmName(config.Algorithm)
	if panicked || name == "" || len(name) > maxAlgorithmNameBytes {
		return normalizedConfig{}, invalidConfig("Algorithm.Name", "must be nonempty and bounded")
	}

	sampling, err := normalizeSampling(config.Sampling)
	if err != nil {
		return normalizedConfig{}, err
	}
	queue, err := normalizeQueue(config.Queue)
	if err != nil {
		return normalizedConfig{}, err
	}
	if config.PermitTTL < 0 {
		return normalizedConfig{}, invalidConfig("PermitTTL", "must not be negative")
	}
	permitTTL := config.PermitTTL
	if permitTTL == 0 {
		permitTTL = defaultPermitTTL
	}
	clock := config.Clock
	if clock == nil {
		clock = systemClock{}
	} else if nilInterface(clock) {
		return normalizedConfig{}, invalidConfig("Clock", "must not be nil")
	}
	classifier := config.Classifier
	if classifier == nil {
		classifier = defaultClassifier
	}
	if config.MaxPriority < 0 || config.MaxPriority > MaxPriority {
		return normalizedConfig{}, invalidConfig("MaxPriority", "is outside the supported range")
	}
	partitions, err := normalizePartitions(config.Partitions)
	if err != nil {
		return normalizedConfig{}, err
	}

	return normalizedConfig{
		minLimit: config.MinLimit, maxLimit: config.MaxLimit, initialLimit: config.InitialLimit,
		algorithm: config.Algorithm, sampling: sampling, queue: queue, permitTTL: permitTTL,
		clock: clock, classifier: classifier, observer: config.Observer,
		maxPriority: config.MaxPriority, partitions: partitions,
	}, nil
}

func normalizeSampling(config SamplingConfig) (SamplingConfig, error) {
	if config.MinDuration < 0 || config.MaxDuration < 0 {
		return SamplingConfig{}, invalidConfig("Sampling.Duration", "must not be negative")
	}
	if config.MinDuration == 0 {
		config.MinDuration = defaultMinWindow
	}
	if config.MaxDuration == 0 {
		config.MaxDuration = defaultMaxWindow
	}
	if config.MaxDuration < config.MinDuration {
		return SamplingConfig{}, invalidConfig("Sampling.MaxDuration", "must not be less than MinDuration")
	}
	if config.MinSamples < 0 || config.Capacity < 0 {
		return SamplingConfig{}, invalidConfig("Sampling.Size", "must not be negative")
	}
	if config.MinSamples == 0 {
		config.MinSamples = defaultMinSamples
	}
	if config.Capacity == 0 {
		config.Capacity = defaultSampleCapacity
	}
	if config.Capacity > MaxSampleCapacity || config.MinSamples > config.Capacity {
		return SamplingConfig{}, invalidConfig("Sampling.Capacity", "must bound at least MinSamples")
	}
	if config.Quantile == 0 {
		config.Quantile = defaultQuantile
	}
	if !unitInterval(config.Quantile) {
		return SamplingConfig{}, invalidConfig("Sampling.Quantile", "must be in (0, 1]")
	}
	if config.BaselineSmoothing == 0 {
		config.BaselineSmoothing = defaultBaselineSmooth
	}
	if !unitInterval(config.BaselineSmoothing) {
		return SamplingConfig{}, invalidConfig("Sampling.BaselineSmoothing", "must be in (0, 1]")
	}
	if config.MaxIncrease < 0 || config.MaxDecrease < 0 || config.MaxIncrease > MaxLimit || config.MaxDecrease > MaxLimit {
		return SamplingConfig{}, invalidConfig("Sampling.Step", "must be within arithmetic bounds")
	}
	if config.MaxIncrease == 0 {
		config.MaxIncrease = 1
	}
	if config.MaxDecrease == 0 {
		config.MaxDecrease = MaxLimit
	}
	return config, nil
}

func normalizeQueue(config QueueConfig) (QueueConfig, error) {
	if config.MaxQueued < 0 || config.MaxQueued > MaxQueued || config.MaxWait < 0 {
		return QueueConfig{}, invalidConfig("Queue", "has invalid bounds")
	}
	if (config.MaxQueued == 0) != (config.MaxWait == 0) {
		return QueueConfig{}, invalidConfig("Queue", "MaxQueued and MaxWait must both be zero or positive")
	}
	return config, nil
}

func normalizePartitions(values []string) (map[string]struct{}, error) {
	if len(values) > MaxPartitions {
		return nil, invalidConfig("Partitions", "exceeds bounded cardinality")
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" || len(value) > MaxPartitionBytes {
			return nil, invalidConfig("Partitions", "contains an invalid key")
		}
		if _, exists := result[value]; exists {
			return nil, invalidConfig("Partitions", "contains a duplicate key")
		}
		result[value] = struct{}{}
	}
	return result, nil
}

func invalidConfig(field, reason string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidConfig, field, reason)
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func safeAlgorithmName(algorithm Algorithm) (name string, panicked bool) {
	defer func() { panicked = recover() != nil }()
	algorithm.algorithm()
	return algorithm.Name(), false
}
