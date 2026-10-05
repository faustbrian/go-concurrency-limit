# Adaptive concurrency comparison harness

This non-releasable module isolates comparison dependencies from the public
`concurrency-limit` module. It compares bounded Gradient2 update and permit
paths across pinned local and external implementations.

The candidates are:

- Netflix `concurrency-limits` commit
  `78a74b9878d38c4c048b0304ce12a162ab7b7222`, represented by a transparent Go
  port of its `Gradient2Limit` update equation;
- Failsafe-Go adaptive limiter v0.9.8 through its public permit API; and
- `platinummonkey/go-concurrency-limits` v1.0.1 through its public Gradient2
  API.

The candidates do not expose identical sampling contracts. The local and
Netflix reference algorithms consume aggregate windows. Platinum consumes one
RTT sample per update. Failsafe-Go measures wall-clock permit duration and owns
its quantile, correlation, and windowing implementation. The normalized
control model therefore gives every candidate one successful aggregate RTT
sample per window and never injects an implementation-specific overload or
drop outcome. The report separates that common contract from
implementation-specific runtime benchmarks and does not present the Netflix Go
port as JVM performance.

Pinned source and license details are in [PROVENANCE.md](PROVENANCE.md) and
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md). Checked-in metrics, raw
benchmarks, convergence data, and per-workload plots are under
[results](results/README.md). Those historical results retain their recorded
dependency versions and are not new measurements of the current harness.

## Report output ownership

`cmd/report` is an unprivileged, operator-invoked generator, not a service
accepting request paths. The optional output-directory argument explicitly
selects its filesystem root; the operator must own and trust that directory
and its ancestors. New output directories use `0700`; an existing directory's
permissions are not changed. Fixed report filenames are opened relative to
`os.Root`, which confines traversal and symlinks to the selected directory.
The root handle is closed before the command exits, including write failures.

CSV and SVG files are set to `0600` on their opened handles before truncation
or writing, including existing files. These mode bits establish privacy on
Unix; Windows operators must enforce equivalent access through filesystem
ACLs. Successful runs overwrite those owned
report names as before. Do not select another user's directory, a privileged
destination, or a root shared with untrusted writers. Confinement does not
prevent hard-link, mount, or device-file effects; filesystem permissions and
ownership remain operator responsibilities. This guarantee relies on Go's
descriptor-backed `os.Root` behavior, not the weaker `GOOS=js` implementation.

Report data consists only of synthetic numeric observations and fixed public
workload/implementation labels, not secrets or captured production payloads.
Seeds remain deterministic to reproduce the workloads; no random value is
used for authentication, cryptographic material, or security decisions. Publish
the generated synthetic reports deliberately rather than weakening these
creation defaults or placing private data in the harness.

Run from the repository workspace with:

```sh
go test ./benchmarks/comparison/...
go test -run '^$' -bench . -benchmem \
  ./benchmarks/comparison/...
```
