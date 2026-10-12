// Package concurrencylimit provides bounded, process-local adaptive in-flight
// admission. A Limiter owns concurrency measurement and learning; callers own
// timeouts, retries, hedges, breakers, quotas, fallbacks, and authorization.
//
// Every admitted Permit requires exactly one terminal Outcome. Execute is the
// preferred helper when one function call owns the permit lifecycle. Queue wait
// is excluded from measured execution latency because a permit's monotonic start
// timestamp is recorded only when admission is granted.
//
// Adaptive state is pod-local. Reset starts a new generation at InitialLimit
// and invalidates old permits; it does not import or infer fleet capacity.
// Construction uses New with explicit bounds and an algorithm. The algorithm
// transfers to that limiter; other callbacks are borrowed. Limiters start no
// workers and the public package uses only the standard library, without cgo or
// required external services.
//
// TTL expiry (also checked by Acquire) and Reset reclaim accounting without
// stopping protected work. They are not hard isolation for uncooperative calls;
// callers own execution deadlines and safe lifecycle boundaries. BeginDrain
// stops admission but does not wait for completion or cancel accepted work.
package concurrencylimit
