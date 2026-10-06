# Changelog

## Unreleased

## 1.1.1 - 2026-10-06

### Changed

- Refresh immutable CI and documentation-parser dependencies without changing
  the root limiter API, runtime behavior, or supported Go toolchain.
- Maintain the non-releasable composition harness with Resilience v2 and
  Retry v2 while retaining the executors' separate legacy Resilience v1.1
  dependency. Historical benchmark results remain unchanged.
- Confine non-releasable benchmark report output to descriptor-owned paths
  with private file and directory permissions.

## 1.1.0 - 2026-10-03

### Changed

- Raise the minimum Go version from 1.26.6 to 1.27.0. Consumers must
  upgrade their Go toolchain and CI images before adopting this release.
  The public limiter API and production implementation remain unchanged.
- Update dependencies in the non-releasable comparison and resilience
  integration modules while preserving recorded benchmark measurements.

- Migrate the non-releasable resilience integration harness to Retry v1.1.0's
  strict policy and execution contracts with explicit known outcomes for its
  in-process operations.

- Adopt the checksum-verified `go-library-tools` v1.4.0 CLI, schema-v2 cohesion
  metadata, repository-local `make cohesion` gate, and immutable W14 workflow
  while retaining package-owned policy and source-specific evidence.
- Reconcile nested-module checksums to their published v1.0.0 archives without
  changing the concurrency-limit API or runtime behavior.

### Documentation

- Complete root and repository-only module navigation, add actionable limiter
  troubleshooting and direct support and security routes, and bind the full
  documentation contract to the module gate.
- Clarify root and nested-module tag formats and correct the immutable v1.0.0
  publication date.

- Add canonical v1 installation, stable Go support, lifecycle and ownership,
  project support, and security-reporting guidance.

- Remove completed implementation plans from the release tree and retain
  package-owned documentation as the maintained reference.
- Link the package entry point to the immutable v1.4.0 Golib ecosystem index
  and resilience-family selection guidance.

## 1.0.0 - 2026-08-26

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
