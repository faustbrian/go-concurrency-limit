# Adaptive limiter resilience composition

This non-releasable integration module proves application-owned composition
through the public adaptive limiter, strict retry, and hedge contracts. Each
in-process attempt reports a known outcome, local admission rejection remains
a permanent retry outcome, and every hedge attempt must acquire its own limiter
permit before invoking downstream work.

The maintained composition selects published Retry v2.1.0, Resilience v2.0.0,
and Hedge v1.1.0. One finite v2 logical scope owns retry and hedge lineage;
an executor borrowing an outer attempt leaves permit completion to that outer
owner. The local-admission and standalone-hedge fixtures remain independent
contracts. This harness has no public release or root API/version change.
The compatible executors retain Resilience v1 as an indirect dependency;
the maintained shared-budget fixture itself uses only the v2 scope.

Run from the repository workspace with:

```sh
go test ./integration/resilience/...
```
