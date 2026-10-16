# mdocs

Documentación a partir de archivos Markdown, pensada para personas y para agentes de IA. Un solo binario convierte tus `.md` en:

- **Un sitio web** con la identidad de tu empresa (colores tomados del logo).
- **Un PDF navegable**: portada, índice con enlaces y números de página, marcadores laterales, pie con «página / total» y enlaces entre documentos que funcionan dentro del PDF.
- **Un `.drawio`** con los diagramas Mermaid como **formas editables** (flowchart, sequence, state, class). Los demás tipos se exportan como imagen.

Funciona sin conexión: Mermaid y las plantillas van embebidas en el binario.

## Usar mdocs como skill (Claude y otros agentes)

La forma recomendada de usar mdocs es pedírselo a un agente: *"documenta este proyecto"*, *"agrega una página de instalación y genera el PDF"*, *"exporta los diagramas a draw.io"*. El skill [`skills/mdocs/SKILL.md`](skills/mdocs/SKILL.md), en formato [Agent Skills](https://code.claude.com/docs/en/skills), le explica al agente:
- cómo encontrar el binario,
- cómo entender el proyecto (`context`, `check`),
- cómo crear y borrar páginas,
- las reglas del Markdown,
- cómo generar el PDF, el draw.io o el sitio estático.

Todos esos comandos tienen salida `--json`.

**Claude Code**: instala el binario (ver *Instalación*) y después:

```bash
mdocs skill install --global          # ~/.claude/skills/mdocs/  → todos tus proyectos
mdocs skill install                   # .claude/skills/mdocs/    → solo este repo (commitéalo para el equipo)
```

Claude lo carga solo cuando la tarea trata de documentación, o puedes invocarlo con `/mdocs`. Con `--with-binary` también copia el ejecutable dentro del skill, por si `mdocs` no está en el PATH.

**claude.ai** (Pro, Max, Team y Enterprise, con ejecución de código activada): descarga `mdocs-skill_<versión>.zip` del [último release](https://github.com/cristianino/md-docs-viewer/releases). Ya trae `SKILL.md` y el binario para Linux, que es el sistema del entorno donde Claude ejecuta código. Súbelo en *Settings → Capabilities → Skills → Upload skill*. También puedes generarlo tú con `mdocs skill zip --with-binary`, ejecutado desde un binario linux/amd64. En claude.ai normalmente no hay Chrome, así que ahí conviene usar `check`, `context`, `new`/`delete` y `export --no-pdf --no-drawio`.

**Otros agentes** (Codex, Cursor, Copilot…): `mdocs init` crea un `AGENTS.md` con las mismas reglas, y `mdocs agents` lo regenera. `mdocs skill print` muestra el skill completo, por si quieres pegarlo en las instrucciones de otra herramienta.

## Instalación

**Desde un release (sin Go):** descarga el paquete de tu sistema en [Releases](https://github.com/cristianino/md-docs-viewer/releases): `mdocs_<versión>_linux_amd64.tar.gz`, `…_windows_amd64.zip`, `…_darwin_arm64.tar.gz`, etc. Cada paquete trae el binario, `brand.config.json`, `docs.config.json`, los `.md` de ejemplo, `logo.svg`, `AGENTS.md`, `skill/` y `deploy/`; no trae código fuente.

```bash
tar xzf mdocs_1.0.0_linux_amd64.tar.gz && cd mdocs_1.0.0_linux_amd64
./mdocs serve                  # http://localhost:3000
```

**Con Docker:** `docker run -p 8080:8080 -v "$PWD:/work:ro" ghcr.io/cristianino/mdocs` (o `:slim`, sin Chrome).

**Desde el código:** se explica en *Inicio rápido* (abajo).

## Requisitos para compilar

- [Go 1.26+](https://go.dev/dl/) para compilar.
- Chrome o Chromium, solo para generar el PDF y el `.drawio`. Si no se detecta solo, usa `--chrome <ruta>` o la variable `MDOCS_CHROME`.

## Inicio rápido

```bash
go build -o mdocs .            # en Windows: go build -o mdocs.exe .

mdocs init mi-portal           # pregunta empresa, título, descripción, año y "powered by" (no por defecto)
cd mi-portal
mdocs new "Guía de uso"        # crea docs/02-guia-de-uso.md y lo registra
mdocs serve                    # http://localhost:3000
mdocs build                    # dist/*.pdf y dist/*.drawio
```

Sin preguntas (scripts e IAs):

```bash
mdocs init --yes --company "Acme" --title "Manual técnico" --description "API interna" --year 2026
```

## Comandos

| Comando | Para qué |
|---|---|
| `mdocs init [carpeta]` | Crea la configuración, `logo.svg`, un documento de ejemplo y `AGENTS.md` |
| `mdocs serve [-p 3000] [--host] [--base-path] [--exports live\|static\|off]` | Visor web; relee `.md` y plantillas en cada request |
| `mdocs build [pdf\|drawio\|all] [-o dist] [--mode auto\|image]` | Genera los archivos sin necesidad de `serve` |
| `mdocs export [-o site] [--base-path /docs]` | Sitio estático con PDF y `.drawio` incluidos (S3, Pages, Netlify, nginx) |
| `mdocs new "<título>"` | Crea `docs/NN-slug.md` y lo registra en el índice |
| `mdocs delete <id> [-y]` | Lo quita del índice y borra el `.md` (avisa si otros documentos lo enlazan) |
| `mdocs add <archivo.md>` | Registra un `.md` existente (y lo crea si no existe) |
| `mdocs index [--prune]` | Registra todos los `.md` sueltos de `docs/`; con `--prune` quita las entradas sin archivo |
| `mdocs remove <id>` | Lo quita del índice sin borrar el archivo |
| `mdocs list` | Lista los documentos registrados |
| `mdocs check [--strict]` | Valida archivos, ids, enlaces, anclas e imágenes (exit 1 si hay errores) |
| `mdocs context` | Resume el proyecto (títulos, anclas, diagramas, problemas) |
| `mdocs agents` | Escribe `AGENTS.md` con las reglas para agentes de IA |
| `mdocs generate-styles` | Toma los colores de `logo.svg` y los guarda en `brand.config.json` |
| `mdocs healthcheck` | Consulta `/healthz` (para el HEALTHCHECK de Docker) |
| `mdocs skill install [--global]` · `skill zip` · `skill print` | Instala el skill para Claude Code, crea el `.zip` para claude.ai o lo muestra |

Flags globales: `-C <carpeta>` para trabajar en otro directorio y `--chrome <ruta>` para indicar Chrome. `new`, `delete`, `index`, `list`, `check`, `context`, `build` y `export` aceptan `--json`.

## Navegación

- **Índice general** en `/indice`, con filtro: todos los documentos y sus secciones.
- **«En esta página»**: un botón arriba a la derecha (tecla `t`) que arranca retraído y resalta la sección visible. En pantallas anchas, el contenido se angosta en vez de quedar tapado.
- **`[TOC]`**: escrito en una línea sola dentro de un `.md`, inserta la tabla de contenido de ese documento, tanto en la web como en el PDF.

## Despliegue

| Dónde | Cómo |
|---|---|
| S3 + CloudFront, GitHub Pages, Netlify, nginx | `mdocs export --base-path /ruta`: HTML estático; Chrome solo hace falta al exportar ([workflow de Pages](deploy/github-pages.yml)) |
| VPS | `mdocs serve --host 127.0.0.1 --base-path /docs` detrás de nginx ([nginx](deploy/nginx.conf), [systemd](deploy/mdocs.service)) |
| Docker | `docker build -t mdocs .` (con Chromium) o `--target slim` (sin Chrome, `--exports static`) ([compose](deploy/docker-compose.yml)) |
| AWS Lambda | Imagen `slim` + Lambda Web Adapter, con PDF y `.drawio` generados en CI ([Dockerfile](deploy/lambda/Dockerfile)) |

Variables de `serve`: `MDOCS_HOST`, `PORT`, `MDOCS_BASE_PATH`, `MDOCS_EXPORTS`, `MDOCS_DIST`, `MDOCS_CHROME`, `MDOCS_NO_SANDBOX`. El detalle está en la wiki, página *Despliegue*.

## Configuración

`brand.config.json` (lo crea `mdocs init`):

```json
{
  "primaryColor": "#1a2035",
  "accentColor": "#0066cc",
  "companyName": "Mi Empresa",
  "docsTitle": "Portal de Documentación",
  "description": "Descripción corta para la portada del PDF",
  "docsYear": "2026",
  "poweredBy": false
}
```

`docs.config.json` es la lista ordenada de documentos (`id`, `file`, `title`, `subtitle`). Lo mantienen `new`, `delete`, `add`, `index` y `remove`.

## Más información

Cada comando, la exportación a PDF y draw.io, las plantillas y la guía para IAs están explicados en la [wiki](https://github.com/cristianino/md-docs-viewer/wiki).

## Contribuir

Los PR hacia `master` deben pasar el CI: formato, `go vet`, tests en Linux, Windows y macOS, una prueba de punta a punta, las imágenes Docker y el empaquetado. Los detalles están en [CONTRIBUTING.md](CONTRIBUTING.md).

## Licencia

MIT. Ver [LICENSE](LICENSE).
