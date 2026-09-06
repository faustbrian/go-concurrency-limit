# Security policy

## Supported versions

The latest stable v1 release receives security fixes. Older releases and the
`main` branch are unsupported; upgrade before reporting unless the issue is a
regression under active development.

| Version | Supported |
| --- | --- |
| Latest stable v1 release | Yes |
| Older releases | No |
| `main` | No |

## Reporting a vulnerability

Do not disclose a suspected vulnerability in a public issue. Use the
repository's [private vulnerability report](https://github.com/faustbrian/go-concurrency-limit/security/advisories/new).
If that route is unavailable, ask a maintainer for a private contact channel
without disclosing the vulnerability.

Do not include credentials, customer identifiers, payloads, or production
traces in a public report or initial contact request.

The limiter retains only bounded numeric aggregates and configured partition
keys. It never retains operation results, errors, contexts, credentials, or
request bodies. Partition membership and priority authorization must be decided
before calling this package. Treat observer implementations as trusted local
code and keep emitted metrics free of caller-controlled high-cardinality
labels.

The module has no network, storage, unsafe, cgo, reflection-based wiring, or
background-worker dependency. Supply-chain verification is performed by the
repository vulnerability, license, SBOM, secret, and clean-consumer gates.
