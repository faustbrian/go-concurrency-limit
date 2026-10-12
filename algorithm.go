package concurrencylimit

import (
	"math"
	"time"
)

// Window is the immutable aggregate supplied to an Algorithm. RecentLatency
// is the configured exact quantile of a bounded recent sample window.
type Window struct {
	// CurrentLimit is the limiter's integer admission limit when this window closed.
	CurrentLimit int
	// Samples is retained eligible duration count, which can be less than all window completions.
	Samples int
	// MaxInFlight is peak accounted concurrency for the window, including non-learning admissions.
	MaxInFlight int
	// RecentLatency is the nearest-rank configured quantile of retained durations.
	RecentLatency time.Duration
	// BaselineLatency is the limiter's smoothed minimum recent-quantile estimate.
	BaselineLatency time.Duration
	// Throughput is retained Samples divided by elapsed window seconds; ring
	// overwrite makes it an estimate rather than all-completion throughput.
	Throughput float64
	// PreviousThroughput is the preceding closed window's throughput estimate.
	PreviousThroughput float64
	// PreviousMaxInFlight is the preceding closed window's peak accounted concurrency.
	PreviousMaxInFlight int
	// Overloads counts eligible overload completions, including overwritten durations.
	Overloads uint64
	// DependencyFailures counts eligible dependency-failure completions in the window.
	DependencyFailures uint64
	generationToken    *lifecycleGeneration
}

// Decision is an algorithm's requested next limit and diagnostic state. The
// limiter applies configured step bounds and absolute min/max bounds.
type Decision struct {
	// Limit is requested integer admission capacity, clamped when the limiter applies it.
	Limit int
	// State must be finite/bounded and equal Algorithm.State after Update.
	// The limiter publishes its applied Limit as State.Estimate.
	State AlgorithmState
}

// AlgorithmState is a bounded, value-only diagnostic description of the most
// recent algorithm update.
type AlgorithmState struct {
	// Name is non-empty and at most 64 bytes; labels are not redacted.
	Name string
	// Reason is optional and at most 128 bytes, without automatic sanitization.
	Reason string
	// Estimate is finite proposed capacity; accepted limiter snapshots rewrite it
	// to applied integer capacity, while standalone algorithms can retain fractions.
	Estimate float64
	// Gradient is finite algorithm-specific diagnostic state (zero when unused).
	Gradient float64
	// QueueEstimate is finite algorithm-specific queue diagnostics, not actual FIFO depth.
	QueueEstimate float64
	// Throughput is finite algorithm-specific throughput diagnostics (zero when unused).
	Throughput float64
}

// Algorithm computes a concurrency limit from completed sampling windows.
// Limiter serializes Update, Reset, and State calls and never holds its state
// lock while invoking them. Ownership transfers to one Limiter at construction;
// do not share mutable instances between limiters or access them concurrently.
// Algorithm calls must return promptly and must not reenter the owner's Reset or
// adaptation paths while algorithmMu is held. Panics/invalid state are contained;
// blocking cannot be forcibly interrupted. Standalone constructor results have
// no internal synchronization; use Reset when explicit cold-start state is needed.
type Algorithm interface {
	// Name returns a non-empty identity of at most 64 bytes at construction.
	Name() string
	// Update consumes a copied window and updates State to exactly its returned
	// Decision.State. Non-finite or mismatched diagnostics reject application.
	Update(Window) Decision
	// Reset replaces algorithm state for the given positive initial capacity.
	Reset(initialLimit int)
	// State returns copied finite diagnostics, not mutable shared collections.
	State() AlgorithmState
	algorithm()
}

type fixedAlgorithm struct {
	limit int
	state AlgorithmState
}

// NewFixedAlgorithm returns a control algorithm that never changes the limit.
func NewFixedAlgorithm() Algorithm {
	return &fixedAlgorithm{state: AlgorithmState{Name: "fixed", Reason: "fixed"}}
}

func (*fixedAlgorithm) Name() string { return "fixed" }
func (*fixedAlgorithm) algorithm()   {}

func (algorithm *fixedAlgorithm) Reset(initialLimit int) {
	algorithm.limit = initialLimit
	algorithm.state = AlgorithmState{Name: algorithm.Name(), Reason: "reset", Estimate: float64(initialLimit)}
}

