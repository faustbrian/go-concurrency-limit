# Documentation

## Getting started

- [Install and quick start](../README.md#install)
- [Executable examples](../example_test.go)
- [Package API](https://pkg.go.dev/github.com/faustbrian/go-concurrency-limit)

## Guides

Start with the [API](api.md), [algorithm equations](algorithms.md), and
[sampling model](sampling.md). Deployment owners should also read
[Kubernetes and HPA](kubernetes.md), [composition](composition.md), and
[operations](operations.md). Evidence and tradeoffs are recorded in
[benchmarks](benchmarks.md), with adoption help in [migration](migration.md),
[FAQ](faq.md), and [security](security.md).

## Repository-only modules

- The [comparison harness](../benchmarks/comparison/README.md) records bounded,
  reproducible comparisons with pinned external implementations.
- The [resilience integration harness](../integration/resilience/README.md)
  proves application-owned retry and hedge composition.

Both nested modules are internal and unreleased. The root module remains the
only supported public import and release surface.

## Support and maintenance

- [Support](../SUPPORT.md)
- [Security policy and reporting guidance](../SECURITY.md)
- [Compatibility policy](../COMPATIBILITY.md)
- [Contribution guide](../CONTRIBUTING.md)
- [Release history](../CHANGELOG.md)
- [License](../LICENSE)
