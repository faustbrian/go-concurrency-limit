# Source documentation audit

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API and defaults](api.md)

## Scope and acceptance

Tier A source/API documentation pass from
`1a014a0f36789dcdd0fa385b2d1aa7c657575a77`. The repository worker owns comments
and narrowly relevant maintained-contract corrections. Preserve executable Go,
directives, dependencies, fixture/provenance locks, example outputs and historical
comparison reports. No behavioral repair or new assurance campaign is bundled.
Direct catalog consumers are the two nested harnesses and Service's
`integration/adoption` and `integration/reference-external` modules. Parent owns
shared checker adoption, ledger, external delivery and exact-source CI.

## Reproducible inventory

```sh
git ls-files '*.go' 'go.mod' '**/go.mod'
cat modules.json packages.json .golib.yaml .go-version
rg -n '^package |^type |^func |^const |^var ' --glob '*.go'
rg -n 'go:generate|Code generated|go:build|\+build|TODO|FIXME|HACK|NOTE|WARNING|Deprecated:|nolint|nosec|lint:ignore' --glob '*.go'
golib source-docs inventory --json
golib source-docs check --json
```

Three modules: public root `github.com/faustbrian/go-concurrency-limit`, internal
`benchmarks/comparison`, and test-only `integration/resilience`. Root production
package is `concurrencylimit`, with internal and `concurrencylimit_test` fixtures;
comparison contains `cmd/report` (`main`),
`internal/netflix` (`netflix`) and a test-only `comparison_test` package;
integration has test-only `resilience_test`. All module catalogs pin Go 1.27.0;
repository development pin is 1.27.2. There are 31 tracked Go files, with no
platform/build-tag variants, generated Go source or Go generation directives.
Existing compiler/analyzer annotations remain unchanged. The Netflix translation
is hand-maintained, with source/license provenance in its owning comparison
module; it must not gain a false generated header. `cmd/report` generates owned
CSV/SVG artifacts, not Go source; historical results/pins stay untouched because
only comments change its implementation. No deprecation or unfinished/warning
marker requires resolution.

| Go file | Reviewed responsibility |
| --- | --- |
| `algorithm.go` | Fixed/AIMD/Vegas/Gradient2 equations, bounds, mutable algorithm ownership and diagnostics. |
| `algorithm_test.go` | Constructor bounds and deterministic control-equation fixtures. |
| `benchmark_test.go` | Uncontended fixed permit lifecycle and direct Fixed/AIMD/Vegas/Gradient2 window updates. |
| `benchmarks/comparison/cmd/report/main.go` | Operator-owned root/file handles, synthetic fixed-seed report model and candidate normalization. |
| `benchmarks/comparison/cmd/report/main_test.go` | Report file modes/confinement, error paths and aggregate comparator lifecycle. |
| `benchmarks/comparison/comparison_test.go` | Test-only pinned candidate trace/benchmark harness with non-equivalent timing APIs. |
| `benchmarks/comparison/internal/netflix/gradient2.go` | Hand-maintained pinned Netflix equation translation, not JVM/lifecycle implementation. |
| `boundary_test.go` | Clock/algorithm/classifier/observer faults and arithmetic/lifecycle boundaries. |
| `clock.go` | Clock/timer panic containment and notification/Stop ownership. |
| `composition_test.go` | Outcome classification vs local drop/ignored learning. |
| `concurrency_test.go` | Concurrent acquire/complete/snapshot/reset and cooperative waiter lifetime. |
| `config.go` | Bounds/defaults, copied partitions and borrowed or transferred dependencies. |
| `config_boundary_test.go` | Nil, unsupported and finite-configuration validation partitions. |
| `config_test.go` | Required bounds/construction policy. |
| `doc.go` | Canonical public package ownership/lifecycle/non-goals. |
| `errors.go` | Admission/completion/lifecycle/clock/classifier sentinels. |
| `example_test.go` | Unchanged executable Execute usage and declared output. |
| `execute.go` | Synchronous protected work, result/error precedence, completion/classifier and panic handling. |
| `execute_test.go` | Cancellation/overload/nil/classifier/operation panic completion semantics. |
| `fuzz_test.go` | Generated history bytes and outcome sequences; no production request fuzzing. |
| `hardening_concurrency_test.go` | Queue cancellation/timeout/reset/drain/grant and generation-token races. |
| `integration/resilience/composition_test.go` | Test-only Retry/Hedge ordering, separate physical admissions and local refusal learning. |
| `kubernetes_test.go` | Pod-local lifecycle/rollout model, not live cluster proof. |
| `limiter.go` | Mutex/algorithm-lock ordering, permits/queue, reap/reset/drain, ring quantile and guarded decisions. |
| `limiter_test.go` | Admission, TTL, metadata, sampling and lifecycle outcomes. |
| `mutation_test.go` | Sample quantile/baseline, defensive state, outcome/error and mutation-sensitive boundaries. |
| `permit.go` | Exactly-once CAS, stale permits, capacity release, sample/observer publication. |
| `reference_model_test.go` | Generated-window AIMD/Vegas/Gradient2 equation oracles and finite diagnostic comparison. |
| `simulation_test.go` | Deterministic cold-start/collapse/recovery control traces. |
| `types.go` | Public metadata, completion, event, snapshot and saturating outcome counters. |
| `workload_campaign_test.go` | Synthetic workload campaign, bounded learning trace and minimum-sample expectations. |

