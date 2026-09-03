# concurrency-limit

[![CI](https://github.com/faustbrian/go-concurrency-limit/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/faustbrian/go-concurrency-limit/actions/workflows/ci.yml)
[![CodeQL](https://img.shields.io/badge/CodeQL-required-blue)](https://github.com/faustbrian/go-concurrency-limit/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Mutation](https://img.shields.io/badge/mutation-100%25_required-blue)](CONTRIBUTING.md#verification)
[![Documentation](https://img.shields.io/badge/docs-checked_in_CI-blue)](docs/)
[![Go Reference](https://pkg.go.dev/badge/github.com/faustbrian/go-concurrency-limit.svg)](https://pkg.go.dev/github.com/faustbrian/go-concurrency-limit)
[![Release](https://img.shields.io/github/v/release/faustbrian/go-concurrency-limit?sort=semver)](https://github.com/faustbrian/go-concurrency-limit/releases)
[![Go](https://img.shields.io/badge/go-1.26.6-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

`concurrency-limit` is a bounded, process-local adaptive in-flight concurrency
limiter for Go. It learns a safe local limit from execution latency, achieved
throughput, utilization, and explicit overload outcomes before queues grow into
widespread timeouts.

Browse the versioned [Golib ecosystem index](https://github.com/faustbrian/go-library-tools/blob/v1.3.0/docs/ecosystem/README.md)
and its [resilience family guidance](https://github.com/faustbrian/go-library-tools/blob/v1.3.0/docs/ecosystem/design-language.md#package-families-and-selection)
to compare adaptive admission with fixed isolation, rate, breaker, retry, and
hedging policies.

It does not implement fixed semaphores, bulkhead partitions, rate quotas,
failure-rate throttling, breaker state, retries, hedges, fallbacks, discovery,
autoscaling, or a distributed control plane.

## Quick start

```go
limiter, err := concurrencylimit.New(concurrencylimit.Config{
    MinLimit:     1,
    MaxLimit:     100,
    InitialLimit: 10,
    Algorithm:    concurrencylimit.NewDefaultAlgorithm(),
})
if err != nil {
    return err
}

value, err := concurrencylimit.Execute(ctx, limiter,
    func(ctx context.Context) (string, error) {
        return dependency.Call(ctx)
    })
```

`NewDefaultAlgorithm` is the conservative Vegas profile used by the published
deterministic workload simulations. Use `NewFixedAlgorithm`,
`NewAIMDAlgorithm`, `NewVegasAlgorithm`, or `NewGradient2Algorithm` when the
deployment requires an explicit control or tuning profile.

For a standalone lifecycle, call `Acquire`, execute the admitted work, then
call `Permit.Complete` exactly once with `OutcomeSuccess`,
`OutcomeDependencyFailure`, `OutcomeLocalDrop`, `OutcomeIgnored`, or
`OutcomeOverload`. Queue wait is excluded from execution latency.

## Operational contract

- Limits and per-update movement are clamped to validated absolute bounds.
- Recent samples, configured partitions, active permits, and optional FIFO
  queueing are memory bounded.
- Sparse traffic cannot update the limit before `MinSamples` and
  `MinDuration` are both satisfied.
- Local rejection, queue timeout, local drop, and ignored/canceled completion
  do not become dependency-capacity samples.
- `ReapExpired` provides bounded abandoned-permit recovery without a
  background goroutine.
- `BeginDrain` rejects new work and releases queued callers. Shutdown
  cancellations should complete as `OutcomeIgnored`.
- `Reset` starts a new pod-local generation at `InitialLimit`; stale permits
  cannot mutate the new state.
- Observer and classifier calls execute outside the limiter state lock. Their
  panics are contained and counted.

## Documentation

- [API and defaults](docs/api.md)
- [Algorithm equations and selection](docs/algorithms.md)
- [Sampling and tuning](docs/sampling.md)
- [Queueing, metadata, and fairness](docs/queueing.md)
- [Composition and shared work budgets](docs/composition.md)
- [Kubernetes lifecycle and HPA](docs/kubernetes.md)
- [Operations, metrics, and dashboards](docs/operations.md)
- [Benchmarks and reproducible simulations](docs/benchmarks.md)
- [Migration](docs/migration.md), [FAQ](docs/faq.md), and
  [security](docs/security.md)

The module uses only the Go standard library and is licensed under MIT.