func (algorithm *fixedAlgorithm) Update(window Window) Decision {
	algorithm.limit = window.CurrentLimit
	algorithm.state = AlgorithmState{
		Name:       algorithm.Name(),
		Reason:     "fixed",
		Estimate:   float64(window.CurrentLimit),
		Throughput: window.Throughput,
	}
	return Decision{Limit: window.CurrentLimit, State: algorithm.state}
}

func (algorithm *fixedAlgorithm) State() AlgorithmState { return algorithm.state }

// AIMDConfig configures additive increase and multiplicative decrease. A
// window decreases on an overload signal or when LatencyLimit is positive and
// exceeded. It increases only when observed in-flight work reached at least
// half the current limit.
type AIMDConfig struct {
	// Increase is additive capacity in [1, MaxLimit]; no zero default is supplied.
	Increase int
	// DecreaseFactor is finite in (0, 1) and multiplies CurrentLimit before floor.
	DecreaseFactor float64
	// LatencyLimit is non-negative; zero disables its threshold and positive
	// values decrease only when RecentLatency strictly exceeds it.
	LatencyLimit time.Duration
}

type aimdAlgorithm struct {
	config AIMDConfig
	state  AlgorithmState
}

// NewAIMDAlgorithm validates explicit AIMD tuning and returns a fresh owned
// instance. Invalid parameters wrap ErrInvalidConfig; no zero defaults apply.
func NewAIMDAlgorithm(config AIMDConfig) (Algorithm, error) {
	if config.Increase < 1 || config.Increase > MaxLimit {
		return nil, invalidConfig("AIMD.Increase", "must be in [1, MaxLimit]")
	}
	if !finite(config.DecreaseFactor) || config.DecreaseFactor <= 0 || config.DecreaseFactor >= 1 {
		return nil, invalidConfig("AIMD.DecreaseFactor", "must be in (0, 1)")
	}
	if config.LatencyLimit < 0 {
		return nil, invalidConfig("AIMD.LatencyLimit", "must not be negative")
	}
	return &aimdAlgorithm{config: config, state: AlgorithmState{Name: "aimd"}}, nil
}

func (*aimdAlgorithm) Name() string { return "aimd" }
func (*aimdAlgorithm) algorithm()   {}

func (algorithm *aimdAlgorithm) Reset(initialLimit int) {
	algorithm.state = AlgorithmState{Name: algorithm.Name(), Reason: "reset", Estimate: float64(initialLimit)}
}

func (algorithm *aimdAlgorithm) Update(window Window) Decision {
	next := window.CurrentLimit
	reason := "application-limited"
	if window.Overloads > 0 || (algorithm.config.LatencyLimit > 0 && window.RecentLatency > algorithm.config.LatencyLimit) {
		next = int(math.Floor(float64(window.CurrentLimit) * algorithm.config.DecreaseFactor))
		reason = "overload"
	} else if window.MaxInFlight >= (window.CurrentLimit+1)/2 {
		next += algorithm.config.Increase
		reason = "capacity"
	}
	algorithm.state = AlgorithmState{
		Name:       algorithm.Name(),
		Reason:     reason,
		Estimate:   float64(next),
		Throughput: window.Throughput,
	}
	return Decision{Limit: next, State: algorithm.state}
}

func (algorithm *aimdAlgorithm) State() AlgorithmState { return algorithm.state }

// VegasConfig configures the Vegas-style queue estimate
// ceil(limit*(1-baseline/recent)). Queue estimates below Alpha increase the
// limit; estimates above Beta decrease it.
type VegasConfig struct {
	// Alpha is a positive low-queue threshold; queue below it increases capacity.
	Alpha int
	// Beta is greater than Alpha and at most MaxLimit; queue above it decreases.
	Beta int
	// Increase is the positive requested gain, at most MaxLimit, before limiter clamps.
	Increase int
	// Decrease is the positive requested reduction, at most MaxLimit, before clamps.
	Decrease int
}

type vegasAlgorithm struct {
	config VegasConfig
	state  AlgorithmState
}

// NewDefaultAlgorithm returns the conservative Vegas profile exercised by the
// package's reproducible constant, bursty, bimodal, heavy-tail, sparse,
// collapse, and recovery simulations.
func NewDefaultAlgorithm() Algorithm {
	return &vegasAlgorithm{
		config: VegasConfig{Alpha: 2, Beta: 4, Increase: 1, Decrease: 1},
		state:  AlgorithmState{Name: "vegas"},
	}
}

