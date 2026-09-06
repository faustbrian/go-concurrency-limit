# FAQ

## Why require an algorithm?

It makes adoption and compatibility explicit. `NewDefaultAlgorithm` is a named
choice rather than a silent constructor behavior.

## Why not learn requests per second?

Concurrency directly represents occupied work. Little's Law relates it to
throughput and latency; a fixed RPS threshold becomes stale when latency or
capacity changes.

## Why did latency increase after a workload change?

The baseline deliberately rises slowly. If the new class is legitimate, the
limit will probe downward and recover as the baseline adapts. Persistent
bimodality may require resource-aligned limiter boundaries.

## Why was a local rejection excluded?

No downstream capacity was consumed, so treating it as dependency failure
would create a false feedback loop.

## Can state be shared across pods?

No. This implementation is intentionally pod-local. A distributed controller
needs membership and failure semantics that are outside this package.

## Does priority reorder callers?

No. Metadata is bounded diagnostics only and admission is FIFO. This prevents
caller-inflated priority and starvation.

## Troubleshooting

### Why does the learned limit remain unchanged?

Inspect `Snapshot` and confirm that eligible completions reach both
`MinSamples` and `MinDuration`. Local rejection, queue timeout, local drop, and
ignored or canceled work intentionally do not train downstream capacity. Low
utilization also prevents upward probing because the dependency has not shown
demand for more concurrency.

### Why are calls rejected or timing out in the queue?

Compare `Snapshot().InFlight` and `Snapshot().Queued` with the configured
limit, `Config.Queue.MaxQueued`, `Config.Queue.MaxWait`, and the caller's
context deadline. An immediate
`ErrLimitExceeded` means queueing is disabled; `ErrQueueFull` means its bounded
capacity is occupied; `ErrQueueTimeout` means the configured wait elapsed.
These are local admission outcomes and must not be treated as evidence that the
downstream dependency failed.

### Why does in-flight work never return to zero?

Every admitted `Permit` must receive exactly one `Complete` call. Prefer
`Execute` when one function owns the lifecycle. For genuinely abandoned work,
configure a bounded `Config.PermitTTL` and call `ReapExpired` explicitly; the
limiter has no background reaper.

### Why does draining not wait for active work?

`BeginDrain` rejects new admission and releases queued callers, but it does not
join admitted operations. The application owns the shutdown deadline and waits
for its work to finish, classifying shutdown cancellation as `OutcomeIgnored`.
Use the snapshot's in-flight count to observe that process.

### Why does a new pod start from the initial limit?

Learning is deliberately process-local. A restart or `Reset` begins a new
generation at `InitialLimit`; state is not shared across replicas. Roll out
gradually and compare per-pod saturation, rejection, and latency rather than
expecting fleet-wide learned state.
