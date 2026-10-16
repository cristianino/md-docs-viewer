# mdocs

Documentación a partir de archivos Markdown, pensada para personas y para agentes de IA. Un solo binario convierte tus `.md` en:

- **Un sitio web** con la identidad de tu empresa (colores tomados del logo).
- **Un PDF navegable**: portada, índice con enlaces y números de página, marcadores laterales, pie con «página / total» y enlaces entre documentos que funcionan dentro del PDF.
- **Un `.drawio`** con los diagramas Mermaid como **formas editables** (flowchart, sequence, state, class). Los demás tipos se exportan como imagen.

Funciona sin conexión: Mermaid y las plantillas van embebidas en el binario.

## Requisitos

- [Go 1.24+](https://go.dev/dl/) para compilar.
- Chrome o Chromium, solo para generar el PDF y el `.drawio`. Si no se detecta solo, usa `--chrome <ruta>` o la variable `MDOCS_CHROME`.

## Inicio rápido

```bash
go build -o mdocs .            # en Windows: go build -o mdocs.exe .

mdocs init mi-portal           # pregunta empresa, título, descripción, año y "powered by" (no por defecto)
cd mi-portal
mdocs add docs/02-guia.md --title "Guía"
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
| `mdocs serve [-p 3000]` | Visor web con recarga en caliente de `.md` y plantillas |
| `mdocs build [pdf\|drawio\|all] [-o dist] [--mode auto\|image]` | Genera los archivos sin necesidad de `serve` |
| `mdocs add <archivo.md>` | Registra un documento (y lo crea si no existe) |
| `mdocs remove <id>` | Lo quita del índice (no borra el archivo) |
| `mdocs list` | Lista los documentos registrados |
| `mdocs check [--strict]` | Valida archivos, ids, enlaces, anclas e imágenes (exit 1 si hay errores) |
| `mdocs context` | Resume el proyecto (títulos, anclas, diagramas, problemas) |
| `mdocs agents` | Escribe `AGENTS.md` con las reglas para agentes de IA |
| `mdocs generate-styles` | Toma los colores de `logo.svg` y los guarda en `brand.config.json` |

Flags globales: `-C <carpeta>` para trabajar en otro directorio y `--chrome <ruta>` para indicar Chrome. `list`, `check`, `context` y `build` aceptan `--json`.

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

`docs.config.json` es la lista ordenada de documentos (`id`, `file`, `title`, `subtitle`). Lo mantienen `mdocs add` y `mdocs remove`.

## Más información

Cada comando, la exportación a PDF y draw.io, las plantillas y la guía para IAs están explicados en la [wiki](https://github.com/cristianino/md-docs-viewer/wiki).

## Licencia

MIT. Ver [LICENSE](LICENSE).
