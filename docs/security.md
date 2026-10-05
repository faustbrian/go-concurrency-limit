# Security threat model and risk register

Version: `CONCURRENCY-LIMIT-TM-1.0`

Reviewed: 2026-09-13

Documentation reconciled: 2026-10-04 against published v1.1.0 and main
`93353a24b107d6292d9b21c331fa32fd671222e9`; this clarifies existing source
contracts and does not represent a new runtime security audit.

Owner: go-concurrency-limit maintainers

## Scope and objectives

This model covers the released v1 root module: construction, adaptive
algorithms, admission and bounded queueing, permit completion and expiry,
execution classification, lifecycle reset and drain, snapshots, and observer
events. The comparison and resilience modules are unreleased test harnesses,
not production or release surfaces.

The package protects process-local availability by bounding admitted and queued
work, rejecting unsafe configuration and algorithm state, and releasing
capacity on every package-owned terminal path. It retains only bounded numeric
aggregates and configured partition keys. It never retains results, errors,
contexts, credentials, request bodies, or arbitrary labels.

The limiter is not authentication, authorization, quota, rate limiting,
distributed coordination, retry, hedging, circuit breaking, or autoscaling.
It performs no network, filesystem, database, process, environment, unsafe,
cgo, reflection-based wiring, or background-worker operation.

## Assets and trust boundaries

- Downstream capacity and local process memory, CPU, goroutines, and latency
  are availability assets.
- The in-flight count, FIFO queue, permit generation, and adaptive samples are
  integrity-sensitive process state.
- Partition and priority metadata can reveal business topology or create
  high-cardinality observability data if an application discloses them.
- Package-generated diagnostics, snapshots, and events contain only bounded
  value fields, not operation results, causes, contexts, or credentials.
  `Execute` returns caller results and errors without redaction, and trusted
  classifiers receive `Completion` payloads and context. Applications own the
  privacy and retention of those values.

| Boundary | Trust assumption | Owned control |
| --- | --- | --- |
| Configuration | application-owned but fallible | finite numeric validation, absolute size bounds, copied partition membership, sealed algorithms, and typed-nil rejection |
| Admission context and metadata | caller-controlled | pre-admission cancellation, one metadata value, bounded priority and partition length, configured partition membership, and bounded FIFO queueing |
| Operation | trusted application code and potentially blocking | one invocation after admission, caller context propagation, exactly one terminal permit outcome, and panic-safe capacity release before re-panic |
| Classifier | trusted in-process callback receiving `Completion` with caller context, operation result, error, and duration | invoked outside the state lock with panic containment and safety counters; the limiter does not retain completion inputs, and the classifier must not retain `Completion.Context` |
| Observer | trusted in-process callback receiving immutable value-only events | invoked outside the state lock with panic containment, bounded event fields, and safety counters |
| Clock and timer | trusted in-process dependency | panic and invalid-timer containment, backward-time normalization, timer stop on queued-call exit, and clock safety counters |
| Adaptive state | package-owned algorithms and bounded samples | finite state checks, step and absolute limit clamps, serialized algorithm calls, exact bounded quantile storage, and stale-generation rejection |
| Lifecycle | application-owned shutdown orchestration | explicit drain, reset, permit TTL, and synchronous reaping with no hidden goroutines |
| Retry and hedge composition | application-owned amplification boundary | local rejections do not invoke work; each attempt must acquire independently and share caller-owned attempt and elapsed-time budgets |

## Material risks and dispositions

Every accepted risk below has an owner, rationale, mitigation, and review
condition. Severity describes impact under the documented trust assumptions.

