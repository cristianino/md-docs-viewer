#!/usr/bin/env bash
# Empaqueta mdocs listo para usar: binario + configuración + documentos.
# No incluye código fuente.
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

cat > "${stage}/INICIO.md" <<TXT
# mdocs ${ver} (${GOOS}/${GOARCH})

Este paquete trae el binario y una documentación de ejemplo lista para servir.

    ./${bin} serve            # http://localhost:3000
    ./${bin} build            # dist/*.pdf y dist/*.drawio (requiere Chrome/Chromium)
    ./${bin} --help

Para empezar un proyecto propio en otra carpeta:

    ./${bin} init mi-portal

Contenido:
  ${bin}                 binario (plantillas y Mermaid embebidos)
  brand.config.json      marca: empresa, título, descripción, año, colores
  docs.config.json       lista de documentos
  docs/                  documentos de ejemplo (.md)
  logo.svg               logo (reemplázalo y ejecuta: ./${bin} generate-styles)
  AGENTS.md              guía para agentes de IA
  skill/SKILL.md         skill para Claude (./${bin} skill install --global)
  deploy/                ejemplos de nginx, systemd, Docker y Lambda

Más información: https://github.com/cristianino/md-docs-viewer/wiki
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
