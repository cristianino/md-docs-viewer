#!/usr/bin/env bash
# Packages mdocs ready to use: binary + configuration + documents.
# No source code is included.
#
#   scripts/package.sh <versión> <goos> <goarch> [carpeta-salida]
#   scripts/package.sh v1.2.0 linux amd64          # → release/mdocs_1.2.0_linux_amd64.tar.gz
#   scripts/package.sh v1.2.0 windows amd64        # → release/mdocs_1.2.0_windows_amd64.zip
set -euo pipefail

VERSION="${1:?versión requerida, p. ej. v1.2.0}"
GOOS="${2:?goos requerido (linux, windows, darwin)}"
GOARCH="${3:?goarch requerido (amd64, arm64)}"
OUT="${4:-release}"

ver="${VERSION#v}"
name="mdocs_${ver}_${GOOS}_${GOARCH}"
stage="$(mktemp -d)/${name}"
mkdir -p "$stage" "$OUT"
OUT="$(cd "$OUT" && pwd)"   # ruta absoluta

bin="mdocs"
[ "$GOOS" = "windows" ] && bin="mdocs.exe"

echo "→ compilando ${GOOS}/${GOARCH} (${VERSION})"
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath \
  -ldflags "-s -w -X github.com/cristianino/mdocs/cmd/mdocs.Version=${ver}" \
  -o "${stage}/${bin}" .

# Lo necesario para "mdocs serve": las plantillas, Mermaid y el logo por
# defecto ya van dentro del binario.
cp brand.config.json docs.config.json logo.svg LICENSE README.md "$stage/"
cp assets/AGENTS.md "$stage/AGENTS.md"
cp -r docs "$stage/docs"
cp -r deploy "$stage/deploy"
cp -r skills/mdocs "$stage/skill"

cat > "${stage}/QUICKSTART.md" <<TXT
# mdocs ${ver} (${GOOS}/${GOARCH})

This package contains the binary and a sample documentation project ready to serve.

    ./${bin} serve            # http://localhost:3000
    ./${bin} build            # dist/*.pdf and dist/*.drawio (needs Chrome/Chromium)
    ./${bin} --help

To start your own project in another folder:

    ./${bin} init my-portal

To use mdocs as a skill in Claude Code:

    ./${bin} skill install --global --with-binary

Contents:
  ${bin}                 binary (templates and Mermaid embedded)
  brand.config.json      brand: company, title, description, year, colors
  docs.config.json       list of documents
  docs/                  sample documents (.md)
  logo.svg               logo (replace it and run: ./${bin} generate-styles)
  AGENTS.md              guide for AI agents
  skill/SKILL.md         agent skill (Claude Code / claude.ai)
  deploy/                nginx, systemd, Docker and Lambda examples

More information: https://github.com/cristianino/md-docs-viewer/wiki
TXT

(
  cd "$(dirname "$stage")"
  if [ "$GOOS" = "windows" ]; then
    if command -v zip >/dev/null; then
      zip -qr "${OUT}/${name}.zip" "$name"
    else
      python3 -c "import shutil,sys; shutil.make_archive(sys.argv[1], 'zip', '.', sys.argv[2])" "${OUT}/${name}" "$name"
    fi
    echo "✓ ${OUT}/${name}.zip"
  else
    tar -czf "${OUT}/${name}.tar.gz" "$name"
    echo "✓ ${OUT}/${name}.tar.gz"
  fi
)
rm -rf "$(dirname "$stage")"
