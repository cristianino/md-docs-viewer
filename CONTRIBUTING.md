# Contributing

## Workflow

Trunk-based development on `master`:

1. Create a short-lived branch from `master`: `feat/...`, `fix/...`, `docs/...`, `ci/...`.
2. Write short commits in English with a conventional prefix (`feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`).
3. Open a PR to `master`. It is merged once CI is green.

## Minimum requirements (enforced by CI)

| Check | Local command |
|---|---|
| Formatting | `gofmt -l .` prints nothing; fix with `gofmt -w .` |
| Static analysis | `go vet ./...` |
| Up-to-date dependencies | `go mod tidy` leaves `go.mod`/`go.sum` unchanged |
| Tests on Linux, Windows and macOS | `go test ./...` (`-race` on Linux) |
| End-to-end | `init`, `new`, `index`, `delete`, `check --strict`, `export`, `skill`, `build all` (Chrome), `serve --base-path` |
| Repository docs are valid | `go run . check --strict` |
| Docker images | `docker build --target full .` and `--target slim .` |
| Packaging | `scripts/package.sh v0.0.0 linux amd64` |

Before opening a PR, this is enough:

```bash
gofmt -l . && go vet ./... && go test ./... && go run . check --strict
```

## Releases

A release is published by pushing a tag:

```bash
git tag v1.2.0 && git push origin v1.2.0      # v1.2.0-rc1 is published as a pre-release
```

You can also start one without git, from *Actions → release → Run workflow*: type the version and the workflow creates the tag on `master`.

The `release` workflow reruns CI and then publishes:
- `mdocs_<version>_<os>_<arch>.tar.gz` / `.zip` for Linux, macOS and Windows (amd64 and arm64), each with the binary, configuration, sample docs, `AGENTS.md`, `skill/` and `deploy/`, and no source code;
- `mdocs-skill_<version>.zip`, the claude.ai skill with `SKILL.md` and the linux/amd64 binary;
- `checksums.txt` (SHA-256);
- the images `ghcr.io/cristianino/mdocs:<version>` (with Chromium) and `:<version>-slim`.
