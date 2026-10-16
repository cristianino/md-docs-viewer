# Contribuir

## Flujo

Desarrollo basado en tronco sobre `master`:

1. Rama corta desde `master`: `feat/...`, `fix/...`, `docs/...`, `ci/...`.
2. Commits cortos en inglés con prefijo convencional (`feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`).
3. PR hacia `master`. Se mergea cuando el CI está en verde.

## Requisitos mínimos (los revisa el CI)

| Revisión | Comando local |
|---|---|
| Formato | `gofmt -l .` (vacío) · para corregir: `gofmt -w .` |
| Análisis estático | `go vet ./...` |
| Dependencias al día | `go mod tidy` sin cambios en `go.mod`/`go.sum` |
| Tests en Linux, Windows y macOS | `go test ./...` (`-race` en Linux) |
| Punta a punta | `init`, `new`, `index`, `delete`, `check --strict`, `export`, `build all` (Chrome), `serve --base-path` |
| Documentación del repo válida | `go run . check --strict` |
| Imágenes Docker | `docker build --target full .` y `--target slim .` |
| Empaquetado | `scripts/package.sh v0.0.0 linux amd64` |

Antes de abrir el PR basta con:

```bash
gofmt -l . && go vet ./... && go test ./... && go run . check --strict
```

## Releases

Se publica un release al subir un tag:

```bash
git tag v1.2.0 && git push origin v1.2.0      # v1.2.0-rc1 se publica como pre-release
```

El workflow `release` vuelve a correr el CI y luego publica:
- `mdocs_<versión>_<os>_<arch>.tar.gz` / `.zip` para Linux, macOS y Windows (amd64 y arm64): binario, configuración, documentos de ejemplo, `AGENTS.md` y `deploy/`, sin código fuente;
- `checksums.txt` (SHA-256);
- imágenes `ghcr.io/cristianino/mdocs:<versión>` (con Chromium) y `:<versión>-slim`.
