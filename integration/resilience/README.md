# Adaptive limiter resilience composition

This non-releasable integration module proves application-owned composition
through the public adaptive limiter, strict retry, and hedge contracts. Each
in-process attempt reports a known outcome, local admission rejection remains
a permanent retry outcome, and every hedge attempt must acquire its own limiter
permit before invoking downstream work.

Run from the repository workspace with:

```sh
go test ./integration/resilience/...
```