## Gaps vs technical rationale

- The root comment already establishes ownership and pod-local scope; fill in
  construction, borrowed/transferred dependencies, cooperative callback/lifecycle
  caveats and stdlib/no-cgo/service boundaries. Harness command/equation/test-only
  package comments must describe their real scope, not public completeness.
- Audit every public declaration, constant, field, interface method and concrete
  public receiver; missing defaults/semantics are concentrated in sampling,
  algorithm windows/configurations, metadata/events/counters and lifecycle APIs.
  The source AST exposes 167 public declarations/members across root and internal
  reference APIs. Private capitalized methods do not require boilerplate merely
  for a proxy.
- Algorithm state transfers to one limiter; algorithmMu serializes its calls,
  and mu owns permit/queue/sampling/counters. Lock order is algorithmMu then mu;
  no user algorithm runs while mu is held. Algorithms must not reenter the owning
  limiter's adaptive/lifecycle path while their serialization lock is held.
  Observers/classifiers run outside both locks but their synchronous blocking
  and reentrant side effects are not canceled/rolled back.
- TTL reclamation occurs both in explicit ReapExpired and before admission.
  It releases bookkeeping without canceling protected work; Reset likewise
  invalidates/reopens state without waiting for old executions. Neither is hard
  isolation of uncooperative work. Limit shrink can leave InFlight above Limit
  until completions/reaping; caller TTL and lifecycle sequencing own that risk.
- Queue nodes own one-element result channels which are never closed. Under mu
  terminal routes remove each node at most once. Queue grants/flushes publish one
  buffered result; a selected cancellation may remove directly instead. If a
  selected cancellation finds a published grant, it completes that permit as
  ignored and still returns the cancellation cause. Timer duration starts after
  initial admission dispatch; no strict wall-clock bound covers blocked callbacks.
- Completion CAS consumes the first valid attempt, even when it later reports
  stale/clock errors; duplicates are not completion joins. Invalid outcomes do
  not consume it. Direct Complete clock failure releases valid capacity, rejects
  queued work and avoids learning. Execute's pre-classification clock sample,
  classifier duration and second completion sample have distinct boundaries;
  callback errors can take precedence over a completion error.
- Eligible samples are success/dependency-failure/overload only. Bounded rings
  retain recent durations, while window outcome counts may include overwritten
  samples. Reported window throughput is retained sample count divided by elapsed
  time, not an exact count of every completion if capacity overwrites samples.
  Sparse expiration is activity-driven. Step/absolute clamping protects bounds,
  not observed dependency safety or an identical external controller contract.
