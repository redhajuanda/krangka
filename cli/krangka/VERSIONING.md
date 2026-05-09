# CLI Versioning Guide

This directory is a nested Go module: `github.com/redhajuanda/krangka/cli/krangka`.

When releasing a new CLI version, use module-scoped git tags so `go install ...@latest` resolves to semver instead of a pseudo-version.

## Required tag format

- `cli/krangka/vX.Y.Z`

Example:

```bash
git tag cli/krangka/v1.0.12
git push origin cli/krangka/v1.0.12
```

## Why this matters

- Root tag `vX.Y.Z` applies to the root module, not this nested module.
- Without `cli/krangka/vX.Y.Z`, Go may resolve `@latest` to a pseudo-version like `v0.0.0-...`.

## Verification commands

```bash
go list -m github.com/redhajuanda/krangka/cli/krangka@latest
go install github.com/redhajuanda/krangka/cli/krangka@latest
krangka --version
```
