# CLI Versioning Guide

This directory is a nested Go module: `github.com/redhajuanda/krangka/cli/krangka`.

## Dual tags on release

Put **both** tags on the **same commit** so the root app and the CLI module each get a proper semver from Git:

| Tag | Applies to |
|-----|------------|
| `vX.Y.Z` | Root module `github.com/redhajuanda/krangka` |
| `cli/krangka/vX.Y.Z` | This nested module (`go install .../cli/krangka@latest`) |

Example:

```bash
git tag -a v1.0.5 -m "Release v1.0.5"
git tag -a cli/krangka/v1.0.5 -m "krangka CLI module v1.0.5"
git push origin v1.0.5 cli/krangka/v1.0.5
```

## Why the nested tag matters

- A root-only tag `vX.Y.Z` does **not** version the nested `cli/krangka` path for Go’s module proxy.
- Without `cli/krangka/vX.Y.Z`, `go install github.com/redhajuanda/krangka/cli/krangka@latest` may resolve to a pseudo-version like `v0.0.0-...`.

## Verification commands

```bash
go list -m github.com/redhajuanda/krangka/cli/krangka@latest
go install github.com/redhajuanda/krangka/cli/krangka@latest
krangka --version
```