// NewVegasAlgorithm validates explicit thresholds/gains and returns a fresh
// instance. Invalid parameters wrap ErrInvalidConfig; this is RPC queue control,
// not TCP protocol conformance or a latency guarantee.
func NewVegasAlgorithm(config VegasConfig) (Algorithm, error) {
	if config.Alpha < 1 {
		return nil, invalidConfig("Vegas.Alpha", "must be positive")
	}
	if config.Beta <= config.Alpha || config.Beta > MaxLimit {
		return nil, invalidConfig("Vegas.Beta", "must be greater than Alpha and at most MaxLimit")
	}
	if config.Increase < 1 || config.Decrease < 1 || config.Increase > MaxLimit || config.Decrease > MaxLimit {
		return nil, invalidConfig("Vegas.Gain", "increase and decrease must be in [1, MaxLimit]")
	}
	return &vegasAlgorithm{config: config, state: AlgorithmState{Name: "vegas"}}, nil
}

func (*vegasAlgorithm) Name() string { return "vegas" }
func (*vegasAlgorithm) algorithm()   {}

func (algorithm *vegasAlgorithm) Reset(initialLimit int) {
	algorithm.state = AlgorithmState{Name: algorithm.Name(), Reason: "reset", Estimate: float64(initialLimit)}
}

func (algorithm *vegasAlgorithm) Update(window Window) Decision {
	next := window.CurrentLimit
	queue := queueEstimate(window.CurrentLimit, window.BaselineLatency, window.RecentLatency)
	reason := "target-queue"
	switch {
	case window.Overloads > 0:
		next -= algorithm.config.Decrease
		reason = "overload"
	case throughputStalled(window):
		next -= algorithm.config.Decrease
		reason = "throughput"
	case window.MaxInFlight < (window.CurrentLimit+1)/2:
		reason = "application-limited"
	case queue < float64(algorithm.config.Alpha):
		next += algorithm.config.Increase
		reason = "low-queue"
	case queue > float64(algorithm.config.Beta):
		next -= algorithm.config.Decrease
		reason = "high-queue"
	}
	algorithm.state = AlgorithmState{
		Name:          algorithm.Name(),
		Reason:        reason,
		Estimate:      float64(next),
		QueueEstimate: queue,
		Throughput:    window.Throughput,
	}
	return Decision{Limit: next, State: algorithm.state}
}

func (algorithm *vegasAlgorithm) State() AlgorithmState { return algorithm.state }

// Gradient2Config configures the reviewed Netflix Gradient2 equation. The
// long-window RTT is an exponential average with alpha 2/(LongWindow+1).
type Gradient2Config struct {
	// LongWindow is in [2, MaxSampleCapacity]; after ten warmup updates its
	// exponential alpha is 2/(LongWindow+1).
	LongWindow int
	// Smoothing is finite in (0, 1] and blends the fractional target estimate.
	Smoothing float64
	// Tolerance is positive, finite and at most MaxLimit; it scales long/short RTT.
	Tolerance float64
	// MinGradient is finite in (0, 1], the lower gradient clamp.
	MinGradient float64
	// QueueSize is an additive target allowance in [0, MaxLimit], not FIFO capacity.
	QueueSize int
}

type gradient2Algorithm struct {
	config        Gradient2Config
	longRTT       float64
	warmupSamples int
	estimate      float64
	state         AlgorithmState
}