- Reset retains identifier sequence and safety counters, resets learned/outcome
  aggregates and drain state, and uses fresh non-zero-sized generation identity
  to reject old windows even when numeric Generation saturates. Counters saturate;
  no observer/event carries raw operation values/errors. Caller partition and
  algorithm diagnostic labels are length-bounded, not automatically redacted.
- Report generator owns operator-selected os.Root/file handles. Private mode is
  applied to an opened file before truncation; it is not atomic all-report
  publication and cannot defeat hostile hard links/devices/privileged roots.
  Deterministic synthetic traces are not production/JVM ranking or a lifecycle
  conformance claim. Fixed seeds are reproducibility, not cryptographic entropy.

Existing tests and pinned comparison provenance ground these comments; unchanged
runtime evidence is not relabeled as newly executed. No external standard
conformance claim is introduced. Correct unsupported rationale instead of
inventing it; application deadlines, callback cooperation, safe label classes,
known reset/reap boundaries and deployment sizing remain caller obligations.

## Smallest sufficient checks

- Shared bounded inventory/check and manual public/private/test rationale review.
- Pinned formatting; all-file code-token/comment-free AST/directive equivalence
  against the recorded base; owned caches in one serialized Go session.
- Native root Go-doc rendering, and stdlib documentation rendering for changed
  nested command/internal/test-only comments without building nested dependencies.
- `.golib.yaml` selects apidiff `api/baseline.txt`; Makefile and package scripts
  have no native API snapshot oracle. Keep it unchanged. Comparison CSV/SVG
  generation does not need rerunning for preserved executable inputs.
- Native Markdown links/anchors and unchanged executable-example structure;
  generation/marker inventory, changed-content secret scan, complete ordinary
  final-diff review and Intent/ACK commit. Material public clarifications go in
  Unreleased. No runtime/race/leak/active-fuzz/mutation/benchmark/generator campaign.

## Results

- Shared `golib source-docs inventory --json` and `check --json`: three modules,
  31 Go files and no findings; package/API/private lifecycle review completed.
- `GOTOOLCHAIN=go1.27.0` pinned `gofmt`: affected Go files formatted. A disposable
  stdlib parser/scanner compared all 31 files to the recorded base: code tokens,
  comment-free AST and compiler/analyzer/security annotations are identical.
  Public declarations/signatures are unchanged. Implicit closing-delimiter
  semicolons are normalized; explicit executable tokens are not discarded.
- `go doc -all .`: native root API output inspected, stable across two final
  renders. Stdlib `go/doc` rendered nested command/reference/test-only comments.
  Unchanged previously rendered sections are reused after review. Native
  apidiff baseline remains unchanged; no configured Go-doc snapshot is missing.
- Repository-native Markdown relative-link/anchor validation and executable
  example/output/error-handling structure passed. Examples and their outputs,
  generator source tokens, provenance and historical report artifacts unchanged.
- Generation/build-variant/marker inventory, changed-content secret scan and
  ordinary complete final-diff review passed with no unresolved finding.
  Intent/ACK profile/message validation and normal commit hooks precede delivery.
- No unchanged runtime, race/leak, active-fuzz, mutation, benchmark, report-generation,
  nested-dependency compilation or aggregate campaign was run for this prose pass.
  Hosted exact-source CI and remote-main delivery remain coordinator-owned.
- Serialized Go session used owned build/module/temp caches, reused for adjacent
  native checks. All three caches, helper and render outputs were removed; the
  committed worktree is retained for coordinator delivery.

No unresolved technical question prevents truthful documentation. Execution
lifetimes, callback cooperation, safe metadata labels and comparison scope remain
caller-owned boundaries, not newly inferred guarantees.

## Objective Enforcement

The qualified public Tools v2.4.0 binary and immutable reusable CI source
provide the strict tracked-source check through `make source-docs-check`,
the existing local check/CI routes and Required aggregation. Release and
selected security/assurance routes retain their existing controls. This
structural status supplements the rendered semantic audit; it does not
certify prose meaning, generator execution or runtime conformance.
