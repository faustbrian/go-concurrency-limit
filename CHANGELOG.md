# Changelog

## Unreleased

### Changed

- Replace copied `go-library-tools` tooling with the pinned v1.0.4 contract
  while retaining package-owned policy, typed package checks, and
  source-specific verification evidence.

### Documentation

- Remove completed implementation plans from the release tree and retain
  package-owned documentation as the maintained reference.

## 1.0.0 - 2026-08-25

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-concurrency-limit` identity while preserving its documented API and behavior.

### Added

- Add bounded standalone permit admission and typed execution helpers with
  exactly-once success, dependency-failure, local-drop, ignored, and overload
  outcomes.
- Add fixed, AIMD, Vegas-style, and Gradient2 algorithms with deterministic
  equations, bounded sampling, throughput correlation, reset state, and
  immutable diagnostics.
- Add optional bounded FIFO queueing, configured metadata cardinality,
  abandoned-permit reaping, graceful drain, pod-local reset, observers,
  simulations, fuzzing, race tests, benchmarks, and operational guidance.
- Reject admission explicitly if the non-wrapping process-local permit
  identifier sequence is exhausted.
- Emit admission/rejection events for queued grants and validate algorithm
  tuning against portable arithmetic bounds.
- Serialize lifecycle reset with algorithm decisions so a pre-reset window
  cannot overwrite cold-start state.
- Match Netflix Gradient2 warm-up averaging and preserve its fractional limit
  between updates instead of truncating adaptation on every window.
- Contain caller-supplied timer cleanup panics so queued admission still
  returns its terminal permit or error without corrupting limiter state.
- Add per-update reference equations, reproducible adversarial workload
  campaigns, metadata fairness checks, and lifecycle race stress coverage.
- Publish pinned comparative workload, convergence, CPU, memory, and allocation
  evidence, plus cross-package retry/hedge and pod lifecycle simulations.
- Release capacity without learning and fail queued admission explicitly when
  permit completion cannot obtain a valid clock timestamp.