// NewGradient2Algorithm validates explicit Gradient2 tuning and returns a fresh
// instance; invalid parameters wrap ErrInvalidConfig. The reviewed equation
// provenance and comparison conditions are recorded in the owning repository
// benchmarks/comparison/PROVENANCE.md, pinned to Netflix revision
// 78a74b9878d38c4c048b0304ce12a162ab7b7222; no JVM/lifecycle parity is implied.
func NewGradient2Algorithm(config Gradient2Config) (Algorithm, error) {
	if config.LongWindow < 2 || config.LongWindow > MaxSampleCapacity {
		return nil, invalidConfig("Gradient2.LongWindow", "must be in [2, MaxSampleCapacity]")
	}
	if !unitInterval(config.Smoothing) {
		return nil, invalidConfig("Gradient2.Smoothing", "must be in (0, 1]")
	}
	if !finite(config.Tolerance) || config.Tolerance <= 0 || config.Tolerance > MaxLimit {
		return nil, invalidConfig("Gradient2.Tolerance", "must be positive, finite, and at most MaxLimit")
	}
	if !finite(config.MinGradient) || config.MinGradient <= 0 || config.MinGradient > 1 {
		return nil, invalidConfig("Gradient2.MinGradient", "must be in (0, 1]")
	}
	if config.QueueSize < 0 || config.QueueSize > MaxLimit {
		return nil, invalidConfig("Gradient2.QueueSize", "must be in [0, MaxLimit]")
	}
	return &gradient2Algorithm{config: config, state: AlgorithmState{Name: "gradient2"}}, nil
}

func (*gradient2Algorithm) Name() string { return "gradient2" }
func (*gradient2Algorithm) algorithm()   {}

func (algorithm *gradient2Algorithm) Reset(initialLimit int) {
	algorithm.longRTT = 0
	algorithm.warmupSamples = 0
	algorithm.estimate = float64(initialLimit)
	algorithm.state = AlgorithmState{Name: algorithm.Name(), Reason: "reset", Estimate: float64(initialLimit)}
}

func (algorithm *gradient2Algorithm) Update(window Window) Decision {
	shortRTT := float64(window.RecentLatency)
	// The pinned reference uses ten warmup means, not LongWindow warmup samples.
	if algorithm.warmupSamples < 10 {
		algorithm.warmupSamples++
		algorithm.longRTT += (shortRTT - algorithm.longRTT) / float64(algorithm.warmupSamples)
	} else {
		alpha := 2 / float64(algorithm.config.LongWindow+1)
		algorithm.longRTT += alpha * (shortRTT - algorithm.longRTT)
	}
	if shortRTT > 0 && algorithm.longRTT/shortRTT > 2 {
		algorithm.longRTT *= 0.95
	}

	// Keep fractions between updates, but resynchronize after the owning limiter
	// applied integer/step bounds; otherwise hidden drift can defeat clamping.
	estimate := algorithm.estimate
	if int(math.Floor(estimate)) != window.CurrentLimit {
		estimate = float64(window.CurrentLimit)
	}
	next := estimate
	gradient := 1.0
	reason := "application-limited"
	if float64(window.MaxInFlight) >= estimate/2 && shortRTT > 0 {
		gradient = clampFloat(algorithm.config.Tolerance*algorithm.longRTT/shortRTT, algorithm.config.MinGradient, 1)
		target := estimate*gradient + float64(algorithm.config.QueueSize)
		next = estimate*(1-algorithm.config.Smoothing) + target*algorithm.config.Smoothing
		reason = "latency-gradient"
	}
	if window.Overloads > 0 && next >= float64(window.CurrentLimit) {
		next = float64(window.CurrentLimit - 1)
		reason = "overload"
	} else if throughputStalled(window) && next >= float64(window.CurrentLimit) {
		next = float64(window.CurrentLimit - 1)
		reason = "throughput"
	}
	algorithm.state = AlgorithmState{
		Name:          algorithm.Name(),
		Reason:        reason,
		Estimate:      next,
		Gradient:      gradient,
		QueueEstimate: float64(algorithm.config.QueueSize),
		Throughput:    window.Throughput,
	}
	algorithm.estimate = next
	return Decision{Limit: int(math.Floor(next)), State: algorithm.state}
}

func (algorithm *gradient2Algorithm) State() AlgorithmState { return algorithm.state }

func queueEstimate(limit int, baseline, recent time.Duration) float64 {
	if baseline <= 0 {
		return 0
	}
	denominator := math.Max(float64(recent), math.SmallestNonzeroFloat64)
	return max(0, math.Ceil(float64(limit)*(1-float64(baseline)/denominator)))
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func unitInterval(value float64) bool { return finite(value) && value > 0 && value <= 1 }

func clampFloat(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

func throughputStalled(window Window) bool {
	return window.PreviousMaxInFlight > 0 && window.MaxInFlight > window.PreviousMaxInFlight &&
		window.PreviousThroughput > 0 && window.Throughput <= window.PreviousThroughput
}
