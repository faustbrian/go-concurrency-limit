# concurrency-limit documentation map

[Repository README](../README.md) · [Current Golib portal](https://github.com/faustbrian/go-library-tools/blob/main/docs/ecosystem/README.md) · [Published ecosystem snapshot](https://github.com/faustbrian/go-library-tools/blob/v1.8.4/docs/ecosystem/README.md)

This map links package-owned adoption and operational guidance. Start with
the repository README for purpose, installation, and ownership boundaries.
Published compatibility sets are frozen multi-package baselines; they do not
certify every current main revision or a deployed application.

## Module boundaries

Import only the modules needed by your application. A deprecated facade,
preview adapter, or internal harness is not interchangeable with a stable
root API; follow its own README and migration guidance before composing it.

| Module | Status | Owner detail |
| --- | --- | --- |
| `github.com/faustbrian/go-concurrency-limit` | stable | [README](../README.md) |
| `github.com/faustbrian/go-concurrency-limit/benchmarks/comparison` | internal; non-releasable | [README](../benchmarks/comparison/README.md) |
| `github.com/faustbrian/go-concurrency-limit/integration/resilience` | internal; non-releasable | [README](../integration/resilience/README.md) |

## Choose and adopt

- [Documentation](README.md).
- [Algorithms and equations](algorithms.md).
- [API and defaults](api.md).
- [Composition](composition.md).
- [FAQ](faq.md).
- [Migration](migration.md).
- [Queueing, metadata, and fairness](queueing.md).
- [Sampling and tuning](sampling.md).

## Operate and evaluate boundaries

- [Kubernetes lifecycle and HPA](kubernetes.md).
- [Operations and dashboards](operations.md).
- [Security threat model and risk register](security.md).

## Owned adapters, examples, and harnesses

- [Adaptive limiter resilience composition](../integration/resilience/README.md).

## Maintainer, specification, and evidence references

- [Adaptive concurrency comparison harness](../benchmarks/comparison/README.md).
- [Adaptive concurrency comparison results](../benchmarks/comparison/results/README.md).
- [Benchmarks and simulation evidence](benchmarks.md).
- [Source documentation audit](source-documentation-audit.md).

## Related next steps

[Contributor setup and checks](../CONTRIBUTING.md), [Compatibility policy](../COMPATIBILITY.md), [Release history](../CHANGELOG.md), [Private vulnerability reporting](../SECURITY.md), [Support](../SUPPORT.md).