| ID | Threat | Severity | Status | Owner, rationale, mitigation, and review condition |
| --- | --- | --- | --- | --- |
| CL-SEC-001 | Attacker-driven demand exhausts downstream capacity, local permits, or the wait queue. | High | Mitigated in package scope | Maintainers own explicit `MaxLimit`, `MaxQueued`, and `MaxWait` rejection plus FIFO admission. Integrators must size them from downstream capacity and apply request deadlines. Review admission, queue, or limit changes and any overload incident. |
| CL-SEC-002 | A deployment configures the public absolute maxima (`MaxLimit` or `MaxQueued`) at values unsafe for its memory or downstream budget. | Medium | Accepted trusted-configuration boundary | Only trusted construction policy selects these values; hostile admission cannot raise them or bypass the configured limit. Deployment owners know the real process and dependency budget, while a smaller universal cap would break valid released-v1 uses. Set deployment-specific limits far below package maxima and load-test them. Review every capacity/topology change or memory-exhaustion incident; a future major version may reconsider absolute maxima with consumer migration evidence. |
| CL-SEC-003 | A caller forges partition or priority metadata to bypass workload authorization or affect ordering. | High | Mitigated by non-authoritative semantics | Integrators own authentication and authorization before admission. Metadata is validated and copied, configured partitions are closed, and neither priority nor partition changes FIFO ordering. Review any scheduling or partition semantic change. |
| CL-SEC-004 | Retry or hedge composition multiplies admitted downstream work or retries local overload. | Medium | Accepted trusted-composition boundary | Only application retry or hedge policy can create this amplification; attacker demand cannot make the limiter invoke or retry work. Integrators own shared attempt and elapsed-time budgets and must acquire once per attempt. The resilience harness proves local admission rejection is not retried and rejected work is not invoked. Review retry/hedge policy or composition changes. |
| CL-SEC-005 | An operation ignores cancellation or abandons a permit, permanently consuming capacity. | Medium | Accepted cooperative-work boundary | Go cannot terminate arbitrary in-process work safely. Integrators must bound operations and honor context. Maintainers provide explicit permit TTL and synchronous `ReapExpired` without leaking a background goroutine. Review incomplete shutdown, expired-permit alerts, or lifecycle redesign. |
| CL-SEC-006 | A classifier, observer, clock, or timer blocks, leaks data, races, or performs unbounded work. | Medium | Accepted trusted-callback boundary | Integrators own callback provenance, synchronization, liveness, and emitted labels. Maintainers invoke classifier and observer callbacks outside the state lock, contain panics, validate timer channels, and retain no callback inputs. Review third-party callbacks, callback contract changes, or sustained safety counters. |
| CL-SEC-007 | Callback or operation panic corrupts admission state or leaks a secret panic value. | Medium | Mitigated for limiter integrity; disclosure remains caller-owned | Maintainers release the permit and contain classifier/observer/clock panics. Operation panics deliberately re-panic the original value so application panic policy remains observable; callers must recover and redact at their trust boundary. Review panic handling or diagnostic policy changes. |
| CL-SEC-008 | Clock rollback, panic, or invalid timers strand queue entries or poison learning. | Medium | Mitigated | Maintainers reject invalid timers, normalize negative elapsed time, stop timers, release or reject affected work, avoid invalid learning, and count failures. Fault and concurrency tests exercise these paths. Review clock abstraction or queue timing changes. |
| CL-SEC-009 | Permit reuse, reset races, expiry races, or identifier exhaustion corrupts in-flight accounting. | High | Mitigated | Maintainers own atomic one-shot completion, generation-bound permit state, locked accounting, stale-permit errors, saturating counters, and explicit identifier exhaustion. Race and lifecycle tests cover completion, reset, drain, queue cancellation, and reaping. The rule-specific G115 annotation in `ReapExpired` relies on the bounded locked counter delta described below. Review permit identity, lifecycle locking, admission limits, counter saturation, or reap iteration and queued-grant ordering changes. |
| CL-SEC-010 | Malformed adaptive state causes overflow, NaN propagation, extreme allocation, or unstable limits. | High | Mitigated | Maintainers seal algorithms to the package, validate finite configuration/state, cap retained samples, use overflow-safe step arithmetic, and clamp every decision. Review algorithm equations, sample retention, or public bounds. |
| CL-SEC-011 | Events or snapshots disclose operation results, errors, contexts, secrets, request bodies, or unbounded labels. | Medium | Mitigated in package scope | Maintainers expose bounded value-only events and snapshots. Integrators must keep configured partition names and observer-added metric labels non-sensitive and low-cardinality. Review event, snapshot, or metadata field additions. |
| CL-SEC-012 | Vulnerable dependencies, unsafe runtime escape, or mutable automation weakens isolation. | Medium | Mitigated subject to current gates | Maintainers own checksum-pinned dependencies, immutable workflow references, vulnerability and license analysis, secret scanning, and safety checks. Review every dependency, action, toolchain, suppression, or scanner finding. |

There are no known Critical or High findings in this model. The accepted
Medium deployment and composition risks require trusted application policy;
attacker-controlled admission cannot raise configured bounds, invoke rejected
work, or cause the limiter itself to retry.

The `ReapExpired` G115 annotation is limited to converting its lifetime-counter
delta to `int`, not the full `uint64` counter. The state mutex remains held from
the before snapshot through conversion, excluding reset or concurrent counter
changes. Saturating increments cannot wrap or decrease the counter; one reap
pass removes each entry permit at most once, and queued grants occur afterward.
Admission and adaptation bound active permits by `MaxLimit` (`1<<30`), so the
delta is nonnegative and no larger than that bound, safely below signed 32-bit
maximum and therefore representable on both 32- and 64-bit platforms.
Maintainers must reassess this exact-rule suppression if locking, the admission
cap, saturation semantics, reap iteration, or grant ordering changes. This is
a representability disposition, not a broad scanner waiver or a new guarantee
that the saturated lifetime counter counts every expiration.

## Compatibility, consumers, and release disposition

The released v1.0.0 API baseline is unchanged on the active v1 line. Owned
consumers are the comparison and resilience harnesses plus go-service adoption
and external-reference integrations. They construct bounded fixed limiters and
use `Acquire`, `Execute`, snapshots, drain, classifiers, and ordinary lifecycle
paths; the audit found no consumer depending on behavior that requires a
breaking security repair.

There is therefore no v2 module or v2 release claim. Any future change to
absolute maxima, panic propagation, callback preemption, metadata authority, or
permit lifecycle must preserve v1 or be prepared on `main` with major Git tags,
the Go-required module/import suffix (such as `/v2`), an independent API
baseline, and consumer migration evidence. Version-specific branches and
source directories must not be introduced. The
repository and its releasable module contain no local replacement directive.

The current module verdict is releasable subject to fresh candidate-revision
gates. Scanner output is execution evidence rather than a durable source claim;
a failed or unavailable required gate blocks a release-ready verdict.

Review this model after a security incident, before a major release, and when
admission, queue, lifecycle, algorithm, callback, clock, metadata, dependency,
or composition boundaries change. Report suspected vulnerabilities through
the private process in [`SECURITY.md`](../SECURITY.md).
